package importer

import (
	"os"
	"path/filepath"

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

// DefaultImporters returns all built-in provider importers.
// Sub-packages are registered here as they are implemented.
func DefaultImporters() []ProviderImporter {
	return []ProviderImporter{}
}
