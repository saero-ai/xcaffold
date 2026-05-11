package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/state"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunApply_BlueprintFlag_MutualExclusion_WithGlobal verifies that
// --blueprint and --global are mutually exclusive on apply.
func TestRunApply_BlueprintFlag_MutualExclusion_WithGlobal(t *testing.T) {
	applyBlueprintFlag = "my-blueprint"
	globalFlag = true
	defer func() {
		applyBlueprintFlag = ""
		globalFlag = false
	}()

	err := runApply(applyCmd, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--blueprint cannot be used with --global")
}

func TestApply_Claude_CLAUDE_MD_WrittenAtProjectRoot(t *testing.T) {
	// Use a minimal in-memory xcaf with project instructions
	dir := t.TempDir()

	projectXcaf := filepath.Join(dir, "project.xcaf")
	content := `---
kind: project
version: "1.0"
name: test
`
	if err := os.WriteFile(projectXcaf, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	xcaffoldDir := filepath.Join(dir, ".xcaffold")
	if err := os.MkdirAll(xcaffoldDir, 0755); err != nil {
		t.Fatal(err)
	}

	contextDir := filepath.Join(dir, "xcaf", "context")
	if err := os.MkdirAll(contextDir, 0755); err != nil {
		t.Fatal(err)
	}
	contextXcaf := filepath.Join(contextDir, "claude.xcaf")
	contextContent := `---
kind: context
version: "1.0"
name: claude
---
Use pnpm. PostgreSQL 16.
`
	if err := os.WriteFile(contextXcaf, []byte(contextContent), 0644); err != nil {
		t.Fatal(err)
	}

	outputDir := filepath.Join(dir, ".claude")
	stateFile := filepath.Join(xcaffoldDir, "project.xcaf.state")

	// Set up targetFlag for this test
	oldTargetFlag := targetFlag
	targetFlag = "claude"
	defer func() { targetFlag = oldTargetFlag }()

	// Act
	err := applyScope(projectXcaf, outputDir, dir, "project")
	if err != nil {
		t.Fatalf("applyScope failed: %v", err)
	}

	// Assert: CLAUDE.md is at project root, not inside .claude/
	claudeMDPath := filepath.Join(dir, "CLAUDE.md")
	if _, err := os.Stat(claudeMDPath); os.IsNotExist(err) {
		t.Errorf("expected CLAUDE.md at %s, not found", claudeMDPath)
	}

	// Assert: CLAUDE.md is NOT inside .claude/
	claudeMDInDotClaude := filepath.Join(outputDir, "CLAUDE.md")
	if _, err := os.Stat(claudeMDInDotClaude); err == nil {
		t.Errorf("CLAUDE.md must NOT exist inside .claude/: %s", claudeMDInDotClaude)
	}

	// Assert: state file exists
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Error("expected state file to exist after apply")
	}
}

// minimalXCAF is a minimal valid project.xcaf for apply tests.
const minimalXCAF = `kind: project
version: "1.0"
name: apply-test
targets: [claude]
`

func TestApplyScope_Project(t *testing.T) {
	dir := t.TempDir()
	xcaf := filepath.Join(dir, "project.xcaf")
	require.NoError(t, os.WriteFile(xcaf, []byte(minimalXCAF), 0600))

	claudeDirPath := filepath.Join(dir, ".claude")

	err := applyScope(xcaf, claudeDirPath, dir, "project")
	require.NoError(t, err)

	stateFile := state.StateFilePath(dir, "")
	_, err = os.Stat(stateFile)
	assert.NoError(t, err, "state file should exist after applyScope")

	// Minimal XCAF has no agents, skills, or rules, so no subdirectories are pre-created.
	// The output directory itself may not be created either when there are no files to write.
	_ = claudeDirPath
}

func TestApplyScope_MissingXCAF(t *testing.T) {
	dir := t.TempDir()
	claudeDirPath := filepath.Join(dir, ".claude")

	err := applyScope(filepath.Join(dir, "nonexistent.xcaf"), claudeDirPath, dir, "project")
	assert.Error(t, err, "should fail when xcaf file does not exist")
}

func TestResolveTargets_FlagOverride(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "xcaf"), 0o755)
	os.WriteFile(filepath.Join(dir, "project.xcaf"), []byte("kind: project\nversion: \"1.0\"\nname: test\ntargets: [gemini]\n"), 0o644)

	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&targetFlag, "target", "", "")
	cmd.Flags().Set("target", "cursor")

	result := resolveTargets(cmd, dir, "")
	require.Equal(t, []string{"cursor"}, result)
}

func TestResolveTargets_BlueprintTargets(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "xcaf", "blueprints"), 0o755)
	os.WriteFile(filepath.Join(dir, "project.xcaf"), []byte("kind: project\nversion: \"1.0\"\nname: test\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "xcaf", "blueprints", "my-bp.xcaf"), []byte("kind: blueprint\nversion: \"1.0\"\nname: my-bp\ntargets: [gemini, antigravity]\nagents: [a]\n"), 0o644)

	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&targetFlag, "target", "", "")

	result := resolveTargets(cmd, dir, "my-bp")
	require.Equal(t, []string{"gemini", "antigravity"}, result)
}

func TestResolveTargets_ProjectTargets(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "xcaf"), 0o755)
	os.WriteFile(filepath.Join(dir, "project.xcaf"), []byte("kind: project\nversion: \"1.0\"\nname: test\ntargets: [claude, cursor]\n"), 0o644)

	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&targetFlag, "target", "", "")

	result := resolveTargets(cmd, dir, "")
	require.Equal(t, []string{"claude", "cursor"}, result)
}

func TestResolveTargets_NoTargets_ReturnsNil(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "xcaf"), 0o755)
	os.WriteFile(filepath.Join(dir, "project.xcaf"), []byte("kind: project\nversion: \"1.0\"\nname: test\n"), 0o644)

	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&targetFlag, "target", "", "")

	result := resolveTargets(cmd, dir, "")
	require.Nil(t, result)
}

func TestResolveTargets_BlueprintNoTargets_ReturnsNil(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "xcaf", "blueprints"), 0o755)
	os.WriteFile(filepath.Join(dir, "project.xcaf"), []byte("kind: project\nversion: \"1.0\"\nname: test\ntargets: [claude]\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "xcaf", "blueprints", "my-bp.xcaf"), []byte("kind: blueprint\nversion: \"1.0\"\nname: my-bp\nagents: [a]\n"), 0o644)

	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&targetFlag, "target", "", "")

	result := resolveTargets(cmd, dir, "my-bp")
	require.Nil(t, result)
}

func TestRunApply_ScopeProject(t *testing.T) {
	dir := t.TempDir()
	xcaf := filepath.Join(dir, "project.xcaf")
	require.NoError(t, os.WriteFile(xcaf, []byte(minimalXCAF), 0600))

	// Set package-level path vars used by runApply.
	xcafPath = xcaf
	projectRoot = dir
	globalFlag = false

	err := runApply(nil, nil)
	require.NoError(t, err)

	stateFile := state.StateFilePath(dir, "")
	_, err = os.Stat(stateFile)
	assert.NoError(t, err, "state file should be written for project scope")
}

func TestRunApply_ScopeGlobal(t *testing.T) {
	dir := t.TempDir()
	xcaf := filepath.Join(dir, "global.xcaf")
	require.NoError(t, os.WriteFile(xcaf, []byte(minimalXCAF), 0600))

	// globalXcafHome is the source directory containing global.xcaf (~/.xcaffold/).
	// Output goes one level up from globalXcafHome into the home dir's .claude/.
	globalXcafPath = xcaf
	globalXcafHome = dir
	globalFlag = true
	defer func() { globalFlag = false }()

	err := runApply(nil, nil)
	require.NoError(t, err)

	// State is written inside globalXcafHome/.xcaffold/
	stateFile := state.StateFilePath(dir, "")
	_, err = os.Stat(stateFile)
	assert.NoError(t, err, "state file should be written for global scope")
}

func TestRunApply_GlobalFlagFalse_CompilesProject(t *testing.T) {
	dir := t.TempDir()
	xcaf := filepath.Join(dir, "project.xcaf")
	require.NoError(t, os.WriteFile(xcaf, []byte(minimalXCAF), 0600))

	xcafPath = xcaf
	projectRoot = dir
	globalFlag = false
	targetFlag = "claude"

	err := runApply(nil, nil)
	require.NoError(t, err)

	stateFile := state.StateFilePath(dir, "")
	_, err = os.Stat(stateFile)
	assert.NoError(t, err, "state file should be written when globalFlag is false")
}

func TestApplyScope_SkipsWhenSourceUnchanged(t *testing.T) {
	dir := t.TempDir()
	xcaf := filepath.Join(dir, "project.xcaf")
	require.NoError(t, os.WriteFile(xcaf, []byte(minimalXCAF), 0600))

	claudeDirPath := filepath.Join(dir, ".claude")

	// First apply — should compile
	applyForce = false
	targetFlag = "claude"
	err := applyScope(xcaf, claudeDirPath, dir, "project")
	require.NoError(t, err)

	stateFile := state.StateFilePath(dir, "")
	_, err = os.Stat(stateFile)
	require.NoError(t, err, "state file should exist after first apply")

	// Read first state to get timestamp
	m1, err := state.ReadState(stateFile)
	require.NoError(t, err)
	ts1 := m1.Targets["claude"].LastApplied

	// Second apply — should skip (same sources)
	err = applyScope(xcaf, claudeDirPath, dir, "project")
	require.NoError(t, err)

	// State timestamp should NOT change (compilation was skipped)
	m2, err := state.ReadState(stateFile)
	require.NoError(t, err)
	assert.Equal(t, ts1, m2.Targets["claude"].LastApplied, "timestamp should not change when sources are unchanged")
}

func TestApplyScope_RecompilesWhenSourceChanged(t *testing.T) {
	dir := t.TempDir()
	xcaf := filepath.Join(dir, "project.xcaf")
	require.NoError(t, os.WriteFile(xcaf, []byte(minimalXCAF), 0600))

	claudeDirPath := filepath.Join(dir, ".claude")

	applyForce = false
	targetFlag = "claude"

	// First apply
	err := applyScope(xcaf, claudeDirPath, dir, "project")
	require.NoError(t, err)

	stateFile := state.StateFilePath(dir, "")
	m1, err := state.ReadState(stateFile)
	require.NoError(t, err)
	ts1 := m1.Targets["claude"].LastApplied

	// Modify source
	modifiedXCAF := `kind: project
version: "1.0"
name: apply-test-modified
`
	require.NoError(t, os.WriteFile(xcaf, []byte(modifiedXCAF), 0600))
	time.Sleep(1 * time.Second)

	// Second apply — should recompile
	err = applyScope(xcaf, claudeDirPath, dir, "project")
	require.NoError(t, err)

	m2, err := state.ReadState(stateFile)
	require.NoError(t, err)
	assert.NotEqual(t, ts1, m2.Targets["claude"].LastApplied, "timestamp should change when sources are modified")
}

func TestApplyScope_ForceRecompiles(t *testing.T) {
	dir := t.TempDir()
	xcaf := filepath.Join(dir, "project.xcaf")
	require.NoError(t, os.WriteFile(xcaf, []byte(minimalXCAF), 0600))

	claudeDirPath := filepath.Join(dir, ".claude")

	targetFlag = "claude"

	// First apply (non-force)
	applyForce = false
	err := applyScope(xcaf, claudeDirPath, dir, "project")
	require.NoError(t, err)

	stateFile := state.StateFilePath(dir, "")
	m1, err := state.ReadState(stateFile)
	require.NoError(t, err)
	ts1 := m1.Targets["claude"].LastApplied

	// Second apply with --force — should recompile despite no changes
	applyForce = true
	time.Sleep(1 * time.Second)
	err = applyScope(xcaf, claudeDirPath, dir, "project")
	require.NoError(t, err)

	m2, err := state.ReadState(stateFile)
	require.NoError(t, err)
	assert.NotEqual(t, ts1, m2.Targets["claude"].LastApplied, "force should always recompile")

	// Reset
	applyForce = false
}

func TestApplyScope_PurgesOrphanedFiles(t *testing.T) {
	dir := t.TempDir()

	// Create initial config with an agent
	xcaf := filepath.Join(dir, "project.xcaf")
	require.NoError(t, os.WriteFile(xcaf, []byte(`---
kind: project
version: "1.0"
name: orphan-test
`), 0600))

	agentDir := filepath.Join(dir, "xcaf", "agents", "dev")
	os.MkdirAll(agentDir, 0755)
	os.WriteFile(filepath.Join(agentDir, "dev.xcaf"), []byte(`---
kind: agent
version: "1.0"
name: dev
description: A developer agent
---
You are a developer.
`), 0644)

	claudeDirPath := filepath.Join(dir, ".claude")

	targetFlag = "claude"
	applyForce = true
	err := applyScope(xcaf, claudeDirPath, dir, "project")
	require.NoError(t, err)

	// Verify agent file exists
	agentFile := filepath.Join(dir, ".claude", "agents", "dev.md")
	_, err = os.Stat(agentFile)
	require.NoError(t, err, "agent file should exist after first apply")

	// Remove the agent from config by removing its file so
	// the second apply finds no agents and must purge the orphaned agent file.
	require.NoError(t, os.RemoveAll(agentDir))

	// Second apply — should purge the orphaned agent file
	err = applyScope(xcaf, claudeDirPath, dir, "project")
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

	xcaf := filepath.Join(dir, "project.xcaf")
	require.NoError(t, os.WriteFile(xcaf, []byte(`---
kind: project
version: "1.0"
name: multi-target-test
targets:
  - claude
  - cursor
`), 0600))
	agentDir := filepath.Join(dir, "xcaf", "agents", "dev")
	os.MkdirAll(agentDir, 0755)
	os.WriteFile(filepath.Join(agentDir, "dev.xcaf"), []byte(`---
kind: agent
version: "1.0"
name: dev
description: A developer agent
---
You are a developer.
`), 0600)

	xcafPath = xcaf
	projectRoot = dir
	globalFlag = false
	applyForce = true
	targetFlag = "claude"
	applyCmd.Flags().Lookup("target").Changed = false

	err := runApply(applyCmd, nil)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, ".claude", "agents", "dev.md"))
	assert.NoError(t, err, ".claude/agents/dev.md should be created for claude target")

	_, err = os.Stat(filepath.Join(dir, ".cursor", "agents", "dev.md"))
	assert.NoError(t, err, ".cursor/agents/dev.md should be created for cursor target")

	stateFile := state.StateFilePath(dir, "")
	_, err = os.Stat(stateFile)
	assert.NoError(t, err, "state file should exist")

	manifest, err := state.ReadState(stateFile)
	require.NoError(t, err)
	_, hasClaude := manifest.Targets["claude"]
	_, hasCursor := manifest.Targets["cursor"]
	assert.True(t, hasClaude, "state should have claude target entry")
	assert.True(t, hasCursor, "state should have cursor target entry")

	applyForce = false
}

// TestRunApply_ExplicitTargetFlag verifies that when --target is explicitly set,
// only that target is compiled even if the project config declares more targets.
func TestRunApply_ExplicitTargetFlag(t *testing.T) {
	dir := t.TempDir()

	xcafContent := `kind: project
version: "1.0"
name: explicit-target-test
targets:
  - claude
  - cursor
`
	xcaf := filepath.Join(dir, "project.xcaf")
	require.NoError(t, os.WriteFile(xcaf, []byte(xcafContent), 0600))

	agentDir := filepath.Join(dir, ".xcaffold", "agents")
	os.MkdirAll(agentDir, 0755)
	os.WriteFile(filepath.Join(agentDir, "dev.xcaf"), []byte(`---
kind: agent
version: "1.0"
name: dev
---
You are a developer.
`), 0644)

	xcafPath = xcaf
	projectRoot = dir
	globalFlag = false
	applyForce = true

	// Simulate --target=claude being explicitly set by marking the flag Changed.
	require.NoError(t, applyCmd.Flags().Set("target", "claude"))

	err := runApply(applyCmd, nil)
	require.NoError(t, err)

	// .cursor/ must not exist — only the claude target should be compiled.
	// Note: .claude/ itself may not exist when the config produces no files to write.
	_, err = os.Stat(filepath.Join(dir, ".cursor"))
	assert.True(t, os.IsNotExist(err), ".cursor/ must NOT be created when --target is explicit")

	// Reset the Changed flag by re-registering with default value
	applyCmd.Flags().Lookup("target").Changed = false
	applyForce = false
}

// TestRunApply_NoTargetsInConfig_NoDefault verifies that when the
// project config has no declared targets and --target is not set, an
// error is returned (no longer defaults to "claude").
func TestRunApply_NoTargetsInConfig_NoDefault(t *testing.T) {
	dir := t.TempDir()

	// Config with no targets field
	xcafContent := `kind: project
version: "1.0"
name: no-targets-test
`
	xcaf := filepath.Join(dir, "project.xcaf")
	require.NoError(t, os.WriteFile(xcaf, []byte(xcafContent), 0600))

	agentDir := filepath.Join(dir, ".xcaffold", "agents")
	os.MkdirAll(agentDir, 0755)
	os.WriteFile(filepath.Join(agentDir, "dev.xcaf"), []byte(`kind: agent
version: "1.0"
name: dev
`), 0644)

	xcafPath = xcaf
	projectRoot = dir
	globalFlag = false
	applyForce = true
	targetFlag = ""
	applyCmd.Flags().Lookup("target").Changed = false

	err := runApply(applyCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no compilation targets configured")

	applyForce = false
}

func TestApplyScope_DryRun_ListsOrphans(t *testing.T) {
	dir := t.TempDir()

	xcaf := filepath.Join(dir, "project.xcaf")
	require.NoError(t, os.WriteFile(xcaf, []byte(`---
kind: project
version: "1.0"
name: orphan-test
`), 0600))
	agentDir := filepath.Join(dir, "xcaf", "agents", "dev")
	os.MkdirAll(agentDir, 0755)
	os.WriteFile(filepath.Join(agentDir, "dev.xcaf"), []byte(`---
kind: agent
version: "1.0"
name: dev
description: A developer agent
---
You are a developer.
`), 0600)

	claudeDirPath := filepath.Join(dir, ".claude")

	targetFlag = "claude"
	applyForce = true
	applyDryRun = false
	err := applyScope(xcaf, claudeDirPath, dir, "project")
	require.NoError(t, err)

	agentFile := filepath.Join(dir, ".claude", "agents", "dev.md")
	_, err = os.Stat(agentFile)
	require.NoError(t, err)

	require.NoError(t, os.RemoveAll(agentDir))

	applyDryRun = true
	err = applyScope(xcaf, claudeDirPath, dir, "project")
	require.NoError(t, err)

	_, err = os.Stat(agentFile)
	assert.NoError(t, err, "dry run should NOT delete orphaned files")

	applyDryRun = false
	applyForce = false
}

// TestApplyScope_RegistryXCAF_ExcludedFromSourceTracking verifies that a
// kind: registry file is not recorded in the lock manifest's source list.
// Without the filter, registry.xcaf is updated on every apply, causing
// SourcesChanged to always return true and defeating smart-skip.
func TestApplyScope_RegistryXCAF_ExcludedFromSourceTracking(t *testing.T) {
	dir := t.TempDir()

	// Config file — the only source that should appear in the lock.
	xcaf := filepath.Join(dir, "project.xcaf")
	require.NoError(t, os.WriteFile(xcaf, []byte(minimalXCAF), 0600))

	// Registry file — must NOT appear in the lock's source list.
	registryXCAF := filepath.Join(dir, "registry.xcaf")
	registryContent := `kind: registry
version: "1"
`
	require.NoError(t, os.WriteFile(registryXCAF, []byte(registryContent), 0600))

	claudeDirPath := filepath.Join(dir, ".claude")

	applyForce = true
	targetFlag = "claude"
	err := applyScope(xcaf, claudeDirPath, dir, "project")
	require.NoError(t, err)

	stateFile := state.StateFilePath(dir, "")
	manifest, err := state.ReadState(stateFile)
	require.NoError(t, err)

	for _, sf := range manifest.SourceFiles {
		assert.NotEqual(t, "registry.xcaf", sf.Path,
			"registry.xcaf must not appear in state manifest source files")
	}

	applyForce = false
}

// TestRunApply_Backup_MultiTarget verifies that --backup creates a backup for
// every declared target, including targets whose sources have not changed since
// the previous apply (i.e. targets that would otherwise hit the smart-skip path).
func TestRunApply_Backup_MultiTarget(t *testing.T) {
	dir := t.TempDir()

	xcaf := filepath.Join(dir, "project.xcaf")
	require.NoError(t, os.WriteFile(xcaf, []byte(`---
kind: project
version: "1.0"
name: backup-multi-target-test
targets:
  - claude
  - cursor
`), 0600))

	agentDir := filepath.Join(dir, "xcaf", "agents", "dev")
	require.NoError(t, os.MkdirAll(agentDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "dev.xcaf"), []byte(`---
kind: agent
version: "1.0"
name: dev
description: A developer agent
---
You are a developer.
`), 0600))

	xcafPath = xcaf
	projectRoot = dir
	globalFlag = false
	applyForce = true
	applyBackup = false
	applyDryRun = false
	targetFlag = "claude"
	applyCmd.Flags().Lookup("target").Changed = false

	// First apply: compile both targets and write state so the smart-skip path
	// will fire for both targets on the second apply.
	require.NoError(t, runApply(applyCmd, nil))

	_, err := os.Stat(filepath.Join(dir, ".claude", "agents", "dev.md"))
	require.NoError(t, err, ".claude/agents/dev.md must exist after first apply")
	_, err = os.Stat(filepath.Join(dir, ".cursor", "agents", "dev.md"))
	require.NoError(t, err, ".cursor/agents/dev.md must exist after first apply")

	// Second apply: sources are unchanged, but --backup is set.
	// Both target directories must be backed up even though compilation is skipped.
	applyForce = false
	applyBackup = true

	require.NoError(t, runApply(applyCmd, nil))

	// Verify a backup exists for the claude target.
	claudeBackups, err := filepath.Glob(filepath.Join(dir, ".claude_bak_*"))
	require.NoError(t, err)
	assert.NotEmpty(t, claudeBackups, "backup directory must be created for claude target")

	// Verify a backup exists for the cursor target.
	cursorBackups, err := filepath.Glob(filepath.Join(dir, ".cursor_bak_*"))
	require.NoError(t, err)
	assert.NotEmpty(t, cursorBackups, "backup directory must be created for cursor target")

	applyBackup = false
	applyForce = false
}

// TestApplyCmd_NoIncludeMemoryFlag verifies that --include-memory is no longer
// registered on applyCmd now that memory rendering is always-on via the orchestrator.
func TestApplyCmd_NoIncludeMemoryFlag(t *testing.T) {
	flag := applyCmd.Flags().Lookup("include-memory")
	require.Nil(t, flag, "--include-memory must not be registered; memory rendering is always-on")
}

// TestApplyCmd_NoReseedFlag verifies that --reseed is no longer registered on
// applyCmd. Use --force to bypass drift detection and overwrite all outputs.
func TestApplyCmd_NoReseedFlag(t *testing.T) {
	flag := applyCmd.Flags().Lookup("reseed")
	require.Nil(t, flag, "--reseed must not be registered; use --force to overwrite outputs")
}

// TestApplyScope_OrchestratorMemory_Claude verifies that the convention-based
// memory compiler discovers .md files under xcaf/agents/<id>/memory/ and seeds
// them into .claude/agent-memory/<agentID>/MEMORY.md.
// Requires convention-based compiler scan to be complete.
func TestApplyScope_OrchestratorMemory_Claude(t *testing.T) {
	t.Skip("requires convention-based compiler: .md discovery not yet wired into apply")
	dir := t.TempDir()

	xcaf := filepath.Join(dir, "project.xcaf")
	require.NoError(t, os.WriteFile(xcaf, []byte(`---
kind: project
version: "1.0"
name: memory-render-test
`), 0600))

	// Memory entry under xcaf/agents/default/memory/ — AgentRef will be "default".
	memDir := filepath.Join(dir, "xcaf", "agents", "default", "memory")
	require.NoError(t, os.MkdirAll(memDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(memDir, "user-role.xcaf"), []byte(`---
kind: memory
version: "1.0"
name: user-role
---
Robert is the founder.
`), 0600))

	claudeDir := filepath.Join(dir, ".claude")
	targetFlag = "claude"
	applyForce = true
	defer func() { applyForce = false }()

	err := applyScope(xcaf, claudeDir, dir, "project")
	require.NoError(t, err)

	// The orchestrator always compiles memory entries when the renderer supports it.
	memFile := filepath.Join(claudeDir, "agent-memory", "default", "MEMORY.md")
	_, err = os.Stat(memFile)
	require.NoError(t, err, "agent-memory file must exist at .claude/agent-memory/default/MEMORY.md")

	data, err := os.ReadFile(memFile)
	require.NoError(t, err)
	require.Contains(t, string(data), "## user-role")
	require.Contains(t, string(data), "Robert is the founder.")
}

// TestApplyScope_OrchestratorMemory_AgentRef verifies that a memory entry placed
// under xcaf/agents/<agentID>/memory/ is routed to agent-memory/<agentID>/MEMORY.md.
// AgentRef is derived from the directory layout at compile time, not from YAML.
// Requires convention-based compiler scan to be complete.
func TestApplyScope_OrchestratorMemory_AgentRef(t *testing.T) {
	t.Skip("requires convention-based compiler: .md discovery not yet wired into apply")
	dir := t.TempDir()

	xcaf := filepath.Join(dir, "project.xcaf")
	require.NoError(t, os.WriteFile(xcaf, []byte(`---
kind: project
version: "1.0"
name: memory-agentref-test
`), 0600))

	// Place the memory file under xcaf/agents/go-cli-developer/memory/ so the parser
	// sets AgentRef = "go-cli-developer" from the segment before "memory".
	agentMemDir := filepath.Join(dir, "xcaf", "agents", "go-cli-developer", "memory")
	require.NoError(t, os.MkdirAll(agentMemDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(agentMemDir, "arch-decisions.xcaf"), []byte(`---
kind: memory
version: "1.0"
name: arch-decisions
---
Use cobra for all commands.
`), 0600))

	claudeDir := filepath.Join(dir, ".claude")
	targetFlag = "claude"
	applyForce = true
	defer func() { applyForce = false }()

	err := applyScope(xcaf, claudeDir, dir, "project")
	require.NoError(t, err)

	memFile := filepath.Join(claudeDir, "agent-memory", "go-cli-developer", "MEMORY.md")
	_, err = os.Stat(memFile)
	require.NoError(t, err, "agent-memory file must exist at .claude/agent-memory/go-cli-developer/MEMORY.md")

	data, err := os.ReadFile(memFile)
	require.NoError(t, err)
	require.Contains(t, string(data), "## arch-decisions")
	require.Contains(t, string(data), "Use cobra for all commands.")
}

// TestApplyScope_OrchestratorMemory_NoEntries_NoDir verifies that when a
// config declares no memory entries, no agent-memory/ directory is created.
func TestApplyScope_OrchestratorMemory_NoEntries_NoDir(t *testing.T) {
	dir := t.TempDir()

	xcaf := filepath.Join(dir, "project.xcaf")
	require.NoError(t, os.WriteFile(xcaf, []byte(minimalXCAF), 0600))

	claudeDir := filepath.Join(dir, ".claude")
	targetFlag = "claude"
	applyForce = true
	defer func() { applyForce = false }()

	err := applyScope(xcaf, claudeDir, dir, "project")
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(claudeDir, "agent-memory"))
	require.True(t, os.IsNotExist(err), "agent-memory/ must not be created when config has no memory entries")
}

// TestCheckFidelityErrors_ErrorLevel verifies that checkFidelityErrors returns
// an error when fidelity notes contain error-level entries.
func TestCheckFidelityErrors_ErrorLevel(t *testing.T) {
	notes := []renderer.FidelityNote{
		{Level: renderer.LevelError, Code: renderer.CodeFieldRequiredForTarget, Kind: "agent", Resource: "test", Reason: "missing description"},
	}
	err := checkFidelityErrors(notes)
	require.Error(t, err, "expected error for error-level fidelity notes")
	assert.Contains(t, err.Error(), "compilation failed")
	assert.Contains(t, err.Error(), "1 error(s)")
}

// TestCheckFidelityErrors_WarningOnly verifies that checkFidelityErrors returns
// nil when notes contain only warning-level entries.
func TestCheckFidelityErrors_WarningOnly(t *testing.T) {
	notes := []renderer.FidelityNote{
		{Level: renderer.LevelWarning, Code: "SOME_WARNING", Reason: "just a warning"},
	}
	err := checkFidelityErrors(notes)
	assert.NoError(t, err, "should not error for warning-level notes")
}

// TestCheckFidelityErrors_Empty verifies that checkFidelityErrors returns
// nil when given an empty notes slice.
func TestCheckFidelityErrors_Empty(t *testing.T) {
	err := checkFidelityErrors(nil)
	assert.NoError(t, err, "should not error for empty notes")
}

// TestCheckFidelityErrors_Mixed verifies that checkFidelityErrors collects
// all error-level notes and includes them in the error message.
func TestCheckFidelityErrors_Mixed(t *testing.T) {
	notes := []renderer.FidelityNote{
		{Level: renderer.LevelWarning, Code: "WARN", Kind: "skill", Resource: "warn-resource", Reason: "warning only"},
		{Level: renderer.LevelError, Code: "ERR1", Kind: "agent", Resource: "err-resource-1", Reason: "missing field"},
		{Level: renderer.LevelInfo, Code: "INFO", Kind: "rule", Resource: "info-resource", Reason: "informational"},
		{Level: renderer.LevelError, Code: "ERR2", Kind: "context", Resource: "err-resource-2", Reason: "bad value"},
	}
	err := checkFidelityErrors(notes)
	require.Error(t, err, "expected error for mixed notes with errors")
	assert.Contains(t, err.Error(), "compilation failed")
	assert.Contains(t, err.Error(), "2 error(s)")
	assert.Contains(t, err.Error(), "ERR1")
	assert.Contains(t, err.Error(), "ERR2")
}

// TestApply_Blueprint_UsesOwnTargets verifies that a blueprint with explicit
// targets compiles only for those targets, ignoring the project's targets.
func TestApply_Blueprint_UsesOwnTargets(t *testing.T) {
	dir := t.TempDir()

	// Project with targets: [claude]
	projectXcaf := filepath.Join(dir, "project.xcaf")
	require.NoError(t, os.WriteFile(projectXcaf, []byte(`---
kind: project
version: "1.0"
name: test-project
targets: [claude]
`), 0o644))

	// Blueprint with targets: [gemini, cursor]
	bpDir := filepath.Join(dir, "xcaf", "blueprints")
	require.NoError(t, os.MkdirAll(bpDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(bpDir, "test-bp.xcaf"), []byte(`---
kind: blueprint
version: "1.0"
name: test-bp
targets: [gemini, cursor]
agents: [my-agent]
`), 0o644))

	// Minimal agent referenced by blueprint
	agentDir := filepath.Join(dir, "xcaf", "agents", "my-agent")
	require.NoError(t, os.MkdirAll(agentDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "my-agent.xcaf"), []byte(`---
kind: agent
version: "1.0"
name: my-agent
description: Test agent
`), 0o644))

	// Set up apply context: blueprint, force compile, no explicit --target flag
	xcafPath = projectXcaf
	projectRoot = dir
	applyBlueprintFlag = "test-bp"
	applyForce = true
	targetFlag = ""
	applyCmd.Flags().Lookup("target").Changed = false
	defer func() {
		applyBlueprintFlag = ""
		applyForce = false
		targetFlag = ""
	}()

	// Run apply with blueprint
	err := runApply(applyCmd, nil)
	require.NoError(t, err, "apply should succeed for blueprint with targets")

	// Verify gemini output exists (blueprint target)
	_, err = os.Stat(filepath.Join(dir, ".gemini"))
	assert.NoError(t, err, ".gemini output directory should exist for blueprint target")

	// Verify cursor output exists (blueprint target)
	_, err = os.Stat(filepath.Join(dir, ".cursor"))
	assert.NoError(t, err, ".cursor output directory should exist for blueprint target")

	// Verify claude output does NOT exist (project target, not blueprint target)
	_, err = os.Stat(filepath.Join(dir, ".claude"))
	assert.True(t, os.IsNotExist(err), ".claude output should NOT exist (not in blueprint targets)")
}

// TestApply_Blueprint_NoTargets_FailsWithError verifies that a blueprint without
// targets and a project without targets returns "no compilation targets configured" error.
func TestApply_Blueprint_NoTargets_FailsWithError(t *testing.T) {
	dir := t.TempDir()

	// Project with NO targets field
	projectXcaf := filepath.Join(dir, "project.xcaf")
	require.NoError(t, os.WriteFile(projectXcaf, []byte(`---
kind: project
version: "1.0"
name: test-project
`), 0o644))

	// Blueprint with NO targets field
	bpDir := filepath.Join(dir, "xcaf", "blueprints")
	require.NoError(t, os.MkdirAll(bpDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(bpDir, "test-bp.xcaf"), []byte(`---
kind: blueprint
version: "1.0"
name: test-bp
agents: [my-agent]
`), 0o644))

	// Minimal agent
	agentDir := filepath.Join(dir, "xcaf", "agents", "my-agent")
	require.NoError(t, os.MkdirAll(agentDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "my-agent.xcaf"), []byte(`---
kind: agent
version: "1.0"
name: my-agent
description: Test agent
`), 0o644))

	// Set up apply context: blueprint, no explicit --target flag
	xcafPath = projectXcaf
	projectRoot = dir
	applyBlueprintFlag = "test-bp"
	applyForce = true
	targetFlag = ""
	applyCmd.Flags().Lookup("target").Changed = false
	defer func() {
		applyBlueprintFlag = ""
		applyForce = false
		targetFlag = ""
	}()

	// Run apply — should fail with "no compilation targets configured"
	err := runApply(applyCmd, nil)
	require.Error(t, err, "apply should fail when blueprint and project have no targets")
	assert.Contains(t, err.Error(), "no compilation targets configured",
		"error should indicate missing targets")
}

// TestApply_Blueprint_FlagOverridesBlueprint verifies that --target flag
// overrides blueprint targets, compiling only the explicitly specified target.
func TestApply_Blueprint_FlagOverridesBlueprint(t *testing.T) {
	dir := t.TempDir()

	// Project with targets: [claude]
	projectXcaf := filepath.Join(dir, "project.xcaf")
	require.NoError(t, os.WriteFile(projectXcaf, []byte(`---
kind: project
version: "1.0"
name: test-project
targets: [claude]
`), 0o644))

	// Blueprint with targets: [gemini, cursor]
	bpDir := filepath.Join(dir, "xcaf", "blueprints")
	require.NoError(t, os.MkdirAll(bpDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(bpDir, "test-bp.xcaf"), []byte(`---
kind: blueprint
version: "1.0"
name: test-bp
targets: [gemini, cursor]
agents: [my-agent]
`), 0o644))

	// Minimal agent
	agentDir := filepath.Join(dir, "xcaf", "agents", "my-agent")
	require.NoError(t, os.MkdirAll(agentDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "my-agent.xcaf"), []byte(`---
kind: agent
version: "1.0"
name: my-agent
description: Test agent
`), 0o644))

	// Set up apply context: blueprint + explicit --target copilot
	// (copilot supports agents, unlike antigravity)
	xcafPath = projectXcaf
	projectRoot = dir
	applyBlueprintFlag = "test-bp"
	applyForce = true
	require.NoError(t, applyCmd.Flags().Set("target", "copilot"))
	defer func() {
		applyBlueprintFlag = ""
		applyForce = false
		applyCmd.Flags().Lookup("target").Changed = false
	}()

	// Run apply with --target copilot
	err := runApply(applyCmd, nil)
	require.NoError(t, err, "apply should succeed with explicit --target")

	// Verify ONLY copilot output exists (.github/ is copilot's target directory)
	_, err = os.Stat(filepath.Join(dir, ".github"))
	assert.NoError(t, err, ".github output should exist for copilot target")

	// Verify blueprint targets do NOT exist
	_, err = os.Stat(filepath.Join(dir, ".gemini"))
	assert.True(t, os.IsNotExist(err), ".gemini should NOT exist (overridden by --target)")

	_, err = os.Stat(filepath.Join(dir, ".cursor"))
	assert.True(t, os.IsNotExist(err), ".cursor should NOT exist (overridden by --target)")
}

// TestApply_WithVarFile verifies that the --var-file flag correctly injects
// variables into the compilation process.
func TestApply_WithVarFile(t *testing.T) {
	dir := t.TempDir()

	// 1. Setup project.xcaf
	projectXcaf := filepath.Join(dir, "project.xcaf")
	require.NoError(t, os.WriteFile(projectXcaf, []byte("kind: project\nversion: \"1.0\"\nname: var-test\ntargets: [claude]\n"), 0o644))

	// 2. Setup an agent that uses a variable
	agentDir := filepath.Join(dir, "xcaf", "agents", "dev")
	require.NoError(t, os.MkdirAll(agentDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "dev.xcaf"), []byte("---\nkind: agent\nversion: \"1.0\"\nname: dev\ndescription: \"Hello ${var.greeting}\"\n"), 0o644))

	// 3. Setup the custom variable file
	varFile := filepath.Join(dir, "custom.vars")
	require.NoError(t, os.WriteFile(varFile, []byte("greeting = World\n"), 0o644))

	// 4. Configure apply command
	xcafPath = projectXcaf
	projectRoot = dir
	globalFlag = false
	applyForce = true
	targetFlag = "claude"
	varFileFlag = varFile // Inject the flag

	// Reset flags after test
	defer func() {
		applyForce = false
		targetFlag = ""
		varFileFlag = ""
	}()

	// 5. Run Apply
	err := runApply(applyCmd, nil)
	require.NoError(t, err)

	// 6. Verify Output
	agentFile := filepath.Join(dir, ".claude", "agents", "dev.md")
	content, err := os.ReadFile(agentFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Hello World")
}

// TestComputeApplyPreview_NewFile verifies that a file not on disk is detected as "new".
func TestComputeApplyPreview_NewFile(t *testing.T) {
	dir := t.TempDir()
	outFiles := map[string]string{"agents/new-agent.md": "# New Agent"}
	entries := computeApplyPreview(outFiles, nil, dir, dir)
	assert.Len(t, entries, 1)
	assert.Equal(t, "new", entries[0].Status)
	assert.Equal(t, "agents/new-agent.md", entries[0].Path)
}

// TestComputeApplyPreview_ChangedFile verifies that a file with different content is detected as "changed".
func TestComputeApplyPreview_ChangedFile(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "agents"), 0755)
	os.WriteFile(filepath.Join(dir, "agents", "reviewer.md"), []byte("old content"), 0644)
	outFiles := map[string]string{"agents/reviewer.md": "new content"}
	entries := computeApplyPreview(outFiles, nil, dir, dir)
	assert.Len(t, entries, 1)
	assert.Equal(t, "changed", entries[0].Status)
	assert.Equal(t, "agents/reviewer.md", entries[0].Path)
}

// TestComputeApplyPreview_UnchangedFile verifies that a file with identical content is detected as "unchanged".
func TestComputeApplyPreview_UnchangedFile(t *testing.T) {
	dir := t.TempDir()
	content := "same content"
	os.MkdirAll(filepath.Join(dir, "agents"), 0755)
	os.WriteFile(filepath.Join(dir, "agents", "reviewer.md"), []byte(content), 0644)
	outFiles := map[string]string{"agents/reviewer.md": content}
	entries := computeApplyPreview(outFiles, nil, dir, dir)
	assert.Len(t, entries, 1)
	assert.Equal(t, "unchanged", entries[0].Status)
	assert.Equal(t, "agents/reviewer.md", entries[0].Path)
}

// TestCountOrphansFromState verifies that orphaned files (in old state but not in new output) are counted.
func TestCountOrphansFromState(t *testing.T) {
	oldManifest := &state.StateManifest{
		Targets: map[string]state.TargetState{
			"claude": {
				Artifacts: []state.Artifact{
					{Path: "agents/old.md"},
					{Path: "agents/removed.md"},
				},
			},
		},
	}
	outFiles := map[string]string{
		"agents/new.md": "content",
	}
	count := countOrphansFromState(oldManifest, "claude", outFiles)
	assert.Equal(t, 2, count)
}

// TestCountOrphansFromState_NoOldState verifies that nil state returns 0 orphans.
func TestCountOrphansFromState_NoOldState(t *testing.T) {
	outFiles := map[string]string{"agents/new.md": "content"}
	count := countOrphansFromState(nil, "claude", outFiles)
	assert.Equal(t, 0, count)
}

// TestRenderApplyPreview_CountsCorrectly verifies that preview rendering counts file statuses correctly.
func TestRenderApplyPreview_CountsCorrectly(t *testing.T) {
	entries := []applyDiffEntry{
		{Path: "agents/new1.md", Status: "new"},
		{Path: "agents/new2.md", Status: "new"},
		{Path: "agents/changed1.md", Status: "changed"},
		{Path: "agents/unchanged1.md", Status: "unchanged"},
		{Path: "agents/unchanged2.md", Status: "unchanged"},
	}
	newC, changedC, unchangedC := renderApplyPreview(entries)
	assert.Equal(t, 2, newC, "should count 2 new files")
	assert.Equal(t, 1, changedC, "should count 1 changed file")
	assert.Equal(t, 2, unchangedC, "should count 2 unchanged files")
}
