package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestListCmd_FlagsRegistered verifies --blueprint and --resolved are registered.
func TestListCmd_FlagsRegistered(t *testing.T) {
	bp := listCmd.Flags().Lookup("blueprint")
	require.NotNil(t, bp, "--blueprint flag must be registered on listCmd")
	assert.Equal(t, "string", bp.Value.Type())

	res := listCmd.Flags().Lookup("resolved")
	require.NotNil(t, res, "--resolved flag must be registered on listCmd")
	assert.Equal(t, "bool", res.Value.Type())
}

// TestRunList_NoXcfDir_ReturnsError verifies that runList returns an error
// when it cannot locate a project.xcf in the working tree.
func TestRunList_NoXcfDir_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	// Point xcfPath at a non-existent file so the command has nothing to parse.
	origXcfPath := xcfPath
	xcfPath = filepath.Join(dir, "project.xcf")
	defer func() { xcfPath = origXcfPath }()

	var buf bytes.Buffer
	listCmd.SetOut(&buf)
	defer listCmd.SetOut(nil)

	err := runList(listCmd, []string{})
	assert.Error(t, err)
}

// TestRunList_EmptyProject_ShowsHeaders verifies that a minimal project with no
// resources prints the "Resources:" and "Blueprints:" section headers.
func TestRunList_EmptyProject_ShowsHeaders(t *testing.T) {
	dir := t.TempDir()
	xcfContent := "kind: project\nversion: \"1.0\"\nname: list-test\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte(xcfContent), 0600))

	origXcfPath := xcfPath
	xcfPath = filepath.Join(dir, "project.xcf")
	defer func() { xcfPath = origXcfPath }()

	var buf bytes.Buffer
	listCmd.SetOut(&buf)
	defer listCmd.SetOut(nil)

	err := runList(listCmd, []string{})
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "Resources:")
	assert.Contains(t, out, "Blueprints:")
}

// TestRunList_WithAgentsAndSkills_ShowsResources verifies that the resources
// section lists agent and skill names. The parser merges the global base config
// so exact counts depend on the user's environment; we assert named resources
// from the local xcf appear in the output.
func TestRunList_WithAgentsAndSkills_ShowsResources(t *testing.T) {
	dir := t.TempDir()
	// Use separate xcf files to avoid the multi-doc project + global format.
	projectXcf := "kind: project\nversion: \"1.0\"\nname: list-count-test\n"
	globalXcf := `kind: global
version: "1.0"
agents:
  developer:
    description: Developer
    model: claude-sonnet-4-5
skills:
  tdd-local:
    description: TDD
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte(projectXcf), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "global.xcf"), []byte(globalXcf), 0600))

	origXcfPath := xcfPath
	xcfPath = filepath.Join(dir, "project.xcf")
	defer func() { xcfPath = origXcfPath }()

	var buf bytes.Buffer
	listCmd.SetOut(&buf)
	defer listCmd.SetOut(nil)

	err := runList(listCmd, []string{})
	require.NoError(t, err)

	out := buf.String()
	// The resources section header must be present.
	assert.Contains(t, out, "agents:")
	assert.Contains(t, out, "skills:")
	// The locally declared resources must appear.
	assert.Contains(t, out, "developer")
	assert.Contains(t, out, "tdd-local")
}

// TestRunList_WithBlueprint_FiltersOutput verifies that --blueprint filters
// resources to only those belonging to the named blueprint.
func TestRunList_WithBlueprint_FiltersOutput(t *testing.T) {
	dir := t.TempDir()
	// project.xcf is the entry-point document.
	projectXcf := "kind: project\nversion: \"1.0\"\nname: list-blueprint-test\n"
	// Agents in a separate global doc.
	globalXcf := `kind: global
version: "1.0"
agents:
  developer:
    description: Developer
    model: claude-sonnet-4-5
  designer:
    description: Designer
    model: claude-sonnet-4-5
`
	// Blueprint as a standalone kind: blueprint document.
	bpXcf := `kind: blueprint
version: "1.0"
name: backend
agents:
  - developer
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte(projectXcf), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "global.xcf"), []byte(globalXcf), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "backend.xcf"), []byte(bpXcf), 0600))

	origXcfPath := xcfPath
	xcfPath = filepath.Join(dir, "project.xcf")
	listBlueprintFlag = "backend"
	defer func() {
		xcfPath = origXcfPath
		listBlueprintFlag = ""
	}()

	var buf bytes.Buffer
	listCmd.SetOut(&buf)
	defer listCmd.SetOut(nil)

	err := runList(listCmd, []string{})
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "backend")
	assert.Contains(t, out, "developer")
	// designer is NOT in the blueprint — should not appear in blueprint listing
	assert.NotContains(t, out, "designer")
}

// TestRunList_UnknownBlueprint_ReturnsError verifies an error is returned when
// --blueprint names a blueprint that does not exist in the config.
func TestRunList_UnknownBlueprint_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	xcfContent := "kind: project\nversion: \"1.0\"\nname: list-test\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte(xcfContent), 0600))

	origXcfPath := xcfPath
	xcfPath = filepath.Join(dir, "project.xcf")
	listBlueprintFlag = "nonexistent"
	defer func() {
		xcfPath = origXcfPath
		listBlueprintFlag = ""
	}()

	var buf bytes.Buffer
	listCmd.SetOut(&buf)
	defer listCmd.SetOut(nil)

	err := runList(listCmd, []string{})
	assert.Error(t, err)
}

// TestRegistryCmd_IsRegistered verifies the registry command is accessible
// from the root command with its new name.
func TestRegistryCmd_IsRegistered(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"registry"})
	require.NoError(t, err)
	assert.Equal(t, "registry", cmd.Use)
}

// TestListCmd_IsRegistered verifies the list command is still accessible
// from the root command after the rename.
func TestListCmd_IsRegistered(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"list"})
	require.NoError(t, err)
	assert.Equal(t, "list", cmd.Use)
}
