package main

import (
	"fmt"
	"os"

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
    model: "claude-3-5-sonnet-20241022"
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

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap a new scaffold.xcf configuration",
	Long: `xcaffold init bootstraps the environment.

┌───────────────────────────────────────────────────────────────────┐
│                          BOOTSTRAP PHASE                          │
└───────────────────────────────────────────────────────────────────┘
 • Generates a blank scaffold.xcf configuration file in the current directory.
 • Provides the baseline schema for agent configuration engineering.

Ready to get started? Run:
  $ xcaffold init`,
	Example: "  $ xcaffold init",
	RunE:    runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	const filename = "scaffold.xcf"

	if _, err := os.Stat(filename); err == nil {
		return fmt.Errorf("%s already exists; delete it first if you want to re-initialize", filename)
	}

	if err := os.WriteFile(filename, []byte(defaultXCFContent), 0644); err != nil {
		return fmt.Errorf("failed to create %s: %w", filename, err)
	}

	fmt.Printf("✓ Created %s\n", filename)
	fmt.Println("  Edit it to define your agents, then run `xcaffold apply`.")
	fmt.Println("  Uncomment the `test:` block to configure the local simulator.")
	return nil
}
