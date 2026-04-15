package main

import (
	"bytes"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/stretchr/testify/assert"
)

func TestPrintFidelityNotes_Suppressed(t *testing.T) {
	var buf bytes.Buffer

	notes := []renderer.FidelityNote{
		renderer.NewNote(
			renderer.LevelWarning,
			"cursor",
			"agent",
			"reviewer",
			"permissionMode",
			renderer.CodeAgentSecurityFieldsDropped,
			"permissionMode dropped",
			"",
		),
	}

	suppressed := map[string]bool{"reviewer": true}
	printed := printFidelityNotes(&buf, notes, suppressed, false)
	assert.Equal(t, 0, printed, "suppressed note must not be printed")
	assert.Empty(t, buf.String())
}

func TestPrintFidelityNotes_StrictMode_PromotesWarningToError(t *testing.T) {
	var buf bytes.Buffer

	notes := []renderer.FidelityNote{
		renderer.NewNote(
			renderer.LevelWarning,
			"cursor",
			"agent",
			"analyst",
			"isolation",
			renderer.CodeAgentSecurityFieldsDropped,
			"isolation dropped",
			"",
		),
	}

	printed := printFidelityNotes(&buf, notes, nil, true)
	assert.Equal(t, 1, printed)
	assert.Contains(t, buf.String(), "ERROR")
}

func TestPrintFidelityNotes_WarnMode_DefaultPrefix(t *testing.T) {
	var buf bytes.Buffer

	notes := []renderer.FidelityNote{
		renderer.NewNote(
			renderer.LevelWarning,
			"cursor", "settings", "global", "permissions",
			renderer.CodeSettingsFieldUnsupported,
			"permissions dropped",
			"",
		),
	}

	printed := printFidelityNotes(&buf, notes, nil, false)
	assert.Equal(t, 1, printed)
	assert.Contains(t, buf.String(), "WARNING (cursor):")
}

func TestBuildSuppressedResourcesMap_PicksUpAgentOverride(t *testing.T) {
	suppress := true
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"quiet": {
					Targets: map[string]ast.TargetOverride{
						"cursor": {SuppressFidelityWarnings: &suppress},
					},
				},
				"loud": {
					Targets: map[string]ast.TargetOverride{
						"cursor": {SuppressFidelityWarnings: nil},
					},
				},
			},
		},
	}

	got := buildSuppressedResourcesMap(config, "cursor")
	assert.True(t, got["quiet"])
	assert.False(t, got["loud"])
}
