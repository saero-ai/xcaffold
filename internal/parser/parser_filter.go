package parser

import (
	"os"
	"path/filepath"
	"strings"
)

// newParseFilter creates a map of directory names to skip during xcf scanning.
func newParseFilter(dir string) map[string]bool {
	ignored := map[string]bool{
		".git":         true,
		".worktrees":   true,
		"node_modules": true,
		"vendor":       true,
		".venv":        true,
		".xcaffold":    true,
		".claude":      true,
		".cursor":      true,
		".gemini":      true,
		".agents":      true,
		"dist":         true,
		"build":        true,
		"coverage":     true,
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
