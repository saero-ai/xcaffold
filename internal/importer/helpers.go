package importer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
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

// FileVisitor is called for each regular file found during a provider directory walk.
// rel is the slash-separated path relative to the walk root. data is the file contents.
type FileVisitor func(rel string, data []byte) error

// WalkProviderDir walks dir recursively, calling visitor for each regular file.
// Symlinks to directories are followed with cycle detection. Directories are
// skipped. rel paths use forward slashes.
func WalkProviderDir(dir string, visitor FileVisitor) error {
	visited := make(map[string]bool)
	return walkProviderDir(dir, dir, "", visitor, visited)
}

// walkProviderDir walks current, computing relative paths from root.
// prefix is prepended when walking a symlinked directory that resolves
// outside root — it maps files back to the symlink's position in the
// original tree.
func walkProviderDir(root, current, prefix string, visitor FileVisitor, visited map[string]bool) error {
	return filepath.WalkDir(current, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			target, err := filepath.EvalSymlinks(path)
			if err != nil {
				return nil
			}
			info, err := os.Stat(target)
			if err != nil {
				return nil
			}
			if info.IsDir() {
				if visited[target] {
					return nil
				}
				visited[target] = true
				symlinkRel, relErr := filepath.Rel(root, path)
				if relErr != nil {
					symlinkRel = filepath.Base(path)
				}
				if prefix != "" {
					symlinkRel = filepath.Join(prefix, symlinkRel)
				}
				return walkProviderDir(target, target, symlinkRel, visitor, visited)
			}
			path = target
		}
		var rel string
		if prefix != "" {
			fileRel, relErr := filepath.Rel(current, path)
			if relErr != nil {
				return fmt.Errorf("rel path: %w", relErr)
			}
			rel = filepath.Join(prefix, fileRel)
		} else {
			var relErr error
			rel, relErr = filepath.Rel(root, path)
			if relErr != nil {
				return fmt.Errorf("rel path: %w", relErr)
			}
		}
		rel = filepath.ToSlash(rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil // non-fatal: skip unreadable files
		}
		return visitor(rel, data)
	})
}

// ExtractHookScript stores a hook shell script for passthrough to xcf/hooks/.
// Scripts are stored in config.ProviderExtras["xcf"]["hooks/<basename>"]
// and emitted to xcf/hooks/<basename> by WriteSplitFiles.
func ExtractHookScript(rel string, data []byte, config *ast.XcaffoldConfig) error {
	dest := "hooks/" + filepath.Base(rel)
	if config.ProviderExtras == nil {
		config.ProviderExtras = make(map[string]map[string][]byte)
	}
	if config.ProviderExtras["xcf"] == nil {
		config.ProviderExtras["xcf"] = make(map[string][]byte)
	}
	config.ProviderExtras["xcf"][dest] = data
	return nil
}
