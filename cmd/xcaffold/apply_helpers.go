package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/saero-ai/xcaffold/internal/state"
)

// performBackup creates a timestamped backup of outputDir.
func performBackup(outputDir, target, backupDirConfig, scopeName string) error {
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		return nil // nothing to backup
	}

	timestamp := "20060102_150405"
	if target == "" {
		target = "unknown"
	}
	bakName := fmt.Sprintf(".%s_bak_%s", target, timestamp)

	var destDir string
	if backupDirConfig != "" {
		destDir = filepath.Join(backupDirConfig, bakName)
	} else {
		destDir = filepath.Join(filepath.Dir(outputDir), bakName)
	}

	if err := os.MkdirAll(filepath.Dir(destDir), 0755); err != nil {
		return err
	}

	fmt.Printf("  %s  Backed up %s\n", colorGreen(glyphOK()), filepath.Base(destDir))
	return copyDir(outputDir, destDir)
}

// copyDir recursively copies src directory to dst.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destPath, data, info.Mode())
	})
}

// hasDriftFromState checks if files in outputDir differ from what state records.
func hasDriftFromState(outputDir, stateFile, baseDir, target string) ([]state.DriftEntry, error) {
	manifest, err := state.ReadState(stateFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	ts, ok := manifest.Targets[target]
	if !ok {
		return nil, nil
	}

	return state.CollectDriftedFiles(baseDir, outputDir, ts), nil
}

// ensureGitignoreEntry appends entry to dir/.gitignore if not already present.
// Creates the file if it does not exist.
func ensureGitignoreEntry(dir, entry string) {
	gitignorePath := filepath.Join(dir, ".gitignore")
	data, _ := os.ReadFile(gitignorePath)
	if strings.Contains(string(data), entry) {
		return
	}
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	if len(data) > 0 && data[len(data)-1] != '\n' {
		_, _ = f.WriteString("\n")
	}
	_, _ = f.WriteString(entry + "\n")
}

// crossScopeOpts holds parameters for cleanCrossScope to reduce function arity.
type crossScopeOpts struct {
	baseDir          string
	outputDir        string
	currentStatePath string
	target           string
	force            bool
}

// cleanCrossScope removes artifacts from other scopes when switching blueprints.
// Searches all state files in baseDir/.xcaffold/ for artifacts of the current target.
// If drift is found and !force, returns an error. If no drift or force, deletes orphans.
func cleanCrossScope(opts crossScopeOpts) error {
	baseDir := opts.baseDir
	outputDir := opts.outputDir
	currentStatePath := opts.currentStatePath
	target := opts.target
	force := opts.force
	stateDir := state.StateDir(baseDir)
	stateFiles, err := state.ListStateFiles(stateDir)
	if err != nil {
		return err
	}

	// Skip the current scope's state file
	currentStatePath = filepath.Clean(currentStatePath)
	for _, otherPath := range stateFiles {
		otherPath = filepath.Clean(otherPath)
		if otherPath == currentStatePath {
			continue // Skip current scope
		}

		// Read the other scope's manifest
		otherManifest, readErr := state.ReadState(otherPath)
		if readErr != nil {
			continue // Skip if can't read
		}

		// Check if this scope has artifacts for our target
		targetState, ok := otherManifest.Targets[target]
		if !ok {
			continue // This scope doesn't have our target
		}

		// Detect drift in the other scope's artifacts
		driftEntries := state.CollectDriftedFiles(baseDir, outputDir, targetState)
		if len(driftEntries) > 0 && !force {
			return fmt.Errorf("cannot switch scope: %d files were manually edited since last apply\n\nRun 'xcaffold import' to sync edits, or use --backup --force to overwrite", len(driftEntries))
		}

		// Delete the other scope's artifacts for this target
		for _, artifact := range targetState.Artifacts {
			var absPath string
			if strings.HasPrefix(artifact.Path, "root:") {
				relPath := strings.TrimPrefix(artifact.Path, "root:")
				absPath = filepath.Clean(filepath.Join(baseDir, relPath))
			} else {
				absPath = filepath.Clean(filepath.Join(outputDir, artifact.Path))
			}
			_ = os.Remove(absPath)
			cleanEmptyDirsUpToTarget(filepath.Dir(absPath), outputDir)
		}

		// Delete the other scope's state file
		_ = os.Remove(otherPath)
	}

	return nil
}

// cleanEmptyDirsUpToTarget recursively deletes empty parent directories
// up to but not including the targetDir itself.
func cleanEmptyDirsUpToTarget(dir, targetDir string) {
	dir = filepath.Clean(dir)
	targetDir = filepath.Clean(targetDir)

	for dir != targetDir && dir != "." && dir != "/" {
		rel, err := filepath.Rel(targetDir, dir)
		if err != nil || strings.HasPrefix(rel, "..") || rel == "." {
			break
		}

		if err := os.Remove(dir); err != nil {
			break // Dir not empty, or permission error
		}
		dir = filepath.Dir(dir)
	}
}

// colorDiff prints diff with ANSI colors.
func colorDiff(diff string) {
	lines := strings.Split(diff, "\n")
	for _, l := range lines {
		switch {
		case strings.HasPrefix(l, "+"):
			fmt.Println("\033[32m" + l + "\033[0m")
		case strings.HasPrefix(l, "-"):
			fmt.Println("\033[31m" + l + "\033[0m")
		case strings.HasPrefix(l, "@"):
			fmt.Println("\033[36m" + l + "\033[0m")
		default:
			fmt.Println(l)
		}
	}
}

// previewDiff shows the unified diff between existing and new content.
func previewDiff(absPath, content string) bool {
	existingData, err := os.ReadFile(absPath)
	existing := ""
	if err == nil {
		existing = string(existingData)
	}
	diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(existing),
		B:        difflib.SplitLines(content),
		FromFile: absPath + " (current)",
		ToFile:   absPath + " (compiled)",
		Context:  3,
	})
	if diff != "" {
		colorDiff(diff)
		return true
	}
	return false
}

// applyFileResult holds the output parameters for applyFile.
type applyFileResult struct {
	hasChanges   bool
	filesWritten int
}

// applyFile writes content to absPath, or previews if dry-run.
func applyFile(absPath, content, scopeName string, result *applyFileResult) error {
	if applyDryRun {
		if previewDiff(absPath, content) {
			result.hasChanges = true
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %q: %w", absPath, err)
	}
	if err := os.WriteFile(absPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write %q: %w", absPath, err)
	}
	result.filesWritten++
	return nil
}

// computeApplyPreview builds a list of diff entries for display.
func computeApplyPreview(outFiles map[string]string, rootFiles map[string]string, outputDir, baseDir string) []applyDiffEntry {
	var entries []applyDiffEntry
	for relPath, content := range outFiles {
		absPath := filepath.Clean(filepath.Join(outputDir, relPath))
		existing, err := os.ReadFile(absPath)
		if err != nil {
			entries = append(entries, applyDiffEntry{Path: relPath, Status: "new"})
		} else if string(existing) != content {
			entries = append(entries, applyDiffEntry{Path: relPath, Status: "changed"})
		} else {
			entries = append(entries, applyDiffEntry{Path: relPath, Status: "unchanged"})
		}
	}
	for relPath, content := range rootFiles {
		absPath := filepath.Clean(filepath.Join(baseDir, relPath))
		existing, err := os.ReadFile(absPath)
		if err != nil {
			entries = append(entries, applyDiffEntry{Path: relPath + "  (root)", Status: "new"})
		} else if string(existing) != content {
			entries = append(entries, applyDiffEntry{Path: relPath + "  (root)", Status: "changed"})
		} else {
			entries = append(entries, applyDiffEntry{Path: relPath + "  (root)", Status: "unchanged"})
		}
	}
	return entries
}

// renderApplyPreview prints the preview table and returns counts of each status.
// Note: deletedCount is not returned because the preview does not track deletions
// (they are handled separately by ctx.cleanOrphans).
func renderApplyPreview(entries []applyDiffEntry) (newCount, changedCount, unchangedCount int) {
	for _, e := range entries {
		switch e.Status {
		case "new":
			newCount++
		case "changed":
			changedCount++
		case "unchanged":
			unchangedCount++
		}
	}
	if changedCount > 0 {
		fmt.Printf("\n  CHANGED (%d %s):\n", changedCount, plural(changedCount, "file", "files"))
		for _, e := range entries {
			if e.Status == "changed" {
				display, _ := formatArtifactPath(e.Path)
				fmt.Printf("    %s  %s\n", colorYellow(glyphSrc()), display)
			}
		}
	}
	if newCount > 0 {
		fmt.Printf("\n  NEW (%d %s):\n", newCount, plural(newCount, "file", "files"))
		for _, e := range entries {
			if e.Status == "new" {
				display, _ := formatArtifactPath(e.Path)
				fmt.Printf("    %s  %s\n", colorGreen("+"), display)
			}
		}
	}
	if unchangedCount > 0 {
		fmt.Printf("\n  UNCHANGED: %d %s\n", unchangedCount, plural(unchangedCount, "file", "files"))
	}
	return
}

// countOrphansFromState counts the number of files tracked in oldManifest
// for the given target that are absent from the new compiled output.
func countOrphansFromState(oldManifest *state.StateManifest, target string, outFiles map[string]string) int {
	if oldManifest == nil {
		return 0
	}
	count := 0
	targetState, ok := oldManifest.Targets[target]
	if !ok {
		return 0
	}
	for _, artifact := range targetState.Artifacts {
		if _, inOut := outFiles[artifact.Path]; !inOut {
			count++
		}
	}
	return count
}

// cleanOrphans removes files from outputDir that were recorded in old state
// for the given target but are absent from the new compiler output.
func (ctx *applyContext) cleanOrphans(hasChanges *bool) {
	orphans := state.FindOrphansFromState(ctx.oldManifest, targetFlag, ctx.out.Files, ctx.out.RootFiles)
	for _, orphanPath := range orphans {
		var absPath string
		if strings.HasPrefix(orphanPath, "root:") {
			relPath := strings.TrimPrefix(orphanPath, "root:")
			absPath = filepath.Clean(filepath.Join(ctx.baseDir, relPath))
		} else {
			absPath = filepath.Clean(filepath.Join(ctx.outputDir, orphanPath))
		}
		if applyDryRun {
			fmt.Printf("    %s  would delete  %s\n", colorYellow(glyphSrc()), filepath.Base(absPath))
			*hasChanges = true
		} else {
			if err := os.Remove(absPath); err == nil {
				fmt.Printf("    %s  deleted  %s\n", colorRed(glyphErr()), filepath.Base(absPath))
				*hasChanges = true
				cleanEmptyDirsUpToTarget(filepath.Dir(absPath), ctx.outputDir)
			} else if os.IsNotExist(err) {
				*hasChanges = true
			}
		}
	}
}
