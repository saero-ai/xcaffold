package cursor_test

import (
	"encoding/json"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer/cursor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCursorRenderer_FullConfig exercises every primitive type (agents, skills,
// rules, MCP) in a single compilation pass and verifies all Cursor-specific
// normalizations are applied correctly.
func TestCursorRenderer_FullConfig(t *testing.T) {
	bg := true

	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "integration-test"},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"worker": {
					Name:         "Worker Agent",
					Description:  "Handles background tasks.",
					Instructions: "Process jobs efficiently.",
					Model:        "claude-opus-4-5",
					Background:   &bg,
					// CC-only fields that must be dropped
					Effort:         "high",
					PermissionMode: "acceptEdits",
					MaxTurns:       5,
				},
			},
			Skills: map[string]ast.SkillConfig{
				"deploy": {
					Name:         "Deploy Skill",
					Description:  "Handles deployment steps.",
					Instructions: "Run the deploy pipeline.",
				},
			},
			Rules: map[string]ast.RuleConfig{
				"formatting": {
					Description:  "Code style rules",
					Instructions: "Always use gofmt.",
					Paths:        []string{"**/*.go"},
				},
				"global-safety": {
					Description:  "Always-active safety rule",
					Instructions: "Never delete production data.",
				},
			},
			MCP: map[string]ast.MCPConfig{
				"remote-api": {
					Type:    "http",
					URL:     "https://mcp.example.com/v1",
					Headers: map[string]string{"Authorization": "Bearer token123"},
				},
			},
		},
	}

	r := cursor.New()
	out, _, err := r.Compile(config, "")
	require.NoError(t, err)
	require.NotNil(t, out)

	// ── Agent ─────────────────────────────────────────────────────────────────

	agentContent, ok := out.Files["agents/worker.md"]
	require.True(t, ok, "expected agents/worker.md in output")

	// is_background: true replaces background: true (Normalization Rule 6)
	assert.Contains(t, agentContent, "is_background: true")
	assert.NotContains(t, agentContent, "\nbackground:")

	// CC-only fields must not appear
	for _, dropped := range []string{"effort:", "permission-mode:", "max-turns:"} {
		assert.NotContains(t, agentContent, dropped, "CC-only field %q must be absent", dropped)
	}

	// Cursor-compatible fields must be present
	assert.Contains(t, agentContent, "name: Worker Agent")
	assert.NotContains(t, agentContent, "model:", "literal model claude-opus-4-5 must be omitted")
	assert.Contains(t, agentContent, "Process jobs efficiently.")

	// ── Skill ─────────────────────────────────────────────────────────────────

	skillContent, ok := out.Files["skills/deploy/SKILL.md"]
	require.True(t, ok, "expected skills/deploy/SKILL.md in output")
	assert.Contains(t, skillContent, "name: Deploy Skill")
	assert.Contains(t, skillContent, "description: Handles deployment steps.")
	assert.Contains(t, skillContent, "Run the deploy pipeline.")

	// ── Rules ─────────────────────────────────────────────────────────────────

	// Rule with paths → globs: in frontmatter, no alwaysApply
	fmtContent, ok := out.Files["rules/formatting.mdc"]
	require.True(t, ok, "expected rules/formatting.mdc in output")
	assert.Contains(t, fmtContent, "globs:")
	assert.Contains(t, fmtContent, "**/*.go")
	assert.NotContains(t, fmtContent, "alwaysApply:")
	assert.NotContains(t, fmtContent, "paths:")

	// Rule without paths → alwaysApply: true
	safetyContent, ok := out.Files["rules/global-safety.mdc"]
	require.True(t, ok, "expected rules/global-safety.mdc in output")
	assert.Contains(t, safetyContent, "alwaysApply: true")
	assert.NotContains(t, safetyContent, "globs:")

	// Rules must use .mdc extension, not .md
	_, hasMd := out.Files["rules/formatting.md"]
	assert.False(t, hasMd, "rules must not emit .md files for the cursor target")

	// ── MCP ───────────────────────────────────────────────────────────────────

	mcpRaw, ok := out.Files["mcp.json"]
	require.True(t, ok, "expected mcp.json in output")

	var envelope map[string]map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(mcpRaw), &envelope))

	servers, ok := envelope["mcpServers"]
	require.True(t, ok, "mcp.json must contain mcpServers envelope")

	entry, ok := servers["remote-api"].(map[string]interface{})
	require.True(t, ok, "remote-api server must be present in mcpServers")

	// url → serverUrl (Normalization Rule 2)
	assert.Equal(t, "https://mcp.example.com/v1", entry["serverUrl"])
	assert.Nil(t, entry["url"], "original url field must be absent")
	// type must be omitted — Cursor infers transport
	assert.Nil(t, entry["type"], "type field must be omitted from mcp.json")

	// ── File count ────────────────────────────────────────────────────────────
	// 1 agent + 1 skill + 2 rules + 1 mcp.json = 5 files
	assert.Len(t, out.Files, 5, "expected exactly 5 output files")
}
