package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/registry"
	"github.com/spf13/cobra"
)

var registryCmd = &cobra.Command{
	Use:    "registry",
	Hidden: true,
	Short:  "List projects registered in the global registry",
	Long:   "Displays a list of all projects managed by xcaffold across your system.",
	RunE:   runRegistry,
}

func init() {
	rootCmd.AddCommand(registryCmd)
}

func runRegistry(cmd *cobra.Command, args []string) error {
	projects, err := registry.List()
	if err != nil {
		return fmt.Errorf("failed to read registry: %w", err)
	}

	cmd.Println()
	cmd.Println("  MANAGED PROJECTS")
	cmd.Println()

	if len(projects) == 0 {
		cmd.Println("  No projects registered yet.")
		cmd.Println("  Run 'xcaffold init' inside a repository to register it.")
	}

	for _, p := range projects {
		printProjectRegistry(cmd, &p)
	}

	printGlobalRegistry(cmd)
	cmd.Println()
	cmd.Printf("  %d projects registered.\n", len(projects))

	return nil
}

// printProjectRegistry outputs a single project entry.
func printProjectRegistry(cmd *cobra.Command, p *registry.Project) {
	targets := "none"
	if len(p.Targets) > 0 {
		targets = strings.Join(p.Targets, ", ")
	}

	resInfo := getProjectResourceInfo(p)
	lastApplied := formatLastApplied(p.LastApplied)

	cmd.Printf("  ● \033[1m%s\033[0m (%s)\n", p.Name, p.Path)
	cmd.Printf("    targets: %s\n", targets)
	cmd.Printf("%s\n", resInfo)
	cmd.Printf("    last applied: %s\n", lastApplied)
	cmd.Println()
}

// getProjectResourceInfo retrieves resource count for a project.
func getProjectResourceInfo(p *registry.Project) string {
	xcafPath := filepath.Join(p.Path, "project.xcaf")
	if _, err := os.Stat(xcafPath); err != nil {
		return "    resources: not found (project.xcaf missing)"
	}
	if cfg, err := parser.ParseDirectory(getParseRoot(xcafPath)); err == nil {
		return fmt.Sprintf("    resources: %d agents, %d skills, %d rules", len(cfg.Agents), len(cfg.Skills), len(cfg.Rules))
	}
	return "    resources: parse error"
}

// formatLastApplied formats the last applied timestamp.
func formatLastApplied(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	val := t.Local()
	if time.Since(val).Hours() < 24 {
		return "today at " + val.Format("15:04")
	}
	if time.Since(val).Hours() < 48 {
		return "yesterday"
	}
	return val.Format("2006-01-02 15:04")
}

// printGlobalRegistry outputs the global registry info.
func printGlobalRegistry(cmd *cobra.Command) {
	cmd.Printf("  GLOBAL (%s)\n", globalXcafPath)
	if _, err := os.Stat(globalXcafPath); err == nil {
		if cfg, err := parser.ParseDirectory(getParseRoot(globalXcafPath)); err == nil {
			cmd.Printf("    resources: %d agents, %d skills, %d rules\n", len(cfg.Agents), len(cfg.Skills), len(cfg.Rules))
		} else {
			cmd.Printf("    resources: parse error\n")
		}
	} else if os.IsNotExist(err) || isFSPathError(err) {
		cmd.Printf("    resources: none (not initialized)\n")
	}
}

// isFSPathError checks if an error is a filesystem path error.
func isFSPathError(err error) bool {
	_, isPathErr := err.(*fs.PathError)
	return isPathErr
}
