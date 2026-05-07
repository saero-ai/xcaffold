package providers

import (
	"fmt"
	"sort"
	"sync"

	"github.com/saero-ai/xcaffold/internal/registry"
	"github.com/saero-ai/xcaffold/internal/renderer"
)

var (
	mu  sync.RWMutex
	reg []ProviderManifest
)

func init() {
	registry.GlobalScanIterator = func(userHome string, r *registry.GlobalScanResult) {
		mu.RLock()
		defer mu.RUnlock()
		for _, m := range reg {
			if m.GlobalScanner != nil {
				m.GlobalScanner(userHome, r)
			}
		}
	}
}

// Register adds m to the global provider registry. It is safe for concurrent
// use and is typically called from a provider package's init() function.
func Register(m ProviderManifest) {
	mu.Lock()
	defer mu.Unlock()
	reg = append(reg, m)
}

// Manifests returns a deep snapshot of all registered manifests. The returned
// slice is independent of the registry — mutations (including map fields like
// KindSupport) do not affect future calls.
func Manifests() []ProviderManifest {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]ProviderManifest, len(reg))
	for i, m := range reg {
		out[i] = m
		if m.KindSupport != nil {
			ks := make(map[string]bool, len(m.KindSupport))
			for k, v := range m.KindSupport {
				ks[k] = v
			}
			out[i].KindSupport = ks
		}
	}
	return out
}

// ManifestFor looks up a provider by its canonical name or any alias listed in
// ValidNames. The second return value is false when no match is found.
func ManifestFor(name string) (ProviderManifest, bool) {
	mu.RLock()
	defer mu.RUnlock()
	for _, m := range reg {
		for _, v := range m.ValidNames {
			if v == name {
				return m, true
			}
		}
	}
	return ProviderManifest{}, false
}

// RegisteredNames returns every valid name token (primary names and aliases)
// across all registered providers in registration order.
func RegisteredNames() []string {
	mu.RLock()
	defer mu.RUnlock()
	var names []string
	for _, m := range reg {
		names = append(names, m.ValidNames...)
	}
	return names
}

// PrimaryNames returns the canonical name of each registered provider,
// sorted alphabetically.
func PrimaryNames() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(reg))
	for _, m := range reg {
		names = append(names, m.Name)
	}
	sort.Strings(names)
	return names
}

// IsRegistered reports whether name matches any registered provider name or alias.
func IsRegistered(name string) bool {
	_, ok := ManifestFor(name)
	return ok
}

// ResolveRenderer returns a new TargetRenderer for the given target name or
// alias. It returns an error when the target is unknown or its NewRenderer
// factory is nil.
func ResolveRenderer(target string) (renderer.TargetRenderer, error) {
	m, ok := ManifestFor(target)
	if !ok {
		return nil, fmt.Errorf("providers: unknown target %q", target)
	}
	if m.NewRenderer == nil {
		return nil, fmt.Errorf("providers: %q has no renderer factory", m.Name)
	}
	return m.NewRenderer(), nil
}

// ResolveModelResolver returns a new ModelResolver for the given target name or
// alias. It returns nil if the target is unknown or does not support model resolution.
func ResolveModelResolver(target string) renderer.ModelResolver {
	m, ok := ManifestFor(target)
	if !ok || m.NewModelResolver == nil {
		return nil
	}
	return m.NewModelResolver()
}

// RegisteredInputDirs returns the input directory for each registered provider
// that has an importer. Returns a slice of directory names (e.g. [".claude", ".cursor"]).
// Duplicates are removed. Order matches registration order.
func RegisteredInputDirs() []string {
	mu.RLock()
	defer mu.RUnlock()
	var dirs []string
	seen := make(map[string]bool)
	for _, m := range reg {
		if m.NewImporter != nil {
			imp := m.NewImporter()
			dir := imp.InputDir()
			if !seen[dir] {
				dirs = append(dirs, dir)
				seen[dir] = true
			}
		}
	}
	return dirs
}

// RegisteredOutputDirs returns every registered default output directory
// (e.g. [".claude", ".cursor", ".agents"]). Duplicates are removed.
func RegisteredOutputDirs() []string {
	mu.RLock()
	defer mu.RUnlock()
	var dirs []string
	seen := make(map[string]bool)
	for _, m := range reg {
		if m.OutputDir != "" && !seen[m.OutputDir] {
			dirs = append(dirs, m.OutputDir)
			seen[m.OutputDir] = true
		}
	}
	return dirs
}

// RegisteredContextFiles returns every registered root context filename
// (e.g. ["CLAUDE.md", "GEMINI.md", "AGENTS.md"]). Duplicates are removed.
func RegisteredContextFiles() []string {
	mu.RLock()
	defer mu.RUnlock()
	var files []string
	seen := make(map[string]bool)
	for _, m := range reg {
		if m.RootContextFile != "" && !seen[m.RootContextFile] {
			files = append(files, m.RootContextFile)
			seen[m.RootContextFile] = true
		}
	}
	return files
}

// SwapRegistryForTest atomically replaces the global registry with next and
// returns the previous contents. This is exported only for testing — call it
// only inside test files via the _test package. Callers should defer the
// restore call returned by their test helper.
func SwapRegistryForTest(next []ProviderManifest) []ProviderManifest {
	mu.Lock()
	defer mu.Unlock()
	prev := reg
	if next == nil {
		reg = nil
	} else {
		reg = make([]ProviderManifest, len(next))
		copy(reg, next)
	}
	return prev
}
