package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunSchemaVersionMigrations_AlreadyCurrent(t *testing.T) {
	dir := t.TempDir()
	xcfFile := filepath.Join(dir, "project.xcf")

	// Write a v1.0 config (already current)
	content := `kind: project
version: "1.0"
name: "test-project"
`
	require.NoError(t, os.WriteFile(xcfFile, []byte(content), 0600))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(origDir) }()

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	err = runSchemaVersionMigrations(cmd, "project.xcf")
	require.NoError(t, err)

	// No bak file should have been written
	_, statErr := os.Stat("project.xcf.bak")
	assert.True(t, os.IsNotExist(statErr), "project.xcf.bak should NOT be written for current version")

	assert.Contains(t, buf.String(), "current")
}

func TestRunSchemaVersionMigrations_NoMigrations(t *testing.T) {
	dir := t.TempDir()
	xcfFile := filepath.Join(dir, "project.xcf")

	// Write a v1.0 config — with empty migrations slice no migration runs
	content := `kind: project
version: "1.0"
name: "test-project"
`
	require.NoError(t, os.WriteFile(xcfFile, []byte(content), 0600))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(origDir) }()

	err = runSchemaVersionMigrations(nil, "project.xcf")
	require.NoError(t, err)

	// No backup should be created when no migrations run
	_, statErr := os.Stat("project.xcf.bak")
	assert.True(t, os.IsNotExist(statErr), "project.xcf.bak should NOT be created when no migrations apply")
}

func TestMigrate_WritesSplitFiles(t *testing.T) {
	// Verify that when a migration rewrites the project, WriteSplitFiles produces
	// the expected split-file layout: project.xcf + xcf/agents/<id>.xcf.
	dir := t.TempDir()

	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{Name: "migrate-test"},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {Name: "dev", Description: "Dev agent"},
			},
		},
	}

	err := WriteSplitFiles(config, dir)
	require.NoError(t, err)

	// project.xcf must contain kind: project and the agent ref
	projData, readErr := os.ReadFile(filepath.Join(dir, "project.xcf"))
	require.NoError(t, readErr)
	assert.Contains(t, string(projData), "kind: project")
	assert.Contains(t, string(projData), "dev")

	// Agent must be written to its own file
	agentData, readErr := os.ReadFile(filepath.Join(dir, "xcf", "agents", "dev.xcf"))
	require.NoError(t, readErr)
	assert.Contains(t, string(agentData), "kind: agent")
	assert.Contains(t, string(agentData), "name: dev")
}
