package main

import (
	"fmt"
	"os"

	"github.com/saero-ai/xcaffold/internal/analyzer"
	"github.com/saero-ai/xcaffold/internal/auth"
	"github.com/saero-ai/xcaffold/internal/generator"
	"github.com/spf13/cobra"
)

var analyzeModel string

var analyzeCmd = &cobra.Command{
	Use:   "analyze [directory]",
	Short: "Analyze the repository and generate a project.xcf",
	Long: `xcaffold analyze reverse-engineers the current repository and builds an intelligent agent configuration blueprint.

┌───────────────────────────────────────────────────────────────────┐
│                              AUDIT PHASE                          │
└───────────────────────────────────────────────────────────────────┘
 1. 🔍 Scans the local directory, ignoring bloat (e.g. node_modules, .git)
 2. 🗜️ Computes a core ProjectSignature representing the exact architecture
 3. 🧠 Prompts Anthropic to generate an adversarial-ready project.xcf
 4. ⚖️ Outputs an audit.json compliance assessment

Generated Artifacts:
 • project.xcf   (The generated deterministic configuration)
 • audit.json     (The LLM-as-a-judge compliance scores)`,
	Example: `  $ xcaffold analyze
  $ xcaffold analyze ./path/to/project
  $ xcaffold analyze -m claude-3-haiku-20240307`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAnalyze,
}

func init() {
	analyzeCmd.Flags().StringVarP(&analyzeModel, "model", "m", "claude-3-7-sonnet-20250219", "The generative model to use")
	rootCmd.AddCommand(analyzeCmd)
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	cmd.Printf("🔍 Scanning directory: %s\n", dir)
	fsys := os.DirFS(dir)
	sig, err := analyzer.ScanProject(fsys)
	if err != nil {
		return fmt.Errorf("failed to scan project: %w", err)
	}

	cmd.Printf("   Found %d core structure files and %d dependency manifests.\n", len(sig.Files), len(sig.DependencyManifests))

	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	genericAPIKey := os.Getenv("XCAFFOLD_LLM_API_KEY")
	genericAPIBase := os.Getenv("XCAFFOLD_LLM_BASE_URL")

	gen, err := generator.New(anthropicKey, genericAPIKey, genericAPIBase, analyzeModel, "", nil)
	if err != nil {
		return fmt.Errorf("failed to initialize generator: %w", err)
	}

	authMsg := "Generative LLM API Key"
	if gen.AuthMode() == auth.AuthModeGenericAPI {
		authMsg = "Platform-Agnostic LLM API"
	} else if gen.AuthMode() == auth.AuthModeAPIKey {
		authMsg = "Target Provider API Key"
	} else if gen.AuthMode() == auth.AuthModeSubscription {
		authMsg = "Platform Subscription (fallback via local CLI config)"
		cmd.Println("   Note: Generation may display an external CLI spinner briefly.")
	}

	cmd.Printf("🧠 Generating project.xcf using %s via %s...\n", analyzeModel, authMsg)

	res, err := gen.Generate(cmd.Context(), sig)
	if err != nil {
		return fmt.Errorf("generation failed: %w", err)
	}

	outPath := "project.xcf" // nolint:goconst
	if err := os.WriteFile(outPath, []byte(res.YAMLConfig), 0600); err != nil {
		return fmt.Errorf("failed to write project.xcf: %w", err)
	}

	auditPath := "audit.json"
	if err := os.WriteFile(auditPath, []byte(res.AuditJSON), 0600); err != nil {
		return fmt.Errorf("failed to write audit.json: %w", err)
	}

	cmd.Printf("✅ Successfully wrote %s\n", outPath)
	cmd.Printf("✅ Successfully wrote %s\n", auditPath)
	cmd.Println("\nRun `xcaffold review audit.json` to read the compliance assessment.")
	return nil
}
