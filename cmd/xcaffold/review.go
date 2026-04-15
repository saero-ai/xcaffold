package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/spf13/cobra"
)

var reviewCmd = &cobra.Command{
	Use:   "review [file]",
	Short: "Universally read and format generated diagnostic files natively in the terminal",
	Long: `xcaffold review acts as a universal parser for all diagnostic files.

┌───────────────────────────────────────────────────────────────────┐
│                        UNIVERSAL PARSER                           │
└───────────────────────────────────────────────────────────────────┘
 Use this command to pretty-print operational artifacts securely in the terminal.

 Supported Formats:
 • scaffold.xcf   -> Renders the AST tree structurally
 • audit.json     -> Visualizes the categorical greenfield/brownfield scores
 • plan.json      -> Displays compilation plan
 • trace.jsonl    -> Formats the execution trace from proxy runs

 Examples:
  $ xcaffold review all          (Loops all files automatically)
  $ xcaffold review audit.json   (Reviews specific artifact)`,
	Example: `  $ xcaffold review all
  $ xcaffold review audit.json
  $ xcaffold review plan.json`,
	Args: cobra.MaximumNArgs(1),
	RunE: runReview,
}

func init() {
	rootCmd.AddCommand(reviewCmd)
}

func runReview(cmd *cobra.Command, args []string) error {
	file := "scaffold.xcf"
	if len(args) > 0 {
		file = args[0]
	}

	if globalFlag {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("could not determine home directory: %w", err)
		}
		dir := filepath.Join(home, ".xcaffold")
		if file == "all" {
			return reviewAllInDir(cmd, dir)
		}
		return reviewFile(cmd, filepath.Join(dir, file))
	}

	// project (default)
	if file == "all" {
		return reviewAll(cmd)
	}
	return reviewFile(cmd, file)
}

func reviewAll(cmd *cobra.Command) error {
	return reviewAllInDir(cmd, "")
}

func reviewAllInDir(cmd *cobra.Command, dir string) error {
	targets := []string{"scaffold.xcf", "audit.json", "plan.json", "trace.jsonl"}
	found := false
	for _, target := range targets {
		path := target
		if dir != "" {
			path = filepath.Join(dir, target)
		}
		if _, err := os.Stat(path); err == nil {
			found = true
			if err := reviewFile(cmd, path); err != nil {
				cmd.Printf("Warning: error reviewing %s: %v\n", path, err)
			}
		}
	}
	if !found {
		if dir != "" {
			cmd.Printf("No diagnostic files found in %s.\n", dir)
		} else {
			cmd.Println("No diagnostic files found in the current directory.")
		}
	}
	return nil
}

func reviewFile(cmd *cobra.Command, file string) error {
	content, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", file, err)
	}

	if strings.HasSuffix(file, ".xcf") {
		return reviewXCF(cmd, content)
	} else if strings.HasSuffix(file, ".json") {
		return reviewJSON(cmd, content)
	} else if strings.HasSuffix(file, ".jsonl") {
		return reviewJSONL(cmd, content)
	}

	cmd.Printf("Unknown file type. Raw contents:\n%s\n", string(content))
	return nil
}

func reviewXCF(cmd *cobra.Command, content []byte) error {
	manifest, err := parser.Parse(bytes.NewReader(content))
	if err != nil {
		return err
	}

	cmd.Println("\n=== XCAFFOLD CONFIGURATION REVIEW ===")
	var projectName string
	if manifest.Project != nil {
		projectName = manifest.Project.Name
	}
	cmd.Printf("Project: %s (v%s)\n", projectName, manifest.Version)
	cmd.Println("=====================================")

	if len(manifest.Agents) > 0 {
		cmd.Println("\n-- AGENTS --")
		for id, agent := range manifest.Agents {
			modelStr := agent.Model
			if modelStr == "" {
				modelStr = "default model"
			}
			cmd.Printf("  %s (%s)\n", id, modelStr)
			if agent.Description != "" {
				cmd.Printf("    Description: %s\n", agent.Description)
			}
			if len(agent.Tools) > 0 {
				cmd.Printf("    Tools:       %s\n", strings.Join(agent.Tools, ", "))
			}
			if len(agent.Assertions) > 0 {
				cmd.Printf("    Assertions:  %d adversarial checks\n", len(agent.Assertions))
			}
			cmd.Println()
		}
	}

	if len(manifest.Skills) > 0 {
		cmd.Println("-- SKILLS --")
		for _, id := range sortedKeys(manifest.Skills) {
			skill := manifest.Skills[id]
			label := id
			if skill.Name != "" {
				label = skill.Name
			}
			cmd.Printf("  %s\n", label)
			if len(skill.AllowedTools) > 0 {
				cmd.Printf("    Tools: %s\n", strings.Join(skill.AllowedTools, ", "))
			}
		}
		cmd.Println()
	}

	if len(manifest.Rules) > 0 {
		cmd.Println("-- RULES --")
		for _, id := range sortedKeys(manifest.Rules) {
			rule := manifest.Rules[id]
			suffix := ""
			if rule.AlwaysApply != nil && *rule.AlwaysApply {
				suffix = " (always-apply)"
			}
			cmd.Printf("  %s%s\n", id, suffix)
		}
		cmd.Println()
	}

	if len(manifest.Hooks) > 0 {
		cmd.Println("-- HOOKS --")
		for _, event := range sortedKeys(manifest.Hooks) {
			groups := manifest.Hooks[event]
			total := 0
			for _, g := range groups {
				total += len(g.Hooks)
			}
			cmd.Printf("  %s: %d handler(s)\n", event, total)
		}
		cmd.Println()
	}

	if len(manifest.MCP) > 0 {
		cmd.Println("-- MCP SERVERS --")
		for _, id := range sortedKeys(manifest.MCP) {
			mcp := manifest.MCP[id]
			typeStr := mcp.Type
			if typeStr == "" {
				typeStr = "unknown"
			}
			cmd.Printf("  %s (%s)\n", id, typeStr)
		}
		cmd.Println()
	}

	if len(manifest.Workflows) > 0 {
		cmd.Println("-- WORKFLOWS --")
		for _, id := range sortedKeys(manifest.Workflows) {
			wf := manifest.Workflows[id]
			label := id
			if wf.Name != "" {
				label = wf.Name
			}
			cmd.Printf("  %s\n", label)
		}
		cmd.Println()
	}

	return nil
}

func reviewJSON(cmd *cobra.Command, content []byte) error {
	// Try parsing as audit.json
	var audit struct {
		Type     string `json:"type"`
		Feedback string `json:"feedback"`
		Scores   struct {
			Security         int `json:"security"`
			PromptQuality    int `json:"prompt_quality"`
			ToolRestrictions int `json:"tool_restrictions"`
		} `json:"scores"`
	}

	if err := json.Unmarshal(content, &audit); err == nil && audit.Type != "" {
		cmd.Println("\n=== XCAFFOLD COMPLIANCE AUDIT ===")
		cmd.Printf("Architecture Type: %s\n\n", strings.ToUpper(audit.Type))
		cmd.Println("  [ SCORES ]")
		cmd.Printf("  - Security:          %d/100\n", audit.Scores.Security)
		cmd.Printf("  - Prompt Quality:    %d/100\n", audit.Scores.PromptQuality)
		cmd.Printf("  - Tool Restrictions: %d/100\n\n", audit.Scores.ToolRestrictions)
		cmd.Println("  [ FEEDBACK ]")
		cmd.Printf("  %s\n\n", audit.Feedback)
		return nil
	}

	// Unmarshal as plan.json (dry-run output)
	var plan map[string]any
	if err := json.Unmarshal(content, &plan); err == nil {
		pretty, _ := json.MarshalIndent(plan, "", "  ")
		cmd.Println(string(pretty))
		return nil
	}

	return fmt.Errorf("failed to determine json schema for review")
}

func reviewJSONL(cmd *cobra.Command, content []byte) error {
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	cmd.Printf("\n=== XCAFFOLD TRACE LOG (%d events) ===\n", len(lines))
	for i, line := range lines {
		var event map[string]any
		_ = json.Unmarshal([]byte(line), &event)
		cmd.Printf(" [%d] %s -> %s\n", i+1, event["timestamp"], event["tool_name"])
	}
	cmd.Println()
	return nil
}
