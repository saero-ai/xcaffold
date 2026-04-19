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
