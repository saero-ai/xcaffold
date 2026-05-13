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
	DependencyManifests map[string]string `json:"dependency_manifests"`
	ProviderConfig      string            `json:"provider_config,omitempty"`
	Files               []string          `json:"files"`
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

// processScanDir handles a directory entry during the project walk.
// It returns fs.SkipDir when the directory should not be descended into.
func processScanDir(path string, d fs.DirEntry, depth int, sig *ProjectSignature) error {
	base := d.Name()
	if ignoreDirs[base] {
		return fs.SkipDir
	}
	// Only walk exactly 1 directory deep to see top-level subfolders (src, cmd, pkg).
	if depth >= 2 {
		return fs.SkipDir
	}
	sig.Files = append(sig.Files, path+"/")
	return nil
}

// processScanFile handles a file entry during the project walk.
func processScanFile(fsys fs.FS, path string, d fs.DirEntry, sig *ProjectSignature) {
	sig.Files = append(sig.Files, path)
	base := d.Name()
	if targetManifests[base] || strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".yaml") {
		content := readTruncated(fsys, path, 5000)
		if content != "" {
			sig.DependencyManifests[path] = content
		}
	}
	if base == "CLAUDE.md" && sig.ProviderConfig == "" {
		// Only capture the first (shallowest) CLAUDE.md encountered.
		sig.ProviderConfig = readTruncated(fsys, path, 5000)
	}
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
			return processScanDir(path, d, depth, sig)
		}

		// Only record root files and files inside exactly 1 level of subdirectories
		// (e.g. .github/workflows/main.yml would be depth 3 and skipped).
		if depth > 2 {
			return nil
		}
		processScanFile(fsys, path, d, sig)
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
