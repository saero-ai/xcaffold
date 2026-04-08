package parser

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"gopkg.in/yaml.v3"
)

// Parse reads a .xcf YAML configuration from the given reader and returns a
// validated XcaffoldConfig. It treats the configuration as a complete, standalone file.
func Parse(r io.Reader) (*ast.XcaffoldConfig, error) {
	config, err := parsePartial(r)
	if err != nil {
		return nil, err
	}
	if err := validateMerged(config); err != nil {
		return nil, fmt.Errorf("invalid .xcf configuration: %w", err)
	}
	return config, nil
}

func parsePartial(r io.Reader) (*ast.XcaffoldConfig, error) {
	config := &ast.XcaffoldConfig{}
	decoder := yaml.NewDecoder(r)
	decoder.KnownFields(true)
	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("failed to parse .xcf YAML: %w", err)
	}
	// Validate only things that are unconditionally true for partials
	if err := validatePartial(config); err != nil {
		return nil, fmt.Errorf("invalid .xcf configuration part: %w", err)
	}
	return config, nil
}

// FileConfig represents a parsed partial configuration along with its origin file path.
// It is used internally to provide accurate error tracing when duplicate IDs are found.
type FileConfig struct {
	Path   string
	Config *ast.XcaffoldConfig
}

// ParseDirectory recursively scans the given directory for all *.xcf files,
// parses them, merges them strictly (erroring on duplicate IDs), and then
// resolves 'extends:' chains.
func ParseDirectory(dir string) (*ast.XcaffoldConfig, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if path != dir && (strings.HasPrefix(name, ".") || name == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".xcf") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory %q: %w", dir, err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no *.xcf files found in directory %q", dir)
	}

	var partials []FileConfig
	for _, f := range files {
		cfg, err := parseFileExact(f)
		if err != nil {
			return nil, err
		}
		partials = append(partials, FileConfig{Path: f, Config: cfg})
	}

	merged, err := mergeAllStrict(partials)
	if err != nil {
		return nil, fmt.Errorf("failed to merge config files in %q: %w", dir, err)
	}

	if merged.Extends != "" {
		merged, err = resolveExtends(dir, merged)
		if err != nil {
			return nil, err
		}
	}

	if err := validateMerged(merged); err != nil {
		return nil, fmt.Errorf("validation failed for project configuration: %w", err)
	}

	return merged, nil
}

func parseFileExact(path string) (*ast.XcaffoldConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open config %q: %w", path, err)
	}
	defer f.Close()

	config, err := parsePartial(f)
	if err != nil {
		return nil, fmt.Errorf("error in %q: %w", path, err)
	}
	return config, nil
}

// ParseFile reads a .xcf YAML configuration from the given path, resolving
// 'extends:' references recursively. Evaluated as a strict, single file entry point.
func ParseFile(path string) (*ast.XcaffoldConfig, error) {
	config, err := parseFileExact(path)
	if err != nil {
		return nil, err
	}
	if config.Extends != "" {
		config, err = resolveExtends(filepath.Dir(path), config)
		if err != nil {
			return nil, err
		}
	}
	if err := validateMerged(config); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	return config, nil
}

func resolveExtends(contextDir string, config *ast.XcaffoldConfig) (*ast.XcaffoldConfig, error) {
	visited := make(map[string]bool)
	return resolveExtendsRecursive(contextDir, config, visited)
}

func resolveExtendsRecursive(contextDir string, config *ast.XcaffoldConfig, visited map[string]bool) (*ast.XcaffoldConfig, error) {
	if config.Extends == "" {
		return config, nil
	}

	var basePath string
	if config.Extends == "global" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("could not resolve 'extends: global': %w", err)
		}
		basePath = filepath.Join(home, ".claude", "global.xcf")
	} else if filepath.IsAbs(config.Extends) {
		basePath = config.Extends
	} else {
		basePath = filepath.Join(contextDir, config.Extends)
	}

	absPath, err := filepath.Abs(basePath)
	if err != nil {
		return nil, fmt.Errorf("could not resolve extends path %q: %w", basePath, err)
	}

	if visited[absPath] {
		return nil, fmt.Errorf("circular extends detected: %q", absPath)
	}
	visited[absPath] = true

	parsed, err := parseFileExact(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load base config %q: %w", config.Extends, err)
	}

	baseConfig, err := resolveExtendsRecursive(filepath.Dir(absPath), parsed, visited)
	if err != nil {
		return nil, err
	}

	return mergeConfigOverride(baseConfig, config), nil
}

// Merge operations

// mergeAllStrict is used to merge files living in the same directory.
// Duplicate maps (like Agents, Skills, etc.) cause errors.
func mergeAllStrict(partials []FileConfig) (*ast.XcaffoldConfig, error) {
	if len(partials) == 0 {
		return &ast.XcaffoldConfig{}, nil
	}
	merged := &ast.XcaffoldConfig{}
	tracker := make(map[string]map[string]string)

	for _, p := range partials {
		var err error
		if merged.Version != "" && p.Config.Version != "" && merged.Version != p.Config.Version {
			return nil, fmt.Errorf("conflicting versions declared: %q vs %q", merged.Version, p.Config.Version)
		}
		if p.Config.Version != "" {
			merged.Version = p.Config.Version
		}

		if p.Config.Project.Name != "" {
			if merged.Project.Name != "" && merged.Project.Name != p.Config.Project.Name {
				return nil, fmt.Errorf("multiple files declare project.name: %q vs %q", merged.Project.Name, p.Config.Project.Name)
			}
			merged.Project = p.Config.Project
		}

		if p.Config.Extends != "" {
			if merged.Extends != "" && merged.Extends != p.Config.Extends {
				return nil, fmt.Errorf("multiple files declare extends: %q vs %q", merged.Extends, p.Config.Extends)
			}
			merged.Extends = p.Config.Extends
		}

		merged.Agents, err = mergeMapStrict(merged.Agents, p.Config.Agents, "agent", p.Path, tracker)
		if err != nil {
			return nil, err
		}

		merged.Skills, err = mergeMapStrict(merged.Skills, p.Config.Skills, "skill", p.Path, tracker)
		if err != nil {
			return nil, err
		}

		merged.Rules, err = mergeMapStrict(merged.Rules, p.Config.Rules, "rule", p.Path, tracker)
		if err != nil {
			return nil, err
		}

		merged.MCP, err = mergeMapStrict(merged.MCP, p.Config.MCP, "mcp", p.Path, tracker)
		if err != nil {
			return nil, err
		}

		merged.Workflows, err = mergeMapStrict(merged.Workflows, p.Config.Workflows, "workflow", p.Path, tracker)
		if err != nil {
			return nil, err
		}

		// Hooks are additive (append handlers)
		merged.Hooks = mergeHooksAdditive(merged.Hooks, p.Config.Hooks)

		// Overwrite test blocks (assuming only one file declares test config)
		if p.Config.Test.CliPath != "" || p.Config.Test.ClaudePath != "" || p.Config.Test.JudgeModel != "" {
			merged.Test = p.Config.Test
		}

		// Overwrite settings and local blocks (last file wins; ParseDirectory is
		// designed for single-settings-file projects).
		merged.Settings = p.Config.Settings
		merged.Local = p.Config.Local
	}
	return merged, nil
}

func mergeMapStrict[K comparable, V any](base, child map[K]V, kind, filePath string, tracker map[string]map[string]string) (map[K]V, error) {
	if base == nil {
		base = make(map[K]V)
	}
	if child == nil {
		return base, nil
	}
	if tracker[kind] == nil {
		tracker[kind] = make(map[string]string)
	}
	for k, v := range child {
		strKey := fmt.Sprintf("%v", k)
		if origin, exists := tracker[kind][strKey]; exists {
			return nil, fmt.Errorf("duplicate %s ID %q found in %s and %s", kind, strKey, filepath.Base(origin), filepath.Base(filePath))
		}
		base[k] = v
		tracker[kind][strKey] = filePath
	}
	return base, nil
}

func mergeHooksAdditive(base, child ast.HookConfig) ast.HookConfig {
	if base == nil {
		return child
	}
	if child == nil {
		return base
	}
	merged := make(ast.HookConfig)
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range child {
		merged[k] = append(merged[k], v...)
	}
	return merged
}

// mergeConfigOverride is used for extends resolution where the child overrides the base entirely.
func mergeConfigOverride(base, child *ast.XcaffoldConfig) *ast.XcaffoldConfig {
	merged := &ast.XcaffoldConfig{
		Version: child.Version, // child overrides version
	}

	if merged.Version == "" {
		merged.Version = base.Version
	}

	merged.Project = base.Project
	if child.Project.Name != "" {
		merged.Project.Name = child.Project.Name
	}
	if child.Project.Description != "" {
		merged.Project.Description = child.Project.Description
	} // etc (other fields ignored for brevity as before)

	merged.Extends = "" // after resolving, extends is empty

	merged.Agents = mergeMapOverride(base.Agents, child.Agents)
	merged.Skills = mergeMapOverride(base.Skills, child.Skills)
	merged.Rules = mergeMapOverride(base.Rules, child.Rules)
	merged.MCP = mergeMapOverride(base.MCP, child.MCP)
	merged.Workflows = mergeMapOverride(base.Workflows, child.Workflows)
	merged.Hooks = mergeHooksAdditive(base.Hooks, child.Hooks)

	merged.Test = base.Test
	if child.Test.CliPath != "" {
		merged.Test.CliPath = child.Test.CliPath
	}
	if child.Test.ClaudePath != "" {
		merged.Test.ClaudePath = child.Test.ClaudePath
	}
	if child.Test.JudgeModel != "" {
		merged.Test.JudgeModel = child.Test.JudgeModel
	}

	return merged
}

func mergeMapOverride[K comparable, V any](base, child map[K]V) map[K]V {
	if base == nil && child == nil {
		return nil
	}
	merged := make(map[K]V)
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range child {
		merged[k] = v // child overrides base completely
	}
	return merged
}

// Validations

func validateID(kind, id string) error {
	if strings.ContainsAny(id, "/\\") || strings.Contains(id, "..") {
		return fmt.Errorf("%s id contains invalid characters: %q", kind, id)
	}
	return nil
}

var knownTools = map[string]bool{
	"Read": true, "Write": true, "Edit": true, "MultiEdit": true,
	"Bash": true, "Glob": true, "Grep": true, "LS": true,
	"WebFetch": true, "WebSearch": true,
	"TodoRead": true, "TodoWrite": true,
	"NotebookRead": true, "NotebookEdit": true,
	"mcp": true,
}

var validHookEvents = map[string]bool{
	"PreToolUse": true, "PostToolUse": true, "PostToolUseFailure": true,
	"PermissionRequest": true, "PermissionDenied": true,
	"SessionStart": true, "SessionEnd": true,
	"UserPromptSubmit": true, "Stop": true, "StopFailure": true,
	"SubagentStart": true, "SubagentStop": true, "TeammateIdle": true,
	"TaskCreated": true, "TaskCompleted": true,
	"PreCompact": true, "PostCompact": true,
	"InstructionsLoaded": true, "ConfigChange": true,
	"CwdChanged": true, "FileChanged": true,
	"WorktreeCreate": true, "WorktreeRemove": true,
	"Elicitation": true, "ElicitationResult": true,
	"Notification": true,
}

func validatePartial(c *ast.XcaffoldConfig) error {
	if err := validateIDs(c); err != nil {
		return err
	}
	if err := validateHookEvents(c.Hooks); err != nil {
		return err
	}
	if err := validateInstructions(c); err != nil {
		return err
	}
	return nil
}

func validateMerged(c *ast.XcaffoldConfig) error {
	if err := validateBase(c); err != nil {
		return err
	}
	if err := validateCrossReferences(c); err != nil {
		return err
	}
	if err := validatePermissions(c); err != nil {
		return err
	}
	return nil
}

// parsePermissionRule parses a permission rule string of the form "ToolName" or
// "ToolName(pattern)". It applies strings.TrimSpace to both the tool name and
// the pattern. Returns (toolName, pattern, nil) on success, or ("", "", err).
func parsePermissionRule(rule string) (toolName, pattern string, err error) {
	idx := strings.Index(rule, "(")
	if idx == -1 {
		// bare tool name
		name := strings.TrimSpace(rule)
		if name == "" {
			return "", "", fmt.Errorf("permissions: empty rule string")
		}
		return name, "", nil
	}
	// has a pattern
	name := strings.TrimSpace(rule[:idx])
	rest := rule[idx+1:]
	if !strings.HasSuffix(rest, ")") {
		return "", "", fmt.Errorf("permissions: malformed rule %q — missing closing parenthesis", rule)
	}
	pat := strings.TrimSpace(rest[:len(rest)-1])
	if pat == "" {
		return "", "", fmt.Errorf("permissions: malformed rule %q — empty pattern", rule)
	}
	return name, pat, nil
}

// validatePermissions validates permission rule strings in settings.permissions
// and checks for agent/settings contradictions.
func validatePermissions(c *ast.XcaffoldConfig) error {
	if c.Settings.Permissions == nil {
		return nil
	}
	p := c.Settings.Permissions

	allowSet := make(map[string]bool)
	denySet := make(map[string]bool)
	askSet := make(map[string]bool)

	for _, rule := range p.Allow {
		name, _, err := parsePermissionRule(rule)
		if err != nil {
			return fmt.Errorf("invalid .xcf configuration: %w", err)
		}
		if !knownTools[name] {
			return fmt.Errorf("permissions: unknown tool %q in allow rule %q", name, rule)
		}
		allowSet[rule] = true
	}
	for _, rule := range p.Deny {
		name, _, err := parsePermissionRule(rule)
		if err != nil {
			return fmt.Errorf("invalid .xcf configuration: %w", err)
		}
		if !knownTools[name] {
			return fmt.Errorf("permissions: unknown tool %q in deny rule %q", name, rule)
		}
		denySet[rule] = true
	}
	for _, rule := range p.Ask {
		name, _, err := parsePermissionRule(rule)
		if err != nil {
			return fmt.Errorf("invalid .xcf configuration: %w", err)
		}
		if !knownTools[name] {
			return fmt.Errorf("permissions: unknown tool %q in ask rule %q", name, rule)
		}
		askSet[rule] = true
	}

	// Contradiction checks
	for rule := range allowSet {
		if denySet[rule] {
			return fmt.Errorf("permissions: rule %q appears in both allow and deny", rule)
		}
		if askSet[rule] {
			return fmt.Errorf("permissions: rule %q appears in both allow and ask", rule)
		}
	}

	// Agent cross-reference checks
	for agentID, agent := range c.Agents {
		// disallowedTools vs settings.permissions.allow
		for _, tool := range agent.DisallowedTools {
			for rule := range allowSet {
				ruleName, _, _ := parsePermissionRule(rule)
				if ruleName == tool {
					return fmt.Errorf("agent %q: tool %q is in disallowedTools but also in settings.permissions.allow", agentID, tool)
				}
			}
		}
		// agent.tools vs settings.permissions.deny (bare deny only)
		for _, tool := range agent.Tools {
			if denySet[tool] {
				return fmt.Errorf("agent %q: tool %q is required by agent but is unconditionally denied in settings.permissions.deny", agentID, tool)
			}
		}
	}

	return nil
}

func validateBase(c *ast.XcaffoldConfig) error {
	if c.Version == "" {
		return fmt.Errorf("version is required (e.g. \"1.0\")")
	}

	if c.Extends == "" {
		name := strings.TrimSpace(c.Project.Name)
		if name == "" {
			return fmt.Errorf("project.name is required and must not be empty unless extending another config")
		}
	}
	return nil
}

func validateResourceIDs[T any](resources map[string]T, kind string) error {
	for id := range resources {
		if err := validateID(kind, id); err != nil {
			return err
		}
	}
	return nil
}

func validateIDs(c *ast.XcaffoldConfig) error {
	if err := validateResourceIDs(c.Agents, "agent"); err != nil {
		return err
	}
	if err := validateResourceIDs(c.Skills, "skill"); err != nil {
		return err
	}
	if err := validateResourceIDs(c.Rules, "rule"); err != nil {
		return err
	}
	if err := validateResourceIDs(c.Hooks, "hook"); err != nil {
		return err
	}
	if err := validateResourceIDs(c.MCP, "mcp"); err != nil {
		return err
	}
	if err := validateResourceIDs(c.Workflows, "workflow"); err != nil {
		return err
	}
	return nil
}

func validateHookEvents(hooks ast.HookConfig) error {
	for event := range hooks {
		if !validHookEvents[event] {
			return fmt.Errorf("unknown hook event %q; see documentation for supported lifecycle events", event)
		}
	}
	return nil
}

func validateInstructions(c *ast.XcaffoldConfig) error {
	for id, agent := range c.Agents {
		if err := validateInstructionOrFile("agent", id, agent.Instructions, agent.InstructionsFile); err != nil {
			return err
		}
	}
	for id, skill := range c.Skills {
		if err := validateInstructionOrFile("skill", id, skill.Instructions, skill.InstructionsFile); err != nil {
			return err
		}
	}
	for id, rule := range c.Rules {
		if err := validateInstructionOrFile("rule", id, rule.Instructions, rule.InstructionsFile); err != nil {
			return err
		}
	}
	for id, wf := range c.Workflows {
		if err := validateInstructionOrFile("workflow", id, wf.Instructions, wf.InstructionsFile); err != nil {
			return err
		}
	}
	return nil
}

func validateInstructionOrFile(kind, id, inst, file string) error {
	if inst != "" && file != "" {
		return fmt.Errorf("%s %q: instructions and instructions_file are mutually exclusive; set one or the other", kind, id)
	}
	return validateInstructionsFile(kind, id, file)
}

func validateCrossReferences(c *ast.XcaffoldConfig) error {
	for agentID, agent := range c.Agents {
		for _, skillID := range agent.Skills {
			if _, ok := c.Skills[skillID]; !ok {
				return fmt.Errorf("agent %q references undefined skill %q", agentID, skillID)
			}
		}
		for _, ruleID := range agent.Rules {
			if _, ok := c.Rules[ruleID]; !ok {
				return fmt.Errorf("agent %q references undefined rule %q", agentID, ruleID)
			}
		}
		for _, mcpID := range agent.MCP {
			if _, ok := c.MCP[mcpID]; !ok {
				return fmt.Errorf("agent %q references undefined mcp server %q", agentID, mcpID)
			}
		}
	}
	for skillID, skill := range c.Skills {
		if skill.Agent != "" {
			if _, ok := c.Agents[skill.Agent]; !ok {
				return fmt.Errorf("skill %q references undefined agent %q", skillID, skill.Agent)
			}
		}
	}
	return nil
}

// Diagnostic represents a single validation finding returned by ValidateFile.
// Severity is either "error" or "warning". Errors cause non-zero exits in
// xcaffold validate; warnings are informational only.
type Diagnostic struct {
	Severity string // "error" or "warning"
	Message  string
}

// knownPlugins is the hardcoded registry of officially supported plugin IDs.
// Plugin validation produces warnings only — custom plugins are not errors.
var knownPlugins = map[string]bool{
	"commit-commands":   true,
	"security-guidance": true,
	"code-review":       true,
	"pr-review-toolkit": true,
}

// ValidateFile parses the .xcf file at path, runs file-existence checks and
// plugin validation, and returns all diagnostics. ParseFile already runs
// validateCrossReferences internally, so this function does not duplicate it.
func ValidateFile(path string) []Diagnostic {
	config, err := ParseFile(path)
	if err != nil {
		return []Diagnostic{{Severity: "error", Message: err.Error()}}
	}
	var diags []Diagnostic
	diags = append(diags, validateFileRefs(config, filepath.Dir(path))...)
	diags = append(diags, validatePlugins(config)...)
	return diags
}

// validateFileRefs checks that instructions_file paths and skill references
// exist on disk, and detects duplicate IDs across resource types.
func validateFileRefs(c *ast.XcaffoldConfig, baseDir string) []Diagnostic {
	var diags []Diagnostic

	// Skill references: warning on missing files
	for id, skill := range c.Skills {
		for _, ref := range skill.References {
			if ref == "" {
				continue
			}
			abs := filepath.Join(baseDir, ref)
			if _, err := os.Stat(abs); os.IsNotExist(err) {
				diags = append(diags, Diagnostic{
					Severity: "warning",
					Message:  fmt.Sprintf("skill %q references file that does not exist: %q", id, ref),
				})
			}
		}
	}

	// instructions_file existence: error on missing files
	checkInstrFile := func(kind, id, instrFile string) {
		if instrFile == "" {
			return
		}
		abs := filepath.Join(baseDir, instrFile)
		if _, err := os.Stat(abs); os.IsNotExist(err) {
			diags = append(diags, Diagnostic{
				Severity: "error",
				Message:  fmt.Sprintf("%s %q instructions_file not found: %q", kind, id, instrFile),
			})
		}
	}

	for id, agent := range c.Agents {
		checkInstrFile("agent", id, agent.InstructionsFile)
	}
	for id, skill := range c.Skills {
		checkInstrFile("skill", id, skill.InstructionsFile)
	}
	for id, rule := range c.Rules {
		checkInstrFile("rule", id, rule.InstructionsFile)
	}
	for id, wf := range c.Workflows {
		checkInstrFile("workflow", id, wf.InstructionsFile)
	}

	// Duplicate ID check across resource types
	type entry struct{ kind string }
	seen := make(map[string][]string) // id -> []resourceType
	for id := range c.Agents {
		seen[id] = append(seen[id], "agent")
	}
	for id := range c.Skills {
		seen[id] = append(seen[id], "skill")
	}
	for id := range c.Rules {
		seen[id] = append(seen[id], "rule")
	}
	for id := range c.Workflows {
		seen[id] = append(seen[id], "workflow")
	}
	for id, types := range seen {
		if len(types) > 1 {
			diags = append(diags, Diagnostic{
				Severity: "warning",
				Message:  fmt.Sprintf("ID %q is used in both %s and %s; this may cause confusion", id, types[0], types[1]),
			})
		}
	}

	return diags
}

// validatePlugins checks settings.enabledPlugins and local.enabledPlugins
// against the knownPlugins registry. Unknown plugins produce warnings only.
func validatePlugins(c *ast.XcaffoldConfig) []Diagnostic {
	var diags []Diagnostic
	check := func(plugins map[string]bool, block string) {
		for id := range plugins {
			if !knownPlugins[id] {
				diags = append(diags, Diagnostic{
					Severity: "warning",
					Message: fmt.Sprintf(
						"%s.enabledPlugins: unknown plugin %q; known plugins: commit-commands, security-guidance, code-review, pr-review-toolkit",
						block, id,
					),
				})
			}
		}
	}
	check(c.Settings.EnabledPlugins, "settings")
	check(c.Local.EnabledPlugins, "local")
	return diags
}

func validateInstructionsFile(kind, id, path string) error {
	if path == "" {
		return nil
	}
	if filepath.IsAbs(path) {
		return fmt.Errorf("%s %q: instructions_file must be a relative path, got absolute path %q", kind, id, path)
	}
	if strings.ContainsAny(path, "\\") || strings.Contains(path, "..") {
		return fmt.Errorf("%s %q: instructions_file contains invalid path characters: %q", kind, id, path)
	}

	// Prevent circular dependencies by blocking references to compiler output directories.
	if strings.HasPrefix(path, ".claude/") || strings.HasPrefix(path, ".cursor/") || strings.HasPrefix(path, ".agents/") || strings.HasPrefix(path, ".antigravity/") {
		return fmt.Errorf("%s %q: instructions_file cannot reference %q to avoid circular dependencies during compilation", kind, id, strings.Split(path, "/")[0])
	}

	return nil
}
