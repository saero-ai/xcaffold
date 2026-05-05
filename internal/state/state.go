package state

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/saero-ai/xcaffold/internal/output"
	"gopkg.in/yaml.v3"
)

var XcaffoldVersion = "1.0.0-dev"

// SourceFile tracks a source .xcf file and its SHA-256 content hash.
type SourceFile struct {
	Path string `yaml:"path"`
	Hash string `yaml:"hash"`
}

// Artifact tracks a single generated file and its SHA-256 content hash.
type Artifact struct {
	Path string `yaml:"path"`
	Hash string `yaml:"hash"`
}

// MemorySeed tracks a single seeded memory file, including its content hash
// at seed time. Used for drift detection on tracked memory entries.
type MemorySeed struct {
	Name     string `yaml:"name"`
	Target   string `yaml:"target"`
	Path     string `yaml:"path"`
	Hash     string `yaml:"hash"`
	SeededAt string `yaml:"seeded-at"`
}

const stateFileVersion = 1

// StateManifest is the schema for the per-blueprint .xcaffold/state.yaml file.
// It groups artifacts under a named target map and records which blueprint
// produced the output.
type StateManifest struct {
	Version         int                    `yaml:"version"`
	XcaffoldVersion string                 `yaml:"xcaffold-version"`
	Blueprint       string                 `yaml:"blueprint,omitempty"`
	BlueprintHash   string                 `yaml:"blueprint-hash,omitempty"`
	SourceFiles     []SourceFile           `yaml:"source-files"`
	Targets         map[string]TargetState `yaml:"targets"`
	MemorySeeds     []MemorySeed           `yaml:"memory-seeds,omitempty"`
}

// TargetState records the last-applied timestamp, artifact list, and source files
// for a single compilation target within a StateManifest.
type TargetState struct {
	LastApplied string       `yaml:"last-applied"`
	Artifacts   []Artifact   `yaml:"artifacts"`
	SourceFiles []SourceFile `yaml:"source-files,omitempty"`
}

// StateOpts holds the inputs needed to build a StateManifest from compiler
// output.
type StateOpts struct {
	Blueprint     string
	BlueprintHash string
	Target        string
	BaseDir       string
	SourceFiles   []string
	MemorySeeds   []MemorySeed
}

// StateDir returns the .xcaffold state directory path for the given project base
// directory. The returned path is always baseDir/.xcaffold.
func StateDir(baseDir string) string {
	return filepath.Join(baseDir, ".xcaffold")
}

// StateFilePath returns the full path to the state file for a given blueprint.
// If blueprintName is empty, the default filename "project.xcf.state" is used.
// The blueprintName is sanitized with filepath.Base to prevent directory traversal.
func StateFilePath(baseDir, blueprintName string) string {
	name := "project"
	if blueprintName != "" {
		name = filepath.Base(blueprintName)
	}
	return filepath.Clean(filepath.Join(baseDir, ".xcaffold", name+".xcf.state"))
}

// ListStateFiles returns a sorted list of all .xcf.state files in the given
// state directory. Returns an empty slice if the directory doesn't exist or
// contains no state files.
func ListStateFiles(stateDir string) ([]string, error) {
	entries, err := os.ReadDir(stateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".xcf.state") {
			files = append(files, filepath.Join(stateDir, entry.Name()))
		}
	}
	sort.Strings(files)
	return files, nil
}

// sortMemorySeeds sorts a slice of MemorySeed in ascending order by Name.
func sortMemorySeeds(seeds []MemorySeed) {
	sort.Slice(seeds, func(i, j int) bool {
		return seeds[i].Name < seeds[j].Name
	})
}

// WriteState serializes a StateManifest to a YAML file, creating parent
// directories as needed. Files are written with 0600 permissions.
func WriteState(manifest *StateManifest, path string) error {
	cleanPath := filepath.Clean(path)

	if err := os.MkdirAll(filepath.Dir(cleanPath), 0700); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	data, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal state manifest: %w", err)
	}

	if err := os.WriteFile(cleanPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write state file to %q: %w", cleanPath, err)
	}

	return nil
}

// ReadState deserializes a StateManifest from a YAML file.
// Uses plain Unmarshal (not KnownFields) because state files are tool-generated
// and must tolerate fields added by newer xcaffold versions.
func ReadState(path string) (*StateManifest, error) {
	cleanPath := filepath.Clean(path)

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file %q: %w", cleanPath, err)
	}

	var manifest StateManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse state file %q: %w", cleanPath, err)
	}

	return &manifest, nil
}

// GenerateState creates a StateManifest from compiler output and metadata.
// If existing is non-nil, its Targets are preserved (merged with the new target).
// Returns an error if opts.Target is empty.
func GenerateState(out *output.Output, opts StateOpts, existing *StateManifest) (*StateManifest, error) {
	if opts.Target == "" {
		return nil, fmt.Errorf("state.GenerateState: target must not be empty")
	}

	now := time.Now().UTC().Format(time.RFC3339)

	manifest := &StateManifest{
		Version:         stateFileVersion,
		XcaffoldVersion: XcaffoldVersion,
		Blueprint:       opts.Blueprint,
		BlueprintHash:   opts.BlueprintHash,
		Targets:         make(map[string]TargetState),
	}

	// Preserve existing targets
	if existing != nil {
		for k, v := range existing.Targets {
			manifest.Targets[k] = v
		}
		// Preserve existing source files if opts doesn't provide new ones
		if len(opts.SourceFiles) == 0 && len(existing.SourceFiles) > 0 {
			manifest.SourceFiles = existing.SourceFiles
		}
	}

	// Build artifacts for the current target
	target := opts.Target

	artifacts := make([]Artifact, 0, len(out.Files)+len(out.RootFiles))
	for path, content := range out.Files {
		hash := sha256.Sum256([]byte(content))
		artifacts = append(artifacts, Artifact{
			Path: path,
			Hash: fmt.Sprintf("sha256:%x", hash),
		})
	}
	for path, content := range out.RootFiles {
		hash := sha256.Sum256([]byte(content))
		artifacts = append(artifacts, Artifact{
			Path: "root:" + path,
			Hash: fmt.Sprintf("sha256:%x", hash),
		})
	}
	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].Path < artifacts[j].Path
	})

	// Build target state with source files
	targetState := TargetState{
		LastApplied: now,
		Artifacts:   artifacts,
	}

	// Hash source files if provided
	if len(opts.SourceFiles) > 0 {
		sourceFiles := make([]SourceFile, 0, len(opts.SourceFiles))
		for _, absPath := range opts.SourceFiles {
			data, err := os.ReadFile(filepath.Clean(absPath))
			if err != nil {
				continue
			}
			relPath, err := filepath.Rel(opts.BaseDir, absPath)
			if err != nil {
				relPath = filepath.Base(absPath)
			}
			hash := sha256.Sum256(data)
			sourceFiles = append(sourceFiles, SourceFile{
				Path: relPath,
				Hash: fmt.Sprintf("sha256:%x", hash),
			})
		}
		sort.Slice(sourceFiles, func(i, j int) bool {
			return sourceFiles[i].Path < sourceFiles[j].Path
		})
		targetState.SourceFiles = sourceFiles
		// Also store at top level for backward compat (old state files)
		manifest.SourceFiles = sourceFiles
	}

	manifest.Targets[target] = targetState

	// Copy and sort memory seeds
	if len(opts.MemorySeeds) > 0 {
		manifest.MemorySeeds = make([]MemorySeed, len(opts.MemorySeeds))
		copy(manifest.MemorySeeds, opts.MemorySeeds)
		for i := range manifest.MemorySeeds {
			if manifest.MemorySeeds[i].SeededAt == "" {
				manifest.MemorySeeds[i].SeededAt = now
			}
		}
		sortMemorySeeds(manifest.MemorySeeds)
	}

	return manifest, nil
}

// SourcesChanged compares the previous state's source file hashes against the
// current source files on disk. Returns true if any source has changed, been
// added, or been removed. Returns true if previous is nil/empty (first run).
func SourcesChanged(previous []SourceFile, currentPaths []string, baseDir string) (bool, error) {
	if len(previous) == 0 {
		return true, nil // first run or no source tracking
	}
	if len(previous) != len(currentPaths) {
		return true, nil // file count changed
	}

	prevByPath := make(map[string]string, len(previous))
	for _, sf := range previous {
		prevByPath[sf.Path] = sf.Hash
	}

	for _, absPath := range currentPaths {
		relPath, err := filepath.Rel(baseDir, absPath)
		if err != nil {
			return true, nil
		}
		prevHash, exists := prevByPath[relPath]
		if !exists {
			return true, nil // new file
		}

		data, err := os.ReadFile(filepath.Clean(absPath))
		if err != nil {
			return true, fmt.Errorf("failed to read source file %q: %w", absPath, err)
		}
		hash := sha256.Sum256(data)
		currentHash := fmt.Sprintf("sha256:%x", hash)
		if currentHash != prevHash {
			return true, nil
		}
	}

	return false, nil
}

// FindOrphansFromState returns artifact paths that exist in the old StateManifest
// for the given target but are absent from newFiles or newRootFiles.
func FindOrphansFromState(old *StateManifest, target string, newFiles map[string]string, newRootFiles map[string]string) []string {
	if old == nil {
		return nil
	}
	ts, ok := old.Targets[target]
	if !ok {
		return nil
	}
	var orphans []string
	for _, artifact := range ts.Artifacts {
		if strings.HasPrefix(artifact.Path, "root:") {
			relPath := strings.TrimPrefix(artifact.Path, "root:")
			if _, exists := newRootFiles[relPath]; !exists {
				orphans = append(orphans, artifact.Path)
			}
		} else {
			if _, exists := newFiles[artifact.Path]; !exists {
				orphans = append(orphans, artifact.Path)
			}
		}
	}
	sort.Strings(orphans)
	return orphans
}
