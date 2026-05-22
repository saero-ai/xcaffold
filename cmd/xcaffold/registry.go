package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/registry"
	"github.com/spf13/cobra"
)

var registryCmd = &cobra.Command{
	Use:   "registry",
	Short: "Manage the project registry",
	Long:  "Manage the global project registry, including listing, adding, removing, and pruning projects.",
	RunE:  runRegistryList,
}

var registryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered projects",
	Long:  "Display all projects in the global registry in tabular format.",
	RunE:  runRegistryList,
}

var registryAddCmd = &cobra.Command{
	Use:   "add PATH",
	Short: "Register a new project",
	Long:  "Register a project from its filesystem path. The project must contain a project.xcaf file.",
	Args:  cobra.ExactArgs(1),
	RunE:  runRegistryAdd,
}

var registryRemoveCmd = &cobra.Command{
	Use:   "remove NAME_OR_PATH",
	Short: "Unregister a project",
	Long:  "Remove a project from the registry by name or path.",
	Args:  cobra.ExactArgs(1),
	RunE:  runRegistryRemove,
}

var registryPruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove stale projects from the registry",
	Long:  "Remove projects that no longer exist on the filesystem from the registry.",
	RunE:  runRegistryPrune,
}

var registryInfoCmd = &cobra.Command{
	Use:   "info NAME_OR_PATH",
	Short: "Show detailed information about a registered project",
	Long:  "Display extended metadata and filesystem status for a registered project.",
	Args:  cobra.ExactArgs(1),
	RunE:  runRegistryInfo,
}

// Registry flags
var (
	registryListJSON    bool
	registryListVerbose bool
	registryPruneDryRun bool
	registryInfoJSON    bool
)

func init() {
	rootCmd.AddCommand(registryCmd)
	registryCmd.AddCommand(registryListCmd, registryAddCmd, registryRemoveCmd, registryPruneCmd, registryInfoCmd)

	registryListCmd.Flags().BoolVar(&registryListJSON, "json", false, "Output as JSON")
	registryListCmd.Flags().BoolVarP(&registryListVerbose, "verbose", "v", false, "Show additional details")

	registryPruneCmd.Flags().BoolVar(&registryPruneDryRun, "dry-run", false, "Show what would be removed without modifying the registry")

	registryInfoCmd.Flags().BoolVar(&registryInfoJSON, "json", false, "Output as JSON")
}

// runRegistryList lists all registered projects (also serves as default for parent command).
func runRegistryList(cmd *cobra.Command, args []string) error {
	projects, err := registry.List()
	if err != nil {
		return fmt.Errorf("failed to read registry: %w", err)
	}

	if registryListJSON {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(projects)
	}

	cmd.Println()
	if len(projects) == 0 {
		cmd.Println("  No projects registered yet.")
		cmd.Println("  Run 'xcaffold init' inside a repository to register it.")
		cmd.Println()
		return nil
	}

	// Sort alphabetically by name
	sort.Slice(projects, func(i, j int) bool { return projects[i].Name < projects[j].Name })

	// Print header
	cmd.Printf("  %-30s %-40s %-20s %-15s %-15s\n", "NAME", "PATH", "TARGETS", "LAST APPLIED", "STATUS")
	cmd.Println(strings.Repeat(" ", 30) + strings.Repeat("-", 100))

	for _, p := range projects {
		status := projectStatus(p)
		targetsStr := "none"
		if len(p.Targets) > 0 {
			if registryListVerbose {
				targetsStr = strings.Join(p.Targets, ", ")
			} else {
				targetsStr = fmt.Sprintf("%d targets", len(p.Targets))
			}
		}
		pathStr := p.Path
		if len(pathStr) > 40 {
			pathStr = "..." + pathStr[len(pathStr)-37:]
		}
		lastApplied := formatLastApplied(p.LastApplied)
		cmd.Printf("  %-30s %-40s %-20s %-15s %-15s\n", p.Name, pathStr, targetsStr, lastApplied, status)

		// In verbose mode, show registration timestamp and config directory
		if registryListVerbose {
			cmd.Printf("    Registered: %s\n", p.Registered.Format("2006-01-02 15:04:05"))
			if p.ConfigDir != "" {
				cmd.Printf("    Config Dir:  %s\n", p.ConfigDir)
			}
		}
	}

	cmd.Println()
	cmd.Printf("  %d projects registered.\n", len(projects))
	cmd.Println()

	return nil
}

// runRegistryAdd registers a new project.
func runRegistryAdd(cmd *cobra.Command, args []string) error {
	path := args[0]

	// Resolve absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("could not resolve path: %w", err)
	}

	// Check if path exists
	if _, err := os.Stat(absPath); err != nil {
		return fmt.Errorf("path does not exist: %s", absPath)
	}

	// Try to infer name and targets from project.xcaf
	var name string
	var targets []string

	projectXcf := filepath.Join(absPath, "project.xcaf")
	if cfg, err := parser.ParseFile(projectXcf); err == nil && cfg.Project != nil {
		name = cfg.Project.Name
		targets = cfg.Project.Targets
	}

	// If no name found, use directory name
	if name == "" {
		name = filepath.Base(absPath)
	}

	// Register the project
	if err := registry.Register(absPath, name, targets, ""); err != nil {
		return fmt.Errorf("failed to register project: %w", err)
	}

	cmd.Printf("Registered project: %s (%s)\n", name, absPath)
	return nil
}

// runRegistryRemove unregisters a project.
func runRegistryRemove(cmd *cobra.Command, args []string) error {
	nameOrPath := args[0]

	removed, err := registry.Unregister(nameOrPath)
	if err != nil {
		return fmt.Errorf("failed to remove project: %w", err)
	}

	cmd.Printf("Removed project: %s (%s)\n", removed.Name, removed.Path)
	return nil
}

// runRegistryPrune removes stale (missing) projects from the registry.
func runRegistryPrune(cmd *cobra.Command, args []string) error {
	// Get total count before pruning
	allProjects, err := registry.List()
	if err != nil {
		return fmt.Errorf("failed to read registry: %w", err)
	}
	totalCount := len(allProjects)

	removed, err := registry.Prune(registryPruneDryRun)
	if err != nil {
		return fmt.Errorf("failed to prune registry: %w", err)
	}

	if len(removed) == 0 {
		cmd.Println()
		cmd.Println("  Registry is clean. 0 stale entries found.")
		cmd.Println()
		return nil
	}

	cmd.Println()
	for _, p := range removed {
		cmd.Printf("  Pruned: \"%s\" (%s) — path does not exist\n", p.Name, p.Path)
	}
	cmd.Println()
	cmd.Printf("  Pruned %d of %d projects.\n", len(removed), totalCount)
	cmd.Println()

	return nil
}

// runRegistryInfo displays detailed information about a project.
func runRegistryInfo(cmd *cobra.Command, args []string) error {
	nameOrPath := args[0]

	info, err := registry.Info(nameOrPath)
	if err != nil {
		return fmt.Errorf("project not found: %w", err)
	}

	if registryInfoJSON {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(info)
	}

	cmd.Println()
	cmd.Printf("  Name:        %s\n", info.Name)
	cmd.Printf("  Path:        %s\n", info.Path)
	cmd.Printf("  Registered:  %s\n", info.Registered.Format("2006-01-02 15:04:05"))
	cmd.Printf("  Last Applied:%s\n", formatLastApplied(info.LastApplied))
	if len(info.Targets) > 0 {
		cmd.Printf("  Targets:     %s\n", strings.Join(info.Targets, ", "))
	} else {
		cmd.Printf("  Targets:     none\n")
	}
	if info.ConfigDir != "" {
		cmd.Printf("  Config Dir:  %s\n", info.ConfigDir)
	}
	cmd.Printf("  Exists:      %v\n", info.Exists)
	cmd.Printf("  Has xcaf/:   %v\n", info.HasXcafDir)
	cmd.Printf("  Has project.xcaf: %v\n", info.HasProjectXcf)
	cmd.Println()

	return nil
}

// projectStatus returns the status of a project (ok, stale, orphan).
func projectStatus(p registry.Project) string {
	if !registry.PathExists(p) {
		return "stale"
	}
	xcafPath := filepath.Join(p.Path, "xcaf")
	projPath := filepath.Join(p.Path, "project.xcaf")
	if _, err := os.Stat(xcafPath); err != nil {
		if _, err := os.Stat(projPath); err != nil {
			return "orphan"
		}
	}
	return "ok"
}

// formatLastApplied formats the last applied timestamp.
func formatLastApplied(t time.Time) string {
	if t.IsZero() {
		return " never"
	}
	val := t.Local()
	if time.Since(val).Hours() < 24 {
		return " today at " + val.Format("15:04")
	}
	if time.Since(val).Hours() < 48 {
		return " yesterday"
	}
	return " " + val.Format("2006-01-02 15:04")
}
