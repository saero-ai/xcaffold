package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/prompt"
	"github.com/saero-ai/xcaffold/internal/registry"
	"github.com/saero-ai/xcaffold/internal/templates"
	"github.com/spf13/cobra"
)

// yesFlag is set by --yes / -y to skip all interactive prompts and
// accept defaults (suitable for CI/CD pipelines).
var yesFlag bool

// templateFlag is set by --template to use a pre-built topology template.
var templateFlag string

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap a new scaffold.xcf configuration",
	Long: `xcaffold init bootstraps the environment.

+-------------------------------------------------------------------+
|                          BOOTSTRAP PHASE                          |
+-------------------------------------------------------------------+
 • Detects existing platform config (.claude/, .cursor/, .agents/) and offers to run 'xcaffold import'.
 • Guides you through an interactive wizard for new projects.
 • Infers project name from the current directory (like npm init).
 • Use --yes / -y to accept all defaults non-interactively (CI/CD).
 • Use --scope global to create a user-wide global.xcf in ~/.claude/ (xcaffold home).

Ready to get started? Run:
  $ xcaffold init`,
	Example: `  $ xcaffold init
  $ xcaffold init --yes
  $ xcaffold init --scope global`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Accept all defaults non-interactively (CI/CD mode)")
	initCmd.Flags().StringVar(&templateFlag, "template", "", "use a topology template (rest-api, cli-tool, frontend-app)")
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
		tryAutoRegister(xcfFile)
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
		tryAutoRegister(xcfFile)
		return nil
	}

	// ── Phase 2: Detect existing config and offer import ─────────────────
	if imported, err := offerImportIfPlatformDirExists(cmd); err != nil {
		return err
	} else if imported {
		return nil
	}

	// ── Phase 3: Interactive wizard ─────────────────────────────────────────
	return runWizard(cmd, xcfFile)
}

func tryAutoRegister(xcfFile string) {
	config, err := parser.ParseFile(xcfFile)
	if err == nil && config.Project.Name != "" {
		cwd, _ := os.Getwd()
		_ = registry.Register(cwd, config.Project.Name, nil)
	}
}

// offerImportIfPlatformDirExists checks for existing platform directories (.claude/, .cursor/, .agents/).
// If found, it enumerates all of them with an interactive checkbox selector so the user can
// choose which directories to consolidate into a single scaffold.xcf. Returns true if import was performed.
//
//nolint:gocyclo
func offerImportIfPlatformDirExists(cmd *cobra.Command) (bool, error) {
	infos := detectAllPlatformDirs(".")
	if len(infos) == 0 {
		return false, nil
	}

	// ── Display detection summary ──────────────────────────────────────
	if len(infos) == 1 {
		cmd.Printf("\n⚡ Detected existing agent configuration:\n\n")
	} else {
		cmd.Printf("\n⚡ Detected existing agent configurations:\n\n")
	}
	for _, info := range infos {
		cmd.Printf("     %s  — %s\n", info.dirName, info.summary())
	}
	cmd.Println()

	// ── Single directory: simple Y/n ───────────────────────────────────
	if len(infos) == 1 {
		info := infos[0]
		cmd.Println("  xcaffold will import this into a single scaffold.xcf.")

		doImport := yesFlag
		if !yesFlag {
			var err error
			doImport, err = prompt.Confirm(fmt.Sprintf("Import %s into scaffold.xcf?", info.dirName), true)
			if err != nil {
				return false, fmt.Errorf("prompt error: %w", err)
			}
		}
		if doImport {
			cmd.Println()
			return true, importScope(info.dirName, "scaffold.xcf", "project")
		}

		cmd.Println("\n  ⚠  Skipping import. Continuing with fresh scaffold.xcf.")
		cmd.Println("     Note: 'xcaffold apply' will overlay your existing target files.")
		cmd.Println()
		return false, nil
	}

	// ── Multiple directories: interactive checkbox selector ─────────────
	cmd.Println("  xcaffold consolidates multiple configs into one scaffold.xcf.")
	cmd.Println("  This lets you compile to any target and switch providers seamlessly.")
	cmd.Println()

	if yesFlag {
		// Non-interactive: import all
		var dirs []string
		for _, info := range infos {
			dirs = append(dirs, info.dirName)
		}
		cmd.Println()
		return true, mergeImportDirs(dirs, "scaffold.xcf")
	}

	// Build interactive options — all pre-selected
	var options []prompt.SelectOption
	for _, info := range infos {
		options = append(options, prompt.SelectOption{
			Label:    fmt.Sprintf("%s — %s", info.dirName, info.summary()),
			Value:    info.dirName,
			Selected: true,
		})
	}

	selected, err := prompt.MultiSelect("Select directories to import", options)
	if err != nil {
		return false, fmt.Errorf("prompt error: %w", err)
	}

	if len(selected) == 0 {
		cmd.Println("\n  ⚠  No directories selected. Continuing with fresh scaffold.xcf.")
		cmd.Println("     Note: 'xcaffold apply' will overlay your existing target files.")
		cmd.Println()
		return false, nil
	}

	if len(selected) == 1 {
		cmd.Println()
		return true, importScope(selected[0], "scaffold.xcf", "project")
	}

	cmd.Println()
	return true, mergeImportDirs(selected, "scaffold.xcf")
}

// platformDirInfo holds summary counts of resources found in a platform dir.
type platformDirInfo struct {
	platform  string
	dirName   string
	agents    int
	skills    int
	rules     int
	workflows int
	exists    bool
}

func (c platformDirInfo) summary() string {
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
	if c.workflows > 0 {
		parts = append(parts, fmt.Sprintf("%d workflow(s)", c.workflows))
	}
	if len(parts) == 0 {
		return "no recognized resources"
	}
	return strings.Join(parts, ", ")
}

// detectAllPlatformDirs scans known platform directories under dir and returns all found, sorted by size.
func detectAllPlatformDirs(dir string) []platformDirInfo {
	platformDirs := []struct{ dir, platform string }{
		{".claude", "claude"},
		{".cursor", "cursor"},
		{".agents", "antigravity"},
	}

	var results []platformDirInfo

	for _, pt := range platformDirs {
		targetPath := filepath.Join(dir, pt.dir)
		if _, err := os.Stat(targetPath); err != nil {
			continue
		}

		info := platformDirInfo{exists: true, platform: pt.platform, dirName: pt.dir}

		if agents, _ := filepath.Glob(filepath.Join(targetPath, "agents", "*.md")); agents != nil {
			info.agents += len(agents)
		}
		if skills, _ := filepath.Glob(filepath.Join(targetPath, "skills", "*", "SKILL.md")); skills != nil {
			info.skills += len(skills)
		}
		if rulesMD, _ := filepath.Glob(filepath.Join(targetPath, "rules", "*.md")); rulesMD != nil {
			info.rules += len(rulesMD)
		}
		if rulesMDC, _ := filepath.Glob(filepath.Join(targetPath, "rules", "*.mdc")); rulesMDC != nil {
			info.rules += len(rulesMDC)
		}
		if workflows, _ := filepath.Glob(filepath.Join(targetPath, "workflows", "*.md")); workflows != nil {
			info.workflows += len(workflows)
		}

		results = append(results, info)
	}

	// Sort by total items descending so the richest configuration is first
	sort.Slice(results, func(i, j int) bool {
		totalI := results[i].agents + results[i].skills + results[i].rules
		totalJ := results[j].agents + results[j].skills + results[j].rules
		return totalI > totalJ
	})

	return results
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
	var content string
	if templateFlag != "" {
		model, _ := resolveTargetMeta(ans.target)
		var err error
		content, err = templates.Render(templateFlag, ans.name, model)
		if err != nil {
			return err
		}
	} else {
		content = buildXCFContent(ans)
	}

	if err := os.WriteFile(xcfFile, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to create %s: %w", xcfFile, err)
	}

	cmd.Printf("\n✓ Created scaffold.xcf\n")
	cmd.Printf("  Project: %s | Target: %s\n", ans.name, ans.target)

	if err := registry.Register(cwd, ans.name, []string{ans.target}); err != nil {
		cmd.Printf("  ⚠ Failed to register project: %v\n", err)
	}

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
	return targetClaude // safe fallback
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

// ── Global scope ────────────────────────────────────────

func initGlobal() error {
	home, err := registry.GlobalHome()
	if err != nil {
		return err
	}
	target := filepath.Join(home, "global.xcf")

	// Re-scan global platform dirs every time --scope global is explicit.
	// This lets users refresh global.xcf after adding new agents to ~/.claude/ etc.
	if err := registry.RebuildGlobalXCF(); err != nil {
		return fmt.Errorf("failed to rebuild global.xcf: %w", err)
	}

	if _, err := os.Stat(target); err == nil {
		fmt.Printf("✓ %s rebuilt from global platform directories.\n", target)
	} else {
		fmt.Printf("✓ %s created.\n", target)
	}
	fmt.Println("  Edit it to define your global agents, then run 'xcaffold apply --scope global'.")
	fmt.Println("  Projects can inherit with 'extends: global' in their scaffold.xcf.")
	fmt.Printf("  Output: ~/.claude/ | ~/.cursor/ | ~/.agents/ (depending on --target)\n")
	return nil
}
