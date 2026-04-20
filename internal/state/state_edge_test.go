package state

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSourcesChanged_NoChange verifies false when hashes match.
func TestSourcesChanged_NoChange(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(src, []byte("version: \"1\"\n"), 0600))

	prev := []SourceFile{
		{Path: "project.xcf", Hash: hashFileContent(t, src)},
	}

	changed, err := SourcesChanged(prev, []string{src}, dir)
	require.NoError(t, err)
	assert.False(t, changed, "should report no change when hashes match")
}

// TestSourcesChanged_ContentModified verifies true when a file changes.
func TestSourcesChanged_ContentModified(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(src, []byte("version: \"1\"\n"), 0600))

	prev := []SourceFile{
		{Path: "project.xcf", Hash: hashFileContent(t, src)},
	}

	// Modify
	require.NoError(t, os.WriteFile(src, []byte("version: \"2\"\n"), 0600))

	changed, err := SourcesChanged(prev, []string{src}, dir)
	require.NoError(t, err)
	assert.True(t, changed, "should report change when content differs")
}

// TestSourcesChanged_FileAdded verifies true when a new source file appears.
func TestSourcesChanged_FileAdded(t *testing.T) {
	dir := t.TempDir()
	src1 := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(src1, []byte("version: \"1\"\n"), 0600))

	prev := []SourceFile{
		{Path: "project.xcf", Hash: hashFileContent(t, src1)},
	}

	src2 := filepath.Join(dir, "agents.xcf")
	require.NoError(t, os.WriteFile(src2, []byte("agents:\n"), 0600))

	changed, err := SourcesChanged(prev, []string{src1, src2}, dir)
	require.NoError(t, err)
	assert.True(t, changed, "should report change when a file is added")
}

// TestSourcesChanged_FileRemoved verifies true when a source file disappears.
func TestSourcesChanged_FileRemoved(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(src, []byte("version: \"1\"\n"), 0600))

	prev := []SourceFile{
		{Path: "project.xcf", Hash: hashFileContent(t, src)},
		{Path: "agents.xcf", Hash: "sha256:0000000000000000000000000000000000000000000000000000000000000000"},
	}

	changed, err := SourcesChanged(prev, []string{src}, dir)
	require.NoError(t, err)
	assert.True(t, changed, "should report change when a file is removed")
}

// TestSourcesChanged_EmptyPrevious verifies true on first run (no previous state).
func TestSourcesChanged_EmptyPrevious(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(src, []byte("version: \"1\"\n"), 0600))

	changed, err := SourcesChanged(nil, []string{src}, dir)
	require.NoError(t, err)
	assert.True(t, changed, "should report change when no previous source files exist")
}

func hashFileContent(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	h := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", h)
}
