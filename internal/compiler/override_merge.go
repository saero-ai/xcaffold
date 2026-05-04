package compiler

import "github.com/saero-ai/xcaffold/internal/ast"

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
	if override.MaxTurns != 0 {
		result.MaxTurns = override.MaxTurns
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
	if len(override.Tools.Values) > 0 {
		result.Tools = ast.ClearableList{Values: append([]string(nil), override.Tools.Values...)}
	}
	if len(override.DisallowedTools.Values) > 0 {
		result.DisallowedTools = ast.ClearableList{Values: append([]string(nil), override.DisallowedTools.Values...)}
	}
	if len(override.Memory) > 0 {
		result.Memory = append(ast.FlexStringSlice(nil), override.Memory...)
	}
	if len(override.Skills.Values) > 0 {
		result.Skills = ast.ClearableList{Values: append([]string(nil), override.Skills.Values...)}
	}
	if len(override.Rules.Values) > 0 {
		result.Rules = ast.ClearableList{Values: append([]string(nil), override.Rules.Values...)}
	}
	if len(override.MCP.Values) > 0 {
		result.MCP = ast.ClearableList{Values: append([]string(nil), override.MCP.Values...)}
	}
	if len(override.Assertions.Values) > 0 {
		result.Assertions = ast.ClearableList{Values: append([]string(nil), override.Assertions.Values...)}
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

	// --- Lists (replace entire list on non-empty) ---
	if len(override.AllowedTools.Values) > 0 {
		result.AllowedTools = ast.ClearableList{Values: append([]string(nil), override.AllowedTools.Values...)}
	}
	if len(override.References.Values) > 0 {
		result.References = ast.ClearableList{Values: append([]string(nil), override.References.Values...)}
	}
	if len(override.Scripts.Values) > 0 {
		result.Scripts = ast.ClearableList{Values: append([]string(nil), override.Scripts.Values...)}
	}
	if len(override.Assets.Values) > 0 {
		result.Assets = ast.ClearableList{Values: append([]string(nil), override.Assets.Values...)}
	}
	if len(override.Examples.Values) > 0 {
		result.Examples = ast.ClearableList{Values: append([]string(nil), override.Examples.Values...)}
	}

	// --- Maps (deep merge — override keys win, base keys preserved) ---
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

	// --- Lists (replace entire list on non-empty) ---
	if len(override.Paths) > 0 {
		result.Paths = append([]string(nil), override.Paths...)
	}
	if len(override.ExcludeAgents) > 0 {
		result.ExcludeAgents = append([]string(nil), override.ExcludeAgents...)
	}

	// --- Maps (deep merge — override keys win, base keys preserved) ---
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
	if len(override.Env) > 0 {
		merged := make(map[string]string, len(base.Env)+len(override.Env))
		for k, v := range base.Env {
			merged[k] = v
		}
		for k, v := range override.Env {
			merged[k] = v
		}
		result.Env = merged
	}
	if len(override.Headers) > 0 {
		merged := make(map[string]string, len(base.Headers)+len(override.Headers))
		for k, v := range base.Headers {
			merged[k] = v
		}
		for k, v := range override.Headers {
			merged[k] = v
		}
		result.Headers = merged
	}
	if len(override.OAuth) > 0 {
		merged := make(map[string]string, len(base.OAuth)+len(override.OAuth))
		for k, v := range base.OAuth {
			merged[k] = v
		}
		for k, v := range override.OAuth {
			merged[k] = v
		}
		result.OAuth = merged
	}

	// Internal provenance fields are intentionally NOT merged.
	return result
}
