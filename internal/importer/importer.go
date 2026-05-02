package importer

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/saero-ai/xcaffold/internal/ast"
)

// ProviderImporter is the symmetric counterpart to renderer.TargetRenderer.
type ProviderImporter interface {
	Provider() string
	InputDir() string
	Classify(rel string, isDir bool) (Kind, Layout)
	Extract(rel string, data []byte, config *ast.XcaffoldConfig) error
	Import(dir string, config *ast.XcaffoldConfig) error
}

// DetectProviders returns importers for providers whose input directory exists
// under root. Order is deterministic (matches the input slice order).
func DetectProviders(root string, all []ProviderImporter) []ProviderImporter {
	var found []ProviderImporter
	for _, imp := range all {
		dir := filepath.Join(root, imp.InputDir())
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			found = append(found, imp)
		}
	}
	return found
}

var (
	registryMu sync.RWMutex
	registry   []ProviderImporter
)

// Register adds a ProviderImporter to the global registry. It is called from
// init() functions in each provider sub-package when that package is imported.
func Register(imp ProviderImporter) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = append(registry, imp)
}

// DefaultImporters returns a snapshot of all registered provider importers.
// The slice is ordered by registration order (i.e. the order in which the
// provider sub-packages were imported by the caller).
func DefaultImporters() []ProviderImporter {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]ProviderImporter, len(registry))
	copy(out, registry)
	return out
}

// ScanDir walks a provider directory and counts resources by Kind using the
// provider's Classify method. It reuses WalkProviderDir for consistent file
// discovery across all providers. The returned map only contains non-zero counts.
func ScanDir(imp ProviderImporter, dir string) map[Kind]int {
	counts := make(map[Kind]int)
	_ = WalkProviderDir(dir, func(rel string, data []byte) error {
		kind, _ := imp.Classify(rel, false)
		if kind != KindUnknown && kind != "" {
			counts[kind]++
		}
		return nil
	})
	return counts
}

// SupportedKinds returns the set of Kinds that a provider's classifier recognizes.
// It probes Classify with representative paths for each kind.
func SupportedKinds(imp ProviderImporter) map[Kind]bool {
	probes := []struct {
		path string
		kind Kind
	}{
		{"agents/test.md", KindAgent},
		{"prompts/test.md", KindAgent},
		{"skills/test/SKILL.md", KindSkill},
		{"rules/test.md", KindRule},
		{"rules/sub/test.md", KindRule},
		{"workflows/test.md", KindWorkflow},
		{"mcp.json", KindMCP},
		{"mcp_config.json", KindMCP},
		{"hooks/test.sh", KindHookScript},
		{"settings.json", KindSettings},
		{"settings.local.json", KindSettings},
		{"agent-memory/test/test.md", KindMemory},
	}

	supported := make(map[Kind]bool)
	for _, p := range probes {
		kind, _ := imp.Classify(p.path, false)
		if kind != KindUnknown && kind != "" {
			supported[p.kind] = true
		}
	}
	return supported
}
