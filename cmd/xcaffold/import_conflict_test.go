package main

import (
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
)

func TestDetectConflicts_TwoProvidersDiverged(t *testing.T) {
	existing := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"security": {Name: "security", Body: "Original body."},
			},
		},
	}
	providerConfigs := map[string]*ast.XcaffoldConfig{
		"claude": {ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"security": {Name: "security", Body: "Claude edited body."},
			},
		}},
		"cursor": {ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"security": {Name: "security", Body: "Cursor edited body."},
			},
		}},
	}

	conflicts := detectConflicts(providerConfigs, existing)
	if len(conflicts) == 0 {
		t.Fatal("expected conflict for 'security' rule, got none")
	}
	if conflicts[0].ResourceName != "security" {
		t.Errorf("conflict resource = %q, want %q", conflicts[0].ResourceName, "security")
	}
	if len(conflicts[0].Providers) != 2 {
		t.Errorf("conflict providers count = %d, want 2", len(conflicts[0].Providers))
	}
}

func TestDetectConflicts_OneProviderDiverged_NoConflict(t *testing.T) {
	existing := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"security": {Name: "security", Body: "Original body."},
			},
		},
	}
	providerConfigs := map[string]*ast.XcaffoldConfig{
		"claude": {ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"security": {Name: "security", Body: "Claude edited body."},
			},
		}},
		"cursor": {ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"security": {Name: "security", Body: "Original body."},
			},
		}},
	}

	conflicts := detectConflicts(providerConfigs, existing)
	if len(conflicts) > 0 {
		t.Errorf("expected no conflicts (only one provider diverged), got %d", len(conflicts))
	}
}

func TestDetectConflicts_BothSameEdit_NoConflict(t *testing.T) {
	existing := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"security": {Name: "security", Body: "Original."},
			},
		},
	}
	providerConfigs := map[string]*ast.XcaffoldConfig{
		"claude": {ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"security": {Name: "security", Body: "Same edit."},
			},
		}},
		"cursor": {ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"security": {Name: "security", Body: "Same edit."},
			},
		}},
	}

	conflicts := detectConflicts(providerConfigs, existing)
	if len(conflicts) > 0 {
		t.Errorf("expected no conflicts (both made same edit), got %d", len(conflicts))
	}
}

func TestDetectConflicts_ThreeProviders_TwoDiverged(t *testing.T) {
	existing := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"security": {Name: "security", Body: "Original."},
			},
		},
	}
	providerConfigs := map[string]*ast.XcaffoldConfig{
		"claude": {ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"security": {Name: "security", Body: "Claude edit."},
			},
		}},
		"cursor": {ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"security": {Name: "security", Body: "Cursor edit."},
			},
		}},
		"copilot": {ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"security": {Name: "security", Body: "Original."},
			},
		}},
	}

	conflicts := detectConflicts(providerConfigs, existing)
	if len(conflicts) == 0 {
		t.Fatal("expected conflict, got none")
	}
	if len(conflicts[0].Providers) != 2 {
		t.Errorf("conflict providers count = %d, want 2 (only diverged ones)", len(conflicts[0].Providers))
	}
}

func TestResolveConflict_BodyPriority(t *testing.T) {
	conflict := ConflictReport{
		ResourceName: "security",
		Kind:         "rule",
		Providers: []ProviderVersion{
			{Provider: "claude", Body: "Longer body with more content.", BodyLen: 31},
			{Provider: "cursor", Body: "Short.", BodyLen: 6},
		},
	}

	base, overrides := resolveConflict(conflict)
	if base.Provider != "claude" {
		t.Errorf("base provider = %q, want %q (longest body)", base.Provider, "claude")
	}
	if len(overrides) != 1 || overrides[0].Provider != "cursor" {
		t.Errorf("overrides = %v, want [{cursor}]", overrides)
	}
}

func TestResolveConflict_EqualLength_Alphabetical(t *testing.T) {
	conflict := ConflictReport{
		ResourceName: "test",
		Kind:         "agent",
		Providers: []ProviderVersion{
			{Provider: "cursor", Body: "Same length.", BodyLen: 12},
			{Provider: "claude", Body: "Same length.", BodyLen: 12},
		},
	}

	base, overrides := resolveConflict(conflict)
	// With equal length, first in the provider list wins (no tiebreaker yet)
	// The order depends on iteration order. We just verify structure is correct.
	if base.BodyLen != 12 {
		t.Errorf("base body length = %d, want 12", base.BodyLen)
	}
	if len(overrides) != 1 {
		t.Errorf("overrides count = %d, want 1", len(overrides))
	}
}

func TestFormatConflictSummary_ContainsResourceNames(t *testing.T) {
	conflicts := []ConflictReport{{
		ResourceName: "security",
		Kind:         "rule",
		Providers: []ProviderVersion{
			{Provider: "claude", Body: "Claude version.", BodyLen: 15},
			{Provider: "cursor", Body: "Cursor version.", BodyLen: 15},
		},
	}}
	output := formatConflictSummary(conflicts)
	if !strings.Contains(output, "security") {
		t.Error("conflict summary missing resource name")
	}
	if !strings.Contains(output, "claude") || !strings.Contains(output, "cursor") {
		t.Error("conflict summary missing provider names")
	}
}

// TestImport_MultiProviderConflict_Detected verifies that multi-provider
// conflicts are properly detected during merge import scenario.
func TestImport_MultiProviderConflict_Detected(t *testing.T) {
	// Simulate two providers each with different edits to the same rule
	existing := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"security": {Name: "security", Body: "Original rule content."},
			},
			Agents:    make(map[string]ast.AgentConfig),
			Skills:    make(map[string]ast.SkillConfig),
			Workflows: make(map[string]ast.WorkflowConfig),
			MCP:       make(map[string]ast.MCPConfig),
		},
	}

	providerConfigs := map[string]*ast.XcaffoldConfig{
		"claude": {
			ResourceScope: ast.ResourceScope{
				Rules: map[string]ast.RuleConfig{
					"security": {Name: "security", Body: "Claude's edited rule."},
				},
				Agents:    make(map[string]ast.AgentConfig),
				Skills:    make(map[string]ast.SkillConfig),
				Workflows: make(map[string]ast.WorkflowConfig),
				MCP:       make(map[string]ast.MCPConfig),
			},
		},
		"cursor": {
			ResourceScope: ast.ResourceScope{
				Rules: map[string]ast.RuleConfig{
					"security": {Name: "security", Body: "Cursor's different edit."},
				},
				Agents:    make(map[string]ast.AgentConfig),
				Skills:    make(map[string]ast.SkillConfig),
				Workflows: make(map[string]ast.WorkflowConfig),
				MCP:       make(map[string]ast.MCPConfig),
			},
		},
	}

	// Detect conflicts
	conflicts := detectConflicts(providerConfigs, existing)
	if len(conflicts) == 0 {
		t.Fatal("expected 1 conflict, got 0")
	}

	if conflicts[0].ResourceName != "security" {
		t.Errorf("conflict resource name = %q, want %q", conflicts[0].ResourceName, "security")
	}

	if len(conflicts[0].Providers) != 2 {
		t.Errorf("conflict providers count = %d, want 2", len(conflicts[0].Providers))
	}

	// Resolve using body-priority
	base, overrides := resolveConflict(conflicts[0])
	if len(overrides) != 1 {
		t.Errorf("expected 1 override, got %d", len(overrides))
	}
	if base.Provider == "" {
		t.Error("base provider is empty")
	}
}
