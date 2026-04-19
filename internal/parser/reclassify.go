package parser

import (
	"fmt"
	"sort"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/importer"
)

// ReclassifyExtras iterates over config.ProviderExtras and attempts to graduate
// each stored file using the corresponding provider's Classify/Extract methods.
//
// Files that are now recognized by the importer are extracted into the typed AST
// and removed from ProviderExtras. Files that remain unknown (KindUnknown) stay
// in ProviderExtras unchanged. When all files for a provider are graduated, the
// provider key is deleted from ProviderExtras entirely.
//
// Map keys are iterated in sorted order for determinism.
//
// The blank imports for provider registration (e.g.
//
//	_ "github.com/saero-ai/xcaffold/internal/importer/claude"
//
// ) must be done by the caller, not here.
func ReclassifyExtras(config *ast.XcaffoldConfig, importers []importer.ProviderImporter) error {
	if len(config.ProviderExtras) == 0 {
		return nil
	}

	byProvider := make(map[string]importer.ProviderImporter, len(importers))
	for _, imp := range importers {
		byProvider[imp.Provider()] = imp
	}

	for provider, files := range config.ProviderExtras {
		imp, ok := byProvider[provider]
		if !ok {
			// No importer registered for this provider — leave as-is.
			continue
		}

		remaining := make(map[string][]byte)
		for _, relPath := range sortedKeys(files) {
			data := files[relPath]
			kind, _ := imp.Classify(relPath, false)
			if kind == importer.KindUnknown {
				remaining[relPath] = data
				continue
			}
			if err := imp.Extract(relPath, data, config); err != nil {
				return fmt.Errorf("reclassify %s/%s: %w", provider, relPath, err)
			}
		}

		if len(remaining) == 0 {
			delete(config.ProviderExtras, provider)
		} else {
			config.ProviderExtras[provider] = remaining
		}
	}

	return nil
}

// sortedKeys returns the keys of m in sorted order.
func sortedKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
