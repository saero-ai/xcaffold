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
			Name:  "developer",
			Model: "sonnet",
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
			Name:  "developer",
			Model: "sonnet",
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
			Name:  "developer",
			Model: "sonnet",
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
			Name:  "developer",
			Model: "sonnet",
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
			Name:  "developer",
			Model: "sonnet",
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
			Name:  "developer",
			Model: "sonnet",
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
