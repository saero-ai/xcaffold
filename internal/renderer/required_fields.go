package renderer

import (
	"fmt"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/schema"
)

// CheckRequiredFields reads per-provider field requirements from the schema
// registry and returns error-level fidelity notes for any required fields
// that are missing from the resource.
func CheckRequiredFields(target, kind, resourceName string, presentFields map[string]string) []FidelityNote {
	ks, ok := schema.LookupKind(kind)
	if !ok {
		return nil
	}

	var notes []FidelityNote
	for _, f := range ks.Fields {
		providerReq, exists := f.Provider[target]
		if !exists {
			continue
		}
		if providerReq != "required" {
			continue
		}
		if _, present := presentFields[f.YAMLKey]; present {
			continue
		}
		notes = append(notes, FidelityNote{
			Level:    LevelError,
			Target:   target,
			Kind:     kind,
			Resource: resourceName,
			Field:    f.YAMLKey,
			Code:     CodeFieldRequiredForTarget,
			Reason:   fmt.Sprintf("field %q is required by %s but missing from resource %q", f.YAMLKey, target, resourceName),
		})
	}
	return notes
}

// extractAgentPresentFields returns a map of YAML keys that have non-zero values
// on the given AgentConfig. Used by the orchestrator to feed CheckRequiredFields.
func extractAgentPresentFields(a ast.AgentConfig) map[string]string {
	m := make(map[string]string)
	if a.Name != "" {
		m["name"] = a.Name
	}
	if a.Description != "" {
		m["description"] = a.Description
	}
	if a.Model != "" {
		m["model"] = a.Model
	}
	if a.Effort != "" {
		m["effort"] = a.Effort
	}
	if a.MaxTurns != 0 {
		m["max-turns"] = "set"
	}
	if len(a.Tools) > 0 {
		m["tools"] = "set"
	}
	if len(a.DisallowedTools) > 0 {
		m["disallowed-tools"] = "set"
	}
	if a.Readonly != nil {
		m["readonly"] = "set"
	}
	if a.PermissionMode != "" {
		m["permission-mode"] = a.PermissionMode
	}
	if a.DisableModelInvocation != nil {
		m["disable-model-invocation"] = "set"
	}
	if a.UserInvocable != nil {
		m["user-invocable"] = "set"
	}
	if a.Background != nil {
		m["background"] = "set"
	}
	if a.Isolation != "" {
		m["isolation"] = a.Isolation
	}
	if len(a.Memory) > 0 {
		m["memory"] = "set"
	}
	if a.InitialPrompt != "" {
		m["initial-prompt"] = a.InitialPrompt
	}
	if len(a.Skills) > 0 {
		m["skills"] = "set"
	}
	if len(a.MCPServers) > 0 {
		m["mcp-servers"] = "set"
	}
	if len(a.Hooks) > 0 {
		m["hooks"] = "set"
	}
	if a.Body != "" {
		m["body"] = "set"
	}
	return m
}

// extractSkillPresentFields returns a map of YAML keys that have non-zero values
// on the given SkillConfig. Used by the orchestrator to feed CheckRequiredFields.
func extractSkillPresentFields(s ast.SkillConfig) map[string]string {
	m := make(map[string]string)
	if s.Name != "" {
		m["name"] = s.Name
	}
	if s.Description != "" {
		m["description"] = s.Description
	}
	if s.WhenToUse != "" {
		m["when-to-use"] = s.WhenToUse
	}
	if s.License != "" {
		m["license"] = s.License
	}
	if len(s.AllowedTools) > 0 {
		m["allowed-tools"] = "set"
	}
	if s.DisableModelInvocation != nil {
		m["disable-model-invocation"] = "set"
	}
	if s.UserInvocable != nil {
		m["user-invocable"] = "set"
	}
	if s.ArgumentHint != "" {
		m["argument-hint"] = s.ArgumentHint
	}
	if s.Body != "" {
		m["body"] = "set"
	}
	return m
}

// extractRulePresentFields returns a map of YAML keys that have non-zero values
// on the given RuleConfig. Used by the orchestrator to feed CheckRequiredFields.
func extractRulePresentFields(r ast.RuleConfig) map[string]string {
	m := make(map[string]string)
	if r.Name != "" {
		m["name"] = r.Name
	}
	if r.Description != "" {
		m["description"] = r.Description
	}
	if r.AlwaysApply != nil {
		m["always-apply"] = "set"
	}
	if r.Activation != "" {
		m["activation"] = r.Activation
	}
	if len(r.Paths) > 0 {
		m["paths"] = "set"
	}
	if len(r.ExcludeAgents) > 0 {
		m["exclude-agents"] = "set"
	}
	if r.Body != "" {
		m["body"] = "set"
	}
	return m
}
