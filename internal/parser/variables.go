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

func LoadVariableStack(baseDir, target, customFile string) (map[string]interface{}, error) {
	basePath := customFile
	if basePath == "" {
		basePath = filepath.Join(baseDir, "xcaf", "project.vars")
	} else if !filepath.IsAbs(basePath) {
		basePath = filepath.Join(baseDir, basePath)
	}

	res, err := parseVarFile(basePath)
	if err != nil {
		return nil, err
	}

	if target != "" {
		targetPath := filepath.Join(baseDir, "xcaf", "project."+target+".vars")
		targetVars, err := parseVarFile(targetPath)
		if err != nil {
			return nil, err
		}
		for k, v := range targetVars {
			res[k] = v
		}
	}

	localPath := filepath.Join(baseDir, "xcaf", "project.vars.local")
	localVars, err := parseVarFile(localPath)
	if err != nil {
		return nil, err
	}
	for k, v := range localVars {
		res[k] = v
	}

	// Resolve variable composition (variables referencing other variables)
	maxPasses := 10
	for pass := 0; pass < maxPasses; pass++ {
		madeChanges := false
		for k, v := range res {
			if strVal, ok := v.(string); ok {
				expanded, err := resolver.ExpandVariables([]byte(strVal), res, nil)
				if err != nil {
					return nil, fmt.Errorf("failed to resolve variable %q: %w", k, err)
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
			return nil, fmt.Errorf("circular variable dependency detected")
		}
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
