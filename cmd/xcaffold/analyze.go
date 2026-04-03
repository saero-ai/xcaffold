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
	Short: "Analyze the repository and generate a scaffold.xcf",
	Long: `xcaffold analyze reverse-engineers the current repository and builds an intelligent agent configuration blueprint.

┌───────────────────────────────────────────────────────────────────┐
│                              AUDIT PHASE                          │
└───────────────────────────────────────────────────────────────────┘
 1. 🔍 Scans the local directory, ignoring bloat (e.g. node_modules, .git)
 2. 🗜️ Computes a core ProjectSignature representing the exact architecture
 3. 🧠 Prompts Anthropic to generate an adversarial-ready scaffold.xcf 
 4. ⚖️ Outputs an audit.json compliance assessment

Generated Artifacts:
 • scaffold.xcf   (The generated deterministic configuration)
 • audit.json     (The LLM-as-a-judge compliance scores)`,
	Example: `  $ xcaffold analyze
  $ xcaffold analyze ./path/to/project
  $ xcaffold analyze -m claude-3-haiku-20240307`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAnalyze,
}

func init() {
	analyzeCmd.Flags().StringVarP(&analyzeModel, "model", "m", "claude-3-5-sonnet-20241022", "The generative model to use")
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

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	gen := generator.New(apiKey, analyzeModel, "", nil)

	authMsg := "Anthropic API Key"
	if gen.AuthMode() == auth.AuthModeSubscription {
		authMsg = "Claude Code Subscription (fallback via claude CLI)"
		cmd.Println("   Note: Generation may display the Claude CLI spinner briefly.")
	}

	cmd.Printf("🧠 Generating scaffold.xcf using %s via %s...\n", analyzeModel, authMsg)

	res, err := gen.Generate(sig)
	if err != nil {
		return fmt.Errorf("generation failed: %w", err)
	}

	outPath := "scaffold.xcf"
	if err := os.WriteFile(outPath, []byte(res.YAMLConfig), 0644); err != nil {
		return fmt.Errorf("failed to write scaffold.xcf: %w", err)
	}

	auditPath := "audit.json"
	if err := os.WriteFile(auditPath, []byte(res.AuditJSON), 0644); err != nil {
		return fmt.Errorf("failed to write audit.json: %w", err)
	}

	cmd.Printf("✅ Successfully wrote %s\n", outPath)
	cmd.Printf("✅ Successfully wrote %s\n", auditPath)
	cmd.Println("\nRun `xcaffold review audit.json` to read the compliance assessment.")
	return nil
}
