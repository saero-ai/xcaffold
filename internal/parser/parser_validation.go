package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
)

// CrossReferenceIssue represents a single cross-reference validation issue.
// These are collected separately from structural errors so they can be warnings.
type CrossReferenceIssue struct {
	AgentID      string
	ResourceType string // "skill", "rule", "mcp"
	ResourceID   string
	Message      string
}

// Diagnostic represents a single validation finding returned by ValidateFile.
// Severity is either "error" or "warning". Errors cause non-zero exits in
// xcaffold validate; warnings are informational only.
type Diagnostic struct {
	Severity string // "error" or "warning"
	Message  string
}

// validateID checks that an ID contains no invalid characters.
func validateID(kind, id string) error {
	if strings.ContainsAny(id, "\\") || strings.Contains(id, "..") {
		return fmt.Errorf("%s id contains invalid characters: %q", kind, id)
	}
	if strings.Contains(id, "/") && kind != "rule" {
		return fmt.Errorf("%s id contains invalid characters: %q", kind, id)
	}
	return nil
}

// knownTools is the set of recognized tool names for permissions and agent tools.
var knownTools = map[string]bool{
	"Read": true, "Write": true, "Edit": true, "MultiEdit": true,
	"Bash": true, "Glob": true, "Grep": true, "LS": true,
	"WebFetch": true, "WebSearch": true,
	"TodoRead": true, "TodoWrite": true,
	"NotebookRead": true, "NotebookEdit": true,
	"Task": true, "Computer": true, "AskUserQuestion": true,
	"Agent": true, "ExitPlanMode": true, "EnterPlanMode": true,
	"mcp": true,
}

// validHookEvents is the set of accepted hook event names.
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
	// Gemini-native events (accepted at parse time; compile-time handles per-target enforcement)
	"BeforeAgent": true, "AfterAgent": true,
	"BeforeModel": true, "AfterModel": true,
	"BeforeToolSelection": true, "PreCompress": true,
	"BeforeTool": true, "AfterTool": true,
}

// validRuleActivations is the set of accepted activation values for rule kind.
var validRuleActivations = map[string]bool{
	ast.RuleActivationAlways:         true,
	ast.RuleActivationPathGlob:       true,
	ast.RuleActivationModelDecided:   true,
	ast.RuleActivationManualMention:  true,
	ast.RuleActivationExplicitInvoke: true,
}

// pathFreeActivations are rule activations that must have an empty paths list.
var pathFreeActivations = map[string]bool{
	ast.RuleActivationAlways:         true,
	ast.RuleActivationModelDecided:   true,
	ast.RuleActivationManualMention:  true,
	ast.RuleActivationExplicitInvoke: true,
}

// validExcludeAgents is the set of accepted values for the exclude-agents field.
var validExcludeAgents = map[string]bool{
	"code-review": true,
	"cloud-agent": true,
}

// validLoweringStrategies is the set of accepted lowering-strategy values for
// workflow targets.<provider>.provider["lowering-strategy"].
var validLoweringStrategies = map[string]bool{
	"rule-plus-skill": true,
	"prompt-file":     true,
	"custom-command":  true,
}

// knownPlugins is the hardcoded registry of officially supported plugin IDs.
// Plugin validation produces warnings only — custom plugins are not errors.
var knownPlugins = map[string]bool{
	"commit-commands":   true,
	"security-guidance": true,
	"code-review":       true,
	"pr-review-toolkit": true,
}

// reservedOutputPrefixes are compiler output directories and well-known agent
// config paths. instructions-file paths starting with these prefixes create
// circular dependencies where the compiler reads its own output, or reference
// files managed by other providers outside the project tree.
var reservedOutputPrefixes = []string{
	"~/.claude/",
	"~/.gemini/",
	".agents/",
	".antigravity/",
	".claude/",
	".cursor/",
	".cursorrules",
	".gemini/",
}

// reservedOutputFilenames are root-level files written directly by the compiler.
// Pointing instructions-file at one of these creates a circular read dependency.
var reservedOutputFilenames = []string{
	"CLAUDE.md",
	"AGENTS.md",
	"GEMINI.md",
}

// reservedOutputPaths are specific files and directories written by the compiler.
// Exact-match and prefix-match are both applied (directory entries end with /).
var reservedOutputPaths = []string{
	".github/copilot-instructions.md",
	".github/instructions/",
	".github/prompts/",
}

// validateRuleActivations enforces activation enum and paths co-constraints
// across all rules in the config. It also validates exclude-agents enum values
// and emits a deprecation warning to stderr when always-apply is used without
// the activation field.
func validateRuleActivations(c *ast.XcaffoldConfig) error {
	for _, rule := range c.Rules {
		if rule.Activation != "" {
			if !validRuleActivations[rule.Activation] {
				return fmt.Errorf(
					"rule %q: activation must be one of: always, path-glob, model-decided, manual-mention, explicit-invoke (got %q)",
					rule.Name, rule.Activation,
				)
			}
			if rule.Activation == ast.RuleActivationPathGlob && len(rule.Paths.Values) == 0 {
				return fmt.Errorf(
					"rule %q: activation %q requires at least one path in paths",
					rule.Name, rule.Activation,
				)
			}
			if pathFreeActivations[rule.Activation] && len(rule.Paths.Values) > 0 {
				return fmt.Errorf(
					"rule %q: paths must be empty when activation is %q",
					rule.Name, rule.Activation,
				)
			}
		}
		for _, agent := range rule.ExcludeAgents.Values {
			if !validExcludeAgents[agent] {
				return fmt.Errorf(
					"rule %q: exclude-agents value %q must be one of: code-review, cloud-agent",
					rule.Name, agent,
				)
			}
		}
		if rule.AlwaysApply != nil && rule.Activation == "" {
			fmt.Fprintf(os.Stderr,
				"DEPRECATION: rule %q uses always-apply without activation; migrate to activation: always\n",
				rule.Name,
			)
		}
	}
	return nil
}

// validateWorkflows enforces semantic constraints on all workflow entries:
//   - steps and top-level instructions/instructions-file are mutually exclusive
//   - every step must have a non-empty name
//   - targets.<provider>.provider["lowering-strategy"] must be a known value
//   - api-version, if set, must be "workflow/v1"
//   - step instructions-file paths may not target reserved output directories
func validateWorkflows(c *ast.XcaffoldConfig) error {
	for id, wf := range c.Workflows {
		// api-version validation
		if wf.ApiVersion != "" && wf.ApiVersion != "workflow/v1" {
			return fmt.Errorf("workflow %q: api-version %q is not supported; only \"workflow/v1\" is accepted", id, wf.ApiVersion)
		}

		// steps vs frontmatter+body mutex
		if len(wf.Steps) > 0 && wf.Body != "" {
			return fmt.Errorf("workflow %q: steps and inline body are mutually exclusive; use steps for multi-step workflows or the markdown body for single-body workflows", id)
		}

		// per-step validations
		for i, step := range wf.Steps {
			if step.Name == "" {
				return fmt.Errorf("workflow %q: step[%d] is missing a required name field", id, i)
			}
			// step body is required
			if step.Body == "" {
				return fmt.Errorf("workflow %q step %q: must define step instructions in markdown body under ## %s", id, step.Name, step.Name)
			}
		}

		// lowering-strategy enum validation across all target providers
		for provider, override := range wf.Targets {
			if override.Provider == nil {
				continue
			}
			if raw, ok := override.Provider["lowering-strategy"]; ok {
				strategy, _ := raw.(string)
				if !validLoweringStrategies[strategy] {
					return fmt.Errorf("workflow %q: targets.%s.provider[\"lowering-strategy\"] %q is invalid; must be one of: rule-plus-skill, prompt-file, custom-command", id, provider, strategy)
				}
			}
		}
	}
	return nil
}

// validatePartial validates a configuration that may be a fragment (kind: agent, kind: skill, etc).
func validatePartial(c *ast.XcaffoldConfig, globalScope bool) error {
	if err := validateIDs(c); err != nil {
		return err
	}
	var hookEvents ast.HookConfig
	if dh, ok := c.Hooks["default"]; ok {
		hookEvents = dh.Events
	}
	if err := validateHookEvents(hookEvents); err != nil {
		return err
	}

	if err := validateRuleActivations(c); err != nil {
		return err
	}
	if err := validateWorkflows(c); err != nil {
		return err
	}
	return nil
}

// validateMergedStructural validates base and permission rules but skips cross-reference checks.
// Used by the validate command to allow separate handling of cross-reference warnings.
func validateMergedStructural(c *ast.XcaffoldConfig) error {
	if err := validateBase(c); err != nil {
		return err
	}
	if err := validatePermissions(c); err != nil {
		return err
	}
	return nil
}

// validateMerged performs full validation including cross-references.
func validateMerged(c *ast.XcaffoldConfig) error {
	if err := validateMergedStructural(c); err != nil {
		return err
	}
	if err := validateCrossReferences(c); err != nil {
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
//
//nolint:gocyclo
func validatePermissions(c *ast.XcaffoldConfig) error {
	settings := c.Settings["default"]
	if settings.Permissions == nil {
		return nil
	}
	p := settings.Permissions

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
		// disallowed-tools vs settings.permissions.allow
		for _, tool := range agent.DisallowedTools.Values {
			for rule := range allowSet {
				ruleName, _, _ := parsePermissionRule(rule)
				if ruleName == tool {
					return fmt.Errorf("agent %q: tool %q is in disallowed-tools but also in settings.permissions.allow", agentID, tool)
				}
			}
		}
		// agent.tools vs settings.permissions.deny (bare deny only)
		for _, tool := range agent.Tools.Values {
			if denySet[tool] {
				return fmt.Errorf("agent %q: tool %q is required by agent but is unconditionally denied in settings.permissions.deny", agentID, tool)
			}
		}
	}

	return nil
}

// validateBase performs base-level validation of version and project fields.
func validateBase(c *ast.XcaffoldConfig) error {
	if c.Version == "" {
		return fmt.Errorf("version is required (e.g. \"1.0\")")
	}

	if c.Extends == "" && c.Project != nil {
		name := strings.TrimSpace(c.Project.Name)
		if name == "" {
			return fmt.Errorf("project.name is required and must not be empty unless extending another config")
		}
	}

	return nil
}

// validateResourceIDs checks all IDs of a given resource type.
func validateResourceIDs[T any](resources map[string]T, kind string) error {
	for id := range resources {
		if err := validateID(kind, id); err != nil {
			return err
		}
	}
	return nil
}

// validateIDs validates all resource IDs in the config.
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
	if err := validateResourceIDs(c.Hooks, "hook-block"); err != nil {
		return err
	}
	if err := validateResourceIDs(c.MCP, "mcp"); err != nil {
		return err
	}
	if err := validateResourceIDs(c.Workflows, "workflow"); err != nil {
		return err
	}
	if err := validateResourceIDs(c.Policies, "policy"); err != nil {
		return err
	}
	return nil
}

// validateHookEvents validates that all hook event names are recognized.
func validateHookEvents(hooks ast.HookConfig) error {
	for event := range hooks {
		if !validHookEvents[event] {
			return fmt.Errorf("unknown hook event %q; see documentation for supported lifecycle events", event)
		}
	}
	return nil
}

// validateCrossReferencesAsList collects all cross-reference issues without stopping
// on the first one. Returns all issues found (may be empty).
func validateCrossReferencesAsList(c *ast.XcaffoldConfig) []CrossReferenceIssue {
	var issues []CrossReferenceIssue

	for agentID, agent := range c.Agents {
		for _, skillID := range agent.Skills.Values {
			if _, ok := c.Skills[skillID]; !ok {
				issues = append(issues, CrossReferenceIssue{
					AgentID:      agentID,
					ResourceType: "skill",
					ResourceID:   skillID,
					Message:      fmt.Sprintf("agent %q references skill %q: not found in project scope (global scope not yet available)", agentID, skillID),
				})
			}
		}
		for _, ruleID := range agent.Rules.Values {
			if _, ok := c.Rules[ruleID]; !ok {
				issues = append(issues, CrossReferenceIssue{
					AgentID:      agentID,
					ResourceType: "rule",
					ResourceID:   ruleID,
					Message:      fmt.Sprintf("agent %q references rule %q: not found in project scope (global scope not yet available)", agentID, ruleID),
				})
			}
		}
		for _, mcpID := range agent.MCP.Values {
			if _, ok := c.MCP[mcpID]; !ok {
				issues = append(issues, CrossReferenceIssue{
					AgentID:      agentID,
					ResourceType: "mcp",
					ResourceID:   mcpID,
					Message:      fmt.Sprintf("agent %q references mcp server %q: not found in project scope (global scope not yet available)", agentID, mcpID),
				})
			}
		}
	}

	return issues
}

// validateCrossReferences is the legacy error-on-first version. For new code, use validateCrossReferencesAsList.
func validateCrossReferences(c *ast.XcaffoldConfig) error {
	issues := validateCrossReferencesAsList(c)
	if len(issues) > 0 {
		return fmt.Errorf("%s", issues[0].Message)
	}
	return nil
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

// validateFileRefs checks that instructions-file paths and skill references
// exist on disk, and detects duplicate IDs across resource types.
//
//nolint:gocyclo
func validateFileRefs(c *ast.XcaffoldConfig, baseDir string) []Diagnostic {
	var diags []Diagnostic

	// Skill subdirectory file sets: warn on missing files for references, scripts, assets, examples
	for id, skill := range c.Skills {
		for _, subdirPaths := range []struct {
			subdir string
			paths  []string
		}{
			{"references", skill.References.Values},
			{"scripts", skill.Scripts.Values},
			{"assets", skill.Assets.Values},
			{"examples", skill.Examples.Values},
		} {
			for _, ref := range subdirPaths.paths {
				if ref == "" {
					continue
				}
				abs := filepath.Join(baseDir, ref)
				if _, err := os.Stat(abs); os.IsNotExist(err) {
					diags = append(diags, Diagnostic{
						Severity: "warning",
						Message:  fmt.Sprintf("skill %q %s file that does not exist: %q", id, subdirPaths.subdir, ref),
					})
				}
			}
		}
	}

	// Duplicate ID check across resource types
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
	check(c.Settings["default"].EnabledPlugins, "settings")
	if c.Project != nil {
		check(c.Project.Local.EnabledPlugins, "local")
	}
	return diags
}
