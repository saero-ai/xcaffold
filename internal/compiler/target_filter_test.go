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
