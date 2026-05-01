package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApply_StateFilePath_UsesXcaffoldDir(t *testing.T) {
	base := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(base, "project.xcf"), []byte("---\nkind: project\nname: test-proj\n---\n"), 0644))

	statePath := state.StateFilePath(base, "")
	assert.Equal(t, filepath.Join(base, ".xcaffold", "project.xcf.state"), statePath)

	_, err := os.Stat(filepath.Join(base, ".xcaffold"))
	assert.True(t, os.IsNotExist(err), ".xcaffold/ must not exist before apply")
}

func TestEnsureGitignoreEntry_CreatesNewFile(t *testing.T) {
	dir := t.TempDir()
	ensureGitignoreEntry(dir, ".xcaffold/")
	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)
	assert.Contains(t, string(data), ".xcaffold/")
}

func TestEnsureGitignoreEntry_AppendsToExisting(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".claude/\n"), 0644))
	ensureGitignoreEntry(dir, ".xcaffold/")
	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)
	assert.Contains(t, string(data), ".claude/")
	assert.Contains(t, string(data), ".xcaffold/")
}

func TestEnsureGitignoreEntry_NoDuplicate(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".xcaffold/\n"), 0644))
	ensureGitignoreEntry(dir, ".xcaffold/")
	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(string(data), ".xcaffold/"))
}

func TestHasDriftFromState_NilState(t *testing.T) {
	entries, err := hasDriftFromState("/nonexistent", "/nonexistent/state", "/nonexistent/base", "claude")
	assert.NoError(t, err)
	assert.Empty(t, entries)
}
