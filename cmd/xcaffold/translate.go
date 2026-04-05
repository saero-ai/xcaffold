package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/bir"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/translator"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	translateSource         string
	translateTarget         string
	translatePlan           bool
	translateSourcePlatform string
)

var translateCmd = &cobra.Command{
	Use:   "translate",
	Short: "Import and decompose agent definitions from other platforms",
	Long: `translate imports agent workflow files from other platforms and decomposes
them into xcaffold primitives (skills, rules, permissions) based on detected intents.

Each input file is analyzed with static intent detection — no LLM calls are made.
Detected intents determine which xcaffold primitive each block maps to:

  IntentProcedure  → skill       (sequential steps)
  IntentConstraint → rule        (directive keywords: MUST, NEVER, ALWAYS)
  IntentAutomation → permission  (// turbo annotations)

Results are injected into scaffold.xcf using instructions_file: references.
Run 'xcaffold apply' afterwards to render the updated config to .claude/.

Use --plan to preview the decomposition without writing any files.

Examples:
  xcaffold translate --source ./workflows/
  xcaffold translate --source ./deploy.md --target claude --plan`,
	RunE: runTranslate,
}

func init() {
	translateCmd.Flags().StringVar(&translateSource, "source", "", "File or directory of workflow markdown files to translate (required)")
	translateCmd.Flags().StringVar(&translateTarget, "target", "claude", "Target platform for output primitives")
	translateCmd.Flags().BoolVar(&translatePlan, "plan", false, "Dry-run: print decomposition plan without writing files")
	translateCmd.Flags().StringVar(&translateSourcePlatform, "source-platform", "gemini", "Source platform of input files (gemini, claude, cursor)")

	if err := translateCmd.MarkFlagRequired("source"); err != nil {
		// MarkFlagRequired only errors if the flag does not exist — unreachable.
		panic(err)
	}
}

func runTranslate(cmd *cobra.Command, args []string) error {
	// Locate and parse existing scaffold.xcf — translate requires an existing project.
	xcfPath := "scaffold.xcf"
	config, err := parser.ParseFile(xcfPath)
	if err != nil {
		return fmt.Errorf("no scaffold.xcf found — run 'xcaffold init' first, then 'xcaffold translate': %w", err)
	}

	// Resolve the base directory (same directory as scaffold.xcf) for writing
	// instructions_file paths relative to the xcf file, following compiler.Compile convention.
	xcfAbs, err := filepath.Abs(xcfPath)
	if err != nil {
		return fmt.Errorf("could not resolve scaffold.xcf path: %w", err)
	}
	baseDir := filepath.Dir(xcfAbs)

	sources, err := resolveSourceFiles(translateSource)
	if err != nil {
		return fmt.Errorf("--source %q: %w", translateSource, err)
	}

	if len(sources) == 0 {
		return fmt.Errorf("no .md files found at %q", translateSource)
	}

	var allResults []translator.TranslationResult
	totalPrimitives := 0

	for _, src := range sources {
		unit, err := bir.ImportWorkflow(src, translateSourcePlatform)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: skipping %s: %v\n", src, err)
			continue
		}

		result := translator.Translate(unit, translateTarget)
		allResults = append(allResults, result)

		fmt.Printf("\n%s\n", filepath.Base(src))
		fmt.Printf("  intents detected: %s\n", formatIntents(unit.Intents))
		fmt.Printf("  primitives:\n")

		for _, p := range result.Primitives {
			fmt.Printf("    [%s] %s\n", p.Kind, p.ID)
			totalPrimitives++
		}
	}

	fmt.Printf("\n%d file(s), %d primitive(s) total → target: %s\n",
		len(sources), totalPrimitives, translateTarget)

	if translatePlan {
		printTranslatePlan(allResults, baseDir)
		fmt.Println("(dry-run — no files written)")
		return nil
	}

	return injectIntoConfig(config, allResults, xcfPath, baseDir)
}

// injectIntoConfig writes external .md files for each primitive and updates
// scaffold.xcf with instructions_file: references, following the import.go pattern.
func injectIntoConfig(config *ast.XcaffoldConfig, results []translator.TranslationResult, xcfPath, baseDir string) error {
	if config.Skills == nil {
		config.Skills = make(map[string]ast.SkillConfig)
	}
	if config.Rules == nil {
		config.Rules = make(map[string]ast.RuleConfig)
	}

	// Collect allow entries across all permission primitives, deduplicating.
	seen := make(map[string]bool)
	var allowEntries []string

	for _, result := range results {
		for _, p := range result.Primitives {
			if strings.TrimSpace(p.Body) == "" {
				continue
			}

			switch p.Kind {
			case "skill":
				destPath := filepath.Join(baseDir, "skills", p.ID, "SKILL.md")
				if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
					return fmt.Errorf("failed to create skills/%s/ directory: %w", p.ID, err)
				}
				if err := os.WriteFile(destPath, []byte(p.Body), 0644); err != nil {
					return fmt.Errorf("failed to write %s: %w", destPath, err)
				}
				// Store path relative to baseDir for instructions_file portability.
				relPath := filepath.Join("skills", p.ID, "SKILL.md")
				config.Skills[p.ID] = ast.SkillConfig{
					Description:      fmt.Sprintf("Translated from workflow %s", p.ID),
					InstructionsFile: relPath,
				}
				fmt.Printf("  wrote %s\n", destPath)

			case "rule":
				destPath := filepath.Join(baseDir, "rules", p.ID+".md")
				if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
					return fmt.Errorf("failed to create rules/ directory: %w", err)
				}
				if err := os.WriteFile(destPath, []byte(p.Body), 0644); err != nil {
					return fmt.Errorf("failed to write %s: %w", destPath, err)
				}
				relPath := filepath.Join("rules", p.ID+".md")
				config.Rules[p.ID] = ast.RuleConfig{
					Description:      fmt.Sprintf("Constraints from workflow %s", p.ID),
					InstructionsFile: relPath,
				}
				fmt.Printf("  wrote %s\n", destPath)

			case "permission":
				for _, entry := range resolveAllowEntries(p.Body) {
					if !seen[entry] {
						seen[entry] = true
						allowEntries = append(allowEntries, entry)
					}
				}
			}
		}
	}

	// Merge permission allow entries into config.Settings.Permissions.
	if len(allowEntries) > 0 {
		if config.Settings.Permissions == nil {
			config.Settings.Permissions = &ast.PermissionsConfig{}
		}
		existing := make(map[string]bool, len(config.Settings.Permissions.Allow))
		for _, e := range config.Settings.Permissions.Allow {
			existing[e] = true
		}
		for _, entry := range allowEntries {
			if !existing[entry] {
				config.Settings.Permissions.Allow = append(config.Settings.Permissions.Allow, entry)
			}
		}
		fmt.Printf("  merged %d permission allow entries into settings.permissions\n", len(allowEntries))
	}

	// Marshal config back to scaffold.xcf, following the import.go pattern exactly.
	header := "# scaffold.xcf — updated by 'xcaffold translate'\n\n"
	out, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to encode scaffold.xcf: %w", err)
	}
	if err := os.WriteFile(xcfPath, append([]byte(header), out...), 0644); err != nil {
		return fmt.Errorf("failed to write scaffold.xcf: %w", err)
	}

	fmt.Printf("\nscaffold.xcf updated. Run 'xcaffold apply' to render to .claude/\n")
	return nil
}

// printTranslatePlan prints what would be injected without writing any files.
func printTranslatePlan(results []translator.TranslationResult, baseDir string) {
	fmt.Println("\n-- plan --")
	for _, result := range results {
		for _, p := range result.Primitives {
			switch p.Kind {
			case "skill":
				fmt.Printf("  skill  %q → skills/%s/SKILL.md\n", p.ID, p.ID)
			case "rule":
				fmt.Printf("  rule   %q → rules/%s.md\n", p.ID, p.ID)
			case "permission":
				entries := resolveAllowEntries(p.Body)
				fmt.Printf("  perm   %q → settings.permissions.allow: %v\n", p.ID, entries)
			}
		}
	}
	_ = baseDir // used for context, not needed in plan output
}

// resolveSourceFiles returns the list of .md files to process.
// If path is a directory, it returns all .md files directly within it (non-recursive).
// If path is a file, it returns a single-element slice containing that file.
func resolveSourceFiles(source string) ([]string, error) {
	abs, err := filepath.Abs(source)
	if err != nil {
		return nil, fmt.Errorf("could not resolve path: %w", err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path does not exist: %s", abs)
		}
		return nil, err
	}

	if !info.IsDir() {
		if !strings.HasSuffix(strings.ToLower(abs), ".md") {
			return nil, fmt.Errorf("source file must be a .md file, got: %s", filepath.Base(abs))
		}
		return []string{abs}, nil
	}

	entries, err := os.ReadDir(abs)
	if err != nil {
		return nil, fmt.Errorf("could not read directory: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			files = append(files, filepath.Join(abs, e.Name()))
		}
	}

	return files, nil
}

// formatIntents returns a human-readable summary of detected intent types.
func formatIntents(intents []bir.FunctionalIntent) string {
	if len(intents) == 0 {
		return "none (fallback: skill)"
	}

	seen := make(map[bir.IntentType]bool)
	var parts []string
	for _, intent := range intents {
		if !seen[intent.Type] {
			seen[intent.Type] = true
			parts = append(parts, string(intent.Type))
		}
	}

	return strings.Join(parts, ", ")
}

// resolveAllowEntries derives Bash permission entries from the primitive body.
// "turbo-all" and generic "turbo" annotations produce broad defaults.
func resolveAllowEntries(body string) []string {
	lower := strings.ToLower(body)
	if strings.Contains(lower, "turbo-all") || strings.Contains(lower, "turbo") {
		return []string{"Bash(git *)", "Bash(go *)"}
	}
	return []string{"Bash(*)"}
}
