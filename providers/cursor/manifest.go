package cursor

import (
	"github.com/saero-ai/xcaffold/internal/importer"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/providers"
)

func init() {
	providers.Register(Manifest)
	importer.Register(NewImporter())
}

// Manifest describes the Cursor provider's capabilities and factories.
var Manifest = providers.ProviderManifest{
	Name:           "cursor",
	OutputDir:      ".cursor",
	ValidNames:     []string{"cursor"},
	RequiredPasses: []string{"inline-imports"},
	DefaultBudget:  0,
	KindSupport: map[string]bool{
		"agent":       true,
		"skill":       true,
		"rule":        true,
		"mcp":         true,
		"hook-script": true,
	},
	RootContextFile: "AGENTS.md",
	NewRenderer:     func() renderer.TargetRenderer { return New() },
	NewImporter:     func() importer.ProviderImporter { return NewImporter() },
}
