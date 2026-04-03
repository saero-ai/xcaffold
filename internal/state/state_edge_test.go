package state

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/saero-ai/xcaffold/internal/compiler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerate_DeterministicOrder verifies that Generate() sorts artifacts
// alphabetically by path regardless of Go's non-deterministic map iteration.
func TestGenerate_DeterministicOrder(t *testing.T) {
	out := &compiler.Output{
		Files: map[string]string{
			"z_last.go":   "package z",
			"a_first.go":  "package a",
			"m_middle.go": "package m",
			"b_second.go": "package b",
		},
	}

	manifest := Generate(out)

	require.Len(t, manifest.Artifacts, 4)
	assert.Equal(t, "a_first.go", manifest.Artifacts[0].Path)
	assert.Equal(t, "b_second.go", manifest.Artifacts[1].Path)
	assert.Equal(t, "m_middle.go", manifest.Artifacts[2].Path)
	assert.Equal(t, "z_last.go", manifest.Artifacts[3].Path)
}

// TestGenerate_HashFormat verifies that every artifact hash is formatted as
// "sha256:" followed by exactly 64 lowercase hex characters.
func TestGenerate_HashFormat(t *testing.T) {
	out := &compiler.Output{
		Files: map[string]string{
			"file1.go": "package main\nfunc main() {}",
			"file2.go": "package lib",
		},
	}

	manifest := Generate(out)

	hashPattern := regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	for _, artifact := range manifest.Artifacts {
		assert.Truef(t, hashPattern.MatchString(artifact.Hash),
			"artifact %q has malformed hash: %q", artifact.Path, artifact.Hash)
	}
}

// TestGenerate_VersionFields verifies that Generate() populates Version,
// XcaffoldVersion, and LastApplied correctly.
func TestGenerate_VersionFields(t *testing.T) {
	before := time.Now().UTC().Truncate(time.Second)

	out := &compiler.Output{
		Files: map[string]string{
			"agent.md": "# Agent",
		},
	}

	manifest := Generate(out)

	after := time.Now().UTC().Add(time.Second)

	assert.Equal(t, lockFileVersion, manifest.Version)
	assert.Equal(t, XcaffoldVersion, manifest.XcaffoldVersion)
	require.NotEmpty(t, manifest.LastApplied, "LastApplied must not be empty")

	ts, err := time.Parse(time.RFC3339, manifest.LastApplied)
	require.NoError(t, err, "LastApplied must be a valid RFC3339 timestamp")
	assert.False(t, ts.Before(before), "LastApplied is before the test started")
	assert.False(t, ts.After(after), "LastApplied is after the test ended")
}

// TestRead_CorruptYAML verifies that Read() returns an error containing
// "failed to parse" when given a file with invalid YAML content.
func TestRead_CorruptYAML(t *testing.T) {
	dir := t.TempDir()
	badFile := filepath.Join(dir, "scaffold.lock")

	err := os.WriteFile(badFile, []byte("{{{{not yaml}}}}"), 0644)
	require.NoError(t, err)

	_, err = Read(badFile)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "failed to parse"),
		"expected error to contain \"failed to parse\", got: %q", err.Error())
}

// TestRead_EmptyFile verifies that Read() on an empty file unmarshals to a
// zero-value LockManifest (Version: 0, nil/empty Artifacts).
func TestRead_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	emptyFile := filepath.Join(dir, "scaffold.lock")

	err := os.WriteFile(emptyFile, []byte(""), 0644)
	require.NoError(t, err)

	manifest, err := Read(emptyFile)
	require.NoError(t, err)
	assert.Equal(t, 0, manifest.Version)
	assert.Empty(t, manifest.Artifacts)
}

// TestWrite_NonExistentDirectory verifies that Write() returns an error
// when the parent directory does not exist.
func TestWrite_NonExistentDirectory(t *testing.T) {
	manifest := &LockManifest{
		Version:         lockFileVersion,
		XcaffoldVersion: XcaffoldVersion,
		LastApplied:     time.Now().UTC().Format(time.RFC3339),
	}

	err := Write(manifest, "/nonexistent/dir/scaffold.lock")
	require.Error(t, err)
}

// TestReadWrite_Roundtrip_MultipleArtifacts verifies full roundtrip fidelity:
// Generate → Write → Read produces an identical manifest.
func TestReadWrite_Roundtrip_MultipleArtifacts(t *testing.T) {
	out := &compiler.Output{
		Files: map[string]string{
			".claude/agents/coder.md":   "# Coder agent",
			".claude/agents/planner.md": "# Planner agent",
			".claude/agents/qa.md":      "# QA agent",
			"cmd/main.go":               "package main",
			"internal/foo/foo.go":       "package foo",
			"internal/bar/bar.go":       "package bar",
		},
	}

	original := Generate(out)

	dir := t.TempDir()
	lockPath := filepath.Join(dir, "scaffold.lock")

	err := Write(original, lockPath)
	require.NoError(t, err)

	roundtripped, err := Read(lockPath)
	require.NoError(t, err)

	assert.Equal(t, original.Version, roundtripped.Version)
	assert.Equal(t, original.XcaffoldVersion, roundtripped.XcaffoldVersion)
	assert.Equal(t, original.ClaudeSchemaVersion, roundtripped.ClaudeSchemaVersion)
	assert.Equal(t, original.LastApplied, roundtripped.LastApplied)
	require.Len(t, roundtripped.Artifacts, len(original.Artifacts))

	for i, orig := range original.Artifacts {
		rt := roundtripped.Artifacts[i]
		assert.Equal(t, orig.Path, rt.Path, "artifact[%d] path mismatch", i)
		assert.Equal(t, orig.Hash, rt.Hash, "artifact[%d] hash mismatch", i)
	}
}
