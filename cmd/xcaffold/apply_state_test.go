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
	require.NoError(t, os.WriteFile(filepath.Join(base, "project.xcaf"), []byte("---\nkind: project\nname: test-proj\n---\n"), 0644))

	statePath := state.StateFilePath(base, "")
	assert.Equal(t, filepath.Join(base, ".xcaffold", "project.xcaf.state"), statePath)

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

// TestApply_BlueprintContext_RootInstructionFileSurvivesReapply verifies that a
// blueprint apply with a default:false context writes CLAUDE.md on first apply
// and that a second blueprint apply does not purge it as an orphan.
//
// This is the end-to-end regression test for the bug where blueprint applies
// wrote an empty body (rendering no root file), and subsequent applies would
// then purge a previously-written CLAUDE.md as an orphan.
func TestApply_BlueprintContext_RootInstructionFileSurvivesReapply(t *testing.T) {
	dir := t.TempDir()

	// Write project.xcaf
	projectXcaf := filepath.Join(dir, "project.xcaf")
	require.NoError(t, os.WriteFile(projectXcaf, []byte(`---
kind: project
version: "1.0"
name: bp-context-test
`), 0644))

	// Write a workspace context (default: nil — applies in bare mode)
	ctxDir := filepath.Join(dir, "xcaf", "context")
	require.NoError(t, os.MkdirAll(ctxDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(ctxDir, "workspace.xcaf"), []byte(`---
kind: context
version: "1.0"
name: workspace
targets: [claude]
---
Workspace-wide instructions.
`), 0644))

	// Write a blueprint-scoped context (default: false — excluded from bare apply)
	require.NoError(t, os.WriteFile(filepath.Join(ctxDir, "task-ctx.xcaf"), []byte(`---
kind: context
version: "1.0"
name: task-ctx
targets: [claude]
default: false
---
Task-specific blueprint instructions.
`), 0644))

	// Write an agent referenced by the blueprint
	agentDir := filepath.Join(dir, "xcaf", "agents", "dev")
	require.NoError(t, os.MkdirAll(agentDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "dev.xcaf"), []byte(`---
kind: agent
version: "1.0"
name: dev
description: Developer agent
---
You are a developer.
`), 0644))

	// Write the blueprint that references the default:false context
	bpDir := filepath.Join(dir, "xcaf", "blueprints")
	require.NoError(t, os.MkdirAll(bpDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(bpDir, "task-bp.xcaf"), []byte(`kind: blueprint
version: "1.0"
name: task-bp
targets: [claude]
agents: [dev]
contexts: [task-ctx]
`), 0644))

	outputDir := filepath.Join(dir, ".claude")
	claudeMDPath := filepath.Join(dir, "CLAUDE.md")

	// First blueprint apply
	oldTarget := targetFlag
	oldBlueprint := applyBlueprintFlag
	oldForce := applyForce
	targetFlag = "claude"
	applyBlueprintFlag = "task-bp"
	applyForce = true
	defer func() {
		targetFlag = oldTarget
		applyBlueprintFlag = oldBlueprint
		applyForce = oldForce
	}()

	stateFile := state.StateFilePath(dir, "task-bp")

	err := applyScope(projectXcaf, outputDir, dir, "project")
	require.NoError(t, err, "first blueprint apply must succeed")

	// CLAUDE.md must exist and contain the blueprint context body
	data, err := os.ReadFile(claudeMDPath)
	require.NoError(t, err, "CLAUDE.md must be written after blueprint apply")
	assert.Contains(t, string(data), "Task-specific blueprint instructions.",
		"CLAUDE.md must contain the blueprint context body")

	// State manifest must record root:CLAUDE.md as an artifact
	manifest, err := state.ReadState(stateFile)
	require.NoError(t, err, "state file must exist after blueprint apply")
	targetState, ok := manifest.Targets["claude"]
	require.True(t, ok, "state must have claude target entry")

	var hasRootCLAUDEMD bool
	for _, a := range targetState.Artifacts {
		if a.Path == "root:CLAUDE.md" {
			hasRootCLAUDEMD = true
			break
		}
	}
	assert.True(t, hasRootCLAUDEMD, "state manifest must record root:CLAUDE.md as an artifact")

	// Second blueprint apply — CLAUDE.md must not be purged
	err = applyScope(projectXcaf, outputDir, dir, "project")
	require.NoError(t, err, "second blueprint apply must succeed")

	_, statErr := os.Stat(claudeMDPath)
	assert.NoError(t, statErr, "CLAUDE.md must still exist after second blueprint apply (must not be orphan-purged)")
}
