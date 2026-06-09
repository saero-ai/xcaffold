package providers_test

import (
	"sync"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/providers"
	_ "github.com/saero-ai/xcaffold/providers/antigravity2"
)

// resetRegistry replaces the global registry with the given slice and returns
// a restore function that callers must defer. This gives each test a clean
// slate without exposing the mutex externally.
func resetRegistry(t *testing.T, initial []providers.ProviderManifest) func() {
	t.Helper()
	saved := providers.SwapRegistryForTest(initial)
	return func() { providers.SwapRegistryForTest(saved) }
}

func TestRegister_ManifestFor_RoundTrip(t *testing.T) {
	defer resetRegistry(t, nil)()

	m := providers.ProviderManifest{
		Name:       "testprovider",
		OutputDir:  ".testprovider",
		ValidNames: []string{"testprovider", "tp"},
	}
	providers.Register(m)

	got, ok := providers.ManifestFor("testprovider")
	if !ok {
		t.Fatal("ManifestFor returned not-found for registered provider")
	}
	if got.Name != m.Name {
		t.Errorf("Name: got %q, want %q", got.Name, m.Name)
	}
	if got.OutputDir != m.OutputDir {
		t.Errorf("OutputDir: got %q, want %q", got.OutputDir, m.OutputDir)
	}
}

func TestManifestFor_AliaslookUp(t *testing.T) {
	defer resetRegistry(t, nil)()

	providers.Register(providers.ProviderManifest{
		Name:       "claude",
		OutputDir:  ".claude",
		ValidNames: []string{"claude", "claude-code"},
	})

	got, ok := providers.ManifestFor("claude-code")
	if !ok {
		t.Fatal("ManifestFor did not find provider by alias")
	}
	if got.Name != "claude" {
		t.Errorf("Name: got %q, want %q", got.Name, "claude")
	}
}

func TestManifestFor_UnknownName(t *testing.T) {
	defer resetRegistry(t, nil)()

	_, ok := providers.ManifestFor("nonexistent")
	if ok {
		t.Error("ManifestFor returned found for unknown provider")
	}
}

func TestRegisteredNames_IncludesAliases(t *testing.T) {
	defer resetRegistry(t, nil)()

	providers.Register(providers.ProviderManifest{
		Name:       "gemini",
		OutputDir:  ".gemini",
		ValidNames: []string{"gemini", "gemini-cli"},
	})

	names := providers.RegisteredNames()
	index := make(map[string]bool, len(names))
	for _, n := range names {
		index[n] = true
	}
	for _, want := range []string{"gemini", "gemini-cli"} {
		if !index[want] {
			t.Errorf("RegisteredNames missing %q", want)
		}
	}
}

func TestIsRegistered(t *testing.T) {
	defer resetRegistry(t, nil)()

	providers.Register(providers.ProviderManifest{
		Name:       "cursor",
		OutputDir:  ".cursor",
		ValidNames: []string{"cursor", "csr"},
	})

	cases := []struct {
		name string
		want bool
	}{
		{"cursor", true},
		{"csr", true},
		{"unknown", false},
	}
	for _, tc := range cases {
		got := providers.IsRegistered(tc.name)
		if got != tc.want {
			t.Errorf("IsRegistered(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestManifests_ReturnsCopy(t *testing.T) {
	defer resetRegistry(t, nil)()

	providers.Register(providers.ProviderManifest{
		Name:       "copilot",
		OutputDir:  ".github",
		ValidNames: []string{"copilot"},
	})

	snap1 := providers.Manifests()
	snap2 := providers.Manifests()

	// Mutating the returned slice must not affect subsequent calls.
	if len(snap1) > 0 {
		snap1[0].Name = "mutated"
	}
	if len(snap2) > 0 && snap2[0].Name == "mutated" {
		t.Error("Manifests() returned a shared reference, not a copy")
	}
}

func TestManifests_DeepCopiesMapFields(t *testing.T) {
	defer resetRegistry(t, nil)()

	providers.Register(providers.ProviderManifest{
		Name:        "test",
		ValidNames:  []string{"test"},
		KindSupport: map[string]bool{"agent": true, "skill": true},
	})

	snap := providers.Manifests()
	snap[0].KindSupport["agent"] = false

	fresh := providers.Manifests()
	if !fresh[0].KindSupport["agent"] {
		t.Error("Manifests() returned a shared KindSupport map reference")
	}
}

func TestRegister_Concurrent(t *testing.T) {
	defer resetRegistry(t, nil)()

	const n = 20
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func(i int) {
			defer wg.Done()
			_ = i // satisfy loop variable use
			providers.Register(providers.ProviderManifest{
				Name:       "p",
				ValidNames: []string{"p"},
			})
		}(i)
	}
	wg.Wait()
	// No race detector trigger is the success criterion.
}

func TestResolveRenderer_UnknownTarget(t *testing.T) {
	defer resetRegistry(t, nil)()

	_, err := providers.ResolveRenderer("does-not-exist")
	if err == nil {
		t.Error("ResolveRenderer should return error for unknown target")
	}
}

func TestResolveRenderer_NilNewRenderer(t *testing.T) {
	defer resetRegistry(t, nil)()

	providers.Register(providers.ProviderManifest{
		Name:        "nilrenderer",
		ValidNames:  []string{"nilrenderer"},
		NewRenderer: nil,
	})

	_, err := providers.ResolveRenderer("nilrenderer")
	if err == nil {
		t.Error("ResolveRenderer should return error when NewRenderer is nil")
	}
}

func TestPrimaryNames_Sorted(t *testing.T) {
	defer resetRegistry(t, nil)()

	// Register providers out of alphabetical order
	providers.Register(providers.ProviderManifest{
		Name:       "gemini",
		OutputDir:  ".gemini",
		ValidNames: []string{"gemini"},
	})
	providers.Register(providers.ProviderManifest{
		Name:       "claude",
		OutputDir:  ".claude",
		ValidNames: []string{"claude"},
	})
	providers.Register(providers.ProviderManifest{
		Name:       "cursor",
		OutputDir:  ".cursor",
		ValidNames: []string{"cursor"},
	})

	got := providers.PrimaryNames()

	// Should return only primary names, sorted alphabetically
	want := []string{"claude", "cursor", "gemini"}
	if len(got) != len(want) {
		t.Fatalf("len(PrimaryNames()) = %d, want %d", len(got), len(want))
	}
	for i, name := range got {
		if name != want[i] {
			t.Errorf("PrimaryNames()[%d] = %q, want %q", i, name, want[i])
		}
	}
}

func TestPrimaryNames_Empty(t *testing.T) {
	defer resetRegistry(t, nil)()

	got := providers.PrimaryNames()
	if len(got) != 0 {
		t.Errorf("PrimaryNames() on empty registry = %v, want []", got)
	}
}

// stubRenderer is a minimal TargetRenderer implementation for testing.
type stubRenderer struct{}

func (s *stubRenderer) Target() string {
	return "stub"
}

func (s *stubRenderer) OutputDir() string {
	return ".stub"
}

func (s *stubRenderer) Capabilities() renderer.CapabilitySet {
	return renderer.CapabilitySet{}
}

func (s *stubRenderer) CompileAgents(map[string]ast.AgentConfig, string) (map[string]string, []renderer.FidelityNote, error) {
	return make(map[string]string), nil, nil
}

func (s *stubRenderer) CompileSkills(map[string]ast.SkillConfig, string) (map[string]string, []renderer.FidelityNote, error) {
	return make(map[string]string), nil, nil
}

func (s *stubRenderer) CompileRules(map[string]ast.RuleConfig, string) (map[string]string, []renderer.FidelityNote, error) {
	return make(map[string]string), nil, nil
}

func (s *stubRenderer) CompileWorkflows(map[string]ast.WorkflowConfig, string) (map[string]string, []renderer.FidelityNote, error) {
	return make(map[string]string), nil, nil
}

func (s *stubRenderer) CompileHooks(ast.HookConfig, string) (map[string]string, []renderer.FidelityNote, error) {
	return make(map[string]string), nil, nil
}

func (s *stubRenderer) CompileSettings(ast.SettingsConfig) (map[string]string, []renderer.FidelityNote, error) {
	return make(map[string]string), nil, nil
}

func (s *stubRenderer) CompileMCP(map[string]ast.MCPConfig) (map[string]string, []renderer.FidelityNote, error) {
	return make(map[string]string), nil, nil
}

func (s *stubRenderer) CompileProjectInstructions(*ast.XcaffoldConfig, string) (map[string]string, map[string]string, []renderer.FidelityNote, error) {
	return make(map[string]string), make(map[string]string), nil, nil
}

func (s *stubRenderer) CompileMemory(*ast.XcaffoldConfig, string, renderer.MemoryOptions) (map[string]string, []renderer.FidelityNote, error) {
	return make(map[string]string), nil, nil
}

func (s *stubRenderer) Finalize(map[string]string, map[string]string) (map[string]string, map[string]string, []renderer.FidelityNote, error) {
	return make(map[string]string), make(map[string]string), nil, nil
}

func TestCheckDeprecation_Active(t *testing.T) {
	defer resetRegistry(t, nil)()

	providers.Register(providers.ProviderManifest{
		Name:       "active-provider",
		OutputDir:  ".active",
		ValidNames: []string{"active-provider"},
		Status:     "",
	})

	warn, err := providers.CheckDeprecation("active-provider")
	if warn != "" {
		t.Errorf("CheckDeprecation on active provider returned warning: %q", warn)
	}
	if err != nil {
		t.Errorf("CheckDeprecation on active provider returned error: %v", err)
	}
}

func TestCheckDeprecation_Deprecated(t *testing.T) {
	defer resetRegistry(t, nil)()

	providers.Register(providers.ProviderManifest{
		Name:         "old-provider",
		OutputDir:    ".old",
		ValidNames:   []string{"old-provider"},
		Status:       "deprecated",
		DeprecatedBy: "new-provider",
	})

	warn, err := providers.CheckDeprecation("old-provider")
	if warn == "" {
		t.Error("CheckDeprecation on deprecated provider returned empty warning")
	}
	if err != nil {
		t.Errorf("CheckDeprecation on deprecated provider returned error: %v", err)
	}
	// Verify the warning mentions both the provider name and the replacement.
	if !contains(warn, "deprecated") {
		t.Errorf("Warning missing 'deprecated': %q", warn)
	}
	if !contains(warn, "new-provider") {
		t.Errorf("Warning missing replacement name: %q", warn)
	}
}

func TestCheckDeprecation_Sunset(t *testing.T) {
	defer resetRegistry(t, nil)()

	providers.Register(providers.ProviderManifest{
		Name:         "sunsetprovider",
		OutputDir:    ".sunset",
		ValidNames:   []string{"sunsetprovider"},
		Status:       "sunset",
		SunsetDate:   "2025-01-01",
		DeprecatedBy: "new-provider",
	})

	warn, err := providers.CheckDeprecation("sunsetprovider")
	if warn != "" {
		t.Errorf("CheckDeprecation on sunset provider returned non-empty warning: %q", warn)
	}
	if err == nil {
		t.Error("CheckDeprecation on sunset provider returned nil error")
	}
	// Verify the error mentions sunset and the date.
	if err != nil {
		if !contains(err.Error(), "sunset") {
			t.Errorf("Error missing 'sunset': %v", err)
		}
		if !contains(err.Error(), "2025-01-01") {
			t.Errorf("Error missing sunset date: %v", err)
		}
	}
}

func TestCheckDeprecation_Unknown(t *testing.T) {
	defer resetRegistry(t, nil)()

	warn, err := providers.CheckDeprecation("does-not-exist")
	if warn != "" {
		t.Errorf("CheckDeprecation on unknown provider returned warning: %q", warn)
	}
	if err != nil {
		t.Errorf("CheckDeprecation on unknown provider returned error: %v", err)
	}
}

func TestResolveRenderer_Deprecated(t *testing.T) {
	defer resetRegistry(t, nil)()

	providers.Register(providers.ProviderManifest{
		Name:         "deprecated-with-renderer",
		OutputDir:    ".deprecated",
		ValidNames:   []string{"deprecated-with-renderer"},
		Status:       "deprecated",
		DeprecatedBy: "new-provider",
		NewRenderer:  func() renderer.TargetRenderer { return &stubRenderer{} },
	})

	_, err := providers.ResolveRenderer("deprecated-with-renderer")
	if err != nil {
		t.Errorf("ResolveRenderer should not error on deprecated provider with factory: %v", err)
	}
}

func TestResolveRenderer_Sunset(t *testing.T) {
	defer resetRegistry(t, nil)()

	providers.Register(providers.ProviderManifest{
		Name:         "sunset-provider",
		OutputDir:    ".sunset",
		ValidNames:   []string{"sunset-provider"},
		Status:       "sunset",
		SunsetDate:   "2025-01-01",
		DeprecatedBy: "new-provider",
		NewRenderer:  func() renderer.TargetRenderer { return &stubRenderer{} },
	})

	_, err := providers.ResolveRenderer("sunset-provider")
	if err == nil {
		t.Error("ResolveRenderer should error on sunset provider")
	}
}

func TestResolveRenderer_Active(t *testing.T) {
	defer resetRegistry(t, nil)()

	providers.Register(providers.ProviderManifest{
		Name:        "active-with-renderer",
		OutputDir:   ".active",
		ValidNames:  []string{"active-with-renderer"},
		Status:      "",
		NewRenderer: func() renderer.TargetRenderer { return &stubRenderer{} },
	})

	r, err := providers.ResolveRenderer("active-with-renderer")
	if err != nil {
		t.Errorf("ResolveRenderer on active provider should not error: %v", err)
	}
	if r == nil {
		t.Error("ResolveRenderer on active provider should return non-nil renderer")
	}
}

func TestPrimaryNames_IncludesAntigravity2(t *testing.T) {
	// Don't reset registry — use the real one with actual provider registrations.
	// This tests that antigravity2 was registered during init().

	names := providers.PrimaryNames()
	found := false
	for _, name := range names {
		if name == "antigravity2" {
			found = true
			break
		}
	}
	if !found {
		t.Error("PrimaryNames() does not include 'antigravity2' — provider may not be registered")
	}
}

// contains is a helper that checks if a string contains a substring.
func contains(s, substr string) bool {
	for i := range len(s) - len(substr) + 1 {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
