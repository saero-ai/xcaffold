package antigravity2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestModelResolver_Antigravity2_DefaultModel verifies that an empty alias
// does not resolve (returns ok=false). Default model handling is the responsibility
// of the caller.
func TestModelResolver_Antigravity2_DefaultModel(t *testing.T) {
	r := NewModelResolver()
	id, ok := r.ResolveAlias("")
	require.False(t, ok)
	assert.Equal(t, "", id)
}

// TestModelResolver_Antigravity2_KnownModels verifies that each of the 8 known
// model IDs resolves to itself (pass-through).
func TestModelResolver_Antigravity2_KnownModels(t *testing.T) {
	r := NewModelResolver()
	models := []string{
		"gemini-3.5-flash",
		"gemini-3.1-pro-high",
		"gemini-3.1-pro-low",
		"gemini-3-flash",
		"claude-sonnet-4-6-thinking",
		"claude-opus-4-6-thinking",
		"gpt-oss-120b",
		"nano-banana-2",
	}

	for _, model := range models {
		id, ok := r.ResolveAlias(model)
		require.True(t, ok, "model %q should resolve", model)
		assert.Equal(t, model, id, "model %q should pass through unchanged", model)
	}
}

// TestModelResolver_Antigravity2_Aliases verifies that common short aliases
// resolve to their canonical model IDs.
func TestModelResolver_Antigravity2_Aliases(t *testing.T) {
	r := NewModelResolver()
	testCases := map[string]string{
		"flash":           "gemini-3.5-flash",
		"pro":             "gemini-3.1-pro-high",
		"pro-low":         "gemini-3.1-pro-low",
		"sonnet-thinking": "claude-sonnet-4-6-thinking",
		"opus-thinking":   "claude-opus-4-6-thinking",
		"gpt-oss":         "gpt-oss-120b",
	}

	for alias, expected := range testCases {
		id, ok := r.ResolveAlias(alias)
		require.True(t, ok, "alias %q should resolve", alias)
		assert.Equal(t, expected, id, "alias %q should map to %q", alias, expected)
	}
}

// TestModelResolver_Antigravity2_PassThrough verifies that unknown model IDs
// are returned unchanged (no error, pass-through behavior).
func TestModelResolver_Antigravity2_PassThrough(t *testing.T) {
	r := NewModelResolver()
	unknownModels := []string{
		"custom-model-v2",
		"experimental-gemini-4",
		"user-fine-tuned-model",
		"local-model-xyz",
	}

	for _, model := range unknownModels {
		id, ok := r.ResolveAlias(model)
		require.False(t, ok, "unknown model %q should not resolve", model)
		assert.Equal(t, "", id, "unresolved model should return empty string")
	}
}

// TestModelResolver_Antigravity2_CrossVendor verifies that cross-vendor model IDs
// (e.g., claude-*, gpt-* from other vendors) are handled without modification
// and return ok=false (not in the known models map).
func TestModelResolver_Antigravity2_CrossVendor(t *testing.T) {
	r := NewModelResolver()
	crossVendorModels := []string{
		"claude-3-sonnet-20240229",
		"claude-3-opus-20240229",
		"gpt-4-turbo",
		"gpt-4o",
		"gpt-3.5-turbo",
	}

	for _, model := range crossVendorModels {
		id, ok := r.ResolveAlias(model)
		require.False(t, ok, "cross-vendor model %q should not be in known models", model)
		assert.Equal(t, "", id, "unresolved cross-vendor model should return empty string")
	}
}
