package main

import (
	"fmt"
	"os"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	translateFrom             string
	translateTo               string
	translateSourceDir        string
	translateOutputDir        string
	translateXcf              string
	translateSaveXcf          string
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
	translateCmd.Flags().StringVar(&translateSaveXcf, "save-xcf", "", "write the imported IR to this path as scaffold.xcf YAML")

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

	// Phase 1: Import — build an xcaffold IR from source material.
	config, err := translateImport()
	if err != nil {
		return err
	}

	// Persist IR to disk when --save-xcf is requested.
	if translateSaveXcf != "" {
		data, err := yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("marshalling xcf IR: %w", err)
		}
		if err := os.WriteFile(translateSaveXcf, data, 0o600); err != nil {
			return fmt.Errorf("writing --save-xcf %q: %w", translateSaveXcf, err)
		}
	}

	// A10 will add Phase 2 (compile/optimize) and Phase 3 (apply/diff/audit).
	// For A9, dry-run terminates here after confirming import succeeded.
	return nil
}

// translateImport loads the xcaffold IR for the translate pipeline.
//
// When --xcf is set, the IR is read directly from that file (skipping import).
// Otherwise, the source directory is scanned via importScope using --from as
// the platform hint, and the resulting config is returned as the IR.
func translateImport() (*ast.XcaffoldConfig, error) {
	if translateXcf != "" {
		config, err := parser.ParseFile(translateXcf)
		if err != nil {
			return nil, fmt.Errorf("loading --xcf %q: %w", translateXcf, err)
		}
		return config, nil
	}

	srcDir := translateSourceDir
	if srcDir == "" {
		var err error
		srcDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("resolving working directory: %w", err)
		}
	}

	return importFromSource(srcDir, translateFrom)
}

// importFromSource scans a provider source directory and returns an xcaffold
// IR config without writing any files to disk.
func importFromSource(sourceDir, _ string) (*ast.XcaffoldConfig, error) {
	config, err := buildConfigFromDir(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("scanning source directory %q: %w", sourceDir, err)
	}
	return config, nil
}
