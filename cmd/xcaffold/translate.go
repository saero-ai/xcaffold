package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	translateFrom             string
	translateTo               string
	translateSourceDir        string
	translateOutputDir        string
	translateXcf              string
	translateSaveXcf          bool
	translateFidelity         string
	translateInstructionsMode string
	translateIncludeMemory    bool
	translateWithMemory       bool
	translateScope            string
	translateVariant          string
	translateAutoMerge        bool
	translateDryRun           bool
	translateDiff             bool
	translateDiffFormat       string
	translateDiffOnly         bool
	translateRegenerate       bool
	translateReseed           bool
	translateIdempotent       bool
	translateAuditOut         string
	translateOptimize         []string
	translateOptimizeTarget   string
)

var translateCmd = &cobra.Command{
	Use:   "translate",
	Short: "Translate agent configurations between provider formats",
	Long: `xcaffold translate converts agent configurations between different provider formats
while maintaining fidelity and structural integrity.

Supported providers:
  • claude    - Claude Code workspace format
  • cursor    - Cursor IDE agent format
  • antigravity - Antigravity platform format
  • copilot   - GitHub Copilot format
  • gemini    - Google Gemini CLI format

Fidelity modes:
  • strict  - Fail on any information loss
  • warn    - Warn on information loss (default)
  • lossy   - Allow information loss silently

Diff formats:
  • unified   - Unified diff format (default)
  • json      - JSON structure diff
  • markdown  - Markdown-formatted diff`,
	Example: `  $ xcaffold translate --from claude --to cursor
  $ xcaffold translate --from antigravity --to cursor --fidelity strict
  $ xcaffold translate --from claude --to cursor --diff --diff-format json`,
	RunE: runTranslate,
}

func init() {
	// Required flags
	translateCmd.Flags().StringVar(&translateFrom, "from", "", "source provider format (required)")
	translateCmd.Flags().StringVar(&translateTo, "to", "", "target provider format (required)")

	// Input/output paths
	translateCmd.Flags().StringVar(&translateSourceDir, "source-dir", "", "source directory containing agent configs")
	translateCmd.Flags().StringVar(&translateOutputDir, "output-dir", "", "output directory for translated configs")
	translateCmd.Flags().StringVar(&translateXcf, "xcf", "", "xcaffold config file path")
	translateCmd.Flags().BoolVar(&translateSaveXcf, "save-xcf", false, "save translated config as scaffold.xcf")

	// Fidelity and translation modes
	translateCmd.Flags().StringVar(&translateFidelity, "fidelity", "warn", "fidelity mode: strict, warn, or lossy")
	translateCmd.Flags().StringVar(&translateInstructionsMode, "instructions-mode", "", "how to handle instructions: inline, file, or mixed")

	// Memory handling
	translateCmd.Flags().BoolVar(&translateIncludeMemory, "include-memory", false, "include memory sections in translation")
	translateCmd.Flags().BoolVar(&translateWithMemory, "with-memory", false, "preserve memory metadata (alias for --include-memory)")

	// Scope and targeting
	translateCmd.Flags().StringVar(&translateScope, "scope", "", "scope of translation: project, global, or both")
	translateCmd.Flags().StringVar(&translateVariant, "variant", "", "configuration variant to translate")
	translateCmd.Flags().BoolVar(&translateAutoMerge, "auto-merge", false, "automatically merge compatible configurations")

	// Dry-run and diff options
	translateCmd.Flags().BoolVar(&translateDryRun, "dry-run", false, "show what would be translated without writing files")
	translateCmd.Flags().BoolVar(&translateDiff, "diff", false, "show diff between source and target format")
	translateCmd.Flags().StringVar(&translateDiffFormat, "diff-format", "unified", "diff output format: unified, json, or markdown")
	translateCmd.Flags().BoolVar(&translateDiffOnly, "diff-only", false, "only show diff, do not write files")

	// Regeneration and validation
	translateCmd.Flags().BoolVar(&translateRegenerate, "regenerate", false, "regenerate all metadata during translation")
	translateCmd.Flags().BoolVar(&translateReseed, "reseed", false, "reseed deterministic IDs")
	translateCmd.Flags().BoolVar(&translateIdempotent, "idempotent-check", false, "verify translation is idempotent")

	// Audit and optimization
	translateCmd.Flags().StringVar(&translateAuditOut, "audit-out", "", "write audit report to file")
	translateCmd.Flags().StringArrayVar(&translateOptimize, "optimize", nil, "optimization rules to apply (repeatable)")
	translateCmd.Flags().StringVar(&translateOptimizeTarget, "optimize-target", "", "target optimization metric")

	rootCmd.AddCommand(translateCmd)
}

func runTranslate(cmd *cobra.Command, args []string) error {
	// Validate required flags
	if translateFrom == "" {
		return fmt.Errorf("required flag --from is not set")
	}
	if translateTo == "" {
		return fmt.Errorf("required flag --to is not set")
	}

	// Validate provider names
	validProviders := map[string]bool{
		"claude":      true,
		"cursor":      true,
		"antigravity": true,
		"copilot":     true,
		"gemini":      true,
	}

	if !validProviders[translateFrom] {
		return fmt.Errorf("invalid --from provider %q: must be one of claude, cursor, antigravity, copilot, gemini", translateFrom)
	}

	if !validProviders[translateTo] {
		return fmt.Errorf("invalid --to provider %q: must be one of claude, cursor, antigravity, copilot, gemini", translateTo)
	}

	// Validate fidelity mode
	validFidelity := map[string]bool{
		"strict": true,
		"warn":   true,
		"lossy":  true,
		"":       true, // empty is OK (uses default)
	}

	if !validFidelity[translateFidelity] {
		return fmt.Errorf("invalid --fidelity value %q: must be one of strict, warn, or lossy", translateFidelity)
	}

	// Validate diff format if --diff is set
	if translateDiff {
		validDiffFormat := map[string]bool{
			"unified":  true,
			"json":     true,
			"markdown": true,
		}

		if !validDiffFormat[translateDiffFormat] {
			return fmt.Errorf("invalid --diff-format value %q: must be one of unified, json, or markdown", translateDiffFormat)
		}
	}

	// A8 stub: validation only, no implementation yet
	return nil
}
