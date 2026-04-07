package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/saero-ai/xcaffold/internal/prompt"
	"github.com/spf13/cobra"
)

// yesFlag is set by --yes / -y to skip all interactive prompts and
// accept defaults (suitable for CI/CD pipelines).
var yesFlag bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap a new scaffold.xcf configuration",
	Long: `xcaffold init bootstraps the environment.

+-------------------------------------------------------------------+
|                          BOOTSTRAP PHASE                          |
+-------------------------------------------------------------------+
 • Detects existing .claude/ and offers to run 'xcaffold import'.
 • Guides you through an interactive wizard for new projects.
 • Infers project name from the current directory (like npm init).
 • Use --yes / -y to accept all defaults non-interactively (CI/CD).
 • Use --scope global to create a user-wide global.xcf in ~/.claude/.

Ready to get started? Run:
  $ xcaffold init`,
	Example: `  $ xcaffold init
  $ xcaffold init --yes
  $ xcaffold init --scope global`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Accept all defaults non-interactively (CI/CD mode)")
	rootCmd.AddCommand(initCmd)
}

// runInit executes the core logic of the init command.
func runInit(cmd *cobra.Command, _ []string) error {
	// ── Phase 0: Welcome Banner ────────────────────────────────────────────
	cmd.Println()
	cmd.Printf("  \033[1mxcaffold\033[0m v%s\n", version)
	cmd.Println("  The deterministic agent configuration compiler.")
	cmd.Println("  ──────────────────────────────────────────────────")
	cmd.Println()
	cmd.Println("  Welcome! Let's scaffold your agents.")

	// ── Phase 1: Idempotency check ─────────────────────────────────────────
	xcfFile := filepath.Join(".", "scaffold.xcf")
	if _, err := os.Stat(xcfFile); err == nil {
		cmd.Println("  ✓ scaffold.xcf already exists. Nothing to do.")
		cmd.Println("  Run 'xcaffold apply' to compile, or 'xcaffold diff' to check for drift.")
		return nil
	}

	// ── Phase 2: Detect existing .claude/ and offer import ─────────────────
	if scopeFlag == scopeGlobal {
		return initGlobal()
	}
	return initProject(cmd)
}

// initProject runs the interactive project-level init wizard.
func initProject(cmd *cobra.Command) error {
	const xcfFile = "scaffold.xcf"

	// ── Phase 1: Idempotency check ─────────────────────────────────────────
	// If scaffold.xcf already exists this is a no-op (like `git init` re-run).
	if _, err := os.Stat(xcfFile); err == nil {
		cmd.Println("scaffold.xcf already exists. Nothing to do.")
		cmd.Println("  Run 'xcaffold apply' to compile, or 'xcaffold diff' to check for drift.")
		return nil
	}

	// ── Phase 2: Detect existing .claude/ and offer import ─────────────────
	if imported, err := offerImportIfClaudeExists(cmd); err != nil {
		return err
	} else if imported {
		return nil
	}

	// ── Phase 3: Interactive wizard ─────────────────────────────────────────
	return runWizard(cmd, xcfFile)
}

// offerImportIfClaudeExists checks for an existing .claude/ directory.
// If found, it summarises its contents and (in interactive mode) asks the
// user whether to run xcaffold import. Returns true if import was performed.
func offerImportIfClaudeExists(cmd *cobra.Command) (bool, error) {
	info, err := detectClaudeDir(".")
	if err != nil || !info.exists {
		return false, nil
	}

	cmd.Printf("\n⚡ Detected existing .claude/ with %s.\n", info.summary())
	cmd.Println("   Let's generate your scaffold.xcf using 'xcaffold import'.")

	doImport := yesFlag // --yes auto-imports (non-destructive default)
	if !yesFlag {
		doImport, err = prompt.Confirm(" Import existing .claude/ now?", true)
		if err != nil {
			return false, fmt.Errorf("prompt error: %w", err)
		}
	}

	if doImport {
		cmd.Println()
		return true, runImport(cmd, nil)
	}

	cmd.Println("\n  ⚠  Skipping import. Continuing with fresh scaffold.xcf.")
	cmd.Println("     Note: 'xcaffold apply' will overlay your existing .claude/ files.")
	cmd.Println()
	return false, nil
}

// claudeDirInfo holds summary counts of resources found in .claude/.
type claudeDirInfo struct {
	exists bool
	agents int
	skills int
	rules  int
}

func (c claudeDirInfo) summary() string {
	parts := []string{}
	if c.agents > 0 {
		parts = append(parts, fmt.Sprintf("%d agent(s)", c.agents))
	}
	if c.skills > 0 {
		parts = append(parts, fmt.Sprintf("%d skill(s)", c.skills))
	}
	if c.rules > 0 {
		parts = append(parts, fmt.Sprintf("%d rule(s)", c.rules))
	}
	if len(parts) == 0 {
		return "no recognized resources"
	}
	return strings.Join(parts, ", ")
}

// detectClaudeDir scans the .claude/ directory under dir and returns counts.
func detectClaudeDir(dir string) (claudeDirInfo, error) {
	claudePath := filepath.Join(dir, ".claude")
	if _, err := os.Stat(claudePath); os.IsNotExist(err) {
		return claudeDirInfo{}, nil
	} else if err != nil {
		return claudeDirInfo{}, err
	}
	info := claudeDirInfo{exists: true}

	if agents, _ := filepath.Glob(filepath.Join(claudePath, "agents", "*.md")); agents != nil {
		info.agents = len(agents)
	}
	if skills, _ := filepath.Glob(filepath.Join(claudePath, "skills", "*", "SKILL.md")); skills != nil {
		info.skills = len(skills)
	}
	if rules, _ := filepath.Glob(filepath.Join(claudePath, "rules", "*.md")); rules != nil {
		info.rules = len(rules)
	}
	return info, nil
}

// runWizard runs the interactive new-project wizard and writes scaffold.xcf.
func runWizard(cmd *cobra.Command, xcfFile string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine working directory: %w", err)
	}
	defaultName := filepath.Base(cwd)

	// ── Collect answers ────────────────────────────────────────────────────
	ans, err := collectWizardAnswers(defaultName)
	if err != nil {
		return err
	}

	// ── Build scaffold.xcf content ─────────────────────────────────────────
	content := buildXCFContent(ans)

	if err := os.WriteFile(xcfFile, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to create %s: %w", xcfFile, err)
	}

	cmd.Printf("\n✓ Created scaffold.xcf\n")
	cmd.Printf("  Project: %s | Target: %s\n", ans.name, ans.target)

	if ans.wantAgent {
		model, _ := resolveTargetMeta(ans.target)
		cmd.Printf("  Starter agent: developer (model: %s)\n", model)
	}
	cmd.Println("\n  Edit your agents, then run 'xcaffold apply'.")

	// ── Optional: offer xcaffold analyze ──────────────────────────────────
	if ans.wantAnalyze {
		if err := offerAnalyze(cmd, ans.target); err != nil {
			return err
		}
	}

	return nil
}

// wizardAnswers holds all answers collected during the interactive wizard.
type wizardAnswers struct {
	name        string
	desc        string
	target      string // "claude", "cursor", "antigravity", or "other"
	wantAgent   bool
	wantAnalyze bool
}

// knownCLIs lists the AI coding CLIs xcaffold knows about, in detection order.
// The first one found on PATH becomes the suggested default target.
var knownCLIs = []struct {
	binary string
	label  string
	target string
	model  string
}{
	{"claude", "Claude Code", "claude", "claude-sonnet-4-6"},
	{"gemini", "Gemini (Antigravity)", "antigravity", "gemini-2.5-pro"},
	{"cursor", "Cursor", "cursor", "cursor-default"},
}

// detectDefaultTarget returns the target label for the first CLI binary
// found on PATH, or an empty string if none are found.
func detectDefaultTarget() string {
	for _, cli := range knownCLIs {
		if _, err := exec.LookPath(cli.binary); err == nil {
			return cli.target
		}
	}
	return "claude" // safe fallback
}

// resolveTargetMeta returns the suggested model and binary name for a target.
func resolveTargetMeta(target string) (model, binary string) {
	for _, cli := range knownCLIs {
		if cli.target == target {
			return cli.model, cli.binary
		}
	}
	return "claude-sonnet-4-6", "claude" // fallback
}

// collectWizardAnswers populates wizard answers. The project name is always
// derived from the CWD folder — no prompt needed. Description is left blank
// so the user can fill it in later. Only three questions are asked:
// target platform (auto-detected, user can override), starter agent, and
// whether to run xcaffold analyze.
func collectWizardAnswers(defaultName string) (ans wizardAnswers, err error) {
	// Name and description are set automatically — never prompted.
	ans.name = defaultName
	ans.desc = ""
	ans.target = detectDefaultTarget()
	ans.wantAgent = true

	if yesFlag {
		return
	}

	// Target platform — auto-detected but user can change.
	time.Sleep(300 * time.Millisecond)
	ans.target, err = prompt.Ask("Target platform (claude / cursor / antigravity)", ans.target)
	if err != nil {
		return
	}

	time.Sleep(300 * time.Millisecond)
	ans.wantAgent, err = prompt.Confirm("Add a starter agent?", true)
	if err != nil {
		return
	}

	time.Sleep(300 * time.Millisecond)
	ans.wantAnalyze, err = prompt.Confirm("Run 'xcaffold analyze' to generate config from your repo?", false)
	return
}

// buildXCFContent generates the scaffold.xcf YAML string from wizard answers.
func buildXCFContent(ans wizardAnswers) string {
	var sb strings.Builder

	model, binary := resolveTargetMeta(ans.target)

	sb.WriteString(fmt.Sprintf("version: \"1.0\"\nproject:\n  name: %q\n  description: %q\n", ans.name, ans.desc))

	if ans.wantAgent {
		sb.WriteString(fmt.Sprintf(`
agents:
  developer:
    description: "General software developer agent."
    instructions: |
      You are a software developer.
      Write clean, maintainable code.
    model: %q
    effort: "high"
    tools: [Bash, Read, Write, Edit, Glob, Grep]

    # Assertions are evaluated by the LLM-as-a-Judge when running
    # 'xcaffold test --judge'. Define expected behavioral constraints here.
    # assertions:
    #   - "The agent must not write files outside the project directory."
    #   - "The agent must run tests before marking a task complete."
`, model))
	} else {
		sb.WriteString(fmt.Sprintf(`
# agents:
#   developer:
#     description: "General software developer agent."
#     instructions: |
#       You are a software developer.
#       Write clean, maintainable code.
#     model: %q
#     effort: "high"
#     tools: [Bash, Read, Write, Edit, Glob, Grep]
`, model))
	}

	sb.WriteString(fmt.Sprintf(`
# Optional: Configure the 'xcaffold test' simulator.
# test:
#   cli_path: %q   # Path to CLI binary. Defaults to '%s' on $PATH.
#   judge_model: %q # Model used for --judge evaluation.
`, binary, binary, model))

	return sb.String()
}

// offerAnalyze runs xcaffold analyze inline when a supported LLM is available
// (any API key env var OR a known CLI binary on PATH), or prints the command
// for the user to run later if nothing is configured.
func offerAnalyze(cmd *cobra.Command, target string) error {
	// Check API key env vars first.
	hasAPIKey := os.Getenv("ANTHROPIC_API_KEY") != "" ||
		os.Getenv("XCAFFOLD_LLM_API_KEY") != ""

	// Then check if the target CLI (or any known CLI) is on PATH.
	_, targetBinary := resolveTargetMeta(target)
	_, cliErr := exec.LookPath(targetBinary)
	hasCLI := cliErr == nil

	if !hasAPIKey && !hasCLI {
		// Nothing available — tell the user what to set up.
		cmd.Println("\n  To generate config from your repo, run:")
		cmd.Println("    $ xcaffold analyze")
		cmd.Println("  Requires one of:")
		cmd.Println("    • ANTHROPIC_API_KEY or XCAFFOLD_LLM_API_KEY env var (direct API)")
		cmd.Println("    • claude / antigravity / cursor CLI installed and on your PATH (subscription)")
		return nil
	}

	// Something is available — run inline.
	if hasCLI && !hasAPIKey {
		cmd.Printf("\n🧠 Running 'xcaffold analyze' via %s CLI subscription...\n", targetBinary)
	} else {
		cmd.Println("\n🧠 Running 'xcaffold analyze' via API key...")
	}
	return runAnalyze(cmd, nil)
}

// ── Global scope (unchanged behaviour) ────────────────────────────────────────

// defaultGlobalXCFContent is the starter template for `xcaffold init --scope global`.
const defaultGlobalXCFContent = `version: "1.0"
project:
  name: "global"
  description: "User-wide agent configuration."

# Agents defined here are available across all projects.
# Project-level scaffold.xcf can inherit with 'extends: global'.
agents:
  developer:
    description: "Default developer agent."
    instructions: |
      You are a software developer.
      Write clean, maintainable code.
    model: "claude-sonnet-4-6"
    effort: "high"
    tools: [Bash, Read, Write, Edit, Glob, Grep]
`

func initGlobal() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine home directory: %w", err)
	}
	dir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create %s: %w", dir, err)
	}
	target := filepath.Join(dir, "global.xcf")
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("%s already exists; delete it first if you want to re-initialize", target)
	}
	if err := os.WriteFile(target, []byte(defaultGlobalXCFContent), 0600); err != nil {
		return fmt.Errorf("failed to create %s: %w", target, err)
	}
	fmt.Printf("Created %s\n", target)
	fmt.Println("  Edit it to define your global agents, then run 'xcaffold apply --scope global'.")
	fmt.Println("  Projects can inherit with 'extends: global' in their scaffold.xcf.")
	return nil
}
