package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/saero-ai/xcaffold/internal/compiler"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/state"
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Compile scaffold.xcf into .claude/ agent files",
	Long: `xcaffold apply deterministically compiles your YAML logic into native Claude Code markdown objects.

┌───────────────────────────────────────────────────────────────────┐
│                          COMPILATION PHASE                        │
└───────────────────────────────────────────────────────────────────┘
 [scaffold.xcf] ──(Compiles)──▶ [.claude/agents/*.md]
       │
   (Locks)──▶ [scaffold.lock]

 • Strict one-way generation (YAML -> MD)
 • Generates a cryptographic SHA-256 state manifest (scaffold.lock)
 • Automatically purges orphaned agents from .claude/ directory

Any manually edited files inside .claude/ will be overwritten.`,
	Example: "  $ xcaffold apply",
	RunE:    runApply,
}

func init() {
	rootCmd.AddCommand(applyCmd)
}

func runApply(cmd *cobra.Command, args []string) error {
	xcfPath := filepath.Clean("scaffold.xcf")

	f, err := os.Open(xcfPath)
	if err != nil {
		return fmt.Errorf("could not open %s: %w", xcfPath, err)
	}
	defer f.Close()

	config, err := parser.Parse(f)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	out, err := compiler.Compile(config)
	if err != nil {
		return fmt.Errorf("compilation error: %w", err)
	}

	// Ensure the output directory exists.
	agentsDir := filepath.Clean(".claude/agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %q: %w", agentsDir, err)
	}

	// Write each compiled file.
	for relPath, content := range out.Files {
		absPath := filepath.Clean(filepath.Join(".claude", relPath))
		if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %q: %w", absPath, err)
		}
		hash := sha256.Sum256([]byte(content))
		fmt.Printf("  ✓ wrote %s  (sha256:%x)\n", absPath, hash)
	}

	// Write the lock file.
	manifest := state.Generate(out)
	if err := state.Write(manifest, "scaffold.lock"); err != nil {
		return fmt.Errorf("failed to write scaffold.lock: %w", err)
	}

	fmt.Println("\n✓ Apply complete. scaffold.lock updated.")
	return nil
}
