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

// ValidateSkillDirectory checks that a skill directory conforms to the canonical
// 4-subdirectory layout. It returns a slice of errors — one per violation found.
// An empty slice means the directory is valid.
func ValidateSkillDirectory(skillDir, skillID string) []error {
	var errs []error

	entries, err := os.ReadDir(skillDir)
	if err != nil {
		return []error{fmt.Errorf("cannot read skill directory %q: %w", skillDir, err)}
	}

	xcfFile := skillID + ".xcf"

	for _, entry := range entries {
		name := entry.Name()

		if entry.IsDir() {
			if !canonicalSkillSubdirs[name] {
				errs = append(errs, fmt.Errorf(
					"unknown subdirectory %q in skill %q; use references/, scripts/, assets/, or examples/",
					name, skillID))
				continue
			}
			errs = append(errs, validateSubdirDepth(filepath.Join(skillDir, name), skillID, name)...)
			errs = append(errs, validateSubdirFileTypes(filepath.Join(skillDir, name), skillID, name)...)
			continue
		}

		if name != xcfFile {
			errs = append(errs, fmt.Errorf(
				"unrecognized file %q at skill root %q; move to references/, scripts/, assets/, or examples/ based on its purpose",
				name, skillID))
		}
	}

	return errs
}

func validateSubdirDepth(subdirPath, skillID, subdirName string) []error {
	var errs []error
	entries, err := os.ReadDir(subdirPath)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		if entry.IsDir() {
			errs = append(errs, fmt.Errorf(
				"nested subdirectory %q inside %s/%s/ is not allowed; max depth is 1",
				entry.Name(), skillID, subdirName))
		}
	}
	return errs
}

func validateSubdirFileTypes(subdirPath, skillID, subdirName string) []error {
	allowed, hasRestriction := subdirAllowedExtensions[subdirName]
	if !hasRestriction {
		return nil
	}

	var errs []error
	entries, _ := os.ReadDir(subdirPath)
	for _, entry := range entries {
		if entry.IsDir() {
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
			errs = append(errs, fmt.Errorf(
				"file %q in %s/%s/ has extension %q which is not typical for %s; consider moving to the appropriate subdirectory",
				entry.Name(), skillID, subdirName, ext, subdirName))
		}
	}
	return errs
}
