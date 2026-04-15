package resolver

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StripFrontmatter removes YAML frontmatter (--- blocks) from markdown content.
func StripFrontmatter(content string) string {
	b := []byte(content)
	if !bytes.HasPrefix(b, []byte("---\n")) && !bytes.HasPrefix(b, []byte("---\r\n")) {
		return content
	}

	// Find the end of the frontmatter block
	endIdx := bytes.Index(b[4:], []byte("\n---"))
	if endIdx == -1 {
		return content // Malformed or missing closing '---', return as-is
	}

	// b[4:] starts after the first '---'. Add 4 (offset) + 4 (length of '\n---')
	startOfContent := 4 + endIdx + 4

	// Trim remaining whitespace (e.g. newline following the closing ---)
	result := bytes.TrimSpace(b[startOfContent:])
	return string(result)
}

// ResolveInstructions returns content from inline, file, or convention path.
// Priority: inline > filePath (resolved relative to baseDir) > conventionPath (relative to baseDir).
// File paths are resolved relative to baseDir. Frontmatter is stripped.
func ResolveInstructions(inline, filePath, conventionPath, baseDir string) (string, error) {
	if inline != "" {
		return inline, nil
	}

	var bestPath string
	if filePath != "" {
		cleaned := filepath.Clean(filePath)
		if strings.HasPrefix(cleaned, "..") {
			return "", fmt.Errorf("instructions-file must be a relative path inside the project: %q traverses above the project root", filePath)
		}
		bestPath = cleaned
		if !filepath.IsAbs(bestPath) {
			bestPath = filepath.Join(baseDir, bestPath)
		}
	} else if conventionPath != "" {
		bestPath = filepath.Join(baseDir, conventionPath)
		if _, err := os.Stat(bestPath); err != nil {
			// Convention path doesn't exist, which is fine, return empty string
			return "", nil
		}
	} else {
		return "", nil // No instructions provided anywhere, completely valid
	}

	b, err := os.ReadFile(bestPath)
	if err != nil {
		if filePath != "" {
			return "", fmt.Errorf("instructions-file %q not found: %w", filePath, err)
		}
		return "", nil // convention fallback miss is silent
	}

	content := StripFrontmatter(string(b))
	return content, nil
}
