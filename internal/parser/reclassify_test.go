package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/importer"
	_ "github.com/saero-ai/xcaffold/internal/importer/claude"
)

// validAgentMarkdown is a minimal agent file that the ClaudeImporter can classify
// and extract into an AgentConfig.
const validAgentMarkdown = `---
name: foo
description: test agent
---
Do the thing.
`

// TestReclassifyExtras_GraduatesKnownFile verifies that a pre-seeded ProviderExtras
// entry for a path the importer now recognises is extracted into the typed AST and
// removed from ProviderExtras.
func TestReclassifyExtras_GraduatesKnownFile(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ProviderExtras: map[string]map[string][]byte{
			"claude": {
				"agents/foo.md": []byte(validAgentMarkdown),
			},
		},
	}

	importers := importer.DefaultImporters()
	require.NotEmpty(t, importers, "expected at least one registered importer (claude)")

	err := ReclassifyExtras(config, importers)
	require.NoError(t, err)

	// The agent should be graduated into config.Agents.
	require.Contains(t, config.Agents, "foo", "expected 'foo' agent to be graduated")
	assert.Equal(t, "foo", config.Agents["foo"].Name)

	// The extras entry for "agents/foo.md" should be removed.
	if claudeExtras, ok := config.ProviderExtras["claude"]; ok {
		assert.NotContains(t, claudeExtras, "agents/foo.md",
			"graduated file must be removed from ProviderExtras")
	}
}

// TestReclassifyExtras_KeepsUnknownFile verifies that a ProviderExtras entry for a
// path the importer does not recognise remains in ProviderExtras unchanged.
func TestReclassifyExtras_KeepsUnknownFile(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ProviderExtras: map[string]map[string][]byte{
			"claude": {
				"statusline": []byte("some opaque content"),
			},
		},
	}

	importers := importer.DefaultImporters()
	err := ReclassifyExtras(config, importers)
	require.NoError(t, err)

	require.Contains(t, config.ProviderExtras, "claude")
	require.Contains(t, config.ProviderExtras["claude"], "statusline",
		"unknown file must remain in ProviderExtras")
	assert.Equal(t, []byte("some opaque content"), config.ProviderExtras["claude"]["statusline"])
}

// TestReclassifyExtras_RemovesEmptyProviderMap verifies that once all files for a
// provider are graduated the provider key itself is deleted from ProviderExtras.
func TestReclassifyExtras_RemovesEmptyProviderMap(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ProviderExtras: map[string]map[string][]byte{
			"claude": {
				"agents/foo.md": []byte(validAgentMarkdown),
			},
		},
	}

	importers := importer.DefaultImporters()
	err := ReclassifyExtras(config, importers)
	require.NoError(t, err)

	assert.NotContains(t, config.ProviderExtras, "claude",
		"provider key must be deleted when all its extras are graduated")
}

// TestReclassifyExtras_NoOpWithoutImporter verifies that extras for a provider with
// no registered importer are left completely unchanged.
func TestReclassifyExtras_NoOpWithoutImporter(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ProviderExtras: map[string]map[string][]byte{
			"unknown-provider": {
				"some/file.md": []byte("content"),
			},
		},
	}

	importers := importer.DefaultImporters()
	err := ReclassifyExtras(config, importers)
	require.NoError(t, err)

	require.Contains(t, config.ProviderExtras, "unknown-provider")
	require.Contains(t, config.ProviderExtras["unknown-provider"], "some/file.md")
}
