package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// defaultXCFContent is the starter template written by `xcaffold init`.
// It includes all top-level blocks as comments so users can see the full schema.
const defaultXCFContent = `version: "1.0"
project:
  name: "new-xcaffold-project"
  description: "A new AI engineering project."

agents:
  developer:
    description: "General software developer agent."
    instructions: |
      You are a software developer.
      Write clean, maintainable code.
    model: "claude-3-7-sonnet-20250219"
    effort: "high"
    tools: [Bash, Read, Write, Edit, Glob, Grep]

    # Assertions are evaluated by the LLM-as-a-Judge when running
    # 'xcaffold test --judge'. Define expected behavioral constraints here.
    # assertions:
    #   - "The agent must not write files outside the project directory."
    #   - "The agent must run tests before marking a task complete."

# Optional: Configure the 'xcaffold test' simulator.
# test:
#   claude_path: ""                          # Path to claude binary. Defaults to 'claude' on $PATH.
#   judge_model: "claude-haiku-4-5-20251001" # Model used for --judge evaluation.
`

// defaultGlobalXCFContent is the starter template written by `xcaffold init --scope global`.
const defaultGlobalXCFContent = `version: "1.0"
project:
  name: "global"
  description: "User-wide agent configuration."

# Agents defined here are available across all projects.
# Project-level scaffold.xcf can override these with 'extends: global'.
agents:
  developer:
    description: "Default developer agent."
    instructions: |
      You are a software developer.
      Write clean, maintainable code.
    model: "claude-sonnet-4-6"
    effort: "high"
    tools: [Bash, Read, Write, Edit, Glob, Grep]
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap a new scaffold.xcf configuration",
	Long: `xcaffold init bootstraps the environment.

+-------------------------------------------------------------------+
|                          BOOTSTRAP PHASE                          |
+-------------------------------------------------------------------+
 • Generates a scaffold.xcf configuration file in the current directory.
 • Use --scope global to create a user-wide global.xcf in ~/.claude/.
 • Provides the baseline schema for agent configuration engineering.

Ready to get started? Run:
  $ xcaffold init`,
	Example: `  $ xcaffold init
  $ xcaffold init --scope global`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	if scopeFlag == "global" {
		return initGlobal()
	}
	return initProject()
}

func initProject() error {
	const filename = "scaffold.xcf"
	if _, err := os.Stat(filename); err == nil {
		return fmt.Errorf("%s already exists; delete it first if you want to re-initialize", filename)
	}
	if err := os.WriteFile(filename, []byte(defaultXCFContent), 0600); err != nil {
		return fmt.Errorf("failed to create %s: %w", filename, err)
	}
	fmt.Printf("Created %s\n", filename)
	fmt.Println("  Edit it to define your agents, then run `xcaffold apply`.")
	return nil
}

func initGlobal() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine home directory: %w", err)
	}
	dir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create %s: %w", dir, err)
	}
	target := filepath.Join(dir, "global.xcf")
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("%s already exists; delete it first if you want to re-initialize", target)
	}
	if err := os.WriteFile(target, []byte(defaultGlobalXCFContent), 0600); err != nil {
		return fmt.Errorf("failed to create %s: %w", target, err)
	}
	fmt.Printf("Created %s\n", target)
	fmt.Println("  Edit it to define your global agents, then run `xcaffold apply --scope global`.")
	fmt.Println("  Projects can inherit with 'extends: global' in their scaffold.xcf.")
	return nil
}
