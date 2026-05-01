package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/prompt"
	"github.com/saero-ai/xcaffold/internal/registry"
	"github.com/saero-ai/xcaffold/internal/templates"
	"github.com/spf13/cobra"
)

var yesFlag bool

var targetsFlag []string
var noPoliciesFlag bool
var jsonManifestFlag bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap a new project.xcf configuration",
	Long: `xcaffold init bootstraps a new project.

 • Detects existing platform config (.claude/, .cursor/, .agents/) and offers to run 'xcaffold import'.
 • Guides you through an interactive wizard for new projects.
 • Generates the Xaff authoring toolkit: agent, xcaffold skill, xcf-conventions rule, and schema references.
 • Infers project name from the current directory.
 • Use --yes / -y to accept all defaults non-interactively (CI/CD).

Ready to get started? Run:
  $ xcaffold init`,
	Example: `  $ xcaffold init
  $ xcaffold init --yes
  $ xcaffold init --target claude`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Accept all defaults non-interactively (CI/CD mode)")
	initCmd.Flags().StringSliceVar(&targetsFlag, "target", nil, "Compilation target(s): claude, cursor, gemini, copilot, antigravity")
	initCmd.Flags().BoolVar(&noPoliciesFlag, "no-policies", false, "Skip generation of starter policies")
	initCmd.Flags().BoolVar(&jsonManifestFlag, "json", false, "Output machine-readable JSON manifest instead of interactive logs")
	rootCmd.AddCommand(initCmd)
}

// runInit executes the core logic of the init command.
func runInit(cmd *cobra.Command, _ []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine working directory: %w", err)
	}
	projectName := filepath.Base(cwd)

	fmt.Println(formatHeader(projectName, "", false, "", ""))
	fmt.Println()

	if globalFlag {
		return initGlobal()
	}

	return initProject(cmd)
}

// initProject runs the interactive project-level init wizard.
func initProject(cmd *cobra.Command) error {
	xcfFile := filepath.Join(".xcaffold", "project.xcf")

	var currentConfig *ast.XcaffoldConfig
	hasExistingScaffold := false
	if _, err := os.Stat(xcfFile); err == nil {
		hasExistingScaffold = true
		currentConfig, _ = parser.ParseFile(xcfFile)
	} else if _, err := os.Stat("project.xcf"); err == nil {
		hasExistingScaffold = true
		xcfFile = "project.xcf"
		currentConfig, _ = parser.ParseFile(xcfFile)
	}

	infos := detectPlatformDirs(".", false)

	// ── Case B: Existing scaffold, no provider dirs ───────────────────────────
	if hasExistingScaffold && len(infos) == 0 {
		cwd, _ := os.Getwd()
		projectName := filepath.Base(cwd)
		fmt.Println(formatHeader(projectName, "", false, "", "already initialized"))
		fmt.Println()
		fmt.Printf("  %s .xcaffold/project.xcf exists.\n", glyphNever())
		fmt.Println()
		fmt.Printf("%s Run 'xcaffold apply' to compile your xcf/ sources.\n", glyphArrow())
		fmt.Printf("  Run 'xcaffold import' to sync provider changes back to xcf/.\n")
		tryAutoRegister(xcfFile)
		return nil
	}

	// ── Case C: Provider dirs detected (offer import) ─────────────────────────
	if len(infos) > 0 {
		fmt.Println()
		if hasExistingScaffold {
			fmt.Println("  project.xcf already exists, but existing compiled configurations were detected.")
		} else {
			fmt.Printf("  %s Detected existing agent configuration(s):\n", glyphOK())
		}
		fmt.Println()

		if currentConfig != nil {
			renderCurrentStateTable(cmd, currentConfig)
			fmt.Println()
		}
		renderCompiledOutputTable(cmd, infos)

		var doImport bool
		if hasExistingScaffold {
			fmt.Println()
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
			fmt.Println()
			if yesFlag {
				doImport = true
			} else {
				fmtStr := "Import %s into project.xcf?"
				if len(infos) > 1 {
					fmt.Println("  xcaffold consolidates multiple configs into one project.xcf.")
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
				fmt.Printf("%s Run 'xcaffold apply' to compile, or 'xcaffold status' to check for drift.\n", glyphArrow())
				tryAutoRegister(xcfFile)
				return nil
			}
			fmt.Printf("\n  %s Skipping import. Continuing with fresh scaffold.\n", glyphNever())
			fmt.Println()
			// Proceed to wizard (Case D)
		} else {
			if hasExistingScaffold {
				_ = os.Remove(xcfFile)
				_ = os.RemoveAll("xcf")
			}
			fmt.Println()

			var importErr error
			if len(infos) == 1 {
				importErr = importScope(infos[0].dirName, xcfFile, "project", infos[0].platform)
			} else {
				if yesFlag {
					importErr = mergeImportDirs(infos, xcfFile)
				} else {
					var options []prompt.SelectOption
					for _, info := range infos {
						options = append(options, prompt.SelectOption{
							Label:    info.dirName,
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
							fmt.Printf("\n  %s No directories selected. Aborting.\n", glyphNever())
							return nil
						}
						fmt.Printf("\n  %s No directories selected. Continuing with fresh scaffold.\n", glyphNever())
						doImport = false
					} else {
						if len(selected) == 1 {
							fmt.Println()
							importErr = importScope(selected[0], xcfFile, "project", selectedPlatform(infos, selected[0]))
						} else {
							fmt.Println()
							var selectedInfos []platformDirInfo
							for _, s := range selected {
								selectedInfos = append(selectedInfos, platformDirInfo{
									dirName:  s,
									platform: selectedPlatform(infos, s),
									exists:   true,
								})
							}
							importErr = mergeImportDirs(selectedInfos, xcfFile)
						}
					}
				}
			}

			if doImport {
				if importErr != nil {
					return importErr
				}

				_ = writeReferenceTemplates(".")

				if injectErr := injectXaffToolkitAfterImport("."); injectErr != nil {
					fmt.Printf("  %s Failed to inject xcaffold toolkit: %v\n", glyphErr(), injectErr)
				} else {
					fmt.Println()
					fmt.Printf("  %s xcf/agents/xaff/\n", colorGreen(glyphOK()))
					fmt.Printf("  %s xcf/skills/xcaffold/\n", colorGreen(glyphOK()))
					fmt.Printf("  %s xcf/rules/xcf-conventions/\n", colorGreen(glyphOK()))
					fmt.Printf("  %s .xcaffold/schemas/                    %s\n",
						colorGreen(glyphOK()), dim("8 references"))
					fmt.Println()
					fmt.Printf("%s Run 'xcaffold validate' then 'xcaffold apply'.\n", glyphArrow())
				}
				return nil
			}
		}
	}

	// ── Case D: New project (wizard) ──────────────────────────────────────────
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

// selectedPlatform returns the platform name for the given dirName from infos.
func selectedPlatform(infos []platformDirInfo, dirName string) string {
	for _, info := range infos {
		if info.dirName == dirName {
			return info.platform
		}
	}
	return "claude"
}

// runWizard runs the interactive new-project wizard.
func runWizard(cmd *cobra.Command, xcfFile string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine working directory: %w", err)
	}
	defaultName := filepath.Base(cwd)

	ans, err := collectWizardAnswers(defaultName)
	if err != nil {
		return err
	}

	if jsonManifestFlag {
		yesFlag = true
		ans.targets = []string{detectDefaultTarget()}
		if len(targetsFlag) > 0 {
			ans.targets = targetsFlag
		}
	}

	if err := writeXCFDirectory(cwd, ans); err != nil {
		return fmt.Errorf("failed to scaffold directory: %w", err)
	}

	if err := writeReferenceTemplates(cwd); err != nil && !jsonManifestFlag {
		fmt.Printf("  %s Failed to write reference templates: %v\n", glyphErr(), err)
		// Non-fatal: continue.
	}

	if err := registry.Register(cwd, ans.name, ans.targets, "."); err != nil && !jsonManifestFlag {
		fmt.Printf("  %s Failed to register project: %v\n", glyphErr(), err)
	}

	if jsonManifestFlag {
		type Manifest struct {
			Project string   `json:"project"`
			Targets []string `json:"targets"`
			Files   []string `json:"files"`
		}

		files := []string{
			".xcaffold/project.xcf",
			"xcf/agents/xaff/agent.xcf",
		}
		for _, t := range ans.targets {
			files = append(files, fmt.Sprintf("xcf/agents/xaff/agent.%s.xcf", t))
		}
		files = append(files,
			"xcf/skills/xcaffold/xcaffold.xcf",
			"xcf/rules/xcf-conventions/xcf-conventions.xcf",
		)
		if !noPoliciesFlag {
			files = append(files,
				"xcf/policies/require-agent-description.xcf",
				"xcf/policies/require-agent-instructions.xcf",
			)
		}
		files = append(files, "xcf/settings.xcf")
		for _, ref := range []string{"agent", "skill", "rule", "workflow", "mcp", "hooks", "memory"} {
			files = append(files, fmt.Sprintf(".xcaffold/schemas/%s.xcf.reference", ref))
		}
		files = append(files, ".xcaffold/schemas/cli-cheatsheet.reference")

		b, err := json.MarshalIndent(Manifest{Project: ans.name, Targets: ans.targets, Files: files}, "", "  ")
		if err == nil {
			cmd.Println(string(b))
		}
		return nil
	}

	fmt.Println()
	fmt.Printf("  %s .xcaffold/project.xcf\n", colorGreen(glyphOK()))
	fmt.Printf("  %s xcf/agents/xaff/                     %s\n",
		colorGreen(glyphOK()), dim(fmt.Sprintf("base + %d %s", len(ans.targets), plural(len(ans.targets), "override", "overrides"))))
	fmt.Printf("  %s xcf/skills/xcaffold/\n", colorGreen(glyphOK()))
	fmt.Printf("  %s xcf/rules/xcf-conventions/\n", colorGreen(glyphOK()))
	if !noPoliciesFlag {
		fmt.Printf("  %s xcf/policies/                         %s\n",
			colorGreen(glyphOK()), dim("2 policies"))
	}
	fmt.Printf("  %s xcf/settings.xcf\n", colorGreen(glyphOK()))
	fmt.Printf("  %s .xcaffold/schemas/                    %s\n",
		colorGreen(glyphOK()), dim("8 references"))
	fmt.Println()
	fmt.Printf("%s Run 'xcaffold validate' then 'xcaffold apply'.\n", glyphArrow())
	fmt.Printf("  Xaff will teach your AI assistant how to use xcaffold.\n")

	return nil
}

// wizardAnswers holds answers collected during the interactive wizard.
type wizardAnswers struct {
	name    string
	desc    string
	targets []string
}

// knownCLIs lists the AI coding CLIs xcaffold knows about, in detection order.
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

// detectDefaultTarget returns the target for the first CLI binary found on PATH.
func detectDefaultTarget() string {
	for _, cli := range knownCLIs {
		if _, err := exec.LookPath(cli.binary); err == nil {
			return cli.target
		}
	}
	return targetClaude
}

// resolveTargetMeta returns the suggested model and binary name for a target.
func resolveTargetMeta(target string) (model, binary string) {
	for _, cli := range knownCLIs {
		if cli.target == target {
			return cli.model, cli.binary
		}
	}
	return "claude-sonnet-4-6", targetClaude
}

// collectWizardAnswers populates wizard answers from flags and optional prompts.
// Only one question is asked: target platforms. Project name is derived from CWD.
func collectWizardAnswers(defaultName string) (ans wizardAnswers, err error) {
	ans.name = defaultName
	ans.desc = ""
	if len(targetsFlag) > 0 {
		ans.targets = targetsFlag
	} else {
		ans.targets = []string{detectDefaultTarget()}
	}

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
	return
}

// writeXCFDirectory generates the xcaffold authoring toolkit scaffold structure.
func writeXCFDirectory(baseDir string, ans wizardAnswers) error {
	dirs := []string{
		filepath.Join(baseDir, "xcf", "agents", "xaff"),
		filepath.Join(baseDir, "xcf", "skills", "xcaffold"),
		filepath.Join(baseDir, "xcf", "rules", "xcf-conventions"),
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

	// .xcaffold/project.xcf
	projectContent := templates.RenderProjectXCF(ans.name, ans.targets)
	outDir := filepath.Join(baseDir, ".xcaffold")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("failed to create .xcaffold/: %w", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "project.xcf"), []byte(projectContent), 0o600); err != nil {
		return err
	}

	// xcf/agents/xaff/agent.xcf (base)
	agentBase := templates.RenderXaffAgentXCF(model, ans.targets)
	if err := os.WriteFile(filepath.Join(baseDir, "xcf", "agents", "xaff", "agent.xcf"), []byte(agentBase), 0o600); err != nil {
		return err
	}

	// xcf/agents/xaff/agent.<provider>.xcf (one per target)
	for _, target := range ans.targets {
		override := templates.RenderXaffOverrideXCF(target)
		filename := fmt.Sprintf("agent.%s.xcf", target)
		if err := os.WriteFile(filepath.Join(baseDir, "xcf", "agents", "xaff", filename), []byte(override), 0o600); err != nil {
			return err
		}
	}

	// xcf/skills/xcaffold/xcaffold.xcf
	skillContent := templates.RenderXcaffoldSkillXCF(ans.targets)
	if err := os.WriteFile(filepath.Join(baseDir, "xcf", "skills", "xcaffold", "xcaffold.xcf"), []byte(skillContent), 0o600); err != nil {
		return err
	}

	// xcf/rules/xcf-conventions/xcf-conventions.xcf
	ruleContent := templates.RenderXcfConventionsRuleXCF(ans.targets)
	if err := os.WriteFile(filepath.Join(baseDir, "xcf", "rules", "xcf-conventions", "xcf-conventions.xcf"), []byte(ruleContent), 0o600); err != nil {
		return err
	}

	// xcf/settings.xcf
	settingsContent := templates.RenderSettingsXCF(ans.targets)
	if err := os.WriteFile(filepath.Join(baseDir, "xcf", "settings.xcf"), []byte(settingsContent), 0o600); err != nil {
		return err
	}

	// xcf/policies/
	if !noPoliciesFlag {
		descPolicy := templates.RenderPolicyDescriptionXCF()
		_ = os.WriteFile(filepath.Join(baseDir, "xcf", "policies", "require-agent-description.xcf"), []byte(descPolicy), 0o600)
		instrPolicy := templates.RenderPolicyInstructionsXCF()
		_ = os.WriteFile(filepath.Join(baseDir, "xcf", "policies", "require-agent-instructions.xcf"), []byte(instrPolicy), 0o600)
	}

	return nil
}

// writeReferenceTemplates writes all 8 .xcaffold/schemas/*.reference files.
func writeReferenceTemplates(baseDir string) error {
	refDir := filepath.Join(baseDir, ".xcaffold", "schemas")
	if err := os.MkdirAll(refDir, 0o755); err != nil {
		return fmt.Errorf("failed to create schemas directory: %w", err)
	}

	refs := map[string]func() string{
		"agent.xcf.reference":      templates.RenderAgentReference,
		"skill.xcf.reference":      templates.RenderSkillReference,
		"rule.xcf.reference":       templates.RenderRuleReference,
		"workflow.xcf.reference":   templates.RenderWorkflowReference,
		"mcp.xcf.reference":        templates.RenderMCPReference,
		"hooks.xcf.reference":      templates.RenderHooksReference,
		"memory.xcf.reference":     templates.RenderMemoryReference,
		"cli-cheatsheet.reference": templates.RenderCLICheatsheet,
	}

	for filename, renderFn := range refs {
		path := filepath.Join(refDir, filename)
		if err := os.WriteFile(path, []byte(renderFn()), 0o600); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}
	return nil
}

// initGlobal reports that global scope is not yet available.
// The implementation is preserved as initGlobalImpl for future enablement.
func initGlobal() error {
	fmt.Printf("\n  %s Global scope is not available yet.\n", glyphErr())
	fmt.Printf("\n%s Run 'xcaffold init' to initialize a project-level scaffold.\n", glyphArrow())
	return nil
}

// initGlobalImpl contains the original global init logic, preserved for future use.
func initGlobalImpl() error {
	home, err := registry.GlobalHome()
	if err != nil {
		return err
	}
	target := filepath.Join(home, "global.xcf")

	if err := registry.RebuildGlobalXCF(); err != nil {
		return fmt.Errorf("failed to rebuild global.xcf: %w", err)
	}

	if _, err := os.Stat(target); err == nil {
		fmt.Printf("%s %s rebuilt from global platform directories.\n", colorGreen(glyphOK()), target)
	} else {
		fmt.Printf("%s %s created.\n", colorGreen(glyphOK()), target)
	}
	fmt.Println("  Edit it to define your global agents, then run 'xcaffold apply --global'.")
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

// injectXaffToolkitAfterImport writes the full Xaff authoring toolkit after an import.
// It replaces injectXcaffoldSkillAfterImport, which only wrote the skill.
func injectXaffToolkitAfterImport(baseDir string) error {
	xcfFile := filepath.Join(baseDir, ".xcaffold", "project.xcf")
	config, err := parser.ParseFileExact(xcfFile)
	if err != nil {
		return fmt.Errorf("parsing imported scaffold: %w", err)
	}

	targets := []string{"claude"}
	if config.Project != nil && len(config.Project.Targets) > 0 {
		targets = config.Project.Targets
	} else if config.Project == nil {
		config.Project = &ast.ProjectConfig{Name: filepath.Base(baseDir), Targets: targets}
	}

	model, _ := resolveTargetMeta(targets[0])

	// Write Xaff agent (base + overrides)
	xaffDir := filepath.Join(baseDir, "xcf", "agents", "xaff")
	if err := os.MkdirAll(xaffDir, 0o755); err != nil {
		return err
	}
	agentBase := templates.RenderXaffAgentXCF(model, targets)
	if err := os.WriteFile(filepath.Join(xaffDir, "agent.xcf"), []byte(agentBase), 0o600); err != nil {
		return err
	}
	for _, target := range targets {
		override := templates.RenderXaffOverrideXCF(target)
		filename := fmt.Sprintf("agent.%s.xcf", target)
		if err := os.WriteFile(filepath.Join(xaffDir, filename), []byte(override), 0o600); err != nil {
			return err
		}
	}

	// Write xcaffold skill (directory-per-resource)
	skillDir := filepath.Join(baseDir, "xcf", "skills", "xcaffold")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		return err
	}
	skillContent := templates.RenderXcaffoldSkillXCF(targets)
	if err := os.WriteFile(filepath.Join(skillDir, "xcaffold.xcf"), []byte(skillContent), 0o600); err != nil {
		return err
	}

	// Write xcf-conventions rule (directory-per-resource)
	ruleDir := filepath.Join(baseDir, "xcf", "rules", "xcf-conventions")
	if err := os.MkdirAll(ruleDir, 0o755); err != nil {
		return err
	}
	ruleContent := templates.RenderXcfConventionsRuleXCF(targets)
	if err := os.WriteFile(filepath.Join(ruleDir, "xcf-conventions.xcf"), []byte(ruleContent), 0o600); err != nil {
		return err
	}

	// Update project.xcf skill refs if needed
	hasSkill := false
	for _, s := range config.Project.SkillRefs {
		if s == "xcaffold" {
			hasSkill = true
			break
		}
	}
	if !hasSkill {
		config.Project.SkillRefs = append(config.Project.SkillRefs, "xcaffold")
		if err := WriteProjectFile(config, baseDir); err != nil {
			return fmt.Errorf("writing updated project.xcf: %w", err)
		}
	}

	return nil
}
