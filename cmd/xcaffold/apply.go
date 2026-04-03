package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/saero-ai/xcaffold/internal/compiler"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/state"
	"github.com/spf13/cobra"
	"strings"
)

var applyDryRun bool

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
	applyCmd.Flags().BoolVar(&applyDryRun, "dry-run", false, "Preview changes without writing to disk")
	rootCmd.AddCommand(applyCmd)
}

func runApply(cmd *cobra.Command, args []string) error {
	config, err := parser.ParseFile(xcfPath)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	out, err := compiler.Compile(config)
	if err != nil {
		return fmt.Errorf("compilation error: %w", err)
	}

	for _, agent := range config.Agents {
		if len(agent.Targets) > 0 {
			fmt.Fprintln(os.Stderr, "Warning: 'targets' block is experimental and currently uncompiled.")
			break
		}
	}

	if applyDryRun {
		fmt.Println("Dry-run preview (no files will be written):")
		fmt.Println()
	} else {
		// Pre-create all required subdirectories.
		for _, subdir := range []string{"agents", "skills", "rules"} {
			if err := os.MkdirAll(filepath.Join(claudeDir, subdir), 0755); err != nil {
				return fmt.Errorf("failed to create output directory %q: %w", subdir, err)
			}
		}
	}

	// Write (or preview) each compiled file.
	hasChanges := false
	for relPath, content := range out.Files {
		absPath := filepath.Clean(filepath.Join(claudeDir, relPath))

		if applyDryRun {
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
				hasChanges = true
				colorDiff(diff)
			}
			continue
		}

		if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %q: %w", absPath, err)
		}
		hash := sha256.Sum256([]byte(content))
		fmt.Printf("  ✓ wrote %s  (sha256:%x)\n", absPath, hash)
	}

	if applyDryRun {
		if !hasChanges {
			fmt.Println("✓ No changes predicted. Current files are up to date.")
		}
		return nil
	}

	// Write the lock file.
	manifest := state.Generate(out)
	if err := state.Write(manifest, lockPath); err != nil {
		return fmt.Errorf("failed to write scaffold.lock: %w", err)
	}

	fmt.Println("\n✓ Apply complete. scaffold.lock updated.")
	return nil
}

// colorDiff prints a unified diff with basic ANSI terminal colors.
func colorDiff(diff string) {
	lines := strings.Split(diff, "\n")
	for _, l := range lines {
		if strings.HasPrefix(l, "+++") || strings.HasPrefix(l, "---") {
			fmt.Printf("\033[1m%s\033[0m\n", l) // bold
		} else if strings.HasPrefix(l, "@@") {
			fmt.Printf("\033[36m%s\033[0m\n", l) // cyan
		} else if strings.HasPrefix(l, "+") {
			fmt.Printf("\033[32m%s\033[0m\n", l) // green
		} else if strings.HasPrefix(l, "-") {
			fmt.Printf("\033[31m%s\033[0m\n", l) // red
		} else {
			fmt.Println(l)
		}
	}
}
