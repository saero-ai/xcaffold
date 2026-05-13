package parser

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/saero-ai/xcaffold/internal/resolver"
	"gopkg.in/yaml.v3"
)

var varNameRegex = regexp.MustCompile("^[a-zA-Z][_a-zA-Z0-9-]*$")

func parseVarFile(path string) (map[string]interface{}, error) {
	vars := make(map[string]interface{})
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return vars, nil
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("%s:%d: invalid variable line (missing =)", path, lineNum)
		}

		key := strings.TrimSpace(parts[0])
		valStr := strings.TrimSpace(parts[1])

		if !varNameRegex.MatchString(key) {
			return nil, fmt.Errorf("%s:%d: invalid variable name %q", path, lineNum, key)
		}

		var val interface{}
		if err := yaml.Unmarshal([]byte(valStr), &val); err != nil {
			return nil, fmt.Errorf("%s:%d: invalid value %q: %w", path, lineNum, valStr, err)
		}
		vars[key] = val
	}

	return vars, scanner.Err()
}

// mergeVarFile loads and merges variables from a file into the result map.
func mergeVarFile(filePath string, res map[string]interface{}) error {
	vars, err := parseVarFile(filePath)
	if err != nil {
		return err
	}
	for k, v := range vars {
		res[k] = v
	}
	return nil
}

// resolveVariableComposition iteratively expands variables that reference other variables.
func resolveVariableComposition(res map[string]interface{}) error {
	maxPasses := 10
	for pass := 0; pass < maxPasses; pass++ {
		madeChanges := false
		for k, v := range res {
			if strVal, ok := v.(string); ok {
				expanded, err := resolver.ExpandVariables([]byte(strVal), res, nil)
				if err != nil {
					return fmt.Errorf("failed to resolve variable %q: %w", k, err)
				}
				if string(expanded) != strVal {
					res[k] = string(expanded)
					madeChanges = true
				}
			}
		}
		if !madeChanges {
			break
		}
		if pass == maxPasses-1 {
			return fmt.Errorf("circular variable dependency detected")
		}
	}
	return nil
}

func LoadVariableStack(baseDir, target, customFile string) (map[string]interface{}, error) {
	res, err := parseVarFile(filepath.Join(baseDir, "xcaf", "project.vars"))
	if err != nil {
		return nil, err
	}

	// Merge custom variables if provided
	if customFile != "" {
		cfPath := customFile
		if !filepath.IsAbs(cfPath) {
			cfPath = filepath.Join(baseDir, cfPath)
		}
		if err := mergeVarFile(cfPath, res); err != nil {
			return nil, err
		}
	}

	// Merge target-specific variables
	if target != "" {
		targetPath := filepath.Join(baseDir, "xcaf", "project."+target+".vars")
		if err := mergeVarFile(targetPath, res); err != nil {
			return nil, err
		}
	}

	// Merge local overrides
	localPath := filepath.Join(baseDir, "xcaf", "project.vars.local")
	if err := mergeVarFile(localPath, res); err != nil {
		return nil, err
	}

	// Resolve variable composition (variables referencing other variables)
	if err := resolveVariableComposition(res); err != nil {
		return nil, err
	}

	return res, nil
}

func LoadEnv(allowed []string) map[string]string {
	res := make(map[string]string)
	for _, name := range allowed {
		if val, ok := os.LookupEnv(name); ok {
			res[name] = val
		}
	}
	return res
}
