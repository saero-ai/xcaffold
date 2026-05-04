package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/importer"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/registry"
	"github.com/saero-ai/xcaffold/providers"
	_ "github.com/saero-ai/xcaffold/providers/antigravity"
	_ "github.com/saero-ai/xcaffold/providers/claude"
	_ "github.com/saero-ai/xcaffold/providers/copilot"
	_ "github.com/saero-ai/xcaffold/providers/cursor"
	_ "github.com/saero-ai/xcaffold/providers/gemini"
	"github.com/spf13/cobra"
)

var (
	importPlan       bool
	importTargetFlag string

	importFilterAgent    string
	importFilterSkill    string
	importFilterRule     string
	importFilterWorkflow string
	importFilterMCP      string
	importFilterHook     bool
	importFilterSetting  bool
	importFilterMemory   bool
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import existing provider config into project.xcf",
	Long: `xcaffold import adopts existing provider configurations into xcaffold.

Detection (Default):
 • Scans .claude/agents/*.md   → extracts to xcf/agents/<id>.md
 • Scans .claude/skills/*/SKILL.md → extracts to xcf/skills/<id>/SKILL.md
 • Scans .claude/rules/*.md    → extracts to xcf/rules/<id>.md
 • Reads .claude/settings.json for MCP and settings context
 • Generates project.xcf manifest referencing discovered resources

Usage:
  $ xcaffold import
  $ xcaffold import --target claude
  $ xcaffold import --plan`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil
		}
		return fmt.Errorf("unexpected argument %q (to filter by name, use --flag=%s syntax)", args[0], args[0])
	},
	RunE: runImport,
}

func init() {
	f := importCmd.Flags()
	f.BoolVar(&importPlan, "plan", false, "Dry-run: print import plan without writing files")
	f.StringVar(&importTargetFlag, "target", "", "Import from specific provider: claude, gemini, cursor, antigravity, copilot")

	// Per-kind resource filtering flags
	f.StringVar(&importFilterAgent, "agent", "", "Import agents (optionally filter by name)")
	f.Lookup("agent").NoOptDefVal = "*"

	f.StringVar(&importFilterSkill, "skill", "", "Import skills (optionally filter by name)")
	f.Lookup("skill").NoOptDefVal = "*"

	f.StringVar(&importFilterRule, "rule", "", "Import rules (optionally filter by name)")
	f.Lookup("rule").NoOptDefVal = "*"

	f.StringVar(&importFilterWorkflow, "workflow", "", "Import workflows (optionally filter by name)")
	f.Lookup("workflow").NoOptDefVal = "*"

	f.StringVar(&importFilterMCP, "mcp", "", "Import MCP servers (optionally filter by name)")
	f.Lookup("mcp").NoOptDefVal = "*"

	f.BoolVar(&importFilterHook, "hook", false, "Import hooks")
	f.BoolVar(&importFilterSetting, "setting", false, "Import settings")
	f.BoolVar(&importFilterMemory, "memory", false, "Import memory")

	rootCmd.AddCommand(importCmd)
}

// applyKindFilters filters the config to include only the resource kinds specified by flags.
// When no filter flags are set, all resources are preserved. When any filter is set, only
// the specified kinds are kept. Named filters (--agent <name>) narrow to a single resource.
func applyKindFilters(config *ast.XcaffoldConfig) {
	// Check if any filter is set by examining the filter variables
	anyFilterSet := importFilterAgent != "" || importFilterSkill != "" ||
		importFilterRule != "" || importFilterWorkflow != "" ||
		importFilterMCP != "" || importFilterHook ||
		importFilterSetting || importFilterMemory

	if !anyFilterSet {
		return
	}

	// Zero out kinds not requested
	if importFilterAgent == "" {
		config.Agents = nil
	}
	if importFilterSkill == "" {
		config.Skills = nil
	}
	if importFilterRule == "" {
		config.Rules = nil
	}
	if importFilterWorkflow == "" {
		config.Workflows = nil
	}
	if importFilterMCP == "" {
		config.MCP = nil
	}
	if !importFilterHook {
		config.Hooks = nil
	}
	if !importFilterSetting {
		config.Settings = nil
	}
	if !importFilterMemory {
		config.Memory = nil
	}

	// Name filters: narrow to specific resource
	if importFilterAgent != "" && importFilterAgent != "*" && config.Agents != nil {
		if agent, ok := config.Agents[importFilterAgent]; ok {
			config.Agents = map[string]ast.AgentConfig{importFilterAgent: agent}
		} else {
			config.Agents = nil
		}
	}
	if importFilterSkill != "" && importFilterSkill != "*" && config.Skills != nil {
		if skill, ok := config.Skills[importFilterSkill]; ok {
			config.Skills = map[string]ast.SkillConfig{importFilterSkill: skill}
		} else {
			config.Skills = nil
		}
	}
	if importFilterRule != "" && importFilterRule != "*" && config.Rules != nil {
		if rule, ok := config.Rules[importFilterRule]; ok {
			config.Rules = map[string]ast.RuleConfig{importFilterRule: rule}
		} else {
			config.Rules = nil
		}
	}
	if importFilterWorkflow != "" && importFilterWorkflow != "*" && config.Workflows != nil {
		if wf, ok := config.Workflows[importFilterWorkflow]; ok {
			config.Workflows = map[string]ast.WorkflowConfig{importFilterWorkflow: wf}
		} else {
			config.Workflows = nil
		}
	}
	if importFilterMCP != "" && importFilterMCP != "*" && config.MCP != nil {
		if mcp, ok := config.MCP[importFilterMCP]; ok {
			config.MCP = map[string]ast.MCPConfig{importFilterMCP: mcp}
		} else {
			config.MCP = nil
		}
	}
}

func sortedProviderNames(providers []importer.ProviderImporter) []string {
	names := make([]string, 0, len(providers))
	for _, imp := range providers {
		names = append(names, imp.Provider())
	}
	sort.Strings(names)
	return names
}

func tagResourcesWithProvider(config *ast.XcaffoldConfig, provider string) {
	to := ast.TargetOverride{}
	for name, agent := range config.Agents {
		if agent.Targets == nil {
			agent.Targets = make(map[string]ast.TargetOverride)
		}
		agent.Targets[provider] = to
		config.Agents[name] = agent
	}
	for name, skill := range config.Skills {
		if skill.Targets == nil {
			skill.Targets = make(map[string]ast.TargetOverride)
		}
		skill.Targets[provider] = to
		config.Skills[name] = skill
	}
	for name, rule := range config.Rules {
		if rule.Targets == nil {
			rule.Targets = make(map[string]ast.TargetOverride)
		}
		rule.Targets[provider] = to
		config.Rules[name] = rule
	}
	for name, wf := range config.Workflows {
		if wf.Targets == nil {
			wf.Targets = make(map[string]ast.TargetOverride)
		}
		wf.Targets[provider] = to
		config.Workflows[name] = wf
	}
}

func runImport(cmd *cobra.Command, args []string) error {
	if globalFlag {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		globalDetected := importer.DetectProviders(home, importer.DefaultImporters())
		if len(globalDetected) == 0 {
			return fmt.Errorf("no global platform directories found (~/.claude/, ~/.cursor/, ~/.agents/)")
		}
		if len(globalDetected) > 1 {
			return mergeImportDirs(globalDetected, globalXcfPath)
		}
		return importScope(globalDetected[0].InputDir(), globalXcfPath, "global", globalDetected[0].Provider())
	}

	// Validate --target if set
	if importTargetFlag != "" {
		validTargets := map[string]bool{
			"claude":      true,
			"gemini":      true,
			"cursor":      true,
			"antigravity": true,
			"copilot":     true,
		}
		if !validTargets[importTargetFlag] {
			return fmt.Errorf("unknown target %q; valid targets: claude, gemini, cursor, antigravity, copilot", importTargetFlag)
		}
	}

	// project (default) — detect providers via ProviderImporter registry.
	detected := importer.DetectProviders(".", importer.DefaultImporters())

	// Filter to specific provider if --target is set
	if importTargetFlag != "" {
		var filtered []importer.ProviderImporter
		for _, imp := range detected {
			if imp.Provider() == importTargetFlag {
				filtered = append(filtered, imp)
			}
		}
		detected = filtered
	}

	if len(detected) > 1 {
		return mergeImportDirs(detected, "project.xcf")
	}
	if len(detected) == 1 {
		imp := detected[0]
		return importScope(imp.InputDir(), "project.xcf", "project", imp.Provider())
	}

	return fmt.Errorf("no supported AI provider configuration found in current directory. Supported providers: Claude Code, Gemini CLI, Cursor, GitHub Copilot, Antigravity")
}

// importScope scans a platform directory and writes a xcf file to xcfDest.
// provider selects provider-specific extraction logic for settings, MCP,
// hooks, project-instruction files, and memory. Supported values: "claude",
// "gemini", "cursor", "copilot", "antigravity". An empty string or unknown
// value falls back to Claude-style extraction (settings.json + hooks.json).
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

	var warnings []string

	// Extract resources and run post-import steps
	extractAndPostProcess(platformDir, provider, config, &warnings)

	// Provider-specific post-import steps
	if err := runProviderPostImport(provider, platformDir, projectDir, config, &warnings); err != nil {
		return err
	}

	// Shared post-import pipeline: memory, context discovery, orphan pruning
	if err := runPostImportSteps(config, projectDir, false); err != nil {
		return err
	}

	// Detect compilation targets
	if config.Project != nil {
		config.Project.Targets = detectTargets(platformDir)
	}

	return finalizeImportScope(xcfDest, scopeName, provider, config, &warnings)
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
		"examples":   "examples",
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
// xcf/skills/<id>/<subdir>/ alongside canonical subdirectories.
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

	// Helper: copy a file to the skill directory with unknown subdirs.
	appendPassthrough := func(src, subdir, filename string) {
		var dest string
		if base != "" {
			dest = filepath.Join(base, "xcf", "skills", id, subdir, filename)
		} else {
			dest = filepath.Join("xcf", "skills", id, subdir, filename)
		}
		if copyErr := copyFile(src, dest); copyErr != nil {
			*warnings = append(*warnings, fmt.Sprintf("failed to copy skill file %s: %v", src, copyErr))
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

// finalizeImportScope handles memory file writing, resource tagging, filtering, and success messages.
func finalizeImportScope(xcfDest, scopeName, provider string, config *ast.XcaffoldConfig, warnings *[]string) error {
	tagResourcesWithProvider(config, provider)
	applyKindFilters(config)

	if importPlan {
		fmt.Printf("Import plan (dry-run):\n")
		fmt.Printf("  Would create %d agents, %d skills, %d rules, %d workflows, %d MCP servers\n",
			len(config.Agents), len(config.Skills), len(config.Rules), len(config.Workflows), len(config.MCP))
		fmt.Printf("  Target directory: %s\n", xcfDest)
		return nil
	}

	if err := WriteSplitFiles(config, "."); err != nil {
		return fmt.Errorf("[%s] failed to write split xcf files: %w", scopeName, err)
	}

	importCount := len(config.Agents) + len(config.Skills) + len(config.Rules) +
		len(config.Workflows) + len(config.MCP)
	fmt.Printf("[%s] ✓ Import complete. Created %s with %d resources.\n", scopeName, xcfDest, importCount)
	fmt.Printf("  Split xcf/ files written to xcf/ directory.\n")
	fmt.Printf("  Resources tagged with targets: [%s]. Remove the targets field to make universal.\n", provider)
	fmt.Println("  Run 'xcaffold apply' when ready to assume management.")

	cwd, _ := os.Getwd()
	if config.Project != nil {
		_ = registry.Register(cwd, config.Project.Name, nil, ".")
	}

	if len(*warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, w := range *warnings {
			fmt.Println(" ⚠", w)
		}
	}
	return nil
}

// extractAndPostProcess handles resource extraction and post-import steps for a single provider.
// It returns the number of resources extracted and mutates the config and warnings in place.
func extractAndPostProcess(platformDir, provider string, config *ast.XcaffoldConfig, warnings *[]string) int {
	importCount := 0

	providerImp := findImporterByProvider(provider)
	if providerImp != nil {
		if err := providerImp.Import(platformDir, config); err != nil {
			*warnings = append(*warnings, fmt.Sprintf("%s import: %v", provider, err))
		}
		// Surface per-file extraction warnings from importers that support it.
		type warningImporter interface {
			GetWarnings() []string
		}
		if wi, ok := providerImp.(warningImporter); ok {
			for _, w := range wi.GetWarnings() {
				*warnings = append(*warnings, fmt.Sprintf("%s: %s", provider, w))
			}
		}
		// Copy skill supporting files
		for id := range config.Skills {
			skillFile := filepath.Join(platformDir, "skills", id, "SKILL.md")
			if _, err := os.Stat(skillFile); err == nil {
				refs, scripts, fileAssets, fileExamples, _ := extractSkillSubdirs(skillFile, id, provider, "", warnings)
				sc := config.Skills[id]
				if len(refs) > 0 {
					sc.References = ast.ClearableList{Values: refs}
				}
				if len(scripts) > 0 {
					sc.Scripts = ast.ClearableList{Values: scripts}
				}
				if len(fileAssets) > 0 {
					sc.Assets = ast.ClearableList{Values: fileAssets}
				}
				if len(fileExamples) > 0 {
					sc.Examples = ast.ClearableList{Values: fileExamples}
				}
				config.Skills[id] = sc
			}
		}
		// Attempt to graduate extras
		if err := parser.ReclassifyExtras(config, importer.DefaultImporters()); err != nil {
			*warnings = append(*warnings, fmt.Sprintf("reclassify extras: %v", err))
		}

		importCount = len(config.Agents) + len(config.Skills) + len(config.Rules) +
			len(config.Workflows) + len(config.MCP)
	}

	return importCount
}

// scanProviderConfigs scans each provider via ProviderImporter and populates a map of provider -> XcaffoldConfig.
// It handles importer invocation, skill subdirectory extraction, extras reclassification, and post-import steps.
func scanProviderConfigs(providers []importer.ProviderImporter, warnings *[]string) map[string]*ast.XcaffoldConfig {
	providerConfigs := make(map[string]*ast.XcaffoldConfig)

	for _, imp := range providers {
		dir := imp.InputDir()
		provider := imp.Provider()
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

		if err := imp.Import(dir, tmpConfig); err != nil {
			*warnings = append(*warnings, fmt.Sprintf("%s import: %v", provider, err))
		}

		for id := range tmpConfig.Skills {
			skillFile := filepath.Join(dir, "skills", id, "SKILL.md")
			if _, err := os.Stat(skillFile); err == nil {
				refs, scripts, fileAssets, fileExamples, _ := extractSkillSubdirs(skillFile, id, provider, "", warnings)
				sc := tmpConfig.Skills[id]
				if len(refs) > 0 {
					sc.References = ast.ClearableList{Values: refs}
				}
				if len(scripts) > 0 {
					sc.Scripts = ast.ClearableList{Values: scripts}
				}
				if len(fileAssets) > 0 {
					sc.Assets = ast.ClearableList{Values: fileAssets}
				}
				if len(fileExamples) > 0 {
					sc.Examples = ast.ClearableList{Values: fileExamples}
				}
				tmpConfig.Skills[id] = sc
			}
		}

		if err := parser.ReclassifyExtras(tmpConfig, importer.DefaultImporters()); err != nil {
			*warnings = append(*warnings, fmt.Sprintf("reclassify extras (%s): %v", provider, err))
		}

		dirAbs, _ := filepath.Abs(dir)
		projectDir := filepath.Dir(dirAbs)
		if err := runProviderPostImport(provider, dir, projectDir, tmpConfig, warnings); err != nil {
			// Note: caller must handle this error
			*warnings = append(*warnings, fmt.Sprintf("post-import error: %v", err))
			continue
		}

		providerConfigs[provider] = tmpConfig
	}

	return providerConfigs
}

// mergeImportDirs consolidates multiple platform directories into a single project.xcf.
// Resources present in multiple providers are compared field-by-field: identical content
// produces a universal base tagged with all providers; different content produces a base
// with the first provider's values plus per-provider override files.
func mergeImportDirs(providers []importer.ProviderImporter, xcfDest string) error {
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

	var warnings []string

	// Collect per-provider configs
	providerConfigs := scanProviderConfigs(providers, &warnings)

	assembleMultiProviderResources(providerConfigs, config)

	// Detect compilation targets from all scanned platform directories.
	var dirNames []string
	for _, imp := range providers {
		dirNames = append(dirNames, imp.InputDir())
	}
	if config.Project != nil {
		config.Project.Targets = detectTargets(dirNames...)
	}

	applyKindFilters(config)

	// Apply --plan guard before writing files
	if importPlan {
		fmt.Printf("Import plan (dry-run):\n")
		fmt.Printf("  Would create %d agents, %d skills, %d rules, %d workflows, %d MCP servers\n",
			len(config.Agents), len(config.Skills), len(config.Rules), len(config.Workflows), len(config.MCP))
		fmt.Printf("  From %d provider directories\n", len(providers))
		return nil
	}

	if memCount, err := writeMemoryFiles(config); err != nil {
		return fmt.Errorf("write memory files: %w", err)
	} else if memCount > 0 {
		fmt.Printf("  Agent memory: %d entry(ies) → xcf/agents/<id>/memory/\n", memCount)
	}

	discoverRootContextFiles(".", config)

	if err := WriteSplitFiles(config, "."); err != nil {
		return fmt.Errorf("[project] failed to write split xcf files: %w", err)
	}

	if err := pruneOrphanMemory(config, "."); err != nil {
		return fmt.Errorf("prune memory: %w", err)
	}

	importCount := len(config.Agents) + len(config.Skills) + len(config.Rules) +
		len(config.Workflows) + len(config.MCP)
	overrideCount := 0
	if config.Overrides != nil {
		for name := range config.Agents {
			overrideCount += len(config.Overrides.AgentProviders(name))
		}
		for name := range config.Skills {
			overrideCount += len(config.Overrides.SkillProviders(name))
		}
		for name := range config.Rules {
			overrideCount += len(config.Overrides.RuleProviders(name))
		}
		for name := range config.Workflows {
			overrideCount += len(config.Overrides.WorkflowProviders(name))
		}
	}
	fmt.Printf("\n[project] ✓ Import complete. Created %s with %d resources from %d directories.\n",
		xcfDest, importCount, len(providers))
	fmt.Printf("  Split xcf/ files written to xcf/ directory.\n")
	fmt.Printf("  Resources tagged with targets: [%s].\n", strings.Join(sortedProviderNames(providers), ", "))
	if overrideCount > 0 {
		fmt.Printf("  %d conflicts detected — override files created. Run 'xcaffold validate' to review.\n", overrideCount)
	}
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

// runPostImportSteps executes the shared post-import steps: memory writing,
// root context discovery, and memory pruning. The injectToolkit parameter
// controls whether toolkit reference templates and xaff toolkit are injected
// (typically false for import, true for init).
func runPostImportSteps(config *ast.XcaffoldConfig, projectDir string, injectToolkit bool) error {
	if memCount, err := writeMemoryFiles(config); err != nil {
		return fmt.Errorf("write memory: %w", err)
	} else if memCount > 0 {
		fmt.Printf("  Agent memory: %d entry(ies) → xcf/agents/<id>/memory/\n", memCount)
	}

	discoverRootContextFiles(projectDir, config)

	if err := pruneOrphanMemory(config, projectDir); err != nil {
		return fmt.Errorf("prune memory: %w", err)
	}

	if injectToolkit {
		_ = writeReferenceTemplates(projectDir)
		if err := injectXaffToolkitAfterImport(projectDir); err != nil {
			return err
		}
	}
	return nil
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
	// Antigravity: KIs are app-managed and cannot be imported from the filesystem.
	if provider == "antigravity" {
		*warnings = append(*warnings,
			"Antigravity Knowledge Items (KIs) are app-managed and cannot be imported from the filesystem")
	}
	return nil
}

// discoverRootContextFiles scans the project root for known root context files
// using the provider manifest registry and populates config.Contexts.
func discoverRootContextFiles(projectDir string, config *ast.XcaffoldConfig) {
	if config.Contexts == nil {
		config.Contexts = make(map[string]ast.ContextConfig)
	}

	for _, m := range providers.Manifests() {
		if m.RootContextFile == "" {
			continue
		}
		fullPath := filepath.Join(projectDir, filepath.FromSlash(m.RootContextFile))
		if data, err := os.ReadFile(fullPath); err == nil {
			config.Contexts[m.Name] = ast.ContextConfig{
				Name:    m.Name,
				Targets: []string{m.Name},
				Body:    string(data),
			}
		}
	}
}
