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

func TestFidelityNote_AllCodes_NoBlanks(t *testing.T) {
	for _, code := range renderer.AllCodes() {
		assert.NotEmpty(t, code, "catalog entry must not be blank")
	}
}

func TestFidelityNote_AllCodes_Unique(t *testing.T) {
	seen := make(map[string]int)
	for _, code := range renderer.AllCodes() {
		seen[code]++
	}
	for code, count := range seen {
		assert.Equal(t, 1, count, "catalog code %q appears %d times; codes must be unique", code, count)
	}
}

// TestFidelityNote_AllCodes_ReferencedByConstant asserts every entry in
// AllCodes() matches an exported Code* constant. This catches the class
// of drift where a new constant is added but not added to the slice
// (or vice-versa), which a simple length assertion would miss.
func TestFidelityNote_AllCodes_ReferencedByConstant(t *testing.T) {
	expected := map[string]bool{
		renderer.CodeRendererKindUnsupported:             true,
		renderer.CodeFieldUnsupported:                    true,
		renderer.CodeFieldTransformed:                    true,
		renderer.CodeActivationDegraded:                  true,
		renderer.CodeInstructionsFlattened:               true,
		renderer.CodeInstructionsClosestWinsForcedConcat: true,
		renderer.CodeMemoryNoNativeTarget:                true,
		renderer.CodeWorkflowLoweredToRulePlusSkill:      true,
		renderer.CodeWorkflowLoweredToPromptFile:         true,
		renderer.CodeReservedOutputPathRejected:          true,
		renderer.CodeSettingsFieldUnsupported:            true,
		renderer.CodeHookInterpolationRequiresEnvSyntax:  true,
		renderer.CodeAgentModelUnmapped:                  true,
		renderer.CodeAgentSecurityFieldsDropped:          true,
		renderer.CodeSkillScriptsDropped:                 true,
		renderer.CodeSkillAssetsDropped:                  true,
	}

	got := make(map[string]bool)
	for _, code := range renderer.AllCodes() {
		got[code] = true
	}

	for code := range expected {
		assert.True(t, got[code], "catalog code %q is declared as a constant but not in AllCodes()", code)
	}
	for code := range got {
		assert.True(t, expected[code], "AllCodes() returns %q which is not declared as an exported constant", code)
	}
	assert.Equal(t, len(expected), len(got), "catalog size mismatch")
}
