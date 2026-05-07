package parser

import (
	"errors"
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
//  1. <dir>/xcaf/provider/<provider>/<relpath>  (new convention, higher priority)
//  2. <dir>/xcaf/extras/<provider>/<relpath>    (legacy convention, lower priority)
//
// Files found in xcaf/provider/ take precedence: if the same <provider>/<relpath>
// exists in both directories, the xcaf/provider/ version is kept and the
// xcaf/extras/ version is silently ignored.
//
// If neither directory exists the function returns nil — absence of extras is
// not an error.
func loadExtras(dir string, config *ast.XcaffoldConfig) error {
	// Load hook scripts as provider-agnostic extras mapped to hooks/.
	if err := walkExtrasDirRoot(filepath.Join(dir, "xcaf", "hooks"), "xcaf", config); err != nil {
		return err
	}

	if err := walkExtrasDir(filepath.Join(dir, "xcaf", "provider"), config, false); err != nil {
		return err
	}
	return walkExtrasDir(filepath.Join(dir, "xcaf", "extras"), config, true)
}

func walkExtrasDirRoot(extrasDir string, fixedProvider string, config *ast.XcaffoldConfig) error {
	info, err := os.Stat(extrasDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat dir %q: %w", extrasDir, err)
	}
	if !info.IsDir() {
		return nil
	}

	return filepath.WalkDir(extrasDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, relErr := filepath.Rel(filepath.Dir(extrasDir), path)
		if relErr != nil {
			return relErr
		}

		if config.ProviderExtras == nil {
			config.ProviderExtras = make(map[string]map[string][]byte)
		}
		if config.ProviderExtras[fixedProvider] == nil {
			config.ProviderExtras[fixedProvider] = make(map[string][]byte)
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read file %q: %w", path, readErr)
		}

		config.ProviderExtras[fixedProvider][filepath.ToSlash(rel)] = data
		return nil
	})
}

// walkExtrasDir walks a single extras-style directory and populates
// config.ProviderExtras.  When skipExisting is true, entries whose key already
// exists in the map are not overwritten (used for the legacy xcaf/extras/ pass).
func walkExtrasDir(extrasDir string, config *ast.XcaffoldConfig, skipExisting bool) error {
	info, err := os.Stat(extrasDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat extras dir %q: %w", extrasDir, err)
	}
	if !info.IsDir() {
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
