// Package renderer defines the TargetRenderer interface implemented by each
// output target (e.g. claude, cursor, antigravity).
package renderer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
)

// MemoryOptions controls how CompileMemory writes memory entries.
type MemoryOptions struct {
	// OutputDir is the resolved path to the provider's memory directory.
	// Required for providers that write memory to disk (claude, gemini).
	OutputDir string
	// DryRun, when true, causes the renderer to compute output without writing
	// any files to disk.
	DryRun bool
}

// MemoryAwareRenderer is an optional interface that renderers may implement
// to receive the set of agent IDs that have associated memory entries. The
// orchestrator calls SetMemoryRefs before CompileAgents so that agent
// frontmatter can reference memory configuration.
type MemoryAwareRenderer interface {
	SetMemoryRefs(agentRefs map[string]bool)
}

// TargetRenderer is the contract for all output-target renderers. It declares
// per-resource compilation methods and Capabilities/Finalize hooks. The
// orchestrator calls these methods directly via Orchestrate(); there is no
// monolithic Compile/Render method on the interface.
type TargetRenderer interface {
	// Target returns the canonical name of this renderer.
	Target() string

	// OutputDir returns the base output directory for this target
	// (e.g. ".claude", ".cursor/rules").
	OutputDir() string

	// Capabilities declares which resource kinds this renderer supports.
	// The orchestrator uses this to decide whether to call a Compile* method
	// or emit a RENDERER_KIND_UNSUPPORTED fidelity note.
	Capabilities() CapabilitySet

	CompileAgents(agents map[string]ast.AgentConfig, baseDir string) (map[string]string, []FidelityNote, error)
	CompileSkills(skills map[string]ast.SkillConfig, baseDir string) (map[string]string, []FidelityNote, error)
	CompileRules(rules map[string]ast.RuleConfig, baseDir string) (map[string]string, []FidelityNote, error)
	CompileWorkflows(workflows map[string]ast.WorkflowConfig, baseDir string) (map[string]string, []FidelityNote, error)
	CompileHooks(hooks ast.HookConfig, baseDir string) (map[string]string, []FidelityNote, error)
	CompileSettings(settings ast.SettingsConfig) (map[string]string, []FidelityNote, error)
	CompileMCP(servers map[string]ast.MCPConfig) (map[string]string, []FidelityNote, error)
	CompileProjectInstructions(config *ast.XcaffoldConfig, baseDir string) (outputDirFiles map[string]string, rootFiles map[string]string, notes []FidelityNote, err error)
	CompileMemory(config *ast.XcaffoldConfig, baseDir string, opts MemoryOptions) (map[string]string, []FidelityNote, error)

	// Finalize is a post-processing pass called after all per-resource methods
	// have run (path normalization, deduplication, etc.).
	Finalize(files map[string]string, rootFiles map[string]string) (map[string]string, map[string]string, []FidelityNote, error)
}

// SlugifyFilename maps a raw memory resource ID to a canonical, path-safe filename
// by replacing slashes and spaces with underscores, downcasing the string,
// and optionally prepending "project_" if not already present.
func SlugifyFilename(raw string) string {
	slug := strings.ToLower(raw)
	slug = strings.ReplaceAll(slug, "/", "_")
	slug = strings.ReplaceAll(slug, " ", "_")
	slug = strings.ReplaceAll(slug, "\\", "_")
	if !strings.HasPrefix(slug, "project_") {
		return "project_" + slug
	}
	return slug
}

// matchedContext is an internal record used by ResolveContextBody to track
// which contexts matched a given target during resolution.
type matchedContext struct {
	name      string
	body      string
	isDefault bool
}

// ValidateContextUniqueness checks that at most one context matches each
// target in the provided list. When multiple contexts match a target, exactly
// one must have Default=true to act as the tie-breaker. Returns an error if
// the ambiguity cannot be resolved (no default, or more than one default).
func ValidateContextUniqueness(contexts map[string]ast.ContextConfig, targets []string) error {
	for _, target := range targets {
		var matching []string
		var defaults []string
		for name, ctx := range contexts {
			applies := len(ctx.Targets) == 0
			if !applies {
				for _, t := range ctx.Targets {
					if t == target {
						applies = true
						break
					}
				}
			}
			if applies {
				matching = append(matching, name)
				if ctx.Default {
					defaults = append(defaults, name)
				}
			}
		}
		sort.Strings(matching)
		sort.Strings(defaults)
		if len(matching) > 1 {
			if len(defaults) == 0 {
				return fmt.Errorf("multiple contexts target %q: [%s]; mark one as default or use --blueprint to select",
					target, strings.Join(matching, ", "))
			}
			if len(defaults) > 1 {
				return fmt.Errorf("multiple contexts marked as default for target %q: [%s]",
					target, strings.Join(defaults, ", "))
			}
		}
	}
	return nil
}

// ResolveContextBody composes bodies of all contexts that match targetName.
// When multiple contexts match, the default context (if any) is placed first,
// followed by the remaining contexts in sorted name order. All bodies are joined
// with "\n\n". When no contexts match, an empty string is returned.
//
// Filtering (e.g. blueprint selection) should happen upstream before calling
// this function. ValidateContextUniqueness may be called before rendering to
// surface configurations where disambiguation is required.
func ResolveContextBody(config *ast.XcaffoldConfig, targetName string) string {
	// Collect names in sorted order to guarantee deterministic iteration.
	names := make([]string, 0, len(config.Contexts))
	for name := range config.Contexts {
		names = append(names, name)
	}
	sort.Strings(names)

	var defaultMatch *matchedContext
	var rest []matchedContext

	for _, name := range names {
		ctx := config.Contexts[name]
		applies := len(ctx.Targets) == 0
		if !applies {
			for _, t := range ctx.Targets {
				if t == targetName {
					applies = true
					break
				}
			}
		}
		if !applies || ctx.Body == "" {
			continue
		}
		mc := matchedContext{name: name, body: ctx.Body, isDefault: ctx.Default}
		if ctx.Default && defaultMatch == nil {
			defaultMatch = &mc
		} else {
			rest = append(rest, mc)
		}
	}

	if defaultMatch == nil && len(rest) == 0 {
		return ""
	}

	// Build ordered slice: default first (if any), then rest in sorted order.
	ordered := make([]string, 0, 1+len(rest))
	if defaultMatch != nil {
		ordered = append(ordered, strings.TrimSpace(defaultMatch.body))
	}
	for _, m := range rest {
		ordered = append(ordered, strings.TrimSpace(m.body))
	}
	return strings.Join(ordered, "\n\n")
}
