package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/saero-ai/xcaffold/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunStatus_NoStateFile(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	err := runStatusWithWriter(dir, "", &buf)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No state found")
}

func TestRunStatus_AllClean(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, ".claude")
	require.NoError(t, os.MkdirAll(filepath.Join(outputDir, "agents"), 0755))

	artifactContent := "# dev"
	require.NoError(t, os.WriteFile(filepath.Join(outputDir, "agents", "dev.md"), []byte(artifactContent), 0644))

	h := sha256.Sum256([]byte(artifactContent))
	m := &state.StateManifest{
		Version:         1,
		XcaffoldVersion: "1.2.0",
		SourceFiles:     []state.SourceFile{{Path: "project.xcf", Hash: "sha256:abc"}},
		Targets: map[string]state.TargetState{
			"claude": {
				LastApplied: time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339),
				Artifacts: []state.Artifact{
					{Path: "agents/dev.md", Hash: fmt.Sprintf("sha256:%x", h)},
				},
			},
		},
	}
	require.NoError(t, state.WriteState(m, state.StateFilePath(dir, "")))

	var buf bytes.Buffer
	require.NoError(t, runStatusWithWriter(dir, "", &buf))

	out := buf.String()
	assert.Contains(t, out, "claude")
	assert.Contains(t, out, "all clean")
	assert.Contains(t, out, "1 artifact")
}

func TestRunStatus_ArtifactDrifted(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, ".claude")
	require.NoError(t, os.MkdirAll(filepath.Join(outputDir, "agents"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(outputDir, "agents", "dev.md"), []byte("# modified"), 0644))

	m := &state.StateManifest{
		Version: 1,
		Targets: map[string]state.TargetState{
			"claude": {
				LastApplied: time.Now().UTC().Format(time.RFC3339),
				Artifacts: []state.Artifact{
					{Path: "agents/dev.md", Hash: "sha256:originalHash"},
				},
			},
		},
	}
	require.NoError(t, state.WriteState(m, state.StateFilePath(dir, "")))

	var buf bytes.Buffer
	require.NoError(t, runStatusWithWriter(dir, "", &buf))
	assert.Contains(t, buf.String(), "drifted")
}

func TestRunStatus_SourceChanged(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(srcPath, []byte("---\nkind: project\nname: x\n---\n"), 0644))

	m := &state.StateManifest{
		Version: 1,
		SourceFiles: []state.SourceFile{
			{Path: "project.xcf", Hash: "sha256:staleHash"},
		},
		Targets: map[string]state.TargetState{
			"claude": {LastApplied: time.Now().UTC().Format(time.RFC3339)},
		},
	}
	require.NoError(t, state.WriteState(m, state.StateFilePath(dir, "")))

	var buf bytes.Buffer
	require.NoError(t, runStatusWithWriter(dir, "", &buf))
	assert.Contains(t, buf.String(), "changed")
	assert.Contains(t, buf.String(), "project.xcf")
}

func TestRunStatus_BlueprintFlag(t *testing.T) {
	dir := t.TempDir()
	m := &state.StateManifest{
		Version:   3,
		Blueprint: "backend",
		Targets: map[string]state.TargetState{
			"claude": {LastApplied: time.Now().UTC().Format(time.RFC3339)},
		},
	}
	require.NoError(t, state.WriteState(m, state.StateFilePath(dir, "backend")))

	var buf bytes.Buffer
	require.NoError(t, runStatusWithWriter(dir, "backend", &buf))
	assert.Contains(t, buf.String(), "backend")
}
