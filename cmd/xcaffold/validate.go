package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/compiler"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/policy"
	"github.com/spf13/cobra"
)

var validateStructural bool

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate a scaffold.xcf configuration",
	Long: `Validate checks the scaffold.xcf file for correctness:

  - YAML syntax and known fields
  - Cross-reference integrity (agent -> skill/rule/MCP IDs exist)
  - File existence (instructions_file and skill references resolve on disk)
  - Plugin validation (enabledPlugins checked against known registry)
  - Structural invariants (with --structural flag)

Exit code 0 means valid. Non-zero means errors found.`,
	Example: `  $ xcaffold validate
  $ xcaffold validate --structural
  $ xcaffold validate --global`,
	RunE:          runValidate,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	validateCmd.Flags().BoolVar(&validateStructural, "structural", false, "run structural invariant checks (orphan resources, missing instructions)")
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	validatePath := xcfPath
	if globalFlag {
		// globalXcfPath is already resolved by resolveGlobalConfig in PersistentPreRunE.
		validatePath = globalXcfPath
	}

	cfg, err := parser.ParseFile(validatePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "validation failed: %v\n", err)
		return err
	}

	fmt.Fprintf(os.Stdout, "syntax and cross-references: ok\n")

	diags := parser.ValidateFile(validatePath)
	hasErrors := false
	if len(diags) > 0 {
		fmt.Fprintf(os.Stdout, "\ndiagnostics:\n")
		for _, d := range diags {
			fmt.Fprintf(os.Stdout, "  [%s] %s\n", d.Severity, d.Message)
			if d.Severity == "error" {
				hasErrors = true
			}
		}
	}
	if hasErrors {
		return fmt.Errorf("validation failed: one or more error diagnostics")
	}

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

	// Policy evaluation (requires compilation)
	if !hasErrors {
		configSnapshot := deepCopyConfig(cfg)
		compiled, notes, compileErr := compiler.Compile(cfg, filepath.Dir(validatePath), targetFlag)
		if compileErr != nil {
			fmt.Fprintf(os.Stdout, "\npolicy check skipped: compilation error: %v\n", compileErr)
		} else {
			printFidelityNotes(os.Stderr, notes, buildSuppressedResourcesMap(cfg, targetFlag), false)
			violations := policy.Evaluate(configSnapshot.Policies, configSnapshot, compiled)
			policyErrors := policy.FilterBySeverity(violations, policy.SeverityError)
			policyWarnings := policy.FilterBySeverity(violations, policy.SeverityWarning)

			if len(policyWarnings) > 0 {
				fmt.Fprintf(os.Stdout, "\n%s", policy.FormatViolations(policyWarnings))
			}
			if len(policyErrors) > 0 {
				fmt.Fprintf(os.Stdout, "\n%s", policy.FormatViolations(policyErrors))
				return fmt.Errorf("validation failed: %d policy error(s) found", len(policyErrors))
			}
			fmt.Fprintf(os.Stdout, "policies: ok\n")
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
