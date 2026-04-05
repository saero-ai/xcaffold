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

// validateID checks a single resource ID for path-traversal characters.
// This is a defence-in-depth measure applied at parse time; the compiler also
// uses filepath.Clean, but we want to reject bad IDs as early as possible.
func validateID(kind, id string) error {
	if strings.ContainsAny(id, "/\\") || strings.Contains(id, "..") {
		return fmt.Errorf("%s id contains invalid characters: %q", kind, id)
	}
	return nil
}

// validHookEvents is the set of lifecycle events recognized by Claude Code.
var validHookEvents = map[string]bool{
	"PreToolUse": true, "PostToolUse": true, "PostToolUseFailure": true,
	"PermissionRequest": true, "PermissionDenied": true,
	"SessionStart": true, "SessionEnd": true,
	"UserPromptSubmit": true, "Stop": true, "StopFailure": true,
	"SubagentStart": true, "SubagentStop": true, "TeammateIdle": true,
	"TaskCreated": true, "TaskCompleted": true,
	"PreCompact": true, "PostCompact": true,
	"InstructionsLoaded": true, "ConfigChange": true,
	"CwdChanged": true, "FileChanged": true,
	"WorktreeCreate": true, "WorktreeRemove": true,
	"Elicitation": true, "ElicitationResult": true,
	"Notification": true,
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

	// Validate all resource IDs for path-traversal characters (Bugs 2-4).
	for id := range c.Agents {
		if err := validateID("agent", id); err != nil {
			return err
		}
	}
	for id := range c.Skills {
		if err := validateID("skill", id); err != nil {
			return err
		}
	}
	for id := range c.Rules {
		if err := validateID("rule", id); err != nil {
			return err
		}
	}
	for id := range c.Hooks {
		if err := validateID("hook", id); err != nil {
			return err
		}
	}

	// Validate hook event names against the supported set.
	for event := range c.Hooks {
		if !validHookEvents[event] {
			return fmt.Errorf("unknown hook event %q; see documentation for supported lifecycle events", event)
		}
	}

	for id := range c.MCP {
		if err := validateID("mcp", id); err != nil {
			return err
		}
	}

	// Validate instructions_file: mutual exclusivity and path safety.
	for id, agent := range c.Agents {
		if agent.Instructions != "" && agent.InstructionsFile != "" {
			return fmt.Errorf("agent %q: instructions and instructions_file are mutually exclusive; set one or the other", id)
		}
		if err := validateInstructionsFile("agent", id, agent.InstructionsFile); err != nil {
			return err
		}
	}
	for id, skill := range c.Skills {
		if skill.Instructions != "" && skill.InstructionsFile != "" {
			return fmt.Errorf("skill %q: instructions and instructions_file are mutually exclusive; set one or the other", id)
		}
		if err := validateInstructionsFile("skill", id, skill.InstructionsFile); err != nil {
			return err
		}
	}
	for id, rule := range c.Rules {
		if rule.Instructions != "" && rule.InstructionsFile != "" {
			return fmt.Errorf("rule %q: instructions and instructions_file are mutually exclusive; set one or the other", id)
		}
		if err := validateInstructionsFile("rule", id, rule.InstructionsFile); err != nil {
			return err
		}
	}

	return nil
}

// validateInstructionsFile checks that an instructions_file path is safe.
// The path must be relative and must not contain path-traversal sequences.
func validateInstructionsFile(kind, id, path string) error {
	if path == "" {
		return nil
	}
	if filepath.IsAbs(path) {
		return fmt.Errorf("%s %q: instructions_file must be a relative path, got absolute path %q", kind, id, path)
	}
	if strings.ContainsAny(path, "\\") || strings.Contains(path, "..") {
		return fmt.Errorf("%s %q: instructions_file contains invalid path characters: %q", kind, id, path)
	}
	return nil
}
