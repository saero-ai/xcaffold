package resolver

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// dirContainsXCF returns true if the directory contains at least one *.xcf file
// at the top level (not recursively). Hidden directories are ignored.
func dirContainsXCF(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, ".xcaffold", "project.xcf")); err == nil {
		return true
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".xcf") {
			return true
		}
	}
	return false
}

// FindConfigDir walks up from start looking for the nearest directory that
// contains at least one *.xcf file. The search stops at (and includes) the
// home boundary. Returns the directory path or an error if none is found.
func FindConfigDir(start, home string) (string, error) {
	curr := start
	for {
		if dirContainsXCF(curr) {
			return curr, nil
		}
		if curr == home {
			break
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break // filesystem root
		}
		curr = parent
	}
	return "", fmt.Errorf("no *.xcf files found between %q and %q\n\nHint: run 'xcaffold init' to create one, or use --config to specify a path", start, home)
}

// FindXCFFiles returns all *.xcf file paths in the given directory (recursive),
// sorted alphabetically. Hidden directories and node_modules are skipped.
func FindXCFFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if path != dir && (strings.HasPrefix(name, ".") || name == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".xcf") && d.Name() != "registry.xcf" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan %q for xcf files: %w", dir, err)
	}
	sort.Strings(files)
	return files, nil
}
