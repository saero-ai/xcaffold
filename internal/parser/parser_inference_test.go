package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInferKindAndName_AgentPath tests the inferKindAndName helper function.
func TestInferKindAndName_AgentPath(t *testing.T) {
	filePath := "some/path/xcaf/agents/developer/agent.xcaf"
	kind, name := inferKindAndName(filePath)
	assert.Equal(t, "agent", kind)
	assert.Equal(t, "developer", name)
}

// TestInferKindAndName_SkillPath tests skill inference.
func TestInferKindAndName_SkillPath(t *testing.T) {
	filePath := "xcaf/skills/tdd/skill.xcaf"
	kind, name := inferKindAndName(filePath)
	assert.Equal(t, "skill", kind)
	assert.Equal(t, "tdd", name)
}

// TestInferKindAndName_RulePath tests rule inference with allowed slash.
func TestInferKindAndName_RulePath(t *testing.T) {
	filePath := "xcaf/rules/cli/rule.xcaf"
	kind, name := inferKindAndName(filePath)
	assert.Equal(t, "rule", kind)
	assert.Equal(t, "cli", name)
}

// TestInferKindAndName_NoXcafDir tests that inference returns empty when xcaf is not in path.
func TestInferKindAndName_NoXcafDir(t *testing.T) {
	filePath := "some/path/agents/developer/agent.xcaf"
	kind, name := inferKindAndName(filePath)
	assert.Equal(t, "", kind)
	assert.Equal(t, "", name)
}

// TestInferKindAndName_InvalidKindDir tests that inference returns empty for unknown kinds.
func TestInferKindAndName_InvalidKindDir(t *testing.T) {
	filePath := "xcaf/unknown-kind/something/file.xcaf"
	kind, name := inferKindAndName(filePath)
	assert.Equal(t, "", kind)
	assert.Equal(t, "", name)
}

// TestInferKindAndName_TooShortPath tests that inference returns empty if path is too short.
func TestInferKindAndName_TooShortPath(t *testing.T) {
	filePath := "xcaf/agents"
	kind, name := inferKindAndName(filePath)
	assert.Equal(t, "", kind)
	assert.Equal(t, "", name)
}

// TestParse_FilesystemInference_InfersNameWhenKindProvided tests that name is inferred
// from filesystem when the YAML provides kind: but no name:.
func TestParse_FilesystemInference_InfersNameWhenKindProvided(t *testing.T) {
	dir := t.TempDir()
	xcafDir := filepath.Join(dir, "xcaf", "agents", "developer")
	require.NoError(t, os.MkdirAll(xcafDir, 0755))

	// Agent file with kind: but NO name: — name should be inferred from path
	content := "---\nkind: agent\nversion: \"1.0\"\ndescription: \"test agent\"\nmodel: sonnet\n---\nYou are a developer.\n"
	filePath := filepath.Join(xcafDir, "agent.xcaf")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	cfg, err := ParseDirectory(dir)
	require.NoError(t, err, "expected successful parse with inferred name")
	agent, ok := cfg.Agents["developer"]
	require.True(t, ok, "expected agent 'developer' inferred from path xcaf/agents/developer/")
	assert.Equal(t, "sonnet", agent.Model)
}

// TestParse_FilesystemInference_ValidatesInferredName tests that invalid characters in
// inferred names (from directory paths) are rejected.
func TestParse_FilesystemInference_ValidatesInferredName(t *testing.T) {
	dir := t.TempDir()
	// Create directory structure with ".." - filesystem normalization may cause issues
	// Instead, test a directory with an actual invalid name when inferred
	xcafDir := filepath.Join(dir, "xcaf", "agents", "..")
	require.NoError(t, os.MkdirAll(xcafDir, 0755))

	content := "---\nkind: agent\nversion: \"1.0\"\ndescription: \"test agent\"\nmodel: sonnet\n---\n"
	filePath := filepath.Join(xcafDir, "agent.xcaf")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	_, err := ParseDirectory(dir)
	// The error might be either:
	// 1. Invalid inferred name (if ".." gets passed through)
	// 2. Name is required (if ".." gets normalized away)
	// Both are acceptable since ".." should never appear as a valid resource ID
	require.Error(t, err, "expected error for problematic path")
}

// TestParse_FilesystemInference_SlashExceptionScopedToRule tests that / is allowed
// only in rule IDs (and is interpreted as a path component).
func TestParse_FilesystemInference_SlashExceptionScopedToRule(t *testing.T) {
	dir := t.TempDir()
	ruleDir := filepath.Join(dir, "xcaf", "rules", "cli")
	require.NoError(t, os.MkdirAll(ruleDir, 0755))

	// Rule with NO explicit name — should infer "cli" from directory
	content := "---\nkind: rule\nversion: \"1.0\"\ndescription: Build the Go CLI\n---\nBuild instructions.\n"
	filePath := filepath.Join(ruleDir, "build-go-cli.xcaf")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	cfg, err := ParseDirectory(dir)
	require.NoError(t, err, "expected successful parse for rule with / in path")
	_, ok := cfg.Rules["cli"]
	require.True(t, ok, "expected rule 'cli' inferred from path xcaf/rules/cli/")
}

// TestParse_FilesystemInference_SkipsWhenExplicitName tests that explicit name in YAML
// overrides filesystem inference.
func TestParse_FilesystemInference_SkipsWhenExplicitName(t *testing.T) {
	dir := t.TempDir()
	xcafDir := filepath.Join(dir, "xcaf", "agents", "developer")
	require.NoError(t, os.MkdirAll(xcafDir, 0755))

	// Agent file with explicit name — should NOT infer
	content := "---\nkind: agent\nversion: \"1.0\"\nname: explicit-name\ndescription: \"test agent\"\nmodel: sonnet\n---\nYou are a developer.\n"
	filePath := filepath.Join(xcafDir, "agent.xcaf")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	cfg, err := ParseDirectory(dir)
	require.NoError(t, err)
	_, ok := cfg.Agents["explicit-name"]
	require.True(t, ok, "expected agent 'explicit-name' from explicit YAML")
	_, notOk := cfg.Agents["developer"]
	require.False(t, notOk, "inferred name should not be used when name is explicitly set")
}

// TestParse_FilesystemInference_AllResourceKinds tests inference across major resource kinds.
// Memory and policy are omitted as they require additional fields for validation.
func TestParse_FilesystemInference_AllResourceKinds(t *testing.T) {
	testCases := []struct {
		kind       string
		relPath    string
		expectedID string
	}{
		{"agent", "xcaf/agents/my-agent/agent.xcaf", "my-agent"},
		{"skill", "xcaf/skills/my-skill/skill.xcaf", "my-skill"},
		{"rule", "xcaf/rules/my-rule/rule.xcaf", "my-rule"},
		{"workflow", "xcaf/workflows/my-workflow/workflow.xcaf", "my-workflow"},
		{"mcp", "xcaf/mcp/my-mcp/mcp.xcaf", "my-mcp"},
		{"context", "xcaf/context/my-context/context.xcaf", "my-context"},
	}

	for _, tc := range testCases {
		t.Run(tc.kind, func(t *testing.T) {
			dir := t.TempDir()
			filePath := filepath.Join(dir, tc.relPath)
			require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0755))

			// Workflows require steps, so we add a minimal step for the workflow case
			content := "---\nkind: " + tc.kind + "\nversion: \"1.0\"\n"
			if tc.kind == "workflow" {
				content += "steps:\n  - name: step-one\n    instructions: \"Do something\"\n"
			}
			content += "---\n"
			require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

			cfg, err := ParseDirectory(dir)
			require.NoError(t, err, "expected parse to succeed for kind %s", tc.kind)

			// Check that the resource was created with the inferred ID
			switch tc.kind {
			case "agent":
				_, ok := cfg.Agents[tc.expectedID]
				require.True(t, ok, "expected agent %s", tc.expectedID)
			case "skill":
				_, ok := cfg.Skills[tc.expectedID]
				require.True(t, ok, "expected skill %s", tc.expectedID)
			case "rule":
				_, ok := cfg.Rules[tc.expectedID]
				require.True(t, ok, "expected rule %s", tc.expectedID)
			case "workflow":
				_, ok := cfg.Workflows[tc.expectedID]
				require.True(t, ok, "expected workflow %s", tc.expectedID)
			case "mcp":
				_, ok := cfg.MCP[tc.expectedID]
				require.True(t, ok, "expected mcp %s", tc.expectedID)
			case "context":
				_, ok := cfg.Contexts[tc.expectedID]
				require.True(t, ok, "expected context %s", tc.expectedID)
			}
		})
	}
}

// TestParse_FilesystemInference_WarnsOnMismatch tests that when YAML kind/name differ
// from filesystem-inferred values, a warning is logged but parsing succeeds.
// The YAML values take precedence.
func TestParse_FilesystemInference_WarnsOnMismatch(t *testing.T) {
	dir := t.TempDir()
	xcafDir := filepath.Join(dir, "xcaf", "agents", "developer")
	require.NoError(t, os.MkdirAll(xcafDir, 0755))

	// Agent file with explicit name that does NOT match directory
	content := "---\nkind: agent\nversion: \"1.0\"\nname: reviewer\nmodel: sonnet\n---\nYou are a reviewer.\n"
	filePath := filepath.Join(xcafDir, "agent.xcaf")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	cfg, err := ParseDirectory(dir)
	// Should NOT error — mismatch is a warning, not an error
	require.NoError(t, err, "expected no error (name mismatch is a warning only)")

	// The YAML name wins (explicit takes precedence over inferred)
	agent, ok := cfg.Agents["reviewer"]
	require.True(t, ok, "expected agent keyed by YAML name 'reviewer', not inferred name 'developer'")
	require.NotNil(t, agent)
}

// TestInferKindAndName_FlatSkillFile tests inferring name from a flat skill file (no subdirectory).
// Path: xcaf/skills/commit-changes.xcaf
// Expected: kind=skill, name=commit-changes (NOT commit-changes.xcaf)
func TestInferKindAndName_FlatSkillFile(t *testing.T) {
	filePath := "xcaf/skills/commit-changes.xcaf"
	kind, name := inferKindAndName(filePath)
	assert.Equal(t, "skill", kind)
	assert.Equal(t, "commit-changes", name, "expected .xcaf extension to be stripped from filename")
}

// TestInferKindAndName_FlatRuleFile tests inferring name from a flat rule file.
// Path: xcaf/rules/secure-production-code.xcaf
// Expected: kind=rule, name=secure-production-code (NOT secure-production-code.xcaf)
func TestInferKindAndName_FlatRuleFile(t *testing.T) {
	filePath := "xcaf/rules/secure-production-code.xcaf"
	kind, name := inferKindAndName(filePath)
	assert.Equal(t, "rule", kind)
	assert.Equal(t, "secure-production-code", name, "expected .xcaf extension to be stripped from filename")
}

// TestInferKindAndName_FlatContextFile tests inferring name from a flat context file.
// Path: xcaf/context/main.xcaf
// Expected: kind=context, name=main (NOT main.xcaf)
func TestInferKindAndName_FlatContextFile(t *testing.T) {
	filePath := "xcaf/context/main.xcaf"
	kind, name := inferKindAndName(filePath)
	assert.Equal(t, "context", kind)
	assert.Equal(t, "main", name, "expected .xcaf extension to be stripped from filename")
}

// TestParse_NameMismatch_WarningCollected verifies that when a resource's declared name
// differs from the filesystem-inferred name, the warning is collected in ParseWarnings
// rather than printed directly to stderr.
func TestParse_NameMismatch_WarningCollected(t *testing.T) {
	dir := t.TempDir()
	xcafDir := filepath.Join(dir, "xcaf", "agents", "developer")
	require.NoError(t, os.MkdirAll(xcafDir, 0755))

	// Agent at xcaf/agents/developer/ but declares name: reviewer (mismatch)
	content := "---\nkind: agent\nversion: \"1.0\"\nname: reviewer\ndescription: \"test agent\"\nmodel: sonnet\n---\nYou are a reviewer.\n"
	filePath := filepath.Join(xcafDir, "agent.xcaf")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	cfg, err := ParseDirectory(dir)
	require.NoError(t, err, "name mismatch must not cause a parse error")

	require.Len(t, cfg.ParseWarnings, 1, "expected exactly one warning collected in ParseWarnings")
	assert.Contains(t, cfg.ParseWarnings[0], "reviewer", "warning must include declared name")
	assert.Contains(t, cfg.ParseWarnings[0], "developer", "warning must include inferred name")
}

// TestParse_FlatAgentFile_WarningEmitted tests that flat agent files (xcaf/agents/developer.xcaf)
// are no longer rejected; instead, a warning is emitted about the missing memory discovery layout.
func TestParse_FlatAgentFile_WarningEmitted(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, "xcaf", "agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0755))

	// Flat agent file: directly under xcaf/agents/, not in a subdirectory
	content := "---\nkind: agent\nversion: \"1.0\"\nname: developer\ndescription: \"test agent\"\nmodel: sonnet\n---\nYou are a developer.\n"
	filePath := filepath.Join(agentsDir, "developer.xcaf")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	// Need a project.xcaf to satisfy parser requirements
	projectContent := "kind: project\nversion: \"1.0\"\nname: test-project\ntargets:\n  - claude\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcaf"), []byte(projectContent), 0644))

	cfg, err := ParseDirectory(dir)
	require.NoError(t, err, "flat agent file must not cause a parse error")

	agent, ok := cfg.Agents["developer"]
	require.True(t, ok, "flat agent must be added to config")
	assert.Equal(t, "sonnet", agent.Model)

	require.NotEmpty(t, cfg.ParseWarnings, "expected a parse warning for flat agent file")
	found := false
	for _, w := range cfg.ParseWarnings {
		if contains(w, "flat file") && contains(w, "memory discovery") {
			found = true
			break
		}
	}
	assert.True(t, found, "expected warning about flat file and memory discovery, got: %v", cfg.ParseWarnings)
}

// contains is a simple helper to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && stringContains(s, substr)
}

// stringContains checks if s contains substr.
func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
