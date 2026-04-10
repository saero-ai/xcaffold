package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/saero-ai/xcaffold/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimalXCF is a minimal valid scaffold.xcf for apply tests.
const minimalXCF = `version: "1"
project:
  name: apply-test
`

func TestApplyScope_Project(t *testing.T) {
	dir := t.TempDir()
	xcf := filepath.Join(dir, "scaffold.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(minimalXCF), 0600))

	claudeDirPath := filepath.Join(dir, ".claude")
	lock := filepath.Join(dir, "scaffold.lock")

	err := applyScope(xcf, claudeDirPath, lock, "project")
	require.NoError(t, err)

	targetLock := filepath.Join(dir, "scaffold.claude.lock")
	_, err = os.Stat(targetLock)
	assert.NoError(t, err, "scaffold.lock should exist after applyScope")

	// .claude/ subdirectories must exist.
	for _, sub := range []string{"agents", "skills", "rules"} {
		_, err = os.Stat(filepath.Join(claudeDirPath, sub))
		assert.NoError(t, err, ".claude/%s should exist", sub)
	}
}

func TestApplyScope_MissingXCF(t *testing.T) {
	dir := t.TempDir()
	claudeDirPath := filepath.Join(dir, ".claude")
	lock := filepath.Join(dir, "scaffold.lock")

	err := applyScope(filepath.Join(dir, "nonexistent.xcf"), claudeDirPath, lock, "project")
	assert.Error(t, err, "should fail when xcf file does not exist")
}

func TestRunApply_ScopeProject(t *testing.T) {
	dir := t.TempDir()
	xcf := filepath.Join(dir, "scaffold.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(minimalXCF), 0600))

	// Set package-level path vars used by runApply.
	xcfPath = xcf
	claudeDir = filepath.Join(dir, ".claude")
	lockPath = filepath.Join(dir, "scaffold.lock")
	globalFlag = false

	err := runApply(nil, nil)
	require.NoError(t, err)

	targetLock := filepath.Join(dir, "scaffold.claude.lock")
	_, err = os.Stat(targetLock)
	assert.NoError(t, err, "scaffold.lock should be written for project scope")
}

func TestRunApply_ScopeGlobal(t *testing.T) {
	dir := t.TempDir()
	xcf := filepath.Join(dir, "global.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(minimalXCF), 0600))

	globalXcfPath = xcf
	globalXcfHome = filepath.Join(dir, ".claude")
	globalLockPath = filepath.Join(dir, "scaffold.lock")
	globalFlag = true
	defer func() { globalFlag = false }()

	err := runApply(nil, nil)
	require.NoError(t, err)

	targetLock := filepath.Join(dir, "scaffold.claude.lock")
	_, err = os.Stat(targetLock)
	assert.NoError(t, err, "scaffold.lock should be written for global scope")
}

func TestRunApply_GlobalFlagFalse_CompilesProject(t *testing.T) {
	dir := t.TempDir()
	xcf := filepath.Join(dir, "scaffold.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(minimalXCF), 0600))

	xcfPath = xcf
	claudeDir = filepath.Join(dir, ".claude")
	lockPath = filepath.Join(dir, "scaffold.lock")
	globalFlag = false
	targetFlag = targetClaude

	err := runApply(nil, nil)
	require.NoError(t, err)

	targetLock := filepath.Join(dir, "scaffold.claude.lock")
	_, err = os.Stat(targetLock)
	assert.NoError(t, err, "project lock should be written when globalFlag is false")
}

func TestApplyScope_SkipsWhenSourceUnchanged(t *testing.T) {
	dir := t.TempDir()
	xcf := filepath.Join(dir, "scaffold.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(minimalXCF), 0600))

	claudeDirPath := filepath.Join(dir, ".claude")
	lock := filepath.Join(dir, "scaffold.lock")

	// First apply — should compile
	applyForce = false
	targetFlag = targetClaude
	err := applyScope(xcf, claudeDirPath, lock, "project")
	require.NoError(t, err)

	targetLock := filepath.Join(dir, "scaffold.claude.lock")
	_, err = os.Stat(targetLock)
	require.NoError(t, err, "lock file should exist after first apply")

	// Read first lock to get timestamp
	m1, err := state.Read(targetLock)
	require.NoError(t, err)
	ts1 := m1.LastApplied

	// Second apply — should skip (same sources)
	err = applyScope(xcf, claudeDirPath, lock, "project")
	require.NoError(t, err)

	// Lock timestamp should NOT change (compilation was skipped)
	m2, err := state.Read(targetLock)
	require.NoError(t, err)
	assert.Equal(t, ts1, m2.LastApplied, "timestamp should not change when sources are unchanged")
}

func TestApplyScope_RecompilesWhenSourceChanged(t *testing.T) {
	dir := t.TempDir()
	xcf := filepath.Join(dir, "scaffold.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(minimalXCF), 0600))

	claudeDirPath := filepath.Join(dir, ".claude")
	lock := filepath.Join(dir, "scaffold.lock")

	applyForce = false
	targetFlag = targetClaude

	// First apply
	err := applyScope(xcf, claudeDirPath, lock, "project")
	require.NoError(t, err)

	targetLock := filepath.Join(dir, "scaffold.claude.lock")
	m1, err := state.Read(targetLock)
	require.NoError(t, err)

	// Modify source
	modifiedXCF := `version: "1"
project:
  name: apply-test-modified
`
	require.NoError(t, os.WriteFile(xcf, []byte(modifiedXCF), 0600))
	time.Sleep(1 * time.Second)

	// Second apply — should recompile
	err = applyScope(xcf, claudeDirPath, lock, "project")
	require.NoError(t, err)

	m2, err := state.Read(targetLock)
	require.NoError(t, err)
	assert.NotEqual(t, m1.LastApplied, m2.LastApplied, "timestamp should change when sources are modified")
}

func TestApplyScope_ForceRecompiles(t *testing.T) {
	dir := t.TempDir()
	xcf := filepath.Join(dir, "scaffold.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(minimalXCF), 0600))

	claudeDirPath := filepath.Join(dir, ".claude")
	lock := filepath.Join(dir, "scaffold.lock")

	targetFlag = targetClaude

	// First apply (non-force)
	applyForce = false
	err := applyScope(xcf, claudeDirPath, lock, "project")
	require.NoError(t, err)

	targetLock := filepath.Join(dir, "scaffold.claude.lock")
	m1, err := state.Read(targetLock)
	require.NoError(t, err)

	// Second apply with --force — should recompile despite no changes
	applyForce = true
	time.Sleep(1 * time.Second)
	err = applyScope(xcf, claudeDirPath, lock, "project")
	require.NoError(t, err)

	m2, err := state.Read(targetLock)
	require.NoError(t, err)
	assert.NotEqual(t, m1.LastApplied, m2.LastApplied, "force should always recompile")

	// Reset
	applyForce = false
}

func TestApplyScope_MigratesLegacyLock(t *testing.T) {
	dir := t.TempDir()
	xcf := filepath.Join(dir, "scaffold.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(minimalXCF), 0600))

	claudeDirPath := filepath.Join(dir, ".claude")
	legacyLock := filepath.Join(dir, "scaffold.lock")

	// Create a legacy lock file
	require.NoError(t, os.WriteFile(legacyLock, []byte("version: 1\nlast_applied: \"2025-01-01T00:00:00Z\"\n"), 0600))

	targetFlag = targetClaude
	applyForce = true // force to avoid skip logic
	err := applyScope(xcf, claudeDirPath, legacyLock, "project")
	require.NoError(t, err)

	// Legacy lock should be gone (was migrated before first read)
	// Target-specific lock should exist with fresh data
	targetLock := filepath.Join(dir, "scaffold.claude.lock")
	_, err = os.Stat(targetLock)
	assert.NoError(t, err, "target-specific lock should exist")

	applyForce = false
}

func TestApplyScope_PurgesOrphanedFiles(t *testing.T) {
	dir := t.TempDir()

	// Create initial config with an agent
	xcfContent := `version: "1"
project:
  name: orphan-test
agents:
  dev:
    name: Developer
    model: sonnet-4
    instructions: |
      You are a developer.
`
	xcf := filepath.Join(dir, "scaffold.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(xcfContent), 0600))

	claudeDirPath := filepath.Join(dir, ".claude")
	lock := filepath.Join(dir, "scaffold.lock")

	targetFlag = targetClaude
	applyForce = true
	err := applyScope(xcf, claudeDirPath, lock, "project")
	require.NoError(t, err)

	// Verify agent file exists
	agentFile := filepath.Join(dir, ".claude", "agents", "dev.md")
	_, err = os.Stat(agentFile)
	require.NoError(t, err, "agent file should exist after first apply")

	// Remove the agent from config
	xcfNoAgent := `version: "1"
project:
  name: orphan-test
`
	require.NoError(t, os.WriteFile(xcf, []byte(xcfNoAgent), 0600))

	// Second apply — should purge the orphaned agent file
	err = applyScope(xcf, claudeDirPath, lock, "project")
	require.NoError(t, err)

	_, err = os.Stat(agentFile)
	assert.True(t, os.IsNotExist(err), "orphaned agent file should be deleted")

	applyForce = false
}

func TestApplyScope_DryRun_ListsOrphans(t *testing.T) {
	dir := t.TempDir()

	xcfContent := `version: "1"
project:
  name: orphan-test
agents:
  dev:
    name: Developer
    model: sonnet-4
    instructions: |
      You are a developer.
`
	xcf := filepath.Join(dir, "scaffold.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(xcfContent), 0600))

	claudeDirPath := filepath.Join(dir, ".claude")
	lock := filepath.Join(dir, "scaffold.lock")

	targetFlag = targetClaude
	applyForce = true
	applyDryRun = false
	err := applyScope(xcf, claudeDirPath, lock, "project")
	require.NoError(t, err)

	agentFile := filepath.Join(dir, ".claude", "agents", "dev.md")
	_, err = os.Stat(agentFile)
	require.NoError(t, err)

	// Remove agent from config
	require.NoError(t, os.WriteFile(xcf, []byte(`version: "1"
project:
  name: orphan-test
`), 0600))

	// Dry run — should NOT delete the file
	applyDryRun = true
	err = applyScope(xcf, claudeDirPath, lock, "project")
	require.NoError(t, err)

	_, err = os.Stat(agentFile)
	assert.NoError(t, err, "dry run should NOT delete orphaned files")

	applyDryRun = false
	applyForce = false
}
