package main

import (
	"fmt"
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
	if ok, err := migrateGlobalScope(cmd); err != nil {
		return err
	} else if ok {
		migratedAnything = true
	}

	// 2. Migrate Project Scope
	if ok, err := migrateProjectScope(cmd); err != nil {
		return err
	} else if ok {
		migratedAnything = true
	}

	if !migratedAnything {
		cmd.Println("Everything is up to date.")
	}

	return nil
}

// migrateGlobalScope moves ~/.claude/global.xcf → ~/.xcaffold/global.xcf.
func migrateGlobalScope(cmd *cobra.Command) (bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, nil
	}

	oldGlobal := filepath.Join(home, ".claude", "global.xcf")
	newGlobal := filepath.Join(home, ".xcaffold", "global.xcf")

	if _, err := os.Stat(oldGlobal); err != nil {
		return false, nil // nothing to migrate
	}
	if _, err := os.Stat(newGlobal); !os.IsNotExist(err) {
		return false, nil // already migrated
	}

	cmd.Printf("Found legacy global config at %s\n", oldGlobal)
	confirm, err := prompt.Confirm("Migrate global layout to ~/.xcaffold/ ?", true)
	if err != nil || !confirm {
		return false, nil
	}

	if err := registry.EnsureGlobalHome(); err != nil {
		return false, err
	}

	data, _ := os.ReadFile(oldGlobal)
	_ = os.WriteFile(newGlobal, data, 0600)

	// Also move lockfile if present
	if lockData, err := os.ReadFile(filepath.Join(home, ".claude", "scaffold.lock")); err == nil {
		_ = os.WriteFile(filepath.Join(home, ".xcaffold", "scaffold.lock"), lockData, 0600)
	}

	cmd.Println("  ✓ Global config migrated.")
	return true, nil
}

// migrateProjectScope rewrites any flat-layout instruction_file paths in scaffold.xcf
// to the full reference-in-place paths, then registers the project.
func migrateProjectScope(cmd *cobra.Command) (bool, error) {
	if _, err := os.Stat("scaffold.xcf"); err != nil {
		return false, nil
	}

	config, err := parser.ParseFile("scaffold.xcf")
	if err != nil {
		return false, nil
	}

	needsUpdate := migrateAgentPaths(config)
	needsUpdate = migrateSkillPaths(config) || needsUpdate
	needsUpdate = migrateRulePaths(config) || needsUpdate

	cwd, _ := os.Getwd()

	if !needsUpdate {
		_ = registry.Register(cwd, config.Project.Name, nil)
		return false, nil
	}

	cmd.Println("Found legacy flat-layout paths in scaffold.xcf")
	confirm, err := prompt.Confirm("Migrate Project layout to reference-in-place?", true)
	if err != nil || !confirm {
		return false, nil
	}

	out, _ := yaml.Marshal(config)
	if err := os.WriteFile("scaffold.xcf", out, 0600); err != nil {
		return false, fmt.Errorf("failed to write scaffold.xcf: %w", err)
	}

	_ = registry.Register(cwd, config.Project.Name, nil)
	cmd.Println("  ✓ Project config migrated and registered.")
	return true, nil
}

func migrateAgentPaths(config *ast.XcaffoldConfig) bool {
	changed := false
	for id, a := range config.Agents {
		if a.InstructionsFile != "" && !strings.Contains(a.InstructionsFile, "/") {
			a.InstructionsFile = filepath.ToSlash(filepath.Join(".claude", "agents", a.InstructionsFile))
			config.Agents[id] = a
			changed = true
		}
	}
	return changed
}

func migrateSkillPaths(config *ast.XcaffoldConfig) bool {
	changed := false
	for id, s := range config.Skills {
		updated := false
		if s.InstructionsFile != "" && !strings.Contains(s.InstructionsFile, "/") {
			s.InstructionsFile = filepath.ToSlash(filepath.Join(".claude", "skills", id, "SKILL.md"))
			updated = true
		}
		for i, ref := range s.References {
			if ref != "" && !strings.Contains(ref, "/") {
				s.References[i] = filepath.ToSlash(filepath.Join(".claude", "skills", id, ref))
				updated = true
			}
		}
		if updated {
			config.Skills[id] = s
			changed = true
		}
	}
	return changed
}

func migrateRulePaths(config *ast.XcaffoldConfig) bool {
	changed := false
	for id, r := range config.Rules {
		if r.InstructionsFile != "" && !strings.Contains(r.InstructionsFile, "/") {
			r.InstructionsFile = filepath.ToSlash(filepath.Join(".claude", "rules", r.InstructionsFile))
			config.Rules[id] = r
			changed = true
		}
	}
	return changed
}
