package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/blueprint"
	"github.com/saero-ai/xcaffold/internal/compiler"
	"github.com/saero-ai/xcaffold/internal/optimizer"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/policy"
	"github.com/saero-ai/xcaffold/internal/prompt"
	"github.com/saero-ai/xcaffold/internal/registry"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/resolver"
	"github.com/saero-ai/xcaffold/internal/state"
	"github.com/saero-ai/xcaffold/providers"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var applyDryRun bool
var applyForce bool
var applyBackup bool
var applyYes bool
var applyProjectFlag string
var applyBlueprintFlag string
var targetFlag string
var varFileFlag string

const scopeGlobal = "global"

// applyContext bundles parameters for the apply phase to reduce function arity.
type applyContext struct {
	config        *ast.XcaffoldConfig
	oldManifest   *state.StateManifest
	out           *output.Output
	sourceFiles   []string
	configPath    string
	projectName   string
	outputDir     string
	baseDir       string
	stateFilePath string
	scopeName     string
}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Compile .xcaf resources into provider-native agent files",
	Long: `Deterministically compiles .xcaf resources into provider-native agent files.

  - Strict one-way generation (YAML -> provider-native markdown/JSON)
  - Generates a SHA-256 state manifest for drift detection (.xcaffold/)
  - Automatically purges orphaned target files

Any manually edited files inside the target directory will be overwritten.`,
	Example: `  $ xcaffold apply
  $ xcaffold apply --dry-run
  $ xcaffold apply --target cursor`,
	RunE:          runApply,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// prepareSourceFiles collects and filters source files for compilation.
// It excludes registry-kind files and includes variable files.
func prepareSourceFiles(baseDir, target, varFile string) ([]string, error) {
	sourceFiles, findErr := resolver.FindXCAFFiles(baseDir)
	if findErr != nil {
		fmt.Fprintf(os.Stderr, "  Warning: failed to scan source files: %v\n", findErr)
	}

	// Filter out non-config XCAF files (e.g. kind: registry) to prevent
	// SourcesChanged from detecting registry mutations as config changes.
	var configSources []string
	for _, f := range sourceFiles {
		data, readErr := os.ReadFile(f)
		if readErr != nil {
			configSources = append(configSources, f)
			continue
		}
		var header struct {
			Kind string `yaml:"kind"`
		}
		if yaml.Unmarshal(data, &header) == nil && header.Kind == "registry" {
			continue
		}
		configSources = append(configSources, f)
	}

	// Add variable files to sources for drift detection
	configSources = append(configSources, resolver.FindVariableFiles(baseDir, target, varFile)...)
	return configSources, findErr
}

// checkSmartSkip determines if compilation can be skipped due to unchanged sources.
// Returns true if sources haven't changed and compilation can be skipped.
func checkSmartSkip(stateFilePath string, sourceFiles []string, baseDir string) bool {
	if applyForce {
		return false
	}

	prevManifest, readErr := state.ReadState(stateFilePath)
	if readErr != nil {
		return false
	}

	ts, exists := prevManifest.Targets[targetFlag]
	if !exists || len(ts.SourceFiles) == 0 {
		return false
	}
	prevSourceFiles := ts.SourceFiles

	changed, _ := state.SourcesChanged(prevSourceFiles, sourceFiles, baseDir)
	return !changed
}

// compileOpts holds parameters for compilation to reduce function arity.
type compileOpts struct {
	target    string
	blueprint string
	varFile   string
}

// compileAndOptimize compiles the config and applies optimization.
// Returns compiled output, fidelity notes, and any error.
func compileAndOptimize(config *ast.XcaffoldConfig, baseDir string, opts compileOpts) (*output.Output, []renderer.FidelityNote, error) {
	out, notes, err := compiler.Compile(config, baseDir, compiler.CompileOpts{
		Target:    opts.target,
		Blueprint: opts.blueprint,
		VarFile:   opts.varFile,
	})
	if err != nil {
		return nil, notes, err
	}

	// Renderers resolve @-imports natively; the optimizer handles targets that don't.
	opt := optimizer.New(opts.target)
	optimized, optNotes, optErr := opt.Run(out.Files)
	if optErr != nil {
		return nil, notes, optErr
	}
	out.Files = optimized
	notes = append(notes, optNotes...)

	// Restore same-provider extras and emit fidelity notes for cross-provider ones.
	notes = applyProviderExtras(config, out, opts.target, notes)

	return out, notes, nil
}

// checkCompilationErrors verifies fidelity notes and policy violations.
// Prints all notes/violations first, then returns an error if any are fatal.
func checkCompilationErrors(config *ast.XcaffoldConfig, out *output.Output, notes []renderer.FidelityNote) error {
	// Check for error-level fidelity notes (e.g., missing required fields).
	filteredNotes := renderer.FilterNotes(notes, buildSuppressedResourcesMap(config, targetFlag))
	printFidelityNotes(os.Stderr, filteredNotes, verboseFlag)
	if err := checkFidelityErrors(filteredNotes); err != nil {
		return &silentError{msg: err.Error()}
	}

	// Security invariants check. Run before policy evaluation.
	// Invariant violations are always fatal — no --verbose gating.
	if errs := policy.RunInvariants(config, out); len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "%s  %s\n", colorRed(glyphErr()), e)
		}
		return &silentError{msg: fmt.Sprintf("apply blocked: %d security invariant(s) violated", len(errs))}
	}

	// Policy evaluation. Run against config post-Compile() — compiler.Compile
	// has already called StripInherited() so globally-inherited resources (e.g.
	// from ~/.xcaffold/global.xcaf) are absent. Policies only evaluate what was
	// actually compiled, preventing spurious violations on user-wide globals.
	violations := policy.Evaluate(config.Policies, config, out)
	policyErrors := policy.FilterBySeverity(violations, policy.SeverityError)
	policyWarnings := policy.FilterBySeverity(violations, policy.SeverityWarning)

	if len(policyWarnings) > 0 && verboseFlag {
		fmt.Fprint(os.Stderr, policy.FormatViolations(policyWarnings))
	}
	if len(policyErrors) > 0 {
		fmt.Fprint(os.Stderr, policy.FormatViolations(policyErrors))
		return &silentError{msg: fmt.Sprintf("apply blocked: %d policy error(s) found", len(policyErrors))}
	}

	return nil
}

// showApplyPreview displays the preview table and prompts for confirmation.
// Returns true if user confirmed or if no changes detected (which means "proceed").
// Returns false only if user explicitly denied the prompt.
func showApplyPreview(oldManifest *state.StateManifest, out *output.Output, outputDir, baseDir string) (bool, error) {
	preview := computeApplyPreview(out.Files, out.RootFiles, outputDir, baseDir)
	orphanCount := countOrphansFromState(oldManifest, targetFlag, out.Files)
	newC, changedC, _ := renderApplyPreview(preview)
	totalChanges := newC + changedC + orphanCount

	if totalChanges == 0 {
		fmt.Printf("\n  %s  No changes. Current files are up to date.\n", colorGreen(glyphOK()))
		return true, nil // Return true to allow state file to be written
	}

	if applyYes {
		return true, nil
	}

	parts := []string{}
	if changedC > 0 {
		parts = append(parts, fmt.Sprintf("%d changed", changedC))
	}
	if newC > 0 {
		parts = append(parts, fmt.Sprintf("%d new", newC))
	}
	if orphanCount > 0 {
		parts = append(parts, fmt.Sprintf("%d deleted", orphanCount))
	}
	msg := fmt.Sprintf("Apply %s?", strings.Join(parts, " + "))
	ok, err := prompt.Confirm(msg, true)
	if err != nil {
		return false, fmt.Errorf("prompt error: %w", err)
	}
	return ok, nil
}

// writeApplyFiles writes all compiled files (both output and root) to disk.
// Updates filesWritten counter and returns it.
func writeApplyFiles(out *output.Output, outputDir, baseDir, scopeName string) (int, error) {
	result := &applyFileResult{}

	for relPath, content := range out.Files {
		absPath := filepath.Clean(filepath.Join(outputDir, relPath))
		if err := applyFile(absPath, content, scopeName, result); err != nil {
			return 0, err
		}
	}

	for relPath, content := range out.RootFiles {
		absPath := filepath.Clean(filepath.Join(baseDir, relPath))
		if err := applyFile(absPath, content, scopeName, result); err != nil {
			return 0, err
		}
	}

	return result.filesWritten, nil
}

// writeApplyState writes the state manifest to disk after a successful apply.
// writeState generates and persists the new state manifest.
func (ctx *applyContext) writeState() error {
	// Compute blueprint hash before writing state.
	var bpHash string
	if applyBlueprintFlag != "" {
		if p, ok := ctx.config.Blueprints[applyBlueprintFlag]; ok {
			bpHash = blueprint.BlueprintHash(p)
		}
	}

	newManifest, err := state.GenerateState(ctx.out, state.StateOpts{
		Blueprint:     applyBlueprintFlag,
		BlueprintHash: bpHash,
		Target:        targetFlag,
		BaseDir:       ctx.baseDir,
		SourceFiles:   ctx.sourceFiles,
	}, ctx.oldManifest)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Warning: failed to generate state: %v\n", err)
	}
	if err := state.WriteState(newManifest, ctx.stateFilePath); err != nil {
		fmt.Fprintf(os.Stderr, "  Warning: failed to write state: %v\n", err)
	}
	return nil
}

// checkFidelityErrors scans fidelity notes for error-level entries and returns
// an error if any are found. Error-level notes (e.g., missing required fields)
// are severe enough to block compilation.
func checkFidelityErrors(notes []renderer.FidelityNote) error {
	var errs []string
	for _, n := range notes {
		if n.Level == renderer.LevelError {
			errs = append(errs, fmt.Sprintf("%s %s/%s: %s", n.Code, n.Kind, n.Resource, n.Reason))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("compilation failed with %d error(s):\n  %s", len(errs), strings.Join(errs, "\n  "))
	}
	return nil
}

// applyDiffEntry represents a single file comparison for preview output.
type applyDiffEntry struct {
	Path   string
	Status string // "new", "changed", "unchanged"
}

// computeApplyPreview compares compiled output against files on disk.
// Only handles NEW/CHANGED/UNCHANGED. Orphan detection is handled by
// existing cleanOrphansFromState() to avoid duplicating logic.
func init() {
	applyCmd.Flags().BoolVar(&applyDryRun, "dry-run", false, "Preview changes without writing to disk")
	applyCmd.Flags().BoolVar(&applyForce, "force", false, "Overwrite customized local files and bypass drift safeguard")
	applyCmd.Flags().BoolVar(&applyBackup, "backup", false, "Backup existing target directory before overwriting")
	applyCmd.Flags().BoolVarP(&applyYes, "yes", "y", false, "Skip confirmation prompt (CI/CD mode)")
	applyCmd.Flags().StringVar(&applyProjectFlag, "project", "", "Apply to an external project registered in the global registry")
	applyCmd.Flags().StringVar(&applyBlueprintFlag, "blueprint", "", "Compile a specific blueprint (default: all resources)")
	applyCmd.Flags().StringVar(&targetFlag, "target", "", fmt.Sprintf("compilation target platform (%s)", strings.Join(providers.PrimaryNames(), ", ")))
	applyCmd.Flags().StringVar(&varFileFlag, "var-file", "", "Load variables from a custom file")
	rootCmd.AddCommand(applyCmd)
}

// currentSchemaVersion is the schema version this build of xcaffold targets.
// Configs with older versions produce an error requiring the user to update the version field.
const currentSchemaVersion = "1.0"

func runApply(cmd *cobra.Command, args []string) error {
	if applyBlueprintFlag != "" && globalFlag {
		return fmt.Errorf("--blueprint cannot be used with --global (blueprints are project-scoped)")
	}

	if applyProjectFlag != "" {
		proj, err := registry.Resolve(applyProjectFlag)
		if err != nil {
			return fmt.Errorf("project %q not found in registry: %w", applyProjectFlag, err)
		}
		globalXcafPath = filepath.Join(proj.Path, "project.xcaf")
		xcafPath = globalXcafPath
		projectRoot = proj.Path
	}

	if globalFlag {
		// globalXcafHome is ~/.xcaffold/ — the source directory.
		// Global artifacts are written one level up (~/), into ~/.claude/ etc.
		globalOutDir := filepath.Join(filepath.Dir(globalXcafHome), compiler.OutputDir(targetFlag))
		return applyScope(globalXcafPath, globalOutDir, globalXcafHome, scopeGlobal)
	}

	// projectRoot is the canonical CWD-level project directory, always set by
	// resolveProjectConfig before any subcommand runs. It is the single source
	// of truth for the project root — never derive it from filepath.Dir(xcafPath)
	// because xcafPath may live inside .xcaffold/.
	if projectRoot == "" {
		// Defensive fallback: should never happen post-resolveProjectConfig.
		return fmt.Errorf("internal error: project root not resolved; run from a directory containing project.xcaf")
	}

	// Determine which targets to compile.
	// Priority: --target flag > blueprint targets > project targets > error
	targets := resolveTargets(cmd, projectRoot, applyBlueprintFlag)
	if targets == nil {
		return fmt.Errorf("no compilation targets configured; set targets in project.xcaf or pass --target")
	}

	for _, t := range targets {
		targetFlag = t
		outDir := filepath.Join(projectRoot, compiler.OutputDir(t))
		if err := applyScope(xcafPath, outDir, projectRoot, "project"); err != nil {
			return err
		}
	}
	_ = registry.UpdateLastApplied(projectRoot)
	return nil
}

// resolveTargets returns the list of compilation targets with 4-tier priority:
// 1. --target flag (if explicitly set by the user)
// 2. blueprint targets (if blueprintName is specified and blueprint has targets)
// 3. project targets (from project.xcaf)
// 4. nil (no targets configured anywhere)
//
// baseDir must be the project root directory (not filepath.Dir(xcafPath)) —
// xcafPath may live inside .xcaffold/ and filepath.Dir would give the wrong dir.
func resolveTargets(cmd *cobra.Command, baseDir string, blueprintName string) []string {
	if cmd != nil && cmd.Flag("target") != nil && cmd.Flag("target").Changed {
		return []string{targetFlag}
	}

	config, err := parser.ParseDirectory(baseDir, parser.WithVarFile(varFileFlag))
	if err != nil {
		return nil
	}

	if blueprintName != "" {
		if bp, ok := config.Blueprints[blueprintName]; ok && len(bp.Targets) > 0 {
			return bp.Targets
		}
		return nil
	}

	if config.Project != nil && len(config.Project.Targets) > 0 {
		return config.Project.Targets
	}

	return nil
}

// applyScope compiles the xcaf configuration at configPath into outputDir.
// baseDir is the project root directory — the canonical source of truth passed
// in by the caller (runApply uses projectRoot; global apply uses globalXcafHome).
// It must never be derived from filepath.Dir(configPath) because configPath may
// live inside .xcaffold/ and filepath.Dir would give the wrong directory.
//
// scopeName is used as a prefix in terminal output so the user can distinguish
// global from project compilation.
func applyScope(configPath, outputDir, baseDir, scopeName string) error {
	projectName := filepath.Base(baseDir)
	lastApplied := findLastApplied(baseDir, applyBlueprintFlag)

	config, err := parseApplyConfig(baseDir, projectName, lastApplied, scopeName)
	if err != nil {
		return err
	}

	stateFilePath, sourceFiles := setupApplyPhase(baseDir, config, outputDir, scopeName)

	// Check for smart skip before compilation
	if checkSmartSkip(stateFilePath, sourceFiles, baseDir) {
		printSmartSkipMessage()
		return nil
	}

	out, err := compileApplyPhase(config, baseDir)
	if err != nil {
		return err
	}

	oldManifest, _ := state.ReadState(stateFilePath)
	if err := prepareApplyPhase(baseDir, outputDir, stateFilePath); err != nil {
		return err
	}

	printTargetsWarning(config)

	if applyDryRun {
		fmt.Println("  Dry-run preview:")
		fmt.Println()
	}

	promptDenied, err := getUserApplyConfirmation(oldManifest, out, outputDir, baseDir)
	if err != nil {
		return err
	}
	fmt.Println()

	if promptDenied {
		return nil
	}

	ctx := &applyContext{
		config:        config,
		configPath:    configPath,
		projectName:   projectName,
		outputDir:     outputDir,
		baseDir:       baseDir,
		stateFilePath: stateFilePath,
		sourceFiles:   sourceFiles,
		oldManifest:   oldManifest,
		out:           out,
		scopeName:     "project",
	}
	return ctx.completeApply()
}

// setupApplyPhase initializes the apply process by setting up state and source files.
func setupApplyPhase(baseDir string, config *ast.XcaffoldConfig, outputDir, scopeName string) (string, []string) {
	stateFilePath := state.StateFilePath(baseDir, applyBlueprintFlag)
	ensureGitignoreEntry(baseDir, ".xcaffold/")
	sourceFiles, _ := prepareSourceFiles(baseDir, targetFlag, varFileFlag)

	if applyBackup && !applyDryRun {
		if err := performApplyBackup(config, outputDir, scopeName); err == nil {
			return stateFilePath, sourceFiles
		}
	}

	return stateFilePath, sourceFiles
}

// compileApplyPhase runs the compilation and optimization phase.
func compileApplyPhase(config *ast.XcaffoldConfig, baseDir string) (*output.Output, error) {
	out, notes, err := compileAndOptimize(config, baseDir, compileOpts{
		target:    targetFlag,
		blueprint: applyBlueprintFlag,
		varFile:   varFileFlag,
	})
	if err != nil {
		fmt.Printf("  %s  Compilation failed: %v\n", colorRed(glyphErr()), err)
		return nil, &silentError{msg: err.Error()}
	}

	if err := checkCompilationErrors(config, out, notes); err != nil {
		return nil, err
	}

	return out, nil
}

// prepareApplyPhase prepares for the confirmation phase by checking drift and cross-scope cleanup.
func prepareApplyPhase(baseDir, outputDir, stateFilePath string) error {
	if !applyDryRun {
		if err := cleanCrossScope(crossScopeOpts{
			baseDir:          baseDir,
			outputDir:        outputDir,
			currentStatePath: stateFilePath,
			target:           targetFlag,
			force:            applyForce,
		}); err != nil {
			return err
		}
	}

	return checkApplyDrift(outputDir, stateFilePath, baseDir)
}

// parseApplyConfig parses and validates the configuration for the apply scope.
func parseApplyConfig(baseDir, projectName, lastApplied, scopeName string) (*ast.XcaffoldConfig, error) {
	config, err := parser.ParseDirectory(baseDir, parser.WithVarFile(varFileFlag))
	if err != nil {
		fmt.Println(formatHeader(headerInfo{
			project:     projectName,
			blueprint:   applyBlueprintFlag,
			isGlobal:    scopeName == scopeGlobal,
			provider:    targetFlag,
			lastApplied: lastApplied,
		}))
		fmt.Println()
		fmt.Printf("  %s  %v\n", colorRed(glyphErr()), err)
		fmt.Println()
		fmt.Printf("%s Run 'xcaffold validate' for detailed diagnostics.\n", glyphArrow())
		return nil, &silentError{msg: err.Error()}
	}

	fmt.Println(formatHeader(headerInfo{
		project:     projectName,
		blueprint:   applyBlueprintFlag,
		isGlobal:    scopeName == scopeGlobal,
		provider:    targetFlag,
		lastApplied: lastApplied,
	}))
	fmt.Println()

	if config.Version != "" && config.Version < currentSchemaVersion {
		return nil, fmt.Errorf("project.xcaf uses schema version %s but xcaffold requires %s — please update the version field in your project.xcaf", config.Version, currentSchemaVersion)
	}

	return config, nil
}

// completeApply finishes the apply operation after user confirmation.
// completeApply finalizes the apply phase: cleans orphans, writes files, updates state.
func (ctx *applyContext) completeApply() error {
	var hasChanges bool
	if !applyDryRun {
		ctx.cleanOrphans(&hasChanges)
	}

	filesWritten, err := writeApplyFiles(ctx.out, ctx.outputDir, ctx.baseDir, ctx.scopeName)
	if err != nil {
		return err
	}

	if applyDryRun {
		if filesWritten == 0 && !hasChanges {
			fmt.Printf("  %s  No changes predicted. Current files are up to date.\n", colorGreen(glyphOK()))
		}
		return nil
	}

	if err := ctx.writeState(); err != nil {
		return err
	}

	printApplySummary(filesWritten)
	updateProjectRegistry(ctx.configPath, ctx.baseDir, ctx.projectName, ctx.config)

	return nil
}

// performApplyBackup creates a backup of the output directory if configured.
func performApplyBackup(config *ast.XcaffoldConfig, outputDir, scopeName string) error {
	var backupDir string
	if config.Project != nil {
		backupDir = config.Project.BackupDir
	}
	return performBackup(outputDir, targetFlag, backupDir, scopeName)
}

// printSmartSkipMessage outputs the smart skip message.
func printSmartSkipMessage() {
	if applyDryRun {
		fmt.Printf("  %s  Sources unchanged. Nothing to compile.\n", colorGreen(glyphOK()))
	} else {
		fmt.Printf("  %s  Sources unchanged. Nothing to compile.\n", colorGreen(glyphOK()))
		fmt.Println()
		fmt.Printf("%s Run 'xcaffold apply --force' to recompile.\n", glyphArrow())
	}
}

// checkApplyDrift checks for drift if not in dry-run or force mode.
func checkApplyDrift(outputDir, stateFilePath, baseDir string) error {
	if applyDryRun || applyForce {
		return nil
	}

	driftEntries, err := hasDriftFromState(outputDir, stateFilePath, baseDir, targetFlag)
	if err == nil && len(driftEntries) > 0 {
		fmt.Fprintf(os.Stderr, "\n  %s  Drift detected in %d %s:\n\n", colorRed(glyphErr()), len(driftEntries), plural(len(driftEntries), "file", "files"))
		for _, d := range driftEntries {
			display, isRoot := formatArtifactPath(d.Path)
			label := d.Status
			if isRoot {
				display += "  (root)"
			}
			fmt.Fprintf(os.Stderr, "    %s  %-10s  %s\n", glyphErr(), label, display)
		}
		fmt.Fprintf(os.Stderr, "  To preserve manual edits, run 'xcaffold import' first.\n\n")
		fmt.Fprintf(os.Stderr, "%s Run 'xcaffold apply --force' to overwrite.\n", glyphArrow())
		return &silentError{msg: "drift detected"}
	}
	return nil
}

// printTargetsWarning outputs a warning if agents have target blocks.
func printTargetsWarning(config *ast.XcaffoldConfig) {
	for _, agent := range config.Agents {
		if len(agent.Targets) > 0 {
			fmt.Fprintf(os.Stderr, "  %s  'targets' block on agents is experimental and currently uncompiled.\n", colorYellow(glyphSrc()))
			break
		}
	}
}

// getUserApplyConfirmation prompts the user for confirmation (if needed) and returns promptDenied.
func getUserApplyConfirmation(oldManifest *state.StateManifest, out *output.Output, outputDir, baseDir string) (bool, error) {
	if applyDryRun || applyForce {
		return false, nil
	}

	ok, err := showApplyPreview(oldManifest, out, outputDir, baseDir)
	return !ok, err
}

// printApplySummary outputs the apply completion summary.
func printApplySummary(filesWritten int) {
	fmt.Println()
	outDirName := compiler.OutputDir(targetFlag)
	fmt.Printf("%s  Apply complete. %d %s written to %s/\n",
		colorGreen(glyphOK()), filesWritten, plural(filesWritten, "file", "files"), outDirName)
	fmt.Printf("  Run 'xcaffold import' to sync manual edits back to .xcaf sources.\n")
}

// updateProjectRegistry registers or updates the project in the registry.
func updateProjectRegistry(configPath, baseDir, projectName string, config *ast.XcaffoldConfig) {
	cwd, _ := os.Getwd()
	configRelDir, _ := filepath.Rel(cwd, filepath.Dir(configPath))
	if configRelDir == "" {
		configRelDir = "."
	}
	registryName := projectName
	if config.Project != nil {
		registryName = config.Project.Name
	}
	_ = registry.Register(cwd, registryName, nil, configRelDir)
	_ = registry.UpdateLastApplied(cwd)
}

// applyProviderExtras merges ProviderExtras from the config into the compiled
// output. Files whose provider key matches target are added to out.Files as-is.
// Files from other providers are skipped and a FidelityNote is appended for
// each skipped path. Provider keys and file paths within each provider are
// sorted before iteration so note order is deterministic.
func applyProviderExtras(config *ast.XcaffoldConfig, out *output.Output, target string, notes []renderer.FidelityNote) []renderer.FidelityNote {
	if len(config.ProviderExtras) == 0 {
		return notes
	}

	// Sort provider keys for deterministic output.
	providers := make([]string, 0, len(config.ProviderExtras))
	for p := range config.ProviderExtras {
		providers = append(providers, p)
	}
	sort.Strings(providers)

	for _, provider := range providers {
		files := config.ProviderExtras[provider]
		if provider == target {
			for relPath, data := range files {
				cleaned := filepath.Clean(relPath)
				if strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
					notes = append(notes, renderer.FidelityNote{
						Level:    renderer.LevelWarning,
						Target:   target,
						Kind:     "provider",
						Resource: relPath,
						Code:     "provider-extras-path-unsafe",
						Reason:   fmt.Sprintf("skipping extras path %q: path traversal detected", relPath),
					})
					continue
				}
				out.Files[cleaned] = string(data)
			}
			continue
		}
		// Cross-provider: sort paths then emit one warning note per file.
		paths := make([]string, 0, len(files))
		for relPath := range files {
			paths = append(paths, relPath)
		}
		sort.Strings(paths)
		for _, relPath := range paths {
			notes = append(notes, renderer.FidelityNote{
				Level:    renderer.LevelWarning,
				Target:   target,
				Kind:     "provider",
				Resource: relPath,
				Code:     "provider-extras-skipped",
				Reason:   fmt.Sprintf("provider-specific artifact from %q not applicable to target %q", provider, target),
			})
		}
	}
	return notes
}

// colorDiff prints a unified diff with basic ANSI terminal colors.
