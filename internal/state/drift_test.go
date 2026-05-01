package state

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func hashContent(data []byte) string {
	h := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(h[:])
}

func TestCollectDriftedFiles_NoArtifacts(t *testing.T) {
	ts := TargetState{}
	entries := CollectDriftedFiles(t.TempDir(), t.TempDir(), ts)
	assert.Empty(t, entries)
}

func TestCollectDriftedFiles_AllSynced(t *testing.T) {
	base := t.TempDir()
	outputDir := t.TempDir()

	content := []byte("# agent\n")
	relPath := "agents/dev.md"
	absPath := filepath.Join(outputDir, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(absPath), 0755))
	require.NoError(t, os.WriteFile(absPath, content, 0644))

	ts := TargetState{
		Artifacts: []Artifact{{Path: relPath, Hash: hashContent(content)}},
	}

	entries := CollectDriftedFiles(base, outputDir, ts)
	assert.Empty(t, entries)
}

func TestCollectDriftedFiles_MissingFile(t *testing.T) {
	base := t.TempDir()
	outputDir := t.TempDir()

	ts := TargetState{
		Artifacts: []Artifact{{Path: "agents/dev.md", Hash: "sha256:abc123"}},
	}

	entries := CollectDriftedFiles(base, outputDir, ts)
	require.Len(t, entries, 1)
	assert.Equal(t, "missing", entries[0].Status)
	assert.Equal(t, "agents/dev.md", entries[0].Path)
}

func TestCollectDriftedFiles_ModifiedFile(t *testing.T) {
	base := t.TempDir()
	outputDir := t.TempDir()

	relPath := "agents/dev.md"
	absPath := filepath.Join(outputDir, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(absPath), 0755))
	require.NoError(t, os.WriteFile(absPath, []byte("# modified\n"), 0644))

	ts := TargetState{
		Artifacts: []Artifact{{Path: relPath, Hash: hashContent([]byte("# original\n"))}},
	}

	entries := CollectDriftedFiles(base, outputDir, ts)
	require.Len(t, entries, 1)
	assert.Equal(t, "modified", entries[0].Status)
	assert.Equal(t, relPath, entries[0].Path)
}

func TestCollectDriftedFiles_RootPrefixMissing(t *testing.T) {
	base := t.TempDir()
	outputDir := t.TempDir()

	ts := TargetState{
		Artifacts: []Artifact{{Path: "root:.gitignore", Hash: "sha256:abc123"}},
	}

	entries := CollectDriftedFiles(base, outputDir, ts)
	require.Len(t, entries, 1)
	assert.Equal(t, "missing", entries[0].Status)
	assert.Equal(t, "root:.gitignore", entries[0].Path)
}

func TestCollectDriftedFiles_RootPrefixSynced(t *testing.T) {
	base := t.TempDir()
	outputDir := t.TempDir()

	content := []byte("# .gitignore\n")
	absPath := filepath.Join(base, ".gitignore")
	require.NoError(t, os.WriteFile(absPath, content, 0644))

	ts := TargetState{
		Artifacts: []Artifact{{Path: "root:.gitignore", Hash: hashContent(content)}},
	}

	entries := CollectDriftedFiles(base, outputDir, ts)
	assert.Empty(t, entries)
}

func TestCollectDriftedFiles_RootPrefixModified(t *testing.T) {
	base := t.TempDir()
	outputDir := t.TempDir()

	absPath := filepath.Join(base, ".gitignore")
	require.NoError(t, os.WriteFile(absPath, []byte("# current\n"), 0644))

	ts := TargetState{
		Artifacts: []Artifact{{Path: "root:.gitignore", Hash: hashContent([]byte("# old content\n"))}},
	}

	entries := CollectDriftedFiles(base, outputDir, ts)
	require.Len(t, entries, 1)
	assert.Equal(t, "modified", entries[0].Status)
	assert.Equal(t, "root:.gitignore", entries[0].Path)
}

func TestCollectDriftedFiles_MixedResults(t *testing.T) {
	base := t.TempDir()
	outputDir := t.TempDir()

	syncedContent := []byte("# synced\n")
	syncedPath := filepath.Join(outputDir, "agents/synced.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(syncedPath), 0755))
	require.NoError(t, os.WriteFile(syncedPath, syncedContent, 0644))

	modifiedPath := filepath.Join(outputDir, "agents/modified.md")
	require.NoError(t, os.WriteFile(modifiedPath, []byte("# new content\n"), 0644))

	ts := TargetState{
		Artifacts: []Artifact{
			{Path: "agents/synced.md", Hash: hashContent(syncedContent)},
			{Path: "agents/missing.md", Hash: "sha256:aaa"},
			{Path: "agents/modified.md", Hash: hashContent([]byte("# old content\n"))},
		},
	}

	entries := CollectDriftedFiles(base, outputDir, ts)
	require.Len(t, entries, 2)

	byPath := make(map[string]DriftEntry)
	for _, e := range entries {
		byPath[e.Path] = e
	}
	assert.Equal(t, "missing", byPath["agents/missing.md"].Status)
	assert.Equal(t, "modified", byPath["agents/modified.md"].Status)
}
