package parser

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
)

// loadExtras reads raw provider-extra files from <dir>/xcf/extras/ into
// config.ProviderExtras. The on-disk layout is:
//
//	<dir>/xcf/extras/<provider>/<relpath>
//
// provider is the first subdirectory under extras/; relpath is everything
// after that, always normalised to forward slashes.
//
// If the extras directory does not exist the function returns nil — the
// absence of extras is not an error.
func loadExtras(dir string, config *ast.XcaffoldConfig) error {
	extrasDir := filepath.Join(dir, "xcf", "extras")

	info, err := os.Stat(extrasDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	return filepath.WalkDir(extrasDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		// path format: <extrasDir>/<provider>/<relpath>
		rel, relErr := filepath.Rel(extrasDir, path)
		if relErr != nil {
			return relErr
		}

		parts := strings.SplitN(filepath.ToSlash(rel), "/", 2)
		if len(parts) < 2 {
			// File directly under xcf/extras/ with no provider subdirectory — skip.
			return nil
		}

		provider := parts[0]
		relPath := parts[1]

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read extras file %q: %w", path, readErr)
		}

		if config.ProviderExtras == nil {
			config.ProviderExtras = make(map[string]map[string][]byte)
		}
		if config.ProviderExtras[provider] == nil {
			config.ProviderExtras[provider] = make(map[string][]byte)
		}
		config.ProviderExtras[provider][relPath] = data
		return nil
	})
}
