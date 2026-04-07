package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/saero-ai/xcaffold/internal/state"
	"github.com/spf13/cobra"
)

var (
	version = "1.0.0-dev"
	commit  = "none"
	date    = "unknown"
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

// scopeFlag holds the value of the global --scope flag.
var scopeFlag string

// globalXcfPath is the resolved path to global.xcf.
var globalXcfPath string

// globalClaudeDir is ~/.claude/ for global scope output.
var globalClaudeDir string

// globalLockPath is ~/.claude/scaffold.lock.
var globalLockPath string

var rootCmd = &cobra.Command{
	Use:   "xcaffold",
	Short: "xcaffold — deterministic agent-as-code orchestration",
	Long: `xcaffold is an open-source, deterministic agent configuration compiler engine for Claude Code.

┌───────────────────────────────────────────────────────────────────┐
│                 THE 8-PHASE ORCHESTRATION ENGINE                  │
└───────────────────────────────────────────────────────────────────┘
 • Bootstrap   [xcaffold init]      Creates base project scaffolding
 • Ingestion   [xcaffold import]    Migrates existing .claude/ states
 • Translation [xcaffold translate] Imports & decomposes cross-platform workflows
 • Audit       [xcaffold analyze]   Inspects repo & builds XCF config
 • Token Cost  [xcaffold plan]      Statically estimates token budget
 • Topology    [xcaffold graph]     Visualizes agent network maps
 • Compilation [xcaffold apply]     Compiles XCF to .claude/ prompts
 • Drift Check [xcaffold diff]      Detects manual config tampering
 • Validation  [xcaffold test]      Runs an LLM-in-the-loop proxy
 • Export      [xcaffold export]    Packages output as a distributable plugin

┌───────────────────────────────────────────────────────────────────┐
│                      DIAGNOSTICS & TELEMETRY                      │
└───────────────────────────────────────────────────────────────────┘
 • Review      [xcaffold review]  Universally parses state files
   ↳ Supports: scaffold.xcf, audit.json, plan.json, trace.jsonl
   ↳ Try: 'xcaffold review all'

┌───────────────────────────────────────────────────────────────────┐
│                           SCOPES                                  │
└───────────────────────────────────────────────────────────────────┘
 • Project  [default]         scaffold.xcf  -> .claude/
 • Global   [--scope global]  global.xcf    -> ~/.claude/
 • Both     [--scope all]     Compiles both scopes

Use 'xcaffold --help' for more information on available commands.`,
	PersistentPreRunE: resolveConfig,
}

func init() {
	state.XcaffoldVersion = version
	rootCmd.Version = fmt.Sprintf("%s (commit: %s, date: %s)", version, commit, date)

	rootCmd.AddCommand(translateCmd)

	rootCmd.PersistentFlags().StringVar(
		&configFlag,
		"config",
		"",
		"Path to scaffold.xcf (default: ./scaffold.xcf). Use for monorepo sub-directories.",
	)
	rootCmd.PersistentFlags().StringVar(
		&scopeFlag,
		"scope",
		"project",
		"Compilation scope: project (default), global, or all",
	)
}

// resolveConfig is a PersistentPreRunE hook that runs before every subcommand.
// It resolves the --config and --scope flags into a stable set of absolute paths
// that all subcommands can use without re-implementing CWD logic.
func resolveConfig(cmd *cobra.Command, args []string) error {
	if cmd.Name() == "review" {
		return nil
	}

	if scopeFlag == scopeGlobal || scopeFlag == scopeAll {
		if err := resolveGlobalConfig(cmd); err != nil {
			return err
		}
		if scopeFlag == scopeGlobal {
			return nil
		}
	}

	if scopeFlag == scopeProject || scopeFlag == scopeAll {
		if err := resolveProjectConfig(cmd); err != nil {
			return err
		}
	}

	return nil
}

func resolveGlobalConfig(cmd *cobra.Command) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine home directory: %w", err)
	}
	globalClaudeDir = filepath.Join(home, ".claude")
	globalLockPath = filepath.Join(globalClaudeDir, "scaffold.lock")

	if configFlag != "" && scopeFlag == scopeGlobal {
		abs, err := filepath.Abs(configFlag)
		if err != nil {
			return fmt.Errorf("--config: could not resolve path %q: %w", configFlag, err)
		}
		globalXcfPath = abs
	} else {
		globalXcfPath = filepath.Join(globalClaudeDir, "global.xcf")
	}

	if cmd.Name() != "init" && cmd.Name() != "import" {
		if _, err := os.Stat(globalXcfPath); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("global.xcf not found at %q\n\nHint: run 'xcaffold init --scope global' to create one", globalXcfPath)
			}
			return fmt.Errorf("could not access %q: %w", globalXcfPath, err)
		}
	}
	return nil
}

func resolveProjectConfig(cmd *cobra.Command) error {
	if cmd.Name() == "init" || cmd.Name() == "import" {
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

	if _, err := os.Stat(xcfAbs); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("scaffold.xcf not found at %q\n\nHint: run 'xcaffold init' to create one, or use --config to specify a path", xcfAbs)
		}
		return fmt.Errorf("could not access %q: %w", xcfAbs, err)
	}

	xcfPath = xcfAbs
	claudeDir = filepath.Join(filepath.Dir(xcfAbs), ".claude")
	lockPath = filepath.Join(filepath.Dir(xcfAbs), "scaffold.lock")
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
