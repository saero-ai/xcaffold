package state

import (
	"crypto/sha256"
	"fmt"
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

	err := os.WriteFile(badFile, []byte("{{{{not yaml}}}}"), 0600)
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

	err := os.WriteFile(emptyFile, []byte(""), 0600)
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

// TestSourcesChanged_NoChange verifies false when hashes match.
func TestSourcesChanged_NoChange(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "scaffold.xcf")
	require.NoError(t, os.WriteFile(src, []byte("version: \"1\"\n"), 0600))

	prev := []SourceFile{
		{Path: "scaffold.xcf", Hash: hashFileContent(t, src)},
	}

	changed, err := SourcesChanged(prev, []string{src}, dir)
	require.NoError(t, err)
	assert.False(t, changed, "should report no change when hashes match")
}

// TestSourcesChanged_ContentModified verifies true when a file changes.
func TestSourcesChanged_ContentModified(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "scaffold.xcf")
	require.NoError(t, os.WriteFile(src, []byte("version: \"1\"\n"), 0600))

	prev := []SourceFile{
		{Path: "scaffold.xcf", Hash: hashFileContent(t, src)},
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
	src1 := filepath.Join(dir, "scaffold.xcf")
	require.NoError(t, os.WriteFile(src1, []byte("version: \"1\"\n"), 0600))

	prev := []SourceFile{
		{Path: "scaffold.xcf", Hash: hashFileContent(t, src1)},
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
	src := filepath.Join(dir, "scaffold.xcf")
	require.NoError(t, os.WriteFile(src, []byte("version: \"1\"\n"), 0600))

	prev := []SourceFile{
		{Path: "scaffold.xcf", Hash: hashFileContent(t, src)},
		{Path: "agents.xcf", Hash: "sha256:0000000000000000000000000000000000000000000000000000000000000000"},
	}

	changed, err := SourcesChanged(prev, []string{src}, dir)
	require.NoError(t, err)
	assert.True(t, changed, "should report change when a file is removed")
}

// TestSourcesChanged_EmptyPrevious verifies true on first run (no previous lock).
func TestSourcesChanged_EmptyPrevious(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "scaffold.xcf")
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

// TestMigrateLegacyLock_RenamesFile verifies that scaffold.lock is renamed
// to scaffold.claude.lock when the target-specific lock does not exist.
func TestMigrateLegacyLock_RenamesFile(t *testing.T) {
	dir := t.TempDir()
	legacyLock := filepath.Join(dir, "scaffold.lock")
	require.NoError(t, os.WriteFile(legacyLock, []byte("version: 1\n"), 0600))

	migrated, err := MigrateLegacyLock(legacyLock, "claude")
	require.NoError(t, err)
	assert.True(t, migrated)

	// Old file should be gone
	_, err = os.Stat(legacyLock)
	assert.True(t, os.IsNotExist(err), "legacy lock should be deleted")

	// New file should exist
	newLock := filepath.Join(dir, "scaffold.claude.lock")
	_, err = os.Stat(newLock)
	assert.NoError(t, err, "target-specific lock should exist")
}

// TestMigrateLegacyLock_SkipsWhenTargetExists verifies no migration when
// the target-specific lock already exists.
func TestMigrateLegacyLock_SkipsWhenTargetExists(t *testing.T) {
	dir := t.TempDir()
	legacyLock := filepath.Join(dir, "scaffold.lock")
	targetLock := filepath.Join(dir, "scaffold.claude.lock")
	require.NoError(t, os.WriteFile(legacyLock, []byte("version: 1\n"), 0600))
	require.NoError(t, os.WriteFile(targetLock, []byte("version: 2\n"), 0600))

	migrated, err := MigrateLegacyLock(legacyLock, "claude")
	require.NoError(t, err)
	assert.False(t, migrated, "should not migrate when target lock already exists")

	// Both files should still exist
	_, err = os.Stat(legacyLock)
	assert.NoError(t, err)
	_, err = os.Stat(targetLock)
	assert.NoError(t, err)
}

// TestMigrateLegacyLock_NoLegacy verifies no error when legacy lock doesn't exist.
func TestMigrateLegacyLock_NoLegacy(t *testing.T) {
	dir := t.TempDir()
	legacyLock := filepath.Join(dir, "scaffold.lock")

	migrated, err := MigrateLegacyLock(legacyLock, "claude")
	require.NoError(t, err)
	assert.False(t, migrated)
}

// TestFindOrphans_DetectsRemoved verifies that files in the old lock but not
// in the new output are returned as orphans.
func TestFindOrphans_DetectsRemoved(t *testing.T) {
	old := &LockManifest{
		Artifacts: []Artifact{
			{Path: "agents/dev.md", Hash: "sha256:aaa"},
			{Path: "agents/qa.md", Hash: "sha256:bbb"},
			{Path: "rules/security.md", Hash: "sha256:ccc"},
		},
	}
	newFiles := map[string]string{
		"agents/dev.md":     "# Dev",
		"rules/security.md": "# Security",
	}

	orphans := FindOrphans(old, newFiles)
	require.Len(t, orphans, 1)
	assert.Equal(t, "agents/qa.md", orphans[0])
}

// TestFindOrphans_NoOrphans verifies empty result when all old files are present.
func TestFindOrphans_NoOrphans(t *testing.T) {
	old := &LockManifest{
		Artifacts: []Artifact{
			{Path: "agents/dev.md", Hash: "sha256:aaa"},
		},
	}
	newFiles := map[string]string{
		"agents/dev.md": "# Dev",
	}

	orphans := FindOrphans(old, newFiles)
	assert.Empty(t, orphans)
}

// TestFindOrphans_NilOldManifest verifies empty result when there is no old lock.
func TestFindOrphans_NilOldManifest(t *testing.T) {
	newFiles := map[string]string{"agents/dev.md": "# Dev"}
	orphans := FindOrphans(nil, newFiles)
	assert.Empty(t, orphans)
}

// TestFindOrphans_SortedOutput verifies orphans are returned in sorted order.
func TestFindOrphans_SortedOutput(t *testing.T) {
	old := &LockManifest{
		Artifacts: []Artifact{
			{Path: "z.md", Hash: "sha256:aaa"},
			{Path: "a.md", Hash: "sha256:bbb"},
			{Path: "m.md", Hash: "sha256:ccc"},
		},
	}
	newFiles := map[string]string{} // all removed

	orphans := FindOrphans(old, newFiles)
	require.Len(t, orphans, 3)
	assert.Equal(t, "a.md", orphans[0])
	assert.Equal(t, "m.md", orphans[1])
	assert.Equal(t, "z.md", orphans[2])
}

// TestGenerate_IncludesSourceHashes verifies that Generate populates SourceFiles
// with hashes of the provided source file paths.
func TestGenerate_IncludesSourceHashes(t *testing.T) {
	dir := t.TempDir()

	// Create two source xcf files
	src1 := filepath.Join(dir, "scaffold.xcf")
	require.NoError(t, os.WriteFile(src1, []byte("version: \"1\"\n"), 0600))
	src2 := filepath.Join(dir, "agents.xcf")
	require.NoError(t, os.WriteFile(src2, []byte("agents:\n  dev:\n    name: Dev\n"), 0600))

	out := &compiler.Output{
		Files: map[string]string{
			"agents/dev.md": "# Dev",
		},
	}

	opts := GenerateOpts{
		Target:      "claude",
		Scope:       "project",
		ConfigDir:   ".",
		SourceFiles: []string{src1, src2},
		BaseDir:     dir,
	}

	manifest := GenerateWithOpts(out, opts)

	assert.Equal(t, "claude", manifest.Target)
	assert.Equal(t, "project", manifest.Scope)
	assert.Equal(t, ".", manifest.ConfigDir)
	require.Len(t, manifest.SourceFiles, 2)

	// SourceFiles should be sorted by path
	assert.Equal(t, "agents.xcf", manifest.SourceFiles[0].Path)
	assert.Equal(t, "scaffold.xcf", manifest.SourceFiles[1].Path)

	hashPattern := regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	for _, sf := range manifest.SourceFiles {
		assert.Truef(t, hashPattern.MatchString(sf.Hash),
			"source file %q has malformed hash: %q", sf.Path, sf.Hash)
	}
}

// TestGenerate_SourceHashChangesOnContentChange verifies that modifying a source
// file changes its hash in the manifest.
func TestGenerate_SourceHashChangesOnContentChange(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "scaffold.xcf")
	require.NoError(t, os.WriteFile(src, []byte("version: \"1\"\n"), 0600))

	out := &compiler.Output{Files: map[string]string{"x.md": "x"}}
	opts := GenerateOpts{
		Target:      "claude",
		Scope:       "project",
		ConfigDir:   ".",
		SourceFiles: []string{src},
		BaseDir:     dir,
	}

	m1 := GenerateWithOpts(out, opts)
	hash1 := m1.SourceFiles[0].Hash

	// Modify the source file
	require.NoError(t, os.WriteFile(src, []byte("version: \"2\"\n"), 0600))

	m2 := GenerateWithOpts(out, opts)
	hash2 := m2.SourceFiles[0].Hash

	assert.NotEqual(t, hash1, hash2, "hash should change when source content changes")
}

// TestGenerate_LockVersionBumped verifies the lock file version is 2.
func TestGenerate_LockVersionBumped(t *testing.T) {
	out := &compiler.Output{Files: map[string]string{"x.md": "x"}}
	opts := GenerateOpts{
		Target:    "claude",
		Scope:     "project",
		ConfigDir: ".",
	}
	manifest := GenerateWithOpts(out, opts)
	assert.Equal(t, 2, manifest.Version)
}

// TestReadWrite_Roundtrip_WithSourceFiles verifies full roundtrip fidelity
// including the new SourceFiles, Target, Scope, and ConfigDir fields.
func TestReadWrite_Roundtrip_WithSourceFiles(t *testing.T) {
	manifest := &LockManifest{
		Version:             2,
		LastApplied:         time.Now().UTC().Format(time.RFC3339),
		XcaffoldVersion:     XcaffoldVersion,
		ClaudeSchemaVersion: claudeSchemaVersion,
		Target:              "claude",
		Scope:               "project",
		ConfigDir:           ".",
		SourceFiles: []SourceFile{
			{Path: "agents.xcf", Hash: "sha256:abc123"},
			{Path: "scaffold.xcf", Hash: "sha256:def456"},
		},
		Artifacts: []Artifact{
			{Path: "agents/dev.md", Hash: "sha256:789abc"},
		},
	}

	dir := t.TempDir()
	lockFile := filepath.Join(dir, "scaffold.claude.lock")

	err := Write(manifest, lockFile)
	require.NoError(t, err)

	roundtripped, err := Read(lockFile)
	require.NoError(t, err)

	assert.Equal(t, manifest.Target, roundtripped.Target)
	assert.Equal(t, manifest.Scope, roundtripped.Scope)
	assert.Equal(t, manifest.ConfigDir, roundtripped.ConfigDir)
	require.Len(t, roundtripped.SourceFiles, 2)
	assert.Equal(t, "agents.xcf", roundtripped.SourceFiles[0].Path)
	assert.Equal(t, "scaffold.xcf", roundtripped.SourceFiles[1].Path)
}
