package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/saero-ai/xcaffold/internal/compiler"
	"github.com/saero-ai/xcaffold/internal/state"
)

func runStatusWithWriter(dir, profileName string, w io.Writer) error {
	statePath := state.StateFilePath(dir, profileName)
	manifest, err := state.ReadState(statePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintln(w, "No state found. Run 'xcaffold apply' to compile.")
			return nil
		}
		return fmt.Errorf("could not read state: %w", err)
	}

	if manifest.Blueprint != "" {
		fmt.Fprintf(w, "Blueprint: %s\n", manifest.Blueprint)
	}

	targetNames := sortedTargetKeys(manifest.Targets)
	for _, name := range targetNames {
		ts := manifest.Targets[name]
		driftCount := countDriftedArtifacts(dir, name, ts)
		elapsed := formatElapsed(ts.LastApplied)

		artifactCount := len(ts.Artifacts)
		noun := "artifact"
		if artifactCount != 1 {
			noun = "artifacts"
		}

		statusMsg := "all clean"
		if driftCount > 0 {
			statusMsg = fmt.Sprintf("%d drifted", driftCount)
		}

		fmt.Fprintf(w, "  %s: applied %s, %d %s (%s)\n",
			name, elapsed, artifactCount, noun, statusMsg)
	}

	if len(manifest.SourceFiles) > 0 {
		fmt.Fprintf(w, "\nSources: %d files\n", len(manifest.SourceFiles))
		for _, sf := range manifest.SourceFiles {
			absPath := filepath.Join(dir, sf.Path)
			data, err := os.ReadFile(absPath)
			if err != nil {
				fmt.Fprintf(w, "  missing  %s\n", sf.Path)
				continue
			}
			h := sha256.Sum256(data)
			actual := fmt.Sprintf("sha256:%x", h)
			if actual != sf.Hash {
				fmt.Fprintf(w, "  changed  %s  <- re-apply needed\n", sf.Path)
			}
		}
	}

	return nil
}

func countDriftedArtifacts(baseDir, targetName string, ts state.TargetState) int {
	outputDir := filepath.Join(baseDir, compiler.OutputDir(targetName))
	count := 0
	for _, artifact := range ts.Artifacts {
		absPath := filepath.Clean(filepath.Join(outputDir, artifact.Path))
		data, err := os.ReadFile(absPath)
		if err != nil {
			count++
			continue
		}
		h := sha256.Sum256(data)
		if fmt.Sprintf("sha256:%x", h) != artifact.Hash {
			count++
		}
	}
	return count
}

func formatElapsed(lastApplied string) string {
	t, err := time.Parse(time.RFC3339, lastApplied)
	if err != nil {
		return "unknown"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%d minutes ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d hours ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%d days ago", int(d.Hours()/24))
	}
}

func sortedTargetKeys(m map[string]state.TargetState) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
