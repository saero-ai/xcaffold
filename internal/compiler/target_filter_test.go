package compiler

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
)

// TestResolveTargetOverrides_UniversalResource_Included verifies that an agent
// with no Targets map is kept for any target and emits no fidelity notes.
func TestResolveTargetOverrides_UniversalResource_Included(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Agents = map[string]ast.AgentConfig{
		"developer": {
			Name:        "developer",
			Description: "test agent",
			Model:       "sonnet",
		},
	}

	notes := resolveTargetOverrides(config, "claude")

	if len(notes) != 0 {
		t.Fatalf("expected 0 notes, got %d: %+v", len(notes), notes)
	}
	if _, ok := config.Agents["developer"]; !ok {
		t.Fatal("expected 'developer' agent to remain in config")
	}
}

// TestResolveTargetOverrides_TargetedResource_IncludedWhenMatch verifies that an
// agent listing the current target in its Targets map is kept and emits no notes.
func TestResolveTargetOverrides_TargetedResource_IncludedWhenMatch(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Agents = map[string]ast.AgentConfig{
		"developer": {
			Name:        "developer",
			Description: "test agent",
			Model:       "sonnet",
			Targets: map[string]ast.TargetOverride{
				"claude": {},
				"gemini": {},
			},
		},
	}

	notes := resolveTargetOverrides(config, "claude")

	if len(notes) != 0 {
		t.Fatalf("expected 0 notes, got %d: %+v", len(notes), notes)
	}
	if _, ok := config.Agents["developer"]; !ok {
		t.Fatal("expected 'developer' agent to remain in config")
	}
}

// TestResolveTargetOverrides_TargetedResource_SkippedWhenNoMatch verifies that an
// agent whose Targets map does not include the current target is removed and one
// warning note is emitted.
func TestResolveTargetOverrides_TargetedResource_SkippedWhenNoMatch(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Agents = map[string]ast.AgentConfig{
		"developer": {
			Name:        "developer",
			Description: "test agent",
			Model:       "sonnet",
			Targets: map[string]ast.TargetOverride{
				"gemini": {},
			},
		},
	}

	notes := resolveTargetOverrides(config, "claude")

	if len(notes) != 1 {
		t.Fatalf("expected 1 note, got %d: %+v", len(notes), notes)
	}
	note := notes[0]
	if note.Level != renderer.LevelWarning {
		t.Errorf("expected warning level, got %q", note.Level)
	}
	if note.Kind != "agent" {
		t.Errorf("expected kind 'agent', got %q", note.Kind)
	}
	if note.Resource != "developer" {
		t.Errorf("expected resource 'developer', got %q", note.Resource)
	}
	if note.Target != "claude" {
		t.Errorf("expected target 'claude', got %q", note.Target)
	}
	if note.Code != CodeResourceTargetSkipped {
		t.Errorf("expected code %q, got %q", CodeResourceTargetSkipped, note.Code)
	}
	if _, ok := config.Agents["developer"]; ok {
		t.Fatal("expected 'developer' agent to be removed from config")
	}
}

// TestResolveTargetOverrides_OverrideMerged verifies that a per-provider override
// is merged into the base resource config.
func TestResolveTargetOverrides_OverrideMerged(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Agents = map[string]ast.AgentConfig{
		"developer": {
			Name:        "developer",
			Description: "test agent",
			Model:       "sonnet",
		},
	}
	config.Overrides = &ast.ResourceOverrides{}
	config.Overrides.AddAgent("developer", "claude", ast.AgentConfig{Model: "opus"})

	notes := resolveTargetOverrides(config, "claude")

	if len(notes) != 0 {
		t.Fatalf("expected 0 notes, got %d: %+v", len(notes), notes)
	}
	agent, ok := config.Agents["developer"]
	if !ok {
		t.Fatal("expected 'developer' agent to remain in config")
	}
	if agent.Model != "opus" {
		t.Errorf("expected merged model 'opus', got %q", agent.Model)
	}
}

// TestResolveTargetOverrides_AfterBlueprintFilter_OnlyFilteredResources validates
// the architectural invariant that resolveTargetOverrides operates correctly on an
// already-narrowed config (as it would be after blueprint filtering). When the
// config contains only a single agent that declares the current target and carries
// a per-provider model override, the agent must be present after the call with the
// override merged in, and no fidelity notes must be emitted.
func TestResolveTargetOverrides_AfterBlueprintFilter_OnlyFilteredResources(t *testing.T) {
	// Simulate a config that has already been narrowed by a blueprint filter:
	// only one agent remains — "developer" — which is scoped to "claude".
	config := &ast.XcaffoldConfig{}
	config.Agents = map[string]ast.AgentConfig{
		"developer": {
			Name:        "developer",
			Description: "test agent",
			Model:       "sonnet",
			Targets: map[string]ast.TargetOverride{
				"claude": {},
			},
		},
	}
	config.Overrides = &ast.ResourceOverrides{}
	config.Overrides.AddAgent("developer", "claude", ast.AgentConfig{Model: "opus"})

	notes := resolveTargetOverrides(config, "claude")

	// No fidelity notes: developer is listed for claude.
	if len(notes) != 0 {
		t.Fatalf("expected 0 notes, got %d: %+v", len(notes), notes)
	}

	// Developer must still be present after the call.
	agent, ok := config.Agents["developer"]
	if !ok {
		t.Fatal("expected 'developer' agent to remain in config")
	}

	// The claude override (Model="opus") must have been merged in.
	if agent.Model != "opus" {
		t.Errorf("expected merged model %q, got %q", "opus", agent.Model)
	}
}

// TestResolveTargetOverrides_SkillSkippedWhenNoMatch verifies that a skill whose
// Targets map does not include the current target is removed and one warning note
// is emitted with kind "skill".
func TestResolveTargetOverrides_SkillSkippedWhenNoMatch(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Skills = map[string]ast.SkillConfig{
		"my-skill": {
			Name: "my-skill",
			Targets: map[string]ast.TargetOverride{
				"gemini": {},
			},
		},
	}

	notes := resolveTargetOverrides(config, "claude")

	if len(notes) != 1 {
		t.Fatalf("expected 1 note, got %d: %+v", len(notes), notes)
	}
	note := notes[0]
	if note.Level != renderer.LevelWarning {
		t.Errorf("expected warning level, got %q", note.Level)
	}
	if note.Kind != "skill" {
		t.Errorf("expected kind 'skill', got %q", note.Kind)
	}
	if note.Resource != "my-skill" {
		t.Errorf("expected resource 'my-skill', got %q", note.Resource)
	}
	if note.Target != "claude" {
		t.Errorf("expected target 'claude', got %q", note.Target)
	}
	if note.Code != CodeResourceTargetSkipped {
		t.Errorf("expected code %q, got %q", CodeResourceTargetSkipped, note.Code)
	}
	if _, ok := config.Skills["my-skill"]; ok {
		t.Fatal("expected 'my-skill' skill to be removed from config")
	}
}

// TestResolveTargetOverrides_RuleOverrideMerged verifies that a per-provider
// override for a rule is merged into the base rule config.
func TestResolveTargetOverrides_RuleOverrideMerged(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Rules = map[string]ast.RuleConfig{
		"security": {
			Name: "security",
			Body: "universal rules",
		},
	}
	config.Overrides = &ast.ResourceOverrides{}
	config.Overrides.AddRule("security", "claude", ast.RuleConfig{Body: "claude-specific rules"})

	notes := resolveTargetOverrides(config, "claude")

	if len(notes) != 0 {
		t.Fatalf("expected 0 notes, got %d: %+v", len(notes), notes)
	}
	rule, ok := config.Rules["security"]
	if !ok {
		t.Fatal("expected 'security' rule to remain in config")
	}
	if rule.Body != "claude-specific rules" {
		t.Errorf("expected merged Body %q, got %q", "claude-specific rules", rule.Body)
	}
}

// TestResolveTargetOverrides_Workflow_SkippedWhenTargetNotMatched verifies that a
// workflow whose Targets map does not include the current target is removed and
// one warning note is emitted with kind "workflow".
func TestResolveTargetOverrides_Workflow_SkippedWhenTargetNotMatched(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Workflows = map[string]ast.WorkflowConfig{
		"deploy": {
			Name:    "deploy",
			Targets: map[string]ast.TargetOverride{"claude": {}},
		},
	}
	notes := resolveTargetOverrides(config, "gemini")
	if _, ok := config.Workflows["deploy"]; ok {
		t.Fatal("deploy workflow should be removed — target gemini not in targets")
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1 fidelity note, got %d", len(notes))
	}
	note := notes[0]
	if note.Level != renderer.LevelWarning {
		t.Errorf("expected warning level, got %q", note.Level)
	}
	if note.Kind != "workflow" {
		t.Errorf("expected kind 'workflow', got %q", note.Kind)
	}
	if note.Resource != "deploy" {
		t.Errorf("expected resource 'deploy', got %q", note.Resource)
	}
	if note.Target != "gemini" {
		t.Errorf("expected target 'gemini', got %q", note.Target)
	}
}

// TestResolveTargetOverrides_MCP_OverrideMerged verifies that a per-provider
// override for an MCP server is merged into the base MCP config.
func TestResolveTargetOverrides_MCP_OverrideMerged(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.MCP = map[string]ast.MCPConfig{
		"server": {Name: "server", Command: "base-cmd"},
	}
	config.Overrides = &ast.ResourceOverrides{}
	config.Overrides.AddMCP("server", "claude", ast.MCPConfig{Command: "claude-cmd"})
	notes := resolveTargetOverrides(config, "claude")
	if config.MCP["server"].Command != "claude-cmd" {
		t.Fatalf("expected override command 'claude-cmd', got %q", config.MCP["server"].Command)
	}
	if len(notes) != 0 {
		t.Fatalf("expected 0 notes for matched override, got %d", len(notes))
	}
}

// TestResolveTargetOverrides_NilOverrides verifies that a nil config.Overrides
// does not panic and that universal resources pass through unchanged.
func TestResolveTargetOverrides_NilOverrides(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Overrides = nil // explicit nil
	config.Agents = map[string]ast.AgentConfig{
		"developer": {
			Name:        "developer",
			Description: "test agent",
			Model:       "sonnet",
		},
	}

	var notes []renderer.FidelityNote
	// Must not panic.
	notes = resolveTargetOverrides(config, "claude")

	if len(notes) != 0 {
		t.Fatalf("expected 0 notes, got %d: %+v", len(notes), notes)
	}
	if _, ok := config.Agents["developer"]; !ok {
		t.Fatal("expected 'developer' agent to remain in config")
	}
}

// TestResolveTargetOverrides_MCP_FilteredWhenTargetNotListed verifies that an
// MCP server whose Targets map does not include the current target is removed
// and one warning note is emitted with kind "mcp".
func TestResolveTargetOverrides_MCP_FilteredWhenTargetNotListed(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.MCP = map[string]ast.MCPConfig{
		"my-server": {
			Name: "my-server",
			Targets: map[string]ast.TargetOverride{
				"gemini": {},
			},
		},
	}

	notes := resolveTargetOverrides(config, "claude")

	if _, ok := config.MCP["my-server"]; ok {
		t.Fatal("expected 'my-server' MCP to be removed — target claude not in Targets")
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1 note, got %d: %+v", len(notes), notes)
	}
	note := notes[0]
	if note.Level != renderer.LevelWarning {
		t.Errorf("expected warning level, got %q", note.Level)
	}
	if note.Kind != "mcp" {
		t.Errorf("expected kind 'mcp', got %q", note.Kind)
	}
	if note.Resource != "my-server" {
		t.Errorf("expected resource 'my-server', got %q", note.Resource)
	}
	if note.Target != "claude" {
		t.Errorf("expected target 'claude', got %q", note.Target)
	}
	if note.Code != CodeResourceTargetSkipped {
		t.Errorf("expected code %q, got %q", CodeResourceTargetSkipped, note.Code)
	}
}

// TestResolveTargetOverrides_Hooks_OverrideApplied verifies that a per-provider
// override for a hook is merged into the base hook config.
func TestResolveTargetOverrides_Hooks_OverrideApplied(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Hooks = map[string]ast.NamedHookConfig{
		"my-hooks": {
			Name:        "my-hooks",
			Description: "base description",
		},
	}
	config.Overrides = &ast.ResourceOverrides{}
	config.Overrides.AddHooks("my-hooks", "claude", ast.NamedHookConfig{
		Description: "claude-specific description",
	})

	notes := resolveTargetOverrides(config, "claude")

	if len(notes) != 0 {
		t.Fatalf("expected 0 notes, got %d: %+v", len(notes), notes)
	}
	hook, ok := config.Hooks["my-hooks"]
	if !ok {
		t.Fatal("expected 'my-hooks' to remain in config")
	}
	if hook.Description != "claude-specific description" {
		t.Errorf("Description: want %q, got %q", "claude-specific description", hook.Description)
	}
}

// TestResolveTargetOverrides_Hooks_SkippedWhenTargetNotListed verifies that a hook
// whose Targets map does not include the current target is removed and one warning
// note is emitted with kind "hooks".
func TestResolveTargetOverrides_Hooks_SkippedWhenTargetNotListed(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Hooks = map[string]ast.NamedHookConfig{
		"my-hooks": {
			Name: "my-hooks",
			Targets: map[string]ast.TargetOverride{
				"gemini": {},
			},
		},
	}

	notes := resolveTargetOverrides(config, "claude")

	if _, ok := config.Hooks["my-hooks"]; ok {
		t.Fatal("expected 'my-hooks' to be removed — target claude not in Targets")
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1 note, got %d: %+v", len(notes), notes)
	}
	note := notes[0]
	if note.Level != renderer.LevelWarning {
		t.Errorf("expected warning level, got %q", note.Level)
	}
	if note.Kind != "hooks" {
		t.Errorf("expected kind 'hooks', got %q", note.Kind)
	}
	if note.Resource != "my-hooks" {
		t.Errorf("expected resource 'my-hooks', got %q", note.Resource)
	}
	if note.Target != "claude" {
		t.Errorf("expected target 'claude', got %q", note.Target)
	}
}

// TestResolveTargetOverrides_Settings_OverrideApplied verifies that a per-provider
// override for settings is merged into the base settings config.
func TestResolveTargetOverrides_Settings_OverrideApplied(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Settings = map[string]ast.SettingsConfig{
		"default": {
			Name:  "default",
			Model: "sonnet",
		},
	}
	config.Overrides = &ast.ResourceOverrides{}
	config.Overrides.AddSettings("default", "claude", ast.SettingsConfig{Model: "opus"})

	notes := resolveTargetOverrides(config, "claude")

	if len(notes) != 0 {
		t.Fatalf("expected 0 notes, got %d: %+v", len(notes), notes)
	}
	settings, ok := config.Settings["default"]
	if !ok {
		t.Fatal("expected 'default' settings to remain in config")
	}
	if settings.Model != "opus" {
		t.Errorf("Model: want %q, got %q", "opus", settings.Model)
	}
}

// TestResolveTargetOverrides_Settings_SkippedWhenTargetNotListed verifies that a
// settings block whose Targets map does not include the current target is removed
// and one warning note is emitted with kind "settings".
func TestResolveTargetOverrides_Settings_SkippedWhenTargetNotListed(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Settings = map[string]ast.SettingsConfig{
		"default": {
			Name: "default",
			Targets: map[string]ast.TargetOverride{
				"gemini": {},
			},
		},
	}

	notes := resolveTargetOverrides(config, "claude")

	if _, ok := config.Settings["default"]; ok {
		t.Fatal("expected 'default' settings to be removed — target claude not in Targets")
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1 note, got %d: %+v", len(notes), notes)
	}
	note := notes[0]
	if note.Kind != "settings" {
		t.Errorf("expected kind 'settings', got %q", note.Kind)
	}
}

// TestResolveTargetOverrides_Policy_OverrideApplied verifies that a per-provider
// override for a policy is merged into the base policy config. Policies have no
// Targets map so there is no filter step — only override application.
func TestResolveTargetOverrides_Policy_OverrideApplied(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Policies = map[string]ast.PolicyConfig{
		"my-policy": {
			Name:     "my-policy",
			Severity: "warning",
			Target:   "agent",
		},
	}
	config.Overrides = &ast.ResourceOverrides{}
	config.Overrides.AddPolicy("my-policy", "claude", ast.PolicyConfig{Severity: "error"})

	notes := resolveTargetOverrides(config, "claude")

	if len(notes) != 0 {
		t.Fatalf("expected 0 notes, got %d: %+v", len(notes), notes)
	}
	policy, ok := config.Policies["my-policy"]
	if !ok {
		t.Fatal("expected 'my-policy' to remain in config")
	}
	if policy.Severity != "error" {
		t.Errorf("Severity: want %q, got %q", "error", policy.Severity)
	}
}

// TestResolveTargetOverrides_Template_OverrideApplied verifies that a per-provider
// override for a template is merged into the base template config. Templates have
// no Targets field so there is no filter step — only override application.
func TestResolveTargetOverrides_Template_OverrideApplied(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Templates = map[string]ast.TemplateConfig{
		"my-template": {
			Name:          "my-template",
			DefaultTarget: "claude",
		},
	}
	config.Overrides = &ast.ResourceOverrides{}
	config.Overrides.AddTemplate("my-template", "claude", ast.TemplateConfig{DefaultTarget: "gemini"})

	notes := resolveTargetOverrides(config, "claude")

	if len(notes) != 0 {
		t.Fatalf("expected 0 notes, got %d: %+v", len(notes), notes)
	}
	tmpl, ok := config.Templates["my-template"]
	if !ok {
		t.Fatal("expected 'my-template' to remain in config")
	}
	if tmpl.DefaultTarget != "gemini" {
		t.Errorf("DefaultTarget: want %q, got %q", "gemini", tmpl.DefaultTarget)
	}
}

// TestResolveTargetOverrides_Context_OverrideApplied verifies that a per-provider
// override for a context is merged into the base context config. Context.Targets is
// []string (not a filter map) so there is no filter step — only override application.
func TestResolveTargetOverrides_Context_OverrideApplied(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Contexts = map[string]ast.ContextConfig{
		"my-context": {
			Name: "my-context",
			Body: "base body",
		},
	}
	config.Overrides = &ast.ResourceOverrides{}
	config.Overrides.AddContext("my-context", "claude", ast.ContextConfig{Body: "claude body"})

	notes := resolveTargetOverrides(config, "claude")

	if len(notes) != 0 {
		t.Fatalf("expected 0 notes, got %d: %+v", len(notes), notes)
	}
	ctx, ok := config.Contexts["my-context"]
	if !ok {
		t.Fatal("expected 'my-context' to remain in config")
	}
	if ctx.Body != "claude body" {
		t.Errorf("Body: want %q, got %q", "claude body", ctx.Body)
	}
}
