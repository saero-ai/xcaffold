package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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
	if scopeFlag == "global" || scopeFlag == "all" {
		if err := planScope(globalXcfPath, "global"); err != nil {
			return err
		}
	}
	if scopeFlag == "project" || scopeFlag == "all" {
		if err := planScope(xcfPath, "project"); err != nil {
			return err
		}
	}
	return nil
}

func planScope(configPath, scopeName string) error {
	config, err := parser.ParseFile(configPath)
	if err != nil {
		return fmt.Errorf("[%s] parse error: %w", scopeName, err)
	}

	fmt.Printf("[%s] Project: %s\n", scopeName, config.Project.Name)
	fmt.Printf("[%s] Agents:  %d\n\n", scopeName, len(config.Agents))

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
		fmt.Printf("[%s] Plan completed with warnings. Review the agents flagged above.\n", scopeName)
	} else {
		fmt.Printf("[%s] ✓ Plan completed. No bloat detected.\n", scopeName)
	}
	fmt.Println()
	fmt.Println("  Note: estimates cover inline instructions/description only.")
	fmt.Println("  Referenced skills and rules are loaded at runtime and may")
	fmt.Println("  increase actual context window usage by 10-100×.")

	// Write plan.json adjacent to the xcf file.
	planData := map[string]any{
		"project": config.Project.Name,
		"agents":  len(config.Agents),
		"tokens":  report,
		"bloat":   hasBloat,
	}
	planBytes, err := json.MarshalIndent(planData, "", "  ")
	if err != nil {
		return fmt.Errorf("[%s] failed to marshal plan.json: %w", scopeName, err)
	}
	planPath := filepath.Join(filepath.Dir(configPath), "plan.json")
	if err := os.WriteFile(planPath, planBytes, 0644); err != nil {
		return fmt.Errorf("[%s] failed to write %s: %w", scopeName, planPath, err)
	}
	fmt.Printf("  %s written.\n", planPath)

	return nil
}
