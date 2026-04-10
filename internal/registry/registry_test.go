package registry

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

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

	xcfHome := filepath.Join(tmp, ".xcaffold")
	if _, err := os.Stat(xcfHome); os.IsNotExist(err) {
		t.Fatal("~/.xcaffold/ was not created")
	}
	if _, err := os.Stat(filepath.Join(xcfHome, "settings.xcf")); os.IsNotExist(err) {
		t.Fatal("settings.xcf was not created")
	}
	if _, err := os.Stat(filepath.Join(xcfHome, "registry.xcf")); os.IsNotExist(err) {
		t.Fatal("registry.xcf was not created")
	}
}

func TestEnsureGlobalHome_Idempotent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	if err := EnsureGlobalHome(); err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	// Write a custom value to settings.xcf
	configPath := filepath.Join(tmp, ".xcaffold", "settings.xcf")
	_ = os.WriteFile(configPath, []byte("default_target: cursor\n"), 0600)

	if err := EnsureGlobalHome(); err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	// Verify custom value was not overwritten
	data, _ := os.ReadFile(configPath)
	if string(data) != "default_target: cursor\n" {
		t.Fatalf("settings.xcf was overwritten: got %q", string(data))
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
	_ = Unregister(testProjectName)

	projects, _ := List()
	if len(projects) != 0 {
		t.Fatalf("expected 0 projects after unregister, got %d", len(projects))
	}
}

func TestUnregister_ByPath(t *testing.T) {
	setupTestHome(t)

	projectPath := t.TempDir()
	_ = Register(projectPath, testProjectName, []string{"claude"}, ".")
	_ = Unregister(projectPath)

	projects, _ := List()
	if len(projects) != 0 {
		t.Fatalf("expected 0 projects after unregister, got %d", len(projects))
	}
}

func TestUnregister_NotFound(t *testing.T) {
	setupTestHome(t)

	if err := Unregister("nonexistent"); err != nil {
		t.Fatalf("Unregister should not error for missing project: %v", err)
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
