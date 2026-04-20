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
	// Prevent the parser from loading the real ~/.xcaffold/global.xcf, which
	// may contain fields not present in the current AST and would cause parse
	// failures unrelated to what this test exercises.
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	dir := t.TempDir()
	xcf := filepath.Join(dir, "project.xcf")
	content := `---
kind: project
version: "1.0"
name: check-perm-test
---
kind: settings
version: "1.0"
permissions:
  deny: [Bash]
---
kind: global
version: "1.0"
agents:
  dev:
    description: Developer
    permission-mode: plan
    disallowed-tools: [Write]
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
	globalFlag = false
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

// TestApply_CheckPermissions_GeminiTarget_ReportsDroppedFields verifies that
// --check-permissions --target gemini emits [WARNING] lines for fields that the
// Gemini renderer drops (effort, permission-mode, isolation), and exits 0.
func TestApply_CheckPermissions_GeminiTarget_ReportsDroppedFields(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	dir := t.TempDir()
	xcf := filepath.Join(dir, "project.xcf")
	content := `---
kind: project
version: "1.0"
name: gemini-check-perm-test
---
kind: global
version: "1.0"
agents:
  dev:
    description: Developer
    effort: high
    permission-mode: plan
    isolation: container
`
	require.NoError(t, os.WriteFile(xcf, []byte(content), 0600))

	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	xcfPath = xcf
	globalFlag = false
	applyCheckPermissions = true
	targetFlag = "gemini"
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

	assert.NoError(t, runErr, "check-permissions with warnings only must exit 0")

	output := buf.String()
	assert.Contains(t, output, "[WARNING]", "at least one warning must be emitted")
	assert.Contains(t, output, "gemini", "warning must reference the gemini target")
}

// TestApply_CheckPermissions_ContradictionExitsOne verifies that a config with
// the same rule in both allow and deny causes the parse to fail (via
// validatePermissions), and the CLI exits non-zero with an appropriate message.
func TestApply_CheckPermissions_ContradictionExitsOne(t *testing.T) {
	// Prevent the parser from loading the real ~/.xcaffold/global.xcf, which
	// may contain fields not present in the current AST and would cause parse
	// failures unrelated to what this test exercises.
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	dir := t.TempDir()
	xcf := filepath.Join(dir, "project.xcf")
	content := `---
kind: project
version: "1.0"
name: contradiction-test
---
kind: settings
version: "1.0"
permissions:
  allow: [Bash]
  deny: [Bash]
`
	require.NoError(t, os.WriteFile(xcf, []byte(content), 0600))

	xcfPath = xcf
	globalFlag = false
	applyCheckPermissions = true
	defer func() {
		applyCheckPermissions = false
	}()

	runErr := runApply(nil, nil)
	require.Error(t, runErr, "contradictory permissions must produce a non-zero exit")
	assert.Contains(t, runErr.Error(), "allow and deny", fmt.Sprintf("got: %v", runErr))
}

// TestApply_CheckPermissions_Global_UsesGlobalXcfHome verifies that
// --check-permissions --global parses globalXcfHome, not filepath.Dir(xcfPath).
// When --global is set, xcfPath is empty, so filepath.Dir("") == "." (CWD),
// which is the wrong directory. The fix must use globalXcfHome instead.
func TestApply_CheckPermissions_Global_UsesGlobalXcfHome(t *testing.T) {
	// Create a temp dir to serve as globalXcfHome with a valid project.xcf.
	globalHome := t.TempDir()
	xcf := filepath.Join(globalHome, "project.xcf")
	content := `---
kind: project
version: "1.0"
name: global-check-perm-test
---
kind: settings
version: "1.0"
permissions:
  deny: [Bash]
---
kind: global
version: "1.0"
agents:
  dev:
    description: Developer
    permission-mode: plan
`
	require.NoError(t, os.WriteFile(xcf, []byte(content), 0600))

	// A separate temp dir to act as CWD — it has no project.xcf.
	// If the bug is present, ParseDirectory(".") (or the CWD) will be used
	// and the call will fail with a "no such file" or parse error.
	cwdStand := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(cwdStand))
	defer func() { _ = os.Chdir(origDir) }()

	// Save and restore package-level state.
	origXcfPath := xcfPath
	origGlobalFlag := globalFlag
	origGlobalXcfHome := globalXcfHome
	origGlobalXcfPath := globalXcfPath
	origCheckPerms := applyCheckPermissions
	origTarget := targetFlag
	defer func() {
		xcfPath = origXcfPath
		globalFlag = origGlobalFlag
		globalXcfHome = origGlobalXcfHome
		globalXcfPath = origGlobalXcfPath
		applyCheckPermissions = origCheckPerms
		targetFlag = origTarget
	}()

	xcfPath = ""
	globalFlag = true
	globalXcfHome = globalHome
	globalXcfPath = xcf
	applyCheckPermissions = true
	targetFlag = targetClaude

	runErr := runApply(nil, nil)
	assert.NoError(t, runErr, "--check-permissions --global must succeed when globalXcfHome has a valid project.xcf")
}
