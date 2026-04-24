// Package renderer defines the TargetRenderer interface implemented by each
// output target (e.g. claude, cursor, antigravity).
package renderer

import (
	"os"
	"path/filepath"
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
	// PriorHashes maps memory entry names to the SHA-256 hash recorded on the
	// last apply. Used by the claude renderer for drift detection.
	PriorHashes map[string]string
	// Reseed instructs tracked and seed-once renderers to overwrite existing
	// memory files regardless of lifecycle or drift state.
	Reseed bool
	// DryRun, when true, causes the renderer to compute output without writing
	// any files to disk.
	DryRun bool
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
