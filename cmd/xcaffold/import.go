package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Migrate an existing .claude/ directory into scaffold.xcf",
	Long: `xcaffold import scans an existing .claude/ directory and generates
a scaffold.xcf configuration file, allowing you to adopt xcaffold on
existing projects.

┌───────────────────────────────────────────────────────────────────┐
│                          IMPORT PHASE                             │
└───────────────────────────────────────────────────────────────────┘
 • Scans .claude/agents/*.md   → extracts to agents/<id>.md
 • Scans .claude/skills/*/SKILL.md → extracts to skills/<id>/SKILL.md
 • Scans .claude/rules/*.md    → extracts to rules/<id>.md
 • Reads .claude/settings.json for MCP and settings context
 • Generates a scaffold.xcf with instructions_file: references
   (no content is inlined — full fidelity is preserved)

Usage:
  $ xcaffold import`,
	Example: "  $ xcaffold import",
	RunE:    runImport,
}

func init() {
	rootCmd.AddCommand(importCmd)
}

func runImport(cmd *cobra.Command, args []string) error {
	if scopeFlag == "global" {
		return importScope(globalClaudeDir, globalXcfPath, "global")
	}
	// project (default)
	return importScope(".claude", "scaffold.xcf", "project")
}

// importScope scans a .claude/ directory at claudeDir and writes a xcf file to xcfDest.
func importScope(claudeDir, xcfDest, scopeName string) error {
	// If the target xcf already exists, abort.
	if _, err := os.Stat(xcfDest); err == nil {
		return fmt.Errorf("[%s] %s already exists. Remove it first to import", scopeName, xcfDest)
	}

	// For global scope, extracted instruction files live inside ~/.claude/imported/
	// so they remain co-located with the global config. For project scope they live
	// in the working directory alongside scaffold.xcf (existing behaviour).
	var extractBase string
	if scopeName == "global" {
		extractBase = filepath.Join(claudeDir, "imported")
	} else {
		extractBase = "."
	}

	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: ast.ProjectConfig{
			Name: "imported-project",
		},
		Agents: make(map[string]ast.AgentConfig),
		Skills: make(map[string]ast.SkillConfig),
		Rules:  make(map[string]ast.RuleConfig),
		Hooks:  make(ast.HookConfig),
		MCP:    make(map[string]ast.MCPConfig),
	}

	importCount := 0
	var warnings []string

	// 1. Import agents — extract content to agents/<id>.md, reference via instructions_file
	agentFiles, _ := filepath.Glob(filepath.Join(claudeDir, "agents", "*.md"))
	for _, f := range agentFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("skipping agent %s: %v", f, err))
			continue
		}
		id := strings.TrimSuffix(filepath.Base(f), ".md")
		if id == "" {
			continue
		}

		destPath := filepath.Join(extractBase, "agents", id+".md")
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("[%s] failed to create agents/ directory: %w", scopeName, err)
		}
		if err := os.WriteFile(destPath, data, 0600); err != nil {
			return fmt.Errorf("[%s] failed to write %s: %w", scopeName, destPath, err)
		}

		config.Agents[id] = ast.AgentConfig{
			Description:      "Imported agent",
			InstructionsFile: destPath,
		}
		importCount++
	}

	// 2. Import skills — Claude Code uses skills/<id>/SKILL.md directory structure
	skillFiles, _ := filepath.Glob(filepath.Join(claudeDir, "skills", "*", "SKILL.md"))
	for _, f := range skillFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("skipping skill %s: %v", f, err))
			continue
		}
		id := filepath.Base(filepath.Dir(f))
		if id == "" || id == "." {
			continue
		}

		destPath := filepath.Join(extractBase, "skills", id, "SKILL.md")
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("[%s] failed to create skills/%s/ directory: %w", scopeName, id, err)
		}
		if err := os.WriteFile(destPath, data, 0600); err != nil {
			return fmt.Errorf("[%s] failed to write %s: %w", scopeName, destPath, err)
		}

		// Copy reference files if a references/ subdirectory exists.
		refSrc := filepath.Join(filepath.Dir(f), "references")
		var refs []string
		if refEntries, err := os.ReadDir(refSrc); err == nil {
			for _, entry := range refEntries {
				if entry.IsDir() {
					continue
				}
				srcRef := filepath.Join(refSrc, entry.Name())
				dstRef := filepath.Join(extractBase, "skills", id, "references", entry.Name())
				if err := os.MkdirAll(filepath.Dir(dstRef), 0755); err != nil {
					return fmt.Errorf("[%s] failed to create references dir: %w", scopeName, err)
				}
				refData, err := os.ReadFile(srcRef)
				if err != nil {
					warnings = append(warnings, fmt.Sprintf("skipping reference %s: %v", srcRef, err))
					continue
				}
				if err := os.WriteFile(dstRef, refData, 0600); err != nil {
					return fmt.Errorf("[%s] failed to write reference %s: %w", scopeName, dstRef, err)
				}
				refs = append(refs, dstRef)
			}
		}

		config.Skills[id] = ast.SkillConfig{
			Description:      "Imported skill",
			InstructionsFile: destPath,
			References:       refs,
		}
		importCount++
	}

	// 3. Import rules — extract to rules/<id>.md
	ruleFiles, _ := filepath.Glob(filepath.Join(claudeDir, "rules", "*.md"))
	for _, f := range ruleFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("skipping rule %s: %v", f, err))
			continue
		}
		id := strings.TrimSuffix(filepath.Base(f), ".md")
		if id == "" {
			continue
		}

		destPath := filepath.Join(extractBase, "rules", id+".md")
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("[%s] failed to create rules/ directory: %w", scopeName, err)
		}
		if err := os.WriteFile(destPath, data, 0600); err != nil {
			return fmt.Errorf("[%s] failed to write %s: %w", scopeName, destPath, err)
		}

		config.Rules[id] = ast.RuleConfig{
			Description:      "Imported rule",
			InstructionsFile: destPath,
		}
		importCount++
	}

	// 4. Parse settings.json for MCP servers and settings.
	settingsPath := filepath.Join(claudeDir, "settings.json")
	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := importSettings(data, config, &importCount, &warnings); err != nil {
			warnings = append(warnings, fmt.Sprintf("settings.json partially imported: %v", err))
		}
	}

	// Generate scaffolding comment header + YAML.
	header := `# scaffold.xcf — generated by 'xcaffold import'
# Edit this file and run 'xcaffold apply' to manage your agent team.
# Each instructions_file: reference points to an external markdown file.
# Tip: run 'xcaffold graph' to visualize your agent topology.

`
	out, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("[%s] failed to encode xcf: %w", scopeName, err)
	}

	if err := os.WriteFile(xcfDest, append([]byte(header), out...), 0600); err != nil {
		return fmt.Errorf("[%s] failed to write %s: %w", scopeName, xcfDest, err)
	}

	fmt.Printf("[%s] ✓ Import complete. Created %s with %d resources.\n", scopeName, xcfDest, importCount)
	fmt.Println("  Instructions extracted to external files — full content preserved.")
	fmt.Println("  Run 'xcaffold apply' when ready to assume management.")
	if len(warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, w := range warnings {
			fmt.Println(" ⚠", w)
		}
	}
	return nil
}

// importSettings parses settings.json and populates MCP, rules, and settings.
func importSettings(data []byte, config *ast.XcaffoldConfig, count *int, warnings *[]string) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// MCP servers
	if mcpRaw, ok := raw["mcpServers"].(map[string]interface{}); ok {
		for id, serverRaw := range mcpRaw {
			serverMap, ok := serverRaw.(map[string]interface{})
			if !ok {
				continue
			}
			mc := ast.MCPConfig{}
			if cmdStr, ok := serverMap["command"].(string); ok {
				mc.Command = cmdStr
			}
			if argsRaw, ok := serverMap["args"].([]interface{}); ok {
				for _, a := range argsRaw {
					if argStr, ok := a.(string); ok {
						mc.Args = append(mc.Args, argStr)
					}
				}
			}
			config.MCP[id] = mc
			*count++
		}
	}

	// Settings block — statusLine (object), enabledPlugins (map), effortLevel
	settings := ast.SettingsConfig{}
	changed := false

	if slRaw, ok := raw["statusLine"].(map[string]interface{}); ok {
		settings.StatusLine = &ast.StatusLineConfig{}
		if t, ok := slRaw["type"].(string); ok {
			settings.StatusLine.Type = t
		}
		if c, ok := slRaw["command"].(string); ok {
			settings.StatusLine.Command = c
		}
		changed = true
	}

	if epRaw, ok := raw["enabledPlugins"].(map[string]interface{}); ok {
		settings.EnabledPlugins = make(map[string]bool)
		for k, v := range epRaw {
			if b, ok := v.(bool); ok {
				settings.EnabledPlugins[k] = b
			}
		}
		changed = true
	}

	if el, ok := raw["effortLevel"].(string); ok {
		settings.EffortLevel = el
		changed = true
	}
	if atk, ok := raw["alwaysThinkingEnabled"].(bool); ok {
		settings.AlwaysThinkingEnabled = &atk
		changed = true
	}

	if changed {
		config.Settings = settings
	}

	return nil
}
