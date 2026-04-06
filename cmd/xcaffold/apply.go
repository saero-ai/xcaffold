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
var targetFlag string

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
	applyCmd.Flags().StringVar(&targetFlag, "target", "", "compilation target platform (claude, cursor, gemini; default: claude)")
	rootCmd.AddCommand(applyCmd)
}

func runApply(cmd *cobra.Command, args []string) error {
	if scopeFlag == "global" || scopeFlag == "all" {
		if err := applyScope(globalXcfPath, globalClaudeDir, globalLockPath, "global"); err != nil {
			return err
		}
	}
	if scopeFlag == "project" || scopeFlag == "all" {
		if err := applyScope(xcfPath, claudeDir, lockPath, "project"); err != nil {
			return err
		}
	}
	return nil
}

// applyScope compiles a single xcf file into outputDir and writes the lock file
// at lockFile. scopeName is used as a prefix in terminal output when running
// --scope all so the user can distinguish the two compilation passes.
func applyScope(configPath, outputDir, lockFile, scopeName string) error {
	config, err := parser.ParseFile(configPath)
	if err != nil {
		return fmt.Errorf("[%s] parse error: %w", scopeName, err)
	}

	// baseDir is the directory containing the xcf file — used by the compiler
	// to resolve instructions_file: and references: paths.
	baseDir := filepath.Dir(configPath)
	out, err := compiler.Compile(config, baseDir, targetFlag)
	if err != nil {
		return fmt.Errorf("[%s] compilation error: %w", scopeName, err)
	}

	// Resolve the target-specific output directory instead of the hardcoded default
	outputDir = filepath.Join(filepath.Dir(outputDir), compiler.OutputDir(targetFlag))

	for _, agent := range config.Agents {
		if len(agent.Targets) > 0 {
			fmt.Fprintf(os.Stderr, "[%s] Warning: 'targets' block is experimental and currently uncompiled.\n", scopeName)
			break
		}
	}

	if applyDryRun {
		fmt.Printf("[%s] Dry-run preview (no files will be written):\n\n", scopeName)
	} else if targetFlag == "" || targetFlag == "claude" {
		// Pre-create baseline subdirectories exclusively for the Claude format contract.
		for _, subdir := range []string{"agents", "skills", "rules"} {
			if err := os.MkdirAll(filepath.Join(outputDir, subdir), 0755); err != nil {
				return fmt.Errorf("[%s] failed to create output directory %q: %w", scopeName, subdir, err)
			}
		}
	}

	// Write (or preview) each compiled file.
	hasChanges := false
	for relPath, content := range out.Files {
		absPath := filepath.Clean(filepath.Join(outputDir, relPath))

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

		if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
			return fmt.Errorf("[%s] failed to create directory for %q: %w", scopeName, absPath, err)
		}
		if err := os.WriteFile(absPath, []byte(content), 0600); err != nil {
			return fmt.Errorf("[%s] failed to write %q: %w", scopeName, absPath, err)
		}
		hash := sha256.Sum256([]byte(content))
		fmt.Printf("  [%s] ✓ wrote %s  (sha256:%x)\n", scopeName, absPath, hash)
	}

	if applyDryRun {
		if !hasChanges {
			fmt.Printf("[%s] ✓ No changes predicted. Current files are up to date.\n", scopeName)
		}
		return nil
	}

	// Write the lock file.
	manifest := state.Generate(out)
	targetLockFile := state.LockFilePath(lockFile, targetFlag)
	if err := state.Write(manifest, targetLockFile); err != nil {
		return fmt.Errorf("[%s] failed to write %s: %w", scopeName, filepath.Base(targetLockFile), err)
	}

	fmt.Printf("\n[%s] ✓ Apply complete. %s updated.\n", scopeName, filepath.Base(targetLockFile))
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
