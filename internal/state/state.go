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

const lockFileVersion = 2

// LockFilePath returns the lock file path for a given target.
// For "claude" or empty string (default), returns basePath unchanged for backward compatibility.
// For other targets, inserts the target name before the extension:
// e.g., "scaffold.lock" + "cursor" → "scaffold.cursor.lock"
func LockFilePath(basePath string, target string) string {
	if target == "" {
		target = "claude"
	}
	ext := filepath.Ext(basePath)
	base := strings.TrimSuffix(basePath, ext)
	return base + "." + target + ext
}

var XcaffoldVersion = "1.0.0-dev"

const claudeSchemaVersion = "alpha"

// SourceFile tracks a source .xcf file and its SHA-256 content hash.
type SourceFile struct {
	Path string `yaml:"path"`
	Hash string `yaml:"hash"`
}

// LockManifest is the schema for the scaffold.lock file.
type LockManifest struct {
	LastApplied         string       `yaml:"last_applied"`
	XcaffoldVersion     string       `yaml:"xcaffold_version"`
	ClaudeSchemaVersion string       `yaml:"claude_schema_version"`
	Target              string       `yaml:"target"`
	Scope               string       `yaml:"scope"`
	ConfigDir           string       `yaml:"config_directory"`
	SourceFiles         []SourceFile `yaml:"source_files"`
	Artifacts           []Artifact   `yaml:"artifacts"`
	Version             int          `yaml:"version"`
}

// Artifact tracks a single generated file and its SHA-256 content hash.
type Artifact struct {
	Path string `yaml:"path"`
	Hash string `yaml:"hash"`
}

// GenerateOpts holds additional metadata for lock manifest generation.
type GenerateOpts struct {
	Target      string
	Scope       string
	ConfigDir   string
	BaseDir     string   // base directory for computing relative source paths
	SourceFiles []string // absolute paths to source .xcf files
}

// GenerateWithOpts creates a LockManifest from compiler output and source metadata.
func GenerateWithOpts(out *output.Output, opts GenerateOpts) *LockManifest {
	manifest := &LockManifest{
		Version:             lockFileVersion,
		LastApplied:         time.Now().UTC().Format(time.RFC3339),
		XcaffoldVersion:     XcaffoldVersion,
		ClaudeSchemaVersion: claudeSchemaVersion,
		Target:              opts.Target,
		Scope:               opts.Scope,
		ConfigDir:           opts.ConfigDir,
		Artifacts:           make([]Artifact, 0, len(out.Files)),
		SourceFiles:         make([]SourceFile, 0, len(opts.SourceFiles)),
	}

	// Hash output artifacts
	for path, content := range out.Files {
		hash := sha256.Sum256([]byte(content))
		manifest.Artifacts = append(manifest.Artifacts, Artifact{
			Path: path,
			Hash: fmt.Sprintf("sha256:%x", hash),
		})
	}
	sort.Slice(manifest.Artifacts, func(i, j int) bool {
		return manifest.Artifacts[i].Path < manifest.Artifacts[j].Path
	})

	// Hash source files
	for _, absPath := range opts.SourceFiles {
		data, err := os.ReadFile(filepath.Clean(absPath))
		if err != nil {
			continue // skip unreadable files
		}
		relPath, err := filepath.Rel(opts.BaseDir, absPath)
		if err != nil {
			relPath = filepath.Base(absPath)
		}
		hash := sha256.Sum256(data)
		manifest.SourceFiles = append(manifest.SourceFiles, SourceFile{
			Path: relPath,
			Hash: fmt.Sprintf("sha256:%x", hash),
		})
	}
	sort.Slice(manifest.SourceFiles, func(i, j int) bool {
		return manifest.SourceFiles[i].Path < manifest.SourceFiles[j].Path
	})

	return manifest
}

// Generate creates a LockManifest from a compiler Output by hashing every
// generated file's content. It does not write to disk.
// Deprecated: Use GenerateWithOpts for new code.
func Generate(out *output.Output) *LockManifest {
	return GenerateWithOpts(out, GenerateOpts{})
}

// Write serializes the manifest to a YAML file at the given path.
// The parent directory must already exist.
func Write(manifest *LockManifest, path string) error {
	cleanPath := filepath.Clean(path)

	data, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal lock manifest: %w", err)
	}

	if err := os.WriteFile(cleanPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write lock file to %q: %w", cleanPath, err)
	}

	return nil
}

// Read deserializes a lock manifest from a YAML file.
func Read(path string) (*LockManifest, error) {
	cleanPath := filepath.Clean(path)

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read lock file %q: %w", cleanPath, err)
	}

	var manifest LockManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse lock file %q: %w", cleanPath, err)
	}

	return &manifest, nil
}

// SourcesChanged compares the previous lock's source file hashes against the
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

// MigrateLegacyLock renames a legacy scaffold.lock to scaffold.<target>.lock
// if the target-specific lock does not already exist. Returns true if migration
// was performed. Returns (false, nil) if no migration was needed.
func MigrateLegacyLock(legacyPath, target string) (bool, error) {
	legacyPath = filepath.Clean(legacyPath)

	if _, err := os.Stat(legacyPath); os.IsNotExist(err) {
		return false, nil
	}

	targetPath := LockFilePath(legacyPath, target)

	if _, err := os.Stat(targetPath); err == nil {
		return false, nil // target-specific lock already exists
	}

	if err := os.Rename(legacyPath, targetPath); err != nil {
		return false, fmt.Errorf("failed to migrate %q to %q: %w", legacyPath, targetPath, err)
	}

	return true, nil
}

// FindOrphans returns artifact paths that exist in the old manifest but not in
// the new compilation output. Results are sorted alphabetically.
func FindOrphans(old *LockManifest, newFiles map[string]string) []string {
	if old == nil {
		return nil
	}

	var orphans []string
	for _, artifact := range old.Artifacts {
		if _, exists := newFiles[artifact.Path]; !exists {
			orphans = append(orphans, artifact.Path)
		}
	}
	sort.Strings(orphans)
	return orphans
}
