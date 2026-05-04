package antigravity

import (
	"github.com/saero-ai/xcaffold/internal/importer"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/providers"
)

func init() {
	providers.Register(Manifest)
	importer.Register(NewImporter())
}

// Manifest describes the Antigravity provider's capabilities and factories.
var Manifest = providers.ProviderManifest{
	Name:           "antigravity",
	OutputDir:      ".agents",
	ValidNames:     []string{"antigravity"},
	RequiredPasses: []string{"flatten-scopes", "inline-imports"},
	DefaultBudget:  0,
	KindSupport: map[string]bool{
		"skill":    true,
		"rule":     true,
		"workflow": true,
		"mcp":      true,
	},
	RootContextFile: "GEMINI.md",
	NewRenderer:     func() renderer.TargetRenderer { return New() },
	NewImporter:     func() importer.ProviderImporter { return NewImporter() },
}
