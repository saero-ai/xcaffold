package renderer

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveContextBody_ReturnsEmpty_NoContexts(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	got := ResolveContextBody(config, "claude")
	assert.Equal(t, "", got)
}

func TestResolveContextBody_SingleMatch_ReturnsBody(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Contexts: map[string]ast.ContextConfig{
				"claude-ctx": {
					Name:    "claude-ctx",
					Targets: []string{"claude"},
					Body:    "claude only",
				},
			},
		},
	}

	got := ResolveContextBody(config, "claude")
	assert.Equal(t, "claude only", got)

	gotGemini := ResolveContextBody(config, "gemini")
	assert.Equal(t, "", gotGemini)
}

// ── ValidateContextUniqueness ─────────────────────────────────────────────────

func TestValidateContextUniqueness_SingleContext_OK(t *testing.T) {
	contexts := map[string]ast.ContextConfig{
		"main": {Name: "main", Body: "hello"},
	}
	err := ValidateContextUniqueness(contexts, []string{"claude"})
	require.NoError(t, err)
}

func TestValidateContextUniqueness_MultipleNoOverlap_OK(t *testing.T) {
	contexts := map[string]ast.ContextConfig{
		"claude-ctx": {Name: "claude-ctx", Body: "for claude", Targets: []string{"claude"}},
		"gemini-ctx": {Name: "gemini-ctx", Body: "for gemini", Targets: []string{"gemini"}},
	}
	err := ValidateContextUniqueness(contexts, []string{"claude", "gemini"})
	require.NoError(t, err)
}

func TestValidateContextUniqueness_MultipleOverlap_NoDefault_Error(t *testing.T) {
	contexts := map[string]ast.ContextConfig{
		"ctx-a": {Name: "ctx-a", Body: "a", Targets: []string{"claude"}},
		"ctx-b": {Name: "ctx-b", Body: "b", Targets: []string{"claude"}},
	}
	err := ValidateContextUniqueness(contexts, []string{"claude"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"claude"`)
	assert.Contains(t, err.Error(), "default")
}

func TestValidateContextUniqueness_MultipleOverlap_WithDefault_OK(t *testing.T) {
	contexts := map[string]ast.ContextConfig{
		"ctx-a": {Name: "ctx-a", Body: "a", Targets: []string{"claude"}},
		"ctx-b": {Name: "ctx-b", Body: "b", Targets: []string{"claude"}, Default: true},
	}
	err := ValidateContextUniqueness(contexts, []string{"claude"})
	require.NoError(t, err)
}

func TestValidateContextUniqueness_MultipleDefaults_Error(t *testing.T) {
	contexts := map[string]ast.ContextConfig{
		"ctx-a": {Name: "ctx-a", Body: "a", Targets: []string{"claude"}, Default: true},
		"ctx-b": {Name: "ctx-b", Body: "b", Targets: []string{"claude"}, Default: true},
	}
	err := ValidateContextUniqueness(contexts, []string{"claude"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple contexts marked as default")
	assert.Contains(t, err.Error(), `"claude"`)
}

// ── ResolveContextBody ────────────────────────────────────────────────────────

// TestResolveContextBody_MultipleMatchComposesAll verifies that when multiple
// contexts match, the resolver concatenates all their bodies: default first,
// then remaining contexts in sorted name order.
func TestResolveContextBody_MultipleMatchComposesAll(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Contexts: map[string]ast.ContextConfig{
				"ctx-main":  {Name: "ctx-main", Body: "main body", Targets: []string{"claude"}},
				"ctx-extra": {Name: "ctx-extra", Body: "extra body", Targets: []string{"claude"}, Default: true},
			},
		},
	}
	got := ResolveContextBody(config, "claude")
	// Default comes first, then remaining in sorted order ("ctx-main").
	assert.Equal(t, "extra body\n\nmain body", got)
}

// TestResolveContextBody_MultipleMatch_NoDefault_SortedOrder verifies that
// when no context is marked as default, all matching bodies are joined in
// sorted name order.
func TestResolveContextBody_MultipleMatch_NoDefault_SortedOrder(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Contexts: map[string]ast.ContextConfig{
				"ctx-b": {Name: "ctx-b", Body: "body b", Targets: []string{"claude"}},
				"ctx-a": {Name: "ctx-a", Body: "body a", Targets: []string{"claude"}},
			},
		},
	}
	got := ResolveContextBody(config, "claude")
	assert.Equal(t, "body a\n\nbody b", got)
}

// TestResolveContextBody_GlobalContext_MatchesAllTargets verifies that a
// context with no Targets set is included for every target.
func TestResolveContextBody_GlobalContext_MatchesAllTargets(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Contexts: map[string]ast.ContextConfig{
				"global-ctx":  {Name: "global-ctx", Body: "global"},
				"claude-only": {Name: "claude-only", Body: "claude specific", Targets: []string{"claude"}},
			},
		},
	}
	got := ResolveContextBody(config, "claude")
	// Both match claude; "claude-only" < "global-ctx" alphabetically.
	assert.Equal(t, "claude specific\n\nglobal", got)

	gotGemini := ResolveContextBody(config, "gemini")
	assert.Equal(t, "global", gotGemini)
}
