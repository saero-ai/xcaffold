package renderer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/output"
)

// SortedKeys returns a sorted slice of keys from a map with string-convertible keys.
// Using this function over ranging directly on a map ensures deterministic output.
func SortedKeys[K ~string, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

// YAMLScalar quotes a string value for safe inclusion in YAML if it contains
// characters that would otherwise need quoting. For simple values it returns
// the string as-is.
func YAMLScalar(s string) string {
	if strings.ContainsAny(s, ":#{}[]|>&*!,'\"\\%@`") {
		return fmt.Sprintf("%q", s)
	}
	return s
}

// StripAllFrontmatter removes all leading YAML frontmatter blocks from content.
// Unlike a single-pass strip, this loops until no further --- delimited blocks
// remain at the top of the string, which handles files that accumulate multiple
// frontmatter sections during template expansion.
func StripAllFrontmatter(content string) string {
	for {
		trimmed := strings.TrimSpace(content)
		if !strings.HasPrefix(trimmed, "---") {
			return content
		}
		after := trimmed[3:]
		idx := strings.Index(after, "\n---")
		if idx < 0 {
			return content
		}
		content = strings.TrimSpace(after[idx+4:])
	}
}

// FlattenToSkillRoot reads files matching paths (globs or literals) relative to
// baseDir and writes them directly to skills/<id>/<filename> in out.Files — no
// subdirectory is created. This is used by providers that co-locate all skill
// files alongside SKILL.md (e.g., Claude examples, Copilot all subdirs).
func FlattenToSkillRoot(id, canonicalName string, paths []string, baseDir string, out *output.Output) error {
	if len(paths) == 0 {
		return nil
	}
	for _, pattern := range paths {
		cleanedPattern := filepath.Clean(pattern)
		if strings.HasPrefix(cleanedPattern, "..") {
			return fmt.Errorf("%s path %q traverses above the project root", canonicalName, pattern)
		}
		absPattern := filepath.Join(baseDir, cleanedPattern)
		matches, err := filepath.Glob(absPattern)
		if err != nil {
			return fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
		}
		if len(matches) == 0 {
			data, readErr := os.ReadFile(absPattern)
			if readErr != nil {
				return fmt.Errorf("%s file %q: %w", canonicalName, pattern, readErr)
			}
			baseName := filepath.Base(absPattern)
			outPath := filepath.Clean(fmt.Sprintf("skills/%s/%s", id, baseName))
			out.Files[outPath] = string(data)
			continue
		}
		for _, match := range matches {
			data, readErr := os.ReadFile(match)
			if readErr != nil {
				return fmt.Errorf("%s file %q: %w", canonicalName, match, readErr)
			}
			baseName := filepath.Base(match)
			outPath := filepath.Clean(fmt.Sprintf("skills/%s/%s", id, baseName))
			out.Files[outPath] = string(data)
		}
	}
	return nil
}

// CompileSkillSubdir reads files from a skill subdirectory (references/, scripts/, assets/)
// and adds them to the output map at skills/<id>/<outputSubdir>/<filename>.
//
// canonicalSubdir is used in error messages and represents the logical name (e.g. "references").
// outputSubdir is the provider-native directory name written to the output path (e.g. "resources").
// Passing the same value for both parameters produces identity translation.
//
// Each pattern in paths is resolved relative to filepath.Join(baseDir, skillSourceDir).
// skillSourceDir is the skill-source root within the project (e.g. "xcaf/skills/<id>").
// Pass an empty string to resolve patterns directly from baseDir (legacy behavior).
// Path traversal above baseDir is rejected. Glob patterns are expanded; literal paths are read directly.
func CompileSkillSubdir(id, canonicalSubdir, outputSubdir string, paths []string, baseDir, skillSourceDir string, out *output.Output) error {
	if len(paths) == 0 {
		return nil
	}

	cleanedSourceDir := filepath.Clean(skillSourceDir)
	if cleanedSourceDir != "." && strings.HasPrefix(cleanedSourceDir, "..") {
		return fmt.Errorf("skill source directory %q contains path traversal", skillSourceDir)
	}

	for _, pattern := range paths {
		// Security: pattern must not traverse above baseDir.
		cleanedPattern := filepath.Clean(pattern)
		if strings.HasPrefix(cleanedPattern, "..") {
			return fmt.Errorf("%s path %q traverses above the project root", canonicalSubdir, pattern)
		}

		absPattern := filepath.Join(baseDir, skillSourceDir, cleanedPattern)

		// Expand glob patterns (e.g. "docs/schema/*.sql")
		matches, err := filepath.Glob(absPattern)
		if err != nil {
			return fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
		}
		if len(matches) == 0 {
			// Treat as a literal path — if missing, it's an error.
			data, readErr := os.ReadFile(absPattern)
			if readErr != nil {
				return fmt.Errorf("%s file %q: %w", canonicalSubdir, pattern, readErr)
			}
			baseName := filepath.Base(absPattern)
			outPath := filepath.Clean(fmt.Sprintf("skills/%s/%s/%s", id, outputSubdir, baseName))
			out.Files[outPath] = string(data)
			continue
		}

		for _, match := range matches {
			data, err := os.ReadFile(match)
			if err != nil {
				return fmt.Errorf("%s file %q: %w", canonicalSubdir, match, err)
			}
			baseName := filepath.Base(match)
			outPath := filepath.Clean(fmt.Sprintf("skills/%s/%s/%s", id, outputSubdir, baseName))
			out.Files[outPath] = string(data)
		}
	}
	return nil
}
