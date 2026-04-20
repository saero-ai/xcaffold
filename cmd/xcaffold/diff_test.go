package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/compiler"
	"github.com/saero-ai/xcaffold/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiffScope_ReportsSourceChanges(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, ".claude")
	require.NoError(t, os.MkdirAll(filepath.Join(outputDir, "agents"), 0755))

	// Write an output file
	agentContent := "# Dev agent"
	require.NoError(t, os.WriteFile(filepath.Join(outputDir, "agents", "dev.md"), []byte(agentContent), 0600))

	// Write source file
	srcFile := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(srcFile, []byte("version: \"1\"\n"), 0600))

	// Create state with matching artifact hash and source hash
	out := &compiler.Output{Files: map[string]string{"agents/dev.md": agentContent}}
	manifest := state.GenerateState(out, state.StateOpts{
		Target:      "claude",
		BaseDir:     dir,
		SourceFiles: []string{srcFile},
	}, nil)

	statePath := state.StateFilePath(dir, "")
	require.NoError(t, state.WriteState(manifest, statePath))

	// No drift — artifacts match
	driftCount, err := diffScope(outputDir, statePath, "claude", "project")
	require.NoError(t, err)
	assert.Equal(t, 0, driftCount)

	// Modify source file -> should report drift
	require.NoError(t, os.WriteFile(srcFile, []byte("version: \"1\"\nproject:\n  name: changed\n"), 0600))
	driftCount, err = diffScope(outputDir, statePath, "claude", "project")
	require.NoError(t, err)
	assert.Equal(t, 1, driftCount, "source change should increment drift count")
}
