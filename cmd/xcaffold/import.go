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
	"github.com/saero-ai/xcaffold/internal/bir"
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
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Migrate an existing directory or translate cross-platform workflows into scaffold.xcf",
	Long: `xcaffold import manages adopting existing configurations into xcaffold.

┌───────────────────────────────────────────────────────────────────┐
│                          IMPORT PHASE                             │
└───────────────────────────────────────────────────────────────────┘
Native Import Mode (Default):
 • Scans .claude/agents/*.md   → extracts to agents/<id>.md
 • Scans .claude/skills/*/SKILL.md → extracts to skills/<id>/SKILL.md
 • Scans .claude/rules/*.md    → extracts to rules/<id>.md
 • Reads .claude/settings.json for MCP and settings context
 • Generates a scaffold.xcf with instructions_file: references

Cross-Platform Translation Mode (--source):
 • Imports agent workflow files from other platforms and decomposes
   them into xcaffold primitives (skills, rules, permissions).
 • Detected intents determine primitive mappings.
 • Results are injected into scaffold.xcf using instructions_file: references.
 • Use --plan to preview the decomposition without writing any files.

Usage:
  $ xcaffold import
  $ xcaffold import --source ./workflows/ --from antigravity
  $ xcaffold import --source .cursor/rules/ --from cursor --plan`,
	RunE: runImport,
}

func init() {
	importCmd.Flags().StringVar(&importSource, "source", "", "File or directory of workflow markdown files to translate")
	importCmd.Flags().StringVar(&importFromPlatform, "from", "auto", "Source platform of input files (antigravity, cursor, etc.)")
	importCmd.Flags().BoolVar(&importPlan, "plan", false, "Dry-run: print decomposition plan without writing files")
	rootCmd.AddCommand(importCmd)
}

func runImport(cmd *cobra.Command, args []string) error {
	if importSource != "" {
		return runTranslateMode()
	}

	if globalFlag {
		return importScope(globalXcfHome, globalXcfPath, "global")
	}

	// project (default) — merge all detected directories
	infos := detectAllPlatformDirs(".")
	if len(infos) > 1 {
		var dirs []string
		for _, info := range infos {
			dirs = append(dirs, info.dirName)
		}
		return mergeImportDirs(dirs, "scaffold.xcf")
	}
	if len(infos) == 1 {
		return importScope(infos[0].dirName, "scaffold.xcf", "project")
	}

	// default fallback
	return importScope(".claude", "scaffold.xcf", "project")
}

func runTranslateMode() error {
	xcfPath := "scaffold.xcf"
	config, err := parser.ParseFile(xcfPath)
	if err != nil {
		return fmt.Errorf("no scaffold.xcf found — run 'xcaffold init' first, then 'xcaffold import --source': %w", err)
	}

	xcfAbs, err := filepath.Abs(xcfPath)
	if err != nil {
		return fmt.Errorf("could not resolve scaffold.xcf path: %w", err)
	}
	baseDir := filepath.Dir(xcfAbs)

	sources, err := resolveSourceFiles(importSource)
	if err != nil {
		return fmt.Errorf("--source %q: %w", importSource, err)
	}

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
func importScope(claudeDir, xcfDest, scopeName string) error {
	if _, err := os.Stat(xcfDest); err == nil {
		return fmt.Errorf("[%s] %s already exists. Remove it first to import", scopeName, xcfDest)
	}
	if err := checkXcfDirPreexistence(xcfDest, scopeName); err != nil {
		return err
	}

	projectName := inferProjectName()
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{Name: projectName},
		ResourceScope: ast.ResourceScope{
			Agents: make(map[string]ast.AgentConfig),
			Skills: make(map[string]ast.SkillConfig),
			Rules:  make(map[string]ast.RuleConfig),
			Hooks:  make(ast.HookConfig),
			MCP:    make(map[string]ast.MCPConfig),
		},
	}

	importCount := 0
	var warnings []string

	if err := extractAgents(claudeDir, scopeName, config, &importCount, &warnings); err != nil {
		return err
	}
	if err := extractSkills(claudeDir, scopeName, config, &importCount, &warnings); err != nil {
		return err
	}
	if err := extractRules(claudeDir, scopeName, config, &importCount, &warnings); err != nil {
		return err
	}
	if err := extractWorkflows(claudeDir, scopeName, config, &importCount, &warnings); err != nil {
		return err
	}

	// 4. Parse settings.json for MCP servers and settings.
	settingsPath := filepath.Join(claudeDir, "settings.json")
	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := importSettings(data, config, &importCount, &warnings); err != nil {
			warnings = append(warnings, fmt.Sprintf("settings.json partially imported: %v", err))
		}
	}

	// 5. Parse hooks.json
	hooksPath := filepath.Join(claudeDir, "hooks.json")
	if data, err := os.ReadFile(hooksPath); err == nil {
		if err := json.Unmarshal(data, &config.Hooks); err != nil {
			warnings = append(warnings, fmt.Sprintf("hooks.json failed to parse: %v", err))
		} else {
			importCount++
		}
	}

	// Detect compilation targets from the scanned platform directory.
	if config.Project != nil {
		config.Project.Targets = detectTargets(claudeDir)
		config.Project.AgentRefs = sortedMapKeysStr(config.Agents)
		config.Project.SkillRefs = sortedMapKeysStr(config.Skills)
		config.Project.RuleRefs = sortedMapKeysStr(config.Rules)
		config.Project.WorkflowRefs = sortedMapKeysStr(config.Workflows)
		config.Project.MCPRefs = sortedMapKeysStr(config.MCP)
	}

	// Write split .xcf files: scaffold.xcf (kind: project) + xcf/**/*.xcf
	if err := WriteSplitFiles(config, "."); err != nil {
		return fmt.Errorf("[%s] failed to write split xcf files: %w", scopeName, err)
	}

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
			if err := json.Unmarshal(hooksBytes, &config.Hooks); err == nil {
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
		config.Settings = settings
	}
}

func extractAgents(claudeDir, scopeName string, config *ast.XcaffoldConfig, count *int, warnings *[]string) error {
	agentFiles, _ := filepath.Glob(filepath.Join(claudeDir, "agents", "*.md"))
	for _, f := range agentFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			*warnings = append(*warnings, fmt.Sprintf("skipping agent %s: %v", f, err))
			continue
		}
		id := strings.TrimSuffix(filepath.Base(f), ".md")
		if id == "" {
			continue
		}

		body := extractBodyAfterFrontmatter(data)

		agentCfg := ast.AgentConfig{Description: "Imported agent"}
		if fm, ok := extractFrontmatter(data); ok {
			_ = yaml.Unmarshal(fm, &agentCfg)
			agentCfg.InstructionsFile = ""
		}
		agentCfg.Instructions = body

		config.Agents[id] = agentCfg
		*count++
	}
	return nil
}

func extractSkills(claudeDir, scopeName string, config *ast.XcaffoldConfig, count *int, warnings *[]string) error {
	skillFiles, _ := filepath.Glob(filepath.Join(claudeDir, "skills", "*", "SKILL.md"))
	for _, f := range skillFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			*warnings = append(*warnings, fmt.Sprintf("skipping skill %s: %v", f, err))
			continue
		}
		id := filepath.Base(filepath.Dir(f))
		if id == "" || id == "." {
			continue
		}

		body := extractBodyAfterFrontmatter(data)

		// Reference files (non-.md data files) are still copied to xcf/skills/<id>/references/
		refs, err := extractSkillRefs(f, scopeName, id, warnings)
		if err != nil {
			return err
		}

		skillCfg := ast.SkillConfig{Description: "Imported skill", References: refs}
		if fm, ok := extractFrontmatter(data); ok {
			_ = yaml.Unmarshal(fm, &skillCfg)
			skillCfg.InstructionsFile = ""
			if len(refs) > 0 {
				skillCfg.References = refs // use copied refs, not frontmatter refs
			}
		}
		skillCfg.Instructions = body

		config.Skills[id] = skillCfg
		*count++
	}
	return nil
}

func extractSkillRefs(skillFile, scopeName, id string, warnings *[]string) ([]string, error) {
	refSrc := filepath.Join(filepath.Dir(skillFile), "references")
	var refs []string
	if refEntries, err := os.ReadDir(refSrc); err == nil {
		for _, entry := range refEntries {
			if entry.IsDir() {
				continue
			}
			srcRef := filepath.Join(refSrc, entry.Name())
			xcfRefDest := filepath.Join("xcf", "skills", id, "references", entry.Name())
			if err := copyFile(srcRef, xcfRefDest); err != nil {
				*warnings = append(*warnings, fmt.Sprintf("failed to copy skill ref %s: %v", srcRef, err))
				continue
			}
			refs = append(refs, filepath.ToSlash(xcfRefDest))
		}
	}
	return refs, nil
}

func extractRules(claudeDir, scopeName string, config *ast.XcaffoldConfig, count *int, warnings *[]string) error {
	ruleFiles, _ := filepath.Glob(filepath.Join(claudeDir, "rules", "*.md"))
	for _, f := range ruleFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			*warnings = append(*warnings, fmt.Sprintf("skipping rule %s: %v", f, err))
			continue
		}
		id := strings.TrimSuffix(filepath.Base(f), ".md")
		if id == "" {
			continue
		}

		body := extractBodyAfterFrontmatter(data)

		ruleCfg := ast.RuleConfig{Description: "Imported rule"}
		if fm, ok := extractFrontmatter(data); ok {
			_ = yaml.Unmarshal(fm, &ruleCfg)
			ruleCfg.InstructionsFile = ""
		}
		ruleCfg.Instructions = body

		config.Rules[id] = ruleCfg
		*count++
	}
	return nil
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
		}
	}
	targets := make([]string, 0, len(targetMap))
	for t := range targetMap {
		targets = append(targets, t)
	}
	sort.Strings(targets)
	return targets
}

// sortedMapKeysStr returns sorted keys from any string-keyed map. Used to build
// ref lists for the kind: project document.
func sortedMapKeysStr[V any](m map[string]V) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
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

// injectIntoConfig writes external .md files for each primitive and updates
// scaffold.xcf with instructions_file: references, following the import.go pattern.
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

	header := "# scaffold.xcf — updated by 'xcaffold import --source'"
	out, err := MarshalMultiKind(config, header)
	if err != nil {
		return fmt.Errorf("failed to encode scaffold.xcf: %w", err)
	}
	if err := os.WriteFile(xcfPath, out, 0600); err != nil {
		return fmt.Errorf("failed to write scaffold.xcf: %w", err)
	}

	fmt.Printf("\nscaffold.xcf updated. Run 'xcaffold apply' to render output\n")
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
		relPath := filepath.Join("skills", p.ID, "SKILL.md")
		config.Skills[p.ID] = ast.SkillConfig{
			Description:      fmt.Sprintf("Translated from workflow %s", p.ID),
			InstructionsFile: relPath,
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
		relPath := filepath.Join("rules", p.ID+".md")
		config.Rules[p.ID] = ast.RuleConfig{
			Description:      fmt.Sprintf("Constraints from workflow %s", p.ID),
			InstructionsFile: relPath,
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
	if config.Settings.Permissions == nil {
		config.Settings.Permissions = &ast.PermissionsConfig{}
	}
	existing := make(map[string]bool, len(config.Settings.Permissions.Allow))
	for _, e := range config.Settings.Permissions.Allow {
		existing[e] = true
	}
	for _, entry := range allowEntries {
		if !existing[entry] {
			config.Settings.Permissions.Allow = append(config.Settings.Permissions.Allow, entry)
		}
	}
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
func resolveSourceFiles(source string) ([]string, error) {
	abs, err := filepath.Abs(source)
	if err != nil {
		return nil, fmt.Errorf("could not resolve path: %w", err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path does not exist: %s", abs)
		}
		return nil, err
	}

	if !info.IsDir() {
		if !strings.HasSuffix(strings.ToLower(abs), ".md") {
			return nil, fmt.Errorf("source file must be a .md file, got: %s", filepath.Base(abs))
		}
		return []string{abs}, nil
	}

	entries, err := os.ReadDir(abs)
	if err != nil {
		return nil, fmt.Errorf("could not read directory: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			files = append(files, filepath.Join(abs, e.Name()))
		}
	}

	return files, nil
}

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

// mergeImportDirs consolidates multiple platform directories into a single scaffold.xcf.
// When the same resource ID exists in multiple directories, the version with the larger
// file (richer content) is kept and the conflict is logged.
//
//nolint:gocyclo
func mergeImportDirs(dirs []string, xcfDest string) error {
	if _, err := os.Stat(xcfDest); err == nil {
		return fmt.Errorf("[project] %s already exists. Remove it first to import", xcfDest)
	}
	if err := checkXcfDirPreexistence(xcfDest, "project"); err != nil {
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
			Hooks:     make(ast.HookConfig),
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

	for _, dir := range dirs {
		fmt.Printf("  Scanning %s ...\n", dir)

		// Extract into a temporary config to compare before merging
		tmpConfig := &ast.XcaffoldConfig{
			ResourceScope: ast.ResourceScope{
				Agents:    make(map[string]ast.AgentConfig),
				Skills:    make(map[string]ast.SkillConfig),
				Rules:     make(map[string]ast.RuleConfig),
				Workflows: make(map[string]ast.WorkflowConfig),
				Hooks:     make(ast.HookConfig),
				MCP:       make(map[string]ast.MCPConfig),
			},
		}
		tmpCount := 0

		if err := extractAgents(dir, "project", tmpConfig, &tmpCount, &warnings); err != nil {
			return err
		}
		if err := extractSkills(dir, "project", tmpConfig, &tmpCount, &warnings); err != nil {
			return err
		}
		if err := extractRules(dir, "project", tmpConfig, &tmpCount, &warnings); err != nil {
			return err
		}
		if err := extractWorkflows(dir, "project", tmpConfig, &tmpCount, &warnings); err != nil {
			return err
		}

		// Parse settings from this dir
		settingsPath := filepath.Join(dir, "settings.json")
		if data, err := os.ReadFile(settingsPath); err == nil {
			if err := importSettings(data, config, &importCount, &warnings); err != nil {
				warnings = append(warnings, fmt.Sprintf("%s/settings.json partially imported: %v", dir, err))
			}
		}

		// Parse hooks.json from this dir
		hooksPath := filepath.Join(dir, "hooks.json")
		if data, err := os.ReadFile(hooksPath); err == nil {
			if err := json.Unmarshal(data, &config.Hooks); err != nil {
				warnings = append(warnings, fmt.Sprintf("%s/hooks.json failed to parse: %v", dir, err))
			} else {
				importCount++
			}
		}

		// Merge agents — richer instructions win on conflict
		for id, ac := range tmpConfig.Agents {
			if _, exists := config.Agents[id]; exists {
				newSize := int64(len(ac.Instructions))
				oldSize := int64(len(config.Agents[id].Instructions))
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
				newSize := int64(len(sc.Instructions))
				oldSize := int64(len(config.Skills[id].Instructions))
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
				newSize := int64(len(rc.Instructions))
				oldSize := int64(len(config.Rules[id].Instructions))
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
				newSize := int64(len(wc.Instructions))
				oldSize := int64(len(config.Workflows[id].Instructions))
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
	}

	// Detect compilation targets from all scanned platform directories.
	if config.Project != nil {
		config.Project.Targets = detectTargets(dirs...)
		config.Project.AgentRefs = sortedMapKeysStr(config.Agents)
		config.Project.SkillRefs = sortedMapKeysStr(config.Skills)
		config.Project.RuleRefs = sortedMapKeysStr(config.Rules)
		config.Project.WorkflowRefs = sortedMapKeysStr(config.Workflows)
		config.Project.MCPRefs = sortedMapKeysStr(config.MCP)
	}

	// Write split .xcf files: scaffold.xcf (kind: project) + xcf/**/*.xcf
	if err := WriteSplitFiles(config, "."); err != nil {
		return fmt.Errorf("[project] failed to write split xcf files: %w", err)
	}

	fmt.Printf("\n[project] ✓ Merge complete. Created %s with %d resources from %d directories.\n",
		xcfDest, importCount, len(dirs))
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

func extractWorkflows(claudeDir, scopeName string, config *ast.XcaffoldConfig, count *int, warnings *[]string) error {
	workflowFiles, _ := filepath.Glob(filepath.Join(claudeDir, "workflows", "*.md"))
	for _, f := range workflowFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			*warnings = append(*warnings, fmt.Sprintf("skipping workflow %s: %v", f, err))
			continue
		}
		id := strings.TrimSuffix(filepath.Base(f), ".md")
		if id == "" {
			continue
		}

		body := extractBodyAfterFrontmatter(data)

		workflowCfg := ast.WorkflowConfig{Description: "Imported workflow"}
		if fm, ok := extractFrontmatter(data); ok {
			_ = yaml.Unmarshal(fm, &workflowCfg)
			workflowCfg.InstructionsFile = ""
		}
		workflowCfg.Instructions = body

		if config.Workflows == nil {
			config.Workflows = make(map[string]ast.WorkflowConfig)
		}
		config.Workflows[id] = workflowCfg
		*count++
	}
	return nil
}
