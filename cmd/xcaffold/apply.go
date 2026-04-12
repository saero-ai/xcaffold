package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/compiler"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/registry"
	"github.com/saero-ai/xcaffold/internal/resolver"
	"github.com/saero-ai/xcaffold/internal/state"
	"github.com/spf13/cobra"
)

var applyDryRun bool
var applyCheckOnly bool
var applyCheckPermissions bool
var applyForce bool
var applyBackup bool
var applyProjectFlag string
var targetFlag string

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Compile scaffold.xcf into .claude/ agent files",
	Long: `xcaffold apply deterministically compiles your YAML logic into native target outputs.

┌───────────────────────────────────────────────────────────────────┐
│                          COMPILATION PHASE                        │
└───────────────────────────────────────────────────────────────────┘
 [scaffold.xcf] ──(Compiles)──▶ [.claude/agents/*.md]
       │
   (Locks)──▶ [scaffold.lock]

 • Strict one-way generation (YAML -> MD)
 • Generates a cryptographic SHA-256 state manifest (scaffold.lock)
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
	applyCmd.Flags().StringVar(&applyProjectFlag, "project", "", "Apply to an external project registered in the global registry")
	applyCmd.Flags().StringVar(&targetFlag, "target", targetClaude, "compilation target platform (claude, cursor, antigravity, agentsmd; default: claude)")
	rootCmd.AddCommand(applyCmd)
}

const (
	targetClaude      = "claude"
	targetAntigravity = "antigravity"
	targetCursor      = "cursor"
)

// currentSchemaVersion is the schema version this build of xcaffold targets.
// Configs with older versions produce a warning prompting the user to migrate.
const currentSchemaVersion = "1.0"

//nolint:gocyclo
func runApply(cmd *cobra.Command, args []string) error {
	if applyProjectFlag != "" {
		proj, err := registry.Resolve(applyProjectFlag)
		if err != nil {
			return fmt.Errorf("project %q not found in registry: %w", applyProjectFlag, err)
		}
		globalXcfPath = filepath.Join(proj.Path, "scaffold.xcf")
		xcfPath = globalXcfPath
		claudeDir = filepath.Join(proj.Path, ".claude")
		lockPath = filepath.Join(proj.Path, "scaffold.lock")
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

		errors, warnings := securityFieldReport(config, targetFlag)

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
		return applyScope(globalXcfPath, globalXcfHome, globalLockPath, "global")
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
		if err := applyScope(xcfPath, outDir, lockPath, "project"); err != nil {
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

// applyScope compiles a single xcf file into outputDir and writes the lock file
// at lockFile. scopeName is used as a prefix in terminal output when running
// so the user can distinguish global from project compilation.
//
//nolint:gocyclo
func applyScope(configPath, outputDir, lockFile, scopeName string) error {
	// baseDir is the directory containing the xcf file — used by the compiler
	// to resolve instructions_file: and references: paths.
	baseDir := filepath.Dir(configPath)

	config, err := parser.ParseDirectory(baseDir)
	if err != nil {
		return fmt.Errorf("[%s] parse error: %w", scopeName, err)
	}

	if config.Version != "" && config.Version < currentSchemaVersion {
		fmt.Fprintf(os.Stderr, "WARNING: scaffold.xcf uses schema version %s; current schema is %s. Run \"xcaffold migrate\" to upgrade.\n", config.Version, currentSchemaVersion)
	}

	// --- Smart compilation skip: compare source hashes ---
	targetLockFile := state.LockFilePath(lockFile, targetFlag)

	// Auto-migrate legacy scaffold.lock → scaffold.<target>.lock
	migrated, migrateErr := state.MigrateLegacyLock(lockFile, targetFlag)
	if migrateErr != nil {
		fmt.Fprintf(os.Stderr, "[%s] Warning: lock migration failed: %v\n", scopeName, migrateErr)
	} else if migrated {
		fmt.Printf("[%s] Migrated %s -> %s\n", scopeName, filepath.Base(lockFile), filepath.Base(targetLockFile))
	}

	sourceFiles, _ := resolver.FindXCFFiles(baseDir)

	if !applyForce {
		prevManifest, readErr := state.Read(targetLockFile)
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

	out, err := compiler.Compile(config, baseDir, targetFlag)
	if err != nil {
		return fmt.Errorf("[%s] compilation error: %w", scopeName, err)
	}

	// Resolve the target-specific output directory instead of the hardcoded default
	outputDir = filepath.Join(filepath.Dir(outputDir), compiler.OutputDir(targetFlag))

	oldManifest, _ := state.Read(targetLockFile)

	if !applyDryRun && !applyForce {
		drift, err := hasDrift(outputDir, targetLockFile)
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
	} else if targetFlag == "" || targetFlag == targetClaude {
		// Pre-create baseline subdirectories exclusively for the Claude format contract.
		for _, subdir := range []string{"agents", "skills", "rules"} {
			if err := os.MkdirAll(filepath.Join(outputDir, subdir), 0755); err != nil {
				return fmt.Errorf("[%s] failed to create output directory %q: %w", scopeName, subdir, err)
			}
		}
	}

	// Write (or preview) each compiled file.
	hasChanges := false

	cleanOrphans(oldManifest, out, outputDir, scopeName, &hasChanges)

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
		return nil
	}

	// Write the lock file with source tracking.
	manifest := state.GenerateWithOpts(out, state.GenerateOpts{
		Target:      targetFlag,
		Scope:       scopeName,
		ConfigDir:   ".",
		SourceFiles: sourceFiles,
		BaseDir:     baseDir,
	})
	if err := state.Write(manifest, targetLockFile); err != nil {
		return fmt.Errorf("[%s] failed to write %s: %w", scopeName, filepath.Base(targetLockFile), err)
	}

	fmt.Printf("\n[%s] ✓ Apply complete. %s updated.\n", scopeName, filepath.Base(targetLockFile))

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

func hasDrift(outputDir, lockFile string) (bool, error) {
	manifest, err := state.Read(lockFile)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // no lock file means no drift (first run)
		}
		return false, err
	}

	for _, artifact := range manifest.Artifacts {
		absPath := filepath.Clean(filepath.Join(outputDir, artifact.Path))
		data, err := os.ReadFile(absPath)
		if err != nil {
			return true, nil // missing file is drift
		}
		actualHash := sha256.Sum256(data)
		actual := fmt.Sprintf("sha256:%x", actualHash)
		if actual != artifact.Hash {
			return true, nil // content drift
		}
	}
	return false, nil
}

func performBackup(outputDir, target, backupDirConfig, scopeName string) error {
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		return nil // nothing to backup
	}

	timestamp := time.Now().Format("20060102_150405")
	bakName := fmt.Sprintf(".%s_bak_%s", target, timestamp)
	if target == "" {
		bakName = fmt.Sprintf(".claude_bak_%s", timestamp)
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
// target by inspecting which security fields in the config would be dropped.
// It is read-only and never modifies any files.
//
// The claude target supports all security fields; cursor and antigravity drop
// settings.Permissions, settings.Sandbox, and per-agent security fields.
func securityFieldReport(config *ast.XcaffoldConfig, target string) (errors, warnings []string) {
	switch target {
	case "cursor", "antigravity":
		label := target

		if config.Settings.Permissions != nil {
			warnings = append(warnings, fmt.Sprintf("%s: settings.permissions will be dropped — no enforcement equivalent", label))
		}
		if config.Settings.Sandbox != nil {
			warnings = append(warnings, fmt.Sprintf("%s: settings.sandbox will be dropped — no sandbox model", label))
		}

		for id, agent := range config.Agents {
			if agent.PermissionMode != "" {
				warnings = append(warnings, fmt.Sprintf("%s: agent %q permissionMode %q will be dropped", label, id, agent.PermissionMode))
			}
			if len(agent.DisallowedTools) > 0 {
				warnings = append(warnings, fmt.Sprintf("%s: agent %q disallowedTools will be dropped — tool restrictions will NOT be enforced", label, id))
			}
			if agent.Isolation != "" {
				warnings = append(warnings, fmt.Sprintf("%s: agent %q isolation %q will be dropped", label, id, agent.Isolation))
			}
		}

		// Agent vs deny conflicts (errors)
		if config.Settings.Permissions != nil {
			for agentID, agent := range config.Agents {
				for _, tool := range agent.Tools {
					for _, denyRule := range config.Settings.Permissions.Deny {
						if denyRule == tool {
							errors = append(errors, fmt.Sprintf("permissions.deny: rule %q conflicts with agent %q tools list", tool, agentID))
						}
					}
				}
			}
		}

	default:
		// claude and other targets support all security fields — no findings
	}
	return errors, warnings
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

func cleanOrphans(oldManifest *state.LockManifest, out *compiler.Output, outputDir, scopeName string, hasChanges *bool) {
	orphans := state.FindOrphans(oldManifest, out.Files)
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
