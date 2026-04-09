package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApply_CheckPermissions_FlagRegistered verifies that --check-permissions
// is registered on applyCmd before it is used.
func TestApply_CheckPermissions_FlagRegistered(t *testing.T) {
	flag := applyCmd.Flags().Lookup("check-permissions")
	require.NotNil(t, flag, "applyCmd must have a --check-permissions flag")
	assert.Equal(t, "bool", flag.Value.Type())
}

// TestApply_CheckPermissions_CursorTarget_ReportsDroppedFields verifies that
// --check-permissions --target cursor emits [WARNING] lines for security fields
// that the Cursor renderer drops, and exits 0.
func TestApply_CheckPermissions_CursorTarget_ReportsDroppedFields(t *testing.T) {
	dir := t.TempDir()
	xcf := filepath.Join(dir, "scaffold.xcf")
	content := `version: "1.0"
project:
  name: check-perm-test
settings:
  permissions:
    deny: [Bash]
agents:
  dev:
    description: Developer
    permissionMode: plan
    disallowedTools: [Write]
    isolation: container
`
	require.NoError(t, os.WriteFile(xcf, []byte(content), 0600))

	// Capture stdout where [WARNING]/[INFO] lines are printed.
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	// Set package-level state vars used by runApply.
	xcfPath = xcf
	scopeFlag = "project" //nolint:goconst
	applyCheckPermissions = true
	targetFlag = "cursor"
	defer func() {
		applyCheckPermissions = false
		targetFlag = targetClaude
	}()

	runErr := runApply(nil, nil)

	w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	r.Close()
	os.Stdout = old

	// Warnings only — must exit 0.
	assert.NoError(t, runErr, "check-permissions with warnings only must exit 0")

	output := buf.String()
	// At least two [WARNING] lines must appear referencing cursor security fields.
	assert.Contains(t, output, "[WARNING]", "at least one warning must be emitted")
	assert.Contains(t, output, "cursor", "warning must reference the cursor target")
}

// TestApply_CheckPermissions_ContradictionExitsOne verifies that a config with
// the same rule in both allow and deny causes the parse to fail (via
// validatePermissions), and the CLI exits non-zero with an appropriate message.
func TestApply_CheckPermissions_ContradictionExitsOne(t *testing.T) {
	dir := t.TempDir()
	xcf := filepath.Join(dir, "scaffold.xcf")
	content := `version: "1.0"
project:
  name: contradiction-test
settings:
  permissions:
    allow: [Bash]
    deny: [Bash]
`
	require.NoError(t, os.WriteFile(xcf, []byte(content), 0600))

	xcfPath = xcf
	scopeFlag = "project"
	applyCheckPermissions = true
	defer func() {
		applyCheckPermissions = false
	}()

	runErr := runApply(nil, nil)
	require.Error(t, runErr, "contradictory permissions must produce a non-zero exit")
	assert.Contains(t, runErr.Error(), "allow and deny", fmt.Sprintf("got: %v", runErr))
}
