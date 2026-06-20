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
		"ctx-b": {Name: "ctx-b", Body: "b", Targets: []string{"claude"}, Default: boolPtr(true)},
	}
	err := ValidateContextUniqueness(contexts, []string{"claude"})
	require.NoError(t, err)
}

func TestValidateContextUniqueness_MultipleDefaults_Error(t *testing.T) {
	contexts := map[string]ast.ContextConfig{
		"ctx-a": {Name: "ctx-a", Body: "a", Targets: []string{"claude"}, Default: boolPtr(true)},
		"ctx-b": {Name: "ctx-b", Body: "b", Targets: []string{"claude"}, Default: boolPtr(true)},
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
				"ctx-extra": {Name: "ctx-extra", Body: "extra body", Targets: []string{"claude"}, Default: boolPtr(true)},
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

// ── default: false — renderer trusts upstream selection ───────────────────────
//
// After the fix, both ValidateContextUniqueness and ResolveContextBody receive
// only the contexts the compiler has already selected. The renderer no longer
// skips contexts based on default:false — that exclusion happens in the compiler
// before blueprint.ApplyBlueprint or before validation in bare-apply mode.
// The tests below verify the post-fix renderer semantics.

// TestValidateContextUniqueness_ExplicitFalse_IncludedInCheck verifies that
// ValidateContextUniqueness treats default:false contexts just like any other
// context: two matching contexts that are both false require one to be the
// explicit default to resolve the ambiguity.
func TestValidateContextUniqueness_ExplicitFalse_IncludedInCheck(t *testing.T) {
	f := false
	contexts := map[string]ast.ContextConfig{
		"ctx-a": {Name: "ctx-a", Body: "a", Targets: []string{"claude"}, Default: &f},
		"ctx-b": {Name: "ctx-b", Body: "b", Targets: []string{"claude"}, Default: &f},
	}
	// Both have Default=false. Neither is the explicit default, so two contexts
	// match with no tie-breaker — the renderer must report ambiguity.
	err := ValidateContextUniqueness(contexts, []string{"claude"})
	require.Error(t, err, "two matching contexts with no default must be flagged as ambiguous")
	assert.Contains(t, err.Error(), `"claude"`)
}

// TestValidateContextUniqueness_MixedNilAndFalse_MultipleMatching_Error verifies
// that when the compiler passes through a mix of nil-default and false-default
// contexts (which it would only do for blueprint applies), multiple matches with
// no explicit default are flagged by the renderer.
func TestValidateContextUniqueness_MixedNilAndFalse_MultipleMatching_Error(t *testing.T) {
	f := false
	contexts := map[string]ast.ContextConfig{
		"ctx-main":    {Name: "ctx-main", Body: "main", Targets: []string{"claude"}},
		"ctx-task-01": {Name: "ctx-task-01", Body: "task 1", Targets: []string{"claude"}, Default: &f},
	}
	// Two contexts match claude with no explicit default — ambiguous.
	err := ValidateContextUniqueness(contexts, []string{"claude"})
	require.Error(t, err, "multiple matching contexts with no default must be flagged as ambiguous")
}

// TestResolveContextBody_RendersExplicitFalseContexts verifies that
// ResolveContextBody renders a context with default:false when the compiler
// has already selected it (blueprint apply path). The renderer trusts upstream
// filtering — it renders everything it receives.
func TestResolveContextBody_RendersExplicitFalseContexts(t *testing.T) {
	f := false
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Contexts: map[string]ast.ContextConfig{
				"ctx-task": {Name: "ctx-task", Body: "task body", Targets: []string{"claude"}, Default: &f},
			},
		},
	}
	// After the fix, the renderer renders the context regardless of Default value.
	result := ResolveContextBody(config, "claude")
	assert.Equal(t, "task body", result, "renderer must render default:false contexts it receives")
}

// ── ResolveContextBodies ──────────────────────────────────────────────────────

// TestResolveContextBodies_EmptyPath_ReturnsRootEntry verifies that a context
// with no Path value produces a map entry under the "" key.
func TestResolveContextBodies_EmptyPath_ReturnsRootEntry(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Contexts: map[string]ast.ContextConfig{
				"root-ctx": {
					Name:    "root-ctx",
					Targets: []string{"claude"},
					Body:    "root body",
				},
			},
		},
	}
	got := ResolveContextBodies(config, "claude")
	require.Len(t, got, 1)
	assert.Equal(t, "root body", got[""])
}

// TestResolveContextBodies_DifferentPaths_ReturnsSeparateEntries verifies that
// contexts with different Path values produce separate map entries.
func TestResolveContextBodies_DifferentPaths_ReturnsSeparateEntries(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Contexts: map[string]ast.ContextConfig{
				"root-ctx": {
					Name:    "root-ctx",
					Targets: []string{"claude"},
					Body:    "root body",
				},
				"backend-ctx": {
					Name:    "backend-ctx",
					Targets: []string{"claude"},
					Body:    "backend body",
					Path:    "backend",
				},
			},
		},
	}
	got := ResolveContextBodies(config, "claude")
	require.Len(t, got, 2)
	assert.Equal(t, "root body", got[""])
	assert.Equal(t, "backend body", got["backend"])
}

// TestResolveContextBodies_SamePath_CombinesBodies verifies that two contexts
// sharing the same target and Path are joined with "\n\n", default first.
func TestResolveContextBodies_SamePath_CombinesBodies(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Contexts: map[string]ast.ContextConfig{
				"ctx-a": {
					Name:    "ctx-a",
					Targets: []string{"claude"},
					Body:    "extra body",
					Path:    "backend",
				},
				"ctx-b": {
					Name:    "ctx-b",
					Targets: []string{"claude"},
					Body:    "default body",
					Path:    "backend",
					Default: boolPtr(true),
				},
			},
		},
	}
	got := ResolveContextBodies(config, "claude")
	require.Len(t, got, 1)
	// Default comes first, then alphabetically sorted remainder.
	assert.Equal(t, "default body\n\nextra body", got["backend"])
}

// TestResolveContextBodies_DeterministicOrder calls ResolveContextBodies 10
// times and asserts identical output on every call.
func TestResolveContextBodies_DeterministicOrder(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Contexts: map[string]ast.ContextConfig{
				"ctx-z": {Name: "ctx-z", Targets: []string{"claude"}, Body: "z body", Path: "sub"},
				"ctx-a": {Name: "ctx-a", Targets: []string{"claude"}, Body: "a body", Path: "sub"},
				"ctx-m": {Name: "ctx-m", Targets: []string{"claude"}, Body: "m body", Path: "sub", Default: boolPtr(true)},
				"root":  {Name: "root", Targets: []string{"claude"}, Body: "root body"},
			},
		},
	}
	first := ResolveContextBodies(config, "claude")
	for i := 0; i < 9; i++ {
		got := ResolveContextBodies(config, "claude")
		assert.Equal(t, first, got, "call %d returned different result", i+2)
	}
}

// boolPtr returns a pointer to the given bool value.
func boolPtr(b bool) *bool { return &b }
