package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseFrontmatter splits YAML frontmatter from a markdown file's body.
// If the data does not start with "---\n", the entire content is returned as
// the body with zero-value metadata. Malformed YAML in the frontmatter block
// returns an error — callers that need graceful fallback should use
// ParseFrontmatterLenient.
func ParseFrontmatter(data []byte, v interface{}) (body string, err error) {
	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		return strings.TrimSpace(content), nil
	}
	parts := strings.SplitN("\n"+content[4:], "\n---", 2)
	if len(parts) < 2 {
		return strings.TrimSpace(content), nil
	}
	if err := yaml.Unmarshal([]byte(parts[0]), v); err != nil {
		return "", fmt.Errorf("frontmatter: %w", err)
	}
	return strings.TrimSpace(strings.TrimPrefix(parts[1], "\n")), nil
}

// ParseFrontmatterLenient is like ParseFrontmatter but returns the body with
// zero-value metadata when the YAML is malformed, instead of an error. Use
// for user-edited files where frontmatter may contain unquoted special chars.
func ParseFrontmatterLenient(data []byte, v interface{}) (body string, err error) {
	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		return strings.TrimSpace(content), nil
	}
	parts := strings.SplitN("\n"+content[4:], "\n---", 2)
	if len(parts) < 2 {
		return strings.TrimSpace(content), nil
	}
	if err := yaml.Unmarshal([]byte(parts[0]), v); err != nil {
		return strings.TrimSpace(strings.TrimPrefix(parts[1], "\n")), nil
	}
	return strings.TrimSpace(strings.TrimPrefix(parts[1], "\n")), nil
}

// MatchGlob matches a relative path against a glob pattern.
// Supports "*" (any single segment) and "**" (any number of segments).
func MatchGlob(pattern, rel string) bool {
	patParts := strings.Split(pattern, "/")
	relParts := strings.Split(rel, "/")
	return matchSegments(patParts, relParts)
}

func matchSegments(pat, rel []string) bool {
	for len(pat) > 0 && len(rel) > 0 {
		switch pat[0] {
		case "**":
			for i := 0; i <= len(rel); i++ {
				if matchSegments(pat[1:], rel[i:]) {
					return true
				}
			}
			return false
		default:
			ok, err := filepath.Match(pat[0], rel[0])
			if err != nil || !ok {
				return false
			}
			pat, rel = pat[1:], rel[1:]
		}
	}
	return len(pat) == 0 && len(rel) == 0
}

// ReadFile reads a file from disk. Thin wrapper over os.ReadFile provided
// so that provider importers use a single import path for file I/O.
func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// AppendUnique appends s to slice only if it is not already present.
func AppendUnique(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}
