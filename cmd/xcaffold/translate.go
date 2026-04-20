package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/compiler"
	"github.com/saero-ai/xcaffold/internal/optimizer"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/renderer"
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
	translateCmd.Flags().StringVar(&translateSaveXcf, "save-xcf", "", "write the imported IR to this path as project.xcf YAML")

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

	// Resolve base directory for the compiler. When a .xcf file was loaded
	// directly, use its parent. Otherwise use the source directory.
	var baseDir string
	if translateXcf != "" {
		baseDir = filepath.Dir(translateXcf)
	} else if translateSourceDir != "" {
		baseDir = translateSourceDir
	} else {
		var err error
		baseDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("resolving working directory: %w", err)
		}
	}

	// Phase 2: Compile + Optimize.
	out, compileNotes, err := compiler.Compile(config, baseDir, translateTo)
	if err != nil {
		return fmt.Errorf("compile error: %w", err)
	}

	opt := optimizer.New(translateTo)
	for _, pass := range translateOptimize {
		opt.AddPass(pass)
	}
	files, passNotes, err := opt.Run(out.Files)
	if err != nil {
		return fmt.Errorf("optimizer error: %w", err)
	}

	var allNotes []renderer.FidelityNote
	allNotes = append(allNotes, compileNotes...)
	allNotes = append(allNotes, passNotes...)

	// Phase 3: Apply.
	//
	// compiler.Compile returns paths relative to the provider output dir
	// (e.g. "rules/X.md", "skills/Y/SKILL.md"). Rewrite the map keys to
	// include the provider prefix (.claude, .agents, …) so that every path
	// used below — dry-run listing, diff, idempotent-check, and write —
	// reflects the final on-disk layout.
	//
	// When --output-dir is set the user controls the destination root; we
	// still prepend the provider subdir so files land at
	// <output-dir>/<prefix>/<relPath>. When --output-dir is unset we
	// fall back to <baseDir> and omit the extra prefix (the outDir already
	// incorporates it via compiler.OutputDir).
	prefix := compiler.OutputDir(translateTo)
	if translateOutputDir != "" {
		prefixed := make(map[string]string, len(files))
		for relPath, content := range files {
			prefixed[filepath.Join(prefix, relPath)] = content
		}
		files = prefixed
	}

	// Precedence: --dry-run > --diff > --idempotent-check > write.

	if translateDryRun {
		fmt.Printf("translate dry-run: %d file(s), %d note(s)\n", len(files), len(allNotes))
		for relPath := range files {
			fmt.Printf("  %s\n", relPath)
		}
		return nil
	}

	if translateDiff {
		return translateShowDiff(files, translateOutputDir, translateDiffFormat)
	}

	if translateIdempotent {
		changed, err := translateDiffFiles(files, translateOutputDir)
		if err != nil {
			return fmt.Errorf("idempotent-check: %w", err)
		}
		if changed > 0 {
			return fmt.Errorf("idempotent-check: %d file(s) would change", changed)
		}
		return nil
	}

	// Write files to disk.
	outDir := translateOutputDir
	if outDir == "" {
		outDir = filepath.Join(baseDir, prefix)
	}

	for relPath, content := range files {
		absPath := filepath.Clean(filepath.Join(outDir, relPath))
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			return fmt.Errorf("translate: mkdir %s: %w", filepath.Dir(absPath), err)
		}
		if err := os.WriteFile(absPath, []byte(content), 0o600); err != nil {
			return fmt.Errorf("translate: write %s: %w", absPath, err)
		}
	}

	// Audit output.
	if translateAuditOut != "" {
		audit := map[string]any{
			"audit-version": "1.0",
			"from":          translateFrom,
			"to":            translateTo,
			"notes":         allNotes,
			"file-count":    len(files),
			"timestamp":     time.Now().UTC().Format(time.RFC3339),
		}
		data, err := json.MarshalIndent(audit, "", "  ")
		if err != nil {
			return fmt.Errorf("translate: marshal audit: %w", err)
		}
		if err := os.WriteFile(translateAuditOut, data, 0o600); err != nil {
			return fmt.Errorf("translate: write audit-out %q: %w", translateAuditOut, err)
		}
	}

	return nil
}

// translateShowDiff prints a diff between compiled files and on-disk state.
func translateShowDiff(files map[string]string, outDir, format string) error {
	for relPath, content := range files {
		var existing string
		if outDir != "" {
			absPath := filepath.Clean(filepath.Join(outDir, relPath))
			data, err := os.ReadFile(absPath)
			if err == nil {
				existing = string(data)
			}
		}
		switch format {
		case "json":
			entry := map[string]string{"path": relPath, "before": existing, "after": content}
			data, _ := json.MarshalIndent(entry, "", "  ")
			fmt.Println(string(data))
		case "markdown":
			fmt.Printf("### %s\n```diff\n", relPath)
			diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
				A:        difflib.SplitLines(existing),
				B:        difflib.SplitLines(content),
				FromFile: relPath + " (current)",
				ToFile:   relPath + " (translated)",
				Context:  3,
			})
			fmt.Print(diff)
			fmt.Println("```")
		default: // unified
			diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
				A:        difflib.SplitLines(existing),
				B:        difflib.SplitLines(content),
				FromFile: relPath + " (current)",
				ToFile:   relPath + " (translated)",
				Context:  3,
			})
			if diff != "" {
				fmt.Print(diff)
			}
		}
	}
	return nil
}

// translateDiffFiles returns the number of files that differ between compiled
// output and on-disk state. Returns an error only on unexpected I/O failures.
func translateDiffFiles(files map[string]string, outDir string) (int, error) {
	changed := 0
	for relPath, content := range files {
		if outDir == "" {
			// No output dir set — treat every file as changed.
			changed++
			continue
		}
		absPath := filepath.Clean(filepath.Join(outDir, relPath))
		existing, err := os.ReadFile(absPath)
		if os.IsNotExist(err) {
			changed++
			continue
		}
		if err != nil {
			return 0, fmt.Errorf("reading %s: %w", absPath, err)
		}
		if strings.TrimRight(string(existing), "\n") != strings.TrimRight(content, "\n") {
			changed++
		}
	}
	return changed, nil
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

// providerScopeSubdir maps a provider name to its conventional config
// subdirectory within a project root. When a user passes --source-dir pointing
// at the project root, we descend into this subdir before scanning.
var providerScopeSubdir = map[string]string{
	"claude":      ".claude",
	"antigravity": ".agents",
	"cursor":      ".cursor",
	"gemini":      ".gemini",
	"copilot":     ".github",
}

// resolveScopeDir resolves the actual scope directory for a given provider and
// source directory. If the conventional subdirectory exists under sourceDir, it
// is returned; otherwise sourceDir is assumed to already point at the scope.
func resolveScopeDir(sourceDir, provider string) string {
	if sub, ok := providerScopeSubdir[provider]; ok {
		candidate := filepath.Join(sourceDir, sub)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return sourceDir
}

// importFromSource scans a provider source directory and returns an xcaffold
// IR config without writing any files to disk.
func importFromSource(sourceDir, fromProvider string) (*ast.XcaffoldConfig, error) {
	scopeDir := resolveScopeDir(sourceDir, fromProvider)
	config, err := buildConfigFromDir(scopeDir, fromProvider)
	if err != nil {
		return nil, fmt.Errorf("scanning source directory %q: %w", scopeDir, err)
	}
	return config, nil
}
