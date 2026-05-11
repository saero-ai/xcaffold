package main

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
)

func TestDiffResources_NewResource(t *testing.T) {
	scanned := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"new-agent": {Name: "new-agent"},
			},
			Skills: make(map[string]ast.SkillConfig),
			Rules:  make(map[string]ast.RuleConfig),
			MCP:    make(map[string]ast.MCPConfig),
		},
	}
	existing := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: make(map[string]ast.AgentConfig),
			Skills: make(map[string]ast.SkillConfig),
			Rules:  make(map[string]ast.RuleConfig),
			MCP:    make(map[string]ast.MCPConfig),
		},
	}

	diff := diffResources(scanned, existing)

	if len(diff.New["agent"]) != 1 {
		t.Fatalf("expected 1 new agent, got %d", len(diff.New["agent"]))
	}
	if diff.New["agent"][0].Name != "new-agent" {
		t.Fatalf("expected name 'new-agent', got %q", diff.New["agent"][0].Name)
	}
	if diff.New["agent"][0].Kind != "agent" {
		t.Fatalf("expected kind 'agent', got %q", diff.New["agent"][0].Kind)
	}

	if diff.TotalNew() != 1 {
		t.Fatalf("expected TotalNew() = 1, got %d", diff.TotalNew())
	}
	if diff.TotalChanged() != 0 {
		t.Fatalf("expected TotalChanged() = 0, got %d", diff.TotalChanged())
	}
	if diff.TotalUnchanged() != 0 {
		t.Fatalf("expected TotalUnchanged() = 0, got %d", diff.TotalUnchanged())
	}
}

func TestDiffResources_UnchangedResource(t *testing.T) {
	agent := ast.AgentConfig{Name: "same-agent"}
	scanned := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"same-agent": agent,
			},
			Skills: make(map[string]ast.SkillConfig),
			Rules:  make(map[string]ast.RuleConfig),
			MCP:    make(map[string]ast.MCPConfig),
		},
	}
	existing := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"same-agent": agent,
			},
			Skills: make(map[string]ast.SkillConfig),
			Rules:  make(map[string]ast.RuleConfig),
			MCP:    make(map[string]ast.MCPConfig),
		},
	}

	diff := diffResources(scanned, existing)

	if len(diff.Unchanged["agent"]) != 1 {
		t.Fatalf("expected 1 unchanged agent, got %d", len(diff.Unchanged["agent"]))
	}
	if len(diff.New["agent"]) != 0 {
		t.Fatalf("expected 0 new agents, got %d", len(diff.New["agent"]))
	}

	if diff.TotalNew() != 0 {
		t.Fatalf("expected TotalNew() = 0, got %d", diff.TotalNew())
	}
	if diff.TotalUnchanged() != 1 {
		t.Fatalf("expected TotalUnchanged() = 1, got %d", diff.TotalUnchanged())
	}
}

func TestDiffResources_ChangedResource(t *testing.T) {
	scanned := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"changed-agent": {Name: "changed-agent", Description: "new description"},
			},
			Skills: make(map[string]ast.SkillConfig),
			Rules:  make(map[string]ast.RuleConfig),
			MCP:    make(map[string]ast.MCPConfig),
		},
	}
	existing := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"changed-agent": {Name: "changed-agent", Description: "old description"},
			},
			Skills: make(map[string]ast.SkillConfig),
			Rules:  make(map[string]ast.RuleConfig),
			MCP:    make(map[string]ast.MCPConfig),
		},
	}

	diff := diffResources(scanned, existing)

	if len(diff.Changed["agent"]) != 1 {
		t.Fatalf("expected 1 changed agent, got %d", len(diff.Changed["agent"]))
	}
	if diff.Changed["agent"][0].Name != "changed-agent" {
		t.Fatalf("expected name 'changed-agent', got %q", diff.Changed["agent"][0].Name)
	}

	if diff.TotalChanged() != 1 {
		t.Fatalf("expected TotalChanged() = 1, got %d", diff.TotalChanged())
	}
	if diff.TotalNew() != 0 {
		t.Fatalf("expected TotalNew() = 0, got %d", diff.TotalNew())
	}
}

func TestDiffResources_XcafOnlyResource(t *testing.T) {
	scanned := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: make(map[string]ast.AgentConfig),
			Skills: make(map[string]ast.SkillConfig),
			Rules:  make(map[string]ast.RuleConfig),
			MCP:    make(map[string]ast.MCPConfig),
		},
	}
	existing := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"xcaf-only": {Name: "xcaf-only"},
			},
			Skills: make(map[string]ast.SkillConfig),
			Rules:  make(map[string]ast.RuleConfig),
			MCP:    make(map[string]ast.MCPConfig),
		},
	}

	diff := diffResources(scanned, existing)

	if len(diff.XcafOnly["agent"]) != 1 {
		t.Fatalf("expected 1 xcaf-only agent, got %d", len(diff.XcafOnly["agent"]))
	}
	if diff.XcafOnly["agent"][0].Name != "xcaf-only" {
		t.Fatalf("expected name 'xcaf-only', got %q", diff.XcafOnly["agent"][0].Name)
	}

	if diff.TotalXcafOnly() != 1 {
		t.Fatalf("expected TotalXcafOnly() = 1, got %d", diff.TotalXcafOnly())
	}
}

func TestDiffResources_IgnoresSourceProvider(t *testing.T) {
	// Test that scanned and existing resources with identical fields except
	// SourceProvider are categorized as "Unchanged", not "Changed".
	// This is the RED phase test - it should fail until stripRuntimeFields is implemented.

	scannedAgent := ast.AgentConfig{
		Name:           "test-agent",
		Description:    "test description",
		SourceProvider: "claude", // This differs in scanned vs existing
		Inherited:      false,
		Body:           "some body",
	}

	existingAgent := ast.AgentConfig{
		Name:           "test-agent",
		Description:    "test description",
		SourceProvider: "", // Empty in existing (from xcaf/)
		Inherited:      false,
		Body:           "some body",
	}

	scanned := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"test-agent": scannedAgent,
			},
			Skills: make(map[string]ast.SkillConfig),
			Rules:  make(map[string]ast.RuleConfig),
			MCP:    make(map[string]ast.MCPConfig),
		},
	}

	existing := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"test-agent": existingAgent,
			},
			Skills: make(map[string]ast.SkillConfig),
			Rules:  make(map[string]ast.RuleConfig),
			MCP:    make(map[string]ast.MCPConfig),
		},
	}

	diff := diffResources(scanned, existing)

	// Without stripRuntimeFields, this currently fails (returns Changed instead of Unchanged)
	if len(diff.Changed["agent"]) != 0 {
		t.Fatalf("expected 0 changed agents (SourceProvider should be ignored), got %d", len(diff.Changed["agent"]))
	}
	if len(diff.Unchanged["agent"]) != 1 {
		t.Fatalf("expected 1 unchanged agent, got %d", len(diff.Unchanged["agent"]))
	}
	if diff.TotalChanged() != 0 {
		t.Fatalf("expected TotalChanged() = 0, got %d", diff.TotalChanged())
	}
	if diff.TotalUnchanged() != 1 {
		t.Fatalf("expected TotalUnchanged() = 1, got %d", diff.TotalUnchanged())
	}
}
