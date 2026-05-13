package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/compiler"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/policy"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/state"
	"github.com/spf13/cobra"
)

var validateBlueprintFlag string
var validateVarFileFlag string

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Check .xcaf syntax, cross-references, and structural invariants",
	Long: `Validate checks the project.xcaf file for correctness:

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
	validateCmd.Flags().StringVar(&validateVarFileFlag, "var-file", "", "Load variables from a custom file")
	_ = validateCmd.Flags().MarkHidden("blueprint")
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	if globalFlag {
		return fmt.Errorf("global scope is not yet available")
	}

	parseRoot := deriveParseRoot(xcafPath)
	printValidateHeader(parseRoot)

	cfg, crossRefIssues, hasErrors, err := validateSyntax(parseRoot, xcafPath)
	if err != nil {
		return err
	}

	_ = validateSkillDirs(cfg, parseRoot, &hasErrors)
	hasErrors = reportHookErrors(cfg, parseRoot, hasErrors)
	structWarnings := reportStructuralWarnings(cfg)
	policyWarnings, _, fieldValidationRan, fieldValidationErrors, err := validatePolicies(hasErrors, cfg, parseRoot)
	if err != nil {
		return err
	}

	if hasErrors {
		fmt.Println()
		fmt.Printf("%s  Validation failed: errors found.\n", colorRed(glyphErr()))
		return fmt.Errorf("validation failed: one or more errors found")
	}

	printValidateSummary(parseRoot, structWarnings, policyWarnings, crossRefIssues,
		fieldValidationRan, fieldValidationErrors)
	return nil
}

// deriveParseRoot extracts the project root from the xcaf file path.
func deriveParseRoot(validatePath string) string {
	parseRoot := filepath.Dir(validatePath)
	if filepath.Base(parseRoot) == ".xcaffold" {
		parseRoot = filepath.Dir(parseRoot)
	}
	return parseRoot
}

// printValidateHeader prints the validation header.
func printValidateHeader(parseRoot string) {
	projectName := filepath.Base(parseRoot)
	lastApplied := findLastApplied(parseRoot, validateBlueprintFlag)
	fmt.Println(formatHeader(projectName, validateBlueprintFlag, false, targetFlag, lastApplied))
	fmt.Println()
}

// reportHookErrors validates and reports hook directory issues.
func reportHookErrors(cfg *ast.XcaffoldConfig, parseRoot string, hasErrors bool) bool {
	hookErrors := validateHooksDirectory(cfg, parseRoot)
	if len(hookErrors) > 0 {
		fmt.Println()
		fmt.Println("  hook directory issues:")
		for _, e := range hookErrors {
			fmt.Printf("    %s  %s\n", colorRed(glyphErr()), e)
		}
		return true
	}
	if len(cfg.Hooks) > 0 {
		fmt.Printf("  %s  hook directories\n", colorGreen(glyphOK()))
	}
	return hasErrors
}

// reportStructuralWarnings validates and reports structural warnings.
func reportStructuralWarnings(cfg *ast.XcaffoldConfig) []string {
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
	return structWarnings
}

// validateSyntax checks YAML syntax, schema, and cross-references.
func validateSyntax(parseRoot, validatePath string) (*ast.XcaffoldConfig, []parser.CrossReferenceIssue, bool, error) {
	cfg, crossRefIssues, err := parser.ParseDirectoryWithCrossRefWarnings(parseRoot, parser.WithSkipGlobal(), parser.WithVarFile(validateVarFileFlag))
	if err != nil {
		fmt.Printf("  %s  syntax and schema\n", colorRed(glyphErr()))
		fmt.Println()
		fmt.Printf("%s  Validation failed: %v\n", colorRed(glyphErr()), err)
		return nil, nil, false, err
	}

	// Tiered output: syntax and schema pass
	fmt.Printf("  %s  syntax and schema\n", colorGreen(glyphOK()))

	// Parse warnings: non-fatal issues during extraction
	if len(cfg.ParseWarnings) > 0 {
		fmt.Println()
		fmt.Println("  parse warnings:")
		for _, w := range cfg.ParseWarnings {
			fmt.Printf("    %s  %s\n", colorYellow(glyphSrc()), w)
		}
	}

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
		return nil, nil, true, fmt.Errorf("validation failed: one or more error diagnostics")
	}

	return cfg, crossRefIssues, false, nil
}

// validateSkillDirs checks skill directory structures and artifacts.
func validateSkillDirs(cfg *ast.XcaffoldConfig, parseRoot string, hasErrors *bool) []string {
	xcafSkillsDir := filepath.Join(parseRoot, "xcaf", "skills")
	if entries, dirErr := os.ReadDir(xcafSkillsDir); dirErr == nil {
		skillDirCount := 0
		skillDirHasIssues := false
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			skillDirCount++
			skillDir := filepath.Join(xcafSkillsDir, entry.Name())
			// Look up the skill's artifacts from the config
			var artifacts []string
			if skill, ok := cfg.Skills[entry.Name()]; ok {
				artifacts = skill.Artifacts
			}
			result := parser.ValidateSkillDirectory(skillDir, entry.Name(), artifacts)
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
				*hasErrors = true
			}
		}
		if skillDirCount > 0 && !skillDirHasIssues {
			fmt.Printf("  %s  skill directories\n", colorGreen(glyphOK()))
		}
	}
	return nil
}

// validatePolicies evaluates policies and compiles the config.
func validatePolicies(hasErrors bool, cfg *ast.XcaffoldConfig, parseRoot string) (
	[]policy.Violation, []policy.Violation, bool, int, error) {
	var policyWarnings, policyErrors []policy.Violation
	fieldValidationRan := false
	fieldValidationErrors := 0

	if hasErrors {
		return policyWarnings, policyErrors, fieldValidationRan, fieldValidationErrors, nil
	}

	configSnapshot := deepCopyConfig(cfg)
	compiled, notes, compileErr := compiler.Compile(cfg, parseRoot, targetFlag, validateBlueprintFlag, validateVarFileFlag)
	if compileErr != nil {
		fmt.Printf("  %s  policies (skipped: compilation error)\n", colorYellow(glyphSrc()))
		return policyWarnings, policyErrors, fieldValidationRan, fieldValidationErrors, nil
	}

	filteredNotes := renderer.FilterNotes(notes, buildSuppressedResourcesMap(cfg, targetFlag))
	printFidelityNotes(os.Stderr, filteredNotes, verboseFlag)

	var err error
	fieldValidationRan, fieldValidationErrors, err = validateFieldSupport(filteredNotes)
	if err != nil {
		return policyWarnings, policyErrors, fieldValidationRan, fieldValidationErrors, err
	}

	err = checkPolicyInvariants(configSnapshot, compiled)
	if err != nil {
		return policyWarnings, policyErrors, fieldValidationRan, fieldValidationErrors, err
	}

	violations := policy.Evaluate(configSnapshot.Policies, configSnapshot, compiled)
	policyErrors = policy.FilterBySeverity(violations, policy.SeverityError)
	policyWarnings = policy.FilterBySeverity(violations, policy.SeverityWarning)

	err = reportPolicyFindings(policyWarnings, policyErrors)
	if err != nil {
		return policyWarnings, policyErrors, fieldValidationRan, fieldValidationErrors, err
	}

	printPolicySummary(policyErrors, policyWarnings)
	return policyWarnings, policyErrors, fieldValidationRan, fieldValidationErrors, nil
}

// validateFieldSupport checks field support for the target provider.
func validateFieldSupport(filteredNotes []renderer.FidelityNote) (bool, int, error) {
	if targetFlag == "" {
		return false, 0, nil
	}

	if err := checkFidelityErrors(filteredNotes); err != nil {
		fmt.Println()
		fmt.Printf("%s  Validation failed: %v\n", colorRed(glyphErr()), err)
		return false, 0, fmt.Errorf("validation failed: %s", err)
	}

	fieldValidationErrors := 0
	for _, n := range filteredNotes {
		if n.Level == renderer.LevelError {
			fieldValidationErrors++
		}
	}
	fmt.Printf("  %s  field validation (%s)\n", colorGreen(glyphOK()), targetFlag)
	return true, fieldValidationErrors, nil
}

// checkPolicyInvariants validates security invariants.
func checkPolicyInvariants(cfg *ast.XcaffoldConfig, compiled *output.Output) error {
	errs := policy.RunInvariants(cfg, compiled)
	if len(errs) == 0 {
		return nil
	}

	fmt.Println()
	fmt.Println("  security invariant errors:")
	for _, e := range errs {
		fmt.Printf("    %s  %s\n", colorRed(glyphErr()), e)
	}
	fmt.Println()
	fmt.Printf("%s  Validation failed: %d security invariant(s) violated.\n",
		colorRed(glyphErr()), len(errs))
	return fmt.Errorf("validation failed: %d security invariant(s) violated", len(errs))
}

// reportPolicyFindings prints policy warnings and errors.
func reportPolicyFindings(warnings, errors []policy.Violation) error {
	if len(warnings) > 0 && verboseFlag {
		fmt.Println()
		fmt.Println("  policy warnings:")
		for _, v := range warnings {
			label := v.ResourceName
			if label == "" {
				label = v.FilePath
			}
			fmt.Printf("    %s  [%s] %s: %s\n", colorYellow(glyphSrc()), v.PolicyName, label, v.Message)
		}
	}

	if len(errors) == 0 {
		return nil
	}

	fmt.Println()
	fmt.Println("  policy errors:")
	for _, v := range errors {
		label := v.ResourceName
		if label == "" {
			label = v.FilePath
		}
		fmt.Printf("    %s  [%s] %s: %s\n", colorRed(glyphErr()), v.PolicyName, label, v.Message)
	}
	fmt.Println()
	fmt.Printf("%s  Validation failed: %d policy %s found.\n",
		colorRed(glyphErr()), len(errors), plural(len(errors), "error", "errors"))
	return fmt.Errorf("validation failed: %d policy error(s) found", len(errors))
}

// printPolicySummary prints the policy validation summary.
func printPolicySummary(errors, warnings []policy.Violation) {
	policiesChecked := len(errors) + len(warnings)
	var policyLabel string
	if policiesChecked == 0 {
		policyLabel = "policies (none configured)"
	} else {
		policyLabel = fmt.Sprintf("policies (%d checked", policiesChecked)
		if len(warnings) > 0 {
			policyLabel += fmt.Sprintf(", %d %s", len(warnings), plural(len(warnings), "warning", "warnings"))
		}
		policyLabel += ")"
	}
	fmt.Printf("  %s  %s\n", colorGreen(glyphOK()), policyLabel)
}

// printValidateSummary outputs the footer summary.
func printValidateSummary(parseRoot string, structWarnings []string, policyWarnings []policy.Violation,
	crossRefIssues []parser.CrossReferenceIssue, fieldValidationRan bool, fieldValidationErrors int) {
	xcafFileCount := countXcafFiles(parseRoot)
	totalWarnings := len(structWarnings) + len(policyWarnings) + len(crossRefIssues)
	fmt.Println()
	fieldSuffix := ""
	if fieldValidationRan {
		fieldSuffix = fmt.Sprintf("  Field validation: %s (%d %s).",
			targetFlag, fieldValidationErrors,
			plural(fieldValidationErrors, "error", "errors"))
	}
	if totalWarnings > 0 {
		fmt.Printf("%s  Validation passed with %d %s.  %d .xcaf files checked.%s\n",
			colorGreen(glyphOK()), totalWarnings, plural(totalWarnings, "warning", "warnings"), xcafFileCount, fieldSuffix)
	} else {
		fmt.Printf("%s  Validation passed.  %d .xcaf files checked.%s\n",
			colorGreen(glyphOK()), xcafFileCount, fieldSuffix)
	}
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

func countXcafFiles(root string) int {
	count := 0
	xcafDir := filepath.Join(root, "xcaf")
	_ = filepath.WalkDir(xcafDir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && filepath.Ext(d.Name()) == ".xcaf" {
			count++
		}
		return nil
	})
	return count
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
