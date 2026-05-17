package main

import (
	"os"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/stretchr/testify/require"
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

// TestImport_Incremental_MergePreservesSourceFile verifies that mergeResourceDiff
// correctly merges provider edits into existing resources while preserving SourceFile.
func TestImport_Incremental_MergePreservesSourceFile(t *testing.T) {
	// Save original flag state
	originalDryRun := importDryRun
	originalYes := importYes
	defer func() {
		importDryRun = originalDryRun
		importYes = originalYes
	}()

	importDryRun = false
	importYes = true

	// Build existing config with two rules in flat layout
	existingConfig := &ast.XcaffoldConfig{
		Kind:    "project",
		Version: "1.0.0",
		ResourceScope: ast.ResourceScope{
			Agents:    make(map[string]ast.AgentConfig),
			Skills:    make(map[string]ast.SkillConfig),
			Rules:     make(map[string]ast.RuleConfig),
			Workflows: make(map[string]ast.WorkflowConfig),
			MCP:       make(map[string]ast.MCPConfig),
		},
	}

	// Add existing rules with SourceFile set (flat layout)
	existingConfig.Rules["no-secrets"] = ast.RuleConfig{
		Name:        "no-secrets",
		Description: "Security rule for SQL injection (old)",
		Body:        "Never use string concatenation for SQL queries.\n",
		SourceFile:  "xcaf/rules/no-secrets.xcaf",
	}
	existingConfig.Rules["formatting"] = ast.RuleConfig{
		Name:        "formatting",
		Description: "Code formatting rule",
		Body:        "Use 2-space indentation.\n",
		SourceFile:  "xcaf/rules/formatting.xcaf",
	}

	// Scanned config (from provider import) with updated no-secrets rule
	scannedConfig := &ast.XcaffoldConfig{
		Kind:    "project",
		Version: "1.0.0",
		ResourceScope: ast.ResourceScope{
			Agents:    make(map[string]ast.AgentConfig),
			Skills:    make(map[string]ast.SkillConfig),
			Rules:     make(map[string]ast.RuleConfig),
			Workflows: make(map[string]ast.WorkflowConfig),
			MCP:       make(map[string]ast.MCPConfig),
		},
	}

	// Updated rule with new description (will be marked as Changed in diff)
	scannedConfig.Rules["no-secrets"] = ast.RuleConfig{
		Name:        "no-secrets",
		Description: "Security rule for SQL injection (updated)",
		Body:        "Never use string concatenation for SQL queries.\nAlways use parameterized queries for SQL to prevent injection.\n",
		SourceFile:  "xcaf/rules/no-secrets.xcaf",
	}

	// Deep copy BOTH configs before diffing (diffResources modifies both in-place)
	scannedConfigCopy := deepCopyConfig(scannedConfig)
	existingConfigCopy := deepCopyConfig(existingConfig)

	// Compute diff - this will strip Body and SourceFile for comparison on the original objects
	diff := diffResources(scannedConfig, existingConfig)

	// Verify that diff correctly identifies "no-secrets" as changed
	// (based on Description difference after stripping runtime fields)
	if len(diff.Changed["rule"]) != 1 {
		t.Fatalf("expected 1 changed rule, got %d", len(diff.Changed["rule"]))
	}
	if diff.Changed["rule"][0].Name != "no-secrets" {
		t.Fatalf("expected changed rule 'no-secrets', got %q", diff.Changed["rule"][0].Name)
	}

	// Merge using the preserved configs
	mergeResourceDiff(existingConfigCopy, scannedConfigCopy, diff)

	// After merge, existingConfigCopy should have updated rule from scanned
	updatedRule, ok := existingConfigCopy.Rules["no-secrets"]
	if !ok {
		t.Fatal("rule 'no-secrets' missing after merge")
	}

	// Check that the updated content was merged
	if updatedRule.Description != "Security rule for SQL injection (updated)" {
		t.Errorf("rule description not updated; expected updated, got: %q", updatedRule.Description)
	}
	if updatedRule.Body != "Never use string concatenation for SQL queries.\nAlways use parameterized queries for SQL to prevent injection.\n" {
		t.Errorf("rule body not updated; got: %q", updatedRule.Body)
	}

	// SourceFile should be preserved during merge
	if updatedRule.SourceFile != "xcaf/rules/no-secrets.xcaf" {
		t.Errorf("SourceFile lost during merge; expected xcaf/rules/no-secrets.xcaf, got %q", updatedRule.SourceFile)
	}

	// Unchanged rule should remain in existingConfigCopy unchanged
	unchangedRule, ok := existingConfigCopy.Rules["formatting"]
	if !ok {
		t.Fatal("rule 'formatting' missing after merge")
	}
	if unchangedRule.Body != "Use 2-space indentation.\n" {
		t.Errorf("unchanged rule was modified; got: %q", unchangedRule.Body)
	}
	if unchangedRule.SourceFile != "xcaf/rules/formatting.xcaf" {
		t.Errorf("formatting rule SourceFile changed; expected xcaf/rules/formatting.xcaf, got %q", unchangedRule.SourceFile)
	}
}

// TestImport_Incremental_AbsorbsRenderedEditsToDisk verifies that the
// rewriteChangedResourcesInPlace function correctly writes resources back
// to their FLAT layout xcaf files (bug fixes B and C).
func TestImport_Incremental_AbsorbsRenderedEditsToDisk(t *testing.T) {
	// Create a temp directory for the test project
	tmpDir := t.TempDir()
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir to temp dir: %v", err)
	}
	defer os.Chdir(oldCwd)

	// Build FLAT-layout project directory
	if err := os.MkdirAll("xcaf/rules", 0755); err != nil {
		t.Fatalf("failed to create xcaf/rules: %v", err)
	}

	// Create an existing xcaf/rules/no-secrets.xcaf (simulates pre-existing source)
	originalNoSecretsXcaf := `---
kind: rule
version: "1.0"
name: no-secrets
description: "Security rule for SQL injection"
activation: always
always-apply: true
---
Never use string concatenation for SQL queries.
`
	if err := os.WriteFile("xcaf/rules/no-secrets.xcaf", []byte(originalNoSecretsXcaf), 0644); err != nil {
		t.Fatalf("failed to write xcaf/rules/no-secrets.xcaf: %v", err)
	}

	// Parse the existing config
	existingConfig, err := parser.ParseDirectory(".")
	if err != nil {
		t.Fatalf("failed to parse xcaf/: %v", err)
	}

	// Manually set SourceFile (in real usage, this would be set by the parser;
	// we're testing the rewrite logic, not the parser)
	noSecretsRule, ok := existingConfig.Rules["no-secrets"]
	require.True(t, ok, "no-secrets rule not found in parsed config")
	noSecretsRule.SourceFile = "xcaf/rules/no-secrets.xcaf"
	existingConfig.Rules["no-secrets"] = noSecretsRule

	// Now simulate incremental import: create a modified version of the rule
	// (as would be extracted from rendered .claude/rules/no-secrets.md after user edit)
	alwaysApplyTrue := true
	scannedConfig := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"no-secrets": {
					Name:        "no-secrets",
					Description: "Security rule for SQL injection (updated)",
					Body:        "Never use string concatenation for SQL queries.\nAlways use parameterized queries for SQL to prevent injection.\n",
					Activation:  "always",
					AlwaysApply: &alwaysApplyTrue,
				},
			},
		},
	}

	// Create a diff showing this rule as changed
	diff := ResourceDiff{
		Changed: map[string][]DiffEntry{
			"rule": {{Kind: "rule", Name: "no-secrets"}},
		},
	}

	// Copy SourceFile from existing to scanned (as incremental import does)
	scannedRule := scannedConfig.Rules["no-secrets"]
	scannedRule.SourceFile = noSecretsRule.SourceFile
	scannedConfig.Rules["no-secrets"] = scannedRule

	// Now test rewriteChangedResourcesInPlace - the key function being tested
	if err := rewriteChangedResourcesInPlace(scannedConfig, diff); err != nil {
		t.Fatalf("rewriteChangedResourcesInPlace failed: %v", err)
	}

	// ASSERTIONS: Check the rewrite worked correctly
	// a. xcaf/rules/no-secrets.xcaf should contain "parameterized queries" (the new edit)
	rewrittenBytes, err := os.ReadFile("xcaf/rules/no-secrets.xcaf")
	require.NoError(t, err, "failed to read rewritten xcaf/rules/no-secrets.xcaf")

	rewrittenStr := string(rewrittenBytes)
	require.True(t, strings.Contains(rewrittenStr, "parameterized queries"),
		"FAIL a: edit not written to xcaf/rules/no-secrets.xcaf; got: %s", rewrittenStr)

	// b. xcaf/rules/no-secrets/rule.xcaf should NOT exist (we use flat layout, not nested)
	_, err = os.Stat("xcaf/rules/no-secrets/rule.xcaf")
	require.True(t, os.IsNotExist(err),
		"FAIL b: nested layout created when flat layout should be used")

	// c. Verify the rewritten file has the correct frontmatter (kind, version, name)
	require.True(t, strings.Contains(rewrittenStr, "kind: rule"),
		"FAIL c: kind not in rewritten file")
	require.True(t, strings.Contains(rewrittenStr, `name: no-secrets`),
		"FAIL c: name not in rewritten file")
	// The description should match what was in scanned config (updated version)
	require.True(t, strings.Contains(rewrittenStr, "Security rule for SQL injection (updated)"),
		"FAIL c: updated description not in rewritten file; got: %s", rewrittenStr)
}

// contains checks if haystack contains needle substring
func contains(haystack, needle string) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
