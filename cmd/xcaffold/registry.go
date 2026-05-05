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
		targets := "none"
		if len(p.Targets) > 0 {
			targets = strings.Join(p.Targets, ", ")
		}

		var resInfo string
		var xcfProjectPath string
		if _, err := os.Stat(filepath.Join(p.Path, "project.xcf")); err == nil {
			xcfProjectPath = filepath.Join(p.Path, "project.xcf")
		}

		if xcfProjectPath != "" {
			if cfg, err := parser.ParseDirectory(getParseRoot(xcfProjectPath)); err == nil {
				resInfo = fmt.Sprintf("    resources: %d agents, %d skills, %d rules", len(cfg.Agents), len(cfg.Skills), len(cfg.Rules))
			} else {
				resInfo = "    resources: parse error"
			}
		} else {
			resInfo = "    resources: not found (project.xcf missing)"
		}

		lastApplied := "never"
		if !p.LastApplied.IsZero() {
			val := p.LastApplied.Local()
			if time.Since(val).Hours() < 24 {
				lastApplied = "today at " + val.Format("15:04")
			} else if time.Since(val).Hours() < 48 {
				lastApplied = "yesterday"
			} else {
				lastApplied = val.Format("2006-01-02 15:04")
			}
		}

		cmd.Printf("  ● \033[1m%s\033[0m (%s)\n", p.Name, p.Path)
		cmd.Printf("    targets: %s\n", targets)
		cmd.Printf("%s\n", resInfo)
		cmd.Printf("    last applied: %s\n", lastApplied)
		cmd.Println()
	}

	// Show global info
	cmd.Printf("  GLOBAL (%s)\n", globalXcfPath)
	if _, err := os.Stat(globalXcfPath); err == nil {
		if cfg, err := parser.ParseDirectory(getParseRoot(globalXcfPath)); err == nil {
			cmd.Printf("    resources: %d agents, %d skills, %d rules\n", len(cfg.Agents), len(cfg.Skills), len(cfg.Rules))
		} else {
			cmd.Printf("    resources: parse error\n")
		}
	} else if os.IsNotExist(err) {
		cmd.Printf("    resources: none (not initialized)\n")
	} else if _, isPathErr := err.(*fs.PathError); isPathErr {
		cmd.Printf("    resources: none (not initialized)\n")
	}

	cmd.Println()
	cmd.Printf("  %d projects registered.\n", len(projects))

	return nil
}
