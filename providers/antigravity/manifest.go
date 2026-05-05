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

// Manifest describes the Antigravity provider's capabilities and factories.
var Manifest = providers.ProviderManifest{
	Name:           "antigravity",
	OutputDir:      ".agents",
	ValidNames:     []string{"antigravity"},
	RequiredPasses: []string{"flatten-scopes", "inline-imports"},
	DefaultBudget:  12000,
	BudgetKind:     "bytes",
	KindSupport: map[string]bool{
		"skill":    true,
		"rule":     true,
		"workflow": true,
		"mcp":      true,
	},
	RootContextFile: "GEMINI.md",
	SubdirMap: map[string]string{
		"examples":  "examples",
		"scripts":   "scripts",
		"resources": "assets",
	},
	SkillMDAsReference: false,
	PostImportWarning:  "Antigravity Knowledge Items (KIs) are app-managed and cannot be imported from the filesystem",
	DisplayLabel:       "Antigravity",
	CLIBinary:          "gemini",
	DefaultModel:       "gemini-2.5-pro",
	NewRenderer:        func() renderer.TargetRenderer { return New() },
	NewModelResolver:   func() renderer.ModelResolver { return NewModelResolver() },
	NewImporter:        func() importer.ProviderImporter { return NewImporter() },
	GlobalScanner:      scanGlobal,
}
