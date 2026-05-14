package renderer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
)

// FlattenOpts holds parameters for FlattenToSkillRoot.
type FlattenOpts struct {
	ID            string
	CanonicalName string
	Paths         []string
	BaseDir       string
}

// SkillSubdirOpts holds parameters for CompileSkillSubdir.
type SkillSubdirOpts struct {
	ID              string
	CanonicalSubdir string
	OutputSubdir    string
	Paths           []string
	BaseDir         string
	SkillSourceDir  string
}

// SkillArtifactContext holds the shared context passed to provider
// compileSkillArtifacts functions.
type SkillArtifactContext struct {
	ID             string
	Skill          ast.SkillConfig
	Caps           CapabilitySet
	BaseDir        string
	SkillSourceDir string
}

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

// FlattenToSkillRoot reads files matching opts.Paths (globs or literals) relative
// to opts.BaseDir and writes them directly to skills/<id>/<filename> in out.Files —
// no subdirectory is created. This is used by providers that co-locate all skill
// files alongside SKILL.md (e.g., Claude examples, Copilot all subdirs).
func FlattenToSkillRoot(opts FlattenOpts, out *output.Output) error {
	if len(opts.Paths) == 0 {
		return nil
	}
	for _, pattern := range opts.Paths {
		cleanedPattern := filepath.Clean(pattern)
		if strings.HasPrefix(cleanedPattern, "..") {
			return fmt.Errorf("%s path %q traverses above the project root", opts.CanonicalName, pattern)
		}
		absPattern := filepath.Join(opts.BaseDir, cleanedPattern)
		matches, err := filepath.Glob(absPattern)
		if err != nil {
			return fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
		}
		if len(matches) == 0 {
			data, readErr := os.ReadFile(absPattern)
			if readErr != nil {
				return fmt.Errorf("%s file %q: %w", opts.CanonicalName, pattern, readErr)
			}
			baseName := filepath.Base(absPattern)
			outPath := filepath.Clean(fmt.Sprintf("skills/%s/%s", opts.ID, baseName))
			out.Files[outPath] = string(data)
			continue
		}
		for _, match := range matches {
			data, readErr := os.ReadFile(match)
			if readErr != nil {
				return fmt.Errorf("%s file %q: %w", opts.CanonicalName, match, readErr)
			}
			baseName := filepath.Base(match)
			outPath := filepath.Clean(fmt.Sprintf("skills/%s/%s", opts.ID, baseName))
			out.Files[outPath] = string(data)
		}
	}
	return nil
}

// DiscoverArtifactFiles walks a single artifact subdirectory inside a skill source
// directory and returns a list of file paths relative to the skill directory.
// Only immediate children are returned (no recursion into nested dirs).
// If the directory does not exist, an empty slice is returned (not an error).
//
// Example: for artifactName="references" and skillSourceDir="xcaf/skills/my-skill",
// if the directory contains guide.md and patterns.md, the result is:
//
//	["references/guide.md", "references/patterns.md"]
func DiscoverArtifactFiles(baseDir, skillSourceDir, artifactName string) ([]string, error) {
	cleaned := filepath.Clean(artifactName)
	if strings.HasPrefix(cleaned, "..") {
		return nil, fmt.Errorf("artifact name %q traverses above the skill directory", artifactName)
	}
	dir := filepath.Join(baseDir, skillSourceDir, cleaned)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading artifact directory %q: %w", dir, err)
	}

	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		paths = append(paths, filepath.Join(artifactName, entry.Name()))
	}
	sort.Strings(paths)
	return paths, nil
}

// ArtifactJob bundles the inputs for non-fatal artifact compilation.
// Used by providers that demote artifact errors to fidelity notes.
type ArtifactJob struct {
	ID        string
	BaseDir   string
	Caps      CapabilitySet
	Files     map[string]string
	SourceDir string // optional; defaults to xcaf/skills/<id>
}

// CompileArtifactsDemoted discovers and compiles artifact subdirs. Errors are
// demoted to fidelity notes so the rest of the resource still compiles.
func CompileArtifactsDemoted(target string, j ArtifactJob, artifacts []string) []FidelityNote {
	var notes []FidelityNote
	sourceDir := j.SourceDir
	if sourceDir == "" {
		sourceDir = filepath.Join("xcaf", "skills", j.ID)
	}
	subOut := &output.Output{Files: make(map[string]string)}

	for _, artifactName := range artifacts {
		outputSubdir, ok := j.Caps.SkillArtifactDirs[artifactName]
		if !ok {
			outputSubdir = artifactName
		}
		paths, err := DiscoverArtifactFiles(j.BaseDir, sourceDir, artifactName)
		if err != nil {
			notes = append(notes, FidelityNote{
				Level:      LevelWarning,
				Target:     target,
				Kind:       "skill",
				Resource:   j.ID,
				Field:      artifactName,
				Code:       CodeSkillReferencesDropped,
				Reason:     fmt.Sprintf("skill %s artifact %s: discover files: %s", j.ID, artifactName, err),
				Mitigation: "Check file paths in " + artifactName,
			})
			continue
		}
		if len(paths) == 0 {
			continue
		}
		if err := CompileSkillSubdir(SkillSubdirOpts{
			ID:              j.ID,
			CanonicalSubdir: artifactName,
			OutputSubdir:    outputSubdir,
			Paths:           paths,
			BaseDir:         j.BaseDir,
			SkillSourceDir:  sourceDir,
		}, subOut); err != nil {
			notes = append(notes, FidelityNote{
				Level:      LevelWarning,
				Target:     target,
				Kind:       "skill",
				Resource:   j.ID,
				Field:      artifactName,
				Code:       CodeSkillReferencesDropped,
				Reason:     err.Error(),
				Mitigation: "Check file paths in " + artifactName,
			})
		}
	}
	for k, v := range subOut.Files {
		j.Files[k] = v
	}
	return notes
}

// CompileSkillSubdir reads files from a skill subdirectory (references/, scripts/, assets/)
// and adds them to the output map at skills/<id>/<outputSubdir>/<filename>.
//
// opts.CanonicalSubdir is used in error messages and represents the logical name
// (e.g. "references"). opts.OutputSubdir is the provider-native directory name
// written to the output path (e.g. "resources"). Passing the same value for both
// produces identity translation.
//
// Each pattern in opts.Paths is resolved relative to
// filepath.Join(opts.BaseDir, opts.SkillSourceDir). opts.SkillSourceDir is the
// skill-source root within the project (e.g. "xcaf/skills/<id>"). Pass an empty
// string to resolve patterns directly from opts.BaseDir (legacy behavior).
// Path traversal above BaseDir is rejected. Glob patterns are expanded; literal
// paths are read directly.
func CompileSkillSubdir(opts SkillSubdirOpts, out *output.Output) error {
	if len(opts.Paths) == 0 {
		return nil
	}

	cleanedSourceDir := filepath.Clean(opts.SkillSourceDir)
	if cleanedSourceDir != "." && strings.HasPrefix(cleanedSourceDir, "..") {
		return fmt.Errorf("skill source directory %q contains path traversal", opts.SkillSourceDir)
	}

	for _, pattern := range opts.Paths {
		// Security: pattern must not traverse above BaseDir.
		cleanedPattern := filepath.Clean(pattern)
		if strings.HasPrefix(cleanedPattern, "..") {
			return fmt.Errorf("%s path %q traverses above the project root", opts.CanonicalSubdir, pattern)
		}

		absPattern := filepath.Join(opts.BaseDir, opts.SkillSourceDir, cleanedPattern)

		// Expand glob patterns (e.g. "docs/schema/*.sql")
		matches, err := filepath.Glob(absPattern)
		if err != nil {
			return fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
		}
		if len(matches) == 0 {
			// Treat as a literal path — if missing, it's an error.
			data, readErr := os.ReadFile(absPattern)
			if readErr != nil {
				return fmt.Errorf("%s file %q: %w", opts.CanonicalSubdir, pattern, readErr)
			}
			baseName := filepath.Base(absPattern)
			outPath := filepath.Clean(fmt.Sprintf("skills/%s/%s/%s", opts.ID, opts.OutputSubdir, baseName))
			out.Files[outPath] = string(data)
			continue
		}

		for _, match := range matches {
			data, err := os.ReadFile(match)
			if err != nil {
				return fmt.Errorf("%s file %q: %w", opts.CanonicalSubdir, match, err)
			}
			baseName := filepath.Base(match)
			outPath := filepath.Clean(fmt.Sprintf("skills/%s/%s/%s", opts.ID, opts.OutputSubdir, baseName))
			out.Files[outPath] = string(data)
		}
	}
	return nil
}

// WorkflowArtifactArgs bundles parameters for AppendWorkflowArtifacts.
type WorkflowArtifactArgs struct {
	Target    string
	Workflows map[string]ast.WorkflowConfig
	BaseDir   string
	Caps      CapabilitySet
	Files     map[string]string
}

// AppendWorkflowArtifacts copies artifact directories for each workflow that
// declares them. Missing optional directories are demoted to fidelity notes so
// the rest of the workflow still compiles. Results are appended to files in place
// and any notes are returned to the caller for aggregation.
func AppendWorkflowArtifacts(args WorkflowArtifactArgs) []FidelityNote {
	var notes []FidelityNote
	for id, wf := range args.Workflows {
		if len(wf.Artifacts) == 0 {
			continue
		}
		artifactNotes := CompileArtifactsDemoted(args.Target, ArtifactJob{
			ID:        id,
			BaseDir:   args.BaseDir,
			Caps:      args.Caps,
			Files:     args.Files,
			SourceDir: filepath.Join("xcaf", "workflows", id),
		}, wf.Artifacts)
		notes = append(notes, artifactNotes...)
	}
	return notes
}
