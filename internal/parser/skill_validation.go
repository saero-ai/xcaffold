package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/providers"
)

var subdirExtensionRules = map[string]bool{
	"references": true,
	"scripts":    true,
	"assets":     true,
	"examples":   true,
}

var subdirAllowedExtensions = map[string][]string{
	"references": {".md", ".mdx", ".json", ".yaml", ".yml", ".toml", ".txt"},
	"scripts":    {".sh", ".bash", ".py", ".js", ".ts", ".ps1"},
	"examples":   {".md", ".txt", ".xcaf"},
}

// SkillValidationResult separates hard errors from advisory warnings produced
// by ValidateSkillDirectory. Errors indicate structural violations that must be
// fixed; Warnings indicate misplaced file types that are allowed but unusual.
type SkillValidationResult struct {
	Errors   []error
	Warnings []error
}

// ValidateSkillDirectory checks that a skill directory conforms to the
// artifacts-based layout. The artifacts parameter lists declared subdirectories
// that should exist and be validated. It returns a SkillValidationResult with
// hard errors (stray files, nested subdirs, missing declared artifacts) and
// advisory warnings (subdirs on disk not in artifacts list). Both slices are
// nil when the directory is fully valid.
// processSkillDirectories walks skill root and validates directories against declared artifacts.
func processSkillDirectories(entries []os.DirEntry, skillDir, skillID string, artifactsMap map[string]bool, declaredArtifactsSeen map[string]bool) ([]error, []error) {
	var errs, warns []error
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() {
			continue
		}
		// Check if this directory is in the artifacts list
		if artifactsMap[name] {
			declaredArtifactsSeen[name] = true
			// It's declared — validate its contents
			subdirPath := filepath.Join(skillDir, name)
			subErrs, subWarns := validateSubdir(subdirPath, skillID, name)
			errs = append(errs, subErrs...)
			warns = append(warns, subWarns...)
		} else {
			// Directory exists but is not declared — warn
			warns = append(warns, fmt.Errorf(
				"subdirectory %q in skill %q is not declared in artifacts and will not be compiled; add it to artifacts: [%s] in the skill config",
				name, skillID, name))
		}
	}
	return errs, warns
}

// processSkillFiles walks skill root files and validates against known patterns.
func processSkillFiles(entries []os.DirEntry, skillID string) []error {
	var errs []error
	legacyXcafFile := skillID + ".xcaf"
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()

		// Skip hidden files (dotfiles like .DS_Store, .gitkeep, .gitignore).
		// These are not user-authored xcaf content and must not be flagged.
		if strings.HasPrefix(name, ".") {
			continue
		}

		// Accept both canonical "skill.xcaf" and legacy "{skillID}.xcaf"
		if name == "skill.xcaf" || strings.EqualFold(name, legacyXcafFile) {
			continue
		}
		// Accept override files: <kind>.<provider>.xcaf pattern
		if isOverrideFilename(name) {
			continue
		}
		errs = append(errs, fmt.Errorf(
			"unrecognized file %q at skill root %q; move it to a declared artifact subdirectory",
			name, skillID))
	}
	return errs
}

// validateDeclaredArtifactsExist checks that all declared artifacts exist on disk.
func validateDeclaredArtifactsExist(artifacts []string, declaredArtifactsSeen map[string]bool, skillID string) []error {
	var errs []error
	for _, declared := range artifacts {
		if !declaredArtifactsSeen[declared] {
			errs = append(errs, fmt.Errorf(
				"declared artifact %q does not exist as subdirectory in skill %q",
				declared, skillID))
		}
	}
	return errs
}

func ValidateSkillDirectory(skillDir, skillID string, artifacts []string) *SkillValidationResult {
	result := &SkillValidationResult{}

	entries, err := os.ReadDir(skillDir)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("cannot read skill directory %q: %w", skillDir, err))
		return result
	}

	// Build a map of declared artifacts for fast lookup
	artifactsMap := make(map[string]bool)
	for _, a := range artifacts {
		artifactsMap[a] = true
	}

	// Track which declared artifacts we've seen on disk
	declaredArtifactsSeen := make(map[string]bool)

	// Validate directories
	errs, warns := processSkillDirectories(entries, skillDir, skillID, artifactsMap, declaredArtifactsSeen)
	result.Errors = append(result.Errors, errs...)
	result.Warnings = append(result.Warnings, warns...)

	// Validate files
	fileErrs := processSkillFiles(entries, skillID)
	result.Errors = append(result.Errors, fileErrs...)

	// Check that all declared artifacts exist
	artifactErrs := validateDeclaredArtifactsExist(artifacts, declaredArtifactsSeen, skillID)
	result.Errors = append(result.Errors, artifactErrs...)

	return result
}

// validateSubdir reads subdirPath once and checks both max-depth and file-type
// constraints in a single pass. It returns (errors, warnings).
// For canonical subdirs (in canonicalSkillSubdirs), it enforces extension checks.
// For custom artifact dirs, any file type is allowed.
func validateSubdir(subdirPath, skillID, subdirName string) ([]error, []error) {
	entries, err := os.ReadDir(subdirPath)
	if err != nil {
		return []error{fmt.Errorf("cannot read %s/%s/: %w", skillID, subdirName, err)}, nil
	}
	var errs []error
	var warns []error
	for _, entry := range entries {
		if entry.IsDir() {
			errs = append(errs, fmt.Errorf(
				"nested subdirectory %q inside %s/%s/ is not allowed; max depth is 1",
				entry.Name(), skillID, subdirName))
			continue
		}
		// Only check file extensions for canonical subdirs
		// Custom artifact dirs allow any file type
		if !subdirExtensionRules[subdirName] {
			continue
		}
		allowed := subdirAllowedExtensions[subdirName]
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

var validOverrideKinds = map[string]bool{
	"agent":    true,
	"skill":    true,
	"rule":     true,
	"workflow": true,
	"mcp":      true,
	"hooks":    true,
	"settings": true,
	"policy":   true,
	"template": true,
	"memory":   true,
}

func isOverrideFilename(name string) bool {
	if !strings.HasSuffix(name, ".xcaf") {
		return false
	}
	base := strings.TrimSuffix(name, ".xcaf")
	parts := strings.Split(base, ".")
	if len(parts) != 2 {
		return false
	}
	return validOverrideKinds[parts[0]] && providers.IsRegistered(parts[1])
}
