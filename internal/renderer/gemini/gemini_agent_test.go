package gemini

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boolPtr(b bool) *bool { return &b }

func TestCompile_Gemini_Agents_Minimal(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"helper": {
					Name:        "helper",
					Description: "Helps.",
				},
			},
		},
	}
	out, notes, err := r.Compile(config, "/tmp/test")
	require.NoError(t, err)

	content, ok := out.Files[".gemini/agents/helper.md"]
	require.True(t, ok, "expected .gemini/agents/helper.md to be present")
	assert.Contains(t, content, "---")
	assert.Contains(t, content, "name: helper")
	assert.Contains(t, content, "description: Helps.")
	assert.Empty(t, notes)
}

func TestCompile_Gemini_Agents_FullSchema(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"analyst": {
					Name:        "analyst",
					Description: "Analyses code.",
					Tools:       []string{"read_file", "grep_search"},
					Model:       "gemini-3-flash-preview",
					MaxTurns:    20,
				},
			},
		},
	}
	out, _, err := r.Compile(config, "/tmp/test")
	require.NoError(t, err)

	content, ok := out.Files[".gemini/agents/analyst.md"]
	require.True(t, ok, "expected .gemini/agents/analyst.md to be present")
	assert.Contains(t, content, "tools:")
	assert.Contains(t, content, "read_file")
	assert.Contains(t, content, "grep_search")
	assert.Contains(t, content, "model: gemini-3-flash-preview")
	assert.Contains(t, content, "max_turns: 20")
}

func TestCompile_Gemini_Agents_ProviderPassthrough(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"runner": {
					Name:        "runner",
					Description: "Runs tasks.",
					Targets: map[string]ast.TargetOverride{
						"gemini": {
							Provider: map[string]any{
								"timeout_mins": 5,
								"temperature":  0.2,
								"kind":         "local",
							},
						},
					},
				},
			},
		},
	}
	out, _, err := r.Compile(config, "/tmp/test")
	require.NoError(t, err)

	content, ok := out.Files[".gemini/agents/runner.md"]
	require.True(t, ok, "expected .gemini/agents/runner.md to be present")
	assert.Contains(t, content, "timeout_mins:")
	assert.Contains(t, content, "temperature:")
	assert.Contains(t, content, "kind: local")
}

func TestCompile_Gemini_Agents_UnsupportedFields(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"restricted": {
					Name:            "restricted",
					Description:     "Has unsupported fields.",
					Effort:          "high",
					PermissionMode:  "strict",
					DisallowedTools: []string{"Bash"},
					Isolation:       "worktree",
					Background:      boolPtr(true),
					Color:           "blue",
				},
			},
		},
	}
	_, notes, err := r.Compile(config, "/tmp/test")
	require.NoError(t, err)

	codes := make(map[string]bool)
	for _, n := range notes {
		codes[n.Code] = true
	}
	assert.True(t, codes[renderer.CodeAgentSecurityFieldsDropped], "expected CodeAgentSecurityFieldsDropped for effort/permission-mode/disallowed-tools/isolation")
	assert.True(t, codes[renderer.CodeFieldUnsupported], "expected CodeFieldUnsupported for background/color")
}

func TestCompile_Gemini_Agents_InlineMCP(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"mcp-agent": {
					Name:        "mcp-agent",
					Description: "Uses an MCP server.",
					MCPServers: map[string]ast.MCPConfig{
						"my-server": {
							Command: "node",
							Args:    []string{"server.js"},
						},
					},
				},
			},
		},
	}
	out, _, err := r.Compile(config, "/tmp/test")
	require.NoError(t, err)

	content, ok := out.Files[".gemini/agents/mcp-agent.md"]
	require.True(t, ok, "expected .gemini/agents/mcp-agent.md to be present")
	assert.Contains(t, content, "mcpServers:")
	assert.Contains(t, content, "my-server")
}

func TestCompile_Gemini_Agents_WithBody(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"auditor": {
					Name:         "auditor",
					Description:  "Security auditor.",
					Instructions: "You are a security auditor.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, "/tmp/test")
	require.NoError(t, err)

	content, ok := out.Files[".gemini/agents/auditor.md"]
	require.True(t, ok, "expected .gemini/agents/auditor.md to be present")
	assert.Contains(t, content, "You are a security auditor.")
}
