package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// configFlag holds the value of the global --config flag.
// It is resolved before any subcommand runs.
var configFlag string

// xcfPath is the resolved, absolute path to the scaffold.xcf file.
// All subcommands should read from this rather than a hardcoded "scaffold.xcf".
var xcfPath string

// claudeDir is the resolved, absolute path to the .claude/ output directory.
var claudeDir string

// lockPath is the resolved, absolute path to scaffold.lock.
var lockPath string

var rootCmd = &cobra.Command{
	Use:   "xcaffold",
	Short: "xcaffold — deterministic agent-as-code orchestration",
	Long: `xcaffold is an open-source, deterministic agent configuration compiler engine for Claude Code.

┌───────────────────────────────────────────────────────────────────┐
│                 THE 6-PHASE ORCHESTRATION ENGINE                  │
└───────────────────────────────────────────────────────────────────┘
 • Bootstrap   [xcaffold init]    Creates base project scaffolding
 • Audit       [xcaffold analyze] Inspects repo & builds XCF config
 • Token Cost  [xcaffold plan]    Statically estimates token budget
 • Compilation [xcaffold apply]   Compiles XCF to .claude/ prompts
 • Drift Check [xcaffold diff]    Detects manual config tampering
 • Validation  [xcaffold test]    Runs an LLM-in-the-loop proxy

┌───────────────────────────────────────────────────────────────────┐
│                      DIAGNOSTICS & TELEMETRY                      │
└───────────────────────────────────────────────────────────────────┘
 • Review      [xcaffold review]  Universally parses state files
   ↳ Supports: scaffold.xcf, audit.json, plan.json, trace.jsonl
   ↳ Try: 'xcaffold review all'

Use 'xcaffold --help' for more information on available commands.`,
	PersistentPreRunE: resolveConfig,
}

func init() {
	rootCmd.PersistentFlags().StringVar(
		&configFlag,
		"config",
		"",
		"Path to scaffold.xcf (default: ./scaffold.xcf). Use for monorepo sub-directories.",
	)
}

// resolveConfig is a PersistentPreRunE hook that runs before every subcommand.
// It resolves the --config flag into a stable set of absolute paths that all
// subcommands can use without re-implementing CWD logic.
func resolveConfig(cmd *cobra.Command, args []string) error {
	// Commands that don't need a config file skip resolution.
	skipCommands := map[string]bool{
		"review": true,
	}
	if skipCommands[cmd.Name()] {
		return nil
	}

	var xcfAbs string
	if configFlag != "" {
		abs, err := filepath.Abs(configFlag)
		if err != nil {
			return fmt.Errorf("--config: could not resolve path %q: %w", configFlag, err)
		}
		xcfAbs = abs
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("could not determine working directory: %w", err)
		}
		xcfAbs = filepath.Join(cwd, "scaffold.xcf")
	}

	// Validate the file exists before any command runs.
	if _, err := os.Stat(xcfAbs); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("scaffold.xcf not found at %q\n\nHint: run 'xcaffold init' to create one, or use --config to specify a path", xcfAbs)
		}
		return fmt.Errorf("could not access %q: %w", xcfAbs, err)
	}

	baseDir := filepath.Dir(xcfAbs)
	xcfPath = xcfAbs
	claudeDir = filepath.Join(baseDir, ".claude")
	lockPath = filepath.Join(baseDir, "scaffold.lock")

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
