package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/saero-ai/xcaffold/internal/state"
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
	// Use a minimal in-memory xcf with project instructions
	dir := t.TempDir()

	xcaffoldDir := filepath.Join(dir, ".xcaffold")
	if err := os.MkdirAll(xcaffoldDir, 0755); err != nil {
		t.Fatal(err)
	}
	projectXcf := filepath.Join(xcaffoldDir, "project.xcf")
	content := `---
kind: project
version: "1.0"
name: test
`
	if err := os.WriteFile(projectXcf, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	contextDir := filepath.Join(dir, "xcf", "context")
	if err := os.MkdirAll(contextDir, 0755); err != nil {
		t.Fatal(err)
	}
	contextXcf := filepath.Join(contextDir, "claude.xcf")
	contextContent := `---
kind: context
version: "1.0"
name: claude
---
Use pnpm. PostgreSQL 16.
`
	if err := os.WriteFile(contextXcf, []byte(contextContent), 0644); err != nil {
		t.Fatal(err)
	}

	outputDir := filepath.Join(dir, ".claude")
	stateFile := filepath.Join(xcaffoldDir, "project.xcf.state")

	// Act
	err := applyScope(projectXcf, outputDir, dir, "project")
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

// minimalXCF is a minimal valid project.xcf for apply tests.
const minimalXCF = `---
kind: project
version: "1.0"
name: apply-test
`

func TestApplyScope_Project(t *testing.T) {
	dir := t.TempDir()
	xcf := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(minimalXCF), 0600))

	claudeDirPath := filepath.Join(dir, ".claude")

	err := applyScope(xcf, claudeDirPath, dir, "project")
	require.NoError(t, err)

	stateFile := state.StateFilePath(dir, "")
	_, err = os.Stat(stateFile)
	assert.NoError(t, err, "state file should exist after applyScope")

	// Minimal XCF has no agents, skills, or rules, so no subdirectories are pre-created.
	// The output directory itself may not be created either when there are no files to write.
	_ = claudeDirPath
}

func TestApplyScope_MissingXCF(t *testing.T) {
	dir := t.TempDir()
	claudeDirPath := filepath.Join(dir, ".claude")

	err := applyScope(filepath.Join(dir, "nonexistent.xcf"), claudeDirPath, dir, "project")
	assert.Error(t, err, "should fail when xcf file does not exist")
}

func TestRunApply_ScopeProject(t *testing.T) {
	dir := t.TempDir()
	xcf := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(minimalXCF), 0600))

	// Set package-level path vars used by runApply.
	xcfPath = xcf
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
	xcf := filepath.Join(dir, "global.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(minimalXCF), 0600))

	// globalXcfHome is the source directory containing global.xcf (~/.xcaffold/).
	// Output goes one level up from globalXcfHome into the home dir's .claude/.
	globalXcfPath = xcf
	globalXcfHome = dir
	globalFlag = true
	defer func() { globalFlag = false }()

	err := runApply(nil, nil)
	require.NoError(t, err)

	// State is written inside globalXcfHome/.xcaffold/
	stateFile := state.StateFilePath(dir, "")
	_, err = os.Stat(stateFile)
	assert.NoError(t, err, "state file should be written for global scope")
}

func TestRunApply_GlobalFlagFalse_CompilesProject(t *testing.T) {
	dir := t.TempDir()
	xcf := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(minimalXCF), 0600))

	xcfPath = xcf
	projectRoot = dir
	globalFlag = false
	targetFlag = targetClaude

	err := runApply(nil, nil)
	require.NoError(t, err)

	stateFile := state.StateFilePath(dir, "")
	_, err = os.Stat(stateFile)
	assert.NoError(t, err, "state file should be written when globalFlag is false")
}

func TestApplyScope_SkipsWhenSourceUnchanged(t *testing.T) {
	dir := t.TempDir()
	xcf := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(minimalXCF), 0600))

	claudeDirPath := filepath.Join(dir, ".claude")

	// First apply — should compile
	applyForce = false
	targetFlag = targetClaude
	err := applyScope(xcf, claudeDirPath, dir, "project")
	require.NoError(t, err)

	stateFile := state.StateFilePath(dir, "")
	_, err = os.Stat(stateFile)
	require.NoError(t, err, "state file should exist after first apply")

	// Read first state to get timestamp
	m1, err := state.ReadState(stateFile)
	require.NoError(t, err)
	ts1 := m1.Targets[targetClaude].LastApplied

	// Second apply — should skip (same sources)
	err = applyScope(xcf, claudeDirPath, dir, "project")
	require.NoError(t, err)

	// State timestamp should NOT change (compilation was skipped)
	m2, err := state.ReadState(stateFile)
	require.NoError(t, err)
	assert.Equal(t, ts1, m2.Targets[targetClaude].LastApplied, "timestamp should not change when sources are unchanged")
}

func TestApplyScope_RecompilesWhenSourceChanged(t *testing.T) {
	dir := t.TempDir()
	xcf := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(minimalXCF), 0600))

	claudeDirPath := filepath.Join(dir, ".claude")

	applyForce = false
	targetFlag = targetClaude

	// First apply
	err := applyScope(xcf, claudeDirPath, dir, "project")
	require.NoError(t, err)

	stateFile := state.StateFilePath(dir, "")
	m1, err := state.ReadState(stateFile)
	require.NoError(t, err)
	ts1 := m1.Targets[targetClaude].LastApplied

	// Modify source
	modifiedXCF := `kind: project
version: "1.0"
name: apply-test-modified
`
	require.NoError(t, os.WriteFile(xcf, []byte(modifiedXCF), 0600))
	time.Sleep(1 * time.Second)

	// Second apply — should recompile
	err = applyScope(xcf, claudeDirPath, dir, "project")
	require.NoError(t, err)

	m2, err := state.ReadState(stateFile)
	require.NoError(t, err)
	assert.NotEqual(t, ts1, m2.Targets[targetClaude].LastApplied, "timestamp should change when sources are modified")
}

func TestApplyScope_ForceRecompiles(t *testing.T) {
	dir := t.TempDir()
	xcf := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(minimalXCF), 0600))

	claudeDirPath := filepath.Join(dir, ".claude")

	targetFlag = targetClaude

	// First apply (non-force)
	applyForce = false
	err := applyScope(xcf, claudeDirPath, dir, "project")
	require.NoError(t, err)

	stateFile := state.StateFilePath(dir, "")
	m1, err := state.ReadState(stateFile)
	require.NoError(t, err)
	ts1 := m1.Targets[targetClaude].LastApplied

	// Second apply with --force — should recompile despite no changes
	applyForce = true
	time.Sleep(1 * time.Second)
	err = applyScope(xcf, claudeDirPath, dir, "project")
	require.NoError(t, err)

	m2, err := state.ReadState(stateFile)
	require.NoError(t, err)
	assert.NotEqual(t, ts1, m2.Targets[targetClaude].LastApplied, "force should always recompile")

	// Reset
	applyForce = false
}

func TestApplyScope_PurgesOrphanedFiles(t *testing.T) {
	dir := t.TempDir()

	// Create initial config with an agent
	xcf := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(`---
kind: project
version: "1.0"
name: orphan-test
`), 0600))

	agentDir := filepath.Join(dir, "xcf", "agents", "dev")
	os.MkdirAll(agentDir, 0755)
	os.WriteFile(filepath.Join(agentDir, "dev.xcf"), []byte(`---
kind: agent
version: "1.0"
name: dev
---
You are a developer.
`), 0644)

	claudeDirPath := filepath.Join(dir, ".claude")

	targetFlag = targetClaude
	applyForce = true
	err := applyScope(xcf, claudeDirPath, dir, "project")
	require.NoError(t, err)

	// Verify agent file exists
	agentFile := filepath.Join(dir, ".claude", "agents", "dev.md")
	_, err = os.Stat(agentFile)
	require.NoError(t, err, "agent file should exist after first apply")

	// Remove the agent from config by removing its file so
	// the second apply finds no agents and must purge the orphaned agent file.
	require.NoError(t, os.RemoveAll(agentDir))

	// Second apply — should purge the orphaned agent file
	err = applyScope(xcf, claudeDirPath, dir, "project")
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

	xcf := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(`---
kind: project
version: "1.0"
name: multi-target-test
targets:
  - claude
  - cursor
`), 0600))
	agentDir := filepath.Join(dir, "xcf", "agents", "dev")
	os.MkdirAll(agentDir, 0755)
	os.WriteFile(filepath.Join(agentDir, "dev.xcf"), []byte(`---
kind: agent
version: "1.0"
name: dev
---
You are a developer.
`), 0600)

	xcfPath = xcf
	projectRoot = dir
	globalFlag = false
	applyForce = true
	targetFlag = targetClaude
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

	xcfContent := `kind: project
version: "1.0"
name: explicit-target-test
targets:
  - claude
  - cursor
`
	xcf := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(xcfContent), 0600))

	agentDir := filepath.Join(dir, ".xcaffold", "agents")
	os.MkdirAll(agentDir, 0755)
	os.WriteFile(filepath.Join(agentDir, "dev.xcf"), []byte(`---
kind: agent
version: "1.0"
name: dev
---
You are a developer.
`), 0644)

	xcfPath = xcf
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
	xcf := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(xcfContent), 0600))

	agentDir := filepath.Join(dir, ".xcaffold", "agents")
	os.MkdirAll(agentDir, 0755)
	os.WriteFile(filepath.Join(agentDir, "dev.xcf"), []byte(`---
kind: agent
version: "1.0"
name: dev
---
You are a developer.
`), 0644)

	xcfPath = xcf
	projectRoot = dir
	globalFlag = false
	applyForce = true
	targetFlag = targetClaude
	applyCmd.Flags().Lookup("target").Changed = false

	err := runApply(applyCmd, nil)
	require.NoError(t, err)

	// State file is written even for configs with no resources to emit.
	stateFile := state.StateFilePath(dir, "")
	_, err = os.Stat(stateFile)
	assert.NoError(t, err, "state file should be written for the default claude target")

	applyForce = false
}

func TestApplyScope_DryRun_ListsOrphans(t *testing.T) {
	dir := t.TempDir()

	xcf := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(`---
kind: project
version: "1.0"
name: orphan-test
`), 0600))
	agentDir := filepath.Join(dir, "xcf", "agents", "dev")
	os.MkdirAll(agentDir, 0755)
	os.WriteFile(filepath.Join(agentDir, "dev.xcf"), []byte(`---
kind: agent
version: "1.0"
name: dev
---
You are a developer.
`), 0600)

	claudeDirPath := filepath.Join(dir, ".claude")

	targetFlag = targetClaude
	applyForce = true
	applyDryRun = false
	err := applyScope(xcf, claudeDirPath, dir, "project")
	require.NoError(t, err)

	agentFile := filepath.Join(dir, ".claude", "agents", "dev.md")
	_, err = os.Stat(agentFile)
	require.NoError(t, err)

	require.NoError(t, os.RemoveAll(agentDir))

	applyDryRun = true
	err = applyScope(xcf, claudeDirPath, dir, "project")
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
	xcf := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(minimalXCF), 0600))

	// Registry file — must NOT appear in the lock's source list.
	registryXCF := filepath.Join(dir, "registry.xcf")
	registryContent := `kind: registry
version: "1"
`
	require.NoError(t, os.WriteFile(registryXCF, []byte(registryContent), 0600))

	claudeDirPath := filepath.Join(dir, ".claude")

	applyForce = true
	targetFlag = targetClaude
	err := applyScope(xcf, claudeDirPath, dir, "project")
	require.NoError(t, err)

	stateFile := state.StateFilePath(dir, "")
	manifest, err := state.ReadState(stateFile)
	require.NoError(t, err)

	for _, sf := range manifest.SourceFiles {
		assert.NotEqual(t, "registry.xcf", sf.Path,
			"registry.xcf must not appear in state manifest source files")
	}

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
// memory compiler discovers .md files under xcf/agents/<id>/memory/ and seeds
// them into .claude/agent-memory/<agentID>/MEMORY.md.
// Requires convention-based compiler scan to be complete.
func TestApplyScope_OrchestratorMemory_Claude(t *testing.T) {
	t.Skip("requires convention-based compiler: .md discovery not yet wired into apply")
	dir := t.TempDir()

	xcf := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(`---
kind: project
version: "1.0"
name: memory-render-test
`), 0600))

	// Memory entry under xcf/agents/default/memory/ — AgentRef will be "default".
	memDir := filepath.Join(dir, "xcf", "agents", "default", "memory")
	require.NoError(t, os.MkdirAll(memDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(memDir, "user-role.xcf"), []byte(`---
kind: memory
version: "1.0"
name: user-role
---
Robert is the founder.
`), 0600))

	claudeDir := filepath.Join(dir, ".claude")
	targetFlag = targetClaude
	applyForce = true
	defer func() { applyForce = false }()

	err := applyScope(xcf, claudeDir, dir, "project")
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
// under xcf/agents/<agentID>/memory/ is routed to agent-memory/<agentID>/MEMORY.md.
// AgentRef is derived from the directory layout at compile time, not from YAML.
// Requires convention-based compiler scan to be complete.
func TestApplyScope_OrchestratorMemory_AgentRef(t *testing.T) {
	t.Skip("requires convention-based compiler: .md discovery not yet wired into apply")
	dir := t.TempDir()

	xcf := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(`---
kind: project
version: "1.0"
name: memory-agentref-test
`), 0600))

	// Place the memory file under xcf/agents/go-cli-developer/memory/ so the parser
	// sets AgentRef = "go-cli-developer" from the segment before "memory".
	agentMemDir := filepath.Join(dir, "xcf", "agents", "go-cli-developer", "memory")
	require.NoError(t, os.MkdirAll(agentMemDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(agentMemDir, "arch-decisions.xcf"), []byte(`---
kind: memory
version: "1.0"
name: arch-decisions
---
Use cobra for all commands.
`), 0600))

	claudeDir := filepath.Join(dir, ".claude")
	targetFlag = targetClaude
	applyForce = true
	defer func() { applyForce = false }()

	err := applyScope(xcf, claudeDir, dir, "project")
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

	xcf := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(minimalXCF), 0600))

	claudeDir := filepath.Join(dir, ".claude")
	targetFlag = targetClaude
	applyForce = true
	defer func() { applyForce = false }()

	err := applyScope(xcf, claudeDir, dir, "project")
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(claudeDir, "agent-memory"))
	require.True(t, os.IsNotExist(err), "agent-memory/ must not be created when config has no memory entries")
}

// TestClaudeProjectMemoryDir_DeterministicEncoding verifies that
// claudeProjectMemoryDir returns the same directory for the same project root
// on repeated calls, and falls back to the working directory without error
// when given an empty projectRoot.
func TestClaudeProjectMemoryDir_DeterministicEncoding(t *testing.T) {
	tmp := t.TempDir()

	dir, err := claudeProjectMemoryDir(tmp)
	require.NoError(t, err)

	dirAgain, err := claudeProjectMemoryDir(tmp)
	require.NoError(t, err)
	require.Equal(t, dir, dirAgain)

	require.Contains(t, dir, ".claude/projects")
	require.True(t, strings.HasSuffix(dir, "memory"))

	// Empty projectRoot falls back to cwd without crashing.
	dirEmpty, err := claudeProjectMemoryDir("")
	require.NoError(t, err)
	require.Contains(t, dirEmpty, ".claude/projects")
	require.True(t, strings.HasSuffix(dirEmpty, "memory"))
}
