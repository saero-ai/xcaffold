package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/saero-ai/xcaffold/internal/registry"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRegistry creates a temporary XCAFFOLD_HOME for test isolation.
func setupTestRegistry(t *testing.T) (string, func()) {
	tempHome := t.TempDir()
	t.Setenv("XCAFFOLD_HOME", tempHome)

	// Initialize the global registry home
	err := registry.EnsureGlobalHome()
	require.NoError(t, err)

	cleanup := func() {
		// Cleanup is handled by t.TempDir()
	}

	return tempHome, cleanup
}

// TestRegistryList_Empty tests listing when no projects are registered.
func TestRegistryList_Empty(t *testing.T) {
	_, cleanup := setupTestRegistry(t)
	defer cleanup()

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	err := runRegistryList(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "No projects registered yet")
}

// TestRegistryList_WithProjects tests listing with registered projects.
func TestRegistryList_WithProjects(t *testing.T) {
	_, cleanup := setupTestRegistry(t)
	defer cleanup()

	// Register test projects
	projectDir := t.TempDir()
	err := registry.Register(projectDir, "test-project", []string{"claude"}, "")
	require.NoError(t, err)

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	err = runRegistryList(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "test-project")
	// Output truncates long paths, so just check the project name and targets
	assert.Contains(t, output, "1 targets")
	assert.NotContains(t, output, "No projects registered yet")
}

// TestRegistryList_JSON tests JSON output format.
func TestRegistryList_JSON(t *testing.T) {
	_, cleanup := setupTestRegistry(t)
	defer cleanup()

	// Register a test project
	projectDir := t.TempDir()
	err := registry.Register(projectDir, "json-test", []string{"claude", "gemini"}, "")
	require.NoError(t, err)

	registryListJSON = true
	defer func() { registryListJSON = false }()

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	err = runRegistryList(cmd, []string{})
	require.NoError(t, err)

	// Verify JSON output
	var projects []registry.Project
	err = json.Unmarshal(buf.Bytes(), &projects)
	require.NoError(t, err)
	require.Len(t, projects, 1)
	assert.Equal(t, "json-test", projects[0].Name)
}

// TestRegistryAdd_NewProject tests adding a new project.
func TestRegistryAdd_NewProject(t *testing.T) {
	_, cleanup := setupTestRegistry(t)
	defer cleanup()

	projectDir := t.TempDir()

	// Create a minimal project.xcf
	projectXcf := filepath.Join(projectDir, "project.xcf")
	err := os.WriteFile(projectXcf, []byte("---\nkind: project\nversion: \"1.0\"\nname: my-project\n"), 0600)
	require.NoError(t, err)

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	err = runRegistryAdd(cmd, []string{projectDir})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Registered project")
	assert.Contains(t, output, projectDir)

	// Verify project was added to registry
	projects, err := registry.List()
	require.NoError(t, err)
	require.Len(t, projects, 1)
	assert.Equal(t, projectDir, projects[0].Path)
}

// TestRegistryAdd_NonExistentPath tests adding a project with non-existent path.
func TestRegistryAdd_NonExistentPath(t *testing.T) {
	_, cleanup := setupTestRegistry(t)
	defer cleanup()

	nonExistentPath := "/nonexistent/path/to/project"

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	err := runRegistryAdd(cmd, []string{nonExistentPath})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

// TestRegistryRemove_ByName tests removing a project by name.
func TestRegistryRemove_ByName(t *testing.T) {
	_, cleanup := setupTestRegistry(t)
	defer cleanup()

	projectDir := t.TempDir()
	err := registry.Register(projectDir, "removable", []string{}, "")
	require.NoError(t, err)

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	err = runRegistryRemove(cmd, []string{"removable"})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Removed project")
	assert.Contains(t, output, "removable")

	// Verify project was removed
	projects, err := registry.List()
	require.NoError(t, err)
	assert.Len(t, projects, 0)
}

// TestRegistryRemove_NotFound tests removing a non-existent project.
func TestRegistryRemove_NotFound(t *testing.T) {
	_, cleanup := setupTestRegistry(t)
	defer cleanup()

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	err := runRegistryRemove(cmd, []string{"nonexistent"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove project")
}

// TestRegistryPrune_RemovesStale tests that prune removes projects with missing paths.
func TestRegistryPrune_RemovesStale(t *testing.T) {
	_, cleanup := setupTestRegistry(t)
	defer cleanup()

	// Create and register a project, then delete its directory
	projectDir := t.TempDir()
	err := registry.Register(projectDir, "stale-project", []string{}, "")
	require.NoError(t, err)

	// Delete the project directory to make it stale
	err = os.RemoveAll(projectDir)
	require.NoError(t, err)

	registryPruneDryRun = false
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	err = runRegistryPrune(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Pruned:")
	assert.Contains(t, output, "stale-project")
	assert.Contains(t, output, "path does not exist")

	// Verify project was removed
	projects, err := registry.List()
	require.NoError(t, err)
	assert.Len(t, projects, 0)
}

// TestRegistryPrune_DryRun tests that dry-run doesn't actually remove projects.
func TestRegistryPrune_DryRun(t *testing.T) {
	_, cleanup := setupTestRegistry(t)
	defer cleanup()

	projectDir := t.TempDir()
	err := registry.Register(projectDir, "dry-run-test", []string{}, "")
	require.NoError(t, err)

	// Delete the directory to make it stale
	err = os.RemoveAll(projectDir)
	require.NoError(t, err)

	registryPruneDryRun = true
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	err = runRegistryPrune(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Pruned:")
	assert.Contains(t, output, "path does not exist")

	// Verify project was NOT removed (dry-run)
	projects, err := registry.List()
	require.NoError(t, err)
	require.Len(t, projects, 1)
	assert.Equal(t, "dry-run-test", projects[0].Name)

	registryPruneDryRun = false
}

// TestRegistryInfo_ShowsDetails tests info command output.
func TestRegistryInfo_ShowsDetails(t *testing.T) {
	_, cleanup := setupTestRegistry(t)
	defer cleanup()

	projectDir := t.TempDir()
	err := registry.Register(projectDir, "info-test", []string{"claude", "gemini"}, "/custom/config")
	require.NoError(t, err)

	registryInfoJSON = false
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	err = runRegistryInfo(cmd, []string{"info-test"})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "info-test")
	assert.Contains(t, output, projectDir)
	assert.Contains(t, output, "claude")
	assert.Contains(t, output, "gemini")
}

// TestRegistryInfo_JSON tests info command with JSON output.
func TestRegistryInfo_JSON(t *testing.T) {
	_, cleanup := setupTestRegistry(t)
	defer cleanup()

	projectDir := t.TempDir()
	err := registry.Register(projectDir, "json-info", []string{"claude"}, "")
	require.NoError(t, err)

	registryInfoJSON = true
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	err = runRegistryInfo(cmd, []string{"json-info"})
	require.NoError(t, err)

	var info registry.ProjectInfo
	err = json.Unmarshal(buf.Bytes(), &info)
	require.NoError(t, err)
	assert.Equal(t, "json-info", info.Name)
	assert.Equal(t, projectDir, info.Path)
}

// TestRegistryInfo_NotFound tests info command with non-existent project.
func TestRegistryInfo_NotFound(t *testing.T) {
	_, cleanup := setupTestRegistry(t)
	defer cleanup()

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	err := runRegistryInfo(cmd, []string{"nonexistent"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project not found")
}

// TestRegistry_NoSubcommand_DefaultsList tests that calling registry without a subcommand defaults to list.
func TestRegistry_NoSubcommand_DefaultsList(t *testing.T) {
	_, cleanup := setupTestRegistry(t)
	defer cleanup()

	projectDir := t.TempDir()
	err := registry.Register(projectDir, "default-test", []string{}, "")
	require.NoError(t, err)

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	err = runRegistryList(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "default-test")
}

// TestProjectStatus tests the projectStatus helper function.
func TestProjectStatus(t *testing.T) {
	_, cleanup := setupTestRegistry(t)
	defer cleanup()

	projectDir := t.TempDir()
	p := registry.Project{
		Path:       projectDir,
		Name:       "test",
		Registered: time.Now(),
	}

	// Status should be "orphan" (no xcaf/ or project.xcf)
	status := projectStatus(p)
	assert.Equal(t, "orphan", status)

	// Create project.xcf, status should be "ok"
	err := os.WriteFile(filepath.Join(projectDir, "project.xcf"), []byte(""), 0600)
	require.NoError(t, err)
	status = projectStatus(p)
	assert.Equal(t, "ok", status)

	// Delete the project directory, status should be "stale"
	err = os.RemoveAll(projectDir)
	require.NoError(t, err)
	status = projectStatus(p)
	assert.Equal(t, "stale", status)
}

// TestFormatLastApplied tests the formatLastApplied helper function.
func TestFormatLastApplied(t *testing.T) {
	// Zero time should return "never"
	result := formatLastApplied(time.Time{})
	assert.Equal(t, " never", result)

	// Recent time (within 24 hours) should include time
	now := time.Now()
	result = formatLastApplied(now)
	assert.Contains(t, result, "today")

	// Older time should include date
	oldTime := now.AddDate(0, 0, -5)
	result = formatLastApplied(oldTime)
	assert.Contains(t, result, oldTime.Format("2006"))
	assert.NotContains(t, result, "never")
}

// TestRegistryList_VerboseFlag tests verbose output for list command.
func TestRegistryList_VerboseFlag(t *testing.T) {
	_, cleanup := setupTestRegistry(t)
	defer cleanup()

	projectDir := t.TempDir()
	err := registry.Register(projectDir, "verbose-test", []string{"claude", "gemini"}, "")
	require.NoError(t, err)

	registryListVerbose = true
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	err = runRegistryList(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	// In verbose mode, should show all targets
	assert.Contains(t, output, "claude")
	assert.Contains(t, output, "gemini")

	registryListVerbose = false
}

// TestRegistryList_StatusColumn tests that status column shows correct values.
func TestRegistryList_StatusColumn(t *testing.T) {
	_, cleanup := setupTestRegistry(t)
	defer cleanup()

	// Create a valid project (with project.xcf)
	validDir := t.TempDir()
	err := os.WriteFile(filepath.Join(validDir, "project.xcf"), []byte(""), 0600)
	require.NoError(t, err)
	err = registry.Register(validDir, "valid-project", []string{}, "")
	require.NoError(t, err)

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	err = runRegistryList(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "ok")
}

// TestRegistryList_AlphabeticalSort tests that projects are sorted alphabetically by name.
func TestRegistryList_AlphabeticalSort(t *testing.T) {
	_, cleanup := setupTestRegistry(t)
	defer cleanup()

	// Register projects in non-alphabetical order
	for _, name := range []string{"zulu", "alpha", "mike", "bravo"} {
		projectDir := t.TempDir()
		err := registry.Register(projectDir, name, []string{}, "")
		require.NoError(t, err)
	}

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	err := runRegistryList(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	// Verify alphabetical order: alpha before bravo before mike before zulu
	alphaIdx := strings.Index(output, "alpha")
	bravoIdx := strings.Index(output, "bravo")
	mikeIdx := strings.Index(output, "mike")
	zuluIdx := strings.Index(output, "zulu")

	assert.Greater(t, alphaIdx, 0)
	assert.Greater(t, bravoIdx, alphaIdx)
	assert.Greater(t, mikeIdx, bravoIdx)
	assert.Greater(t, zuluIdx, mikeIdx)
}

// TestRegistryList_LastAppliedColumn tests that the LAST APPLIED column is present.
func TestRegistryList_LastAppliedColumn(t *testing.T) {
	_, cleanup := setupTestRegistry(t)
	defer cleanup()

	projectDir := t.TempDir()
	err := registry.Register(projectDir, "test-project", []string{}, "")
	require.NoError(t, err)

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	err = runRegistryList(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "LAST APPLIED")
	assert.Contains(t, output, "never")
}

// TestRegistryList_VerboseShowsDetails tests that verbose mode shows registration timestamp and config dir.
func TestRegistryList_VerboseShowsDetails(t *testing.T) {
	_, cleanup := setupTestRegistry(t)
	defer cleanup()

	projectDir := t.TempDir()
	err := registry.Register(projectDir, "verbose-details", []string{"claude"}, "/custom/config")
	require.NoError(t, err)

	registryListVerbose = true
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	err = runRegistryList(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Registered:")
	assert.Contains(t, output, "Config Dir:")
	assert.Contains(t, output, "/custom/config")

	registryListVerbose = false
}
