package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/saero-ai/xcaffold/internal/registry"
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

// globalXcfHome is where global.xcf lives ~/.xcaffold/ by convention.
var globalXcfHome string

// globalLockPath is ~/.xcaffold/scaffold.lock.
var globalLockPath string

var rootCmd = &cobra.Command{
	Use:   "xcaffold",
	Short: "xcaffold — deterministic agent-as-code orchestration",
	Long: `xcaffold is an open-source, deterministic agent configuration compiler engine for Claude Code.

 ┌───────────────────────────────────────────────────────────────────┐
 │                 THE 8-PHASE ORCHESTRATION ENGINE                  │
 └───────────────────────────────────────────────────────────────────┘
  • Bootstrap   [xcaffold init]      Creates base project scaffolding
  • Ingestion   [xcaffold import]    Migrates dirs & translates via --source
  • Audit       [xcaffold analyze]   Inspects repo & builds XCF config
  • Topology    [xcaffold graph]     Visualizes agent topology maps
  • Compilation [xcaffold apply]     Compiles XCF (use --check to validate syntax)
  • Drift Check [xcaffold diff]      Detects manual config tampering
  • Validation  [xcaffold test]      Runs an LLM-in-the-loop proxy
  • Export      [xcaffold export]    Packages output as a distributable plugin

 ┌───────────────────────────────────────────────────────────────────┐
 │                            UTILITIES                              │
 └───────────────────────────────────────────────────────────────────┘
  • Validate    [xcaffold validate]  Checks syntax, cross-refs, and structural invariants
  • Review      [xcaffold review]    Universally parses state files
    ↳ Supports: scaffold.xcf, audit.json, plan.json, trace.jsonl
    ↳ Try: 'xcaffold review all'
  • Registry    [xcaffold list]      Lists all managed projects
  • Migration   [xcaffold migrate]   Upgrades legacy layouts

 ┌───────────────────────────────────────────────────────────────────┐
 │                           SCOPES                                  │
 └───────────────────────────────────────────────────────────────────┘
  • Project  [default]         scaffold.xcf           -> .claude/ | .cursor/ | .agents/
  • Global   [--scope global]  ~/.xcaffold/global.xcf -> ~/.claude/ | ~/.cursor/ | ~/.agents/
  • Both     [--scope all]     Compiles both scopes

Use 'xcaffold --help' for more information on available commands.`,
	PersistentPreRunE: resolveConfig,
}

func init() {
	state.XcaffoldVersion = version
	rootCmd.Version = fmt.Sprintf("%s (commit: %s, date: %s)", version, commit, date)

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

	if err := registry.EnsureGlobalHome(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not initialize global home: %v\n", err)
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
	globalXcfHome = filepath.Join(home, ".xcaffold")
	globalLockPath = filepath.Join(globalXcfHome, "scaffold.lock")

	if configFlag != "" && scopeFlag == scopeGlobal {
		abs, err := filepath.Abs(configFlag)
		if err != nil {
			return fmt.Errorf("--config: could not resolve path %q: %w", configFlag, err)
		}
		globalXcfPath = abs
	} else {
		globalXcfPath = filepath.Join(globalXcfHome, "global.xcf")
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
	if cmd.Name() == "init" || cmd.Name() == "import" || cmd.Name() == "list" || cmd.Name() == "migrate" {
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

		// Walk up to find scaffold.xcf (stops at $HOME)
		home, _ := os.UserHomeDir()
		curr := cwd
		for {
			candidate := filepath.Join(curr, "scaffold.xcf")
			if _, err := os.Stat(candidate); err == nil {
				xcfAbs = candidate
				break
			}
			if curr == home {
				xcfAbs = filepath.Join(cwd, "scaffold.xcf") // fallback to allow error handling below
				break
			}
			parent := filepath.Dir(curr)
			if parent == curr {
				xcfAbs = filepath.Join(cwd, "scaffold.xcf")
				break
			}
			curr = parent
		}
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
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
