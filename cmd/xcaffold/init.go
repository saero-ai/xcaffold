package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/saero-ai/xcaffold/internal/ast"
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

// noReferencesFlag is set by --no-references to skip reference template generation.
var noReferencesFlag bool

var targetsFlag []string
var noPoliciesFlag bool
var jsonManifestFlag bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap a new project.xcf configuration",
	Long: `xcaffold init bootstraps the environment.

+-------------------------------------------------------------------+
|                          BOOTSTRAP PHASE                          |
+-------------------------------------------------------------------+
 • Detects existing platform config (.claude/, .cursor/, .agents/) and offers to run 'xcaffold import'.
 • Guides you through an interactive wizard for new projects.
 • Infers project name from the current directory (like npm init).
 • Use --yes / -y to accept all defaults non-interactively (CI/CD).
 • Use --global to create a user-wide global.xcf in ~/.xcaffold/ (xcaffold home).

Ready to get started? Run:
  $ xcaffold init`,
	Example: `  $ xcaffold init
  $ xcaffold init --yes
  $ xcaffold init --global`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Accept all defaults non-interactively (CI/CD mode)")
	initCmd.Flags().StringSliceVar(&targetsFlag, "target", nil, "Generate output tailored to specific target(s) (comma-separated)")
	initCmd.Flags().StringVar(&templateFlag, "template", "", "use a topology template (rest-api, cli-tool, frontend-app)")
	initCmd.Flags().BoolVar(&noReferencesFlag, "no-references", false, "Skip generation of xcf/references/ field reference templates")
	initCmd.Flags().BoolVar(&noPoliciesFlag, "no-policies", false, "Skip generation of starter policies")
	initCmd.Flags().BoolVar(&jsonManifestFlag, "json", false, "Output machine-readable JSON manifest instead of interactive logs")
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

	// ── Phase 1: Global flag takes priority ───────────────────────────────
	if globalFlag {
		return initGlobal()
	}

	// ── Phase 2: Existing project.xcf — offer re-import ─────────────────
	return initProject(cmd)
}

// initProject runs the interactive project-level init wizard.
func initProject(cmd *cobra.Command) error {
	const xcfFile = "project.xcf"

	var currentConfig *ast.XcaffoldConfig
	hasExistingScaffold := false
	if _, err := os.Stat(xcfFile); err == nil {
		hasExistingScaffold = true
		currentConfig, _ = parser.ParseFile(xcfFile)
	}

	infos := detectAllPlatformDirs(".")

	// ── Phase 1: Existing scaffold, NO native dirs ────────────────────────
	if hasExistingScaffold && len(infos) == 0 {
		cmd.Println()
		cmd.Println("  project.xcf already exists, but no provider directories were found.")
		cmd.Println("  Run 'xcaffold apply' to compile, or 'xcaffold diff' to check for drift.")
		tryAutoRegister(xcfFile)
		return nil
	}

	// ── Phase 2: Native Dirs detected (Offer Import) ──────────────────────
	if len(infos) > 0 {
		cmd.Println()
		if hasExistingScaffold {
			cmd.Println("  project.xcf already exists, but existing compiled configurations were detected.")
		} else {
			cmd.Println("  ⚡ Detected existing agent configuration(s):")
		}
		cmd.Println()

		if currentConfig != nil {
			renderCurrentStateTable(cmd, currentConfig)
			cmd.Println()
		}
		renderCompiledOutputTable(cmd, infos)

		var doImport bool
		if hasExistingScaffold {
			cmd.Println()
			if yesFlag {
				doImport = true
			} else {
				var err error
				doImport, err = prompt.Confirm("Re-import from source directories? (overwrites project.xcf and xcf/)", false)
				if err != nil {
					return fmt.Errorf("prompt error: %w", err)
				}
			}
		} else {
			cmd.Println()
			if yesFlag {
				doImport = true
			} else {
				fmtStr := "Import %s into project.xcf?"
				if len(infos) > 1 {
					cmd.Println("  xcaffold consolidates multiple configs into one project.xcf.")
					fmtStr = "Import these directories into project.xcf?"
				} else {
					fmtStr = fmt.Sprintf(fmtStr, infos[0].dirName)
				}

				var err error
				doImport, err = prompt.Confirm(fmtStr, true)
				if err != nil {
					return fmt.Errorf("prompt error: %w", err)
				}
			}
		}

		if !doImport {
			if hasExistingScaffold {
				cmd.Println("  Run 'xcaffold apply' to compile, or 'xcaffold diff' to check for drift.")
				tryAutoRegister(xcfFile)
				return nil
			} else {
				cmd.Println("\n  ⚠  Skipping import. Continuing with fresh project.xcf.")
				cmd.Println("     Note: 'xcaffold apply' will overlay your existing target files.")
				cmd.Println()
				// Proceed to Phase 3
			}
		} else {
			if hasExistingScaffold {
				_ = os.Remove(xcfFile)
				_ = os.RemoveAll("xcf")
			}
			cmd.Println()

			var importErr error
			if len(infos) == 1 {
				importErr = importScope(infos[0].dirName, xcfFile, "project", infos[0].platform)
			} else {
				if yesFlag {
					var dirs []string
					for _, info := range infos {
						dirs = append(dirs, info.dirName)
					}
					importErr = mergeImportDirs(dirs, xcfFile)
				} else {
					var options []prompt.SelectOption
					for _, info := range infos {
						options = append(options, prompt.SelectOption{
							Label:    fmt.Sprintf("%s", info.dirName),
							Value:    info.dirName,
							Selected: true,
						})
					}
					selected, err := prompt.MultiSelect("Select directories to import", options)
					if err != nil {
						return fmt.Errorf("prompt error: %w", err)
					}
					if len(selected) == 0 {
						if hasExistingScaffold {
							cmd.Println("\n  ⚠  No directories selected. Aborting.")
							return nil
						} else {
							cmd.Println("\n  ⚠  No directories selected. Continuing with fresh project.xcf.")
							// Proceed to Phase 3 instead of aborting
							doImport = false
						}
					} else {
						if len(selected) == 1 {
							cmd.Println()
							importErr = importScope(selected[0], xcfFile, "project", selectedPlatform(infos, selected[0]))
						} else {
							cmd.Println()
							importErr = mergeImportDirs(selected, xcfFile)
						}
					}
				}
			}

			if doImport {
				if importErr != nil {
					return importErr
				}

				_ = writeReferenceTemplates(".")

				if injectErr := injectXcaffoldSkillAfterImport("."); injectErr != nil {
					cmd.Printf("  ⚠ Failed to inject /xcaffold skill: %v\n", injectErr)
				} else {
					cmd.Println("\n💡 AI Assistant Integration:")
					cmd.Println("  A complementary /xcaffold AI skill was generated in xcf/skills/xcaffold.xcf.")
					cmd.Println("  Run 'xcaffold apply' to instantly teach AI assistants in this project how to use xcaffold.")
					cmd.Println("  To install this skill globally for your preferred provider, run:")
					cmd.Println("    $ xcaffold init --global")
					cmd.Println("    $ xcaffold apply --global")
				}
				return nil
			}
		}
	}

	// ── Phase 3: Interactive wizard ─────────────────────────────────────────
	if hasExistingScaffold {
		return nil
	}
	return runWizard(cmd, xcfFile)
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
		{".gemini", "gemini"},
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
		// Count rules recursively to include nested subdirectory rules.
		_ = filepath.WalkDir(filepath.Join(targetPath, "rules"), func(_ string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			name := strings.ToLower(d.Name())
			if strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".mdc") {
				info.rules++
			}
			return nil
		})
		if workflows, _ := filepath.Glob(filepath.Join(targetPath, "workflows", "*.md")); workflows != nil {
			info.workflows += len(workflows)
		}

		results = append(results, info)
	}

	// Sort by total items descending so the richest configuration is first
	sort.Slice(results, func(i, j int) bool {
		totalI := results[i].agents + results[i].skills + results[i].rules + results[i].workflows
		totalJ := results[j].agents + results[j].skills + results[j].rules + results[j].workflows
		return totalI > totalJ
	})

	return results
}

// selectedPlatform returns the platform name for the given dirName from infos.
// Falls back to "claude" if not found (safe default).
func selectedPlatform(infos []platformDirInfo, dirName string) string {
	for _, info := range infos {
		if info.dirName == dirName {
			return info.platform
		}
	}
	return "claude"
}

// runWizard runs the interactive new-project wizard and writes project.xcf.
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

	if jsonManifestFlag {
		yesFlag = true // silent mode implicitly accepts defaults if questions not answered
		ans.targets = []string{detectDefaultTarget()}
		if len(targetsFlag) > 0 {
			ans.targets = targetsFlag
		}
	}

	// ── Build project.xcf content ──────────────────────────────────────────
	if templateFlag != "" {
		model, _ := resolveTargetMeta(ans.targets[0])
		content, err := templates.Render(templateFlag, ans.name, model)
		if err != nil {
			return err
		}
		if err := os.WriteFile(xcfFile, []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to create %s: %w", xcfFile, err)
		}
	} else {
		if err := writeXCFDirectory(cwd, ans); err != nil {
			return fmt.Errorf("failed to scaffold directory: %w", err)
		}
	}

	if err := writeReferenceTemplates(cwd); err != nil && !jsonManifestFlag {
		cmd.Printf("  ⚠ Failed to write reference templates: %v\n", err)
		// Non-fatal: continue with init.
	} else if !noReferencesFlag && !jsonManifestFlag {
		cmd.Println("  Created xcf/references/ — field reference for resource kinds")
	}

	if err := registry.Register(cwd, ans.name, ans.targets, "."); err != nil && !jsonManifestFlag {
		cmd.Printf("  ⚠ Failed to register project: %v\n", err)
	}

	if jsonManifestFlag {
		type Manifest struct {
			Project string   `json:"project"`
			Targets []string `json:"targets"`
			Files   []string `json:"files"`
		}

		files := []string{"project.xcf", "xcf/rules/conventions.xcf", "xcf/settings.xcf"}
		if !noPoliciesFlag {
			files = append(files, "xcf/policies/safety.xcf")
		}
		if ans.wantAgent {
			files = append(files, "xcf/agents/developer.xcf")
		}

		b, err := json.MarshalIndent(Manifest{Project: ans.name, Targets: ans.targets, Files: files}, "", "  ")
		if err == nil {
			cmd.Println(string(b))
		}
		return nil
	}

	cmd.Printf("\n✓ Created project.xcf\n")
	cmd.Printf("  Project: %s | Targets: %s\n", ans.name, strings.Join(ans.targets, ", "))

	if ans.wantAgent {
		model, _ := resolveTargetMeta(ans.targets[0])
		cmd.Printf("  Starter agent: developer (model: %s)\n", model)
	}
	cmd.Println("\n  Edit your agents, then run 'xcaffold apply'.")

	cmd.Println("\n💡 AI Assistant Integration:")
	cmd.Println("  A complementary /xcaffold AI skill was generated in xcf/skills/xcaffold.xcf.")
	cmd.Println("  Run 'xcaffold apply' to instantly teach AI assistants in this project how to use xcaffold.")
	cmd.Println("  To install this skill globally for your preferred provider, run:")
	cmd.Println("    $ xcaffold init --global")
	cmd.Println("    $ xcaffold apply --global")

	// ── Optional: offer xcaffold analyze ──────────────────────────────────
	if ans.wantAnalyze && len(ans.targets) > 0 {
		if err := offerAnalyze(cmd, ans.targets[0]); err != nil {
			return err
		}
	}

	return nil
}

// wizardAnswers holds all answers collected during the interactive wizard.
type wizardAnswers struct {
	name        string
	desc        string
	targets     []string // list of providers, e.g. ["claude", "cursor"]
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
	{targetClaude, "Claude Code", targetClaude, "claude-sonnet-4-6"},
	{"gemini", "Gemini (Antigravity)", targetAntigravity, "gemini-2.5-pro"},
	{targetCursor, "Cursor", targetCursor, "cursor-default"},
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
	return "claude-sonnet-4-6", targetClaude // fallback
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
	if len(targetsFlag) > 0 {
		ans.targets = targetsFlag
	} else {
		ans.targets = []string{detectDefaultTarget()}
	}
	ans.wantAgent = true

	if yesFlag {
		return
	}

	if len(targetsFlag) == 0 {
		time.Sleep(300 * time.Millisecond)
		options := []prompt.SelectOption{
			{Label: "Claude Code", Value: "claude", Selected: ans.targets[0] == "claude"},
			{Label: "Cursor", Value: "cursor", Selected: ans.targets[0] == "cursor"},
			{Label: "Gemini", Value: "gemini", Selected: ans.targets[0] == "gemini"},
			{Label: "GitHub Copilot", Value: "copilot", Selected: ans.targets[0] == "copilot"},
			{Label: "Antigravity", Value: "antigravity", Selected: ans.targets[0] == "antigravity"},
		}
		selected, promptErr := prompt.MultiSelect("Target platforms (space to select)", options)
		if promptErr != nil {
			err = promptErr
			return
		}
		if len(selected) > 0 {
			ans.targets = selected
		}
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

// writeXCFDirectory generates the multi-file scaffold structure.
func writeXCFDirectory(baseDir string, ans wizardAnswers) error {
	// Ensure xcf/ directories exist
	dirs := []string{
		filepath.Join(baseDir, "xcf", "agents"),
		filepath.Join(baseDir, "xcf", "skills"),
		filepath.Join(baseDir, "xcf", "rules"),
	}
	if !noPoliciesFlag {
		dirs = append(dirs, filepath.Join(baseDir, "xcf", "policies"))
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", d, err)
		}
	}

	model, _ := resolveTargetMeta(ans.targets[0])

	// project.xcf
	projectContent := templates.RenderProjectXCF(ans.name, ans.targets)
	if err := os.WriteFile(filepath.Join(baseDir, "project.xcf"), []byte(projectContent), 0o600); err != nil {
		return err
	}

	// rules & policies & settings
	ruleContent := templates.RenderRuleXCF(ans.targets)
	_ = os.WriteFile(filepath.Join(baseDir, "xcf", "rules", "conventions.xcf"), []byte(ruleContent), 0o600)

	settingsContent := templates.RenderSettingsXCF(ans.targets)
	_ = os.WriteFile(filepath.Join(baseDir, "xcf", "settings.xcf"), []byte(settingsContent), 0o600)

	if !noPoliciesFlag {
		policyContent := templates.RenderPolicyXCF()
		_ = os.WriteFile(filepath.Join(baseDir, "xcf", "policies", "safety.xcf"), []byte(policyContent), 0o600)
	}

	// starter agent
	if ans.wantAgent {
		agentContent := templates.RenderAgentXCF("developer", model, ans.targets)
		_ = os.WriteFile(filepath.Join(baseDir, "xcf", "agents", "developer.xcf"), []byte(agentContent), 0o600)
	}

	// xcaffold skill
	skillContent := templates.RenderXcaffoldSkillXCF(ans.targets)
	_ = os.WriteFile(filepath.Join(baseDir, "xcf", "skills", "xcaffold.xcf"), []byte(skillContent), 0o600)

	return nil
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

// writeReferenceTemplates creates xcf/references/<kind>.xcf.reference files
// inside baseDir. The files are documentation artifacts and are NOT parsed
// by xcaffold. When the --no-references flag is set, this is a no-op.
func writeReferenceTemplates(baseDir string) error {
	if noReferencesFlag {
		return nil
	}

	refDir := filepath.Join(baseDir, "xcf", "references")
	if err := os.MkdirAll(refDir, 0o755); err != nil {
		return fmt.Errorf("failed to create references directory: %w", err)
	}

	agentRef := filepath.Join(refDir, "agent.xcf.reference")
	if err := os.WriteFile(agentRef, []byte(templates.RenderAgentReference()), 0o600); err != nil {
		return fmt.Errorf("failed to write agent reference: %w", err)
	}

	skillRef := filepath.Join(refDir, "skill.xcf.reference")
	if err := os.WriteFile(skillRef, []byte(templates.RenderSkillReference()), 0o600); err != nil {
		return fmt.Errorf("failed to write skill reference: %w", err)
	}

	return nil
}

// ── Global scope ────────────────────────────────────────

func initGlobal() error {
	home, err := registry.GlobalHome()
	if err != nil {
		return err
	}
	target := filepath.Join(home, "global.xcf")

	// Re-scan global platform dirs every time --global is used.
	// This lets users refresh global.xcf after adding new agents to ~/.claude/ etc.
	if err := registry.RebuildGlobalXCF(); err != nil {
		return fmt.Errorf("failed to rebuild global.xcf: %w", err)
	}

	if _, err := os.Stat(target); err == nil {
		fmt.Printf("✓ %s rebuilt from global platform directories.\n", target)
	} else {
		fmt.Printf("✓ %s created.\n", target)
	}
	fmt.Println("  Edit it to define your global agents, then run 'xcaffold apply --global'.")
	fmt.Println("  Projects can inherit with 'extends: global' in their project.xcf.")
	fmt.Printf("  Output: ~/.claude/ | ~/.cursor/ | ~/.agents/ (depending on --target)\n")
	return nil
}

func tryAutoRegister(xcfFile string) {
	config, err := parser.ParseFile(xcfFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to parse %s for auto-registration: %v\n", xcfFile, err)
		return
	}
	if config.Project != nil && config.Project.Name != "" {
		cwd, _ := os.Getwd()
		_ = registry.Register(cwd, config.Project.Name, nil, ".")
	}
}

// injectXcaffoldSkillAfterImport parses the imported project.xcf, adds the xcaffold skill to the
// skills list if not present, and statically writes the xcf/skills/xcaffold.xcf template.
func injectXcaffoldSkillAfterImport(baseDir string) error {
	xcfFile := filepath.Join(baseDir, "project.xcf")
	config, err := parser.ParseFile(xcfFile)
	if err != nil {
		return fmt.Errorf("parsing imported scaffold: %w", err)
	}

	targets := []string{"claude"} // fallback
	if config.Project != nil && len(config.Project.Targets) > 0 {
		targets = config.Project.Targets
	} else if config.Project == nil {
		config.Project = &ast.ProjectConfig{Name: filepath.Base(baseDir), Targets: targets}
	}

	hasSkill := false
	for _, s := range config.Project.SkillRefs {
		if s == "xcaffold" {
			hasSkill = true
			break
		}
	}
	if !hasSkill {
		config.Project.SkillRefs = append(config.Project.SkillRefs, "xcaffold")
		if err := WriteSplitFiles(config, baseDir); err != nil {
			return fmt.Errorf("writing updated split config: %w", err)
		}
	}

	// Always ensure the xcf/skills/xcaffold.xcf is written
	skillsDir := filepath.Join(baseDir, "xcf", "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		return err
	}
	skillContent := templates.RenderXcaffoldSkillXCF(targets)
	return os.WriteFile(filepath.Join(skillsDir, "xcaffold.xcf"), []byte(skillContent), 0o600)
}
