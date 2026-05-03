package gemini

import (
	"strings"
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
	out, notes, err := renderer.Orchestrate(r, config, "/tmp/test")
	require.NoError(t, err)

	content, ok := out.Files["agents/helper.md"]
	require.True(t, ok, "expected agents/helper.md (relative to OutputDir) to be present")
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
	out, _, err := renderer.Orchestrate(r, config, "/tmp/test")
	require.NoError(t, err)

	content, ok := out.Files["agents/analyst.md"]
	require.True(t, ok, "expected agents/analyst.md (relative to OutputDir) to be present")
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
	out, _, err := renderer.Orchestrate(r, config, "/tmp/test")
	require.NoError(t, err)

	content, ok := out.Files["agents/runner.md"]
	require.True(t, ok, "expected agents/runner.md (relative to OutputDir) to be present")
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
	_, notes, err := renderer.Orchestrate(r, config, "/tmp/test")
	require.NoError(t, err)

	codes := make(map[string]bool)
	for _, n := range notes {
		codes[n.Code] = true
	}
	// All unsupported fields — including security fields — now emit FIELD_UNSUPPORTED
	// via the orchestrator's CheckFieldSupport. AGENT_SECURITY_FIELDS_DROPPED is
	// no longer emitted by the renderer.
	assert.True(t, codes[renderer.CodeFieldUnsupported], "expected CodeFieldUnsupported for background/color/permission-mode/disallowed-tools/isolation/effort")
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
	out, _, err := renderer.Orchestrate(r, config, "/tmp/test")
	require.NoError(t, err)

	content, ok := out.Files["agents/mcp-agent.md"]
	require.True(t, ok, "expected agents/mcp-agent.md (relative to OutputDir) to be present")
	assert.Contains(t, content, "mcpServers:")
	assert.Contains(t, content, "my-server")
}

func TestRenderAgents_ModelAlias_Translated(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"test-agent": {
					Name:        "test-agent",
					Description: "A test agent",
					Model:       "sonnet-4",
					Body:        "Do things.",
				},
			},
		},
	}
	r := New()
	out, notes, err := renderer.Orchestrate(r, config, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_ = notes

	agentFile, ok := out.Files["agents/test-agent.md"]
	if !ok {
		t.Fatal("expected agents/test-agent.md in output")
	}
	if strings.Contains(agentFile, "model: sonnet-4") {
		t.Errorf("agent file contains raw alias 'sonnet-4'; expected translated model ID")
	}
}

func TestCompile_Gemini_Agents_WithBody(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"auditor": {
					Name:        "auditor",
					Description: "Security auditor.",
					Body:        "You are a security auditor.",
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "/tmp/test")
	require.NoError(t, err)

	content, ok := out.Files["agents/auditor.md"]
	require.True(t, ok, "expected agents/auditor.md (relative to OutputDir) to be present")
	assert.Contains(t, content, "You are a security auditor.")
}
