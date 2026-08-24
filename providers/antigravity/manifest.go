package antigravity

import (
	"github.com/saero-ai/xcaffold/internal/importer"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/providers"
)

func init() {
	providers.Register(Manifest)
	importer.Register(NewImporter())
	renderer.RegisterModelResolver("antigravity", NewModelResolver())
}

// Manifest describes the unified Antigravity provider's capabilities and factories
// across Antigravity CLI (agy), Antigravity IDE, and Antigravity 2.0 runtime.
var Manifest = providers.ProviderManifest{
	Name:           "antigravity",
	OutputDir:      ".agents",
	ValidNames:     []string{"antigravity", "agy"},
	RequiredPasses: []string{"inline-imports"},
	DefaultBudget:  12000,
	BudgetKind:     "bytes",
	KindSupport: map[string]bool{
		"agent":    true,
		"skill":    true,
		"rule":     true,
		"workflow": true,
		"mcp":      true,
		"hook":     true,
		"settings": true,
		"memory":   false,
	},
	RootContextFile: "GEMINI.md",
	SubdirMap: map[string]string{
		"examples":   "examples",
		"scripts":    "scripts",
		"resources":  "assets",
		"references": "examples",
	},
	SkillMDAsReference: false,
	PostImportWarning:  "",
	DisplayLabel:       "Antigravity",
	CLIBinary:          "agy",
	DefaultModel:       "gemini-3.1-pro",
	NewRenderer:        func() renderer.TargetRenderer { return New() },
	NewModelResolver:   func() renderer.ModelResolver { return NewModelResolver() },
	NewImporter:        func() importer.ProviderImporter { return NewImporter() },
	GlobalScanner:      scanGlobal,
}
