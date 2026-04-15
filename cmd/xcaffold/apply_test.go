package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunApply_CheckOnly_ReturnsErrorOnErrorDiagnostic verifies that
// --check returns a non-zero exit (non-nil error) when ValidateFile produces
// an error-severity diagnostic.  The xcf file points to a non-existent
// instructions-file to trigger the error diagnostic.
func TestRunApply_CheckOnly_ReturnsErrorOnErrorDiagnostic(t *testing.T) {
	dir := t.TempDir()

	// instructions-file pointing to a missing file → validateFileRefs emits
	// a Severity:"error" diagnostic.
	xcfContent := `---
kind: project
version: "1.0"
name: check-error-test
---
kind: global
version: "1.0"
agents:
  dev:
    description: Developer
    model: claude-sonnet-4-5
    instructions-file: missing-instructions.md
`
	xcf := filepath.Join(dir, "scaffold.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(xcfContent), 0600))

	xcfPath = xcf
	claudeDir = filepath.Join(dir, ".claude")
	lockPath = filepath.Join(dir, "scaffold.lock")
	globalFlag = false
	applyCheckOnly = true
	defer func() { applyCheckOnly = false }()

	err := runApply(nil, nil)
	assert.Error(t, err, "--check must return non-zero when diagnostics contain errors")
}

// minimalXCF is a minimal valid scaffold.xcf for apply tests.
const minimalXCF = `kind: project
version: "1.0"
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
	modifiedXCF := `kind: project
version: "1.0"
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
	xcfContent := `---
kind: project
version: "1.0"
name: orphan-test
---
kind: global
version: "1.0"
agents:
  dev:
    description: Developer
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
	xcfNoAgent := `kind: project
version: "1.0"
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

// TestRunApply_MultiTarget verifies that when a kind: project document declares
// multiple targets and --target is not explicitly set, runApply compiles for
// all declared targets.
func TestRunApply_MultiTarget(t *testing.T) {
	dir := t.TempDir()

	// kind: project document with two targets and an agent so both renderers
	// produce output files (empty compile → no output dir created).
	xcfContent := `kind: project
version: "1.0"
name: multi-target-test
targets:
  - claude
  - cursor
---
kind: agent
version: "1.0"
name: dev
description: "Developer agent."
instructions: |
  You are a developer.
model: "claude-sonnet-4-5"
`
	xcf := filepath.Join(dir, "scaffold.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(xcfContent), 0600))

	xcfPath = xcf
	claudeDir = filepath.Join(dir, ".claude")
	lockPath = filepath.Join(dir, "scaffold.lock")
	globalFlag = false
	applyForce = true
	targetFlag = targetClaude // default value — not Changed
	applyCmd.Flags().Lookup("target").Changed = false

	err := runApply(applyCmd, nil)
	require.NoError(t, err)

	// Both target output directories must exist after compilation with content.
	_, err = os.Stat(filepath.Join(dir, ".claude"))
	assert.NoError(t, err, ".claude/ should be created for claude target")

	_, err = os.Stat(filepath.Join(dir, ".cursor"))
	assert.NoError(t, err, ".cursor/ should be created for cursor target")

	// Both lock files must exist
	_, err = os.Stat(filepath.Join(dir, "scaffold.claude.lock"))
	assert.NoError(t, err, "scaffold.claude.lock should exist")

	_, err = os.Stat(filepath.Join(dir, "scaffold.cursor.lock"))
	assert.NoError(t, err, "scaffold.cursor.lock should exist")

	applyForce = false
}

// TestRunApply_ExplicitTargetFlag verifies that when --target is explicitly set,
// only that target is compiled even if the project config declares more targets.
func TestRunApply_ExplicitTargetFlag(t *testing.T) {
	dir := t.TempDir()

	xcfContent := `kind: project
version: "1.0"
name: explicit-target-test
targets:
  - claude
  - cursor
`
	xcf := filepath.Join(dir, "scaffold.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(xcfContent), 0600))

	xcfPath = xcf
	claudeDir = filepath.Join(dir, ".claude")
	lockPath = filepath.Join(dir, "scaffold.lock")
	globalFlag = false
	applyForce = true

	// Simulate --target=claude being explicitly set by marking the flag Changed.
	require.NoError(t, applyCmd.Flags().Set("target", "claude"))

	err := runApply(applyCmd, nil)
	require.NoError(t, err)

	// Only .claude/ should be created; .cursor/ must not exist
	_, err = os.Stat(filepath.Join(dir, ".claude"))
	assert.NoError(t, err, ".claude/ should be created")

	_, err = os.Stat(filepath.Join(dir, ".cursor"))
	assert.True(t, os.IsNotExist(err), ".cursor/ must NOT be created when --target is explicit")

	// Reset the Changed flag by re-registering with default value
	applyCmd.Flags().Lookup("target").Changed = false
	applyForce = false
}

// TestRunApply_NoTargetsInConfig_DefaultsToClaude verifies that when the
// project config has no declared targets and --target is not set, the
// default "claude" target is used.
func TestRunApply_NoTargetsInConfig_DefaultsToClaude(t *testing.T) {
	dir := t.TempDir()

	// Config with no targets field
	xcfContent := `kind: project
version: "1.0"
name: no-targets-test
`
	xcf := filepath.Join(dir, "scaffold.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(xcfContent), 0600))

	xcfPath = xcf
	claudeDir = filepath.Join(dir, ".claude")
	lockPath = filepath.Join(dir, "scaffold.lock")
	globalFlag = false
	applyForce = true
	targetFlag = targetClaude
	applyCmd.Flags().Lookup("target").Changed = false

	err := runApply(applyCmd, nil)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, ".claude"))
	assert.NoError(t, err, ".claude/ should be created as the default target")

	applyForce = false
}

func TestApplyScope_DryRun_ListsOrphans(t *testing.T) {
	dir := t.TempDir()

	xcfContent := `---
kind: project
version: "1.0"
name: orphan-test
---
kind: global
version: "1.0"
agents:
  dev:
    description: Developer
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
	require.NoError(t, os.WriteFile(xcf, []byte(`kind: project
version: "1.0"
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

// TestApplyScope_RegistryXCF_ExcludedFromSourceTracking verifies that a
// kind: registry file is not recorded in the lock manifest's source list.
// Without the filter, registry.xcf is updated on every apply, causing
// SourcesChanged to always return true and defeating smart-skip.
func TestApplyScope_RegistryXCF_ExcludedFromSourceTracking(t *testing.T) {
	dir := t.TempDir()

	// Config file — the only source that should appear in the lock.
	xcf := filepath.Join(dir, "scaffold.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(minimalXCF), 0600))

	// Registry file — must NOT appear in the lock's source list.
	registryXCF := filepath.Join(dir, "registry.xcf")
	registryContent := `kind: registry
version: "1"
`
	require.NoError(t, os.WriteFile(registryXCF, []byte(registryContent), 0600))

	claudeDirPath := filepath.Join(dir, ".claude")
	lock := filepath.Join(dir, "scaffold.lock")

	applyForce = true
	targetFlag = targetClaude
	err := applyScope(xcf, claudeDirPath, lock, "project")
	require.NoError(t, err)

	targetLock := filepath.Join(dir, "scaffold.claude.lock")
	manifest, err := state.Read(targetLock)
	require.NoError(t, err)

	for _, sf := range manifest.SourceFiles {
		assert.NotEqual(t, "registry.xcf", sf.Path,
			"registry.xcf must not appear in lock manifest source files")
	}

	applyForce = false
}

// TestApplyCmd_IncludeMemoryFlag_Registered verifies the --include-memory flag
// is registered on applyCmd with the correct default.
func TestApplyCmd_IncludeMemoryFlag_Registered(t *testing.T) {
	flag := applyCmd.Flags().Lookup("include-memory")
	require.NotNil(t, flag, "--include-memory flag must be registered")
	require.Equal(t, "false", flag.DefValue)
}

// TestApplyCmd_ReseedFlag_Registered verifies the --reseed flag is registered
// on applyCmd with the correct default.
func TestApplyCmd_ReseedFlag_Registered(t *testing.T) {
	flag := applyCmd.Flags().Lookup("reseed")
	require.NotNil(t, flag, "--reseed flag must be registered")
	require.Equal(t, "false", flag.DefValue)
}

// TestApplyCmd_ReseedImpliesIncludeMemory verifies the memoryPassEnabled
// helper treats --reseed as implying --include-memory.
func TestApplyCmd_ReseedImpliesIncludeMemory(t *testing.T) {
	require.True(t, memoryPassEnabled(false, true), "reseed=true must enable memory pass even when include-memory is false")
	require.True(t, memoryPassEnabled(true, false), "include-memory=true must enable memory pass")
	require.True(t, memoryPassEnabled(true, true), "both flags set must enable memory pass")
	require.False(t, memoryPassEnabled(false, false), "neither flag set must not enable memory pass")
}

// TestRunMemoryPass_Cursor_EmitsFidelityNote verifies that running the memory
// pass against the cursor target emits a MEMORY_NO_NATIVE_TARGET fidelity note.
func TestRunMemoryPass_Cursor_EmitsFidelityNote(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {Name: "user-role", Type: "user", Instructions: "test body"},
			},
		},
	}
	seeds, notes, err := runMemoryPass(config, t.TempDir(), "cursor", t.TempDir(), nil, false, false)
	require.NoError(t, err)
	require.Empty(t, seeds, "cursor memory pass must not produce lock seeds")
	require.Len(t, notes, 1)
	require.Equal(t, renderer.CodeMemoryNoNativeTarget, notes[0].Code)
}

// TestRunMemoryPass_Antigravity_WritesKnowledgeFiles verifies that the
// antigravity memory pass writes knowledge/<name>.md files to outputDir and
// returns a state.MemorySeed per file.
func TestRunMemoryPass_Antigravity_WritesKnowledgeFiles(t *testing.T) {
	outputDir := t.TempDir()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"team-context": {Name: "team-context", Type: "project", Instructions: "we ship weekly"},
			},
		},
	}
	seeds, _, err := runMemoryPass(config, t.TempDir(), "antigravity", outputDir, nil, false, false)
	require.NoError(t, err)
	require.Len(t, seeds, 1)
	require.Equal(t, "antigravity", seeds[0].Target)

	// Verify the knowledge file was actually written to disk.
	written := filepath.Join(outputDir, "knowledge", "team-context.md")
	_, err = os.Stat(written)
	require.NoError(t, err, "antigravity knowledge file must be written to disk")
}

// TestRunMemoryPass_DryRun_SkipsWrites verifies dry-run mode does not write
// files or return seeds.
func TestRunMemoryPass_DryRun_SkipsWrites(t *testing.T) {
	outputDir := t.TempDir()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"ctx": {Name: "ctx", Type: "project", Instructions: "x"},
			},
		},
	}
	seeds, _, err := runMemoryPass(config, t.TempDir(), "antigravity", outputDir, nil, true, false)
	require.NoError(t, err)
	require.Empty(t, seeds, "dry-run must not produce seeds")

	// Knowledge dir must not exist after dry-run.
	_, err = os.Stat(filepath.Join(outputDir, "knowledge"))
	require.True(t, os.IsNotExist(err), "dry-run must not create knowledge/ directory")
}

// TestRunMemoryPass_NoMemoryEntries_NoOp verifies the memory pass is a no-op
// when the config declares no memory entries.
func TestRunMemoryPass_NoMemoryEntries_NoOp(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	seeds, notes, err := runMemoryPass(config, t.TempDir(), "claude", t.TempDir(), nil, false, false)
	require.NoError(t, err)
	require.Empty(t, seeds)
	require.Empty(t, notes)
}

// TestRunMemoryPass_DryRun_Claude_LogsIntent verifies that dry-run mode for
// the claude target logs a DRY-RUN intent message to stderr and produces no seeds.
func TestRunMemoryPass_DryRun_Claude_LogsIntent(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {Name: "user-role", Type: "user", Instructions: "test"},
			},
		},
	}
	// Capture stderr.
	origStderr := os.Stderr
	rPipe, wPipe, _ := os.Pipe()
	os.Stderr = wPipe

	seeds, _, err := runMemoryPass(config, t.TempDir(), "claude", t.TempDir(), nil, true, false)
	wPipe.Close()
	os.Stderr = origStderr

	require.NoError(t, err)
	require.Empty(t, seeds, "dry-run must produce no seeds")

	captured, _ := io.ReadAll(rPipe)
	require.Contains(t, string(captured), "DRY-RUN", "dry-run must log intent to stderr")
}
