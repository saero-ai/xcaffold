package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/compiler"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/policy"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/state"
	"github.com/spf13/cobra"
)

var validateBlueprintFlag string

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Check .xcf syntax, cross-references, and structural invariants",
	Long: `Validate checks the project.xcf file for correctness:

  - YAML syntax and known fields
  - Cross-reference integrity (agent -> skill/rule/MCP IDs exist)
  - File existence (skill references resolve on disk)
  - Plugin validation (enabledPlugins checked against known registry)
  - Structural invariants (Bash tool without hook guard)

Exit code 0 means valid. Non-zero means errors found.`,
	Example: `  $ xcaffold validate
  $ xcaffold validate --global`,
	RunE:          runValidate,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	validateCmd.Flags().StringVar(&targetFlag, "target", "", "Validate field support for a specific provider target")
	validateCmd.Flags().StringVar(&validateBlueprintFlag, "blueprint", "", "Validate only the named blueprint")
	_ = validateCmd.Flags().MarkHidden("blueprint")
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	if globalFlag {
		return fmt.Errorf("global scope is not yet available")
	}

	validatePath := xcfPath

	parseRoot := filepath.Dir(validatePath)
	if filepath.Base(parseRoot) == ".xcaffold" {
		parseRoot = filepath.Dir(parseRoot)
	}

	// Header breadcrumb.
	projectName := filepath.Base(parseRoot)
	lastApplied := findLastApplied(parseRoot, validateBlueprintFlag)
	fmt.Println(formatHeader(projectName, validateBlueprintFlag, false, targetFlag, lastApplied))
	fmt.Println()

	cfg, crossRefIssues, err := parser.ParseDirectoryWithCrossRefWarnings(parseRoot, parser.WithSkipGlobal())
	if err != nil {
		fmt.Printf("  %s  syntax and schema\n", colorRed(glyphErr()))
		fmt.Println()
		fmt.Printf("%s  Validation failed: %v\n", colorRed(glyphErr()), err)
		return err
	}

	// Tiered output: syntax and schema pass
	fmt.Printf("  %s  syntax and schema\n", colorGreen(glyphOK()))

	// Cross-references: show as warnings, not errors
	if len(crossRefIssues) > 0 {
		fmt.Println()
		fmt.Println("  cross-references:")
		for _, issue := range crossRefIssues {
			fmt.Printf("    %s  %s\n", colorYellow(glyphSrc()), issue.Message)
		}
	}

	diags := parser.ValidateFile(validatePath)
	hasErrors := false
	if len(diags) > 0 {
		fmt.Println()
		fmt.Println("  diagnostics:")
		for _, d := range diags {
			g := colorYellow(glyphSrc())
			if d.Severity == "error" {
				g = colorRed(glyphErr())
				hasErrors = true
			}
			fmt.Printf("    %s  %s\n", g, d.Message)
		}
	}
	if hasErrors {
		fmt.Println()
		fmt.Printf("%s  Validation failed: diagnostics contain errors.\n", colorRed(glyphErr()))
		return fmt.Errorf("validation failed: one or more error diagnostics")
	}

	// Skill directory structures.
	xcfSkillsDir := filepath.Join(parseRoot, "xcf", "skills")
	if entries, dirErr := os.ReadDir(xcfSkillsDir); dirErr == nil {
		skillDirCount := 0
		skillDirHasIssues := false
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			skillDirCount++
			skillDir := filepath.Join(xcfSkillsDir, entry.Name())
			result := parser.ValidateSkillDirectory(skillDir, entry.Name())
			if len(result.Errors) > 0 || len(result.Warnings) > 0 {
				if !skillDirHasIssues {
					fmt.Println()
					fmt.Println("  skill directory issues:")
				}
				skillDirHasIssues = true
				for _, e := range result.Errors {
					fmt.Printf("    %s  %s: %s\n", colorRed(glyphErr()), entry.Name(), e)
				}
				for _, w := range result.Warnings {
					fmt.Printf("    %s  %s: %s\n", colorYellow(glyphSrc()), entry.Name(), w)
				}
			}
			if len(result.Errors) > 0 {
				hasErrors = true
			}
		}
		if skillDirCount > 0 && !skillDirHasIssues {
			fmt.Printf("  %s  skill directories\n", colorGreen(glyphOK()))
		}
	}

	// Hook directory structure.
	hookErrors := validateHooksDirectory(cfg, parseRoot)
	if len(hookErrors) > 0 {
		fmt.Println()
		fmt.Println("  hook directory issues:")
		for _, e := range hookErrors {
			fmt.Printf("    %s  %s\n", colorRed(glyphErr()), e)
		}
		hasErrors = true
	} else if len(cfg.Hooks) > 0 {
		fmt.Printf("  %s  hook directories\n", colorGreen(glyphOK()))
	}

	// Structural checks.
	structWarnings := runStructuralChecks(cfg)
	if len(structWarnings) > 0 {
		fmt.Println()
		fmt.Println("  structural warnings:")
		for _, w := range structWarnings {
			fmt.Printf("    %s  %s\n", colorYellow(glyphSrc()), w)
		}
	} else {
		fmt.Printf("  %s  structural checks\n", colorGreen(glyphOK()))
	}

	// Policy evaluation (requires compilation).
	var policyWarnings, policyErrors []policy.Violation
	policiesChecked := 0
	fieldValidationRan := false
	fieldValidationErrors := 0
	if !hasErrors {
		configSnapshot := deepCopyConfig(cfg)
		compiled, notes, compileErr := compiler.Compile(cfg, parseRoot, targetFlag, validateBlueprintFlag)
		if compileErr != nil {
			fmt.Printf("  %s  policies (skipped: compilation error)\n", colorYellow(glyphSrc()))
		} else {
			filteredNotes := renderer.FilterNotes(notes, buildSuppressedResourcesMap(cfg, targetFlag))
			printFidelityNotes(os.Stderr, filteredNotes, false)
			if targetFlag != "" {
				if err := checkFidelityErrors(filteredNotes); err != nil {
					fmt.Println()
					fmt.Printf("%s  Validation failed: %v\n", colorRed(glyphErr()), err)
					return fmt.Errorf("validation failed: %s", err)
				}
				fieldValidationRan = true
				for _, n := range filteredNotes {
					if n.Level == renderer.LevelError {
						fieldValidationErrors++
					}
				}
				fmt.Printf("  %s  field validation (%s)\n", colorGreen(glyphOK()), targetFlag)
			}
			violations := policy.Evaluate(configSnapshot.Policies, configSnapshot, compiled)
			policyErrors = policy.FilterBySeverity(violations, policy.SeverityError)
			policyWarnings = policy.FilterBySeverity(violations, policy.SeverityWarning)
			policiesChecked = len(policyErrors) + len(policyWarnings)
			if policiesChecked == 0 {
				policiesChecked = countBuiltinPolicies()
			}

			if len(policyWarnings) > 0 {
				fmt.Println()
				fmt.Println("  policy warnings:")
				for _, v := range policyWarnings {
					label := v.ResourceName
					if label == "" {
						label = v.FilePath
					}
					fmt.Printf("    %s  [%s] %s: %s\n", colorYellow(glyphSrc()), v.PolicyName, label, v.Message)
				}
			}

			if len(policyErrors) > 0 {
				fmt.Println()
				fmt.Println("  policy errors:")
				for _, v := range policyErrors {
					label := v.ResourceName
					if label == "" {
						label = v.FilePath
					}
					fmt.Printf("    %s  [%s] %s: %s\n", colorRed(glyphErr()), v.PolicyName, label, v.Message)
				}
				fmt.Println()
				fmt.Printf("%s  Validation failed: %d policy %s found.\n",
					colorRed(glyphErr()), len(policyErrors), plural(len(policyErrors), "error", "errors"))
				return fmt.Errorf("validation failed: %d policy error(s) found", len(policyErrors))
			}

			policyLabel := fmt.Sprintf("policies (%d checked", policiesChecked)
			if len(policyWarnings) > 0 {
				policyLabel += fmt.Sprintf(", %d %s", len(policyWarnings), plural(len(policyWarnings), "warning", "warnings"))
			}
			policyLabel += ")"
			fmt.Printf("  %s  %s\n", colorGreen(glyphOK()), policyLabel)
		}
	}

	if hasErrors {
		fmt.Println()
		fmt.Printf("%s  Validation failed: errors found.\n", colorRed(glyphErr()))
		return fmt.Errorf("validation failed: one or more errors found")
	}

	// Footer.
	xcfFileCount := countXcfFiles(parseRoot)
	totalWarnings := len(structWarnings) + len(policyWarnings) + len(crossRefIssues)
	fmt.Println()
	fieldSuffix := ""
	if fieldValidationRan {
		fieldSuffix = fmt.Sprintf("  Field validation: %s (%d %s).",
			targetFlag, fieldValidationErrors,
			plural(fieldValidationErrors, "error", "errors"))
	}
	if totalWarnings > 0 {
		fmt.Printf("%s  Validation passed with %d %s.  %d .xcf files checked.%s\n",
			colorGreen(glyphOK()), totalWarnings, plural(totalWarnings, "warning", "warnings"), xcfFileCount, fieldSuffix)
	} else {
		fmt.Printf("%s  Validation passed.  %d .xcf files checked.%s\n",
			colorGreen(glyphOK()), xcfFileCount, fieldSuffix)
	}
	return nil
}

func findLastApplied(baseDir, blueprint string) string {
	stPath := state.StateFilePath(baseDir, blueprint)
	manifest, err := state.ReadState(stPath)
	if err != nil {
		return ""
	}
	var best time.Time
	var result string
	for _, ts := range manifest.Targets {
		t, parseErr := time.Parse(time.RFC3339, ts.LastApplied)
		if parseErr == nil && t.After(best) {
			best = t
			result = ts.LastApplied
		}
	}
	return result
}

func countXcfFiles(root string) int {
	count := 0
	xcfDir := filepath.Join(root, "xcf")
	_ = filepath.WalkDir(xcfDir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && filepath.Ext(d.Name()) == ".xcf" {
			count++
		}
		return nil
	})
	return count
}

func countBuiltinPolicies() int {
	entries, err := os.ReadDir("internal/policy/builtin")
	if err != nil {
		return 4
	}
	return len(entries)
}

// runStructuralChecks performs non-fatal invariant checks on the config.
func runStructuralChecks(cfg *ast.XcaffoldConfig) []string {
	var warnings []string
	warnings = append(warnings, checkBashWithoutHook(cfg)...)
	return warnings
}

func checkBashWithoutHook(cfg *ast.XcaffoldConfig) []string {
	projectHasPreToolUse := false
	if dh, ok := cfg.Hooks["default"]; ok {
		_, projectHasPreToolUse = dh.Events["PreToolUse"]
	}
	var warnings []string
	for agentID, agent := range cfg.Agents {
		hasBash := false
		for _, tool := range agent.Tools.Values {
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
