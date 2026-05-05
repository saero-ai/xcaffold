package parser

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/providers"
)

// newParseFilter creates a map of directory names to skip during xcf scanning.
// It includes generic build/cache directories plus registered provider input directories.
func newParseFilter(dir string) map[string]bool {
	ignored := map[string]bool{
		".git":         true,
		".worktrees":   true,
		"node_modules": true,
		"vendor":       true,
		".venv":        true,
		"dist":         true,
		"build":        true,
		"coverage":     true,
	}

	// Add registered provider input directories dynamically.
	// This way, new providers automatically exclude their input dirs from xcf scanning.
	for _, providerDir := range providers.RegisteredInputDirs() {
		ignored[providerDir] = true
	}

	if data, err := os.ReadFile(filepath.Join(dir, ".gitignore")); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				clean := strings.TrimPrefix(line, "/")
				clean = strings.TrimSuffix(clean, "/")
				if !strings.ContainsAny(clean, "*?[") {
					ignored[clean] = true
				}
			}
		}
	}
	return ignored
}
