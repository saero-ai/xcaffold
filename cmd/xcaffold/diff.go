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
  $ xcaffold diff`,
	Example: "  $ xcaffold diff",
	RunE:    runDiff,
}

func init() {
	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
	manifest, err := state.Read(lockPath)
	if err != nil {
		return fmt.Errorf("could not read scaffold.lock: %w\n\nHint: run 'xcaffold apply' first", err)
	}

	driftCount := 0
	for _, artifact := range manifest.Artifacts {
		absPath := filepath.Clean(filepath.Join(claudeDir, artifact.Path))

		data, err := os.ReadFile(absPath)
		if err != nil {
			fmt.Printf("  ✗ MISSING  %s\n", absPath)
			driftCount++
			continue
		}

		actualHash := sha256.Sum256(data)
		actual := fmt.Sprintf("sha256:%x", actualHash)

		if actual != artifact.Hash {
			fmt.Printf("  ✗ DRIFTED  %s\n", absPath)
			fmt.Printf("    expected: %s\n", artifact.Hash)
			fmt.Printf("    actual:   %s\n", actual)
			driftCount++
		} else {
			fmt.Printf("  ✓ clean    %s\n", absPath)
		}
	}

	fmt.Println()
	if driftCount > 0 {
		return fmt.Errorf("drift detected in %d file(s) — run 'xcaffold apply' to restore managed state", driftCount)
	}

	fmt.Println("✓ No drift detected. All managed files are in sync.")
	return nil
}
