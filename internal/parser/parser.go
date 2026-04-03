package parser

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"gopkg.in/yaml.v3"
)

// Parse reads a .xcf YAML configuration from the given reader and returns a
// validated XcaffoldConfig. It does not resolve 'extends:' references.
func Parse(r io.Reader) (*ast.XcaffoldConfig, error) {
	config := &ast.XcaffoldConfig{}
	decoder := yaml.NewDecoder(r)
	decoder.KnownFields(true)
	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("failed to parse .xcf YAML: %w", err)
	}
	if err := validate(config); err != nil {
		return nil, fmt.Errorf("invalid .xcf configuration: %w", err)
	}
	return config, nil
}

// ParseFile reads a .xcf YAML configuration from the given path, resolving
// 'extends:' references recursively.
func ParseFile(path string) (*ast.XcaffoldConfig, error) {
	return parseFileRecursive(path, make(map[string]bool))
}

func parseFileRecursive(path string, visited map[string]bool) (*ast.XcaffoldConfig, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("could not resolve path %q: %w", path, err)
	}

	if visited[absPath] {
		return nil, fmt.Errorf("circular extends detected: %q", absPath)
	}
	visited[absPath] = true

	f, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("could not open config %q: %w", absPath, err)
	}
	defer f.Close()

	config, err := Parse(f)
	if err != nil {
		return nil, fmt.Errorf("error in %q: %w", absPath, err)
	}

	if config.Extends == "" {
		return config, nil
	}

	// Resolve the extends path relative to the current file's directory.
	basePath := filepath.Join(filepath.Dir(absPath), config.Extends)
	baseConfig, err := parseFileRecursive(basePath, visited)
	if err != nil {
		return nil, fmt.Errorf("failed to load base config %q: %w", config.Extends, err)
	}

	return mergeConfig(baseConfig, config), nil
}

func mergeConfig(base, child *ast.XcaffoldConfig) *ast.XcaffoldConfig {
	merged := &ast.XcaffoldConfig{
		Version: child.Version, // child overrides version
	}

	merged.Project = base.Project
	if child.Project.Name != "" {
		merged.Project.Name = child.Project.Name
	}
	if child.Project.Description != "" {
		merged.Project.Description = child.Project.Description
	}

	merged.Agents = mergeMap(base.Agents, child.Agents)
	merged.Skills = mergeMap(base.Skills, child.Skills)
	merged.Rules = mergeMap(base.Rules, child.Rules)
	merged.Hooks = mergeMap(base.Hooks, child.Hooks)
	merged.MCP = mergeMap(base.MCP, child.MCP)

	// Test config merge
	merged.Test = base.Test
	if child.Test.ClaudePath != "" {
		merged.Test.ClaudePath = child.Test.ClaudePath
	}
	if child.Test.JudgeModel != "" {
		merged.Test.JudgeModel = child.Test.JudgeModel
	}

	return merged
}

func mergeMap[K comparable, V any](base, child map[K]V) map[K]V {
	if base == nil && child == nil {
		return nil
	}
	merged := make(map[K]V)
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range child {
		merged[k] = v // child overrides base completely
	}
	return merged
}

// validate performs semantic validation on a parsed config.
func validate(c *ast.XcaffoldConfig) error {
	if c.Version == "" {
		return fmt.Errorf("version is required (e.g. \"1.0\")")
	}

	// If the config extends another, the project name can be omitted and inherited.
	if c.Extends == "" {
		name := strings.TrimSpace(c.Project.Name)
		if name == "" {
			return fmt.Errorf("project.name is required and must not be empty unless extending another config")
		}
	}

	for id := range c.Agents {
		if strings.ContainsAny(id, "/\\") || strings.Contains(id, "..") {
			return fmt.Errorf("agent id contains invalid characters: %q", id)
		}
	}
	return nil
}
