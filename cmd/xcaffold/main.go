package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/saero-ai/xcaffold/internal/registry"
	"github.com/saero-ai/xcaffold/internal/resolver"
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

// globalFlag indicates whether to operate on the user-wide global config.
var globalFlag bool

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
  • Compilation [xcaffold apply]     Evaluates policies and compiles XCF to agent outputs
  • Drift Check [xcaffold diff]      Detects manual config tampering
  • Validation  [xcaffold test]      Runs an LLM-in-the-loop proxy
  • Export      [xcaffold export]    Packages output as a distributable plugin

 ┌───────────────────────────────────────────────────────────────────┐
 │                            UTILITIES                              │
 └───────────────────────────────────────────────────────────────────┘
  • Validate    [xcaffold validate]  Checks syntax, cross-refs, and enforces project policies
  • Review      [xcaffold review]    Universally parses state files
    ↳ Supports: scaffold.xcf, audit.json, plan.json, trace.jsonl
    ↳ Try: 'xcaffold review all'
  • Registry    [xcaffold list]      Lists all managed projects
  • Migration   [xcaffold migrate]   Upgrades legacy layouts

 ┌───────────────────────────────────────────────────────────────────┐
 │                           SCOPES                                  │
 └───────────────────────────────────────────────────────────────────┘
  • Project  [default]         scaffold.xcf           -> .claude/ | .cursor/ | .agents/
  • Global   [--global / -g]   ~/.xcaffold/global.xcf -> ~/.claude/ | ~/.cursor/ | ~/.agents/

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
	rootCmd.PersistentFlags().BoolVarP(
		&globalFlag,
		"global",
		"g",
		false,
		"Operate on user-wide global config (~/.xcaffold/global.xcf)",
	)
}

// resolveConfig is a PersistentPreRunE hook that runs before every subcommand.
// It resolves the --config and --global flags into a stable set of absolute paths
// that all subcommands can use without re-implementing CWD logic.
func resolveConfig(cmd *cobra.Command, args []string) error {
	if cmd.Name() == "review" {
		return nil
	}

	if err := registry.EnsureGlobalHome(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not initialize global home: %v\n", err)
	}

	if globalFlag {
		return resolveGlobalConfig(cmd)
	}
	return resolveProjectConfig(cmd)
}

func resolveGlobalConfig(cmd *cobra.Command) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine home directory: %w", err)
	}
	globalXcfHome = filepath.Join(home, ".xcaffold")
	globalLockPath = filepath.Join(globalXcfHome, "scaffold.lock")

	if configFlag != "" && globalFlag {
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
				return fmt.Errorf("global.xcf not found at %q\n\nHint: run 'xcaffold init --global' to create one", globalXcfPath)
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
	var configDir string
	if configFlag != "" {
		abs, err := filepath.Abs(configFlag)
		if err != nil {
			return fmt.Errorf("--config: could not resolve path %q: %w", configFlag, err)
		}
		info, err := os.Stat(abs)
		if err != nil {
			return fmt.Errorf("--config: %q does not exist: %w", configFlag, err)
		}
		if info.IsDir() {
			configDir = abs
		} else {
			// Single file: use its parent directory as the config dir
			configDir = filepath.Dir(abs)
		}
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("could not determine working directory: %w", err)
		}
		home, _ := os.UserHomeDir()
		dir, err := resolver.FindConfigDir(cwd, home)
		if err != nil {
			return err
		}
		configDir = dir
	}

	// For backward compatibility, check if scaffold.xcf exists as the
	// canonical entry point. If not, use the config directory itself.
	candidate := filepath.Join(configDir, "scaffold.xcf")
	if _, err := os.Stat(candidate); err == nil {
		xcfPath = candidate
	} else {
		xcfPath = configDir
	}

	claudeDir = filepath.Join(configDir, ".claude")
	lockPath = filepath.Join(configDir, "scaffold.lock")
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
