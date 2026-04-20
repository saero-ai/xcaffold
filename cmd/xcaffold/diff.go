package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/saero-ai/xcaffold/internal/compiler"
	"github.com/saero-ai/xcaffold/internal/resolver"
	"github.com/saero-ai/xcaffold/internal/state"
	"github.com/spf13/cobra"
)

var diffTargetFlag string
var diffBlueprintFlag string

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Detect drift between state manifest and compilation targets on disk",
	Long: `xcaffold diff flags manual tampering and shadow-edits in your workspace.

┌───────────────────────────────────────────────────────────────────┐
│                          DRIFT CHECK PHASE                        │
└───────────────────────────────────────────────────────────────────┘
 • Recomputes SHA-256 hashes for all output files natively
 • Compares current file hashes against the state manifest truth state
 • Warns you if humans or external agents have mutated the generated files

Usage:
  $ xcaffold diff
  $ xcaffold diff --global
  $ xcaffold diff --target cursor
  $ xcaffold diff --target antigravity`,
	Example: "  $ xcaffold diff --target cursor",
	RunE:    runDiff,
}

func init() {
	diffCmd.Flags().StringVar(&diffTargetFlag, "target", "", "compilation target platform (claude, cursor, antigravity, copilot, gemini; default: claude)")
	diffCmd.Flags().StringVar(&diffBlueprintFlag, "blueprint", "", "Show drift for a specific blueprint (default: all resources)")
	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
	if diffBlueprintFlag != "" && globalFlag {
		return fmt.Errorf("--blueprint cannot be used with --global (blueprints are project-scoped)")
	}

	effectiveTarget := diffTargetFlag
	if effectiveTarget == "" {
		effectiveTarget = "claude"
	}

	if globalFlag {
		targetDir := filepath.Join(filepath.Dir(globalXcfHome), compiler.OutputDir(diffTargetFlag))
		stateFile := state.StateFilePath(filepath.Dir(globalXcfHome), diffBlueprintFlag)
		drift, err := diffScope(targetDir, stateFile, effectiveTarget, "global")
		if err != nil {
			return err
		}
		fmt.Println()
		if drift > 0 {
			return fmt.Errorf("drift detected in %d file(s) — run 'xcaffold apply --global --target %s' to restore managed state", drift, diffTargetFlag)
		}
		fmt.Println("No drift detected. All managed files are in sync.")
		return nil
	}

	targetDir := filepath.Join(filepath.Dir(claudeDir), compiler.OutputDir(diffTargetFlag))
	stateFile := state.StateFilePath(filepath.Dir(claudeDir), diffBlueprintFlag)
	drift, err := diffScope(targetDir, stateFile, effectiveTarget, "project")
	if err != nil {
		return err
	}
	fmt.Println()
	if drift > 0 {
		return fmt.Errorf("drift detected in %d file(s) — run 'xcaffold apply --target %s' to restore managed state", drift, diffTargetFlag)
	}
	fmt.Println("No drift detected. All managed files are in sync.")
	return nil
}

// diffScope reads the state file at stateFile and compares each artifact's
// recorded SHA-256 hash for the given target against the file on disk inside
// outputDir. scopeName is used as a prefix in output lines so the user can
// distinguish global from project passes.
func diffScope(outputDir, stateFile, target, scopeName string) (int, error) {
	manifest, err := state.ReadState(stateFile)
	if err != nil {
		hint := "xcaffold apply"
		if scopeName == "global" {
			hint = "xcaffold apply --global"
		}
		return 0, fmt.Errorf("[%s] could not read state file: %w\n\nHint: run '%s' first", scopeName, err, hint)
	}

	ts, ok := manifest.Targets[target]
	if !ok {
		return 0, fmt.Errorf("[%s] no state found for target %q", scopeName, target)
	}

	driftCount := 0
	for _, artifact := range ts.Artifacts {
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

	if len(manifest.SourceFiles) > 0 {
		// State files live at <baseDir>/.xcaffold/<name>.xcf.state, so strip two
		// path components to recover the project root.
		baseDir := filepath.Dir(filepath.Dir(stateFile))
		currentSources, err := resolver.FindXCFFiles(baseDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[%s] Warning: failed to scan source files: %v\n", scopeName, err)
			return driftCount, nil
		}

		prevByPath := make(map[string]string)
		for _, sf := range manifest.SourceFiles {
			prevByPath[sf.Path] = sf.Hash
		}

		currByPath := make(map[string]string)
		for _, absPath := range currentSources {
			rel, err := filepath.Rel(baseDir, absPath)
			if err != nil {
				continue
			}
			data, err := os.ReadFile(absPath)
			if err == nil {
				hash := sha256.Sum256(data)
				currByPath[rel] = fmt.Sprintf("sha256:%x", hash)
			}
		}

		for _, sf := range manifest.SourceFiles {
			if currHash, exists := currByPath[sf.Path]; !exists {
				fmt.Printf("  [%s] SRC DELETED %s\n", scopeName, sf.Path)
				driftCount++
			} else if currHash != sf.Hash {
				fmt.Printf("  [%s] SRC DRIFTED %s\n", scopeName, sf.Path)
				fmt.Printf("    expected: %s\n", sf.Hash)
				fmt.Printf("    actual:   %s\n", currHash)
				driftCount++
			}
		}
		for rel := range currByPath {
			if _, exists := prevByPath[rel]; !exists {
				fmt.Printf("  [%s] SRC ADDED   %s\n", scopeName, rel)
				driftCount++
			}
		}
	}

	return driftCount, nil
}
