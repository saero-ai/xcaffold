package copilot

import (
	"github.com/saero-ai/xcaffold/internal/importer"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/providers"
)

func init() {
	providers.Register(Manifest)
	importer.Register(NewImporter())
}

// Manifest describes the GitHub Copilot provider's capabilities and factories.
var Manifest = providers.ProviderManifest{
	Name:           "copilot",
	OutputDir:      ".github",
	ValidNames:     []string{"copilot"},
	RequiredPasses: []string{"flatten-scopes", "inline-imports"},
	DefaultBudget:  0,
	KindSupport: map[string]bool{
		"agent": true,
		"skill": true,
		"rule":  true,
		"mcp":   true,
	},
	RootContextFile: ".github/copilot-instructions.md",
	NewRenderer:     func() renderer.TargetRenderer { return New() },
	NewImporter:     func() importer.ProviderImporter { return NewImporter() },
}
