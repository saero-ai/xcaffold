package gemini

import (
	"github.com/saero-ai/xcaffold/internal/importer"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/providers"
)

func init() {
	providers.Register(Manifest)
	importer.Register(NewImporter())
}

// Manifest describes the Gemini CLI provider's capabilities and factories.
var Manifest = providers.ProviderManifest{
	Name:           "gemini",
	OutputDir:      ".gemini",
	ValidNames:     []string{"gemini"},
	RequiredPasses: []string{"inline-imports"},
	DefaultBudget:  0,
	KindSupport: map[string]bool{
		"agent":       true,
		"skill":       true,
		"rule":        true,
		"mcp":         true,
		"hook-script": true,
		"settings":    true,
	},
	RootContextFile: "GEMINI.md",
	NewRenderer:     func() renderer.TargetRenderer { return New() },
	NewImporter:     func() importer.ProviderImporter { return NewImporter() },
}
