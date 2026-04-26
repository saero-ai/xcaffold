package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/resolver"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWalkUp_FindsFileWithinHome verifies the walk-up locates a directory
// containing project.xcf when it sits inside the home boundary.
func TestWalkUp_FindsFileWithinHome(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	project := filepath.Join(home, "project")
	sub := filepath.Join(project, "sub")
	require.NoError(t, os.MkdirAll(sub, 0755))

	xcf := filepath.Join(project, "project.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte("version: \"1\"\n"), 0600))

	got, err := resolver.FindConfigDir(sub, home)
	require.NoError(t, err)
	assert.Equal(t, project, got, "should find directory containing project.xcf")
}

// TestWalkUp_StopsAtHome verifies the walk-up does NOT traverse above $HOME.
func TestWalkUp_StopsAtHome(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	sub := filepath.Join(home, "sub")
	require.NoError(t, os.MkdirAll(sub, 0755))

	xcfAboveHome := filepath.Join(root, "project.xcf")
	require.NoError(t, os.WriteFile(xcfAboveHome, []byte("version: \"1\"\n"), 0600))

	_, err := resolver.FindConfigDir(sub, home)
	assert.Error(t, err, "walk-up must not cross the home boundary")
}

// TestWalkUp_FindsAtHome verifies project.xcf at $HOME itself is found.
func TestWalkUp_FindsAtHome(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	sub := filepath.Join(home, "sub")
	require.NoError(t, os.MkdirAll(sub, 0755))

	require.NoError(t, os.WriteFile(filepath.Join(home, "project.xcf"), []byte("version: \"1\"\n"), 0600))

	got, err := resolver.FindConfigDir(sub, home)
	require.NoError(t, err)
	assert.Equal(t, home, got, "project.xcf at $HOME itself must be found")
}

// TestWalkUp_CwdIsHome_NoXcf verifies fallback when cwd equals home and no xcf.
func TestWalkUp_CwdIsHome_NoXcf(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	require.NoError(t, os.MkdirAll(home, 0755))

	_, err := resolver.FindConfigDir(home, home)
	assert.Error(t, err, "should error when no xcf found")
}

// TestWalkUp_FindsAnyXcfFile verifies the walk-up finds agents.xcf (not just project.xcf).
func TestWalkUp_FindsAnyXcfFile(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	project := filepath.Join(home, "project")
	sub := filepath.Join(project, "sub")
	require.NoError(t, os.MkdirAll(sub, 0755))

	require.NoError(t, os.WriteFile(filepath.Join(project, "agents.xcf"), []byte("agents:\n"), 0600))

	got, err := resolver.FindConfigDir(sub, home)
	require.NoError(t, err)
	assert.Equal(t, project, got)
}

// TestWalkUp_FindsXcfDirectory verifies walk-up returns the directory when
// multiple .xcf files exist.
func TestWalkUp_FindsXcfDirectory(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	project := filepath.Join(home, "project")
	sub := filepath.Join(project, "sub")
	require.NoError(t, os.MkdirAll(sub, 0755))

	require.NoError(t, os.WriteFile(filepath.Join(project, "agents.xcf"), []byte("agents:\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(project, "rules.xcf"), []byte("rules:\n"), 0600))

	got, err := resolver.FindConfigDir(sub, home)
	require.NoError(t, err)
	assert.Equal(t, project, got)
}

func TestResolveProjectConfig_PrefersXcaffoldDir(t *testing.T) {
	configDir := t.TempDir()

	scaffoldDir := filepath.Join(configDir, ".xcaffold")
	require.NoError(t, os.MkdirAll(scaffoldDir, 0755))

	scaffoldXcf := filepath.Join(scaffoldDir, "project.xcf")
	require.NoError(t, os.WriteFile(scaffoldXcf, []byte("version: \"1\""), 0600))

	rootXcf := filepath.Join(configDir, "project.xcf")
	require.NoError(t, os.WriteFile(rootXcf, []byte("version: \"2\""), 0600))

	configFlag = configDir
	defer func() { configFlag = "" }()

	originalXcf := xcfPath
	defer func() { xcfPath = originalXcf }()

	oldProjectRoot := projectRoot
	defer func() { projectRoot = oldProjectRoot }()

	cmd := &cobra.Command{Use: "apply"}
	err := resolveProjectConfig(cmd)
	require.NoError(t, err)
	assert.Equal(t, scaffoldXcf, xcfPath)
}

func TestResolveProjectConfig_FallbackToRoot(t *testing.T) {
	configDir := t.TempDir()

	rootXcf := filepath.Join(configDir, "project.xcf")
	require.NoError(t, os.WriteFile(rootXcf, []byte("version: \"1\""), 0600))

	configFlag = configDir
	defer func() { configFlag = "" }()

	originalXcf := xcfPath
	defer func() { xcfPath = originalXcf }()

	cmd := &cobra.Command{Use: "apply"}
	err := resolveProjectConfig(cmd)
	require.NoError(t, err)
	assert.Equal(t, rootXcf, xcfPath)
}

func TestProjectParseRoot_StandardLayout(t *testing.T) {
	oldXcfPath := xcfPath
	defer func() { xcfPath = oldXcfPath }()
	oldProjectRoot := projectRoot
	defer func() { projectRoot = oldProjectRoot }()
	projectRoot = ""
	xcfPath = "/foo/bar/.xcaffold/project.xcf"
	assert.Equal(t, "/foo/bar", projectParseRoot())
}

func TestProjectParseRoot_LegacyRootLayout(t *testing.T) {
	oldXcfPath := xcfPath
	defer func() { xcfPath = oldXcfPath }()
	oldProjectRoot := projectRoot
	defer func() { projectRoot = oldProjectRoot }()
	projectRoot = ""
	xcfPath = "/foo/bar/project.xcf"
	assert.Equal(t, "/foo/bar", projectParseRoot())
}
