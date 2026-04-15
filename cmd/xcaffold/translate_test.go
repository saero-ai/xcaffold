package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTranslateCmd_RequiresFrom(t *testing.T) {
	translateFrom = ""
	translateTo = "claude"
	err := runTranslate(translateCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "from")
}

func TestTranslateCmd_RequiresTo(t *testing.T) {
	translateFrom = "antigravity"
	translateTo = ""
	err := runTranslate(translateCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "to")
}

func TestTranslateCmd_InvalidFromProvider(t *testing.T) {
	translateFrom = "invalid-provider"
	translateTo = "claude"
	err := runTranslate(translateCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid-provider")
}

func TestTranslateCmd_InvalidToProvider(t *testing.T) {
	translateFrom = "claude"
	translateTo = "unknown"
	err := runTranslate(translateCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown")
}

func TestTranslateCmd_FidelityModeValidation(t *testing.T) {
	translateFrom = "claude"
	translateTo = "cursor"
	translateFidelity = "invalid"
	err := runTranslate(translateCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fidelity")
}

func TestTranslateCmd_DiffFormatValidation(t *testing.T) {
	translateFrom = "claude"
	translateTo = "cursor"
	translateFidelity = "warn"
	translateDiff = true
	translateDiffFormat = "xml"
	err := runTranslate(translateCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "diff-format")
}

func TestTranslateCmd_SaveXcf_WritesFile(t *testing.T) {
	// Build a minimal .claude/rules/ source directory.
	srcDir := t.TempDir()
	rulesDir := filepath.Join(srcDir, "rules")
	require.NoError(t, os.MkdirAll(rulesDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(rulesDir, "style.md"),
		[]byte("# Style\n\nUse consistent naming."),
		0o644,
	))

	xcfOut := filepath.Join(t.TempDir(), "out.xcf")

	// Reset global flag state before the call.
	translateFrom = "claude"
	translateTo = "antigravity"
	translateSourceDir = srcDir
	translateXcf = ""
	translateSaveXcf = xcfOut
	translateDryRun = true
	translateFidelity = "warn"
	translateDiff = false
	translateDiffFormat = "unified"

	err := runTranslate(translateCmd, []string{})
	require.NoError(t, err)
	require.FileExists(t, xcfOut, "--save-xcf path must exist after dry-run")
}

func TestTranslateCmd_XcfFlag_SkipsImport(t *testing.T) {
	// Write a minimal scaffold.xcf so the parser has something to load.
	// The kind: project document is flat — name is a top-level field, not
	// nested under a "project:" key.
	xcfDir := t.TempDir()
	xcfPath := filepath.Join(xcfDir, "scaffold.xcf")
	minimalXcf := `kind: project
version: "1.0"
name: test-project
`
	require.NoError(t, os.WriteFile(xcfPath, []byte(minimalXcf), 0o644))

	// Reset global flag state.
	translateFrom = "claude"
	translateTo = "antigravity"
	translateSourceDir = "" // must be ignored when --xcf is set
	translateXcf = xcfPath
	translateSaveXcf = ""
	translateDryRun = true
	translateFidelity = "warn"
	translateDiff = false
	translateDiffFormat = "unified"

	err := runTranslate(translateCmd, []string{})
	require.NoError(t, err, "--xcf bypass must succeed without --source-dir")
}
