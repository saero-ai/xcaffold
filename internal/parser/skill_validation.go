package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var canonicalSkillSubdirs = map[string]bool{
	"references": true,
	"scripts":    true,
	"assets":     true,
	"examples":   true,
}

var subdirAllowedExtensions = map[string][]string{
	"references": {".md", ".mdx", ".json", ".yaml", ".yml", ".toml", ".txt"},
	"scripts":    {".sh", ".bash", ".py", ".js", ".ts", ".ps1"},
	"examples":   {".md", ".txt"},
}

// SkillValidationResult separates hard errors from advisory warnings produced
// by ValidateSkillDirectory. Errors indicate structural violations that must be
// fixed; Warnings indicate misplaced file types that are allowed but unusual.
type SkillValidationResult struct {
	Errors   []error
	Warnings []error
}

// ValidateSkillDirectory checks that a skill directory conforms to the canonical
// 4-subdirectory layout. It returns a SkillValidationResult with hard errors
// (unknown subdirs, stray files, nested subdirs) and advisory warnings
// (misplaced file types). Both slices are nil when the directory is fully valid.
func ValidateSkillDirectory(skillDir, skillID string) *SkillValidationResult {
	result := &SkillValidationResult{}

	entries, err := os.ReadDir(skillDir)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("cannot read skill directory %q: %w", skillDir, err))
		return result
	}

	legacyXcfFile := skillID + ".xcf"

	for _, entry := range entries {
		name := entry.Name()

		if entry.IsDir() {
			if !canonicalSkillSubdirs[name] {
				result.Errors = append(result.Errors, fmt.Errorf(
					"unknown subdirectory %q in skill %q; use references/, scripts/, assets/, or examples/",
					name, skillID))
				continue
			}
			subdirPath := filepath.Join(skillDir, name)
			errs, warns := validateSubdir(subdirPath, skillID, name)
			result.Errors = append(result.Errors, errs...)
			result.Warnings = append(result.Warnings, warns...)
			continue
		}

		// Skip hidden files (dotfiles like .DS_Store, .gitkeep, .gitignore).
		// These are not user-authored xcf content and must not be flagged.
		if strings.HasPrefix(name, ".") {
			continue
		}

		// Accept both canonical "skill.xcf" and legacy "{skillID}.xcf"
		if name == "skill.xcf" || strings.EqualFold(name, legacyXcfFile) {
			continue
		}
		result.Errors = append(result.Errors, fmt.Errorf(
			"unrecognized file %q at skill root %q; move to references/, scripts/, assets/, or examples/ based on its purpose",
			name, skillID))
	}

	return result
}

// validateSubdir reads subdirPath once and checks both max-depth and file-type
// constraints in a single pass. It returns (errors, warnings).
func validateSubdir(subdirPath, skillID, subdirName string) ([]error, []error) {
	entries, err := os.ReadDir(subdirPath)
	if err != nil {
		return []error{fmt.Errorf("cannot read %s/%s/: %w", skillID, subdirName, err)}, nil
	}
	allowed := subdirAllowedExtensions[subdirName]
	var errs []error
	var warns []error
	for _, entry := range entries {
		if entry.IsDir() {
			errs = append(errs, fmt.Errorf(
				"nested subdirectory %q inside %s/%s/ is not allowed; max depth is 1",
				entry.Name(), skillID, subdirName))
			continue
		}
		if allowed == nil {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		found := false
		for _, a := range allowed {
			if ext == a {
				found = true
				break
			}
		}
		if !found {
			warns = append(warns, fmt.Errorf(
				"file %q in %s/%s/ has extension %q which is not typical for %s; consider moving to the appropriate subdirectory",
				entry.Name(), skillID, subdirName, ext, subdirName))
		}
	}
	return errs, warns
}
