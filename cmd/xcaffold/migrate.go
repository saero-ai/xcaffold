package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/prompt"
	"github.com/saero-ai/xcaffold/internal/registry"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate legacy xcaffold layouts to the centralized architecture",
	Long:  "Detects legacy flat layouts and ~/.claude/global.xcf, and updates them to the reference-in-place model and ~/.xcaffold/ registry architecture.",
	RunE:  runMigrate,
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}

func runMigrate(cmd *cobra.Command, args []string) error {
	migratedAnything := false

	// 1. Migrate Global Scope
	home, err := os.UserHomeDir()
	if err == nil {
		oldGlobal := filepath.Join(home, ".claude", "global.xcf")
		newGlobal := filepath.Join(home, ".xcaffold", "global.xcf")
		if _, err := os.Stat(oldGlobal); err == nil {
			if _, newErr := os.Stat(newGlobal); os.IsNotExist(newErr) {
				cmd.Printf("Found legacy global config at %s\n", oldGlobal)
				confirm, err := prompt.Confirm("Migrate global layout to ~/.xcaffold/ ?", true)
				if err == nil && confirm {
					if err := registry.EnsureGlobalHome(); err == nil {
						data, _ := os.ReadFile(oldGlobal)
						os.WriteFile(newGlobal, data, 0600)
						// Also move lockfile if present
						oldLock := filepath.Join(home, ".claude", "scaffold.lock")
						if lockData, err := os.ReadFile(oldLock); err == nil {
							os.WriteFile(filepath.Join(home, ".xcaffold", "scaffold.lock"), lockData, 0600)
						}
						cmd.Println("  ✓ Global config migrated.", "")
						migratedAnything = true
					}
				}
			}
		}
	}

	// 2. Migrate Project Scope (Check scaffold.xcf)
	if _, err := os.Stat("scaffold.xcf"); err == nil {
		config, err := parser.ParseFile("scaffold.xcf")
		if err == nil {
			needsUpdate := false
			for id, a := range config.Agents {
				if !strings.Contains(a.InstructionsFile, "/") && a.InstructionsFile != "" {
					needsUpdate = true
					config.Agents[id] = ast.AgentConfig{
						Description:      a.Description,
						InstructionsFile: filepath.ToSlash(filepath.Join(".claude", "agents", a.InstructionsFile)),
						Instructions:     a.Instructions,
						Model:            a.Model,
						Effort:           a.Effort,
						Tools:            a.Tools,
						Skills:           a.Skills,
						Rules:            a.Rules,
						MCP:              a.MCP,
						Assertions:       a.Assertions,
					}
				}
			}

			if needsUpdate {
				cmd.Println("Found legacy flat-layout paths in scaffold.xcf")
				confirm, err := prompt.Confirm("Migrate Project layout to reference-in-place?", true)
				if err == nil && confirm {
					out, _ := yaml.Marshal(config)
					os.WriteFile("scaffold.xcf", out, 0600)
					cwd, _ := os.Getwd()
					_ = registry.Register(cwd, config.Project.Name, []string{"claude"})
					cmd.Println("  ✓ Project config migrated and registered.")
					migratedAnything = true
				}
			} else {
				// ensure it's registered
				cwd, _ := os.Getwd()
				_ = registry.Register(cwd, config.Project.Name, []string{"claude"})
			}
		}
	}

	if !migratedAnything {
		cmd.Println("Everything is up to date.")
	}

	return nil
}
