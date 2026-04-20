package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiffScope_ReadsV3StateManifest(t *testing.T) {
	base := t.TempDir()
	outputDir := filepath.Join(base, ".claude")
	require.NoError(t, os.MkdirAll(filepath.Join(outputDir, "agents"), 0755))

	artifactContent := "# dev agent"
	require.NoError(t, os.WriteFile(filepath.Join(outputDir, "agents", "dev.md"), []byte(artifactContent), 0644))

	h := sha256.Sum256([]byte(artifactContent))
	m := &state.StateManifest{
		Version:         1,
		XcaffoldVersion: "1.2.0",
		Targets: map[string]state.TargetState{
			"claude": {
				LastApplied: "2026-04-20T00:00:00Z",
				Artifacts: []state.Artifact{
					{Path: "agents/dev.md", Hash: fmt.Sprintf("sha256:%x", h)},
				},
			},
		},
	}
	statePath := state.StateFilePath(base, "")
	require.NoError(t, state.WriteState(m, statePath))

	drift, err := diffScope(outputDir, statePath, "claude", "project")
	require.NoError(t, err)
	assert.Equal(t, 0, drift, "clean artifact should report 0 drift")
}

func TestDiffScope_DetectsDriftedArtifact(t *testing.T) {
	base := t.TempDir()
	outputDir := filepath.Join(base, ".claude")
	require.NoError(t, os.MkdirAll(filepath.Join(outputDir, "agents"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(outputDir, "agents", "dev.md"), []byte("# modified"), 0644))

	m := &state.StateManifest{
		Version: 1,
		Targets: map[string]state.TargetState{
			"claude": {
				Artifacts: []state.Artifact{
					{Path: "agents/dev.md", Hash: "sha256:originalHash"},
				},
			},
		},
	}
	statePath := state.StateFilePath(base, "")
	require.NoError(t, state.WriteState(m, statePath))

	drift, err := diffScope(outputDir, statePath, "claude", "project")
	require.NoError(t, err)
	assert.Equal(t, 1, drift)
}

func TestDiffScope_MissingStateFile(t *testing.T) {
	_, err := diffScope("/tmp/nonexistent/.claude", "/tmp/nonexistent/.xcaffold/project.xcf.state", "claude", "project")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project.xcf.state")
}

func TestDiffScope_MissingTarget(t *testing.T) {
	base := t.TempDir()
	outputDir := filepath.Join(base, ".claude")

	m := &state.StateManifest{
		Version: 1,
		Targets: map[string]state.TargetState{
			"cursor": {
				Artifacts: []state.Artifact{},
			},
		},
	}
	statePath := state.StateFilePath(base, "")
	require.NoError(t, state.WriteState(m, statePath))

	_, err := diffScope(outputDir, statePath, "claude", "project")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "claude")
}
