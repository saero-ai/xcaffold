// Package providers holds the global ProviderManifest registry. Each provider
// sub-package calls Register in its init() function to advertise its
// capabilities. Core layers query the registry instead of importing individual
// provider packages directly, which keeps the dependency graph acyclic.
package providers

import (
	"github.com/saero-ai/xcaffold/internal/importer"
	"github.com/saero-ai/xcaffold/internal/registry"
	"github.com/saero-ai/xcaffold/internal/renderer"
)

// ProviderManifest describes a single output-target provider: the renderer and
// importer factories, supported resource kinds, valid name tokens (primary name
// plus aliases), and other metadata used by the compiler and CLI.
type ProviderManifest struct {
	// Name is the canonical provider identifier (e.g. "claude", "cursor").
	Name string

	// OutputDir is the default output directory produced by this provider
	// (e.g. ".claude", ".cursor/rules"). May be overridden by the user.
	OutputDir string

	// ValidNames contains the canonical name plus all accepted aliases.
	// ManifestFor and IsRegistered match against every entry in this slice.
	ValidNames []string

	// RequiredPasses lists optimizer pass names that must run before rendering
	// for this provider.
	RequiredPasses []string

	// DefaultBudget is the provider's recommended token/character budget for
	// project-instructions output, 0 means unlimited.
	DefaultBudget int

	// BudgetKind specifies the unit of measurement for DefaultBudget.
	// Valid values: "lines", "bytes", "items", or "" (empty = no budget).
	BudgetKind string

	// KindSupport maps resource kind identifiers to whether this provider
	// supports them (true = supported, false = known-unsupported).
	KindSupport map[string]bool

	// RootContextFile is the filename written at the project root when the
	// provider requires one (e.g. "CLAUDE.md", "GEMINI.md"). Empty for
	// providers that do not emit a root file.
	RootContextFile string

	// SubdirMap maps provider-native subdirectory names to canonical xcaffold
	// subdir names (references, scripts, assets, examples). An empty string
	// value means the subdir has no canonical mapping and its files are routed
	// to the provider-native passthrough directory.
	SubdirMap map[string]string

	// SkillMDAsReference indicates whether .md files alongside SKILL.md
	// (not in a subdirectory) should be treated as references during import.
	// Currently true for Claude Code only.
	SkillMDAsReference bool

	// RootMCPPaths lists root-level MCP config file paths (relative to project root)
	// that live outside the provider's input directory. Empty for most providers.
	RootMCPPaths []string

	// PostImportWarning is an optional warning message emitted after import.
	// Used for provider-specific notices (e.g., "KIs are app-managed").
	PostImportWarning string

	// PluginDir is the output directory name for distributable plugin packages
	// (e.g. ".claude-plugin"). Empty means this provider does not support
	// plugin export.
	PluginDir string

	// DisplayLabel is the human-readable name shown in interactive prompts
	// and CLI output (e.g. "Claude Code", "GitHub Copilot"). If empty, Name
	// is used.
	DisplayLabel string

	// CLIBinary is the name of the CLI binary on PATH used for auto-detection
	// during init and for LLM client subprocess calls. Empty means no CLI
	// binary is associated with this provider.
	CLIBinary string

	// DefaultModel is the suggested default model ID when the user has not
	// specified one. Used by xcaffold test and content generation features.
	// Empty means no default is suggested.
	DefaultModel string

	// NewRenderer is a factory that returns a fresh TargetRenderer for this
	// provider. It must be non-nil for ResolveRenderer to succeed.
	NewRenderer func() renderer.TargetRenderer

	// NewModelResolver is a factory that returns a fresh ModelResolver for this
	// provider. It may be nil for providers that do not support model selection.
	NewModelResolver func() renderer.ModelResolver

	// NewImporter is a factory that returns a fresh ProviderImporter for this
	// provider. May be nil for output-only providers.
	NewImporter func() importer.ProviderImporter

	// GlobalScanner discovers global resources for this provider in the user's
	// home directory and merges them into the shared scan result. May be nil
	// for providers with no global configuration.
	GlobalScanner func(userHome string, r *registry.GlobalScanResult)
}
