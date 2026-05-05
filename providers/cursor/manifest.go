package cursor

import (
	"github.com/saero-ai/xcaffold/internal/importer"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/providers"
)

func init() {
	providers.Register(Manifest)
	importer.Register(NewImporter())
	renderer.RegisterModelResolver("cursor", NewModelResolver())
}

// Manifest describes the Cursor provider's capabilities and factories.
var Manifest = providers.ProviderManifest{
	Name:           "cursor",
	OutputDir:      ".cursor",
	ValidNames:     []string{"cursor"},
	RequiredPasses: []string{"inline-imports"},
	DefaultBudget:  500,
	BudgetKind:     "lines",
	KindSupport: map[string]bool{
		"agent":       true,
		"skill":       true,
		"rule":        true,
		"mcp":         true,
		"hook-script": true,
	},
	RootContextFile: "AGENTS.md",
	SubdirMap: map[string]string{
		"references": "references",
		"scripts":    "scripts",
		"assets":     "assets",
	},
	SkillMDAsReference: false,
	DisplayLabel:       "Cursor",
	CLIBinary:          "cursor",
	DefaultModel:       "cursor-default",
	NewRenderer:        func() renderer.TargetRenderer { return New() },
	NewModelResolver:   func() renderer.ModelResolver { return NewModelResolver() },
	NewImporter:        func() importer.ProviderImporter { return NewImporter() },
}
