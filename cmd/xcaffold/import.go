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
 • Scans .claude/agents/*.md into agent configurations
 • Scans .claude/skills/*.md into skill configurations
 • Reads .claude/settings.json context rules
 • Produces a new scaffold.xcf

Usage:
  $ xcaffold import`,
	Example: "  $ xcaffold import",
	RunE:    runImport,
}

func init() {
	rootCmd.AddCommand(importCmd)
}

func runImport(cmd *cobra.Command, args []string) error {
	// If a scaffold.xcf already exists, we should abort unless forced.
	// For simplicity in this implementation, we just warn and abort.
	if _, err := os.Stat("scaffold.xcf"); err == nil {
		return fmt.Errorf("scaffold.xcf already exists. Remove it first to import.")
	}

	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: ast.ProjectConfig{
			Name: "imported-project",
		},
		Agents: make(map[string]ast.AgentConfig),
		Skills: make(map[string]ast.SkillConfig),
		Rules:  make(map[string]ast.RuleConfig),
		Hooks:  make(map[string]ast.HookConfig),
		MCP:    make(map[string]ast.MCPConfig),
	}

	importCount := 0

	// 1. Import agents
	agentFiles, _ := filepath.Glob(filepath.Join(".claude", "agents", "*.md"))
	for _, f := range agentFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		id := strings.TrimSuffix(filepath.Base(f), ".md")
		agent := ast.AgentConfig{
			Instructions: strings.TrimSpace(string(data)),
			Description:  "Imported agent",
		}
		config.Agents[id] = agent
		importCount++
	}

	// 2. Import skills
	skillFiles, _ := filepath.Glob(filepath.Join(".claude", "skills", "*.md"))
	for _, f := range skillFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		id := strings.TrimSuffix(filepath.Base(f), ".md")
		skill := ast.SkillConfig{
			Instructions: strings.TrimSpace(string(data)),
			Description:  "Imported skill",
		}
		config.Skills[id] = skill
		importCount++
	}

	// 3. Attempt to read settings.json for rules/mcp
	settingsPath := filepath.Join(".claude", "settings.json")
	if data, err := os.ReadFile(settingsPath); err == nil {
		var settings map[string]interface{}
		if err := json.Unmarshal(data, &settings); err == nil {
			// Extract custom prompt rules if they exist
			if customRules, ok := settings["customRules"].([]interface{}); ok {
				for i, r := range customRules {
					ruleMap, ok := r.(map[string]interface{})
					if !ok {
						continue
					}
					ruleID := fmt.Sprintf("imported-rule-%d", i+1)
					rc := ast.RuleConfig{}
					if instr, ok := ruleMap["rules"].(string); ok {
						rc.Instructions = instr
					}
					if paths, ok := ruleMap["paths"].([]interface{}); ok {
						for _, p := range paths {
							if pathStr, ok := p.(string); ok {
								rc.Paths = append(rc.Paths, pathStr)
							}
						}
					}
					config.Rules[ruleID] = rc
					importCount++
				}
			}
			// Extract MCP servers
			if mcpServers, ok := settings["mcpServers"].(map[string]interface{}); ok {
				for id, serverRaw := range mcpServers {
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
					importCount++
				}
			}
		}
	}

	// Generate the YAML
	out, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to encode scaffold.xcf: %w", err)
	}

	if err := os.WriteFile("scaffold.xcf", out, 0644); err != nil {
		return fmt.Errorf("failed to write scaffold.xcf: %w", err)
	}

	fmt.Printf("✓ Import complete. Created scaffold.xcf with %d imported resources.\n", importCount)
	fmt.Println("  Run 'xcaffold apply' when ready to assume management.")

	return nil
}
