package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/blueprint"
	"github.com/saero-ai/xcaffold/internal/compiler"
	"github.com/saero-ai/xcaffold/internal/optimizer"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/policy"
	"github.com/saero-ai/xcaffold/internal/registry"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/resolver"
	"github.com/saero-ai/xcaffold/internal/state"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var applyDryRun bool
var applyForce bool
var applyBackup bool
var applyProjectFlag string
var applyBlueprintFlag string
var targetFlag string

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Compile .xcf resources into provider-native agent files",
	Long: `Deterministically compiles .xcf resources into provider-native agent files.

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

func init() {
	applyCmd.Flags().BoolVar(&applyDryRun, "dry-run", false, "Preview changes without writing to disk")
	applyCmd.Flags().BoolVar(&applyForce, "force", false, "Overwrite customized local files and bypass drift safeguard")
	applyCmd.Flags().BoolVar(&applyBackup, "backup", false, "Backup existing target directory before overwriting")
	applyCmd.Flags().StringVar(&applyProjectFlag, "project", "", "Apply to an external project registered in the global registry")
	applyCmd.Flags().StringVar(&applyBlueprintFlag, "blueprint", "", "Compile a specific blueprint (default: all resources)")
	applyCmd.Flags().StringVar(&targetFlag, "target", targetClaude, "compilation target platform (claude, cursor, antigravity, copilot, gemini; default: claude)")
	rootCmd.AddCommand(applyCmd)
}

const (
	targetClaude      = "claude"
	targetAntigravity = "antigravity"
	targetCursor      = "cursor"
	targetCopilot     = "copilot"
	targetGemini      = "gemini"
)

// currentSchemaVersion is the schema version this build of xcaffold targets.
// Configs with older versions produce an error requiring the user to update the version field.
const currentSchemaVersion = "1.0"

//nolint:gocyclo
func runApply(cmd *cobra.Command, args []string) error {
	if applyBlueprintFlag != "" && globalFlag {
		return fmt.Errorf("--blueprint cannot be used with --global (blueprints are project-scoped)")
	}

	if applyProjectFlag != "" {
		proj, err := registry.Resolve(applyProjectFlag)
		if err != nil {
			return fmt.Errorf("project %q not found in registry: %w", applyProjectFlag, err)
		}
		globalXcfPath = filepath.Join(proj.Path, "project.xcf")
		xcfPath = globalXcfPath
		projectRoot = proj.Path
	}

	if globalFlag {
		// globalXcfHome is ~/.xcaffold/ — the source directory.
		// Global artifacts are written one level up (~/), into ~/.claude/ etc.
		globalOutDir := filepath.Join(filepath.Dir(globalXcfHome), compiler.OutputDir(targetFlag))
		return applyScope(globalXcfPath, globalOutDir, globalXcfHome, "global")
	}

	// projectRoot is the canonical CWD-level project directory, always set by
	// resolveProjectConfig before any subcommand runs. It is the single source
	// of truth for the project root — never derive it from filepath.Dir(xcfPath)
	// because xcfPath may live inside .xcaffold/.
	if projectRoot == "" {
		// Defensive fallback: should never happen post-resolveProjectConfig.
		return fmt.Errorf("internal error: project root not resolved; run from a directory containing a project.xcf or .xcaffold/project.xcf")
	}

	// Determine which targets to compile.
	// When --target is explicitly set by the user, honour it exclusively.
	// Otherwise, read the declared targets from the project config and compile
	// for each one. Fall back to "claude" for configs that predate targets:.
	targets := resolveTargets(cmd, projectRoot)

	for _, t := range targets {
		targetFlag = t
		outDir := filepath.Join(projectRoot, compiler.OutputDir(t))
		if err := applyScope(xcfPath, outDir, projectRoot, "project"); err != nil {
			return err
		}
	}
	_ = registry.UpdateLastApplied(projectRoot)
	return nil
}

// resolveTargets returns the list of compilation targets for a project apply.
// When cmd is non-nil and --target was explicitly changed by the user, that
// single value is returned. Otherwise the declared targets list from the
// project config is used, falling back to ["claude"] when no targets are
// declared.
//
// baseDir must be the project root directory (not filepath.Dir(xcfPath)) —
// xcfPath may live inside .xcaffold/ and filepath.Dir would give the wrong dir.
func resolveTargets(cmd *cobra.Command, baseDir string) []string {
	if cmd != nil && cmd.Flag("target") != nil && cmd.Flag("target").Changed {
		return []string{targetFlag}
	}

	config, err := parser.ParseDirectory(baseDir)
	if err == nil && config.Project != nil && len(config.Project.Targets) > 0 {
		return config.Project.Targets
	}

	return []string{targetClaude}
}

// applyScope compiles the xcf configuration at configPath into outputDir.
// baseDir is the project root directory — the canonical source of truth passed
// in by the caller (runApply uses projectRoot; global apply uses globalXcfHome).
// It must never be derived from filepath.Dir(configPath) because configPath may
// live inside .xcaffold/ and filepath.Dir would give the wrong directory.
//
// scopeName is used as a prefix in terminal output so the user can distinguish
// global from project compilation.
//
//nolint:gocyclo
func applyScope(configPath, outputDir, baseDir, scopeName string) error {
	projectName := filepath.Base(baseDir)
	lastApplied := findLastApplied(baseDir, applyBlueprintFlag)

	config, err := parser.ParseDirectory(baseDir)
	if err != nil {
		fmt.Println(formatHeader(projectName, applyBlueprintFlag, scopeName == "global", targetFlag, lastApplied))
		fmt.Println()
		fmt.Printf("  %s  %v\n", colorRed(glyphErr()), err)
		fmt.Println()
		fmt.Printf("%s Run 'xcaffold validate' for detailed diagnostics.\n", glyphArrow())
		return &silentError{msg: err.Error()}
	}

	fmt.Println(formatHeader(projectName, applyBlueprintFlag, scopeName == "global", targetFlag, lastApplied))
	fmt.Println()

	if config.Version != "" && config.Version < currentSchemaVersion {
		return fmt.Errorf("project.xcf uses schema version %s but xcaffold requires %s — please update the version field in your project.xcf", config.Version, currentSchemaVersion)
	}

	// --- Smart compilation skip: compare source hashes ---
	stateFilePath := state.StateFilePath(baseDir, applyBlueprintFlag)

	ensureGitignoreEntry(baseDir, ".xcaffold/")

	sourceFiles, findErr := resolver.FindXCFFiles(baseDir)
	if findErr != nil {
		fmt.Fprintf(os.Stderr, "  Warning: failed to scan source files: %v\n", findErr)
	}

	// Filter out non-config XCF files (e.g. kind: registry) to prevent
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
	sourceFiles = configSources

	if applyBackup && !applyDryRun {
		var backupDir string
		if config.Project != nil {
			backupDir = config.Project.BackupDir
		}
		if err := performBackup(outputDir, targetFlag, backupDir, scopeName); err != nil {
			return fmt.Errorf("backup failed: %w", err)
		}
	}

	if !applyForce {
		prevManifest, readErr := state.ReadState(stateFilePath)
		if readErr == nil && len(prevManifest.SourceFiles) > 0 {
			changed, _ := state.SourcesChanged(prevManifest.SourceFiles, sourceFiles, baseDir)
			if !changed {
				if applyDryRun {
					fmt.Printf("  %s  Sources unchanged. Nothing to compile.\n", colorGreen(glyphOK()))
				} else {
					fmt.Printf("  %s  Sources unchanged. Nothing to compile.\n", colorGreen(glyphOK()))
					fmt.Println()
					fmt.Printf("%s Run 'xcaffold apply --force' to recompile.\n", glyphArrow())
				}
				return nil
			}
		}
	}
	// --- End smart skip ---

	// compiler.Compile mutates config in-place — it calls StripInherited() to
	// remove globally-inherited resources before rendering. After Compile returns,
	// config reflects exactly what was compiled (no global-scope bleed-through).
	out, notes, err := compiler.Compile(config, baseDir, targetFlag, applyBlueprintFlag)
	if err != nil {
		fmt.Printf("  %s  Compilation failed: %v\n", colorRed(glyphErr()), err)
		return &silentError{msg: err.Error()}
	}

	// Renderers resolve @-imports natively; the optimizer handles targets that don't.
	opt := optimizer.New(targetFlag)
	optimized, optNotes, optErr := opt.Run(out.Files)
	if optErr != nil {
		return fmt.Errorf("optimizer error: %w", optErr)
	}
	out.Files = optimized
	notes = append(notes, optNotes...)

	// Restore same-provider extras and emit fidelity notes for cross-provider ones.
	notes = applyProviderExtras(config, out, targetFlag, notes)

	// Check for error-level fidelity notes (e.g., missing required fields).
	// Print all notes first (so the user sees them), then fail if any are errors.
	filteredNotes := renderer.FilterNotes(notes, buildSuppressedResourcesMap(config, targetFlag))
	printFidelityNotes(os.Stderr, filteredNotes, false)
	if err := checkFidelityErrors(filteredNotes); err != nil {
		return &silentError{msg: err.Error()}
	}

	// Policy evaluation. Run against config post-Compile() — compiler.Compile
	// has already called StripInherited() so globally-inherited resources (e.g.
	// from ~/.xcaffold/global.xcf) are absent. Policies only evaluate what was
	// actually compiled, preventing spurious violations on user-wide globals.
	violations := policy.Evaluate(config.Policies, config, out)
	policyErrors := policy.FilterBySeverity(violations, policy.SeverityError)
	policyWarnings := policy.FilterBySeverity(violations, policy.SeverityWarning)

	if len(policyWarnings) > 0 {
		fmt.Fprint(os.Stderr, policy.FormatViolations(policyWarnings))
	}
	if len(policyErrors) > 0 {
		fmt.Fprint(os.Stderr, policy.FormatViolations(policyErrors))
		return &silentError{msg: fmt.Sprintf("apply blocked: %d policy error(s) found", len(policyErrors))}
	}

	oldManifest, _ := state.ReadState(stateFilePath)

	if !applyDryRun && !applyForce {
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
	}

	for _, agent := range config.Agents {
		if len(agent.Targets) > 0 {
			fmt.Fprintf(os.Stderr, "  %s  'targets' block on agents is experimental and currently uncompiled.\n", colorYellow(glyphSrc()))
			break
		}
	}

	if applyDryRun {
		fmt.Println("  Dry-run preview:")
		fmt.Println()
	}

	// Write (or preview) each compiled file.
	hasChanges := false
	filesWritten := 0

	cleanOrphansFromState(oldManifest, targetFlag, out, outputDir, baseDir, scopeName, &hasChanges)

	for relPath, content := range out.Files {
		absPath := filepath.Clean(filepath.Join(outputDir, relPath))
		if err := applyFile(absPath, content, scopeName, &hasChanges, &filesWritten); err != nil {
			return err
		}
	}

	for relPath, content := range out.RootFiles {
		absPath := filepath.Clean(filepath.Join(baseDir, relPath))
		if err := applyFile(absPath, content, scopeName, &hasChanges, &filesWritten); err != nil {
			return err
		}
	}

	if applyDryRun {
		if !hasChanges {
			fmt.Printf("  %s  No changes predicted. Current files are up to date.\n", colorGreen(glyphOK()))
		}
		return nil
	}

	// Compute blueprint hash before writing state.
	var bpHash string
	if applyBlueprintFlag != "" {
		if p, ok := config.Blueprints[applyBlueprintFlag]; ok {
			bpHash = blueprint.BlueprintHash(p)
		}
	}

	// Write the state file with source tracking.
	newManifest, err := state.GenerateState(out, state.StateOpts{
		Blueprint:     applyBlueprintFlag,
		BlueprintHash: bpHash,
		Target:        targetFlag,
		BaseDir:       baseDir,
		SourceFiles:   sourceFiles,
	}, oldManifest)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Warning: failed to generate state: %v\n", err)
	}
	if err := state.WriteState(newManifest, stateFilePath); err != nil {
		fmt.Fprintf(os.Stderr, "  Warning: failed to write state: %v\n", err)
	}

	fmt.Println()
	outDirName := compiler.OutputDir(targetFlag)
	fmt.Printf("%s  Apply complete. %d %s written to %s/\n",
		colorGreen(glyphOK()), filesWritten, plural(filesWritten, "file", "files"), outDirName)
	fmt.Printf("  Run 'xcaffold import' to sync manual edits back to .xcf sources.\n")

	// Ensure the project is registered and the timestamp is updated.
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

	return nil
}

// applyProviderExtras merges ProviderExtras from the config into the compiled
// output. Files whose provider key matches target are added to out.Files as-is.
// Files from other providers are skipped and a FidelityNote is appended for
// each skipped path. Provider keys and file paths within each provider are
// sorted before iteration so note order is deterministic.
func applyProviderExtras(config *ast.XcaffoldConfig, out *compiler.Output, target string, notes []renderer.FidelityNote) []renderer.FidelityNote {
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
func colorDiff(diff string) {
	lines := strings.Split(diff, "\n")
	for _, l := range lines {
		switch {
		case strings.HasPrefix(l, "+"):
			fmt.Println("\033[32m" + l + "\033[0m")
		case strings.HasPrefix(l, "-"):
			fmt.Println("\033[31m" + l + "\033[0m")
		case strings.HasPrefix(l, "@"):
			fmt.Println("\033[36m" + l + "\033[0m")
		default:
			fmt.Println(l)
		}
	}
}

func previewDiff(absPath, content string) bool {
	existingData, err := os.ReadFile(absPath)
	existing := ""
	if err == nil {
		existing = string(existingData)
	}
	diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(existing),
		B:        difflib.SplitLines(content),
		FromFile: absPath + " (current)",
		ToFile:   absPath + " (compiled)",
		Context:  3,
	})
	if diff != "" {
		colorDiff(diff)
		return true
	}
	return false
}

func applyFile(absPath, content, scopeName string, hasChanges *bool, filesWritten *int) error {
	if applyDryRun {
		if previewDiff(absPath, content) {
			*hasChanges = true
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %q: %w", absPath, err)
	}
	if err := os.WriteFile(absPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write %q: %w", absPath, err)
	}
	*filesWritten++
	return nil
}

func performBackup(outputDir, target, backupDirConfig, scopeName string) error {
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		return nil // nothing to backup
	}

	timestamp := time.Now().Format("20060102_150405")
	bakName := fmt.Sprintf(".%s_bak_%s", target, timestamp)
	if target == "" {
		bakName = fmt.Sprintf(".%s_bak_%s", targetClaude, timestamp)
	}

	var destDir string
	if backupDirConfig != "" {
		destDir = filepath.Join(backupDirConfig, bakName)
	} else {
		destDir = filepath.Join(filepath.Dir(outputDir), bakName)
	}

	if err := os.MkdirAll(filepath.Dir(destDir), 0755); err != nil {
		return err
	}

	fmt.Printf("  %s  Backed up %s\n", colorGreen(glyphOK()), filepath.Base(destDir))
	return copyDir(outputDir, destDir)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destPath, data, info.Mode())
	})
}

// cleanOrphansFromState removes files from outputDir that were recorded in old
// for the given target but are absent from the new compiler output.
func cleanOrphansFromState(oldManifest *state.StateManifest, target string, out *compiler.Output, outputDir, baseDir, scopeName string, hasChanges *bool) {
	orphans := state.FindOrphansFromState(oldManifest, target, out.Files, out.RootFiles)
	for _, orphanPath := range orphans {
		var absPath string
		if strings.HasPrefix(orphanPath, "root:") {
			relPath := strings.TrimPrefix(orphanPath, "root:")
			absPath = filepath.Clean(filepath.Join(baseDir, relPath))
		} else {
			absPath = filepath.Clean(filepath.Join(outputDir, orphanPath))
		}
		if applyDryRun {
			fmt.Printf("    %s  would delete  %s\n", colorYellow(glyphSrc()), filepath.Base(absPath))
			*hasChanges = true
		} else {
			if err := os.Remove(absPath); err == nil {
				fmt.Printf("    %s  deleted  %s\n", colorRed(glyphErr()), filepath.Base(absPath))
				*hasChanges = true
				cleanEmptyDirsUpToTarget(filepath.Dir(absPath), outputDir)
			} else if os.IsNotExist(err) {
				*hasChanges = true
			}
		}
	}
}

func hasDriftFromState(outputDir, stateFile, baseDir, target string) ([]state.DriftEntry, error) {
	manifest, err := state.ReadState(stateFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	ts, ok := manifest.Targets[target]
	if !ok {
		return nil, nil
	}

	return state.CollectDriftedFiles(baseDir, outputDir, ts), nil
}

// ensureGitignoreEntry appends entry to dir/.gitignore if not already present.
// Creates the file if it does not exist.
func ensureGitignoreEntry(dir, entry string) {
	gitignorePath := filepath.Join(dir, ".gitignore")
	data, _ := os.ReadFile(gitignorePath)
	if strings.Contains(string(data), entry) {
		return
	}
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	if len(data) > 0 && data[len(data)-1] != '\n' {
		f.WriteString("\n")
	}
	f.WriteString(entry + "\n")
}

// cleanEmptyDirsUpToTarget recursively deletes empty parent directories
// up to but not including the targetDir itself.
func cleanEmptyDirsUpToTarget(dir, targetDir string) {
	dir = filepath.Clean(dir)
	targetDir = filepath.Clean(targetDir)

	for dir != targetDir && dir != "." && dir != "/" {
		rel, err := filepath.Rel(targetDir, dir)
		if err != nil || strings.HasPrefix(rel, "..") || rel == "." {
			break
		}

		if err := os.Remove(dir); err != nil {
			break // Dir not empty, or permission error
		}
		dir = filepath.Dir(dir)
	}
}
