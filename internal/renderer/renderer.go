// Package renderer defines the TargetRenderer interface implemented by each
// output target (e.g. claude, cursor, antigravity).
package renderer

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
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

// TargetRenderer renders a compiled file map for a specific target environment.
// All renderers implement both the top-level Compile entry point and the
// per-resource methods used by the orchestrator.
type TargetRenderer interface {
	// Target returns the canonical name of this renderer (e.g. "claude").
	Target() string

	// OutputDir returns the base output directory for this target
	// (e.g. ".claude", ".cursor/rules").
	OutputDir() string

	// Compile translates an XcaffoldConfig into a compiler Output. It is the
	// semantic entry point for callers that hold a TargetRenderer reference.
	// Implementations delegate to Orchestrate so per-resource methods are used.
	Compile(config *ast.XcaffoldConfig, baseDir string) (*output.Output, []FidelityNote, error)

	// Render wraps a file map in an output.Output. Retained for callers that
	// have already assembled a file map and need the Output envelope.
	Render(files map[string]string) *output.Output

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
	CompileProjectInstructions(project *ast.ProjectConfig, baseDir string) (map[string]string, []FidelityNote, error)

	// Finalize is a post-processing pass called after all per-resource methods
	// have run (path normalization, deduplication, etc.).
	Finalize(files map[string]string) (map[string]string, []FidelityNote, error)
}
