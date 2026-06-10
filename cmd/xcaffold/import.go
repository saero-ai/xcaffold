package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/importer"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/prompt"
	"github.com/saero-ai/xcaffold/internal/registry"
	providerspkg "github.com/saero-ai/xcaffold/providers"
	_ "github.com/saero-ai/xcaffold/providers/antigravity"
	_ "github.com/saero-ai/xcaffold/providers/claude"
	_ "github.com/saero-ai/xcaffold/providers/copilot"
	_ "github.com/saero-ai/xcaffold/providers/cursor"
	_ "github.com/saero-ai/xcaffold/providers/gemini"
	"github.com/spf13/cobra"
)

var (
	importDryRun     bool
	importForce      bool
	importYes        bool
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

// importScopeContext groups the scope parameters for finalizeImportScope.
type importScopeContext struct {
	xcafDest   string
	outputRoot string
	scopeName  string
	provider   string
}

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import existing provider config into project.xcaf",
	Long: `xcaffold import adopts existing provider configurations into xcaffold.

Detection (Default):
 • Scans .claude/agents/*.md   → extracts to xcaf/agents/<id>.md
 • Scans .claude/skills/*/SKILL.md → extracts to xcaf/skills/<id>/SKILL.md
 • Scans .claude/rules/*.md    → extracts to xcaf/rules/<id>.md
 • Reads .claude/settings.json for MCP and settings context
 • Generates project.xcaf manifest referencing discovered resources

Incremental Import:
 • If project.xcaf or xcaf/ already exists, shows a diff of what would change
 • Use --force to delete existing state and reimport from scratch
 • Use --yes to skip confirmation prompts (CI/CD mode)

Usage:
  $ xcaffold import
  $ xcaffold import --target claude
  $ xcaffold import --dry-run
  $ xcaffold import --force --yes`,
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
	f.BoolVar(&importDryRun, "dry-run", false, "Preview changes without writing to disk")
	f.BoolVar(&importForce, "force", false, "Delete project.xcaf and xcaf/, then reimport from scratch")
	f.BoolVarP(&importYes, "yes", "y", false, "Skip confirmation prompts (CI/CD mode)")
	f.StringVar(&importTargetFlag, "target", "", fmt.Sprintf("import from specific provider: %s", strings.Join(providerspkg.PrimaryNames(), ", ")))

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
	applyKindZeroFilters(config)

	// Name filters: narrow to specific resource
	applyNameFilters(config)
}

// applyKindZeroFilters zeros out kinds not requested via flags.
func applyKindZeroFilters(config *ast.XcaffoldConfig) {
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
}

// applyNameFilters narrows resources to specific names when provided.
func applyNameFilters(config *ast.XcaffoldConfig) {
	filterNamedResource[ast.AgentConfig](
		&config.Agents, importFilterAgent)
	filterNamedResource[ast.SkillConfig](
		&config.Skills, importFilterSkill)
	filterNamedResource[ast.RuleConfig](
		&config.Rules, importFilterRule)
	filterNamedResource[ast.WorkflowConfig](
		&config.Workflows, importFilterWorkflow)
	filterNamedResource[ast.MCPConfig](
		&config.MCP, importFilterMCP)
}

// filterNamedResource narrows a resource map to a single named entry if filter is not "*".
func filterNamedResource[T any](resources *map[string]T, filter string) {
	if filter == "" || filter == "*" || *resources == nil {
		return
	}
	if res, ok := (*resources)[filter]; ok {
		*resources = map[string]T{filter: res}
	} else {
		*resources = nil
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
	for name, mcp := range config.MCP {
		if mcp.Targets == nil {
			mcp.Targets = make(map[string]ast.TargetOverride)
		}
		mcp.Targets[provider] = to
		config.MCP[name] = mcp
	}
	for name, hook := range config.Hooks {
		if hook.Targets == nil {
			hook.Targets = make(map[string]ast.TargetOverride)
		}
		hook.Targets[provider] = to
		config.Hooks[name] = hook
	}
	for name, setting := range config.Settings {
		if setting.Targets == nil {
			setting.Targets = make(map[string]ast.TargetOverride)
		}
		setting.Targets[provider] = to
		config.Settings[name] = setting
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
		outputRoot := filepath.Dir(filepath.Dir(globalXcafPath)) // ~/.xcaffold
		if len(globalDetected) > 1 {
			return mergeImportDirs(globalDetected, globalXcafPath, outputRoot)
		}
		return importScope(globalDetected[0].InputDir(), globalXcafPath, "global", globalDetected[0].Provider(), outputRoot)
	}

	// Validate and normalize --target if set
	if importTargetFlag != "" {
		if !providerspkg.IsRegistered(importTargetFlag) {
			validTargets := providerspkg.RegisteredNames()
			sort.Strings(validTargets)
			return fmt.Errorf("unknown target %q; valid targets: %s", importTargetFlag, strings.Join(validTargets, ", "))
		}
		// Normalize alias to canonical name for consistent state tracking
		canonical, _ := providerspkg.CanonicalName(importTargetFlag)
		importTargetFlag = canonical
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
		return mergeImportDirs(detected, "project.xcaf", ".")
	}
	if len(detected) == 1 {
		imp := detected[0]
		return importScope(imp.InputDir(), "project.xcaf", "project", imp.Provider(), ".")
	}

	return fmt.Errorf("no supported AI provider configuration found in current directory. Supported providers: %s", describeSupportedProviders())
}

// importScope scans a platform directory and writes a xcaf file to xcafDest.
// provider selects provider-specific extraction logic for settings, MCP,
// hooks, project-instruction files, and memory. The provider name must match
// a registered provider (see providers.RegisteredNames()).
// outputRoot is the directory where the xcaf/ tree will be written.
func importScope(platformDir, xcafDest, scopeName, provider, outputRoot string) error {
	// Check if the provider is deprecated and emit warning
	if deprecationWarning, _ := providerspkg.CheckDeprecation(provider); deprecationWarning != "" {
		fmt.Fprintf(os.Stderr, "\n%s %s\n", colorYellow("⚠"), deprecationWarning)
	}

	if shouldPromptForceDelete(xcafDest, outputRoot) {
		fmt.Fprintf(os.Stderr, "\n  %s  --force will DELETE project.xcaf and xcaf/ directory.\n", colorYellow(glyphSrc()))
		fmt.Fprintf(os.Stderr, "     All manual edits to xcaf files will be lost.\n\n")
		doForce := importYes
		if !importYes {
			var err error
			doForce, err = prompt.Confirm("Proceed with destructive reimport?", false)
			if err != nil {
				return fmt.Errorf("prompt error: %w", err)
			}
		}
		if !doForce {
			return nil
		}
		_ = os.Remove(xcafDest)
		_ = os.RemoveAll(filepath.Join(outputRoot, "xcaf"))
	}

	xcafDirPath := filepath.Join(outputRoot, "xcaf")
	if fileExists(xcafDest) || fileExists(xcafDirPath) {
		return incrementalImport(platformDir, xcafDest, scopeName, provider, outputRoot)
	}

	projectDir, err := deriveProjectDir(platformDir)
	if err != nil {
		return err
	}

	config := newImportConfig()
	var warnings []string

	extractAndPostProcess(platformDir, provider, config, &warnings)
	if err := runProviderPostImport(provider, projectDir, config, &warnings); err != nil {
		return err
	}
	if !importDryRun {
		if err := runPostImportSteps(config, outputRoot, false); err != nil {
			return err
		}
	}

	if config.Project != nil {
		config.Project.Targets = detectTargets(platformDir, provider)
	}

	return finalizeImportScope(importScopeContext{xcafDest, outputRoot, scopeName, provider}, config, &warnings)
}

// shouldPromptForceDelete checks if the force delete prompt should be shown.
func shouldPromptForceDelete(xcafDest, outputRoot string) bool {
	xcafExists := fileExists(xcafDest)
	xcafDirExists := fileExists(filepath.Join(outputRoot, "xcaf"))
	return importForce && (xcafExists || xcafDirExists)
}

// fileExists checks if a path exists without panicking on errors.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// deriveProjectDir computes the project root from the provider directory path.
func deriveProjectDir(platformDir string) (string, error) {
	platformAbs, err := filepath.Abs(platformDir)
	if err != nil {
		return "", fmt.Errorf("resolving provider dir: %w", err)
	}
	return filepath.Dir(platformAbs), nil
}

// newImportConfig creates a new XcaffoldConfig for import.
// describeSupportedProviders returns a human-readable list of supported providers
// with display labels, appending "(deprecated)" for non-active providers.
func describeSupportedProviders() string {
	manifests := providerspkg.Manifests()
	var descriptions []string
	for _, m := range manifests {
		label := m.DisplayLabel
		if label == "" {
			label = m.Name
		}
		if m.Status == "deprecated" || m.Status == "sunset" {
			label += " (deprecated)"
		}
		descriptions = append(descriptions, label)
	}
	sort.Strings(descriptions)
	return strings.Join(descriptions, ", ")
}

func newImportConfig() *ast.XcaffoldConfig {
	return &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{Name: inferProjectName()},
		ResourceScope: ast.ResourceScope{
			Agents: make(map[string]ast.AgentConfig),
			Skills: make(map[string]ast.SkillConfig),
			Rules:  make(map[string]ast.RuleConfig),
			MCP:    make(map[string]ast.MCPConfig),
		},
	}
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
			// Skip if this MCP server was already extracted by the main importer.
			// This guard prevents the post-import pass from overwriting a richer
			// resource with a poorer one (e.g., one with serverUrl and disabledTools).
			if _, exists := config.MCP[id]; exists {
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
			if urlStr, ok := serverMap["serverUrl"].(string); ok {
				mc.URL = urlStr
			}
			if disabledToolsRaw, ok := serverMap["disabledTools"].([]interface{}); ok {
				for _, t := range disabledToolsRaw {
					if toolStr, ok := t.(string); ok {
						mc.DisabledTools = append(mc.DisabledTools, toolStr)
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

// detectTargets derives compilation target names from platform directory base names
// by consulting the importer registry. It maps InputDir() names to Provider() names.
// When an explicit provider is set (e.g., --target antigravity2), that provider is
// returned after canonicalization. Otherwise, all matching providers are detected,
// filtered to prefer active (non-deprecated/non-sunset) ones when multiple share
// an input dir, and returned sorted.
func detectTargets(platformDir, explicitProvider string) []string {
	// If an explicit provider was passed (via --target), return only that (canonicalized)
	if explicitProvider != "" {
		if canonical, ok := providerspkg.CanonicalName(explicitProvider); ok {
			return []string{canonical}
		}
		return []string{explicitProvider}
	}

	dirBase := filepath.Base(filepath.Clean(platformDir))
	matchedImporters := findImportersForDir(dirBase)
	targetMap := filterProvidersForDetection(matchedImporters)

	targets := make([]string, 0, len(targetMap))
	for t := range targetMap {
		targets = append(targets, t)
	}
	sort.Strings(targets)
	return targets
}

// findImportersForDir returns all importers whose InputDir matches the base name.
func findImportersForDir(dirBase string) []importer.ProviderImporter {
	var matched []importer.ProviderImporter
	for _, imp := range importer.DefaultImporters() {
		if filepath.Base(filepath.Clean(imp.InputDir())) == dirBase {
			matched = append(matched, imp)
		}
	}
	return matched
}

// filterProvidersForDetection filters matched importers to a target map,
// preferring active providers when multiple share the same input dir.
func filterProvidersForDetection(matched []importer.ProviderImporter) map[string]bool {
	targetMap := make(map[string]bool)

	if len(matched) <= 1 {
		for _, imp := range matched {
			targetMap[imp.Provider()] = true
		}
		return targetMap
	}

	// Multiple importers matched — collect and filter to active providers
	var manifests []providerspkg.ProviderManifest
	for _, imp := range matched {
		if m, ok := providerspkg.ManifestFor(imp.Provider()); ok {
			manifests = append(manifests, m)
		}
	}

	activeManifests := providerspkg.PreferActiveProviders(manifests)
	if len(activeManifests) > 0 {
		for _, m := range activeManifests {
			targetMap[m.Name] = true
		}
		return targetMap
	}

	// Fall back to all matches if all were deprecated/sunset
	for _, imp := range matched {
		targetMap[imp.Provider()] = true
	}
	return targetMap
}

// finalizeImportScope handles memory file writing, resource tagging, filtering, and success messages.
func finalizeImportScope(ctx importScopeContext, config *ast.XcaffoldConfig, warnings *[]string) error {
	tagResourcesWithProvider(config, ctx.provider)
	applyKindFilters(config)

	if importDryRun {
		fmt.Printf("Import plan (dry-run):\n")
		fmt.Printf("  Would create %d agents, %d skills, %d rules, %d workflows, %d MCP servers\n",
			len(config.Agents), len(config.Skills), len(config.Rules), len(config.Workflows), len(config.MCP))
		fmt.Printf("  Target directory: %s\n", ctx.xcafDest)
		return nil
	}

	if err := WriteSplitFiles(config, ctx.outputRoot); err != nil {
		return fmt.Errorf("[%s] failed to write split xcaf files: %w", ctx.scopeName, err)
	}

	importCount := len(config.Agents) + len(config.Skills) + len(config.Rules) +
		len(config.Workflows) + len(config.MCP)
	fmt.Printf("[%s] ✓ Import complete. Created %s with %d resources.\n", ctx.scopeName, ctx.xcafDest, importCount)
	fmt.Printf("  Split xcaf/ files written to xcaf/ directory.\n")
	fmt.Printf("  Resources tagged with targets: [%s]. Remove the targets field to make universal.\n", ctx.provider)
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
		manifest, _ := providerspkg.ManifestFor(provider)
		for id := range config.Skills {
			skillFile := filepath.Join(platformDir, "skills", id, "SKILL.md")
			if _, err := os.Stat(skillFile); err == nil {
				discoveredDirs, subdirsErr := extractSkillSubdirs(skillExtractionCtx{skillFile, id, ""}, &manifest, warnings)
				if subdirsErr != nil {
					*warnings = append(*warnings, fmt.Sprintf("extractSkillSubdirs %s: %v", id, subdirsErr))
				}
				sc := config.Skills[id]
				if len(discoveredDirs) > 0 {
					sc.Artifacts = discoveredDirs
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
