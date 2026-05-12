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
	version = "0.2.0-dev"
	commit  = "none"
	date    = "unknown"
)

// configFlag holds the value of the global --config flag.
// It is resolved before any subcommand runs.
var configFlag string

// xcafPath is the resolved, absolute path to the project.xcaf file.
// All subcommands should read from this rather than a hardcoded filename.
var xcafPath string

// projectRoot is the resolved, absolute path to the project's config directory.
var projectRoot string

// globalFlag indicates whether to operate on the user-wide global config.
var globalFlag bool

// globalXcafPath is the resolved path to global.xcaf.
var globalXcafPath string

// globalXcafHome is where global.xcaf lives ~/.xcaffold/ by convention.
var globalXcafHome string

// noColorFlag disables colored output in TTY.
var noColorFlag bool

// verboseFlag enables verbose output (fidelity notes, policy warnings).
var verboseFlag bool

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
	Long: `Deterministic agent configuration compiler.
Compiles .xcaf YAML into provider-native agent files (.claude/, .cursor/, .agents/).`,
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
		"Path to project.xcaf (default: ./project.xcaf). Use for monorepo sub-directories.",
	)
	rootCmd.PersistentFlags().BoolVarP(
		&globalFlag,
		"global",
		"g",
		false,
		"Operate on user-wide global config (~/.xcaffold/global.xcaf)",
	)
	rootCmd.PersistentFlags().BoolVar(
		&noColorFlag,
		"no-color",
		false,
		"disable color output",
	)
	rootCmd.PersistentFlags().BoolVar(
		&verboseFlag,
		"verbose",
		false,
		"show fidelity notes and policy warnings",
	)
	_ = rootCmd.PersistentFlags().MarkHidden("global")
	rootCmd.PersistentFlags().String("xcaf", "", "Display schema for a resource kind")
	rootCmd.PersistentFlags().String("out", "", "Generate template .xcaf file (use with --xcaf)")
	rootCmd.Flag("out").NoOptDefVal = "."
	rootCmd.SetHelpFunc(rootHelpFunc)
}

func rootHelpFunc(cmd *cobra.Command, args []string) {
	xcafKind, _ := cmd.Flags().GetString("xcaf")
	if xcafKind != "" {
		outPath, _ := cmd.Flags().GetString("out")
		outChanged := cmd.Flags().Changed("out")
		if err := runHelpXcaf(cmd, xcafKind, outPath, outChanged); err != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), err)
		}
		return
	}

	if cmd.Name() != "xcaffold" {
		if cmd.Long != "" {
			fmt.Fprintln(cmd.OutOrStdout(), cmd.Long)
			fmt.Fprintln(cmd.OutOrStdout())
		}
		fmt.Fprint(cmd.OutOrStdout(), cmd.UsageString())
		return
	}

	fmt.Printf("%s %s deterministic agent configuration compiler\n", bold("xcaffold"), glyphDot())
	fmt.Println()
	fmt.Printf("  %s  xcaffold [command]\n", dim("Usage:"))
	fmt.Println()
	fmt.Printf("  %s\n", dim("Commands:"))
	for _, c := range cmd.Commands() {
		if c.Hidden || c.Name() == "help" {
			continue
		}
		fmt.Printf("    %-12s%s\n", c.Name(), c.Short)
	}
	fmt.Println()
	fmt.Printf("  %s\n", dim("Flags:"))
	fmt.Printf("    --config <path>   Path to project.xcaf (default: ./project.xcaf)\n")
	fmt.Printf("    --no-color        Disable color output\n")
	fmt.Printf("    -h, --help        Show this help\n")
	fmt.Printf("    -v, --version     Show version\n")
	fmt.Println()
	fmt.Printf("%s Run 'xcaffold [command] --help' for details on any command.\n", glyphArrow())
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
	globalXcafHome = filepath.Join(home, ".xcaffold")

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
		globalXcafPath = abs
	} else {
		globalXcafPath = filepath.Join(globalXcafHome, "global.xcaf")
	}
	return nil
}

func resolveProjectConfig(cmd *cobra.Command) error {
	if cmd.Name() == "init" || cmd.Name() == "import" || cmd.Name() == "registry" || cmd.Name() == "help" {
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

	// Look for project.xcaf at the root of configDir
	xcafPath = filepath.Join(configDir, "project.xcaf")
	if _, err := os.Stat(xcafPath); err != nil {
		xcafPath = configDir
	}

	projectRoot = configDir
	return nil
}

// projectParseRoot returns the base directory to scan for .xcaf files.
// getParseRoot returns the base directory to scan for .xcaf files.
// It handles the case where the manifest is in .xcaffold/*.xcaf.
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
	return getParseRoot(xcafPath)
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
