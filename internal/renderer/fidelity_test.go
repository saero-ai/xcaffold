package renderer_test

import (
	"encoding/json"
	"testing"

	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFidelityNote_JSON_RoundTrip(t *testing.T) {
	note := renderer.NewNote(
		renderer.LevelWarning,
		"cursor",
		"agent",
		"code-review",
		"permissionMode",
		"AGENT_SECURITY_FIELDS_DROPPED",
		"permissionMode has no Cursor equivalent and was dropped",
		"Remove permissionMode from the cursor target override",
	)

	data, err := json.Marshal(note)
	require.NoError(t, err)

	var got renderer.FidelityNote
	require.NoError(t, json.Unmarshal(data, &got))

	assert.Equal(t, renderer.LevelWarning, got.Level)
	assert.Equal(t, "cursor", got.Target)
	assert.Equal(t, "agent", got.Kind)
	assert.Equal(t, "code-review", got.Resource)
	assert.Equal(t, "permissionMode", got.Field)
	assert.Equal(t, "AGENT_SECURITY_FIELDS_DROPPED", got.Code)
	assert.Equal(t, "permissionMode has no Cursor equivalent and was dropped", got.Reason)
	assert.Equal(t, "Remove permissionMode from the cursor target override", got.Mitigation)
}

func TestFidelityNote_JSON_OmitsEmptyField(t *testing.T) {
	note := renderer.NewNote(
		renderer.LevelInfo,
		"claude",
		"settings",
		"global",
		"",
		"FIELD_TRANSFORMED",
		"mcpServers merged from settings block",
		"",
	)

	data, err := json.Marshal(note)
	require.NoError(t, err)

	assert.NotContains(t, string(data), `"field"`)
	assert.NotContains(t, string(data), `"mitigation"`)
}
