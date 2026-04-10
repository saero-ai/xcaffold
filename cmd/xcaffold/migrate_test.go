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

func TestMigrateTo1_1_FieldRename(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{
			Test: ast.TestConfig{
				ClaudePath: "claude",
				CliPath:    "",
			},
		},
	}
	err := migrateTo1_1(config)
	require.NoError(t, err)
	assert.Equal(t, "1.1", config.Version)
	assert.Equal(t, "claude", config.Project.Test.CliPath)
	assert.Equal(t, "", config.Project.Test.ClaudePath)
}

func TestMigrateTo1_1_CliPathAlreadySet(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{
			Test: ast.TestConfig{
				ClaudePath: "old",
				CliPath:    "new",
			},
		},
	}
	err := migrateTo1_1(config)
	require.NoError(t, err)
	assert.Equal(t, "1.1", config.Version)
	assert.Equal(t, "new", config.Project.Test.CliPath)
	assert.Equal(t, "", config.Project.Test.ClaudePath)
}

func TestMigrateTo1_1_NeitherPathSet(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{
			Test: ast.TestConfig{
				ClaudePath: "",
				CliPath:    "",
			},
		},
	}
	err := migrateTo1_1(config)
	require.NoError(t, err)
	assert.Equal(t, "1.1", config.Version)
	assert.Equal(t, "", config.Project.Test.CliPath)
	assert.Equal(t, "", config.Project.Test.ClaudePath)
}

func TestRunSchemaVersionMigrations_Idempotent(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	dir := t.TempDir()
	xcfFile := filepath.Join(dir, "scaffold.xcf")

	// Write a v1.0 config with claude_path
	content := `version: "1.0"
project:
  name: "test-project"
  test:
    claude_path: "claude"
`
	require.NoError(t, os.WriteFile(xcfFile, []byte(content), 0600))

	// Change to the temp dir so runSchemaVersionMigrations finds "scaffold.xcf"
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(origDir) }()

	// First run — should apply migration
	err = runSchemaVersionMigrations(nil, "scaffold.xcf")
	require.NoError(t, err)

	// Verify the bak file was written on the first run
	_, err = os.Stat("scaffold.xcf.bak")
	assert.NoError(t, err, "scaffold.xcf.bak should exist after first run")

	// Remove the bak to detect if a second run writes another one
	require.NoError(t, os.Remove("scaffold.xcf.bak"))

	// Second run — version is now 1.1, no migration should run
	err = runSchemaVersionMigrations(nil, "scaffold.xcf")
	require.NoError(t, err)

	// No new bak should have been created on second run
	_, err = os.Stat("scaffold.xcf.bak")
	assert.True(t, os.IsNotExist(err), "scaffold.xcf.bak should NOT be written on second run")

	// Verify the final file has version 1.1 and cli_path
	data, err := os.ReadFile("scaffold.xcf")
	require.NoError(t, err)
	assert.Contains(t, string(data), "1.1")
	assert.Contains(t, string(data), "cli_path")
	assert.NotContains(t, string(data), "claude_path")
}

func TestRunSchemaVersionMigrations_AlreadyCurrent(t *testing.T) {
	dir := t.TempDir()
	xcfFile := filepath.Join(dir, "scaffold.xcf")

	// Write a v1.1 config (already current)
	content := `version: "1.1"
project:
  name: "test-project"
`
	require.NoError(t, os.WriteFile(xcfFile, []byte(content), 0600))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(origDir) }()

	// Capture output via a fake cmd
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	err = runSchemaVersionMigrations(cmd, "scaffold.xcf")
	require.NoError(t, err)

	// No bak file should have been written
	_, statErr := os.Stat("scaffold.xcf.bak")
	assert.True(t, os.IsNotExist(statErr), "scaffold.xcf.bak should NOT be written for current version")

	assert.Contains(t, buf.String(), "current")
}
