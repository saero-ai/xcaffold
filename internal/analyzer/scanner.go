package analyzer

import (
	"encoding/json"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
)

// ProjectSignature is the highly compressed deterministic summary of the repository.
type ProjectSignature struct {
	Files               []string          `json:"files"`
	DependencyManifests map[string]string `json:"dependency_manifests"`
	ClaudeConfig        string            `json:"claude_config,omitempty"`
}

// targetManifests specifies the files we want to read fully to determine project context.
var targetManifests = map[string]bool{
	"package.json":       true,
	"go.mod":             true,
	"Cargo.toml":         true,
	"requirements.txt":   true,
	"pyproject.toml":     true,
	"Makefile":           true,
	"docker-compose.yml": true,
	".gitlab-ci.yml":     true,
}

// ignoreDirs specifies directories we should never walk into.
var ignoreDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	".next":        true,
	".vercel":      true,
}

// ScanProject walks the provided filesystem (usually the repository root)
// and extracts a dense, deterministic signature of the project architecture
// without reading source code logic.
func ScanProject(fsys fs.FS) (*ProjectSignature, error) {
	sig := &ProjectSignature{
		Files:               make([]string, 0),
		DependencyManifests: make(map[string]string),
	}

	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == "." {
			return nil
		}

		// Calculate depth. E.g., "src" is depth 1, "src/main.go" is depth 2.
		depth := strings.Count(filepath.ToSlash(path), "/") + 1

		if d.IsDir() {
			base := d.Name()
			// Fast prune ignored directories
			if ignoreDirs[base] {
				return fs.SkipDir
			}
			// Only walk exactly 1 directory deep to see top-level subfolders (src, cmd, pkg).
			// We skip drilling into the subfolders themselves.
			if depth >= 2 {
				return fs.SkipDir
			}
			sig.Files = append(sig.Files, path+"/")
			return nil
		}

		// Only record root files and files inside exactly 1 level of subdirectories
		// (e.g. .github/workflows/main.yml would be depth 3 and skipped)
		if depth > 2 {
			return nil
		}

		// If it's a file, add to the structure array
		sig.Files = append(sig.Files, path)

		base := d.Name()
		if targetManifests[base] || strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".yaml") {
			// Read constraints: grab manifest files or root-level YAMLs (CI config)
			content := readTruncated(fsys, path, 5000)
			if content != "" {
				sig.DependencyManifests[path] = content
			}
		}

		if base == "CLAUDE.md" {
			// Only capture the first (shallowest) CLAUDE.md encountered.
			// The walker visits root before subdirectories, so this preserves
			// the project-root file and ignores any src/CLAUDE.md etc. (Bug 13)
			if sig.ClaudeConfig == "" {
				sig.ClaudeConfig = readTruncated(fsys, path, 5000)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return sig, nil
}

// String provides a JSON representation of the signature for passing to the LLM.
func (s *ProjectSignature) String() string {
	b, _ := json.MarshalIndent(s, "", "  ")
	return string(b)
}

func readTruncated(fsys fs.FS, path string, maxBytes int) string {
	f, err := fsys.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	// Read up to maxBytes
	buf := make([]byte, maxBytes)
	n, err := io.ReadFull(f, buf)
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return string(buf[:n])
	}
	if err == nil {
		return string(buf) + "\n... (truncated)"
	}
	return ""
}
