package compiler

import (
	"fmt"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/renderer/antigravity"
	"github.com/saero-ai/xcaffold/internal/renderer/claude"
	"github.com/saero-ai/xcaffold/internal/renderer/copilot"
	"github.com/saero-ai/xcaffold/internal/renderer/cursor"
	"github.com/saero-ai/xcaffold/internal/resolver"
)

const (
	TargetClaude      = "claude"
	TargetCursor      = "cursor"
	TargetAntigravity = "antigravity"
	TargetCopilot     = "copilot"
)

// Output is an alias for output.Output, preserved for backward compatibility.
// All callers that reference compiler.Output continue to work without changes.
type Output = output.Output

// Compile translates an XcaffoldConfig AST into platform-native files.
// target selects the output platform: "claude" (default), "cursor", "antigravity".
// If target is empty, defaults to "claude" for backward compatibility.
//
// When a project config has both root-level resources (global scope, from extends
// or implicit global loading) and project-level resources (inside the project: block),
// the compiler merges them before rendering. Project resources override global
// resources by ID. After merging, inherited resources are stripped so global
// configurations are not physically duplicated into local project directories.
func Compile(config *ast.XcaffoldConfig, baseDir string, target string) (*Output, []renderer.FidelityNote, error) {
	if target == "" {
		target = TargetClaude
	}

	if config.Project != nil {
		mergeResourceScope(&config.ResourceScope, &config.Project.ResourceScope)
	}

	if err := resolver.ResolveAttributes(config); err != nil {
		return nil, nil, fmt.Errorf("attribute resolution failed: %w", err)
	}

	config.StripInherited()

	switch target {
	case TargetClaude:
		r := claude.New()
		return r.Compile(config, baseDir)
	case TargetCursor:
		r := cursor.New()
		return r.Compile(config, baseDir)
	case TargetAntigravity:
		r := antigravity.New()
		return r.Compile(config, baseDir)
	case TargetCopilot:
		r := copilot.New()
		return r.Compile(config, baseDir)
	default:
		return nil, nil, fmt.Errorf("unsupported target %q: supported targets are \"claude\", \"cursor\", \"antigravity\", \"copilot\"", target)
	}
}

// mergeResourceScope overlays project-scoped resources onto the root scope.
// For map-based resources (agents, skills, rules, MCP, workflows), project
// entries override global entries with the same ID. Hooks are additive.
func mergeResourceScope(root, project *ast.ResourceScope) {
	root.Agents = mergeMap(root.Agents, project.Agents)
	root.Skills = mergeMap(root.Skills, project.Skills)
	root.Rules = mergeMap(root.Rules, project.Rules)
	root.MCP = mergeMap(root.MCP, project.MCP)
	root.Workflows = mergeMap(root.Workflows, project.Workflows)

	// Hooks are additive — project hooks append to global hooks.
	if project.Hooks != nil {
		if root.Hooks == nil {
			root.Hooks = make(ast.HookConfig)
		}
		for event, groups := range project.Hooks {
			root.Hooks[event] = append(root.Hooks[event], groups...)
		}
	}
}

// mergeMap copies base entries then overlays child entries (child wins on conflict).
func mergeMap[K comparable, V any](base, child map[K]V) map[K]V {
	if base == nil && child == nil {
		return nil
	}
	merged := make(map[K]V, len(base)+len(child))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range child {
		merged[k] = v
	}
	return merged
}

// OutputDir returns the target-specific root directory for compilation outputs
// (e.g. .claude, .cursor, .agents).
func OutputDir(target string) string {
	if target == "" {
		target = TargetClaude
	}
	switch target {
	case TargetClaude:
		return claude.New().OutputDir()
	case TargetCursor:
		return cursor.New().OutputDir()
	case TargetAntigravity:
		return antigravity.New().OutputDir()
	default:
		return ".claude"
	}
}
