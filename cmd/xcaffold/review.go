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

	switch scopeFlag {
	case "global":
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("could not determine home directory: %w", err)
		}
		dir := filepath.Join(home, ".claude")
		if file == scopeAll {
			return reviewAllInDir(cmd, dir)
		}
		return reviewFile(cmd, filepath.Join(dir, file))
	case scopeAll:
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("could not determine home directory: %w", err)
		}
		globalDir := filepath.Join(home, ".claude")
		if file == scopeAll {
			cmd.Println("-- project scope --")
			if err := reviewAll(cmd); err != nil {
				return err
			}
			cmd.Println("-- global scope --")
			return reviewAllInDir(cmd, globalDir)
		}
		// Specific file: review in both project CWD and global dir.
		cmd.Println("-- project scope --")
		_ = reviewFile(cmd, file)
		cmd.Println("-- global scope --")
		return reviewFile(cmd, filepath.Join(globalDir, file))
	default:
		// "project" (default)
		if file == "all" {
			return reviewAll(cmd)
		}
		return reviewFile(cmd, file)
	}
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
	cmd.Printf("Project: %s (v%s)\n", manifest.Project.Name, manifest.Version)
	cmd.Println("=====================================")

	if len(manifest.Agents) > 0 {
		cmd.Println("\n-- AGENTS --")
		for id, agent := range manifest.Agents {
			modelStr := agent.Model
			if modelStr == "" {
				modelStr = "default model"
			}
			cmd.Printf(" 🤖 %s (%s)\n", id, modelStr)
			if agent.Description != "" {
				cmd.Printf("    Description: %s\n", agent.Description)
			}
			if len(agent.Tools) > 0 {
				cmd.Printf("    Tools:       %s\n", strings.Join(agent.Tools, ", "))
			}

			// We only count assertions if there's a test block? Wait in ast.go, Assertions is embedded directly on AgentConfig?
			// Let's check ast.go: Yes, `agent.Assertions` is NOT agent.Test.Assertions! Because `TestConfig` is root-level.
			if len(agent.Assertions) > 0 {
				cmd.Printf("    Assertions:  %d adversarial checks\n", len(agent.Assertions))
			}
			cmd.Println()
		}
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
