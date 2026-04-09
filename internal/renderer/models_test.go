package renderer_test

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/stretchr/testify/assert"
)

func TestIsMappedModel_KnownAlias_Claude_ReturnsTrue(t *testing.T) {
	assert.True(t, renderer.IsMappedModel("sonnet-4", "claude"),
		"sonnet-4 is a known alias with a claude mapping")
}

func TestIsMappedModel_KnownAlias_Cursor_ReturnsFalse(t *testing.T) {
	assert.False(t, renderer.IsMappedModel("sonnet-4", "cursor"),
		"sonnet-4 has no cursor mapping — cursor does not support per-agent models")
}

func TestIsMappedModel_LiteralModelID_ReturnsFalse(t *testing.T) {
	assert.False(t, renderer.IsMappedModel("claude-sonnet-4-5", "claude"),
		"literal model IDs are not in modelAliases — IsMappedModel must return false")
}

func TestIsMappedModel_LiteralModelID_Cursor_ReturnsFalse(t *testing.T) {
	assert.False(t, renderer.IsMappedModel("claude-sonnet-4-5", "cursor"),
		"literal model ID on cursor target must return false")
}

func TestIsMappedModel_EmptyAlias_ReturnsFalse(t *testing.T) {
	assert.False(t, renderer.IsMappedModel("", "claude"),
		"empty alias is never mapped")
}

func TestIsMappedModel_Antigravity_ReturnsFalse(t *testing.T) {
	assert.False(t, renderer.IsMappedModel("sonnet-4", "antigravity"),
		"antigravity has no model support — always false")
}
