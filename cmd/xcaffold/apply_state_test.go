package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/output"
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
// ── cleanOrphans root-file boundary ───────────────────────────────────────────

// TestCleanOrphans_RootFileOrphan_UsesBaseDirBoundary verifies that when a
// root: prefixed orphan is deleted, cleanEmptyDirsUpToTarget is called with
// ctx.baseDir as the boundary, not ctx.outputDir. This prevents cleanOrphans
// from attempting to remove directories outside the output dir when cleaning
// up root-scoped artifacts such as CLAUDE.md.
func TestCleanOrphans_RootFileOrphan_UsesBaseDirBoundary(t *testing.T) {
	base := t.TempDir()
	outputDir := filepath.Join(base, ".claude")
	require.NoError(t, os.MkdirAll(outputDir, 0755))

	// Place an orphan root file in a subdirectory under baseDir (not outputDir).
	// If cleanEmptyDirsUpToTarget used outputDir as boundary, it would attempt
	// to walk above the subdir into baseDir and could wrongly delete directories.
	subDir := filepath.Join(base, "docs", "generated")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	orphanFile := filepath.Join(subDir, "context.md")
	require.NoError(t, os.WriteFile(orphanFile, []byte("old content"), 0644))

	oldManifest := &state.StateManifest{
		Targets: map[string]state.TargetState{
			"claude": {
				Artifacts: []state.Artifact{
					{Path: "root:docs/generated/context.md"},
				},
			},
		},
	}

	oldTarget := targetFlag
	targetFlag = "claude"
	defer func() { targetFlag = oldTarget }()

	ctx := &applyContext{
		oldManifest: oldManifest,
		out:         &output.Output{Files: map[string]string{}, RootFiles: map[string]string{}},
		baseDir:     base,
		outputDir:   outputDir,
	}

	hasChanges := false
	ctx.cleanOrphans(&hasChanges)

	// The orphan file must be deleted.
	_, statErr := os.Stat(orphanFile)
	assert.True(t, os.IsNotExist(statErr), "orphan root file must be deleted")

	// The subdir and its parent must be cleaned (they were empty after deletion).
	_, statErr = os.Stat(subDir)
	assert.True(t, os.IsNotExist(statErr), "empty orphan subdirectory must be cleaned")

	// baseDir itself must not be deleted — it is the boundary.
	_, statErr = os.Stat(base)
	assert.NoError(t, statErr, "baseDir must not be removed during orphan cleanup")

	// outputDir must not be touched — orphan was root-scoped, not output-scoped.
	_, statErr = os.Stat(outputDir)
	assert.NoError(t, statErr, "outputDir must not be removed during root-orphan cleanup")

	assert.True(t, hasChanges, "hasChanges must be set when orphan is deleted")
}

// ── countOrphansFromState root file awareness ────────────────────────────────

// TestCountOrphans_RootFilePresent_NotCountedAsOrphan verifies that a root:
// artifact whose relative path exists in rootFiles is not counted as an orphan.
func TestCountOrphans_RootFilePresent_NotCountedAsOrphan(t *testing.T) {
	oldManifest := &state.StateManifest{
		Targets: map[string]state.TargetState{
			"claude": {
				Artifacts: []state.Artifact{
					{Path: "agents/dev.md"},
					{Path: "root:CLAUDE.md"},
				},
			},
		},
	}
	outFiles := map[string]string{"agents/dev.md": "content"}
	rootFiles := map[string]string{"CLAUDE.md": "instructions"}

	count := countOrphansFromState(oldManifest, "claude", outFiles, rootFiles)
	assert.Equal(t, 0, count, "root file present in rootFiles must not be counted as orphan")
}

// TestCountOrphans_RootFileAbsent_CountedAsOrphan verifies that a root:
// artifact whose relative path is absent from rootFiles is counted as an orphan.
func TestCountOrphans_RootFileAbsent_CountedAsOrphan(t *testing.T) {
	oldManifest := &state.StateManifest{
		Targets: map[string]state.TargetState{
			"claude": {
				Artifacts: []state.Artifact{
					{Path: "agents/dev.md"},
					{Path: "root:CLAUDE.md"},
				},
			},
		},
	}
	// Both outFiles and rootFiles match — only CLAUDE.md is absent from rootFiles.
	outFiles := map[string]string{"agents/dev.md": "content"}
	rootFiles := map[string]string{} // CLAUDE.md not present

	count := countOrphansFromState(oldManifest, "claude", outFiles, rootFiles)
	assert.Equal(t, 1, count, "root file absent from rootFiles must be counted as orphan")
}

// ── End-to-end regression test ────────────────────────────────────────────────

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
