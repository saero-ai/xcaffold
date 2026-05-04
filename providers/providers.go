// Package providers holds the global ProviderManifest registry. Each provider
// sub-package calls Register in its init() function to advertise its
// capabilities. Core layers query the registry instead of importing individual
// provider packages directly, which keeps the dependency graph acyclic.
package providers

import (
	"github.com/saero-ai/xcaffold/internal/importer"
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

	// KindSupport maps resource kind identifiers to whether this provider
	// supports them (true = supported, false = known-unsupported).
	KindSupport map[string]bool

	// RootContextFile is the filename written at the project root when the
	// provider requires one (e.g. "CLAUDE.md", "GEMINI.md"). Empty for
	// providers that do not emit a root file.
	RootContextFile string

	// NewRenderer is a factory that returns a fresh TargetRenderer for this
	// provider. It must be non-nil for ResolveRenderer to succeed.
	NewRenderer func() renderer.TargetRenderer

	// NewImporter is a factory that returns a fresh ProviderImporter for this
	// provider. May be nil for output-only providers.
	NewImporter func() importer.ProviderImporter
}
