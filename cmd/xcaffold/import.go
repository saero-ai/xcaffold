package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/bir"
	"github.com/saero-ai/xcaffold/internal/importer"
	_ "github.com/saero-ai/xcaffold/internal/importer/antigravity"
	_ "github.com/saero-ai/xcaffold/internal/importer/claude"
	_ "github.com/saero-ai/xcaffold/internal/importer/copilot"
	_ "github.com/saero-ai/xcaffold/internal/importer/cursor"
	_ "github.com/saero-ai/xcaffold/internal/importer/gemini"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/registry"
	"github.com/saero-ai/xcaffold/internal/translator"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	importSource       string
	importFromPlatform string
	importPlan         bool
	importWithMemory   bool
	autoMergeFlag      string
)

// htmlCommentAttrRE matches key="value" pairs inside HTML comment lines.
// Declared at package scope to avoid recompilation on every call.
var htmlCommentAttrRE = regexp.MustCompile(`(\w[\w-]*)="([^"]*)"`)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import existing provider config into project.xcf",
	Long: `xcaffold import manages adopting existing configurations into xcaffold.

┌───────────────────────────────────────────────────────────────────┐
│                          IMPORT PHASE                             │
└───────────────────────────────────────────────────────────────────┘
Native Import Mode (Default):
 • Scans .claude/agents/*.md   → extracts to agents/<id>.md
 • Scans .claude/skills/*/SKILL.md → extracts to skills/<id>/SKILL.md
 • Scans .claude/rules/*.md    → extracts to rules/<id>.md
 • Reads .claude/settings.json for MCP and settings context
 • Generates a project.xcf with instructions-file: references

Cross-Platform Translation Mode (--source):
 • Imports agent workflow files from other platforms and decomposes
   them into xcaffold primitives (skills, rules, permissions).
 • Detected intents determine primitive mappings.
 • Results are injected into project.xcf using instructions-file: references.
 • Use --plan to preview the decomposition without writing any files.

Usage:
  $ xcaffold import
  $ xcaffold import --source ./workflows/ --from antigravity
  $ xcaffold import --source .cursor/rules/ --from cursor --plan
  $ xcaffold import --source .gemini/ --from gemini`,
	RunE: runImport,
}

func init() {
	importCmd.Flags().StringVar(&importSource, "source", "", "File or directory of workflow markdown files to translate")
	importCmd.Flags().StringVar(&importFromPlatform, "from", "auto", "Source platform of input files (antigravity, claude, cursor, gemini, copilot)")
	importCmd.Flags().BoolVar(&importPlan, "plan", false, "Dry-run: print decomposition plan without writing files")
	importCmd.Flags().BoolVar(&importWithMemory, "with-memory", false, "Snapshot agent-written memory into xcf/agents/<id>/memory/ sidecars")
	importCmd.Flags().StringVar(&autoMergeFlag, "auto-merge", "", "Merge divergent variants: union")
	rootCmd.AddCommand(importCmd)
}

func runImport(cmd *cobra.Command, args []string) error {
	if importSource != "" && !importWithMemory {
		return runTranslateMode()
	}

	var importErr error

	if importSource == "" {
		if globalFlag {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("cannot determine home directory: %w", err)
			}
			dirs := detectPlatformDirs(home, true)
			if len(dirs) == 0 {
				return fmt.Errorf("no global platform directories found (~/.claude/, ~/.cursor/, ~/.agents/)")
			}
			if len(dirs) > 1 {
				importErr = mergeImportDirs(dirs, globalXcfPath)
			} else {
				importErr = importScope(dirs[0].dirName, globalXcfPath, "global", dirs[0].platform)
			}
		} else {
			// project (default) — detect providers via ProviderImporter registry.
			detected := importer.DetectProviders(".", importer.DefaultImporters())
			if len(detected) > 1 {
				var provDirs []platformDirInfo
				for _, imp := range detected {
					provDirs = append(provDirs, platformDirInfo{
						dirName:  imp.InputDir(),
						platform: imp.Provider(),
						exists:   true,
					})
				}
				importErr = mergeImportDirs(provDirs, "project.xcf")
			} else if len(detected) == 1 {
				imp := detected[0]
				importErr = importScope(imp.InputDir(), "project.xcf", "project", imp.Provider())
			} else {
				// No provider directories found
				importErr = fmt.Errorf("no supported AI provider configuration found in current directory. Supported providers: Claude Code, Gemini CLI, Cursor, GitHub Copilot, Antigravity")
			}
		}
	} else {
		// --source is set together with --with-memory: run translate mode first.
		importErr = runTranslateMode()
	}

	if importErr != nil {
		return importErr
	}

	if importWithMemory {
		memSummary, err := runMemorySnapshot(cmd, importSource, importFromPlatform, importPlan)
		if err != nil {
			return fmt.Errorf("memory snapshot: %w", err)
		}
		printMemorySnapshotSummary(cmd, memSummary, importPlan)
	}

	return nil
}

// runProjectInstructionsDiscovery runs extractProjectInstructions for the primary
// provider, then checks for a secondary provider's instruction files and invokes
// detectAndMergeVariants if found.
//
// Secondary provider detection:
//   - primary=claude and AGENTS.md found in tree → secondary=cursor
//   - primary=cursor and CLAUDE.md found in tree → secondary=claude
//   - Other combinations are not yet implemented; they are logged and skipped.
//
// The autoMergeFlag ("union") is forwarded to detectAndMergeVariants.
// If the xcf config file cannot be parsed (e.g., it doesn't exist yet or has no
// project block), the function returns nil without error — project instructions
// discovery is best-effort and must not block the import.

// providerInstructionsFilename returns the canonical root instruction filename
// for the given provider, or "" if the provider does not have one.

// anyInstructionFileExists reports whether filename exists anywhere under root —
// either as a direct child (root instruction file) or within a subdirectory.
// This is used to gate project-instruction discovery: we run discovery whenever
// ANY scoped instruction file exists, not only when the root-level file exists.

// detectSecondaryProvider returns the secondary provider name when a second
// provider's instruction files are present alongside the primary provider's tree.
// Returns "" when no secondary is detected or the combination is not yet supported.
func detectSecondaryProvider(projectDir, primaryProvider string) string {
	switch primaryProvider {
	case "claude":
		// If AGENTS.md exists anywhere in the tree, cursor is a secondary provider.
		if fileExistsInTree(projectDir, "AGENTS.md") {
			return "cursor"
		}
	case "cursor":
		// If CLAUDE.md exists anywhere in the tree, claude is a secondary provider.
		if fileExistsInTree(projectDir, "CLAUDE.md") {
			return "claude"
		}
		// Other combinations (e.g., antigravity+cursor) are not yet implemented.
		// Add them here as new provider pairs are defined.
	}
	return ""
}

// directoryFilter maintains a list of exact directory names to ignore
// during tree traversals (e.g., .git, .worktrees, node_modules).
type directoryFilter struct {
	ignored map[string]bool
}

// newDirectoryFilter creates a filter pre-populated with standard blocked directories
// and extracts basic top-level exclusions from the project's .gitignore if present.
func newDirectoryFilter(projectDir string) *directoryFilter {
	filter := &directoryFilter{
		ignored: map[string]bool{
			".git":         true,
			".worktrees":   true,
			"node_modules": true,
			"vendor":       true,
			".venv":        true,
			".xcaffold":    true,
			"xcf":          true,
			".claude":      true,
			".cursor":      true,
			".gemini":      true,
			".agents":      true,
			"dist":         true,
			"build":        true,
			"coverage":     true,
		},
	}

	gitignorePath := filepath.Join(projectDir, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			// Extremely naive exclusion parser: if it targets a top-level dir (e.g., "target/", "/target")
			clean := strings.TrimPrefix(line, "/")
			clean = strings.TrimSuffix(clean, "/")
			// Ignore complex glob patterns for this fast-path skipping
			if !strings.ContainsAny(clean, "*?[") {
				filter.ignored[clean] = true
			}
		}
	}
	return filter
}

// shouldSkip reports whether the directory name should be skipped.
func (f *directoryFilter) shouldSkip(name string) bool {
	return f.ignored[name]
}

// fileExistsInTree reports whether a file with the given name exists anywhere
// under rootDir (recursive). It skips standard blocked directories. Returns false on any walk error.
func fileExistsInTree(rootDir, name string) bool {
	found := false
	filter := newDirectoryFilter(rootDir)

	_ = filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if d.IsDir() {
			if path != rootDir && filter.shouldSkip(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() == name {
			found = true
		}
		return nil
	})
	return found
}

// detectPlatformDirs scans known provider directories under baseDir and returns
// all found entries, sorted by total resource count descending. When skipEmpty
// is true, directories with no detected resources are excluded from the result.
// dirName in each returned entry is the absolute path to the provider directory.
func detectPlatformDirs(baseDir string, skipEmpty bool) []platformDirInfo {
	platformDirs := []struct{ dir, platform string }{
		{".claude", "claude"},
		{".cursor", "cursor"},
		{".agents", "antigravity"},
		{".gemini", "gemini"},
	}

	var results []platformDirInfo

	for _, pt := range platformDirs {
		targetPath := filepath.Join(baseDir, pt.dir)
		if _, err := os.Stat(targetPath); err != nil {
			continue
		}

		info := platformDirInfo{exists: true, platform: pt.platform, dirName: targetPath}

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

		if skipEmpty && info.agents+info.skills+info.rules+info.workflows == 0 {
			continue
		}

		results = append(results, info)
	}

	sort.Slice(results, func(i, j int) bool {
		totalI := results[i].agents + results[i].skills + results[i].rules + results[i].workflows
		totalJ := results[j].agents + results[j].skills + results[j].rules + results[j].workflows
		return totalI > totalJ
	})

	return results
}

func runTranslateMode() error {
	xcfPath := "project.xcf"
	config, err := parser.ParseFileExact(xcfPath)
	if err != nil {
		return fmt.Errorf("no project.xcf found — run 'xcaffold init' first, then 'xcaffold import --source': %w", err)
	}

	xcfAbs, err := filepath.Abs(xcfPath)
	if err != nil {
		return fmt.Errorf("could not resolve project.xcf path: %w", err)
	}
	baseDir := filepath.Dir(xcfAbs)

	sources := []string{importSource}

	if len(sources) == 0 {
		return fmt.Errorf("no .md files found at %q", importSource)
	}

	var allResults []translator.TranslationResult
	totalPrimitives := 0
	targetFlag := targetClaude

	for _, src := range sources {
		unit, err := bir.ImportWorkflow(src, importFromPlatform)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: skipping %s: %v\n", src, err)
			continue
		}

		result := translator.Translate(unit, targetFlag)
		allResults = append(allResults, result)

		fmt.Printf("\n%s\n", filepath.Base(src))
		fmt.Printf("  intents detected: %s\n", formatIntents(unit.Intents))
		fmt.Printf("  primitives:\n")

		for _, p := range result.Primitives {
			fmt.Printf("    [%s] %s\n", p.Kind, p.ID)
			totalPrimitives++
		}
	}

	fmt.Printf("\n%d file(s), %d primitive(s) total\n",
		len(sources), totalPrimitives)

	if importPlan {
		printTranslatePlan(allResults, baseDir)
		fmt.Println("(dry-run — no files written)")
		return nil
	}

	return injectIntoConfig(config, allResults, xcfPath, baseDir)
}

// importScope scans a platform directory and writes a xcf file to xcfDest.
// provider selects provider-specific extraction logic for settings, MCP,
// hooks, project-instruction files, and memory. Supported values match the
// platform field from platformDirInfo: "claude", "gemini", "cursor",
// "copilot", "antigravity". An empty string or unknown value falls back to
// Claude-style extraction (settings.json + hooks.json).
func importScope(platformDir, xcfDest, scopeName, provider string) error {
	if _, err := os.Stat(xcfDest); err == nil {
		return fmt.Errorf("[%s] %s already exists. Remove it first to import", scopeName, xcfDest)
	}
	if err := checkXcfDirPreexistence(".", scopeName); err != nil {
		return err
	}

	// projectDir is the directory that contains the provider sub-directory.
	// For a project-local import (e.g. .claude/ inside the project root),
	// this is the current working directory. We compute it from platformDir
	// so it works for both relative and absolute paths.
	platformAbs, err := filepath.Abs(platformDir)
	if err != nil {
		return fmt.Errorf("resolving provider dir: %w", err)
	}
	projectDir := filepath.Dir(platformAbs)

	projectName := inferProjectName()
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{Name: projectName},
		ResourceScope: ast.ResourceScope{
			Agents: make(map[string]ast.AgentConfig),
			Skills: make(map[string]ast.SkillConfig),
			Rules:  make(map[string]ast.RuleConfig),
			MCP:    make(map[string]ast.MCPConfig),
		},
	}

	importCount := 0
	var warnings []string

	// ── 1. Delegate resource extraction to the registered ProviderImporter. ──────
	// Each provider importer handles its own file layout, frontmatter parsing, and
	// JSON key splitting.
	providerImp := findImporterByProvider(provider)
	if providerImp != nil {
		if err := providerImp.Import(platformDir, config); err != nil {
			warnings = append(warnings, fmt.Sprintf("%s import: %v", provider, err))
		}
		// Surface per-file extraction warnings from importers that support it.
		// Importers that expose a Warnings []string field satisfy this interface.
		type warningImporter interface {
			GetWarnings() []string
		}
		if wi, ok := providerImp.(warningImporter); ok {
			for _, w := range wi.GetWarnings() {
				warnings = append(warnings, fmt.Sprintf("%s: %s", provider, w))
			}
		}
		// Copy skill supporting files — ProviderImporter populates AST only; side-car
		// files under known subdirectories must still be copied to xcf/skills/<id>/.
		for id := range config.Skills {
			skillFile := filepath.Join(platformDir, "skills", id, "SKILL.md")
			if _, err := os.Stat(skillFile); err == nil {
				refs, scripts, fileAssets, fileExamples, _ := extractSkillSubdirs(skillFile, id, provider, "", &warnings)
				sc := config.Skills[id]
				if len(refs) > 0 {
					sc.References = refs
				}
				if len(scripts) > 0 {
					sc.Scripts = scripts
				}
				if len(fileAssets) > 0 {
					sc.Assets = fileAssets
				}
				if len(fileExamples) > 0 {
					sc.Examples = fileExamples
				}
				config.Skills[id] = sc
			}
		}
		// Attempt to graduate any extras the importer stashed in ProviderExtras.
		// A file placed in extras during Import() (e.g. because it was unrecognised
		// at scan time) may now be classifiable — ReclassifyExtras runs a second
		// pass and promotes those files into the typed AST.
		if err := parser.ReclassifyExtras(config, importer.DefaultImporters()); err != nil {
			warnings = append(warnings, fmt.Sprintf("reclassify extras: %v", err))
		}

		// Count resources extracted by the importer.
		importCount += len(config.Agents) + len(config.Skills) + len(config.Rules) +
			len(config.Workflows) + len(config.MCP)
	}

	// ── 2. Provider-specific post-import steps (cross-boundary files, out-of-tree
	// memory sources, unsupported-provider warnings).
	if err := runProviderPostImport(provider, platformDir, projectDir, config, &warnings); err != nil {
		return err
	}

	// ── 3. Root context file discovery ──────────────────────────────────────────
	discoverRootContextFiles(projectDir, config)

	// ── 4. Agent memory ─────────────────────────────────────────────────────────
	// Memory is extracted into config.Memory by the ProviderImporter during
	// Import() (e.g. claude extracts agent-memory/**). Write each entry as a
	// plain .md file to xcf/agents/<agentID>/memory/<name>.md so the compiler
	// discovers them via convention-based filesystem scanning.
	if memCount, err := writeMemoryFiles(config); err != nil {
		return err
	} else if memCount > 0 {
		fmt.Printf("  Agent memory: %d entry(ies) → xcf/agents/<id>/memory/\n", memCount)
	}
	// Detect compilation targets from the scanned platform directory.
	if config.Project != nil {
		config.Project.Targets = detectTargets(platformDir)
		config.Project.AgentRefs = sortedAgentRefs(config.Agents)
		config.Project.SkillRefs = sortedMapKeys(config.Skills)
		config.Project.RuleRefs = sortedMapKeys(config.Rules)
		config.Project.WorkflowRefs = sortedMapKeys(config.Workflows)
		config.Project.MCPRefs = sortedMapKeys(config.MCP)
		// Propagate instructions-file from project-instruction discovery.
	}

	// Write split .xcf files: project.xcf (kind: project) + xcf/**/*.xcf
	if err := WriteSplitFiles(config, "."); err != nil {
		return fmt.Errorf("[%s] failed to write split xcf files: %w", scopeName, err)
	}

	// Prune orphan memory imported from raw provider sidecars
	if err := pruneOrphanMemory(config, "."); err != nil {
		return fmt.Errorf("prune memory: %w", err)
	}

	// Removed invalid project instructions block

	fmt.Printf("[%s] ✓ Import complete. Created %s with %d resources.\n", scopeName, xcfDest, importCount)
	fmt.Printf("  Split xcf/ files written to xcf/ directory.\n")
	fmt.Println("  Run 'xcaffold apply' when ready to assume management.")

	cwd, _ := os.Getwd()
	if config.Project != nil {
		_ = registry.Register(cwd, config.Project.Name, nil, ".")
	}

	if len(warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, w := range warnings {
			fmt.Println(" ⚠", w)
		}
	}
	return nil
}

// snapshotAgentMemoryDir copies individual .md memory files from an in-project
// agent-memory directory (e.g. .claude/agent-memory/) into xcf/agents/<id>/memory/.
// MEMORY.md index files are skipped (auto-generated by apply).
// Each sub-directory (agent name) becomes xcf/agents/<agentName>/memory/.
// Returns the number of agent memory directories successfully snapshotted.
func snapshotAgentMemoryDir(agentMemDir string) (int, error) {
	entries, err := os.ReadDir(agentMemDir)
	if err != nil {
		return 0, fmt.Errorf("reading agent-memory dir: %w", err)
	}
	snapshotted := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		agentName := entry.Name()
		srcDir := filepath.Join(agentMemDir, agentName)
		dstDir := filepath.Join("xcf", "agents", agentName, "memory")
		if err := copyDirContents(srcDir, dstDir); err != nil {
			return snapshotted, fmt.Errorf("copying agent memory for %q: %w", agentName, err)
		}
		snapshotted++
	}
	return snapshotted, nil
}

// copyDirContents recursively copies all files from src into dst,
// creating dst and any intermediate directories as needed.
func copyDirContents(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip auto-generated memory index files.
		if !d.IsDir() && d.Name() == "MEMORY.md" {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o600)
	})
}

// importSettings parses settings.json and populates MCP, rules, and settings.
func importSettings(data []byte, config *ast.XcaffoldConfig, count *int, warnings *[]string) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	importMCPServers(raw, config, count)
	importStatusAndPlugins(raw, config)

	// Import hooks if they exist in settings.json
	if hooksRaw, ok := raw["hooks"]; ok {
		hooksBytes, err := json.Marshal(hooksRaw)
		if err == nil {
			var hooks ast.HookConfig
			if err := json.Unmarshal(hooksBytes, &hooks); err == nil {
				config.Hooks = map[string]ast.NamedHookConfig{"default": {Name: "default", Events: hooks}}
				*count++
			}
		}
	}

	return nil
}

func importMCPServers(raw map[string]interface{}, config *ast.XcaffoldConfig, count *int) {
	if mcpRaw, ok := raw["mcpServers"].(map[string]interface{}); ok {
		for id, serverRaw := range mcpRaw {
			serverMap, ok := serverRaw.(map[string]interface{})
			if !ok {
				continue
			}
			mc := ast.MCPConfig{}
			if cmdStr, ok := serverMap["command"].(string); ok {
				mc.Command = cmdStr
			}
			if argsRaw, ok := serverMap["args"].([]interface{}); ok {
				for _, a := range argsRaw {
					if argStr, ok := a.(string); ok {
						mc.Args = append(mc.Args, argStr)
					}
				}
			}
			if envRaw, ok := serverMap["env"].(map[string]interface{}); ok {
				mc.Env = make(map[string]string, len(envRaw))
				for k, v := range envRaw {
					if vStr, ok := v.(string); ok {
						mc.Env[k] = vStr
					}
				}
			}
			config.MCP[id] = mc
			*count++
		}
	}
}

func importStatusAndPlugins(raw map[string]interface{}, config *ast.XcaffoldConfig) {
	settings := ast.SettingsConfig{}
	changed := false

	if slRaw, ok := raw["statusLine"].(map[string]interface{}); ok {
		settings.StatusLine = &ast.StatusLineConfig{}
		if t, ok := slRaw["type"].(string); ok {
			settings.StatusLine.Type = t
		}
		if c, ok := slRaw["command"].(string); ok {
			settings.StatusLine.Command = c
		}
		changed = true
	}

	if epRaw, ok := raw["enabledPlugins"].(map[string]interface{}); ok {
		settings.EnabledPlugins = make(map[string]bool)
		for k, v := range epRaw {
			if b, ok := v.(bool); ok {
				settings.EnabledPlugins[k] = b
			}
		}
		changed = true
	}

	if el, ok := raw["effortLevel"].(string); ok {
		settings.EffortLevel = el
		changed = true
	}
	if atk, ok := raw["alwaysThinkingEnabled"].(bool); ok {
		settings.AlwaysThinkingEnabled = &atk
		changed = true
	}

	if changed {
		config.Settings = map[string]ast.SettingsConfig{"default": settings}
	}
}

// providerSubdirMap maps provider-native subdirectory names to the canonical
// xcaffold subdir names (references, scripts, assets, examples). An empty
// string value means the subdir has no canonical mapping and its files are
// routed to the provider-native passthrough directory.
var providerSubdirMap = map[string]map[string]string{
	"claude": {
		"references": "references",
		"scripts":    "scripts",
	},
	"gemini": {
		"references": "references",
		"scripts":    "scripts",
		"assets":     "assets",
	},
	"cursor": {
		"references": "references",
		"scripts":    "scripts",
		"assets":     "assets",
	},
	"copilot": {}, // co-located — classify by extension
	"antigravity": {
		"examples":  "examples",
		"scripts":   "scripts",
		"resources": "assets",
	},
}

// extractSkillSubdirs scans the skill directory for known canonical and
// provider-native subdirectories, copies their files to xcf/skills/<id>/,
// and returns slices of copied paths grouped by canonical category.
//
// outDir is the base directory for output paths (xcf/skills/<id>/...).  When
// empty, the current working directory is used.
//
// For Claude imports, any .md file alongside SKILL.md (not in a subdirectory)
// is treated as a reference.
//
// Files from subdirectories that have no canonical mapping are copied to
// xcf/provider/<provider>/skills/<id>/<subdir>/.
func extractSkillSubdirs(skillFile, id, provider, outDir string, warnings *[]string) (refs, scripts, assets, examples []string, err error) {
	skillDir := filepath.Dir(skillFile)

	// Determine the base for output paths.
	base := outDir

	subdirMap := providerSubdirMap[provider] // nil if provider unknown
	if subdirMap == nil {
		*warnings = append(*warnings, fmt.Sprintf("extractSkillSubdirs: unknown provider %q — all subdirectory files routed to passthrough", provider))
	}

	// Walk all direct children of the skill directory.
	entries, readErr := os.ReadDir(skillDir)
	if readErr != nil {
		// If the directory cannot be read at all, return empty (not an error).
		return nil, nil, nil, nil, nil
	}

	// Helper: copy a file and append to the appropriate slice.
	appendCopied := func(src, canonicalSubdir, filename string) {
		// The xcf-relative path is always outDir-agnostic — it is what gets
		// stored in AST SkillConfig fields (References, Scripts, Assets, Examples).
		xcfRelPath := filepath.ToSlash(filepath.Join("xcf", "skills", id, canonicalSubdir, filename))
		var dest string
		if base != "" {
			dest = filepath.Join(base, "xcf", "skills", id, canonicalSubdir, filename)
		} else {
			dest = filepath.Join("xcf", "skills", id, canonicalSubdir, filename)
		}
		if copyErr := copyFile(src, dest); copyErr != nil {
			*warnings = append(*warnings, fmt.Sprintf("failed to copy skill file %s: %v", src, copyErr))
			return
		}
		switch canonicalSubdir {
		case "references":
			refs = append(refs, xcfRelPath)
		case "scripts":
			scripts = append(scripts, xcfRelPath)
		case "assets":
			assets = append(assets, xcfRelPath)
		case "examples":
			examples = append(examples, xcfRelPath)
		}
	}

	// Helper: copy a file to the provider passthrough directory.
	appendPassthrough := func(src, subdir, filename string) {
		var dest string
		if base != "" {
			dest = filepath.Join(base, "xcf", "provider", provider, "skills", id, subdir, filename)
		} else {
			dest = filepath.Join("xcf", "provider", provider, "skills", id, subdir, filename)
		}
		if copyErr := copyFile(src, dest); copyErr != nil {
			*warnings = append(*warnings, fmt.Sprintf("failed to copy provider file %s: %v", src, copyErr))
		}
	}

	for _, entry := range entries {
		name := entry.Name()

		if entry.IsDir() {
			// Determine canonical mapping for this subdir.
			var canonicalSubdir string
			if subdirMap != nil {
				canonicalSubdir = subdirMap[name] // empty string = no mapping
			}

			// Walk files in this subdir (non-recursive — one level only).
			subEntries, _ := os.ReadDir(filepath.Join(skillDir, name))
			for _, sub := range subEntries {
				if sub.IsDir() {
					continue
				}
				src := filepath.Join(skillDir, name, sub.Name())
				if canonicalSubdir != "" {
					appendCopied(src, canonicalSubdir, sub.Name())
				} else {
					appendPassthrough(src, name, sub.Name())
				}
			}
			continue
		}

		// Non-directory entry alongside SKILL.md.
		// For Claude: any .md file that is not SKILL.md is treated as a reference.
		if provider == "claude" && strings.ToLower(name) != "skill.md" && strings.HasSuffix(strings.ToLower(name), ".md") {
			appendCopied(filepath.Join(skillDir, name), "references", name)
		}
	}

	return refs, scripts, assets, examples, nil
}

// extractBodyAfterFrontmatter returns the markdown body that follows the YAML frontmatter block.
// If the data has no frontmatter (does not start with "---\n"), the entire content is returned.
// Leading and trailing whitespace is trimmed from the returned body.
func extractBodyAfterFrontmatter(data []byte) string {
	if !bytes.HasPrefix(data, []byte("---\n")) && !bytes.HasPrefix(data, []byte("---\r\n")) {
		return strings.TrimSpace(string(data))
	}
	// Find the closing "---" marker after the opening one.
	idx := bytes.Index(data[4:], []byte("\n---"))
	if idx == -1 {
		return strings.TrimSpace(string(data))
	}
	// Skip past the closing "---\n" line to get to the body.
	bodyStart := 4 + idx + len("\n---")
	if bodyStart >= len(data) {
		return ""
	}
	// Consume a trailing newline after the closing marker.
	if data[bodyStart] == '\n' {
		bodyStart++
	}
	return strings.TrimSpace(string(data[bodyStart:]))
}

// detectTargets derives compilation target names from platform directory base names.
// ".claude" → "claude", ".agents" → "antigravity", ".cursor" → "cursor".
// The result is sorted for deterministic output.
func detectTargets(baseDirs ...string) []string {
	targetMap := map[string]bool{}
	for _, dir := range baseDirs {
		switch filepath.Base(filepath.Clean(dir)) {
		case ".claude":
			targetMap["claude"] = true
		case ".agents":
			targetMap["antigravity"] = true
		case ".cursor":
			targetMap["cursor"] = true
		case ".gemini":
			targetMap["gemini"] = true
		case ".github":
			targetMap["copilot"] = true
		}
	}
	targets := make([]string, 0, len(targetMap))
	for t := range targetMap {
		targets = append(targets, t)
	}
	sort.Strings(targets)
	return targets
}

// extractFrontmatter isolates the YAML section between `---` markers at the start of a markdown file.
func extractFrontmatter(data []byte) ([]byte, bool) {
	if !bytes.HasPrefix(data, []byte("---\n")) && !bytes.HasPrefix(data, []byte("---\r\n")) {
		return nil, false
	}
	// The frontmatter must be closed by another '---'.
	// data[4:] starts after the first '---' block.
	idx := bytes.Index(data[4:], []byte("\n---"))
	if idx == -1 {
		return nil, false
	}
	return data[4 : 4+idx], true
}

// writeSidecar writes content to path, creating parent directories as needed.
// The file is written with 0o600 permissions.
func writeSidecar(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating sidecar directory: %w", err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return fmt.Errorf("writing sidecar %s: %w", path, err)
	}
	return nil
}

type instructionsXCFDoc struct {
	Kind         string `yaml:"kind"`
	Version      string `yaml:"version"`
	Name         string `yaml:"name"`
	Instructions string `yaml:"instructions"`
}

// extractProjectInstructions discovers provider instruction files in projectDir,
// writes sidecars under xcf/instructions/, and populates cfg.Project with
// InstructionsFile and InstructionsScopes entries.
// provider is one of: claude, gemini, cursor, copilot, antigravity.

// pathToSlug converts a relative directory path to a sidecar filename slug.
// "packages/worker" → "packages-worker"

// extractCopilotInstructions handles Copilot's dual-mode instruction discovery.
// If .github/copilot-instructions.md exists, flat mode is used: the file is written
// as a sidecar and set as cfg.Project.InstructionsFile.
// Otherwise, falls back to cursor-style nested AGENTS.md discovery.

// lenientUnmarshal attempts to unmarshal YAML, and if it fails, applies a sanitizer
// to auto-quote string values that contain colons (which otherwise break yaml mappings)
// and tries again.
func lenientUnmarshal(data []byte, v interface{}) error {
	err := yaml.Unmarshal(data, v)
	if err == nil {
		return nil
	}
	sanitized := sanitizeFrontmatter(data)
	if fallbackErr := yaml.Unmarshal(sanitized, v); fallbackErr == nil {
		return nil
	}
	return err
}

// sanitizeFrontmatter auto-quotes top-level scalar values that contain unquoted colons.
func sanitizeFrontmatter(data []byte) []byte {
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if len(line) == 0 || line[0] == ' ' || line[0] == '\t' || line[0] == '-' {
			continue
		}
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 {
			val := strings.TrimSpace(parts[1])
			if strings.Contains(val, ":") && len(val) > 0 {
				switch val[0] {
				case '"', '\'', '[', '{', '>', '|':
					continue
				default:
					escapedVal := strings.ReplaceAll(val, "\"", "\\\"")
					lines[i] = fmt.Sprintf("%s: \"%s\"", parts[0], escapedVal)
				}
			}
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

// injectIntoConfig writes external .md files for each primitive and updates
// project.xcf with instructions-file: references, following the import.go pattern.
func injectIntoConfig(config *ast.XcaffoldConfig, results []translator.TranslationResult, xcfPath, baseDir string) error {
	if config.Skills == nil {
		config.Skills = make(map[string]ast.SkillConfig)
	}
	if config.Rules == nil {
		config.Rules = make(map[string]ast.RuleConfig)
	}

	seen := make(map[string]bool)
	var allowEntries []string

	for _, result := range results {
		for _, p := range result.Primitives {
			if strings.TrimSpace(p.Body) == "" {
				continue
			}
			if err := injectPrimitive(&p, config, baseDir, &allowEntries, seen); err != nil {
				return err
			}
		}
	}

	injectAllowEntries(config, allowEntries)

	if err := WriteSplitFiles(config, filepath.Dir(xcfPath)); err != nil {
		return fmt.Errorf("failed to write project.xcf: %w", err)
	}

	fmt.Printf("\nproject.xcf updated. Run 'xcaffold apply' to render output\n")
	return nil
}

func injectPrimitive(p *translator.TargetPrimitive, config *ast.XcaffoldConfig, baseDir string, allowEntries *[]string, seen map[string]bool) error {
	switch p.Kind {
	case "skill":
		destPath := filepath.Join(baseDir, "skills", p.ID, "SKILL.md")
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("failed to create skills/%s/ directory: %w", p.ID, err)
		}
		if err := os.WriteFile(destPath, []byte(p.Body), 0600); err != nil {
			return fmt.Errorf("failed to write %s: %w", destPath, err)
		}

		config.Skills[p.ID] = ast.SkillConfig{
			Description: fmt.Sprintf("Translated from workflow %s", p.ID),
			Body:        p.Body,
		}
		fmt.Printf("  wrote %s\n", destPath)

	case "rule":
		destPath := filepath.Join(baseDir, "rules", p.ID+".md")
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("failed to create rules/ directory: %w", err)
		}
		if err := os.WriteFile(destPath, []byte(p.Body), 0600); err != nil {
			return fmt.Errorf("failed to write %s: %w", destPath, err)
		}

		config.Rules[p.ID] = ast.RuleConfig{
			Description: fmt.Sprintf("Constraints from workflow %s", p.ID),
			Body:        p.Body,
		}
		fmt.Printf("  wrote %s\n", destPath)

	case "permission":
		for _, entry := range resolveAllowEntries(p.Body) {
			if !seen[entry] {
				seen[entry] = true
				*allowEntries = append(*allowEntries, entry)
			}
		}
	}
	return nil
}

func injectAllowEntries(config *ast.XcaffoldConfig, allowEntries []string) {
	if len(allowEntries) == 0 {
		return
	}
	// Retrieve (or create) the "default" settings entry.
	settings := config.Settings["default"]
	if settings.Permissions == nil {
		settings.Permissions = &ast.PermissionsConfig{}
	}
	existing := make(map[string]bool, len(settings.Permissions.Allow))
	for _, e := range settings.Permissions.Allow {
		existing[e] = true
	}
	for _, entry := range allowEntries {
		if !existing[entry] {
			settings.Permissions.Allow = append(settings.Permissions.Allow, entry)
		}
	}
	if config.Settings == nil {
		config.Settings = make(map[string]ast.SettingsConfig)
	}
	config.Settings["default"] = settings
	fmt.Printf("  merged %d permission allow entries into settings.permissions\n", len(allowEntries))
}

// printTranslatePlan prints what would be injected without writing any files.
func printTranslatePlan(results []translator.TranslationResult, baseDir string) {
	fmt.Println("\n-- plan --")
	for _, result := range results {
		for _, p := range result.Primitives {
			switch p.Kind {
			case "skill":
				fmt.Printf("  skill  %q → skills/%s/SKILL.md\n", p.ID, p.ID)
			case "rule":
				fmt.Printf("  rule   %q → rules/%s.md\n", p.ID, p.ID)
			case "permission":
				entries := resolveAllowEntries(p.Body)
				fmt.Printf("  perm   %q → settings.permissions.allow: %v\n", p.ID, entries)
			}
		}
	}
	_ = baseDir // used for context, not needed in plan output
}

// resolveSourceFiles returns the list of .md files to process.
// If path is a directory, it returns all .md files directly within it (non-recursive).
// If path is a file, it returns a single-element slice containing that file.

// formatIntents returns a human-readable summary of detected intent types.
func formatIntents(intents []bir.FunctionalIntent) string {
	if len(intents) == 0 {
		return "none (fallback: skill)"
	}

	seen := make(map[bir.IntentType]bool)
	var parts []string
	for _, intent := range intents {
		if !seen[intent.Type] {
			seen[intent.Type] = true
			parts = append(parts, string(intent.Type))
		}
	}

	return strings.Join(parts, ", ")
}

// resolveAllowEntries derives Bash permission entries from the primitive body.
// "turbo-all" and generic "turbo" annotations produce broad defaults.
func resolveAllowEntries(body string) []string {
	lower := strings.ToLower(body)
	if strings.Contains(lower, "turbo-all") || strings.Contains(lower, "turbo") {
		return []string{"Bash(git *)", "Bash(go *)"}
	}
	return []string{"Bash(*)"}
}

// mergeImportDirs consolidates multiple platform directories into a single project.xcf.
// When the same resource ID exists in multiple directories, the version with the larger
// file (richer content) is kept and the conflict is logged.
//
//nolint:gocyclo
func mergeImportDirs(providerDirs []platformDirInfo, xcfDest string) error {
	if _, err := os.Stat(xcfDest); err == nil {
		return fmt.Errorf("[project] %s already exists. Remove it first to import", xcfDest)
	}
	if err := checkXcfDirPreexistence(".", "project"); err != nil {
		return err
	}

	projectName := inferProjectName()
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{Name: projectName},
		ResourceScope: ast.ResourceScope{
			Agents:    make(map[string]ast.AgentConfig),
			Skills:    make(map[string]ast.SkillConfig),
			Rules:     make(map[string]ast.RuleConfig),
			Workflows: make(map[string]ast.WorkflowConfig),
			MCP:       make(map[string]ast.MCPConfig),
		},
	}

	importCount := 0
	var warnings []string

	// Track which directory each resource came from for dedup logging
	agentSources := make(map[string]string)
	skillSources := make(map[string]string)
	ruleSources := make(map[string]string)
	workflowSources := make(map[string]string)

	for _, pdi := range providerDirs {
		dir := pdi.dirName
		provider := pdi.platform
		fmt.Printf("  Scanning %s ...\n", dir)

		tmpConfig := &ast.XcaffoldConfig{
			ResourceScope: ast.ResourceScope{
				Agents:    make(map[string]ast.AgentConfig),
				Skills:    make(map[string]ast.SkillConfig),
				Rules:     make(map[string]ast.RuleConfig),
				Workflows: make(map[string]ast.WorkflowConfig),
				MCP:       make(map[string]ast.MCPConfig),
			},
		}

		// Use ProviderImporter instead of legacy extractors
		imp := findImporterByProvider(provider)
		if imp == nil {
			warnings = append(warnings, fmt.Sprintf("no registered importer for provider %q, skipping %s", provider, dir))
			continue
		}
		if err := imp.Import(dir, tmpConfig); err != nil {
			warnings = append(warnings, fmt.Sprintf("%s import: %v", provider, err))
		}

		// Skill subdirs (same pattern as importScope)
		for id := range tmpConfig.Skills {
			skillFile := filepath.Join(dir, "skills", id, "SKILL.md")
			if _, err := os.Stat(skillFile); err == nil {
				refs, scripts, fileAssets, fileExamples, _ := extractSkillSubdirs(skillFile, id, provider, "", &warnings)
				sc := tmpConfig.Skills[id]
				if len(refs) > 0 {
					sc.References = refs
				}
				if len(scripts) > 0 {
					sc.Scripts = scripts
				}
				if len(fileAssets) > 0 {
					sc.Assets = fileAssets
				}
				if len(fileExamples) > 0 {
					sc.Examples = fileExamples
				}
				tmpConfig.Skills[id] = sc
			}
		}

		// Reclassify extras
		if err := parser.ReclassifyExtras(tmpConfig, importer.DefaultImporters()); err != nil {
			warnings = append(warnings, fmt.Sprintf("reclassify extras (%s): %v", provider, err))
		}

		// Provider-specific post-import
		dirAbs, _ := filepath.Abs(dir)
		projectDir := filepath.Dir(dirAbs)
		if err := runProviderPostImport(provider, dir, projectDir, tmpConfig, &warnings); err != nil {
			return err
		}

		// Merge agents — richer instructions win on conflict
		for id, ac := range tmpConfig.Agents {
			if _, exists := config.Agents[id]; exists {
				newSize := int64(len(ac.Body))
				oldSize := int64(len(config.Agents[id].Body))
				if newSize > oldSize {
					config.Agents[id] = ac
					fmt.Printf("    ⚠ Duplicate agent '%s' — keeping %s version (larger)\n", id, dir)
					agentSources[id] = dir
				} else {
					fmt.Printf("    ⚠ Duplicate agent '%s' — keeping %s version (larger)\n", id, agentSources[id])
				}
			} else {
				config.Agents[id] = ac
				agentSources[id] = dir
				importCount++
			}
		}

		// Merge skills
		for id, sc := range tmpConfig.Skills {
			if _, exists := config.Skills[id]; exists {
				newSize := int64(len(sc.Body))
				oldSize := int64(len(config.Skills[id].Body))
				if newSize > oldSize {
					config.Skills[id] = sc
					fmt.Printf("    ⚠ Duplicate skill '%s' — keeping %s version (larger)\n", id, dir)
					skillSources[id] = dir
				} else {
					fmt.Printf("    ⚠ Duplicate skill '%s' — keeping %s version (larger)\n", id, skillSources[id])
				}
			} else {
				config.Skills[id] = sc
				skillSources[id] = dir
				importCount++
			}
		}

		// Merge rules
		for id, rc := range tmpConfig.Rules {
			if _, exists := config.Rules[id]; exists {
				newSize := int64(len(rc.Body))
				oldSize := int64(len(config.Rules[id].Body))
				if newSize > oldSize {
					config.Rules[id] = rc
					fmt.Printf("    ⚠ Duplicate rule '%s' — keeping %s version (larger)\n", id, dir)
					ruleSources[id] = dir
				} else {
					fmt.Printf("    ⚠ Duplicate rule '%s' — keeping %s version (larger)\n", id, ruleSources[id])
				}
			} else {
				config.Rules[id] = rc
				ruleSources[id] = dir
				importCount++
			}
		}

		// Merge workflows
		for id, wc := range tmpConfig.Workflows {
			if _, exists := config.Workflows[id]; exists {
				newSize := int64(len(wc.Body))
				oldSize := int64(len(config.Workflows[id].Body))
				if newSize > oldSize {
					config.Workflows[id] = wc
					fmt.Printf("    ⚠ Duplicate workflow '%s' — keeping %s version (larger)\n", id, dir)
					workflowSources[id] = dir
				} else {
					fmt.Printf("    ⚠ Duplicate workflow '%s' — keeping %s version (larger)\n", id, workflowSources[id])
				}
			} else {
				config.Workflows[id] = wc
				workflowSources[id] = dir
				importCount++
			}
		}

		// Merge MCP servers (no dedup needed — unique server IDs are typical)
		for id, mc := range tmpConfig.MCP {
			if _, exists := config.MCP[id]; !exists {
				config.MCP[id] = mc
				importCount++
			}
		}

		// Merge memory (union — first-seen wins on key collision)
		if tmpConfig.Memory != nil {
			if config.Memory == nil {
				config.Memory = make(map[string]ast.MemoryConfig)
			}
			for k, mc := range tmpConfig.Memory {
				if _, exists := config.Memory[k]; !exists {
					config.Memory[k] = mc
					importCount++
				}
			}
		}

		// Merge hooks (union per event, not overwrite)
		for hookName, namedHook := range tmpConfig.Hooks {
			if config.Hooks == nil {
				config.Hooks = make(map[string]ast.NamedHookConfig)
			}
			if _, exists := config.Hooks[hookName]; !exists {
				config.Hooks[hookName] = namedHook
			}
		}

		// Merge settings (first-seen wins)
		for name, sc := range tmpConfig.Settings {
			if config.Settings == nil {
				config.Settings = make(map[string]ast.SettingsConfig)
			}
			if _, exists := config.Settings[name]; !exists {
				config.Settings[name] = sc
			}
		}
	} // end for loop

	// Detect compilation targets from all scanned platform directories.
	var dirNames []string
	for _, pdi := range providerDirs {
		dirNames = append(dirNames, pdi.dirName)
	}
	if config.Project != nil {
		config.Project.Targets = detectTargets(dirNames...)
		config.Project.AgentRefs = sortedAgentRefs(config.Agents)
		config.Project.SkillRefs = sortedMapKeys(config.Skills)
		config.Project.RuleRefs = sortedMapKeys(config.Rules)
		config.Project.WorkflowRefs = sortedMapKeys(config.Workflows)
		config.Project.MCPRefs = sortedMapKeys(config.MCP)
	}

	// Write memory files to xcf/agents/<id>/memory/
	if memCount, err := writeMemoryFiles(config); err != nil {
		return fmt.Errorf("write memory files: %w", err)
	} else if memCount > 0 {
		fmt.Printf("  Agent memory: %d entry(ies) → xcf/agents/<id>/memory/\n", memCount)
	}

	// Write split .xcf files: project.xcf (kind: project) + xcf/**/*.xcf
	if err := WriteSplitFiles(config, "."); err != nil {
		return fmt.Errorf("[project] failed to write split xcf files: %w", err)
	}

	// Prune orphan memory
	if err := pruneOrphanMemory(config, "."); err != nil {
		return fmt.Errorf("prune memory: %w", err)
	}

	// Root context file discovery
	discoverRootContextFiles(".", config)

	fmt.Printf("\n[project] ✓ Merge complete. Created %s with %d resources from %d directories.\n",
		xcfDest, importCount, len(providerDirs))
	fmt.Printf("  Split xcf/ files written to xcf/ directory.\n")
	fmt.Println("  Run 'xcaffold apply' when ready to compile to your target platforms.")

	cwd, _ := os.Getwd()
	if config.Project != nil {
		_ = registry.Register(cwd, config.Project.Name, nil, ".")
	}

	if len(warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, w := range warnings {
			fmt.Println(" ⚠", w)
		}
	}
	return nil
}

// checkXcfDirPreexistence returns an error if a xcf/ directory already exists
// adjacent to xcfDest. Callers must remove it before re-importing.
func checkXcfDirPreexistence(xcfDest, scopeName string) error {
	xcfSourceDir := filepath.Join(filepath.Dir(xcfDest), "xcf")
	if _, err := os.Stat(xcfSourceDir); err == nil {
		return fmt.Errorf("[%s] xcf/ directory already exists. Remove it first to re-import", scopeName)
	}
	return nil
}

// inferProjectName derives a project name from the current working directory.
func inferProjectName() string {
	wd, err := os.Getwd()
	if err != nil {
		return "imported-project"
	}
	base := filepath.Base(wd)
	if base == "." || base == "/" || base == "" {
		return "imported-project"
	}
	return base
}

// copyFile copies the file at src to dst, creating parent directories as needed.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(dst), err)
	}
	return os.WriteFile(dst, data, 0600)
}

// fileSize returns the size of a file in bytes, or 0 if it can't be read.
func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

// runMemorySnapshot performs the memory import pass for --with-memory.
// When fromPlatform is "gemini" it reads xcaffold-seeded blocks from GEMINI.md
// in the resolved gemini directory. For all other platforms (and "auto") it
// imports from the Claude project memory directory.
//
// NOTE: The --with-memory BIR snapshot path writes flat .md files to
// xcf/agents/ without per-agent subdirectories. This is a known limitation:
// BIR import functions have no agent-to-entry mapping. The primary import
// path (WriteSplitFiles from parsed config.Memory) writes to the correct
// per-agent layout xcf/agents/<id>/memory/. The --with-memory flag is a
// raw snapshot escape hatch and does not produce the canonical layout.
func runMemorySnapshot(cmd *cobra.Command, source string, fromPlatform string, planOnly bool) (*bir.ImportSummary, error) {
	sidecarDir := filepath.Join("xcf", "agents")

	if fromPlatform == "gemini" {
		gDir, err := geminiMemoryDir()
		if err != nil {
			return nil, fmt.Errorf("resolving gemini directory: %w", err)
		}
		return bir.ImportGeminiMemory(gDir, bir.ImportOpts{
			PlanOnly:   planOnly,
			SidecarDir: sidecarDir,
		})
	}

	memDir, err := resolveClaudeMemoryDir(source, fromPlatform)
	if err != nil {
		return nil, err
	}

	return bir.ImportClaudeMemory(memDir, bir.ImportOpts{
		PlanOnly:   planOnly,
		SidecarDir: sidecarDir,
	})
}

// resolveClaudeMemoryDir determines the memory directory to import from.
// If source is a valid directory, it is used directly.
// Otherwise, the function derives ~/.claude/projects/<encoded-cwd>/memory/
// via claudeProjectMemoryDir.
func resolveClaudeMemoryDir(source, fromPlatform string) (string, error) {
	if source != "" {
		info, err := os.Stat(source)
		if err == nil && info.IsDir() {
			return source, nil
		}
	}

	// Derive ~/.claude/projects/<encoded-cwd>/memory/ using the shared helper.
	// Empty string causes claudeProjectMemoryDir to fall back to os.Getwd().
	memDir, err := claudeProjectMemoryDir("")
	if err != nil {
		return "", err
	}

	info, err := os.Stat(memDir)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("claude memory directory not found at %s; pass --source <dir> to specify a location", memDir)
	}

	return memDir, nil
}

// printMemorySnapshotSummary writes the outcome of a memory snapshot pass.
func printMemorySnapshotSummary(cmd *cobra.Command, s *bir.ImportSummary, planOnly bool) {
	out := cmd.OutOrStdout()
	if planOnly {
		fmt.Fprintf(out, "memory snapshot plan\n  would import: %d entries\n", s.WouldImport)
		return
	}
	fmt.Fprintf(out, "memory snapshot complete\n  imported: %d entries\n", s.Imported)
	if s.Skipped > 0 {
		fmt.Fprintf(out, "  skipped (already exists): %d\n", s.Skipped)
	}
	if len(s.Written) > 0 {
		fmt.Fprintln(out, "  written:")
		for _, w := range s.Written {
			fmt.Fprintf(out, "           %s\n", w)
		}
	}
	if s.Imported > 0 {
		fmt.Fprintln(out, "\nrun xcaffold apply to compile these memory entries into the target provider")
	}
}

// detectAndMergeVariants runs the multi-provider divergence algorithm.
// It discovers instruction files for the second provider and compares their
// content with existing scope entries byte-for-byte. Identical content is
// collapsed to a single entry; divergent content populates Variants with
// per-provider sidecar paths and Reconciliation metadata.
// autoMergeUnion concatenates both content blobs into the existing sidecar.
//
// Scopes present only in the secondary provider's tree are intentionally not
// added to the primary config; this function reconciles overlapping paths only.
// New scopes from a secondary provider should be added via a separate import pass.

// parseProvenanceMarkers splits a flat-singleton file into root content and
// individual scope entries using xcaffold:scope HTML comments.
// Returns (scopes, rootContent, error).

// parseHTMLCommentAttrs extracts key="value" pairs from an HTML comment line.
func parseHTMLCommentAttrs(line string) map[string]string {
	attrs := map[string]string{}
	for _, match := range htmlCommentAttrRE.FindAllStringSubmatch(line, -1) {
		attrs[match[1]] = match[2]
	}
	return attrs
}

// findImporterByProvider returns the registered ProviderImporter for the given
// provider name, or nil when no importer is registered for that provider.
func findImporterByProvider(provider string) importer.ProviderImporter {
	for _, imp := range importer.DefaultImporters() {
		if imp.Provider() == provider {
			return imp
		}
	}
	return nil
}

// pruneOrphanMemory removes xcf/agents/<id>/memory/ directories for agents
// that are not present in the current import scope. Agents referenced only via
// config.Memory (e.g. global agents whose project-scoped memory was imported)
// are preserved even when they have no entry in config.Agents.
// After pruning, any now-empty agent directory (no .xcf file, no memory/) is
// also removed.
func pruneOrphanMemory(config *ast.XcaffoldConfig, rootDir string) error {
	agentsDir := filepath.Join(rootDir, "xcf", "agents")
	// If agentsDir doesn't exist, nothing to prune.
	if _, err := os.Stat(agentsDir); os.IsNotExist(err) {
		return nil
	}

	validAgents := make(map[string]bool)
	for id := range config.Agents {
		validAgents[id] = true
	}

	// Build a set of agents that have explicitly imported memory entries.
	// These are preserved even if they have no agent definition (e.g. global
	// agents like ~/.claude/agents/ceo.md whose project-scoped memory was
	// imported from .claude/agent-memory/ceo/).
	memoryAgents := make(map[string]bool)
	for memPath := range config.Memory {
		agentID := strings.SplitN(filepath.ToSlash(memPath), "/", 2)[0]
		memoryAgents[agentID] = true
	}

	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		agentID := entry.Name()
		memDir := filepath.Join(agentsDir, agentID, "memory")
		if _, err := os.Stat(memDir); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		// Prune the memory dir only when the agent is absent from both the
		// declared agents and the explicitly imported memory entries.
		if !validAgents[agentID] && !memoryAgents[agentID] {
			if err := os.RemoveAll(memDir); err != nil {
				return err
			}
		}
	}

	// Remove any agent directories that are now empty (no .xcf file and no
	// memory/ subdirectory). These are artifacts left when the memory dir was
	// pruned above or was never populated.
	entries, err = os.ReadDir(agentsDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		agentDir := filepath.Join(agentsDir, entry.Name())
		dirEntries, err := os.ReadDir(agentDir)
		if err == nil && len(dirEntries) == 0 {
			_ = os.Remove(agentDir)
		}
	}

	return nil
}

// claudeProjectMemoryDir returns the Claude project memory directory for a
// given project root: ~/.claude/projects/<encoded-projectRoot>/memory/.
//
// Path encoding follows Claude's own convention: forward slashes are replaced
// with hyphens. If projectRoot is empty or ".", os.Getwd() is used as a fallback.
func claudeProjectMemoryDir(projectRoot string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	if projectRoot == "" || projectRoot == "." {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolving working directory: %w", err)
		}
		projectRoot = cwd
	}
	projectRoot = filepath.Clean(projectRoot)
	if !filepath.IsAbs(projectRoot) {
		abs, err := filepath.Abs(projectRoot)
		if err != nil {
			return "", fmt.Errorf("resolving absolute project root: %w", err)
		}
		projectRoot = abs
	}
	encoded := strings.ReplaceAll(projectRoot, "/", "-")
	return filepath.Join(home, ".claude", "projects", encoded, "memory"), nil
}

// writeMemoryFiles writes each memory entry in config to a plain .md file under
// xcf/agents/<agentID>/memory/<name>.md, mirroring the convention the compiler
// uses to discover memory at build time. Returns the number of files written.
func writeMemoryFiles(config *ast.XcaffoldConfig) (int, error) {
	if len(config.Memory) == 0 {
		return 0, nil
	}
	count := 0
	for _, k := range sortedMapKeys(config.Memory) {
		mem := config.Memory[k]
		parts := strings.SplitN(filepath.ToSlash(k), "/", 2)
		var agentID, memName string
		if len(parts) == 2 {
			agentID = parts[0]
			memName = parts[1]
		} else {
			agentID = mem.AgentRef
			if agentID == "" {
				agentID = k
			}
			memName = k
		}
		outPath := filepath.Join("xcf", "agents", agentID, "memory", filepath.FromSlash(memName)+".md")
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return count, fmt.Errorf("create memory dir: %w", err)
		}
		if err := os.WriteFile(filepath.Clean(outPath), []byte(mem.Content), 0644); err != nil {
			return count, fmt.Errorf("write memory file: %w", err)
		}
		count++
	}
	return count, nil
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

// runProviderPostImport executes provider-specific post-import steps that fall
// outside the scope of the ProviderImporter interface (cross-boundary files,
// out-of-tree memory sources, unsupported-provider warnings).
func runProviderPostImport(provider, _ /* platformDir */ string, projectDir string, config *ast.XcaffoldConfig, warnings *[]string) error {
	// Claude: root .mcp.json lives outside .claude/ — import it here.
	if provider == "claude" || provider == "" {
		rootMCPPath := filepath.Join(projectDir, ".mcp.json")
		if data, err := os.ReadFile(rootMCPPath); err == nil {
			count := 0
			if err := importSettings(data, config, &count, warnings); err != nil {
				*warnings = append(*warnings, fmt.Sprintf(".mcp.json partially imported: %v", err))
			}
		}
	}
	// Gemini: snapshot memory from ~/.gemini/.
	if provider == "gemini" {
		if gDir, err := geminiMemoryDir(); err == nil {
			if memSum, err := bir.ImportGeminiMemory(gDir, bir.ImportOpts{
				SidecarDir: filepath.Join("xcf", "agents"),
			}); err == nil && memSum.Imported > 0 {
				fmt.Printf("  Gemini memory: snapshotted %d entry(ies) → xcf/agents/<id>/memory/\n", memSum.Imported)
			}
		}
	}
	// Antigravity: KIs are app-managed and cannot be imported from the filesystem.
	if provider == "antigravity" {
		*warnings = append(*warnings,
			"Antigravity Knowledge Items (KIs) are app-managed and cannot be imported from the filesystem")
	}
	return nil
}

// discoverRootContextFiles scans the project root for known root context files
// and populates config.Contexts.
func discoverRootContextFiles(projectDir string, config *ast.XcaffoldConfig) {
	if config.Contexts == nil {
		config.Contexts = make(map[string]ast.ContextConfig)
	}

	files := []struct {
		path   string
		name   string
		target string
	}{
		{"CLAUDE.md", "claude", "claude"},
		{"GEMINI.md", "gemini", "gemini"},
		{".github/copilot-instructions.md", "copilot", "copilot"},
	}

	for _, f := range files {
		fullPath := filepath.Join(projectDir, filepath.FromSlash(f.path))
		if data, err := os.ReadFile(fullPath); err == nil {
			config.Contexts[f.name] = ast.ContextConfig{
				Name:    f.name,
				Targets: []string{f.target},
				Body:    string(data),
			}
		}
	}

	// Handle AGENTS.md which is shared by Cursor and Antigravity
	if data, err := os.ReadFile(filepath.Join(projectDir, "AGENTS.md")); err == nil {
		name := "antigravity"
		target := "antigravity"
		if _, err := os.Stat(filepath.Join(projectDir, ".cursor")); err == nil {
			name = "cursor"
			target = "cursor"
		}
		config.Contexts[name] = ast.ContextConfig{
			Name:    name,
			Targets: []string{target},
			Body:    string(data),
		}
	}
}
