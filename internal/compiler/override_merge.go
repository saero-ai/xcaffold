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

	// --- Scalars (replace on non-zero) ---
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
	if override.Mode != "" {
		result.Mode = override.Mode
	}
	if override.PermissionMode != "" {
		result.PermissionMode = override.PermissionMode
	}
	if override.Isolation != "" {
		result.Isolation = override.Isolation
	}
	if override.When != "" {
		result.When = override.When
	}
	if override.Color != "" {
		result.Color = override.Color
	}
	if override.InitialPrompt != "" {
		result.InitialPrompt = override.InitialPrompt
	}

	// --- Bool pointers (replace on non-nil) ---
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

	// --- Lists (replace entire list on non-empty) ---
	if len(override.Tools) > 0 {
		result.Tools = append([]string(nil), override.Tools...)
	}
	if len(override.DisallowedTools) > 0 {
		result.DisallowedTools = append([]string(nil), override.DisallowedTools...)
	}
	if len(override.Memory) > 0 {
		result.Memory = append(ast.FlexStringSlice(nil), override.Memory...)
	}
	if len(override.Skills) > 0 {
		result.Skills = append([]string(nil), override.Skills...)
	}
	if len(override.Rules) > 0 {
		result.Rules = append([]string(nil), override.Rules...)
	}
	if len(override.MCP) > 0 {
		result.MCP = append([]string(nil), override.MCP...)
	}
	if len(override.Assertions) > 0 {
		result.Assertions = append([]string(nil), override.Assertions...)
	}

	// --- Maps (deep merge — override keys win, base keys preserved) ---
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

	// --- Body (replace when non-empty, inherit when absent) ---
	if override.Body != "" {
		result.Body = override.Body
	}

	// Internal provenance fields are intentionally NOT merged.
	// result.Inherited and result.SourceProvider carry base values.

	return result
}
