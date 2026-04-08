package main

import (
	"fmt"
	"os"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/spf13/cobra"
)

var validateStructural bool

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate a scaffold.xcf configuration",
	Long: `Validate checks the scaffold.xcf file for correctness:

  - YAML syntax and known fields
  - Cross-reference integrity (agent -> skill/rule/MCP IDs exist)
  - Structural invariants (with --structural flag)

Exit code 0 means valid. Non-zero means errors found.`,
	Example: `  $ xcaffold validate
  $ xcaffold validate --structural`,
	RunE:          runValidate,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	validateCmd.Flags().BoolVar(&validateStructural, "structural", false, "run structural invariant checks (orphan resources, missing instructions)")
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	cfg, err := parser.ParseFile(xcfPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "validation failed: %v\n", err)
		return err
	}

	fmt.Fprintf(os.Stdout, "syntax and cross-references: ok\n")

	if validateStructural {
		warnings := runStructuralChecks(cfg)
		if len(warnings) > 0 {
			fmt.Fprintf(os.Stdout, "\nstructural warnings:\n")
			for _, w := range warnings {
				fmt.Fprintf(os.Stdout, "  - %s\n", w)
			}
		} else {
			fmt.Fprintf(os.Stdout, "structural checks: ok\n")
		}
	}

	fmt.Fprintf(os.Stdout, "\nvalidation passed\n")
	return nil
}

// runStructuralChecks performs non-fatal invariant checks on the config.
func runStructuralChecks(cfg *ast.XcaffoldConfig) []string {
	var warnings []string
	warnings = append(warnings, checkOrphanSkills(cfg)...)
	warnings = append(warnings, checkOrphanRules(cfg)...)
	warnings = append(warnings, checkMissingInstructions(cfg)...)
	warnings = append(warnings, checkBashWithoutHook(cfg)...)
	return warnings
}

func checkOrphanSkills(cfg *ast.XcaffoldConfig) []string {
	referenced := make(map[string]bool)
	for _, agent := range cfg.Agents {
		for _, s := range agent.Skills {
			referenced[s] = true
		}
	}
	var warnings []string
	for skillID := range cfg.Skills {
		if !referenced[skillID] {
			warnings = append(warnings, fmt.Sprintf("skill %q is defined but not referenced by any agent", skillID))
		}
	}
	return warnings
}

func checkOrphanRules(cfg *ast.XcaffoldConfig) []string {
	referenced := make(map[string]bool)
	for _, agent := range cfg.Agents {
		for _, r := range agent.Rules {
			referenced[r] = true
		}
	}
	var warnings []string
	for ruleID, rule := range cfg.Rules {
		if rule.AlwaysApply != nil && *rule.AlwaysApply {
			continue
		}
		if len(rule.Paths) > 0 {
			continue
		}
		if !referenced[ruleID] {
			warnings = append(warnings, fmt.Sprintf("rule %q is defined but not referenced by any agent and has no paths or alwaysApply", ruleID))
		}
	}
	return warnings
}

func checkMissingInstructions(cfg *ast.XcaffoldConfig) []string {
	var warnings []string
	for agentID, agent := range cfg.Agents {
		if agent.Instructions == "" && agent.InstructionsFile == "" {
			warnings = append(warnings, fmt.Sprintf("agent %q has no instructions or instructions_file", agentID))
		}
	}
	return warnings
}

func checkBashWithoutHook(cfg *ast.XcaffoldConfig) []string {
	_, projectHasPreToolUse := cfg.Hooks["PreToolUse"]
	var warnings []string
	for agentID, agent := range cfg.Agents {
		hasBash := false
		for _, tool := range agent.Tools {
			if tool == "Bash" {
				hasBash = true
				break
			}
		}
		if !hasBash {
			continue
		}
		_, agentHasPreToolUse := agent.Hooks["PreToolUse"]
		if !projectHasPreToolUse && !agentHasPreToolUse {
			warnings = append(warnings, fmt.Sprintf("agent %q has Bash tool but no PreToolUse hook for command validation", agentID))
		}
	}
	return warnings
}
