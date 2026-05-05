package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/importer"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/prompt"
	"github.com/saero-ai/xcaffold/internal/registry"
	"github.com/saero-ai/xcaffold/internal/templates"
	"github.com/saero-ai/xcaffold/providers"
	"github.com/spf13/cobra"
)

var yesFlag bool

var targetsFlag []string
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
	RunE:         runInit,
	SilenceUsage: true,
}

func init() {
	initCmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Accept all defaults non-interactively (CI/CD mode)")
	initCmd.Flags().StringSliceVar(&targetsFlag, "target", nil, fmt.Sprintf("compilation target(s): %s", strings.Join(providers.PrimaryNames(), ", ")))
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
	fmt.Println("  Initializing xcaffold project.")

	if globalFlag {
		return initGlobal()
	}

	return initProject(cmd)
}

// initProject runs the interactive project-level init wizard.
func initProject(cmd *cobra.Command) error {
	xcfFile := "project.xcf"

	var currentConfig *ast.XcaffoldConfig
	hasExistingScaffold := false
	if _, err := os.Stat(xcfFile); err == nil {
		hasExistingScaffold = true
		currentConfig, _ = parser.ParseFile(xcfFile)
	}

	detected := importer.DetectProviders(".", importer.DefaultImporters())

	// ── Case B: Existing scaffold, no provider dirs ───────────────────────────
	if hasExistingScaffold && len(detected) == 0 {
		cwd, _ := os.Getwd()
		projectName := filepath.Base(cwd)
		fmt.Println(formatHeader(projectName, "", false, "", "already initialized"))
		fmt.Println()
		fmt.Printf("  %s project.xcf exists.\n", glyphNever())
		fmt.Println()
		fmt.Printf("%s Run 'xcaffold apply' to compile your xcf/ sources.\n", glyphArrow())
		fmt.Printf("  Run 'xcaffold import' to sync provider changes back to xcf/.\n")
		tryAutoRegister(xcfFile)
		return nil
	}

	// ── Case C: Provider dirs detected (offer import) ─────────────────────────
	if len(detected) > 0 {
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
		renderCompiledOutputTable(cmd, detected)

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
				if len(detected) > 1 {
					fmt.Println("  xcaffold consolidates multiple configs into one project.xcf.")
					fmtStr = "Import these directories into project.xcf?"
				} else {
					fmtStr = fmt.Sprintf(fmtStr, detected[0].InputDir())
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
			if len(detected) == 1 {
				importErr = importScope(detected[0].InputDir(), xcfFile, "project", detected[0].Provider())
			} else {
				if yesFlag {
					importErr = mergeImportDirs(detected, xcfFile)
				} else {
					var options []prompt.SelectOption
					for _, imp := range detected {
						options = append(options, prompt.SelectOption{
							Label:    imp.InputDir(),
							Value:    imp.InputDir(),
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
							// Find the importer for the selected directory
							var selectedProvider string
							for _, imp := range detected {
								if imp.InputDir() == selected[0] {
									selectedProvider = imp.Provider()
									break
								}
							}
							importErr = importScope(selected[0], xcfFile, "project", selectedProvider)
						} else {
							fmt.Println()
							var selectedImps []importer.ProviderImporter
							for _, s := range selected {
								for _, imp := range detected {
									if imp.InputDir() == s {
										selectedImps = append(selectedImps, imp)
										break
									}
								}
							}
							importErr = mergeImportDirs(selectedImps, xcfFile)
						}
					}
				}
			}

			if doImport {
				if importErr != nil {
					return importErr
				}
				importedConfig, parseErr := parser.ParseFile(xcfFile)
				if parseErr != nil {
					return fmt.Errorf("parsing imported scaffold: %w", parseErr)
				}
				if err := runPostImportSteps(importedConfig, ".", true); err != nil {
					fmt.Printf("  %s Failed to inject xcaffold toolkit: %v\n", glyphErr(), err)
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

// copyToolkitFiles copies files from the embedded toolkit FS to disk.
// paths maps embed paths (under "toolkit/") to disk paths (relative to baseDir).
func copyToolkitFiles(baseDir string, paths map[string]string) error {
	for embedPath, diskRel := range paths {
		data, err := templates.ToolkitFS.ReadFile(embedPath)
		if err != nil {
			return fmt.Errorf("reading embedded %s: %w", embedPath, err)
		}
		outPath := filepath.Join(baseDir, diskRel)
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(outPath, data, 0o644); err != nil {
			return err
		}
	}
	return nil
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
		detected := detectDefaultTarget()
		if detected != "" {
			ans.targets = []string{detected}
		}
		if len(targetsFlag) > 0 {
			ans.targets = targetsFlag
		}
		if len(ans.targets) == 0 {
			return fmt.Errorf("--target is required with --json when no CLI is detected on PATH")
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
			"project.xcf",
			"xcf/agents/xaff/agent.xcf",
		}
		for _, t := range ans.targets {
			files = append(files, fmt.Sprintf("xcf/agents/xaff/agent.%s.xcf", t))
		}
		files = append(files,
			"xcf/skills/xcaffold/skill.xcf",
			"xcf/skills/xcaffold/references/operating-guide.md",
			"xcf/skills/xcaffold/references/authoring-guide.md",
			"xcf/rules/xcf-conventions/rule.xcf",
		)
		for _, ref := range []string{"agent", "skill", "rule", "workflow", "mcp", "hooks", "memory"} {
			files = append(files, fmt.Sprintf(".xcaffold/schemas/%s-reference.md", ref))
		}
		files = append(files, ".xcaffold/schemas/cli-cheatsheet.md")

		b, err := json.MarshalIndent(Manifest{Project: ans.name, Targets: ans.targets, Files: files}, "", "  ")
		if err == nil {
			cmd.Println(string(b))
		}
		return nil
	}

	fmt.Println()
	fmt.Printf("  %s project.xcf\n", colorGreen(glyphOK()))
	fmt.Printf("  %s xcf/agents/xaff/                     %s\n",
		colorGreen(glyphOK()), dim(fmt.Sprintf("base + %d %s", len(ans.targets), plural(len(ans.targets), "override", "overrides"))))
	fmt.Printf("  %s xcf/skills/xcaffold/\n", colorGreen(glyphOK()))
	fmt.Printf("  %s xcf/rules/xcf-conventions/\n", colorGreen(glyphOK()))
	fmt.Printf("  %s .xcaffold/schemas/                    %s\n",
		colorGreen(glyphOK()), dim("8 references"))
	fmt.Println()
	fmt.Printf("%s Run 'xcaffold validate' then 'xcaffold apply'.\n", glyphArrow())
	fmt.Printf("  Includes Xaff agent, xcaffold skill, and xcf-conventions rule.\n")

	return nil
}

// wizardAnswers holds answers collected during the interactive wizard.
type wizardAnswers struct {
	name    string
	desc    string
	targets []string
}

// detectDefaultTarget returns the target for the first CLI binary found on PATH.
// Returns an empty string if no CLI is found.
func detectDefaultTarget() string {
	for _, m := range providers.Manifests() {
		if m.CLIBinary != "" {
			if _, err := exec.LookPath(m.CLIBinary); err == nil {
				return m.Name
			}
		}
	}
	return ""
}

// resolveTargetMeta returns the suggested model and binary name for a target.
// Returns empty strings if the target is not found.
func resolveTargetMeta(target string) (model, binary string) {
	m, ok := providers.ManifestFor(target)
	if !ok {
		return "", ""
	}
	return m.DefaultModel, m.CLIBinary
}

// collectWizardAnswers populates wizard answers from flags and optional prompts.
// Only one question is asked: target platforms. Project name is derived from CWD.
func collectWizardAnswers(defaultName string) (ans wizardAnswers, err error) {
	ans.name = defaultName
	ans.desc = ""
	if len(targetsFlag) > 0 {
		ans.targets = targetsFlag
	} else {
		detected := detectDefaultTarget()
		if detected != "" {
			ans.targets = []string{detected}
		}
	}

	if yesFlag {
		if len(ans.targets) == 0 {
			err = fmt.Errorf("--target is required with --yes when no CLI is detected on PATH")
			return
		}
		return
	}

	if len(targetsFlag) == 0 {
		time.Sleep(300 * time.Millisecond)
		defaultTarget := ""
		if len(ans.targets) > 0 {
			defaultTarget = ans.targets[0]
		}

		var options []prompt.SelectOption
		for _, m := range providers.Manifests() {
			label := m.DisplayLabel
			if label == "" {
				label = m.Name
			}
			options = append(options, prompt.SelectOption{
				Label:    label,
				Value:    m.Name,
				Selected: defaultTarget == m.Name,
			})
		}
		sort.Slice(options, func(i, j int) bool {
			return options[i].Label < options[j].Label
		})

		selected, promptErr := prompt.MultiSelect("Target platforms (space to select)", options)
		if promptErr != nil {
			err = promptErr
			return
		}
		if len(selected) > 0 {
			ans.targets = selected
		} else {
			err = fmt.Errorf("no target platforms selected — at least one is required")
			return
		}
	}
	return
}

// writeXCFDirectory generates the xcaffold authoring toolkit scaffold structure.
func writeXCFDirectory(baseDir string, ans wizardAnswers) error {
	// Build the file map: embedded path → disk path
	files := map[string]string{
		"toolkit/agents/xaff/agent.xcf":                         "xcf/agents/xaff/agent.xcf",
		"toolkit/skills/xcaffold/skill.xcf":                     "xcf/skills/xcaffold/skill.xcf",
		"toolkit/skills/xcaffold/references/operating-guide.md": "xcf/skills/xcaffold/references/operating-guide.md",
		"toolkit/skills/xcaffold/references/authoring-guide.md": "xcf/skills/xcaffold/references/authoring-guide.md",
		"toolkit/rules/xcf-conventions/rule.xcf":                "xcf/rules/xcf-conventions/rule.xcf",
	}

	// Add provider override files for selected targets only
	for _, target := range ans.targets {
		embedKey := fmt.Sprintf("toolkit/agents/xaff/agent.%s.xcf", target)
		diskKey := fmt.Sprintf("xcf/agents/xaff/agent.%s.xcf", target)
		files[embedKey] = diskKey
	}

	if err := copyToolkitFiles(baseDir, files); err != nil {
		return err
	}

	// Write project.xcf
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{
			Name:    ans.name,
			Targets: ans.targets,
		},
	}
	return WriteProjectFile(config, baseDir)
}

// writeReferenceTemplates writes all 8 .xcaffold/schemas/ reference files.
func writeReferenceTemplates(baseDir string) error {
	schemas := []string{
		"agent-reference.md",
		"skill-reference.md",
		"rule-reference.md",
		"workflow-reference.md",
		"mcp-reference.md",
		"hooks-reference.md",
		"memory-reference.md",
		"cli-cheatsheet.md",
	}
	files := make(map[string]string, len(schemas))
	for _, s := range schemas {
		files[fmt.Sprintf("toolkit/schemas/%s", s)] = fmt.Sprintf(".xcaffold/schemas/%s", s)
	}
	return copyToolkitFiles(baseDir, files)
}

// initGlobal reports that global scope is not yet available.
func initGlobal() error {
	fmt.Printf("\n  %s Global scope is not available yet.\n", glyphErr())
	fmt.Printf("\n%s Run 'xcaffold init' to initialize a project-level scaffold.\n", glyphArrow())
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
	// Check for project.xcf in root
	xcfFile := filepath.Join(baseDir, "project.xcf")
	config, err := parser.ParseFileExact(xcfFile)
	if err != nil {
		return fmt.Errorf("parsing imported scaffold: %w", err)
	}

	var targets []string
	if config.Project != nil && len(config.Project.Targets) > 0 {
		targets = config.Project.Targets
	} else if config.Project == nil {
		config.Project = &ast.ProjectConfig{Name: filepath.Base(baseDir), Targets: targets}
	}

	// Build the file map: embedded path → disk path
	files := map[string]string{
		"toolkit/agents/xaff/agent.xcf":                         "xcf/agents/xaff/agent.xcf",
		"toolkit/skills/xcaffold/skill.xcf":                     "xcf/skills/xcaffold/skill.xcf",
		"toolkit/skills/xcaffold/references/operating-guide.md": "xcf/skills/xcaffold/references/operating-guide.md",
		"toolkit/skills/xcaffold/references/authoring-guide.md": "xcf/skills/xcaffold/references/authoring-guide.md",
		"toolkit/rules/xcf-conventions/rule.xcf":                "xcf/rules/xcf-conventions/rule.xcf",
	}

	// Add provider override files for the targets in the project config
	for _, target := range targets {
		embedKey := fmt.Sprintf("toolkit/agents/xaff/agent.%s.xcf", target)
		diskKey := fmt.Sprintf("xcf/agents/xaff/agent.%s.xcf", target)
		files[embedKey] = diskKey
	}

	if err := copyToolkitFiles(baseDir, files); err != nil {
		return err
	}

	return nil
}
