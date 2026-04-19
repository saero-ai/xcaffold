package main

import (
	"sort"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/compiler"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApplyProviderExtras_RestoresSameProviderExtras verifies that extras
// whose provider key matches the active target are added to out.Files.
func TestApplyProviderExtras_RestoresSameProviderExtras(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ProviderExtras: map[string]map[string][]byte{
			"claude": {
				"statusline": []byte("some opaque content"),
			},
		},
	}
	out := &compiler.Output{Files: map[string]string{}}
	var notes []renderer.FidelityNote

	notes = applyProviderExtras(config, out, "claude", notes)

	require.Contains(t, out.Files, "statusline", "same-provider extra must be added to output files")
	assert.Equal(t, "some opaque content", out.Files["statusline"])
	assert.Empty(t, notes, "same-provider extras must not produce fidelity notes")
}

// TestApplyProviderExtras_EmitsFidelityNoteForCrossProviderExtras verifies that
// extras from a different provider are skipped and a warning note is emitted.
func TestApplyProviderExtras_EmitsFidelityNoteForCrossProviderExtras(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ProviderExtras: map[string]map[string][]byte{
			"cursor": {
				".cursor/rules/some-rule.md": []byte("cursor-specific content"),
			},
		},
	}
	out := &compiler.Output{Files: map[string]string{}}
	var notes []renderer.FidelityNote

	notes = applyProviderExtras(config, out, "claude", notes)

	assert.NotContains(t, out.Files, ".cursor/rules/some-rule.md", "cross-provider extra must NOT be added to output files")
	require.Len(t, notes, 1)
	assert.Equal(t, renderer.LevelWarning, notes[0].Level)
	assert.Equal(t, "claude", notes[0].Target)
	assert.Equal(t, "extras", notes[0].Kind)
	assert.Equal(t, ".cursor/rules/some-rule.md", notes[0].Resource)
	assert.Equal(t, "provider-extras-skipped", notes[0].Code)
}

// TestApplyProviderExtras_MultiProvider verifies correct split when both
// same-provider and cross-provider extras exist simultaneously.
func TestApplyProviderExtras_MultiProvider(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ProviderExtras: map[string]map[string][]byte{
			"claude": {
				"statusline": []byte("claude data"),
			},
			"cursor": {
				".cursor/rules/rule.md": []byte("cursor data"),
			},
		},
	}
	out := &compiler.Output{Files: map[string]string{}}
	var notes []renderer.FidelityNote

	notes = applyProviderExtras(config, out, "claude", notes)

	require.Contains(t, out.Files, "statusline")
	assert.NotContains(t, out.Files, ".cursor/rules/rule.md")
	require.Len(t, notes, 1)
	assert.Equal(t, "provider-extras-skipped", notes[0].Code)
}

// TestApplyProviderExtras_NilExtras_NoOp verifies no panic and no side effects
// when ProviderExtras is nil.
func TestApplyProviderExtras_NilExtras_NoOp(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	out := &compiler.Output{Files: map[string]string{}}
	var notes []renderer.FidelityNote

	notes = applyProviderExtras(config, out, "claude", notes)

	assert.Empty(t, out.Files)
	assert.Empty(t, notes)
}

// TestApplyProviderExtras_RejectsPathTraversal verifies that same-provider
// extras with path-traversal sequences or absolute paths are rejected and a
// warning FidelityNote is emitted for each unsafe path.
func TestApplyProviderExtras_RejectsPathTraversal(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ProviderExtras: map[string]map[string][]byte{
			"claude": {
				"../../etc/passwd":  []byte("malicious"),
				"agents/legit.md":   []byte("safe"),
				"/absolute/path.md": []byte("also bad"),
			},
		},
	}
	out := &compiler.Output{Files: make(map[string]string)}
	var notes []renderer.FidelityNote

	notes = applyProviderExtras(config, out, "claude", notes)

	assert.NotContains(t, out.Files, "../../etc/passwd", "traversal path must not be written")
	assert.NotContains(t, out.Files, "/absolute/path.md", "absolute path must not be written")
	require.Contains(t, out.Files, "agents/legit.md", "safe relative path must be written")
	assert.Equal(t, "safe", out.Files["agents/legit.md"])

	// Two unsafe paths must each produce exactly one warning note.
	require.Len(t, notes, 2)
	codes := make([]string, len(notes))
	for i, n := range notes {
		codes[i] = n.Code
		assert.Equal(t, renderer.LevelWarning, n.Level)
		assert.Equal(t, "claude", n.Target)
		assert.Equal(t, "extras", n.Kind)
	}
	assert.Contains(t, codes, "provider-extras-path-unsafe")
	assert.Contains(t, codes, "provider-extras-path-unsafe")
}

// TestApplyProviderExtras_NoteOrderIsDeterministic verifies that multiple
// cross-provider extras emit notes in sorted path order.
func TestApplyProviderExtras_NoteOrderIsDeterministic(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ProviderExtras: map[string]map[string][]byte{
			"cursor": {
				"z-file": []byte("z"),
				"a-file": []byte("a"),
				"m-file": []byte("m"),
			},
		},
	}
	out := &compiler.Output{Files: map[string]string{}}
	var notes []renderer.FidelityNote

	notes = applyProviderExtras(config, out, "claude", notes)

	require.Len(t, notes, 3)
	resources := make([]string, len(notes))
	for i, n := range notes {
		resources[i] = n.Resource
	}
	require.True(t, sort.StringsAreSorted(resources), "fidelity notes must be emitted in sorted resource order, got: %v", resources)
}
