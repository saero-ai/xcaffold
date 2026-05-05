package claude

import (
	"github.com/saero-ai/xcaffold/internal/importer"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/providers"
)

func init() {
	providers.Register(Manifest)
	importer.Register(NewImporter())
	renderer.RegisterModelResolver("claude", NewModelResolver())
}

// Manifest describes the Claude Code provider's capabilities and factories.
var Manifest = providers.ProviderManifest{
	Name:           "claude",
	OutputDir:      ".claude",
	ValidNames:     []string{"claude"},
	RequiredPasses: []string{},
	DefaultBudget:  200,
	BudgetKind:     "lines",
	KindSupport: map[string]bool{
		"agent":       true,
		"skill":       true,
		"rule":        true,
		"mcp":         true,
		"hook-script": true,
		"settings":    true,
		"memory":      true,
	},
	RootContextFile: "CLAUDE.md",
	SubdirMap: map[string]string{
		"references": "references",
		"scripts":    "scripts",
		"examples":   "examples",
	},
	SkillMDAsReference: true,
	RootMCPPaths:       []string{".mcp.json"},
	PluginDir:          ".claude-plugin",
	DisplayLabel:       "Claude Code",
	CLIBinary:          "claude",
	DefaultModel:       "claude-sonnet-4-6",
	NewRenderer:        func() renderer.TargetRenderer { return New() },
	NewModelResolver:   func() renderer.ModelResolver { return NewModelResolver() },
	NewImporter:        func() importer.ProviderImporter { return NewImporter() },
	GlobalScanner:      scanGlobal,
}
