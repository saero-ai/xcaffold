package claude

import (
	"github.com/saero-ai/xcaffold/internal/importer"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/providers"
)

func init() {
	providers.Register(Manifest)
	importer.Register(NewImporter())
}

// Manifest describes the Claude Code provider's capabilities and factories.
var Manifest = providers.ProviderManifest{
	Name:           "claude",
	OutputDir:      ".claude",
	ValidNames:     []string{"claude"},
	RequiredPasses: []string{},
	DefaultBudget:  0,
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
	NewRenderer:     func() renderer.TargetRenderer { return New() },
	NewImporter:     func() importer.ProviderImporter { return NewImporter() },
}
