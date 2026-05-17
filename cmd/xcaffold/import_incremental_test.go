package main

import (
	"os"
	"path/filepath"
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
	if err := rewriteChangedResourcesInPlace(scannedConfig, diff, ""); err != nil {
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

// TestImport_RoundTrip_NoFalsePositives verifies that apply → import → diff
// produces zero changes for a project with skills containing allowed-tools and agents
// with mcpServers. This tests the complete round-trip fidelity for normalized fields.
func TestImport_RoundTrip_NoFalsePositives(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()

	orig, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(orig) }()
	require.NoError(t, os.Chdir(tmp))

	// Setup: Create .claude/ with skills and agents
	setupRoundTripProject(t, tmp)

	// Step 1: Apply - render to .claude/ directory
	applyInDir(t, tmp)

	// Step 2: Import - read back from .claude/ directory
	importInDir(t, tmp, "claude")

	// Step 3: Diff - check for changes (should be zero)
	diff := diffInDir(t, tmp, "claude")

	// Verify: Zero false positives
	require.Equal(t, 0, diff.TotalChanged(),
		"Expected zero changes after round-trip (apply → import → diff), but got %d changed resources",
		diff.TotalChanged())
}

// setupRoundTripProject creates a minimal .claude/ directory with skills and agents
// that contain fields prone to normalization issues.
func setupRoundTripProject(t *testing.T, tmp string) {
	t.Helper()

	// Create .claude/skills/test-skill/ with allowed-tools
	skillDir := filepath.Join(tmp, ".claude", "skills", "test-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))

	skillContent := `---
name: test-skill
description: Test skill
allowed-tools:
  - Bash
  - Read
  - Write
when-to-use: Use this for testing.
---

# Test Skill

This skill demonstrates normalization of allowed-tools field.
`
	require.NoError(t, os.WriteFile(
		filepath.Join(skillDir, "SKILL.md"),
		[]byte(skillContent),
		0600,
	))

	// Create .claude/agents/test-agent/ with mcpServers
	agentsDir := filepath.Join(tmp, ".claude", "agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0755))

	agentContent := `---
name: test-agent
description: Test agent
model: claude-opus-4-5
mcpServers:
  my-server:
    command: node
    args:
      - server.js
---

# Test Agent

This agent demonstrates normalization of mcpServers field.
`
	require.NoError(t, os.WriteFile(
		filepath.Join(agentsDir, "test-agent.md"),
		[]byte(agentContent),
		0600,
	))

	// Create .claude/settings.json
	require.NoError(t, os.WriteFile(
		filepath.Join(tmp, ".claude", "settings.json"),
		[]byte("{}"),
		0600,
	))
}

// applyInDir runs the apply command in a specific directory.
// This renders the current project.xcaf to provider-specific formats.
func applyInDir(t *testing.T, dir string) {
	t.Helper()

	// First, import from .claude/ to create project.xcaf
	importInDir(t, dir, "claude")

	// Then apply to re-render to .claude/
	originalApplyBlueprintFlag := applyBlueprintFlag
	originalApplyGlobalFlag := globalFlag
	originalApplyTargetFlag := targetFlag
	originalProjectRoot := projectRoot

	defer func() {
		applyBlueprintFlag = originalApplyBlueprintFlag
		globalFlag = originalApplyGlobalFlag
		targetFlag = originalApplyTargetFlag
		projectRoot = originalProjectRoot
	}()

	applyBlueprintFlag = ""
	globalFlag = false
	targetFlag = "claude"
	projectRoot = dir
	xcafPath = filepath.Join(dir, "project.xcaf")

	// Remove .claude to start fresh for apply
	_ = os.RemoveAll(filepath.Join(dir, ".claude"))

	// Run apply
	err := applyScope(filepath.Join(dir, "project.xcaf"), filepath.Join(dir, ".claude"), dir, "project")
	require.NoError(t, err)
}

// importInDir runs the import command in a specific directory.
// This creates or updates project.xcaf from provider-specific directories.
func importInDir(t *testing.T, dir string, provider string) {
	t.Helper()

	originalImportDryRun := importDryRun
	originalImportForce := importForce
	originalImportYes := importYes
	originalProjectRoot := projectRoot

	defer func() {
		importDryRun = originalImportDryRun
		importForce = originalImportForce
		importYes = originalImportYes
		projectRoot = originalProjectRoot
	}()

	importDryRun = false
	importForce = true
	importYes = true
	projectRoot = dir

	// Run importScope for the specified provider
	err := importScope("."+provider, "project.xcaf", "project", provider)
	require.NoError(t, err)
}

// diffInDir runs the diff command and returns the results.
func diffInDir(t *testing.T, dir string, provider string) ResourceDiff {
	t.Helper()

	originalProjectRoot := projectRoot
	defer func() { projectRoot = originalProjectRoot }()

	projectRoot = dir

	// Parse the current project.xcaf
	config, err := parser.ParseDirectory(dir, parser.WithSkipGlobal())
	require.NoError(t, err)

	// Scan the provider directory
	imp := findImporterByProvider(provider)
	require.NotNil(t, imp)

	scanned := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents:    make(map[string]ast.AgentConfig),
			Skills:    make(map[string]ast.SkillConfig),
			Rules:     make(map[string]ast.RuleConfig),
			Workflows: make(map[string]ast.WorkflowConfig),
			MCP:       make(map[string]ast.MCPConfig),
		},
	}
	err = imp.Import("."+provider, scanned)
	require.NoError(t, err)

	// Diff
	diff := diffResources(scanned, config)
	return diff
}

// setupFlatProject creates a flat-layout project structure with one rule
// in xcaf/rules/ directory instead of nested subdirectories.
func setupFlatProject(t *testing.T, dir string) {
	t.Helper()
	xcafDir := filepath.Join(dir, "xcaf")
	rulesDir := filepath.Join(xcafDir, "rules")
	require.NoError(t, os.MkdirAll(rulesDir, 0755))

	// Create a flat-layout rule file
	ruleContent := `---
kind: rule
version: 1.0
name: existing-rule
---
Existing rule body
`
	require.NoError(t, os.WriteFile(
		filepath.Join(rulesDir, "existing-rule.xcaf"),
		[]byte(ruleContent),
		0644,
	))

	// Create a project.xcaf
	projectContent := `---
kind: config
version: 1.0
---
rules:
  existing-rule:
    name: existing-rule
    sourceFile: xcaf/rules/existing-rule.xcaf
`
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "project.xcaf"),
		[]byte(projectContent),
		0644,
	))
}

// TestImport_NewResourceUsesDetectedFlatLayout verifies that new resources
// (without SourceFile set) are written using the detected layout.
// In a flat project, new rules should go to xcaf/rules/name.xcaf,
// not xcaf/rules/name/rule.xcaf (nested).
func TestImport_NewResourceUsesDetectedFlatLayout(t *testing.T) {
	dir := t.TempDir()
	setupFlatProject(t, dir)

	// Save original working directory and change to test directory
	oldCwd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldCwd)
	require.NoError(t, os.Chdir(dir))

	xcafDir := filepath.Join(dir, "xcaf")

	// Create a NEW rule (no SourceFile) and rewrite it
	newRule := ast.RuleConfig{
		Name: "new-rule",
		Body: "---\nkind: rule\n---\nNew rule body",
	}

	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"new-rule": newRule,
			},
		},
	}

	// Detect layout and rewrite the new resource
	layout := detectLayout(xcafDir, "rule")
	require.Equal(t, layoutFlat, layout, "project should have flat layout")

	err = rewriteResourceInPlace(cfg, "rule", "new-rule", rewriteOpts{layout: layout})
	require.NoError(t, err)

	// Assert: file exists at flat location
	flatPath := filepath.Join(dir, "xcaf", "rules", "new-rule.xcaf")
	_, err = os.Stat(flatPath)
	require.NoError(t, err, "expected flat layout file at %q", flatPath)

	// Verify file content
	data, err := os.ReadFile(flatPath)
	require.NoError(t, err)
	require.Contains(t, string(data), "kind: rule", "file should contain rule metadata")

	// Assert: nested directory was NOT created
	nestedPath := filepath.Join(dir, "xcaf", "rules", "new-rule", "rule.xcaf")
	_, err = os.Stat(nestedPath)
	require.True(t, os.IsNotExist(err), "nested layout file should not exist at %q", nestedPath)
}

// TestImport_NewResourceUsesDetectedNestedLayout verifies that new resources
// in a nested-layout project are written to the nested structure.
func TestImport_NewResourceUsesDetectedNestedLayout(t *testing.T) {
	dir := t.TempDir()

	// Save original working directory and change to test directory
	oldCwd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldCwd)
	require.NoError(t, os.Chdir(dir))

	xcafDir := filepath.Join(dir, "xcaf")
	rulesDir := filepath.Join(xcafDir, "rules")

	// Create a nested-layout structure
	nestedDir := filepath.Join(rulesDir, "existing-rule")
	require.NoError(t, os.MkdirAll(nestedDir, 0755))

	ruleContent := `---
kind: rule
version: 1.0
name: existing-rule
---
Existing rule body
`
	require.NoError(t, os.WriteFile(
		filepath.Join(nestedDir, "rule.xcaf"),
		[]byte(ruleContent),
		0644,
	))

	// Create a NEW rule (no SourceFile) and rewrite it
	newRule := ast.RuleConfig{
		Name: "new-rule",
		Body: "---\nkind: rule\n---\nNew rule body",
	}

	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"new-rule": newRule,
			},
		},
	}

	// Detect layout and rewrite the new resource
	layout := detectLayout(xcafDir, "rule")
	require.Equal(t, layoutNested, layout, "project should have nested layout")

	err = rewriteResourceInPlace(cfg, "rule", "new-rule", rewriteOpts{layout: layout})
	require.NoError(t, err)

	// Assert: file exists at nested location
	nestedPath := filepath.Join(dir, "xcaf", "rules", "new-rule", "rule.xcaf")
	_, err = os.Stat(nestedPath)
	require.NoError(t, err, "expected nested layout file at %q", nestedPath)

	// Assert: flat file was NOT created
	flatPath := filepath.Join(dir, "xcaf", "rules", "new-rule.xcaf")
	_, err = os.Stat(flatPath)
	require.True(t, os.IsNotExist(err), "flat layout file should not exist at %q", flatPath)
}

// TestImport_TargetEndToEnd_ThreadsFromCLI verifies that the --target CLI flag
// threads through incrementalImport to rewriteChangedResourcesInPlace for override routing.
// When target="claude" and a rule is detected as changed, it should create or skip
// an override file based on deduplication.
func TestImport_TargetEndToEnd_ThreadsFromCLI(t *testing.T) {
	// Save original flag state
	originalImportDryRun := importDryRun
	originalImportForce := importForce
	originalImportYes := importYes
	originalImportTargetFlag := importTargetFlag
	defer func() {
		importDryRun = originalImportDryRun
		importForce = originalImportForce
		importYes = originalImportYes
		importTargetFlag = originalImportTargetFlag
	}()

	importDryRun = false
	importForce = false
	importYes = true
	importTargetFlag = "claude"

	dir := t.TempDir()

	// Setup: Create a nested project structure with a base rule
	xcafDir := filepath.Join(dir, "xcaf")
	rulesSecurityDir := filepath.Join(xcafDir, "rules", "security")
	require.NoError(t, os.MkdirAll(rulesSecurityDir, 0755))

	// Base rule file
	baseRule := `---
kind: rule
version: 1.0
name: security
description: Base security rule
---
Base rule content`
	require.NoError(t, os.WriteFile(
		filepath.Join(rulesSecurityDir, "rule.xcaf"),
		[]byte(baseRule),
		0644,
	))

	// Create a project.xcaf
	projectContent := `---
kind: config
version: 1.0
---
rules:
  security:
    name: security
    sourceFile: xcaf/rules/security/rule.xcaf
`
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "project.xcaf"),
		[]byte(projectContent),
		0644,
	))

	// Save original CWD and chdir
	oldCwd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldCwd)
	require.NoError(t, os.Chdir(dir))

	// Simulate import with target="claude" — create a mock .claude/rules/ with a different rule
	platformDir := filepath.Join(dir, ".claude")
	rulesDir := filepath.Join(platformDir, "rules")
	require.NoError(t, os.MkdirAll(rulesDir, 0755))

	// Different version of the rule (as if Claude rendered it differently)
	claudeRule := `---
name: security
description: Claude-specific security rule
activation: always
---
Claude-specific rule content`
	require.NoError(t, os.WriteFile(
		filepath.Join(rulesDir, "security.md"),
		[]byte(claudeRule),
		0644,
	))

	// Create settings.json (required by importers)
	require.NoError(t, os.WriteFile(
		filepath.Join(platformDir, "settings.json"),
		[]byte("{}"),
		0644,
	))

	// Call incrementalImport with target="claude"
	err = incrementalImport(platformDir, "project.xcaf", "project", "claude")
	require.NoError(t, err, "incrementalImport failed")

	// Verify the result: the rule should be in xcaf/rules/security/rule.xcaf with updated content
	updatedPath := filepath.Join(dir, "xcaf", "rules", "security", "rule.xcaf")
	data, err := os.ReadFile(updatedPath)
	require.NoError(t, err, "expected updated rule file at %q", updatedPath)

	// The file should contain the updated content from Claude
	require.Contains(t, string(data), "Claude-specific", "expected Claude-specific content in updated rule")
}

// TestImport_Deduplication_DirectCheck verifies the deduplication logic works
// when comparing identical rule bodies.
func TestImport_Deduplication_DirectCheck(t *testing.T) {
	dir := t.TempDir()
	oldCwd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldCwd)
	require.NoError(t, os.Chdir(dir))

	// Create a rule file
	require.NoError(t, os.MkdirAll("xcaf/rules", 0755))
	ruleContent := `---
kind: rule
version: "1.0"
name: security
description: Test rule
---
Test body`
	require.NoError(t, os.WriteFile("xcaf/rules/security.xcaf", []byte(ruleContent), 0644))

	// Create a rule config with same body but different description
	// The deduplication should only compare body content, not metadata
	rule := ast.RuleConfig{
		Name:        "security",
		Description: "Updated description",
		Body:        "Test body",
		SourceFile:  "xcaf/rules/security.xcaf",
	}

	// Test deduplication function - should return true because body is unchanged
	isUnchanged := isRuleContentUnchanged(rule)
	require.True(t, isUnchanged, "rule body should be detected as unchanged")
}

// TestImport_Deduplication_ChangedBody verifies deduplication detects when body changes.
func TestImport_Deduplication_ChangedBody(t *testing.T) {
	dir := t.TempDir()
	oldCwd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldCwd)
	require.NoError(t, os.Chdir(dir))

	// Create a rule file
	require.NoError(t, os.MkdirAll("xcaf/rules", 0755))
	ruleContent := `---
kind: rule
version: "1.0"
name: security
---
Original body`
	require.NoError(t, os.WriteFile("xcaf/rules/security.xcaf", []byte(ruleContent), 0644))

	// Create a rule config with changed body
	rule := ast.RuleConfig{
		Name:       "security",
		Body:       "Updated body",
		SourceFile: "xcaf/rules/security.xcaf",
	}

	// Test deduplication function - should return false because body changed
	isUnchanged := isRuleContentUnchanged(rule)
	require.False(t, isUnchanged, "rule body should be detected as changed")
}
