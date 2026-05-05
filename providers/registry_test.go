package providers_test

import (
	"sync"
	"testing"

	"github.com/saero-ai/xcaffold/providers"
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
