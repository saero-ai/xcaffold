package renderer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/output"
)

// CompileSkillSubdir reads files from a skill subdirectory (references/, scripts/, assets/)
// and adds them to the output map at skills/<id>/<subdir>/<filename>.
//
// Each pattern in paths is resolved relative to baseDir. Path traversal above
// baseDir is rejected. Glob patterns are expanded; literal paths are read directly.
//
// Supported subdirs: references, scripts, assets.
func CompileSkillSubdir(id, subdir string, paths []string, baseDir string, out *output.Output) error {
	if len(paths) == 0 {
		return nil
	}

	for _, pattern := range paths {
		// Security: pattern must not traverse above baseDir.
		cleanedPattern := filepath.Clean(pattern)
		if strings.HasPrefix(cleanedPattern, "..") {
			return fmt.Errorf("%s path %q traverses above the project root", subdir, pattern)
		}

		absPattern := filepath.Join(baseDir, cleanedPattern)

		// Expand glob patterns (e.g. "docs/schema/*.sql")
		matches, err := filepath.Glob(absPattern)
		if err != nil {
			return fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
		}
		if len(matches) == 0 {
			// Treat as a literal path — if missing, it's an error.
			data, readErr := os.ReadFile(absPattern)
			if readErr != nil {
				return fmt.Errorf("%s file %q: %w", subdir, pattern, readErr)
			}
			baseName := filepath.Base(absPattern)
			outPath := filepath.Clean(fmt.Sprintf("skills/%s/%s/%s", id, subdir, baseName))
			out.Files[outPath] = string(data)
			continue
		}

		for _, match := range matches {
			data, err := os.ReadFile(match)
			if err != nil {
				return fmt.Errorf("%s file %q: %w", subdir, match, err)
			}
			baseName := filepath.Base(match)
			outPath := filepath.Clean(fmt.Sprintf("skills/%s/%s/%s", id, subdir, baseName))
			out.Files[outPath] = string(data)
		}
	}
	return nil
}
