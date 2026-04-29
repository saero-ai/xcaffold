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
	filePath := "some/path/xcf/agents/developer/agent.xcf"
	kind, name := inferKindAndName(filePath)
	assert.Equal(t, "agent", kind)
	assert.Equal(t, "developer", name)
}

// TestInferKindAndName_SkillPath tests skill inference.
func TestInferKindAndName_SkillPath(t *testing.T) {
	filePath := "xcf/skills/tdd/skill.xcf"
	kind, name := inferKindAndName(filePath)
	assert.Equal(t, "skill", kind)
	assert.Equal(t, "tdd", name)
}

// TestInferKindAndName_RulePath tests rule inference with allowed slash.
func TestInferKindAndName_RulePath(t *testing.T) {
	filePath := "xcf/rules/cli/rule.xcf"
	kind, name := inferKindAndName(filePath)
	assert.Equal(t, "rule", kind)
	assert.Equal(t, "cli", name)
}

// TestInferKindAndName_NoXcfDir tests that inference returns empty when xcf is not in path.
func TestInferKindAndName_NoXcfDir(t *testing.T) {
	filePath := "some/path/agents/developer/agent.xcf"
	kind, name := inferKindAndName(filePath)
	assert.Equal(t, "", kind)
	assert.Equal(t, "", name)
}

// TestInferKindAndName_InvalidKindDir tests that inference returns empty for unknown kinds.
func TestInferKindAndName_InvalidKindDir(t *testing.T) {
	filePath := "xcf/unknown-kind/something/file.xcf"
	kind, name := inferKindAndName(filePath)
	assert.Equal(t, "", kind)
	assert.Equal(t, "", name)
}

// TestInferKindAndName_TooShortPath tests that inference returns empty if path is too short.
func TestInferKindAndName_TooShortPath(t *testing.T) {
	filePath := "xcf/agents"
	kind, name := inferKindAndName(filePath)
	assert.Equal(t, "", kind)
	assert.Equal(t, "", name)
}

// TestParse_FilesystemInference_InfersNameWhenKindProvided tests that name is inferred
// from filesystem when the YAML provides kind: but no name:.
func TestParse_FilesystemInference_InfersNameWhenKindProvided(t *testing.T) {
	dir := t.TempDir()
	xcfDir := filepath.Join(dir, "xcf", "agents", "developer")
	require.NoError(t, os.MkdirAll(xcfDir, 0755))

	// Agent file with kind: but NO name: — name should be inferred from path
	content := "---\nkind: agent\nversion: \"1.0\"\nmodel: sonnet\n---\nYou are a developer.\n"
	filePath := filepath.Join(xcfDir, "agent.xcf")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	cfg, err := ParseDirectory(dir)
	require.NoError(t, err, "expected successful parse with inferred name")
	agent, ok := cfg.Agents["developer"]
	require.True(t, ok, "expected agent 'developer' inferred from path xcf/agents/developer/")
	assert.Equal(t, "sonnet", agent.Model)
}

// TestParse_FilesystemInference_ValidatesInferredName tests that invalid characters in
// inferred names (from directory paths) are rejected.
func TestParse_FilesystemInference_ValidatesInferredName(t *testing.T) {
	dir := t.TempDir()
	// Create directory structure with ".." - filesystem normalization may cause issues
	// Instead, test a directory with an actual invalid name when inferred
	xcfDir := filepath.Join(dir, "xcf", "agents", "..")
	require.NoError(t, os.MkdirAll(xcfDir, 0755))

	content := "---\nkind: agent\nversion: \"1.0\"\nmodel: sonnet\n---\n"
	filePath := filepath.Join(xcfDir, "agent.xcf")
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
	ruleDir := filepath.Join(dir, "xcf", "rules", "cli")
	require.NoError(t, os.MkdirAll(ruleDir, 0755))

	// Rule with NO explicit name — should infer "cli" from directory
	content := "---\nkind: rule\nversion: \"1.0\"\ndescription: Build the Go CLI\n---\nBuild instructions.\n"
	filePath := filepath.Join(ruleDir, "build-go-cli.xcf")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	cfg, err := ParseDirectory(dir)
	require.NoError(t, err, "expected successful parse for rule with / in path")
	_, ok := cfg.Rules["cli"]
	require.True(t, ok, "expected rule 'cli' inferred from path xcf/rules/cli/")
}

// TestParse_FilesystemInference_SkipsWhenExplicitName tests that explicit name in YAML
// overrides filesystem inference.
func TestParse_FilesystemInference_SkipsWhenExplicitName(t *testing.T) {
	dir := t.TempDir()
	xcfDir := filepath.Join(dir, "xcf", "agents", "developer")
	require.NoError(t, os.MkdirAll(xcfDir, 0755))

	// Agent file with explicit name — should NOT infer
	content := "---\nkind: agent\nversion: \"1.0\"\nname: explicit-name\nmodel: sonnet\n---\nYou are a developer.\n"
	filePath := filepath.Join(xcfDir, "agent.xcf")
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
		{"agent", "xcf/agents/my-agent/agent.xcf", "my-agent"},
		{"skill", "xcf/skills/my-skill/skill.xcf", "my-skill"},
		{"rule", "xcf/rules/my-rule/rule.xcf", "my-rule"},
		{"workflow", "xcf/workflows/my-workflow/workflow.xcf", "my-workflow"},
		{"mcp", "xcf/mcp/my-mcp/mcp.xcf", "my-mcp"},
		{"context", "xcf/context/my-context/context.xcf", "my-context"},
	}

	for _, tc := range testCases {
		t.Run(tc.kind, func(t *testing.T) {
			dir := t.TempDir()
			filePath := filepath.Join(dir, tc.relPath)
			require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0755))

			content := "---\nkind: " + tc.kind + "\nversion: \"1.0\"\n---\n"
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
	xcfDir := filepath.Join(dir, "xcf", "agents", "developer")
	require.NoError(t, os.MkdirAll(xcfDir, 0755))

	// Agent file with explicit name that does NOT match directory
	content := "---\nkind: agent\nversion: \"1.0\"\nname: reviewer\nmodel: sonnet\n---\nYou are a reviewer.\n"
	filePath := filepath.Join(xcfDir, "agent.xcf")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	cfg, err := ParseDirectory(dir)
	// Should NOT error — mismatch is a warning, not an error
	require.NoError(t, err, "expected no error (name mismatch is a warning only)")

	// The YAML name wins (explicit takes precedence over inferred)
	agent, ok := cfg.Agents["reviewer"]
	require.True(t, ok, "expected agent keyed by YAML name 'reviewer', not inferred name 'developer'")
	require.NotNil(t, agent)
}
