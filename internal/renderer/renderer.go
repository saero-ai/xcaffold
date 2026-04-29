// Package renderer defines the TargetRenderer interface implemented by each
// output target (e.g. claude, cursor, antigravity).
package renderer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"gopkg.in/yaml.v3"
)

// ResolveInstructionsContent returns inline instructions or reads InstructionsFile
// relative to baseDir. Returns an empty string on any read error or when both
// are empty. This is the shared low-level helper used by all renderers; it
// intentionally swallows file read errors (missing files are treated as empty).
func ResolveInstructionsContent(inline, file, baseDir string) string {
	if inline != "" {
		return inline
	}
	if file == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(baseDir, file))
	if err != nil {
		return ""
	}
	if strings.HasSuffix(file, ".xcf") {
		return extractXCFInstructions(data)
	}
	return string(data)
}

// extractXCFInstructions parses a .xcf instructions sidecar and returns the
// value of the top-level "instructions" field. If the YAML is malformed or
// the field is absent, the raw bytes are returned unchanged.
func extractXCFInstructions(data []byte) string {
	var doc struct {
		Instructions string `yaml:"instructions"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return string(data)
	}
	if doc.Instructions == "" {
		return string(data)
	}
	return doc.Instructions
}

// ResolveScopeContent returns the effective content for an InstructionsScope,
// preferring a provider-specific variant when one is declared under
// scope.Variants[provider]. Falls back to the scope's own Instructions /
// InstructionsFile pair.
func ResolveScopeContent(scope ast.InstructionsScope, provider, baseDir string) string {
	if v, ok := scope.Variants[provider]; ok {
		return ResolveInstructionsContent("", v.InstructionsFile, baseDir)
	}
	return ResolveInstructionsContent(scope.Instructions, scope.InstructionsFile, baseDir)
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
	CompileProjectInstructions(project *ast.ProjectConfig, baseDir string) (outputDirFiles map[string]string, rootFiles map[string]string, notes []FidelityNote, err error)
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

// ResolveContextBody returns the body for the single context that applies to
// targetName. When multiple contexts match, the one with Default=true is
// selected. When no contexts match, an empty string is returned.
//
// Callers should invoke ValidateContextUniqueness before rendering to surface
// ambiguous configurations as actionable errors before output is written.
func ResolveContextBody(config *ast.XcaffoldConfig, targetName string) string {
	// Collect names in sorted order to guarantee deterministic iteration.
	names := make([]string, 0, len(config.Contexts))
	for name := range config.Contexts {
		names = append(names, name)
	}
	sort.Strings(names)

	var matching []matchedContext
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
			matching = append(matching, matchedContext{name: name, body: ctx.Body, isDefault: ctx.Default})
		}
	}

	if len(matching) == 0 {
		return ""
	}
	if len(matching) == 1 {
		return strings.TrimSpace(matching[0].body)
	}
	// Multiple match — select the one marked as default.
	for _, m := range matching {
		if m.isDefault {
			return strings.TrimSpace(m.body)
		}
	}
	// No default found — ValidateContextUniqueness should have caught this.
	// Fall back to the first sorted match to avoid an empty result.
	return strings.TrimSpace(matching[0].body)
}
