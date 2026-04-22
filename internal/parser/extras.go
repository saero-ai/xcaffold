package parser

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
)

// loadExtras reads raw provider-extra files into config.ProviderExtras.
// It checks two directories in priority order:
//
//  1. <dir>/xcf/provider/<provider>/<relpath>  (new convention, higher priority)
//  2. <dir>/xcf/extras/<provider>/<relpath>    (legacy convention, lower priority)
//
// Files found in xcf/provider/ take precedence: if the same <provider>/<relpath>
// exists in both directories, the xcf/provider/ version is kept and the
// xcf/extras/ version is silently ignored.
//
// If neither directory exists the function returns nil — absence of extras is
// not an error.
func loadExtras(dir string, config *ast.XcaffoldConfig) error {
	if err := walkExtrasDir(filepath.Join(dir, "xcf", "provider"), config, false); err != nil {
		return err
	}
	return walkExtrasDir(filepath.Join(dir, "xcf", "extras"), config, true)
}

// walkExtrasDir walks a single extras-style directory and populates
// config.ProviderExtras.  When skipExisting is true, entries whose key already
// exists in the map are not overwritten (used for the legacy xcf/extras/ pass).
func walkExtrasDir(extrasDir string, config *ast.XcaffoldConfig, skipExisting bool) error {
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
			// File directly under the extras dir with no provider subdirectory — skip.
			return nil
		}

		provider := parts[0]
		relPath := parts[1]

		if config.ProviderExtras == nil {
			config.ProviderExtras = make(map[string]map[string][]byte)
		}
		if config.ProviderExtras[provider] == nil {
			config.ProviderExtras[provider] = make(map[string][]byte)
		}

		if skipExisting {
			if _, exists := config.ProviderExtras[provider][relPath]; exists {
				return nil
			}
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read extras file %q: %w", path, readErr)
		}

		config.ProviderExtras[provider][relPath] = data
		return nil
	})
}
