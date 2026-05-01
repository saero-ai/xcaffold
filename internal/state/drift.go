package state

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

// DriftEntry describes a single artifact that has drifted from its recorded state.
type DriftEntry struct {
	Status string // "missing" or "modified"
	Path   string // artifact path as stored (may have "root:" prefix)
}

// CollectDriftedFiles compares artifacts recorded in ts against files on disk.
// baseDir is the project root (for "root:"-prefixed paths).
// outputDir is the provider output directory (for all other paths).
func CollectDriftedFiles(baseDir, outputDir string, ts TargetState) []DriftEntry {
	var entries []DriftEntry
	for _, artifact := range ts.Artifacts {
		var absPath string
		if strings.HasPrefix(artifact.Path, "root:") {
			relPath := strings.TrimPrefix(artifact.Path, "root:")
			absPath = filepath.Clean(filepath.Join(baseDir, relPath))
		} else {
			absPath = filepath.Clean(filepath.Join(outputDir, artifact.Path))
		}

		data, err := os.ReadFile(absPath)
		if err != nil {
			entries = append(entries, DriftEntry{"missing", artifact.Path})
			continue
		}
		h := sha256.Sum256(data)
		if "sha256:"+hex.EncodeToString(h[:]) != artifact.Hash {
			entries = append(entries, DriftEntry{"modified", artifact.Path})
		}
	}
	return entries
}
