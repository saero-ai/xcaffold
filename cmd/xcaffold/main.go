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

// xcfPath is the resolved, absolute path to the project.xcf file.
// All subcommands should read from this rather than a hardcoded filename.
var xcfPath string

// projectRoot is the resolved, absolute path to the project's config directory.
var projectRoot string

// globalFlag indicates whether to operate on the user-wide global config.
var globalFlag bool

// globalXcfPath is the resolved path to global.xcf.
var globalXcfPath string

// globalXcfHome is where global.xcf lives ~/.xcaffold/ by convention.
var globalXcfHome string

// noColorFlag disables colored output in TTY.
var noColorFlag bool

type driftDetectedError struct {
	msg string
}

func (e *driftDetectedError) Error() string {
	return e.msg
}

type silentError struct {
	msg string
}

func (e *silentError) Error() string {
	return e.msg
}

var rootCmd = &cobra.Command{
	Use:   "xcaffold",
	Short: "xcaffold — deterministic agent configuration compiler",
	Long: `xcaffold is an open-source, deterministic agent configuration compiler.

Scopes:
  Project  [default]         project.xcf            -> .claude/ | .cursor/ | .agents/
  Global   [--global / -g]   ~/.xcaffold/global.xcf -> ~/.claude/ | ~/.cursor/ | ~/.agents/`,
	PersistentPreRunE: resolveConfig,
	SilenceErrors:     true,
}

func init() {
	state.XcaffoldVersion = version
	rootCmd.Version = fmt.Sprintf("%s (commit: %s, date: %s)", version, commit, date)

	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	rootCmd.PersistentFlags().StringVar(
		&configFlag,
		"config",
		"",
		"Path to project.xcf (default: ./project.xcf). Use for monorepo sub-directories.",
	)
	rootCmd.PersistentFlags().BoolVarP(
		&globalFlag,
		"global",
		"g",
		false,
		"Operate on user-wide global config (~/.xcaffold/global.xcf)",
	)
	rootCmd.PersistentFlags().BoolVar(
		&noColorFlag,
		"no-color",
		false,
		"disable color output",
	)
}

// resolveConfig is a PersistentPreRunE hook that runs before every subcommand.
// It resolves the --config and --global flags into a stable set of absolute paths
// that all subcommands can use without re-implementing CWD logic.
func resolveConfig(cmd *cobra.Command, args []string) error {
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

	if cmd.Name() != "init" && cmd.Name() != "import" {
		fmt.Println(formatHeader("~", "", true, "", ""))
		fmt.Println()
		fmt.Printf("  %s  Global scope is not yet available.\n", glyphErr())
		fmt.Println()
		fmt.Printf("%s Run 'xcaffold %s' for project-scoped operation.\n", glyphArrow(), cmd.Name())
		return &silentError{msg: "global scope is not yet available"}
	}

	if configFlag != "" && globalFlag {
		abs, err := filepath.Abs(configFlag)
		if err != nil {
			return fmt.Errorf("--config: could not resolve path %q: %w", configFlag, err)
		}
		globalXcfPath = abs
	} else {
		globalXcfPath = filepath.Join(globalXcfHome, "global.xcf")
	}
	return nil
}

func resolveProjectConfig(cmd *cobra.Command) error {
	if cmd.Name() == "init" || cmd.Name() == "import" || cmd.Name() == "registry" {
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

	candidates := []string{
		filepath.Join(configDir, ".xcaffold", "project.xcf"),
		filepath.Join(configDir, "project.xcf"),
	}
	xcfPath = configDir
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			xcfPath = c
			break
		}
	}

	projectRoot = configDir
	return nil
}

// projectParseRoot returns the base directory to scan for .xcf files.
// getParseRoot returns the base directory to scan for .xcf files.
// It handles the case where the manifest is in .xcaffold/*.xcf.
func getParseRoot(manifestPath string) string {
	dir := filepath.Dir(manifestPath)
	if filepath.Base(dir) == ".xcaffold" {
		return filepath.Dir(dir)
	}
	return dir
}

func projectParseRoot() string {
	if projectRoot != "" {
		return projectRoot
	}
	return getParseRoot(xcfPath)
}

func main() {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	err := rootCmd.Execute()
	if err != nil {
		if _, ok := err.(*silentError); ok {
			os.Exit(1)
		}
		if _, ok := err.(*driftDetectedError); ok {
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
