package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// claudeProjectMemoryDir returns the Claude project memory directory for a
// given project root: ~/.claude/projects/<encoded-projectRoot>/memory/.
//
// Both xcaffold import --with-memory and xcaffold apply --include-memory use
// this function so they always resolve the same directory for the same project.
//
// projectRoot must be the absolute directory of the project (e.g. the directory
// containing project.xcf, or the registered project path for --project flag
// usage). If projectRoot is empty or ".", os.Getwd() is used as a fallback.
//
// Path encoding follows Claude's own convention: forward slashes are replaced
// with hyphens.
func claudeProjectMemoryDir(projectRoot string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	if projectRoot == "" || projectRoot == "." {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolving working directory: %w", err)
		}
		projectRoot = cwd
	}
	projectRoot = filepath.Clean(projectRoot)
	if !filepath.IsAbs(projectRoot) {
		abs, err := filepath.Abs(projectRoot)
		if err != nil {
			return "", fmt.Errorf("resolving absolute project root: %w", err)
		}
		projectRoot = abs
	}
	encoded := strings.ReplaceAll(projectRoot, "/", "-")
	return filepath.Join(home, ".claude", "projects", encoded, "memory"), nil
}
