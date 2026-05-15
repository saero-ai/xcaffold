package main

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
)

// TestIncrementalImport_KindFilter_AgentOnly_DiffContainsOnlyAgents verifies that
// when --agent is set, the diff only contains agent entries.
func TestIncrementalImport_KindFilter_AgentOnly_DiffContainsOnlyAgents(t *testing.T) {
	// Save original filter state
	originalAgent := importFilterAgent
	originalSkill := importFilterSkill
	originalRule := importFilterRule
	defer func() {
		importFilterAgent = originalAgent
		importFilterSkill = originalSkill
		importFilterRule = originalRule
	}()

	// Simulate --agent * flag (include all agents, exclude others)
	importFilterAgent = "*"
	importFilterSkill = ""
	importFilterRule = ""

	scannedConfig := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {Description: "test agent"},
			},
			Skills: map[string]ast.SkillConfig{
				"tdd": {},
			},
			Rules: map[string]ast.RuleConfig{
				"security": {},
			},
		},
	}

	existingConfig := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents:    make(map[string]ast.AgentConfig),
			Skills:    make(map[string]ast.SkillConfig),
			Rules:     make(map[string]ast.RuleConfig),
			Workflows: make(map[string]ast.WorkflowConfig),
			MCP:       make(map[string]ast.MCPConfig),
		},
	}

	// Apply filters to scanned config BEFORE diffing
	applyKindFilters(scannedConfig)

	diff := diffResources(scannedConfig, existingConfig)

	// Verify diff only contains agents
	if len(diff.New["agent"]) != 1 {
		t.Errorf("expected 1 agent in diff.New, got %d", len(diff.New["agent"]))
	}
	if len(diff.New["skill"]) != 0 {
		t.Errorf("expected 0 skills in diff.New, got %d", len(diff.New["skill"]))
	}
	if len(diff.New["rule"]) != 0 {
		t.Errorf("expected 0 rules in diff.New, got %d", len(diff.New["rule"]))
	}
}

// TestIncrementalImport_KindFilter_MultiKind_DiffContainsOnlyRequested verifies that
// when multiple kind filters are set, the diff only contains those kinds.
func TestIncrementalImport_KindFilter_MultiKind_DiffContainsOnlyRequested(t *testing.T) {
	// Save original filter state
	originalAgent := importFilterAgent
	originalSkill := importFilterSkill
	originalRule := importFilterRule
	defer func() {
		importFilterAgent = originalAgent
		importFilterSkill = originalSkill
		importFilterRule = originalRule
	}()

	// Simulate --agent * --skill * flags
	importFilterAgent = "*"
	importFilterSkill = "*"
	importFilterRule = ""

	scannedConfig := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {Description: "test agent"},
			},
			Skills: map[string]ast.SkillConfig{
				"tdd": {},
			},
			Rules: map[string]ast.RuleConfig{
				"security": {},
			},
		},
	}

	existingConfig := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents:    make(map[string]ast.AgentConfig),
			Skills:    make(map[string]ast.SkillConfig),
			Rules:     make(map[string]ast.RuleConfig),
			Workflows: make(map[string]ast.WorkflowConfig),
			MCP:       make(map[string]ast.MCPConfig),
		},
	}

	// Apply filters to scanned config BEFORE diffing
	applyKindFilters(scannedConfig)

	diff := diffResources(scannedConfig, existingConfig)

	// Verify diff contains agents and skills but not rules
	if len(diff.New["agent"]) != 1 {
		t.Errorf("expected 1 agent in diff.New, got %d", len(diff.New["agent"]))
	}
	if len(diff.New["skill"]) != 1 {
		t.Errorf("expected 1 skill in diff.New, got %d", len(diff.New["skill"]))
	}
	if len(diff.New["rule"]) != 0 {
		t.Errorf("expected 0 rules in diff.New, got %d", len(diff.New["rule"]))
	}
}

// TestIncrementalImport_KindFilter_NamedAgent_DiffContainsOnlyNamed verifies that
// when a specific agent name is provided via --agent <name>, the diff only contains
// that specific agent.
func TestIncrementalImport_KindFilter_NamedAgent_DiffContainsOnlyNamed(t *testing.T) {
	// Save original filter state
	originalAgent := importFilterAgent
	originalSkill := importFilterSkill
	defer func() {
		importFilterAgent = originalAgent
		importFilterSkill = originalSkill
	}()

	// Simulate --agent dev flag (specific agent name)
	importFilterAgent = "dev"
	importFilterSkill = ""

	scannedConfig := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev":      {Description: "test agent"},
				"reviewer": {Description: "test agent"},
			},
			Skills: map[string]ast.SkillConfig{
				"tdd": {},
			},
		},
	}

	existingConfig := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents:    make(map[string]ast.AgentConfig),
			Skills:    make(map[string]ast.SkillConfig),
			Rules:     make(map[string]ast.RuleConfig),
			Workflows: make(map[string]ast.WorkflowConfig),
			MCP:       make(map[string]ast.MCPConfig),
		},
	}

	// Apply filters to scanned config BEFORE diffing
	applyKindFilters(scannedConfig)

	diff := diffResources(scannedConfig, existingConfig)

	// Verify diff contains only the "dev" agent
	agentEntries := diff.New["agent"]
	if len(agentEntries) != 1 {
		t.Errorf("expected 1 agent in diff.New, got %d", len(agentEntries))
	}
	if len(agentEntries) > 0 && agentEntries[0].Name != "dev" {
		t.Errorf("expected agent 'dev', got %q", agentEntries[0].Name)
	}
	if len(diff.New["skill"]) != 0 {
		t.Errorf("expected 0 skills in diff.New, got %d", len(diff.New["skill"]))
	}
}

// TestConfirmAndExecuteImport_DryRun_DoesNotCallWrite verifies that with
// --dry-run, the write function is never called.
func TestConfirmAndExecuteImport_DryRun_DoesNotCallWrite(t *testing.T) {
	// Save original state
	originalDryRun := importDryRun
	originalYes := importYes
	defer func() {
		importDryRun = originalDryRun
		importYes = originalYes
	}()

	importDryRun = true
	importYes = false

	ctx := incrementalImportCtx{
		xcafDest:  "project.xcaf",
		scopeName: "project",
		config:    &ast.XcaffoldConfig{},
	}

	diff := ResourceDiff{
		New: map[string][]DiffEntry{
			"agent": {
				{Kind: "agent", Name: "dev"},
			},
		},
	}

	writeCalled := false
	writeFunc := func() error {
		writeCalled = true
		return nil
	}

	err := confirmAndExecuteImport(ctx, diff, writeFunc)
	if err != nil {
		t.Fatalf("confirmAndExecuteImport returned error: %v", err)
	}

	if writeCalled {
		t.Error("write function should not be called with --dry-run")
	}
}
