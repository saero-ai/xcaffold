package state

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/saero-ai/xcaffold/internal/compiler"
	"gopkg.in/yaml.v3"
)

const lockFileVersion = 1

// LockFilePath returns the lock file path for a given target.
// For "claude" or empty string (default), returns basePath unchanged for backward compatibility.
// For other targets, inserts the target name before the extension:
// e.g., "scaffold.lock" + "cursor" → "scaffold.cursor.lock"
func LockFilePath(basePath string, target string) string {
	if target == "" || target == "claude" {
		return basePath
	}
	ext := filepath.Ext(basePath)
	base := strings.TrimSuffix(basePath, ext)
	return base + "." + target + ext
}

var XcaffoldVersion = "1.0.0-dev"

const claudeSchemaVersion = "alpha"

// LockManifest is the schema for the scaffold.lock file.
type LockManifest struct {
	Version             int        `yaml:"version"`
	LastApplied         string     `yaml:"last_applied"`
	XcaffoldVersion     string     `yaml:"xcaffold_version"`
	ClaudeSchemaVersion string     `yaml:"claude_schema_version"`
	Artifacts           []Artifact `yaml:"artifacts"`
}

// Artifact tracks a single generated file and its SHA-256 content hash.
type Artifact struct {
	Path string `yaml:"path"`
	Hash string `yaml:"hash"`
}

// Generate creates a LockManifest from a compiler Output by hashing every
// generated file's content. It does not write to disk.
func Generate(out *compiler.Output) *LockManifest {
	manifest := &LockManifest{
		Version:             lockFileVersion,
		LastApplied:         time.Now().UTC().Format(time.RFC3339),
		XcaffoldVersion:     XcaffoldVersion,
		ClaudeSchemaVersion: claudeSchemaVersion,
		Artifacts:           make([]Artifact, 0, len(out.Files)),
	}

	for path, content := range out.Files {
		hash := sha256.Sum256([]byte(content))
		manifest.Artifacts = append(manifest.Artifacts, Artifact{
			Path: path,
			Hash: fmt.Sprintf("sha256:%x", hash),
		})
	}

	// Sort artifacts by path to ensure deterministic lock file output.
	// Go map iteration order is non-deterministic, so without sorting,
	// identical configs produce scaffold.lock files that differ on each run.
	sort.Slice(manifest.Artifacts, func(i, j int) bool {
		return manifest.Artifacts[i].Path < manifest.Artifacts[j].Path
	})

	return manifest
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
