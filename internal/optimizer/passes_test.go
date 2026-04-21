package optimizer_test

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/optimizer"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// ApplyFlattenScopes tests
// ---------------------------------------------------------------------------

func TestApplyFlattenScopes_MergesNestedRules(t *testing.T) {
	files := map[string]string{
		"rules/top-level.md":      "# Top Level",
		"rules/api/auth.md":       "# Auth Rules",
		"rules/api/validation.md": "# Validation Rules",
		"rules/api/rate-limit.md": "# Rate Limit",
		"agents/ceo.md":           "# CEO Agent",
	}

	result, notes, err := optimizer.ApplyFlattenScopes(files)
	require.NoError(t, err)

	assert.Equal(t, "# Top Level", result["rules/top-level.md"])
	assert.Contains(t, result, "rules/api.md")
	assert.NotContains(t, result, "rules/api/auth.md")
	assert.NotContains(t, result, "rules/api/validation.md")
	assert.NotContains(t, result, "rules/api/rate-limit.md")

	merged := result["rules/api.md"]
	assert.Contains(t, merged, "# Auth Rules")
	assert.Contains(t, merged, "# Rate Limit")
	assert.Contains(t, merged, "# Validation Rules")

	assert.Equal(t, "# CEO Agent", result["agents/ceo.md"])
	assert.GreaterOrEqual(t, len(notes), 1)
}

func TestApplyFlattenScopes_NoNesting_NoChange(t *testing.T) {
	files := map[string]string{
		"rules/auth.md": "# Auth",
		"agents/ceo.md": "# CEO",
	}
	result, notes, err := optimizer.ApplyFlattenScopes(files)
	require.NoError(t, err)
	assert.Equal(t, files, result)
	assert.Empty(t, notes)
}

func TestApplyFlattenScopes_SingleChildGroup_LeftUnchanged(t *testing.T) {
	// A nested rules file with only one child should not be merged into parent.md.
	files := map[string]string{
		"rules/api/only-child.md": "# Only Child",
		"rules/top.md":            "# Top",
	}
	result, notes, err := optimizer.ApplyFlattenScopes(files)
	require.NoError(t, err)
	assert.Contains(t, result, "rules/api/only-child.md")
	assert.NotContains(t, result, "rules/api.md")
	assert.Empty(t, notes)
}

// ---------------------------------------------------------------------------
// ApplyInlineImports tests
// ---------------------------------------------------------------------------

func TestApplyInlineImports_ResolvesDirective(t *testing.T) {
	files := map[string]string{
		"rules/main.md":   "# Main\n\n@import rules/shared.md\n\n## Footer",
		"rules/shared.md": "# Shared Content",
		"agents/ceo.md":   "No imports here",
	}

	result, notes, err := optimizer.ApplyInlineImports(files)
	require.NoError(t, err)

	assert.Equal(t, "# Main\n\n# Shared Content\n\n## Footer", result["rules/main.md"])
	assert.Equal(t, "# Shared Content", result["rules/shared.md"])
	assert.Equal(t, "No imports here", result["agents/ceo.md"])
	assert.GreaterOrEqual(t, len(notes), 1)
}

func TestApplyInlineImports_MissingTarget_Warning(t *testing.T) {
	files := map[string]string{
		"rules/main.md": "# Main\n\n@import rules/missing.md\n",
	}

	result, notes, err := optimizer.ApplyInlineImports(files)
	require.NoError(t, err)
	assert.Contains(t, result["rules/main.md"], "@import rules/missing.md")
	assert.GreaterOrEqual(t, len(notes), 1)
	assert.Equal(t, renderer.LevelWarning, notes[0].Level)
}

func TestApplyInlineImports_NoDirectives_NoChange(t *testing.T) {
	files := map[string]string{
		"rules/auth.md": "# Auth rules\nNo imports.",
	}
	result, notes, err := optimizer.ApplyInlineImports(files)
	require.NoError(t, err)
	assert.Equal(t, files, result)
	assert.Empty(t, notes)
}
