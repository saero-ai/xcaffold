package compiler

import (
	"github.com/saero-ai/xcaffold/internal/ast"
)

// mergeClearableList implements tri-state merge for ClearableList fields:
//   - Cleared=true:             clear the field (override wins)
//   - Values non-empty:         replace the field (override wins)
//   - Zero-value (both false):  inherit the base value
func mergeClearableList(base, override ast.ClearableList) ast.ClearableList {
	if override.Cleared {
		return ast.ClearableList{Cleared: true}
	}
	if len(override.Values) > 0 {
		return ast.ClearableList{Values: append([]string(nil), override.Values...)}
	}
	return base
}

// mergeAgentConfig merges override into base using provider-override semantics:
//
//   - Scalars: override replaces base when the override value is non-zero.
//   - Lists: override replaces the entire base list when the override slice is non-empty.
//   - Maps: deep merge — override keys win, base keys not present in override are preserved.
//   - Body: override replaces when non-empty; base is inherited when override is empty.
//   - Bool pointers: override replaces when non-nil.
//
// Internal fields (Inherited, SourceProvider) are never merged; they are
// carried from base to preserve provenance metadata.
func mergeAgentConfig(base, override ast.AgentConfig) ast.AgentConfig {
	result := base
	mergeAgentScalars(&result, override)
	mergeAgentBoolPtrs(&result, override)
	mergeAgentLists(&result, override)
	mergeAgentMaps(&result, base, override)
	mergeAgentBody(&result, override)
	// Internal provenance fields are intentionally NOT merged.
	// result.Inherited and result.SourceProvider carry base values.
	return result
}

func mergeAgentScalars(result *ast.AgentConfig, override ast.AgentConfig) {
	if override.Name != "" {
		result.Name = override.Name
	}
	if override.Description != "" {
		result.Description = override.Description
	}
	if override.Model != "" {
		result.Model = override.Model
	}
	if override.Effort != "" {
		result.Effort = override.Effort
	}
	if override.MaxTurns != nil {
		v := *override.MaxTurns
		result.MaxTurns = &v
	}
	if override.PermissionMode != "" {
		result.PermissionMode = override.PermissionMode
	}
	if override.Isolation != "" {
		result.Isolation = override.Isolation
	}
	if override.Color != "" {
		result.Color = override.Color
	}
	if override.InitialPrompt != "" {
		result.InitialPrompt = override.InitialPrompt
	}
}

func mergeAgentBoolPtrs(result *ast.AgentConfig, override ast.AgentConfig) {
	if override.Readonly != nil {
		v := *override.Readonly
		result.Readonly = &v
	}
	if override.DisableModelInvocation != nil {
		v := *override.DisableModelInvocation
		result.DisableModelInvocation = &v
	}
	if override.UserInvocable != nil {
		v := *override.UserInvocable
		result.UserInvocable = &v
	}
	if override.Background != nil {
		v := *override.Background
		result.Background = &v
	}
}

func mergeAgentLists(result *ast.AgentConfig, override ast.AgentConfig) {
	result.Tools = mergeClearableList(result.Tools, override.Tools)
	result.DisallowedTools = mergeClearableList(result.DisallowedTools, override.DisallowedTools)
	result.Skills = mergeClearableList(result.Skills, override.Skills)
	result.Rules = mergeClearableList(result.Rules, override.Rules)
	result.MCP = mergeClearableList(result.MCP, override.MCP)
	result.Assertions = mergeClearableList(result.Assertions, override.Assertions)
	// Memory stays FlexStringSlice — not part of ClearableList migration.
	if len(override.Memory) > 0 {
		result.Memory = append(ast.FlexStringSlice(nil), override.Memory...)
	}
}

func mergeAgentMaps(result *ast.AgentConfig, base, override ast.AgentConfig) {
	if len(override.MCPServers) > 0 {
		merged := make(map[string]ast.MCPConfig, len(base.MCPServers)+len(override.MCPServers))
		for k, v := range base.MCPServers {
			merged[k] = v
		}
		for k, v := range override.MCPServers {
			merged[k] = v
		}
		result.MCPServers = merged
	}

	// HookConfig is map[string][]HookMatcherGroup — deep merge: override event
	// keys replace base event keys; base events not present in override are kept.
	if len(override.Hooks) > 0 {
		merged := make(ast.HookConfig, len(base.Hooks)+len(override.Hooks))
		for k, v := range base.Hooks {
			merged[k] = v
		}
		for k, v := range override.Hooks {
			merged[k] = v
		}
		result.Hooks = merged
	}

	// Targets map: deep merge — override target keys win.
	result.Targets = mergeTargetMap(base.Targets, override.Targets)
}

func mergeAgentBody(result *ast.AgentConfig, override ast.AgentConfig) {
	if override.Body != "" {
		result.Body = override.Body
	}
}

// mergeSkillConfig merges override into base using provider-override semantics.
// See mergeAgentConfig for the full description of merge rules.
func mergeSkillConfig(base, override ast.SkillConfig) ast.SkillConfig {
	result := base

	// --- Scalars (replace on non-zero) ---
	if override.Name != "" {
		result.Name = override.Name
	}
	if override.Description != "" {
		result.Description = override.Description
	}
	if override.WhenToUse != "" {
		result.WhenToUse = override.WhenToUse
	}
	if override.License != "" {
		result.License = override.License
	}
	if override.ArgumentHint != "" {
		result.ArgumentHint = override.ArgumentHint
	}

	// --- Bool pointers (replace on non-nil) ---
	if override.DisableModelInvocation != nil {
		v := *override.DisableModelInvocation
		result.DisableModelInvocation = &v
	}
	if override.UserInvocable != nil {
		v := *override.UserInvocable
		result.UserInvocable = &v
	}

	// --- Lists (tri-state: Cleared wins, Values replace, zero-value inherits) ---
	result.AllowedTools = mergeClearableList(result.AllowedTools, override.AllowedTools)

	// --- Maps (deep merge — override keys win, base keys preserved) ---
	result.Targets = mergeTargetMap(base.Targets, override.Targets)

	// --- Body (replace when non-empty, inherit when absent) ---
	if override.Body != "" {
		result.Body = override.Body
	}

	// Internal provenance fields are intentionally NOT merged.
	return result
}

// mergeRuleConfig merges override into base using provider-override semantics.
// See mergeAgentConfig for the full description of merge rules.
func mergeRuleConfig(base, override ast.RuleConfig) ast.RuleConfig {
	result := base

	// --- Scalars (replace on non-zero) ---
	if override.Name != "" {
		result.Name = override.Name
	}
	if override.Description != "" {
		result.Description = override.Description
	}
	if override.Activation != "" {
		result.Activation = override.Activation
	}

	// --- Bool pointers (replace on non-nil) ---
	if override.AlwaysApply != nil {
		v := *override.AlwaysApply
		result.AlwaysApply = &v
	}

	// --- Lists (tri-state: Cleared wins, Values replace, zero-value inherits) ---
	result.Paths = mergeClearableList(result.Paths, override.Paths)
	result.ExcludeAgents = mergeClearableList(result.ExcludeAgents, override.ExcludeAgents)

	// --- Maps (deep merge — override keys win, base keys preserved) ---
	result.Targets = mergeTargetMap(base.Targets, override.Targets)

	// --- Body (replace when non-empty, inherit when absent) ---
	if override.Body != "" {
		result.Body = override.Body
	}

	// Internal provenance fields are intentionally NOT merged.
	return result
}

// mergeWorkflowConfig merges override into base using provider-override semantics.
// See mergeAgentConfig for the full description of merge rules.
func mergeWorkflowConfig(base, override ast.WorkflowConfig) ast.WorkflowConfig {
	result := base

	// --- Scalars (replace on non-zero) ---
	if override.ApiVersion != "" {
		result.ApiVersion = override.ApiVersion
	}
	if override.Name != "" {
		result.Name = override.Name
	}
	if override.Description != "" {
		result.Description = override.Description
	}

	// --- Lists (replace entire list on non-empty) ---
	// Steps is a slice of structs: override replaces the entire base slice.
	if len(override.Steps) > 0 {
		result.Steps = append([]ast.WorkflowStep(nil), override.Steps...)
	}

	// --- Maps (deep merge — override keys win, base keys preserved) ---
	result.Targets = mergeTargetMap(base.Targets, override.Targets)

	// --- Bool pointer (nil=inherit, non-nil=replace) ---
	if override.AlwaysApply != nil {
		result.AlwaysApply = override.AlwaysApply
	}

	// --- Pointer (nil=inherit, non-nil=replace) ---
	if override.Activation != nil {
		result.Activation = override.Activation
	}

	// --- ClearableList (nil=inherit, cleared=clear, values=replace) ---
	result.Paths = mergeClearableList(base.Paths, override.Paths)

	// --- Lists (replace entire list on non-empty) ---
	if len(override.Artifacts) > 0 {
		result.Artifacts = append([]string(nil), override.Artifacts...)
	}

	// --- Body (replace when non-empty, inherit when absent) ---
	if override.Body != "" {
		result.Body = override.Body
	}

	// Internal provenance fields are intentionally NOT merged.
	return result
}

// mergeMCPConfig merges override into base using provider-override semantics.
// MCPConfig has no Body or Targets field. Maps (Env, Headers, OAuth) are
// deep-merged; all other fields follow standard scalar/list/bool-pointer rules.
// See mergeAgentConfig for the full description of merge rules.
func mergeMCPConfig(base, override ast.MCPConfig) ast.MCPConfig {
	result := base

	// --- Scalars (replace on non-zero) ---
	if override.Name != "" {
		result.Name = override.Name
	}
	if override.Type != "" {
		result.Type = override.Type
	}
	if override.Command != "" {
		result.Command = override.Command
	}
	if override.URL != "" {
		result.URL = override.URL
	}
	if override.Cwd != "" {
		result.Cwd = override.Cwd
	}
	if override.AuthProviderType != "" {
		result.AuthProviderType = override.AuthProviderType
	}

	// --- Bool pointers (replace on non-nil) ---
	if override.Disabled != nil {
		v := *override.Disabled
		result.Disabled = &v
	}

	// --- Lists (replace entire list on non-empty) ---
	if len(override.Args) > 0 {
		result.Args = append([]string(nil), override.Args...)
	}
	if len(override.DisabledTools) > 0 {
		result.DisabledTools = append([]string(nil), override.DisabledTools...)
	}

	// --- Maps (deep merge — override keys win, base keys preserved) ---
	result.Env = mergeStringMap(base.Env, override.Env)
	result.Headers = mergeStringMap(base.Headers, override.Headers)
	result.OAuth = mergeStringMap(base.OAuth, override.OAuth)

	// Internal provenance fields are intentionally NOT merged.
	return result
}

// mergeNamedHookConfig merges override into base using provider-override semantics.
// See mergeAgentConfig for the full description of merge rules.
func mergeNamedHookConfig(base, override ast.NamedHookConfig) ast.NamedHookConfig {
	result := base

	if override.Name != "" {
		result.Name = override.Name
	}
	if override.Description != "" {
		result.Description = override.Description
	}

	if len(override.Artifacts) > 0 {
		result.Artifacts = append([]string(nil), override.Artifacts...)
	}

	if len(override.Events) > 0 {
		merged := make(ast.HookConfig, len(base.Events)+len(override.Events))
		for k, v := range base.Events {
			merged[k] = v
		}
		for k, v := range override.Events {
			merged[k] = v
		}
		result.Events = merged
	}

	if len(override.Targets) > 0 {
		merged := make(map[string]ast.TargetOverride, len(base.Targets)+len(override.Targets))
		for k, v := range base.Targets {
			merged[k] = v
		}
		for k, v := range override.Targets {
			merged[k] = v
		}
		result.Targets = merged
	}

	// Internal provenance fields are intentionally NOT merged.
	return result
}

// mergePolicyConfig merges override into base using provider-override semantics.
// PolicyConfig has no Targets map or Body field.
// See mergeAgentConfig for the full description of merge rules.
func mergePolicyConfig(base, override ast.PolicyConfig) ast.PolicyConfig {
	result := base

	if override.Name != "" {
		result.Name = override.Name
	}
	if override.Description != "" {
		result.Description = override.Description
	}
	if override.Severity != "" {
		result.Severity = override.Severity
	}
	if override.Target != "" {
		result.Target = override.Target
	}

	if override.Match != nil {
		result.Match = override.Match
	}

	if len(override.Require) > 0 {
		result.Require = append([]ast.PolicyRequire(nil), override.Require...)
	}
	if len(override.Deny) > 0 {
		result.Deny = append([]ast.PolicyDeny(nil), override.Deny...)
	}

	// Internal provenance fields are intentionally NOT merged.
	return result
}

// mergeTemplateConfig merges override into base using provider-override semantics.
// TemplateConfig has no Targets map.
// See mergeAgentConfig for the full description of merge rules.
func mergeTemplateConfig(base, override ast.TemplateConfig) ast.TemplateConfig {
	result := base

	if override.Name != "" {
		result.Name = override.Name
	}
	if override.Description != "" {
		result.Description = override.Description
	}
	if override.DefaultTarget != "" {
		result.DefaultTarget = override.DefaultTarget
	}

	if override.Body != "" {
		result.Body = override.Body
	}

	// Internal provenance fields are intentionally NOT merged.
	return result
}

// mergeContextConfig merges override into base using provider-override semantics.
// ContextConfig.Targets is []string (not a map), so it follows list semantics:
// override replaces the entire base slice when non-empty.
// See mergeAgentConfig for the full description of merge rules.
func mergeContextConfig(base, override ast.ContextConfig) ast.ContextConfig {
	result := base

	if override.Name != "" {
		result.Name = override.Name
	}
	if override.Description != "" {
		result.Description = override.Description
	}
	if override.Default {
		result.Default = override.Default
	}

	if len(override.Targets) > 0 {
		result.Targets = append([]string(nil), override.Targets...)
	}

	if override.Body != "" {
		result.Body = override.Body
	}

	// Internal provenance fields are intentionally NOT merged.
	return result
}

// mergeSettingsConfig merges override into base using provider-override semantics.
// See mergeAgentConfig for the full description of merge rules.
func mergeSettingsConfig(base, override ast.SettingsConfig) ast.SettingsConfig {
	result := base
	mergeSettingsScalars(&result, override)
	mergeSettingsBoolPtrs(&result, override)
	mergeSettingsStructPtrs(&result, override)
	mergeSettingsLists(&result, override)
	mergeSettingsMaps(&result, base, override)
	// Internal provenance fields are intentionally NOT merged.
	return result
}

func mergeSettingsScalars(result *ast.SettingsConfig, override ast.SettingsConfig) {
	if override.Name != "" {
		result.Name = override.Name
	}
	if override.Description != "" {
		result.Description = override.Description
	}
	if override.EffortLevel != "" {
		result.EffortLevel = override.EffortLevel
	}
	if override.DefaultShell != "" {
		result.DefaultShell = override.DefaultShell
	}
	if override.Language != "" {
		result.Language = override.Language
	}
	if override.OutputStyle != "" {
		result.OutputStyle = override.OutputStyle
	}
	if override.PlansDirectory != "" {
		result.PlansDirectory = override.PlansDirectory
	}
	if override.Model != "" {
		result.Model = override.Model
	}
	if override.OtelHeadersHelper != "" {
		result.OtelHeadersHelper = override.OtelHeadersHelper
	}
	if override.AutoMemoryDirectory != "" {
		result.AutoMemoryDirectory = override.AutoMemoryDirectory
	}
}

func mergeSettingsBoolPtrs(result *ast.SettingsConfig, override ast.SettingsConfig) {
	if override.IncludeGitInstructions != nil {
		v := *override.IncludeGitInstructions
		result.IncludeGitInstructions = &v
	}
	if override.SkipDangerousModePermissionPrompt != nil {
		v := *override.SkipDangerousModePermissionPrompt
		result.SkipDangerousModePermissionPrompt = &v
	}
	if override.AutoMemoryEnabled != nil {
		v := *override.AutoMemoryEnabled
		result.AutoMemoryEnabled = &v
	}
	if override.DisableAllHooks != nil {
		v := *override.DisableAllHooks
		result.DisableAllHooks = &v
	}
	if override.Attribution != nil {
		v := *override.Attribution
		result.Attribution = &v
	}
	if override.RespectGitignore != nil {
		v := *override.RespectGitignore
		result.RespectGitignore = &v
	}
	if override.DisableSkillShellExecution != nil {
		v := *override.DisableSkillShellExecution
		result.DisableSkillShellExecution = &v
	}
	if override.AlwaysThinkingEnabled != nil {
		v := *override.AlwaysThinkingEnabled
		result.AlwaysThinkingEnabled = &v
	}
}

func mergeSettingsStructPtrs(result *ast.SettingsConfig, override ast.SettingsConfig) {
	if override.CleanupPeriodDays != nil {
		v := *override.CleanupPeriodDays
		result.CleanupPeriodDays = &v
	}
	if override.Permissions != nil {
		result.Permissions = override.Permissions
	}
	if override.Sandbox != nil {
		result.Sandbox = override.Sandbox
	}
	if override.StatusLine != nil {
		result.StatusLine = override.StatusLine
	}
	if override.Agent != nil {
		result.Agent = override.Agent
	}
	if override.Worktree != nil {
		result.Worktree = override.Worktree
	}
	if override.AutoMode != nil {
		result.AutoMode = override.AutoMode
	}
}

func mergeSettingsLists(result *ast.SettingsConfig, override ast.SettingsConfig) {
	if len(override.MdExcludes) > 0 {
		result.MdExcludes = append([]string(nil), override.MdExcludes...)
	}
	if len(override.AvailableModels) > 0 {
		result.AvailableModels = append([]string(nil), override.AvailableModels...)
	}
}

func mergeSettingsMaps(result *ast.SettingsConfig, base, override ast.SettingsConfig) {
	result.Env = mergeStringMap(base.Env, override.Env)

	if len(override.MCPServers) > 0 {
		merged := make(map[string]ast.MCPConfig, len(base.MCPServers)+len(override.MCPServers))
		for k, v := range base.MCPServers {
			merged[k] = v
		}
		for k, v := range override.MCPServers {
			merged[k] = v
		}
		result.MCPServers = merged
	}

	if len(override.EnabledPlugins) > 0 {
		merged := make(map[string]bool, len(base.EnabledPlugins)+len(override.EnabledPlugins))
		for k, v := range base.EnabledPlugins {
			merged[k] = v
		}
		for k, v := range override.EnabledPlugins {
			merged[k] = v
		}
		result.EnabledPlugins = merged
	}

	// HookConfig is map[string][]HookMatcherGroup — deep merge: override event
	// keys replace base event keys; base events not present in override are kept.
	if len(override.Hooks) > 0 {
		merged := make(ast.HookConfig, len(base.Hooks)+len(override.Hooks))
		for k, v := range base.Hooks {
			merged[k] = v
		}
		for k, v := range override.Hooks {
			merged[k] = v
		}
		result.Hooks = merged
	}

	if len(override.Targets) > 0 {
		merged := make(map[string]ast.TargetOverride, len(base.Targets)+len(override.Targets))
		for k, v := range base.Targets {
			merged[k] = v
		}
		for k, v := range override.Targets {
			merged[k] = v
		}
		result.Targets = merged
	}
}

// mergeTargetMap deep-merges two Targets maps. If a target exists in both,
// their TargetOverride structs are deep-merged.
func mergeTargetMap(base, override map[string]ast.TargetOverride) map[string]ast.TargetOverride {
	if len(override) == 0 {
		return base
	}
	merged := make(map[string]ast.TargetOverride, len(base)+len(override))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range override {
		if baseV, exists := base[k]; exists {
			merged[k] = mergeTargetOverride(baseV, v)
		} else {
			merged[k] = v
		}
	}
	return merged
}

// mergeTargetOverride deep-merges two TargetOverride structs.
func mergeTargetOverride(base, override ast.TargetOverride) ast.TargetOverride {
	result := base
	if override.SuppressFidelityWarnings != nil {
		v := *override.SuppressFidelityWarnings
		result.SuppressFidelityWarnings = &v
	}
	if override.SkipSynthesis != nil {
		v := *override.SkipSynthesis
		result.SkipSynthesis = &v
	}
	if len(override.Hooks) > 0 {
		result.Hooks = mergeStringMap(base.Hooks, override.Hooks)
	}
	if len(override.Provider) > 0 {
		result.Provider = mergeAnyMap(base.Provider, override.Provider)
	}
	return result
}

// mergeStringMap deep-merges two string maps: override keys win, base keys not
// present in override are preserved. Returns base unchanged when override is empty.
func mergeStringMap(base, override map[string]string) map[string]string {
	if len(override) == 0 {
		return base
	}
	merged := make(map[string]string, len(base)+len(override))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range override {
		merged[k] = v
	}
	return merged
}

// mergeAnyMap deep-merges two any maps: override keys win, base keys not
// present in override are preserved.
func mergeAnyMap(base, override map[string]any) map[string]any {
	if len(override) == 0 {
		return base
	}
	merged := make(map[string]any, len(base)+len(override))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range override {
		merged[k] = v
	}
	return merged
}
