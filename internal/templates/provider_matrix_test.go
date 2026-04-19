package templates

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderMatrix_Agent_AllProviders(t *testing.T) {
	all := []string{"claude", "cursor", "gemini", "copilot", "antigravity"}
	out := RenderMatrix("agent", all)

	// Must be a comment block
	require.True(t, strings.HasPrefix(out, "#"), "matrix must start with #")

	// All providers appear as column headers
	for _, p := range all {
		assert.Contains(t, out, p, "provider %q must appear in matrix header", p)
	}

	// Key fields appear as rows
	assert.Contains(t, out, "model")
	assert.Contains(t, out, "effort")
	assert.Contains(t, out, "tools")
	assert.Contains(t, out, "instructions")
}

func TestRenderMatrix_Agent_SingleProvider_OnlyOneColumn(t *testing.T) {
	out := RenderMatrix("agent", []string{"claude"})

	assert.Contains(t, out, "claude")
	assert.NotContains(t, out, "cursor")
	assert.NotContains(t, out, "gemini")
	assert.NotContains(t, out, "copilot")
	assert.NotContains(t, out, "antigravity")
}

func TestRenderMatrix_Agent_TwoProviders_ShowsDropped(t *testing.T) {
	out := RenderMatrix("agent", []string{"claude", "cursor"})

	// effort is claude-only, must show dropped for cursor
	assert.Contains(t, out, "effort")
	assert.Contains(t, out, "dropped")
}

func TestRenderMatrix_Rule_ContainsActivationRow(t *testing.T) {
	out := RenderMatrix("rule", []string{"claude", "cursor"})
	assert.Contains(t, out, "activation")
	assert.Contains(t, out, "paths")
}

func TestRenderMatrix_Settings_ContainsPermissionsRow(t *testing.T) {
	out := RenderMatrix("settings", []string{"claude", "cursor"})
	assert.Contains(t, out, "permissions")
	assert.Contains(t, out, "mcp-servers")
}

func TestRenderMatrix_UnknownKind_ReturnsEmptyString(t *testing.T) {
	out := RenderMatrix("unknown-kind", []string{"claude"})
	assert.Empty(t, out)
}
