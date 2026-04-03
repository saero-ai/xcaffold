package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/saero-ai/xcaffold/internal/analyzer"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/spf13/cobra"
)

// tokenWarningThreshold is the estimated token count above which we warn
// that an agent's instructions may cause context window overflow.
const tokenWarningThreshold = 50_000

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Dry-run: analyse the .xcf config without writing files",
	Long: `xcaffold plan performs a static dry-run analysis of your agent topologies.

┌───────────────────────────────────────────────────────────────────┐
│                           TOKEN COST PHASE                        │
└───────────────────────────────────────────────────────────────────┘
 • Parses your scaffold.xcf AST map
 • Measures predicted token counts against model context-window thresholds
 • Identifies configuration bloat BEFORE you write to disk

Generated Artifacts:
 • plan.json   (The token math and budget analysis)`,
	Example: "  $ xcaffold plan",
	RunE:    runPlan,
}

func init() {
	rootCmd.AddCommand(planCmd)
}

func runPlan(cmd *cobra.Command, args []string) error {
	config, err := parser.ParseFile(xcfPath)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	fmt.Printf("Project: %s\n", config.Project.Name)
	fmt.Printf("Agents:  %d\n\n", len(config.Agents))

	a := analyzer.New()
	report := a.AnalyzeTokens(config)

	hasBloat := false
	for agentID, tokens := range report {
		fmt.Printf("  [~] %s: ~%d estimated tokens\n", agentID, tokens)
		if tokens > tokenWarningThreshold {
			fmt.Printf("       ⚠  WARNING: context window bloat detected (> %d tokens)\n", tokenWarningThreshold)
			hasBloat = true
		}
	}

	fmt.Println()
	if hasBloat {
		fmt.Println("Plan completed with warnings. Review the agents flagged above.")
	} else {
		fmt.Println("✓ Plan completed. No bloat detected.")
	}

	// Write plan.json to disk as documented.
	planData := map[string]any{
		"project": config.Project.Name,
		"agents":  len(config.Agents),
		"tokens":  report,
		"bloat":   hasBloat,
	}
	planBytes, err := json.MarshalIndent(planData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plan.json: %w", err)
	}
	if err := os.WriteFile("plan.json", planBytes, 0644); err != nil {
		return fmt.Errorf("failed to write plan.json: %w", err)
	}
	fmt.Println("  plan.json written.")

	return nil
}
