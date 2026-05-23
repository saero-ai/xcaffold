package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGlobalScanResult_IncludesPolicies(t *testing.T) {
	r := NewScanResult()
	r.Policies["no-secrets"] = GlobalPolicyEntry{InstructionsFile: "policy.md"}

	if len(r.Policies) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(r.Policies))
	}
	if r.Policies["no-secrets"].InstructionsFile != "policy.md" {
		t.Fatalf("unexpected instructions file")
	}
}

func TestGlobalScanResult_IncludesContexts(t *testing.T) {
	r := NewScanResult()
	r.Contexts["project-instructions"] = GlobalContextEntry{InstructionsFile: "CLAUDE.md"}

	if len(r.Contexts) != 1 {
		t.Fatalf("expected 1 context, got %d", len(r.Contexts))
	}
}

func TestMarshalGlobalXCAF_EmitsPoliciesSection(t *testing.T) {
	r := NewScanResult()
	r.Policies["no-secrets"] = GlobalPolicyEntry{InstructionsFile: "policy.md"}

	out := marshalGlobalXCAF(&r)

	if !strings.Contains(string(out), "policies:") {
		t.Fatal("output should contain policies: section")
	}
	if !strings.Contains(string(out), "no-secrets:") {
		t.Fatal("output should contain policy name")
	}
}

func TestMarshalGlobalXCAF_EmitsContextsSection(t *testing.T) {
	r := NewScanResult()
	r.Contexts["instructions"] = GlobalContextEntry{InstructionsFile: "CLAUDE.md"}

	out := marshalGlobalXCAF(&r)

	if !strings.Contains(string(out), "contexts:") {
		t.Fatal("output should contain contexts: section")
	}
}

const testProjectName = "my-app"

// setupTestHome redirects $HOME to a temp dir and ensures the global home exists.
func setupTestHome(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	if err := EnsureGlobalHome(); err != nil {
		t.Fatalf("EnsureGlobalHome failed: %v", err)
	}
	return tmp
}

func TestEnsureGlobalHome_CreatesDirectory(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	if err := EnsureGlobalHome(); err != nil {
		t.Fatalf("EnsureGlobalHome failed: %v", err)
	}

	xcafHome := filepath.Join(tmp, ".xcaffold")
	if _, err := os.Stat(xcafHome); os.IsNotExist(err) {
		t.Fatal("~/.xcaffold/ was not created")
	}

	// settings.xcaf must NOT be created (removed in file taxonomy update)
	if _, err := os.Stat(filepath.Join(xcafHome, "settings.xcaf")); err == nil {
		t.Fatal("settings.xcaf should not be created")
	}

	if _, err := os.Stat(filepath.Join(xcafHome, "registry.xcaf")); os.IsNotExist(err) {
		t.Fatal("registry.xcaf was not created")
	}

	// global.xcaf bootstrap is deferred; must NOT be created.
	if _, err := os.Stat(filepath.Join(xcafHome, "global.xcaf")); err == nil {
		t.Fatal("global.xcaf should not be created (bootstrap deferred)")
	}

	registryData, err := os.ReadFile(filepath.Join(xcafHome, "registry.xcaf"))
	if err != nil {
		t.Fatalf("registry.xcaf was not created: %v", err)
	}
	if !strings.Contains(string(registryData), "kind: registry") {
		t.Errorf("registry.xcaf should contain 'kind: registry', got:\n%s", string(registryData))
	}
}

func TestEnsureGlobalHome_Idempotent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	if err := EnsureGlobalHome(); err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	// Write a project entry to registry.xcaf and verify EnsureGlobalHome does not overwrite it.
	home := filepath.Join(tmp, ".xcaffold")
	registryPath := filepath.Join(home, "registry.xcaf")
	customRegistry := "kind: registry\nprojects:\n  - path: /some/project\n    name: sentinel\n"
	_ = os.WriteFile(registryPath, []byte(customRegistry), 0600)

	if err := EnsureGlobalHome(); err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	// Verify custom registry was not overwritten
	data, _ := os.ReadFile(registryPath)
	if string(data) != customRegistry {
		t.Fatalf("registry.xcaf was overwritten: got %q", string(data))
	}
}

func TestGlobalXCAF_KindGlobal(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Create a fake .claude/agents/ directory with an agent so scanner finds something.
	agentsDir := filepath.Join(tmp, ".claude", "agents")
	_ = os.MkdirAll(agentsDir, 0755)
	_ = os.WriteFile(filepath.Join(agentsDir, "test-agent.md"), []byte("# Test Agent"), 0600)

	data := buildGlobalXCAF()
	content := string(data)

	if !strings.Contains(content, "kind: global") {
		t.Errorf("global.xcaf should contain 'kind: global', got:\n%s", content)
	}
	if strings.Contains(content, "project:") {
		t.Errorf("global.xcaf should NOT contain 'project:' block, got:\n%s", content)
	}
}

func TestRegistryXCAF_KindRegistry(t *testing.T) {
	setupTestHome(t)

	projectPath := t.TempDir()
	if err := Register(projectPath, "test-proj", []string{"claude"}, "."); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	home, _ := GlobalHome()
	data, err := os.ReadFile(filepath.Join(home, "registry.xcaf"))
	if err != nil {
		t.Fatalf("could not read registry.xcaf: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "kind: registry") {
		t.Errorf("registry.xcaf should contain 'kind: registry', got:\n%s", content)
	}
}

func TestRegister_NewProject(t *testing.T) {
	setupTestHome(t)

	projectPath := t.TempDir()
	if err := Register(projectPath, testProjectName, []string{"claude"}, "."); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	projects, err := List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
	if projects[0].Name != testProjectName {
		t.Errorf("expected name 'my-app', got %q", projects[0].Name)
	}
	if projects[0].Targets[0] != "claude" {
		t.Errorf("expected target 'claude', got %q", projects[0].Targets[0])
	}
	if projects[0].Registered.IsZero() {
		t.Error("Registered timestamp should not be zero")
	}
}

func TestRegister_UpdateExisting(t *testing.T) {
	setupTestHome(t)

	projectPath := t.TempDir()
	_ = Register(projectPath, testProjectName, []string{"claude"}, ".")
	_ = Register(projectPath, "my-app-renamed", []string{"claude", "cursor"}, ".")

	projects, _ := List()
	if len(projects) != 1 {
		t.Fatalf("expected 1 project after update, got %d", len(projects))
	}
	if projects[0].Name != "my-app-renamed" {
		t.Errorf("expected updated name 'my-app-renamed', got %q", projects[0].Name)
	}
	if len(projects[0].Targets) != 2 {
		t.Errorf("expected 2 targets, got %d", len(projects[0].Targets))
	}
}

func TestRegister_NameCollision(t *testing.T) {
	setupTestHome(t)

	pathA := filepath.Join(t.TempDir(), "org-a", "api")
	pathB := filepath.Join(t.TempDir(), "org-b", "api")
	_ = os.MkdirAll(pathA, 0755)
	_ = os.MkdirAll(pathB, 0755)

	_ = Register(pathA, "api", []string{"claude"}, ".")
	_ = Register(pathB, "api", []string{"claude"}, ".")

	projects, _ := List()
	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}

	names := map[string]bool{}
	for _, p := range projects {
		names[p.Name] = true
	}
	if len(names) != 2 {
		t.Errorf("expected 2 unique names, got %d: %v", len(names), names)
	}
}

func TestUnregister_ByName(t *testing.T) {
	setupTestHome(t)

	projectPath := t.TempDir()
	_ = Register(projectPath, testProjectName, []string{"claude"}, ".")
	_, _ = Unregister(testProjectName)

	projects, _ := List()
	if len(projects) != 0 {
		t.Fatalf("expected 0 projects after unregister, got %d", len(projects))
	}
}

func TestUnregister_ByPath(t *testing.T) {
	setupTestHome(t)

	projectPath := t.TempDir()
	_ = Register(projectPath, testProjectName, []string{"claude"}, ".")
	_, _ = Unregister(projectPath)

	projects, _ := List()
	if len(projects) != 0 {
		t.Fatalf("expected 0 projects after unregister, got %d", len(projects))
	}
}

func TestUnregister_NotFound_OldTest(t *testing.T) {
	setupTestHome(t)

	// Old test — still registers projects, but now expects an error for missing project
	_, err := Unregister("nonexistent")
	if err == nil {
		t.Fatal("Unregister should error for missing project")
	}
	if !strings.Contains(err.Error(), "no project found") {
		t.Errorf("error should contain 'no project found', got: %v", err)
	}
}

func TestUnregister_ReturnsRemovedProject(t *testing.T) {
	setupTestHome(t)

	projectPath := t.TempDir()
	if err := Register(projectPath, testProjectName, []string{"claude"}, "."); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	removed, err := Unregister(testProjectName)
	if err != nil {
		t.Fatalf("Unregister should not error for found project: %v", err)
	}

	if removed.Name != testProjectName {
		t.Errorf("expected removed project name %q, got %q", testProjectName, removed.Name)
	}

	projects, _ := List()
	if len(projects) != 0 {
		t.Fatalf("expected 0 projects after unregister, got %d", len(projects))
	}
}

func TestUnregister_NotFound_ReturnsError(t *testing.T) {
	setupTestHome(t)

	_, err := Unregister("ghost")
	if err == nil {
		t.Fatal("Unregister should return error when project not found")
	}
	if !strings.Contains(err.Error(), "no project found") {
		t.Errorf("error should contain 'no project found', got: %v", err)
	}
}

func TestList_Empty(t *testing.T) {
	setupTestHome(t)

	projects, err := List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(projects) != 0 {
		t.Fatalf("expected 0 projects, got %d", len(projects))
	}
}

func TestList_MultipleProjects(t *testing.T) {
	setupTestHome(t)

	_ = Register(t.TempDir(), "app-1", []string{"claude"}, ".")
	_ = Register(t.TempDir(), "app-2", []string{"cursor"}, ".")
	_ = Register(t.TempDir(), "app-3", []string{"claude", "cursor"}, ".")

	projects, _ := List()
	if len(projects) != 3 {
		t.Fatalf("expected 3 projects, got %d", len(projects))
	}
}

func TestResolve_ByName(t *testing.T) {
	setupTestHome(t)

	projectPath := t.TempDir()
	_ = Register(projectPath, testProjectName, []string{"claude"}, ".")

	p, err := Resolve(testProjectName)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if p.Name != testProjectName {
		t.Errorf("expected name 'my-app', got %q", p.Name)
	}
}

func TestResolve_ByPath(t *testing.T) {
	setupTestHome(t)

	projectPath := t.TempDir()
	_ = Register(projectPath, testProjectName, []string{"claude"}, ".")

	p, err := Resolve(projectPath)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if p.Name != testProjectName {
		t.Errorf("expected name 'my-app', got %q", p.Name)
	}
}

func TestResolve_NotFound(t *testing.T) {
	setupTestHome(t)

	_, err := Resolve("nonexistent")
	if err == nil {
		t.Fatal("Resolve should return error for unknown project")
	}
}

func TestUpdateLastApplied_SetsTimestamp(t *testing.T) {
	setupTestHome(t)

	projectPath := t.TempDir()
	_ = Register(projectPath, testProjectName, []string{"claude"}, ".")

	before := time.Now().UTC()
	_ = UpdateLastApplied(projectPath)

	p, _ := Resolve(testProjectName)
	if p.LastApplied.IsZero() {
		t.Fatal("LastApplied should not be zero after update")
	}
	if p.LastApplied.Before(before) {
		t.Error("LastApplied should be >= the time before the call")
	}
}

func TestUpdateLastApplied_NoMatch(t *testing.T) {
	setupTestHome(t)

	if err := UpdateLastApplied("/nonexistent/path"); err != nil {
		t.Fatalf("UpdateLastApplied should not error for missing project: %v", err)
	}
}

func TestRegister_StoresConfigDir(t *testing.T) {
	setupTestHome(t)

	projectPath := t.TempDir()
	if err := Register(projectPath, "with-configdir", []string{"claude"}, "xcaffold"); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	proj, err := Resolve("with-configdir")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if proj.ConfigDir != "xcaffold" {
		t.Fatalf("expected ConfigDir %q, got %q", "xcaffold", proj.ConfigDir)
	}
}

func TestRegister_DefaultConfigDir(t *testing.T) {
	setupTestHome(t)

	projectPath := t.TempDir()
	if err := Register(projectPath, "default-configdir", []string{"claude"}, "."); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	proj, err := Resolve("default-configdir")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if proj.ConfigDir != "." {
		t.Fatalf("expected ConfigDir %q, got %q", ".", proj.ConfigDir)
	}
}

func TestGlobalHome_XcaffoldHomeOverride(t *testing.T) {
	customDir := t.TempDir()
	t.Setenv("XCAFFOLD_HOME", customDir)
	home, err := GlobalHome()
	if err != nil {
		t.Fatalf("GlobalHome returned error: %v", err)
	}
	if home != customDir {
		t.Errorf("expected GlobalHome() == %q, got %q", customDir, home)
	}
}

func TestPathExists_ValidPath(t *testing.T) {
	setupTestHome(t)
	projectPath := t.TempDir()
	p := Project{Path: projectPath}
	if !PathExists(p) {
		t.Errorf("PathExists should return true for existing directory %q", projectPath)
	}
}

func TestPathExists_NonexistentPath(t *testing.T) {
	setupTestHome(t)
	p := Project{Path: "/nonexistent/path/that/does/not/exist"}
	if PathExists(p) {
		t.Error("PathExists should return false for nonexistent path")
	}
}

func TestPrune_RemovesNonExistentPaths(t *testing.T) {
	setupTestHome(t)

	// Register 3 projects
	path1 := t.TempDir()
	path2 := t.TempDir()
	path3 := t.TempDir()

	_ = Register(path1, "proj-1", []string{"claude"}, ".")
	_ = Register(path2, "proj-2", []string{"claude"}, ".")
	_ = Register(path3, "proj-3", []string{"claude"}, ".")

	// Delete one directory
	_ = os.RemoveAll(path2)

	// Prune with dry-run=false
	pruned, err := Prune(false)
	if err != nil {
		t.Fatalf("Prune failed: %v", err)
	}

	// Should return 1 entry (the deleted one)
	if len(pruned) != 1 {
		t.Errorf("expected 1 pruned entry, got %d", len(pruned))
	}
	if pruned[0].Name != "proj-2" {
		t.Errorf("expected pruned entry to be proj-2, got %q", pruned[0].Name)
	}

	// List should now show only 2 projects
	remaining, _ := List()
	if len(remaining) != 2 {
		t.Errorf("expected 2 remaining projects after prune, got %d", len(remaining))
	}
}

func TestPrune_PreservesValidPaths(t *testing.T) {
	setupTestHome(t)

	// Register 3 projects with all dirs existing
	path1 := t.TempDir()
	path2 := t.TempDir()
	path3 := t.TempDir()

	_ = Register(path1, "proj-1", []string{"claude"}, ".")
	_ = Register(path2, "proj-2", []string{"claude"}, ".")
	_ = Register(path3, "proj-3", []string{"claude"}, ".")

	// Prune with all paths existing
	pruned, err := Prune(false)
	if err != nil {
		t.Fatalf("Prune failed: %v", err)
	}

	// Should return 0 entries (nothing to prune)
	if len(pruned) != 0 {
		t.Errorf("expected 0 pruned entries, got %d", len(pruned))
	}

	// List should still show 3 projects
	remaining, _ := List()
	if len(remaining) != 3 {
		t.Errorf("expected 3 projects after prune, got %d", len(remaining))
	}
}

func TestPrune_EmptyRegistry(t *testing.T) {
	setupTestHome(t)

	// Registry is empty by default after setup
	pruned, err := Prune(false)
	if err != nil {
		t.Fatalf("Prune failed: %v", err)
	}

	// Should return 0 entries
	if len(pruned) != 0 {
		t.Errorf("expected 0 pruned entries from empty registry, got %d", len(pruned))
	}
}

func TestPrune_DryRun(t *testing.T) {
	setupTestHome(t)

	// Register 2 projects
	path1 := t.TempDir()
	path2 := t.TempDir()

	_ = Register(path1, "proj-1", []string{"claude"}, ".")
	_ = Register(path2, "proj-2", []string{"claude"}, ".")

	// Delete one directory
	_ = os.RemoveAll(path2)

	// Prune with dry-run=true
	pruned, err := Prune(true)
	if err != nil {
		t.Fatalf("Prune failed: %v", err)
	}

	// Should return 1 entry (the deleted one)
	if len(pruned) != 1 {
		t.Errorf("expected 1 pruned entry, got %d", len(pruned))
	}

	// List should still show 2 projects (registry unchanged)
	remaining, _ := List()
	if len(remaining) != 2 {
		t.Errorf("expected 2 projects after dry-run prune, got %d (should be unchanged)", len(remaining))
	}
}

func TestWriteProjects_AtomicWrite(t *testing.T) {
	setupTestHome(t)

	// Register a project
	projectPath := t.TempDir()
	_ = Register(projectPath, "test-proj", []string{"claude"}, ".")

	// Verify that no .registry.xcaf.tmp file is left behind
	home, _ := GlobalHome()
	tmpPath := filepath.Join(home, ".registry.xcaf.tmp")

	// After Register (which calls writeProjects), the tmp file should not exist
	if _, err := os.Stat(tmpPath); err == nil {
		t.Error("temporary .registry.xcaf.tmp file should not exist after successful write")
	}

	// Verify the actual registry file exists and is valid
	projects, err := List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(projects) != 1 {
		t.Errorf("expected 1 project in registry, got %d", len(projects))
	}
}

func TestInfo_ExistingProject(t *testing.T) {
	setupTestHome(t)

	projectPath := t.TempDir()
	// Create xcaf/ subdir and project.xcaf file
	_ = os.Mkdir(filepath.Join(projectPath, "xcaf"), 0755)
	_ = os.WriteFile(filepath.Join(projectPath, "project.xcaf"), []byte("kind: project"), 0600)

	_ = Register(projectPath, "existing-proj", []string{"claude"}, ".")

	info, err := Info("existing-proj")
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}

	if !info.Exists {
		t.Error("Exists should be true for registered project")
	}
	if !info.HasXcafDir {
		t.Error("HasXcafDir should be true when xcaf/ exists")
	}
	if !info.HasProjectXcf {
		t.Error("HasProjectXcf should be true when project.xcaf exists")
	}
	if info.Name != "existing-proj" {
		t.Errorf("Name should be 'existing-proj', got %q", info.Name)
	}
	if info.Path != projectPath {
		t.Errorf("Path should be %q, got %q", projectPath, info.Path)
	}
}

func TestInfo_StaleProject(t *testing.T) {
	setupTestHome(t)

	projectPath := t.TempDir()
	_ = Register(projectPath, "stale-proj", []string{"claude"}, ".")

	// Delete the project directory
	_ = os.RemoveAll(projectPath)

	info, err := Info("stale-proj")
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}

	if info.Exists {
		t.Error("Exists should be false for deleted project")
	}
	if info.HasXcafDir {
		t.Error("HasXcafDir should be false when path doesn't exist")
	}
	if info.HasProjectXcf {
		t.Error("HasProjectXcf should be false when path doesn't exist")
	}
}

func TestInfo_NotFound(t *testing.T) {
	setupTestHome(t)

	_, err := Info("nonexistent")
	if err == nil {
		t.Fatal("Info should return error for unknown project")
	}
	if !strings.Contains(err.Error(), "no project found") {
		t.Errorf("error should contain 'no project found', got: %v", err)
	}
}

func TestInfo_ByPath(t *testing.T) {
	setupTestHome(t)

	projectPath := t.TempDir()
	_ = os.Mkdir(filepath.Join(projectPath, "xcaf"), 0755)
	_ = os.WriteFile(filepath.Join(projectPath, "project.xcaf"), []byte("kind: project"), 0600)

	_ = Register(projectPath, "path-lookup", []string{"claude"}, ".")

	// Resolve by absolute path instead of name
	info, err := Info(projectPath)
	if err != nil {
		t.Fatalf("Info by path failed: %v", err)
	}

	if !info.Exists {
		t.Error("Exists should be true")
	}
	if info.Name != "path-lookup" {
		t.Errorf("Name should be 'path-lookup', got %q", info.Name)
	}
}
