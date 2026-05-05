package copilot

import (
	"github.com/saero-ai/xcaffold/internal/importer"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/providers"
)

func init() {
	providers.Register(Manifest)
	importer.Register(NewImporter())
	renderer.RegisterModelResolver("copilot", NewModelResolver())
}

// Manifest describes the GitHub Copilot provider's capabilities and factories.
var Manifest = providers.ProviderManifest{
	Name:           "copilot",
	OutputDir:      ".github",
	ValidNames:     []string{"copilot"},
	RequiredPasses: []string{"flatten-scopes", "inline-imports"},
	DefaultBudget:  4000,
	BudgetKind:     "bytes",
	KindSupport: map[string]bool{
		"agent": true,
		"skill": true,
		"rule":  true,
		"mcp":   true,
	},
	RootContextFile:    ".github/copilot-instructions.md",
	SubdirMap:          map[string]string{}, // co-located — classify by extension
	SkillMDAsReference: false,
	DisplayLabel:       "GitHub Copilot",
	CLIBinary:          "copilot",
	DefaultModel:       "gpt-4o",
	NewRenderer:        func() renderer.TargetRenderer { return New() },
	NewModelResolver:   func() renderer.ModelResolver { return NewModelResolver() },
	NewImporter:        func() importer.ProviderImporter { return NewImporter() },
}
