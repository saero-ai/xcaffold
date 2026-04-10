package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCmd_ValidConfig(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	dir := t.TempDir()
	xcf := filepath.Join(dir, "scaffold.xcf")
	content := `
version: "1.0"
project:
  name: "test"
agents:
  developer:
    description: "Dev agent"
    skills: [deploy]
skills:
  deploy:
    description: "Deploy skill"
`
	require.NoError(t, os.WriteFile(xcf, []byte(content), 0600))

	// Set the package-level xcfPath directly (PersistentPreRunE would normally do this)
	oldPath := xcfPath
	xcfPath = xcf
	defer func() { xcfPath = oldPath }()

	err := runValidate(validateCmd, []string{})
	assert.NoError(t, err)
}

func TestValidateCmd_InvalidCrossRef(t *testing.T) {
	dir := t.TempDir()
	xcf := filepath.Join(dir, "scaffold.xcf")
	content := `
version: "1.0"
project:
  name: "test"
agents:
  developer:
    description: "Dev agent"
    skills: [nonexistent]
`
	require.NoError(t, os.WriteFile(xcf, []byte(content), 0600))

	oldPath := xcfPath
	xcfPath = xcf
	defer func() { xcfPath = oldPath }()

	err := runValidate(validateCmd, []string{})
	assert.Error(t, err)
}

func TestValidate_GlobalFlag_FileNotFound(t *testing.T) {
	home := t.TempDir() // no .xcaffold/global.xcf inside
	t.Setenv("HOME", home)

	validateGlobal = true
	defer func() { validateGlobal = false }()

	err := runValidate(validateCmd, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "global.xcf not found")
}

func TestValidate_GlobalFlag_InvalidYAML(t *testing.T) {
	home := t.TempDir()
	xcaffoldDir := filepath.Join(home, ".xcaffold")
	require.NoError(t, os.MkdirAll(xcaffoldDir, 0700))

	globalXCF := filepath.Join(xcaffoldDir, "global.xcf")
	require.NoError(t, os.WriteFile(globalXCF, []byte(":\tinvalid: yaml: :::\n"), 0600))

	t.Setenv("HOME", home)
	validateGlobal = true
	defer func() { validateGlobal = false }()

	err := runValidate(validateCmd, []string{})
	require.Error(t, err)
}

func TestValidate_GlobalFlag_ValidFile(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	home := t.TempDir()
	xcaffoldDir := filepath.Join(home, ".xcaffold")
	require.NoError(t, os.MkdirAll(xcaffoldDir, 0700))

	globalXCF := filepath.Join(xcaffoldDir, "global.xcf")
	content := `version: "1.0"
project:
  name: global-test
`
	require.NoError(t, os.WriteFile(globalXCF, []byte(content), 0600))

	t.Setenv("HOME", home)
	validateGlobal = true
	defer func() { validateGlobal = false }()

	err := runValidate(validateCmd, []string{})
	assert.NoError(t, err)
}

func TestValidateCmd_StructuralChecks(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	dir := t.TempDir()
	xcf := filepath.Join(dir, "scaffold.xcf")
	content := `
version: "1.0"
project:
  name: "test"
agents:
  developer:
    description: "Dev agent"
skills:
  orphan:
    description: "No agent references this skill"
`
	require.NoError(t, os.WriteFile(xcf, []byte(content), 0600))

	oldPath := xcfPath
	xcfPath = xcf
	defer func() { xcfPath = oldPath }()

	validateStructural = true
	defer func() { validateStructural = false }()

	err := runValidate(validateCmd, []string{})
	// Structural checks warn but don't fail
	assert.NoError(t, err)
}
