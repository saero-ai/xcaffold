package antigravity2

import (
	"github.com/saero-ai/xcaffold/internal/importer"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/providers"
)

func init() {
	providers.Register(Manifest)
	importer.Register(NewImporter())
	renderer.RegisterModelResolver("antigravity2", NewModelResolver())
}

// Manifest describes the Antigravity 2.0 provider's capabilities and factories.
var Manifest = providers.ProviderManifest{
	Name:           "antigravity2",
	OutputDir:      ".agents",
	ValidNames:     []string{"antigravity2", "antigravity-2.0", "antigravity-2", "agy"},
	RequiredPasses: []string{"flatten-scopes", "inline-imports"},
	DefaultBudget:  16000,
	BudgetKind:     "bytes",
	KindSupport: map[string]bool{
		"agent":    true,
		"skill":    true,
		"rule":     true,
		"workflow": true,
		"mcp":      true,
		"hook":     true,
		"settings": true,
		"memory":   true,
	},
	RootContextFile: "GEMINI.md",
	SubdirMap: map[string]string{
		"examples":  "examples",
		"scripts":   "scripts",
		"resources": "assets",
	},
	SkillMDAsReference: false,
	PostImportWarning:  "",
	DisplayLabel:       "Antigravity 2.0",
	CLIBinary:          "agy",
	DefaultModel:       "gemini-3.5-flash",
	NewRenderer:        func() renderer.TargetRenderer { return New() },
	NewModelResolver:   func() renderer.ModelResolver { return NewModelResolver() },
	NewImporter:        func() importer.ProviderImporter { return NewImporter() },
	GlobalScanner:      scanGlobal,
}
