package main

import (
	"crypto/sha256"
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
	"github.com/saero-ai/xcaffold/internal/renderer/antigravity"
	"github.com/saero-ai/xcaffold/internal/renderer/claude"
	"github.com/saero-ai/xcaffold/internal/renderer/copilot"
	"github.com/saero-ai/xcaffold/internal/renderer/cursor"
	"github.com/saero-ai/xcaffold/internal/renderer/gemini"
	"github.com/saero-ai/xcaffold/internal/resolver"
	"github.com/saero-ai/xcaffold/internal/state"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var applyDryRun bool
var applyCheckOnly bool
var applyCheckPermissions bool
var applyForce bool
var applyBackup bool
var applyIncludeMemory bool
var applyReseed bool
var applyProjectFlag string
var applyBlueprintFlag string
var targetFlag string

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Compile project.xcf into .claude/ agent files",
	Long: `xcaffold apply deterministically compiles your YAML logic into native target outputs.

┌───────────────────────────────────────────────────────────────────┐
│                          COMPILATION PHASE                        │
└───────────────────────────────────────────────────────────────────┘
 [project.xcf] ──(Compiles)──▶ [.claude/agents/*.md]
       │
   (State)──▶ [.xcaffold/project.xcf.state]

 • Strict one-way generation (YAML -> MD)
 • Generates a cryptographic SHA-256 state manifest (.xcaffold/)
 • Automatically purges orphaned target files

Any manually edited files inside the target directory will be overwritten.

Validation:
 Use the --check flag to validate your YAML syntax without compiling.`,
	Example: `  $ xcaffold apply
  $ xcaffold apply --check
  $ xcaffold apply --global
  $ xcaffold apply --dry-run    (replaces the former 'plan' command)`,
	RunE: runApply,
}

func init() {
	applyCmd.Flags().BoolVar(&applyDryRun, "dry-run", false, "Preview changes without writing to disk")
	applyCmd.Flags().BoolVar(&applyCheckOnly, "check", false, "Check configuration syntax without compiling")
	applyCmd.Flags().BoolVar(&applyCheckPermissions, "check-permissions", false, "Report security field drops and permission contradictions, then exit")
	applyCmd.Flags().BoolVar(&applyForce, "force", false, "Overwrite customized local files and bypass drift safeguard")
	applyCmd.Flags().BoolVar(&applyBackup, "backup", false, "Backup existing target directory before overwriting")
	applyCmd.Flags().BoolVar(&applyIncludeMemory, "include-memory", false, "Seed memory entries to the target provider as part of this apply")
	applyCmd.Flags().BoolVar(&applyReseed, "reseed", false, "Overwrite existing memory files (bypass seed-once guard and drift check); implies --include-memory")
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

// memoryPassEnabled reports whether the memory rendering pass should run for
// this apply invocation. --reseed implies --include-memory.
func memoryPassEnabled(includeMemory, reseed bool) bool {
	return includeMemory || reseed
}

// currentSchemaVersion is the schema version this build of xcaffold targets.
// Configs with older versions produce a warning prompting the user to migrate.
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

	if applyCheckOnly {
		if globalFlag {
			if _, err := parser.ParseDirectory(globalXcfHome); err != nil {
				return fmt.Errorf("[global] parse error: %w", err)
			}
			fmt.Println("[global] ✓ Syntax is valid")
			diags := parser.ValidateFile(globalXcfPath)
			printDiagnostics(diags)
			for _, d := range diags {
				if d.Severity == "error" {
					return fmt.Errorf("[global] validation failed with errors")
				}
			}
		} else {
			if _, err := parser.ParseDirectory(filepath.Dir(xcfPath)); err != nil {
				return fmt.Errorf("[project] parse error: %w", err)
			}
			fmt.Println("[project] ✓ Syntax is valid")
			diags := parser.ValidateFile(xcfPath)
			printDiagnostics(diags)
			for _, d := range diags {
				if d.Severity == "error" {
					return fmt.Errorf("[project] validation failed with errors")
				}
			}
		}
		return nil
	}

	if applyCheckPermissions {
		// Parse runs validatePermissions — any contradiction surfaces as a parse
		// error before we reach this block. The structured report only shows target
		// fidelity findings for configs that already pass parsing.
		var parseDir string
		if globalFlag {
			parseDir = globalXcfHome
		} else {
			parseDir = filepath.Dir(xcfPath)
		}
		config, err := parser.ParseDirectory(parseDir)
		if err != nil {
			return fmt.Errorf("parse error: %w", err)
		}

		secRenderer, err := rendererForTarget(targetFlag)
		if err != nil {
			return fmt.Errorf("unknown target for security check: %w", err)
		}
		errors, warnings := securityFieldReport(config, secRenderer)

		for _, w := range warnings {
			fmt.Printf("[WARNING] %s\n", w)
		}
		for _, e := range errors {
			fmt.Printf("[ERROR]   %s\n", e)
		}

		if len(errors) == 0 && len(warnings) == 0 {
			fmt.Printf("[INFO]    %s: all security fields are supported\n", targetFlag)
		}

		if len(errors) > 0 {
			return fmt.Errorf("check-permissions: %d error(s) found", len(errors))
		}
		return nil
	}

	if globalFlag {
		return applyScope(globalXcfPath, globalXcfHome, "global")
	}

	// Determine which targets to compile.
	// When --target is explicitly set by the user, honour it exclusively.
	// Otherwise, read the declared targets from the project config and compile
	// for each one. Fall back to "claude" for configs that predate targets:.
	targets := resolveTargets(cmd, xcfPath)

	baseDir := filepath.Dir(xcfPath)
	for _, t := range targets {
		targetFlag = t
		outDir := filepath.Join(baseDir, compiler.OutputDir(t))
		if err := applyScope(xcfPath, outDir, "project"); err != nil {
			return err
		}
	}
	_ = registry.UpdateLastApplied(baseDir)
	return nil
}

// resolveTargets returns the list of compilation targets for a project apply.
// When cmd is non-nil and --target was explicitly changed by the user, that
// single value is returned. Otherwise the declared targets list from the
// project config is used, falling back to ["claude"] when no targets are
// declared.
func resolveTargets(cmd *cobra.Command, xcfFilePath string) []string {
	if cmd != nil && cmd.Flag("target") != nil && cmd.Flag("target").Changed {
		return []string{targetFlag}
	}

	baseDir := filepath.Dir(xcfFilePath)
	config, err := parser.ParseDirectory(baseDir)
	if err == nil && config.Project != nil && len(config.Project.Targets) > 0 {
		return config.Project.Targets
	}

	return []string{targetClaude}
}

// printDiagnostics prints ValidateFile diagnostics to stderr. Warnings do not
// change the exit code; this helper is informational only.
func printDiagnostics(diags []parser.Diagnostic) {
	if len(diags) == 0 {
		return
	}
	for _, d := range diags {
		fmt.Fprintf(os.Stderr, "  [%s] %s\n", d.Severity, d.Message)
	}
}

// applyScope compiles a single xcf file into outputDir. scopeName is used as a
// prefix in terminal output so the user can distinguish global from project
// compilation.
//
//nolint:gocyclo
func applyScope(configPath, outputDir, scopeName string) error {
	// baseDir is the directory containing the xcf file — used by the compiler
	// to resolve instructions-file: and references: paths.
	baseDir := filepath.Dir(configPath)

	config, err := parser.ParseDirectory(baseDir)
	if err != nil {
		return fmt.Errorf("[%s] parse error: %w", scopeName, err)
	}

	if config.Version != "" && config.Version < currentSchemaVersion {
		fmt.Fprintf(os.Stderr, "WARNING: project.xcf uses schema version %s; current schema is %s. Run \"xcaffold migrate\" to upgrade.\n", config.Version, currentSchemaVersion)
	}

	// --- Smart compilation skip: compare source hashes ---
	stateFilePath := state.StateFilePath(baseDir, applyBlueprintFlag)

	ensureGitignoreEntry(baseDir, ".xcaffold/")

	sourceFiles, findErr := resolver.FindXCFFiles(baseDir)
	if findErr != nil {
		fmt.Fprintf(os.Stderr, "[%s] Warning: failed to scan source files: %v\n", scopeName, findErr)
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

	if !applyForce {
		prevManifest, readErr := state.ReadState(stateFilePath)
		if readErr == nil && len(prevManifest.SourceFiles) > 0 {
			changed, _ := state.SourcesChanged(prevManifest.SourceFiles, sourceFiles, baseDir)
			if !changed {
				if applyDryRun {
					fmt.Printf("[%s] No source files changed. Nothing to compile.\n", scopeName)
				} else {
					fmt.Printf("[%s] Sources unchanged — skipping compilation. Use --force to recompile.\n", scopeName)
				}
				return nil
			}
		}
	}
	// --- End smart skip ---

	configSnapshot := deepCopyConfig(config)

	out, notes, err := compiler.Compile(config, baseDir, targetFlag, applyBlueprintFlag)
	if err != nil {
		return fmt.Errorf("[%s] compilation error: %w", scopeName, err)
	}

	// Renderers resolve @-imports natively; the optimizer handles targets that don't.
	opt := optimizer.New(targetFlag)
	optimized, optNotes, optErr := opt.Run(out.Files)
	if optErr != nil {
		return fmt.Errorf("[%s] optimizer error: %w", scopeName, optErr)
	}
	out.Files = optimized
	notes = append(notes, optNotes...)

	// Restore same-provider extras and emit fidelity notes for cross-provider ones.
	notes = applyProviderExtras(config, out, targetFlag, notes)

	printFidelityNotes(os.Stderr, renderer.FilterNotes(notes, buildSuppressedResourcesMap(config, targetFlag)), false)

	// Policy evaluation
	violations := policy.Evaluate(configSnapshot.Policies, configSnapshot, out)
	policyErrors := policy.FilterBySeverity(violations, policy.SeverityError)
	policyWarnings := policy.FilterBySeverity(violations, policy.SeverityWarning)

	if len(policyWarnings) > 0 {
		fmt.Fprint(os.Stderr, policy.FormatViolations(policyWarnings))
	}
	if len(policyErrors) > 0 {
		fmt.Fprint(os.Stderr, policy.FormatViolations(policyErrors))
		return fmt.Errorf("[%s] apply blocked: %d policy error(s) found", scopeName, len(policyErrors))
	}

	// Resolve the target-specific output directory instead of the hardcoded default
	outputDir = filepath.Join(filepath.Dir(outputDir), compiler.OutputDir(targetFlag))

	oldManifest, _ := state.ReadState(stateFilePath)

	if !applyDryRun && !applyForce {
		drift, err := hasDriftFromState(outputDir, stateFilePath, targetFlag)
		if err == nil && drift {
			return fmt.Errorf("[%s] drift detected! Target directory contains unrecorded changes. Use --force to overwrite", scopeName)
		}
	}

	if applyBackup && !applyDryRun {
		var backupDir string
		if config.Project != nil {
			backupDir = config.Project.BackupDir
		}
		if err := performBackup(outputDir, targetFlag, backupDir, scopeName); err != nil {
			return fmt.Errorf("[%s] backup failed: %w", scopeName, err)
		}
	}

	for _, agent := range config.Agents {
		if len(agent.Targets) > 0 {
			fmt.Fprintf(os.Stderr, "[%s] Warning: 'targets' block is experimental and currently uncompiled.\n", scopeName)
			break
		}
	}

	if applyDryRun {
		fmt.Printf("[%s] Dry-run preview (no files will be written):\n\n", scopeName)
	}

	// Write (or preview) each compiled file.
	hasChanges := false

	cleanOrphansFromState(oldManifest, targetFlag, out, outputDir, scopeName, &hasChanges)

	for relPath, content := range out.Files {
		absPath := filepath.Clean(filepath.Join(outputDir, relPath))
		if err := applyFile(absPath, content, scopeName, &hasChanges); err != nil {
			return err
		}
	}

	if applyDryRun {
		if !hasChanges {
			fmt.Printf("[%s] ✓ No changes predicted. Current files are up to date.\n", scopeName)
		}
		// Log memory dry-run intent even though we are exiting early.
		// priorSeeds from oldManifest are not yet available here; pass nil —
		// this only affects drift-reporting precision, not the intent message.
		if memoryPassEnabled(applyIncludeMemory, applyReseed) {
			if memR, err := rendererForTarget(targetFlag); err == nil {
				_, _, _ = runMemoryPass(config, memR, baseDir, outputDir, nil, true, applyReseed)
			}
		}
		return nil
	}

	// Memory rendering pass. Runs per-target when --include-memory or --reseed
	// is set. Collected seeds are recorded in the state file.
	var memSeeds []state.MemorySeed
	if memoryPassEnabled(applyIncludeMemory, applyReseed) {
		var priorSeeds []state.MemorySeed
		if oldManifest != nil {
			priorSeeds = oldManifest.MemorySeeds
		}
		memR, err := rendererForTarget(targetFlag)
		if err != nil {
			return fmt.Errorf("memory pass: %w", err)
		}
		seeds, memNotes, memErr := runMemoryPass(config, memR, baseDir, outputDir, priorSeeds, applyDryRun, applyReseed)
		if len(memNotes) > 0 {
			printFidelityNotes(os.Stderr, memNotes, false)
		}
		if memErr != nil {
			// Memory drift or other soft errors do not halt apply in v1 — the
			// primary compile has already succeeded. Surface the message and
			// skip seed recording for this target.
			fmt.Fprintf(os.Stderr, "[%s] memory: %v\n", scopeName, memErr)
		} else {
			memSeeds = seeds
		}
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
		MemorySeeds:   memSeeds,
	}, oldManifest)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[%s] Warning: failed to generate state: %v\n", scopeName, err)
	}
	if err := state.WriteState(newManifest, stateFilePath); err != nil {
		fmt.Fprintf(os.Stderr, "[%s] Warning: failed to write state: %v\n", scopeName, err)
	}

	fmt.Printf("\n[%s] ✓ Apply complete. State updated.\n", scopeName)

	// Ensure the project is registered and the timestamp is updated.
	cwd, _ := os.Getwd()
	configRelDir, _ := filepath.Rel(cwd, filepath.Dir(configPath))
	if configRelDir == "" {
		configRelDir = "."
	}
	var projectName string
	if config.Project != nil {
		projectName = config.Project.Name
	}
	_ = registry.Register(cwd, projectName, nil, configRelDir)
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
						Kind:     "extras",
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
				Kind:     "extras",
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

func applyFile(absPath, content, scopeName string, hasChanges *bool) error {
	if applyDryRun {
		if previewDiff(absPath, content) {
			*hasChanges = true
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return fmt.Errorf("[%s] failed to create directory for %q: %w", scopeName, absPath, err)
	}
	if err := os.WriteFile(absPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("[%s] failed to write %q: %w", scopeName, absPath, err)
	}
	hash := sha256.Sum256([]byte(content))
	fmt.Printf("  [%s] ✓ wrote %s  (sha256:%x)\n", scopeName, absPath, hash)
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

	fmt.Printf("[%s] Backing up %s -> %s\n", scopeName, outputDir, destDir)
	return copyDir(outputDir, destDir)
}

// securityFieldReport returns [ERROR] and [WARNING] findings for the given
// renderer by inspecting which security fields in the config would be dropped.
// It is read-only and never modifies any files.
func securityFieldReport(config *ast.XcaffoldConfig, r renderer.TargetRenderer) (errorsOut, warnings []string) {
	caps := r.Capabilities()
	sf := caps.SecurityFields
	target := r.Target()

	// Get the active settings (first available key after blueprint filtering).
	var es ast.SettingsConfig
	for _, s := range config.Settings {
		es = s
		break
	}

	// If all security fields are supported, no findings.
	if sf.Permissions && sf.Sandbox && sf.PermissionMode && sf.DisallowedTools && sf.Isolation && sf.Effort {
		return nil, nil
	}

	if !sf.Permissions && es.Permissions != nil {
		warnings = append(warnings, fmt.Sprintf("%s: settings.permissions will be dropped — no enforcement equivalent", target))
	}
	if !sf.Sandbox && es.Sandbox != nil {
		warnings = append(warnings, fmt.Sprintf("%s: settings.sandbox will be dropped — no sandbox model", target))
	}

	agentIDs := make([]string, 0, len(config.Agents))
	for id := range config.Agents {
		agentIDs = append(agentIDs, id)
	}
	sort.Strings(agentIDs)

	for _, id := range agentIDs {
		agent := config.Agents[id]
		if !sf.Effort && agent.Effort != "" {
			warnings = append(warnings, fmt.Sprintf("%s: agent %q effort %q will be dropped", target, id, agent.Effort))
		}
		if !sf.PermissionMode && agent.PermissionMode != "" {
			warnings = append(warnings, fmt.Sprintf("%s: agent %q permission-mode %q will be dropped", target, id, agent.PermissionMode))
		}
		if !sf.DisallowedTools && len(agent.DisallowedTools) > 0 {
			warnings = append(warnings, fmt.Sprintf("%s: agent %q disallowed-tools will be dropped — tool restrictions will NOT be enforced", target, id))
		}
		if !sf.Isolation && agent.Isolation != "" {
			warnings = append(warnings, fmt.Sprintf("%s: agent %q isolation %q will be dropped", target, id, agent.Isolation))
		}
	}

	// Agent vs deny conflicts
	if es.Permissions != nil {
		for _, id := range agentIDs {
			agent := config.Agents[id]
			for _, tool := range agent.Tools {
				for _, denyRule := range es.Permissions.Deny {
					if denyRule == tool {
						errorsOut = append(errorsOut, fmt.Sprintf("permissions.deny: rule %q conflicts with agent %q tools list", tool, id))
					}
				}
			}
		}
	}

	return errorsOut, warnings
}

func rendererForTarget(target string) (renderer.TargetRenderer, error) {
	switch target {
	case targetClaude:
		return claude.New(), nil
	case targetCursor:
		return cursor.New(), nil
	case targetGemini:
		return gemini.New(), nil
	case targetCopilot:
		return copilot.New(), nil
	case targetAntigravity:
		return antigravity.New(), nil
	default:
		return nil, fmt.Errorf("unknown target %q", target)
	}
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
func cleanOrphansFromState(oldManifest *state.StateManifest, target string, out *compiler.Output, outputDir, scopeName string, hasChanges *bool) {
	orphans := state.FindOrphansFromState(oldManifest, target, out.Files)
	for _, orphanPath := range orphans {
		absPath := filepath.Clean(filepath.Join(outputDir, orphanPath))
		if applyDryRun {
			fmt.Printf("  [%s] \033[31m[- DELETE]\033[0m %s\n", scopeName, absPath)
			*hasChanges = true
		} else {
			if err := os.Remove(absPath); err == nil {
				fmt.Printf("  [%s] ✓ deleted %s\n", scopeName, absPath)
				*hasChanges = true
				cleanEmptyDirsUpToTarget(filepath.Dir(absPath), outputDir)
			} else if os.IsNotExist(err) {
				*hasChanges = true
			}
		}
	}
}

// hasDriftFromState checks whether any artifact recorded for target in the
// StateManifest at stateFile has been modified on disk since the last apply.
func hasDriftFromState(outputDir, stateFile, target string) (bool, error) {
	manifest, err := state.ReadState(stateFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	ts, ok := manifest.Targets[target]
	if !ok {
		return false, nil
	}

	for _, artifact := range ts.Artifacts {
		absPath := filepath.Clean(filepath.Join(outputDir, artifact.Path))
		data, err := os.ReadFile(absPath)
		if err != nil {
			return true, nil // missing file is drift
		}
		actualHash := sha256.Sum256(data)
		actual := fmt.Sprintf("sha256:%x", actualHash)
		if actual != artifact.Hash {
			return true, nil
		}
	}
	return false, nil
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

// runMemoryPass executes the memory rendering pass for a single target.
// Returns the seeds to record in the state file (empty on dry-run or targets
// without native memory), fidelity notes to print, and an error if a hard
// failure occurred (e.g., drift detected without --reseed for tracked
// lifecycle entries on the Claude target).
func runMemoryPass(config *ast.XcaffoldConfig, r renderer.TargetRenderer, baseDir, outputDir string, priorSeeds []state.MemorySeed, dryRun, reseed bool) ([]state.MemorySeed, []renderer.FidelityNote, error) {
	if config == nil || len(config.Memory) == 0 {
		return nil, nil, nil
	}

	target := r.Target()

	// Resolve the provider-specific memory output directory. Providers that do
	// not write memory to disk (cursor, copilot) receive an empty string and
	// CompileMemory is expected to treat that as a no-op / notes-only path.
	memDir, err := resolveMemoryOutputDir(target, baseDir, outputDir)
	if err != nil {
		return nil, nil, fmt.Errorf("memory pass: %w", err)
	}

	if dryRun {
		fmt.Fprintf(os.Stderr, "[DRY-RUN] would compile %d memory entries for %s (dir: %s)\n", len(config.Memory), target, memDir)
		return nil, nil, nil
	}

	// Build prior-hash map scoped to this target for drift detection.
	priorHashes := make(map[string]string, len(priorSeeds))
	for _, s := range priorSeeds {
		if s.Target == target {
			priorHashes[s.Name] = s.Hash
		}
	}

	opts := renderer.MemoryOptions{
		OutputDir:   memDir,
		PriorHashes: priorHashes,
		Reseed:      reseed,
	}
	files, notes, err := r.CompileMemory(config, baseDir, opts)
	if err != nil {
		return nil, notes, err
	}

	// Antigravity requires a post-compile disk-write pass: CompileMemory returns
	// the knowledge items as an in-memory map; writeAntigravityKnowledgeItems
	// materialises them to disk and produces state seeds for the lock file.
	if target == targetAntigravity {
		seeds, writeErr := writeAntigravityKnowledgeItems(outputDir, files)
		return seeds, notes, writeErr
	}

	// Seeds are tracked inside the claude MemoryRenderer instantiated by
	// CompileMemory. Seed extraction will be wired when CompileMemory is
	// enhanced to expose the produced seeds; return nil for now.
	return nil, notes, nil
}

// resolveMemoryOutputDir returns the filesystem directory that the memory
// renderer for the given provider should write into. For providers without
// native memory persistence (cursor, copilot) it returns an empty string so
// that CompileMemory operates in notes-only mode.
func resolveMemoryOutputDir(provider, baseDir, fallbackOutputDir string) (string, error) {
	switch provider {
	case targetClaude:
		return claudeProjectMemoryDir(baseDir)
	case targetGemini:
		return geminiMemoryDir()
	case targetAntigravity:
		return fallbackOutputDir, nil
	default:
		// cursor, copilot — no disk memory directory.
		return "", nil
	}
}

// geminiMemoryDir returns the directory where Gemini's GEMINI.md context file
// lives. The default is ~/.gemini/ following Gemini CLI conventions. The
// XCAFFOLD_GEMINI_DIR environment variable overrides this for testing and
// non-standard installations.
func geminiMemoryDir() (string, error) {
	if override := os.Getenv("XCAFFOLD_GEMINI_DIR"); override != "" {
		return filepath.Clean(override), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory for gemini target: %w", err)
	}
	return filepath.Join(home, ".gemini"), nil
}

// convertClaudeSeeds copies the Claude renderer's local MemorySeed slice into
// the state.MemorySeed shape for lock manifest persistence. The two types have
// identical fields but live in different packages to avoid an import cycle.
func convertClaudeSeeds(in []claude.MemorySeed) []state.MemorySeed {
	out := make([]state.MemorySeed, len(in))
	for i, s := range in {
		out[i] = state.MemorySeed{
			Name:      s.Name,
			Target:    s.Target,
			Path:      s.Path,
			Hash:      s.Hash,
			SeededAt:  s.SeededAt,
			Lifecycle: s.Lifecycle,
		}
	}
	return out
}

// writeAntigravityKnowledgeItems writes each in-memory knowledge file produced
// by the Antigravity memory renderer to disk under outputDir. Returns one
// state.MemorySeed per file with lifecycle "seed-once".
func writeAntigravityKnowledgeItems(outputDir string, files map[string]string) ([]state.MemorySeed, error) {
	if len(files) == 0 {
		return nil, nil
	}
	seeds := make([]state.MemorySeed, 0, len(files))
	now := time.Now().UTC().Format(time.RFC3339)
	for relPath, content := range files {
		dest := filepath.Clean(filepath.Join(outputDir, relPath))
		if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
			return nil, fmt.Errorf("antigravity memory: mkdir %s: %w", filepath.Dir(dest), err)
		}
		if err := os.WriteFile(dest, []byte(content), 0o600); err != nil {
			return nil, fmt.Errorf("antigravity memory: write %s: %w", dest, err)
		}
		sum := sha256.Sum256([]byte(content))
		// Derive the entry name from the relative path: knowledge/<name>.md → <name>.
		name := strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath))
		seeds = append(seeds, state.MemorySeed{
			Name:      name,
			Target:    targetAntigravity,
			Path:      dest,
			Hash:      fmt.Sprintf("sha256:%x", sum),
			SeededAt:  now,
			Lifecycle: "seed-once",
		})
	}
	return seeds, nil
}
