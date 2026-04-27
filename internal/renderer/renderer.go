// Package renderer defines the TargetRenderer interface implemented by each
// output target (e.g. claude, cursor, antigravity).
package renderer

import (
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
)

// ResolveContextBody aggregates context bodies for the specified target by
// deterministically visiting all matched Contexts and concatenating their content.
func ResolveContextBody(config *ast.XcaffoldConfig, targetName string) string {
	var bodies []string

	var names []string
	for name := range config.Contexts {
		names = append(names, name)
	}
	sort.Strings(names)

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

		if applies && ctx.Body != "" {
			bodies = append(bodies, strings.TrimSpace(ctx.Body))
		}
	}

	return strings.Join(bodies, "\n\n")
}

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
	// Target returns the canonical name of this renderer (e.g. "claude").
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
