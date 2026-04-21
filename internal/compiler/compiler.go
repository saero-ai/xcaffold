package compiler

import (
	"fmt"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/blueprint"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/renderer/antigravity"
	"github.com/saero-ai/xcaffold/internal/renderer/claude"
	"github.com/saero-ai/xcaffold/internal/renderer/copilot"
	"github.com/saero-ai/xcaffold/internal/renderer/cursor"
	"github.com/saero-ai/xcaffold/internal/renderer/gemini"
	"github.com/saero-ai/xcaffold/internal/resolver"
)

const (
	TargetClaude      = "claude"
	TargetCursor      = "cursor"
	TargetAntigravity = "antigravity"
	TargetCopilot     = "copilot"
	TargetGemini      = "gemini"
)

// Output is an alias for output.Output, preserved for backward compatibility.
// All callers that reference compiler.Output continue to work without changes.
type Output = output.Output

// Compile translates an XcaffoldConfig AST into platform-native files.
// target selects the output platform: "claude", "cursor", "antigravity", "copilot", "gemini".
// An empty target defaults to "claude" for backward compatibility.
// blueprintName narrows compilation to the named blueprint's resource subset.
// If blueprintName is empty, all resources are compiled.
//
// When a project config has both root-level resources (global scope, from extends
// or implicit global loading) and project-level resources (inside the project: block),
// the compiler merges them before rendering. Project resources override global
// resources by ID. After merging, inherited resources are stripped so global
// configurations are not physically duplicated into local project directories.
func Compile(config *ast.XcaffoldConfig, baseDir string, target string, blueprintName string) (*Output, []renderer.FidelityNote, error) {
	if target == "" {
		target = TargetClaude
	}

	if config.Project != nil {
		mergeResourceScope(&config.ResourceScope, &config.Project.ResourceScope)
	}

	if err := resolver.ResolveAttributes(config); err != nil {
		return nil, nil, fmt.Errorf("attribute resolution failed: %w", err)
	}

	// Blueprint resolution: resolve extends chains and transitive deps before filtering.
	if len(config.Blueprints) > 0 {
		if err := blueprint.ResolveBlueprintExtends(config.Blueprints); err != nil {
			return nil, nil, fmt.Errorf("blueprint extends resolution failed: %w", err)
		}
		if errs := blueprint.ValidateBlueprintRefs(config.Blueprints, &config.ResourceScope); len(errs) > 0 {
			msgs := make([]string, len(errs))
			for i, e := range errs {
				msgs[i] = e.Error()
			}
			return nil, nil, fmt.Errorf("blueprint validation errors:\n%s", strings.Join(msgs, "\n"))
		}
	}
	if blueprintName != "" {
		if p, ok := config.Blueprints[blueprintName]; ok {
			blueprint.ResolveTransitiveDeps(&p, &config.ResourceScope)
			config.Blueprints[blueprintName] = p
		}
	}

	// Blueprint filtering: narrow the resource scope to the named blueprint's subset.
	if blueprintName != "" {
		var err error
		config, err = blueprint.ApplyBlueprint(config, blueprintName)
		if err != nil {
			return nil, nil, fmt.Errorf("blueprint filter failed: %w", err)
		}
	}

	config.StripInherited()

	r, err := resolveRenderer(target)
	if err != nil {
		return nil, nil, err
	}
	return renderer.Orchestrate(r, config, baseDir)
}

// resolveRenderer returns the TargetRenderer for the given target name.
func resolveRenderer(target string) (renderer.TargetRenderer, error) {
	switch target {
	case TargetClaude:
		return claude.New(), nil
	case TargetCursor:
		return cursor.New(), nil
	case TargetAntigravity:
		return antigravity.New(), nil
	case TargetCopilot:
		return copilot.New(), nil
	case TargetGemini:
		return gemini.New(), nil
	default:
		return nil, fmt.Errorf("unsupported target %q: supported targets are \"claude\", \"cursor\", \"antigravity\", \"copilot\", \"gemini\"", target)
	}
}

// mergeResourceScope overlays project-scoped resources onto the root scope.
// For map-based resources (agents, skills, rules, MCP, workflows), project
// entries override global entries with the same ID.
func mergeResourceScope(root, project *ast.ResourceScope) {
	root.Agents = mergeMap(root.Agents, project.Agents)
	root.Skills = mergeMap(root.Skills, project.Skills)
	root.Rules = mergeMap(root.Rules, project.Rules)
	root.MCP = mergeMap(root.MCP, project.MCP)
	root.Workflows = mergeMap(root.Workflows, project.Workflows)
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
// (e.g. .claude, .cursor, .agents). Empty target defaults to ".claude" for
// backward compatibility. Unknown targets return an empty string.
func OutputDir(target string) string {
	if target == "" {
		target = TargetClaude
	}
	r, err := resolveRenderer(target)
	if err != nil {
		return ""
	}
	return r.OutputDir()
}
