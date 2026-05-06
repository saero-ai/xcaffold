package parser

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var varNameRegex = regexp.MustCompile("^[a-z][a-z0-9-]*$")

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
			return nil, fmt.Errorf("%s:%d: invalid variable name %q (must be kebab-case)", path, lineNum, key)
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
		basePath = filepath.Join(baseDir, "xcf", "project.vars")
	} else if !filepath.IsAbs(basePath) {
		basePath = filepath.Join(baseDir, basePath)
	}

	res, err := parseVarFile(basePath)
	if err != nil {
		return nil, err
	}

	if target != "" {
		targetPath := filepath.Join(baseDir, "xcf", fmt.Sprintf("project.%s.vars", target))
		targetVars, err := parseVarFile(targetPath)
		if err != nil {
			return nil, err
		}
		for k, v := range targetVars {
			res[k] = v
		}
	}

	localPath := filepath.Join(baseDir, "xcf", "project.vars.local")
	localVars, err := parseVarFile(localPath)
	if err != nil {
		return nil, err
	}
	for k, v := range localVars {
		res[k] = v
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
