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
	"github.com/saero-ai/xcaffold/internal/state"
	"github.com/spf13/cobra"
)

// Migration defines a single schema-version upgrade step.
//
//nolint:govet
type Migration struct {
	FromVersion string
	ToVersion   string
	Description string
	Apply       func(config *ast.XcaffoldConfig) error
}

// migrations is the ordered list of schema version upgrade steps.
var migrations = []Migration{}

// runSchemaVersionMigrations reads project.xcf at configPath, applies all
// applicable version migrations in order, and overwrites the file when any
// migration ran. The original file is preserved as project.xcf.bak.
func runSchemaVersionMigrations(cmd *cobra.Command, configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("could not read %s: %w", configPath, err)
	}

	config, err := parser.ParseDirectory(filepath.Dir(configPath))
	if err != nil {
		return fmt.Errorf("could not parse %s: %w", configPath, err)
	}

	anyApplied := false
	for _, m := range migrations {
		if config.Version != m.FromVersion {
			continue
		}
		if cmd != nil {
			cmd.Printf("  Applying migration: %s\n", m.Description)
		}
		if err := m.Apply(config); err != nil {
			return fmt.Errorf("migration %q failed: %w", m.Description, err)
		}
		anyApplied = true
	}

	if !anyApplied {
		if cmd != nil {
			cmd.Printf("Schema version is current (v%s).\n", config.Version)
		}
		return nil
	}

	// Back up original before overwriting
	bakPath := configPath + ".bak"
	if err := os.WriteFile(bakPath, data, 0600); err != nil {
		return fmt.Errorf("could not write backup %s: %w", bakPath, err)
	}

	if err := WriteSplitFiles(config, filepath.Dir(configPath)); err != nil {
		return fmt.Errorf("could not write %s: %w", configPath, err)
	}

	return nil
}

var migrateCmd = &cobra.Command{
	Use:    "migrate",
	Hidden: true,
	Short:  "Upgrade legacy xcaffold layouts to current schema",
	Long: `Migrate upgrades project.xcf file layouts and schema versions:

  - Schema version migrations (applies any pending version upgrades)
  - Project scope migration (flat paths → reference-in-place)`,
	RunE: runMigrate,
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}

func runMigrate(cmd *cobra.Command, args []string) error {
	migratedAnything := false
	configDir := "."

	// 0. Schema version migrations (must run before scope migrations so field
	//    transformations operate on the expected schema before path rewrites).
	if _, err := os.Stat("project.xcf"); err == nil {
		if err := runSchemaVersionMigrations(cmd, "project.xcf"); err != nil {
			return err
		}
	}

	// 1. Migrate Project Scope
	if ok, err := migrateProjectScope(cmd); err != nil {
		return err
	} else if ok {
		migratedAnything = true
	}

	// 2. Rename scaffold.xcf → project.xcf if present.
	if err := migrateRenameScaffoldXcf(configDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	// 3. Migrate legacy scaffold*.lock files to .xcaffold/ state files.
	if err := migrateStateLockFiles(configDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	if !migratedAnything {
		cmd.Println("Everything is up to date.")
	}

	return nil
}

// migrateRenameScaffoldXcf detects scaffold.xcf and offers to rename it to project.xcf.
func migrateRenameScaffoldXcf(baseDir string) error {
	oldPath := filepath.Join(baseDir, "scaffold.xcf")
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return nil // nothing to migrate
	}

	newPath := filepath.Join(baseDir, "project.xcf")
	if _, err := os.Stat(newPath); err == nil {
		fmt.Fprintf(os.Stderr, "  scaffold.xcf exists but project.xcf already exists — skipping rename\n")
		return nil
	}

	// Backup
	backupPath := oldPath + ".backup"
	data, err := os.ReadFile(oldPath)
	if err != nil {
		return fmt.Errorf("failed to read scaffold.xcf: %w", err)
	}
	if err := os.WriteFile(backupPath, data, 0600); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("failed to rename scaffold.xcf to project.xcf: %w", err)
	}

	fmt.Printf("  Renamed scaffold.xcf → project.xcf (backup: scaffold.xcf.backup)\n")
	return nil
}

// migrateStateLockFiles migrates legacy lock files to .xcaffold/ state files.
func migrateStateLockFiles(baseDir string) error {
	migrated, err := state.MigrateStateFiles(baseDir)
	if err != nil {
		return fmt.Errorf("lock file migration failed: %w", err)
	}
	if migrated {
		fmt.Printf("  Migrated lock files → .xcaffold/project.xcf.state\n")
	}
	return nil
}

// migrateProjectScope rewrites any flat-layout instruction_file paths in
// project.xcf to the full reference-in-place paths, then registers the project.
func migrateProjectScope(cmd *cobra.Command) (bool, error) {
	configFile := "project.xcf"
	if _, err := os.Stat(configFile); err != nil {
		return false, nil
	}

	config, err := parser.ParseDirectory(".")
	if err != nil {
		return false, nil
	}

	needsUpdate := migrateAgentPaths(config)
	needsUpdate = migrateSkillPaths(config) || needsUpdate
	needsUpdate = migrateRulePaths(config) || needsUpdate

	cwd, _ := os.Getwd()

	if !needsUpdate {
		if config.Project != nil {
			_ = registry.Register(cwd, config.Project.Name, nil, ".")
		}
		return false, nil
	}

	cmd.Printf("Found legacy flat-layout paths in %s\n", configFile)
	confirm, err := prompt.Confirm("Migrate Project layout to reference-in-place?", true)
	if err != nil || !confirm {
		return false, nil
	}

	if err := WriteSplitFiles(config, "."); err != nil {
		return false, fmt.Errorf("failed to write %s: %w", configFile, err)
	}

	if config.Project != nil {
		_ = registry.Register(cwd, config.Project.Name, nil, ".")
	}
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
