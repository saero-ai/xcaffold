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

	// SupportsGlobalScope returns whether this renderer supports user-level
	// (global) scope compilation. Providers that return false will cause
	// a compilation error when targeted with --global.
	SupportsGlobalScope() bool

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
// (target, path) pair in the provided list. When multiple contexts match a
// pair, exactly one must have Default=true to act as the tie-breaker. Returns
// an error if the ambiguity cannot be resolved (no default, or more than one
// default).
func ValidateContextUniqueness(contexts map[string]ast.ContextConfig, targets []string) error {
	type scope struct{ target, path string }

	// group collects context names and defaults per (target, path) scope.
	type group struct {
		matching []string
		defaults []string
	}
	groups := make(map[scope]*group)

	for _, target := range targets {
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
			if !applies {
				continue
			}
			key := scope{target: target, path: ctx.Path}
			if groups[key] == nil {
				groups[key] = &group{}
			}
			groups[key].matching = append(groups[key].matching, name)
			if ctx.Default != nil && *ctx.Default {
				groups[key].defaults = append(groups[key].defaults, name)
			}
		}
	}

	// Evaluate each scope independently.
	for key, g := range groups {
		sort.Strings(g.matching)
		sort.Strings(g.defaults)
		if len(g.matching) <= 1 {
			continue
		}
		if len(g.defaults) == 0 {
			if key.path != "" {
				return fmt.Errorf("multiple contexts target %q at path %q: [%s]; mark one as default or use --blueprint to select",
					key.target, key.path, strings.Join(g.matching, ", "))
			}
			return fmt.Errorf("multiple contexts target %q: [%s]; mark one as default or use --blueprint to select",
				key.target, strings.Join(g.matching, ", "))
		}
		if len(g.defaults) > 1 {
			if key.path != "" {
				return fmt.Errorf("multiple contexts marked as default for target %q at path %q: [%s]",
					key.target, key.path, strings.Join(g.defaults, ", "))
			}
			return fmt.Errorf("multiple contexts marked as default for target %q: [%s]",
				key.target, strings.Join(g.defaults, ", "))
		}
	}
	return nil
}

// ResolveContextBodies groups all contexts that match targetName by their Path
// field and composes each group's bodies. Within each group, the default context
// (if any) is placed first, followed by the remaining contexts in sorted name
// order. Bodies within a group are joined with "\n\n". Returns a map from path
// to composed body; groups with no body content are omitted.
//
// Filtering (e.g. blueprint selection) should happen upstream before calling
// this function. ValidateContextUniqueness may be called before rendering to
// surface configurations where disambiguation is required.
func ResolveContextBodies(config *ast.XcaffoldConfig, targetName string) map[string]string {
	// Collect names in sorted order to guarantee deterministic iteration.
	names := make([]string, 0, len(config.Contexts))
	for name := range config.Contexts {
		names = append(names, name)
	}
	sort.Strings(names)

	// defaultByPath holds at most one default match per path key.
	defaultByPath := map[string]*matchedContext{}
	// restByPath holds all non-default matches per path key.
	restByPath := map[string][]matchedContext{}

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
		mc := matchedContext{name: name, body: ctx.Body, isDefault: ctx.Default != nil && *ctx.Default}
		path := ctx.Path
		if mc.isDefault && defaultByPath[path] == nil {
			defaultByPath[path] = &mc
		} else {
			restByPath[path] = append(restByPath[path], mc)
		}
	}

	// Collect all distinct path keys.
	pathSet := map[string]struct{}{}
	for p := range defaultByPath {
		pathSet[p] = struct{}{}
	}
	for p := range restByPath {
		pathSet[p] = struct{}{}
	}

	result := make(map[string]string, len(pathSet))
	for path := range pathSet {
		def := defaultByPath[path]
		rest := restByPath[path]

		ordered := make([]string, 0, 1+len(rest))
		if def != nil {
			ordered = append(ordered, strings.TrimSpace(def.body))
		}
		for _, m := range rest {
			ordered = append(ordered, strings.TrimSpace(m.body))
		}
		if body := strings.Join(ordered, "\n\n"); body != "" {
			result[path] = body
		}
	}
	return result
}

// ResolveContextBody composes bodies of all contexts that match targetName
// and have no Path set (the root path ""). When multiple contexts match, the
// default context (if any) is placed first, followed by the remaining contexts
// in sorted name order. All bodies are joined with "\n\n". When no contexts
// match, an empty string is returned.
//
// This is a convenience wrapper around ResolveContextBodies for callers that
// only need the root-path body.
func ResolveContextBody(config *ast.XcaffoldConfig, targetName string) string {
	return ResolveContextBodies(config, targetName)[""]
}
