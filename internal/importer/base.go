package importer

import (
	"fmt"
	"path/filepath"

	"github.com/saero-ai/xcaffold/internal/ast"
)

// BaseImporter provides trivial getters shared by all provider importers.
// Providers embed this to satisfy Provider(), InputDir(), GetWarnings().
type BaseImporter struct {
	ProviderName string
	Dir          string
	Warnings     []string
}

func (b *BaseImporter) Provider() string      { return b.ProviderName }
func (b *BaseImporter) InputDir() string      { return b.Dir }
func (b *BaseImporter) GetWarnings() []string { return b.Warnings }

// AppendWarning adds a non-fatal warning.
func (b *BaseImporter) AppendWarning(msg string) {
	b.Warnings = append(b.Warnings, msg)
}

// RunImport walks the provider directory, classifies each file, and calls
// Extract for recognized kinds. Unclassified files are skipped. Extraction
// errors are non-fatal — they are stored in ProviderExtras and recorded as warnings.
func RunImport(imp ProviderImporter, rootDir string, config *ast.XcaffoldConfig) error {
	providerDir := filepath.Join(rootDir, imp.InputDir())
	return WalkProviderDir(providerDir, func(rel string, data []byte) error {
		kind, _ := imp.Classify(rel, false)
		if kind == KindUnknown {
			return nil
		}
		if err := imp.Extract(rel, data, config); err != nil {
			storeExtractionWarning(config, imp.Provider(), rel, data)
			if wa, ok := imp.(interface{ AppendWarning(string) }); ok {
				wa.AppendWarning(fmt.Sprintf("skipped %q: %v", rel, err))
			}
		}
		return nil
	})
}

func storeExtractionWarning(config *ast.XcaffoldConfig, provider, rel string, data []byte) {
	if config.ProviderExtras == nil {
		config.ProviderExtras = make(map[string]map[string][]byte)
	}
	if config.ProviderExtras[provider] == nil {
		config.ProviderExtras[provider] = make(map[string][]byte)
	}
	config.ProviderExtras[provider][rel] = data
}
