package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunSchemaVersionMigrations_AlreadyCurrent(t *testing.T) {
	dir := t.TempDir()
	xcfFile := filepath.Join(dir, "scaffold.xcf")

	// Write a v1.0 config (already current)
	content := `version: "1.0"
project:
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

	err = runSchemaVersionMigrations(cmd, "scaffold.xcf")
	require.NoError(t, err)

	// No bak file should have been written
	_, statErr := os.Stat("scaffold.xcf.bak")
	assert.True(t, os.IsNotExist(statErr), "scaffold.xcf.bak should NOT be written for current version")

	assert.Contains(t, buf.String(), "current")
}

func TestRunSchemaVersionMigrations_NoMigrations(t *testing.T) {
	dir := t.TempDir()
	xcfFile := filepath.Join(dir, "scaffold.xcf")

	// Write a v1.0 config — with empty migrations slice no migration runs
	content := `version: "1.0"
project:
  name: "test-project"
`
	require.NoError(t, os.WriteFile(xcfFile, []byte(content), 0600))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(origDir) }()

	err = runSchemaVersionMigrations(nil, "scaffold.xcf")
	require.NoError(t, err)

	// No backup should be created when no migrations run
	_, statErr := os.Stat("scaffold.xcf.bak")
	assert.True(t, os.IsNotExist(statErr), "scaffold.xcf.bak should NOT be created when no migrations apply")
}
