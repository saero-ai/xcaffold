package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/saero-ai/xcaffold/internal/state"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Detect drift between scaffold.lock and .claude/ files on disk",
	Long: `xcaffold diff flags manual tampering and shadow-edits in your workspace.

┌───────────────────────────────────────────────────────────────────┐
│                          DRIFT CHECK PHASE                        │
└───────────────────────────────────────────────────────────────────┘
 • Recomputes SHA-256 hashes for all '.claude/' files natively
 • Compares current file hashes against the scaffold.lock truth state
 • Warns you if humans or external agents have mutated the system prompts

Usage:
  $ xcaffold diff
  $ xcaffold diff --scope global
  $ xcaffold diff --scope all`,
	Example: "  $ xcaffold diff",
	RunE:    runDiff,
}

func init() {
	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
	totalDrift := 0

	if scopeFlag == "global" || scopeFlag == "all" {
		drift, err := diffScope(globalClaudeDir, globalLockPath, "global")
		if err != nil {
			return err
		}
		totalDrift += drift
	}
	if scopeFlag == "project" || scopeFlag == "all" {
		drift, err := diffScope(claudeDir, lockPath, "project")
		if err != nil {
			return err
		}
		totalDrift += drift
	}

	fmt.Println()
	if totalDrift > 0 {
		return fmt.Errorf("drift detected in %d file(s) — run 'xcaffold apply' to restore managed state", totalDrift)
	}
	fmt.Println("No drift detected. All managed files are in sync.")
	return nil
}

// diffScope reads the lock file at lockFile and compares each artifact's
// recorded SHA-256 hash against the file on disk inside outputDir.
// scopeName is used as a prefix in output lines when running --scope all
// so the user can distinguish the two passes.
func diffScope(outputDir, lockFile, scopeName string) (int, error) {
	manifest, err := state.Read(lockFile)
	if err != nil {
		return 0, fmt.Errorf("[%s] could not read lock file: %w\n\nHint: run 'xcaffold apply --scope %s' first", scopeName, err, scopeName)
	}

	driftCount := 0
	for _, artifact := range manifest.Artifacts {
		absPath := filepath.Clean(filepath.Join(outputDir, artifact.Path))

		data, err := os.ReadFile(absPath)
		if err != nil {
			fmt.Printf("  [%s] MISSING  %s\n", scopeName, absPath)
			driftCount++
			continue
		}

		actualHash := sha256.Sum256(data)
		actual := fmt.Sprintf("sha256:%x", actualHash)

		if actual != artifact.Hash {
			fmt.Printf("  [%s] DRIFTED  %s\n", scopeName, absPath)
			fmt.Printf("    expected: %s\n", artifact.Hash)
			fmt.Printf("    actual:   %s\n", actual)
			driftCount++
		} else {
			fmt.Printf("  [%s] clean    %s\n", scopeName, absPath)
		}
	}

	return driftCount, nil
}
