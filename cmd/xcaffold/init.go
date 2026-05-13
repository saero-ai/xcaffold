package main

import (
	"bytes"
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
var upgradeFlag bool

var targetsFlag []string
var jsonManifestFlag bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap a new project.xcaf configuration",
	Long: `xcaffold init bootstraps a new project.

 • Detects existing platform config (.claude/, .cursor/, .agents/) and offers to run 'xcaffold import'.
 • Guides you through an interactive wizard for new projects.
 • Generates the Xaff authoring toolkit: agent, xcaffold skill, xcaf-conventions rule, and schema references.
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
	initCmd.Flags().BoolVar(&upgradeFlag, "upgrade", false, "Force-refresh toolkit files to latest embedded versions")
	initCmd.Flags().StringSliceVar(&targetsFlag, "target", nil, fmt.Sprintf("compilation target(s): %s", strings.Join(providers.PrimaryNames(), ", ")))
	initCmd.Flags().BoolVar(&jsonManifestFlag, "json", false, "Output machine-readable JSON manifest instead of interactive logs")
	rootCmd.AddCommand(initCmd)
}

// toolkitDiff tracks which toolkit files are updated, new, or unchanged.
type toolkitDiff struct {
	Updated   []string
	New       []string
	Unchanged []string
}

// buildToolkitFileMap constructs a map of embedded toolkit file paths to disk-relative paths.
// Targets are used to include provider-specific override files.
func buildToolkitFileMap(targets []string) map[string]string {
	files := map[string]string{
		"toolkit/agents/xaff/agent.xcaf":                           "xcaf/agents/xaff/agent.xcaf",
		"toolkit/skills/xcaffold/skill.xcaf":                       "xcaf/skills/xcaffold/skill.xcaf",
		"toolkit/skills/xcaffold/references/operating-guide.md":    "xcaf/skills/xcaffold/references/operating-guide.md",
		"toolkit/skills/xcaffold/references/authoring-guide.md":    "xcaf/skills/xcaffold/references/authoring-guide.md",
		"toolkit/skills/xcaffold/references/agent-reference.md":    "xcaf/skills/xcaffold/references/agent-reference.md",
		"toolkit/skills/xcaffold/references/skill-reference.md":    "xcaf/skills/xcaffold/references/skill-reference.md",
		"toolkit/skills/xcaffold/references/rule-reference.md":     "xcaf/skills/xcaffold/references/rule-reference.md",
		"toolkit/skills/xcaffold/references/workflow-reference.md": "xcaf/skills/xcaffold/references/workflow-reference.md",
		"toolkit/skills/xcaffold/references/mcp-reference.md":      "xcaf/skills/xcaffold/references/mcp-reference.md",
		"toolkit/skills/xcaffold/references/hooks-reference.md":    "xcaf/skills/xcaffold/references/hooks-reference.md",
		"toolkit/skills/xcaffold/references/memory-reference.md":   "xcaf/skills/xcaffold/references/memory-reference.md",
		"toolkit/skills/xcaffold/references/context-reference.md":  "xcaf/skills/xcaffold/references/context-reference.md",
		"toolkit/skills/xcaffold/references/settings-reference.md": "xcaf/skills/xcaffold/references/settings-reference.md",
		"toolkit/skills/xcaffold/references/cli-cheatsheet.md":     "xcaf/skills/xcaffold/references/cli-cheatsheet.md",
		"toolkit/rules/xcaf-conventions/rule.xcaf":                 "xcaf/rules/xcaf-conventions/rule.xcaf",
	}

	for _, target := range targets {
		embedKey := fmt.Sprintf("toolkit/agents/xaff/agent.%s.xcaf", target)
		diskKey := fmt.Sprintf("xcaf/agents/xaff/agent.%s.xcaf", target)
		files[embedKey] = diskKey
	}

	return files
}

// compareToolkitFiles compares embedded toolkit files against existing xcaf/ files.
// It returns a toolkitDiff indicating which files are updated, new, or unchanged.
func compareToolkitFiles(baseDir string, targets []string) toolkitDiff {
	files := buildToolkitFileMap(targets)
	var diff toolkitDiff

	for embedPath, diskRel := range files {
		diskPath := filepath.Join(baseDir, diskRel)
		embedded, err := templates.ToolkitFS.ReadFile(embedPath)
		if err != nil {
			continue
		}
		existing, err := os.ReadFile(diskPath)
		if err != nil {
			diff.New = append(diff.New, diskRel)
			continue
		}
		if !bytes.Equal(embedded, existing) {
			diff.Updated = append(diff.Updated, diskRel)
		} else {
			diff.Unchanged = append(diff.Unchanged, diskRel)
		}
	}

	sort.Strings(diff.Updated)
	sort.Strings(diff.New)
	sort.Strings(diff.Unchanged)

	return diff
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
// It orchestrates four cases: existing scaffold with no providers (B),
// provider dirs detected (C), or new project wizard (D).
func initProject(cmd *cobra.Command) error {
	xcafFile := "project.xcaf"

	var currentConfig *ast.XcaffoldConfig
	hasExistingScaffold := false
	if _, err := os.Stat(xcafFile); err == nil {
		hasExistingScaffold = true
		currentConfig, _ = parser.ParseFile(xcafFile)
	}

	detected := importer.DetectProviders(".", importer.DefaultImporters())

	// Case B: Existing scaffold, no provider dirs
	if hasExistingScaffold && len(detected) == 0 {
		return handleExistingScaffoldNoProviders(cmd, xcafFile)
	}

	// Case C: Provider dirs detected (offer import)
	if len(detected) > 0 {
		handled, err := handleProviderDetected(cmd, detected, hasExistingScaffold, currentConfig, xcafFile)
		if err != nil || handled {
			return err
		}
		// Fall through to wizard if not handled
	}

	// Case D: New project (wizard)
	if hasExistingScaffold {
		return nil
	}
	return runWizard(cmd, xcafFile)
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
func runWizard(cmd *cobra.Command, xcafFile string) error {
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
		if err := resolveInitTargets(&ans); err != nil {
			return err
		}
	}

	if err := writeXCAFDirectory(cwd, ans); err != nil {
		return fmt.Errorf("failed to scaffold directory: %w", err)
	}

	if err := registry.Register(cwd, ans.name, ans.targets, "."); err != nil && !jsonManifestFlag {
		fmt.Printf("  %s Failed to register project: %v\n", glyphErr(), err)
	}

	if jsonManifestFlag {
		return outputInitManifest(cmd, &ans)
	}

	printInitSuccess(&ans)
	return nil
}

// resolveInitTargets determines the targets for JSON manifest mode.
func resolveInitTargets(ans *wizardAnswers) error {
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
	return nil
}

// outputInitManifest generates and outputs the JSON manifest for init.
func outputInitManifest(cmd *cobra.Command, ans *wizardAnswers) error {
	type Manifest struct {
		Project string   `json:"project"`
		Targets []string `json:"targets"`
		Files   []string `json:"files"`
	}

	files := buildInitFiles(ans.targets)
	b, err := json.MarshalIndent(Manifest{Project: ans.name, Targets: ans.targets, Files: files}, "", "  ")
	if err == nil {
		cmd.Println(string(b))
	}
	return nil
}

// buildInitFiles constructs the list of files created by init.
func buildInitFiles(targets []string) []string {
	files := []string{
		"project.xcaf",
		"xcaf/agents/xaff/agent.xcaf",
	}
	for _, t := range targets {
		files = append(files, fmt.Sprintf("xcaf/agents/xaff/agent.%s.xcaf", t))
	}
	files = append(files,
		"xcaf/skills/xcaffold/skill.xcaf",
		"xcaf/skills/xcaffold/references/operating-guide.md",
		"xcaf/skills/xcaffold/references/authoring-guide.md",
		"xcaf/rules/xcaf-conventions/rule.xcaf",
	)
	for _, ref := range []string{"agent", "skill", "rule", "workflow", "mcp", "hooks", "memory", "context", "settings"} {
		files = append(files, fmt.Sprintf("xcaf/skills/xcaffold/references/%s-reference.md", ref))
	}
	files = append(files, "xcaf/skills/xcaffold/references/cli-cheatsheet.md")
	return files
}

// printInitSuccess outputs the success message for init.
func printInitSuccess(ans *wizardAnswers) {
	fmt.Println()
	fmt.Printf("  %s project.xcaf\n", colorGreen(glyphOK()))
	fmt.Printf("  %s xcaf/agents/xaff/                     %s\n",
		colorGreen(glyphOK()), dim(fmt.Sprintf("base + %d %s", len(ans.targets), plural(len(ans.targets), "override", "overrides"))))
	fmt.Printf("  %s xcaf/skills/xcaffold/\n", colorGreen(glyphOK()))
	fmt.Printf("  %s xcaf/rules/xcaf-conventions/\n", colorGreen(glyphOK()))
	fmt.Printf("  %s xcaf/skills/xcaffold/references/    %s\n",
		colorGreen(glyphOK()), dim("12 references"))
	fmt.Println()
	fmt.Printf("%s Run 'xcaffold validate' then 'xcaffold apply'.\n", glyphArrow())
	fmt.Printf("  Includes Xaff agent, xcaffold skill, and xcaf-conventions rule.\n")
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

// handleExistingScaffoldNoProviders handles the case where project.xcaf exists
// but no provider directories are detected (Case B).
func handleExistingScaffoldNoProviders(cmd *cobra.Command, xcafFile string) error {
	cwd, _ := os.Getwd()
	projectName := filepath.Base(cwd)
	fmt.Println(formatHeader(projectName, "", false, "", "already initialized"))
	fmt.Println()
	fmt.Printf("  %s project.xcaf exists.\n", glyphNever())
	fmt.Println()
	fmt.Printf("%s Run 'xcaffold apply' to compile your xcaf/ sources.\n", glyphArrow())
	fmt.Printf("  Run 'xcaffold import' to sync provider changes back to xcaf/.\n")
	tryAutoRegister(xcafFile)
	return nil
}

// handleToolkitUpdate manages the toolkit file comparison and update flow
// when re-initializing an existing scaffold without importing.
func handleToolkitUpdate(currentConfig *ast.XcaffoldConfig) error {
	targets := extractTargets(currentConfig)
	diff := compareToolkitFiles(".", targets)

	if len(diff.Updated)+len(diff.New) == 0 && !upgradeFlag {
		fmt.Printf("\n  %s  Toolkit files are up to date.\n", colorGreen(glyphOK()))
		fmt.Printf("  %s  project.xcaf exists.\n", colorGreen(glyphOK()))
		return nil
	}

	printToolkitPreview(diff)
	doUpdate := yesFlag || upgradeFlag
	if !doUpdate {
		var err error
		doUpdate, err = prompt.Confirm("Update toolkit files?", false)
		if err != nil {
			return fmt.Errorf("prompt error: %w", err)
		}
	}

	if doUpdate {
		if err := applyToolkitUpdate(".", targets, diff); err != nil {
			return err
		}
	}
	return nil
}

// extractTargets extracts targets from the current config.
func extractTargets(currentConfig *ast.XcaffoldConfig) []string {
	if currentConfig != nil && currentConfig.Project != nil {
		return currentConfig.Project.Targets
	}
	return nil
}

// printToolkitPreview displays the toolkit diff preview.
func printToolkitPreview(diff toolkitDiff) {
	fmt.Println()
	fmt.Println("  Toolkit:")
	for _, f := range diff.Unchanged {
		if upgradeFlag {
			fmt.Printf("    %s  force-update    %s\n", colorYellow(glyphSrc()), f)
		} else {
			fmt.Printf("    %s  unchanged      %s\n", colorGreen(glyphOK()), f)
		}
	}
	for _, f := range diff.Updated {
		fmt.Printf("    %s  updated        %s\n", colorYellow(glyphSrc()), f)
	}
	for _, f := range diff.New {
		fmt.Printf("    %s  new            %s\n", colorGreen("+"), f)
	}
	fmt.Printf("\n  %d updated, %d new, %d unchanged\n",
		len(diff.Updated), len(diff.New), len(diff.Unchanged))
}

// applyToolkitUpdate copies toolkit files and prints success message.
func applyToolkitUpdate(basePath string, targets []string, diff toolkitDiff) error {
	toolkitFiles := buildToolkitFileMap(targets)
	filtered := make(map[string]string)
	filesToCopy := diff.Updated
	filesToCopy = append(filesToCopy, diff.New...)
	if upgradeFlag {
		filesToCopy = append(filesToCopy, diff.Unchanged...)
	}
	for embed, disk := range toolkitFiles {
		for _, f := range filesToCopy {
			if disk == f {
				filtered[embed] = disk
			}
		}
	}
	if err := copyToolkitFiles(basePath, filtered); err != nil {
		return fmt.Errorf("toolkit update: %w", err)
	}

	fileCount := len(diff.Updated) + len(diff.New)
	if upgradeFlag {
		fileCount += len(diff.Unchanged)
		fmt.Printf("\n  %s  Force-refreshed %d file(s).\n", colorGreen(glyphOK()), fileCount)
	} else {
		fmt.Printf("\n  %s  Updated %d file(s), added %d file(s).\n",
			colorGreen(glyphOK()), len(diff.Updated), len(diff.New))
	}
	return nil
}

// handleProviderImport orchestrates the provider import flow,
// handling single and multi-directory imports.
func handleProviderImport(cmd *cobra.Command, detected []importer.ProviderImporter, xcafFile string) error {
	importErr := selectAndImportProviders(detected, xcafFile)
	if importErr != nil {
		return importErr
	}

	if err := injectXaffToolkitAfterImport("."); err != nil {
		fmt.Printf("  %s Failed to inject xcaffold toolkit: %v\n", glyphErr(), err)
	} else {
		printImportSuccess()
	}
	return nil
}

// selectAndImportProviders handles provider selection and import.
func selectAndImportProviders(detected []importer.ProviderImporter, xcafFile string) error {
	if len(detected) == 1 {
		return importScope(detected[0].InputDir(), xcafFile, "project", detected[0].Provider())
	}

	if yesFlag {
		return mergeImportDirs(detected, xcafFile)
	}

	selected, err := promptMultiSelectDirs(detected)
	if err != nil {
		return err
	}
	if len(selected) == 0 {
		fmt.Printf("\n  %s No directories selected. Continuing with fresh scaffold.\n", glyphNever())
		return nil
	}

	if len(selected) == 1 {
		fmt.Println()
		provider := findProviderForDir(detected, selected[0])
		return importScope(selected[0], xcafFile, "project", provider)
	}

	fmt.Println()
	selectedImps := filterImportersByDirs(detected, selected)
	return mergeImportDirs(selectedImps, xcafFile)
}

// promptMultiSelectDirs prompts the user to select directories.
func promptMultiSelectDirs(detected []importer.ProviderImporter) ([]string, error) {
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
		return nil, fmt.Errorf("prompt error: %w", err)
	}
	return selected, nil
}

// findProviderForDir finds the provider for a given directory.
func findProviderForDir(detected []importer.ProviderImporter, dir string) string {
	for _, imp := range detected {
		if imp.InputDir() == dir {
			return imp.Provider()
		}
	}
	return ""
}

// filterImportersByDirs returns importers matching the given directories.
func filterImportersByDirs(detected []importer.ProviderImporter, dirs []string) []importer.ProviderImporter {
	var result []importer.ProviderImporter
	for _, s := range dirs {
		for _, imp := range detected {
			if imp.InputDir() == s {
				result = append(result, imp)
				break
			}
		}
	}
	return result
}

// printImportSuccess prints the import success message.
func printImportSuccess() {
	fmt.Println()
	fmt.Printf("  %s xcaf/agents/xaff/\n", colorGreen(glyphOK()))
	fmt.Printf("  %s xcaf/skills/xcaffold/\n", colorGreen(glyphOK()))
	fmt.Printf("  %s xcaf/rules/xcaf-conventions/\n", colorGreen(glyphOK()))
	fmt.Printf("  %s xcaf/skills/xcaffold/references/    %s\n",
		colorGreen(glyphOK()), dim("12 references"))
	fmt.Println()
	fmt.Printf("%s Run 'xcaffold validate' then 'xcaffold apply'.\n", glyphArrow())
}

// handleProviderDetected orchestrates the flow when provider directories are detected
// (Case C), offering import with toolkit update or fallback to new project wizard.
func handleProviderDetected(cmd *cobra.Command, detected []importer.ProviderImporter,
	hasExistingScaffold bool, currentConfig *ast.XcaffoldConfig, xcafFile string) (bool, error) {

	printProviderDetectionHeader(hasExistingScaffold)
	if currentConfig != nil {
		renderCurrentStateTable(cmd, currentConfig)
		fmt.Println()
	}
	renderCompiledOutputTable(cmd, detected)

	doImport, err := promptForImport(detected, hasExistingScaffold)
	if err != nil {
		return false, err
	}

	if !doImport {
		if hasExistingScaffold {
			if err := handleToolkitUpdate(currentConfig); err != nil {
				return false, err
			}
			tryAutoRegister(xcafFile)
			return true, nil
		}
		fmt.Printf("\n  %s Skipping import. Continuing with fresh scaffold.\n", glyphNever())
		fmt.Println()
		return false, nil
	}

	if err := handleProviderImport(cmd, detected, xcafFile); err != nil {
		return false, err
	}
	return true, nil
}

// printProviderDetectionHeader prints the initial detection message.
func printProviderDetectionHeader(hasExistingScaffold bool) {
	fmt.Println()
	if hasExistingScaffold {
		fmt.Println("  project.xcaf already exists, but existing compiled configurations were detected.")
	} else {
		fmt.Printf("  %s Detected existing agent configuration(s):\n", glyphOK())
	}
	fmt.Println()
}

// promptForImport asks the user whether to import detected providers.
func promptForImport(detected []importer.ProviderImporter, hasExistingScaffold bool) (bool, error) {
	if hasExistingScaffold {
		return promptForReImport()
	}
	return promptForNewImport(detected)
}

// promptForReImport handles the import prompt for existing scaffold.
func promptForReImport() (bool, error) {
	fmt.Println()
	fmt.Fprintf(os.Stderr, "  %s  Re-init will update toolkit files and can reimport from detected providers.\n", colorYellow(glyphSrc()))
	fmt.Fprintf(os.Stderr, "      User-authored resources (agents, skills, rules) are preserved.\n")
	fmt.Fprintf(os.Stderr, "      To import provider changes, use 'xcaffold import' instead.\n\n")
	if yesFlag {
		return true, nil
	}
	doImport, err := prompt.Confirm("Import detected providers into project.xcaf?", false)
	if err != nil {
		return false, fmt.Errorf("prompt error: %w", err)
	}
	return doImport, nil
}

// promptForNewImport handles the import prompt for new projects.
func promptForNewImport(detected []importer.ProviderImporter) (bool, error) {
	fmt.Println()
	if yesFlag {
		return true, nil
	}
	fmtStr := "Import %s into project.xcaf?"
	if len(detected) > 1 {
		fmt.Println("  xcaffold consolidates multiple configs into one project.xcaf.")
		fmtStr = "Import these directories into project.xcaf?"
	} else {
		fmtStr = fmt.Sprintf(fmtStr, detected[0].InputDir())
	}
	doImport, err := prompt.Confirm(fmtStr, true)
	if err != nil {
		return false, fmt.Errorf("prompt error: %w", err)
	}
	return doImport, nil
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

// writeXCAFDirectory generates the xcaffold authoring toolkit scaffold structure.
func writeXCAFDirectory(baseDir string, ans wizardAnswers) error {
	files := buildToolkitFileMap(ans.targets)

	if err := copyToolkitFiles(baseDir, files); err != nil {
		return err
	}

	// Write project.xcaf
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{
			Name:    ans.name,
			Targets: ans.targets,
		},
	}
	return WriteProjectFile(config, baseDir)
}

// initGlobal reports that global scope is not yet available.
func initGlobal() error {
	fmt.Printf("\n  %s Global scope is not available yet.\n", glyphErr())
	fmt.Printf("\n%s Run 'xcaffold init' to initialize a project-level scaffold.\n", glyphArrow())
	return nil
}

func tryAutoRegister(xcafFile string) {
	config, err := parser.ParseFile(xcafFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to parse %s for auto-registration: %v\n", xcafFile, err)
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
	// Check for project.xcaf in root
	xcafFile := filepath.Join(baseDir, "project.xcaf")
	config, err := parser.ParseFileExact(xcafFile)
	if err != nil {
		return fmt.Errorf("parsing imported scaffold: %w", err)
	}

	var targets []string
	if config.Project != nil && len(config.Project.Targets) > 0 {
		targets = config.Project.Targets
	} else if config.Project == nil {
		config.Project = &ast.ProjectConfig{Name: filepath.Base(baseDir), Targets: targets}
	}

	files := buildToolkitFileMap(targets)
	if err := copyToolkitFiles(baseDir, files); err != nil {
		return err
	}

	return nil
}
