package codex

import (
	"github.com/saero-ai/xcaffold/internal/importer"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/providers"
)

func init() {
	providers.Register(Manifest)
	importer.Register(NewImporter())
	renderer.RegisterModelResolver("codex", NewModelResolver())
}

var Manifest = providers.ProviderManifest{
	Name:           "codex",
	OutputDir:      ".codex",
	ValidNames:     []string{"codex"},
	RequiredPasses: []string{"inline-imports"},
	DefaultBudget:  0,
	BudgetKind:     "",
	KindSupport: map[string]bool{
		"agent":       true,
		"skill":       true,
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
	DisplayLabel:       "Codex (Preview)",
	CLIBinary:          "codex",
	DefaultModel:       "gpt-5.5",
	NewRenderer:        func() renderer.TargetRenderer { return New() },
	NewModelResolver:   func() renderer.ModelResolver { return NewModelResolver() },
	NewImporter:        func() importer.ProviderImporter { return NewImporter() },
}
