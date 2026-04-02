package compiler

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompile_SingleAgent(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Project: ast.ProjectConfig{Name: "test-project"},
		Agents: map[string]ast.AgentConfig{
			"developer": {
				Description:  "An expert developer.",
				Instructions: "You are a software developer.\nWrite clean code.\n",
				Model:        "claude-3-5-sonnet-20241022",
				Effort:       "high",
				Tools:        []string{"Bash", "Read", "Write"},
			},
		},
	}

	out, err := Compile(config)
	require.NoError(t, err)
	require.NotNil(t, out)

	content, ok := out.Files["agents/developer.md"]
	require.True(t, ok, "expected agents/developer.md to be compiled")

	assert.Contains(t, content, "description: An expert developer.")
	assert.Contains(t, content, "model: claude-3-5-sonnet-20241022")
	assert.Contains(t, content, "model_setting_effort: high")
	assert.Contains(t, content, "tools: [Bash, Read, Write]")
	assert.Contains(t, content, "You are a software developer.")
}

func TestCompile_MultipleAgents(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Project: ast.ProjectConfig{Name: "multi-agent-project"},
		Agents: map[string]ast.AgentConfig{
			"frontend": {Description: "Frontend specialist."},
			"backend":  {Description: "Backend specialist."},
		},
	}

	out, err := Compile(config)
	require.NoError(t, err)
	assert.Len(t, out.Files, 2)
	assert.Contains(t, out.Files, "agents/frontend.md")
	assert.Contains(t, out.Files, "agents/backend.md")
}

func TestCompile_AgentWithBlockedTools(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Project: ast.ProjectConfig{Name: "secure-project"},
		Agents: map[string]ast.AgentConfig{
			"readonly": {
				Description:  "Read-only agent.",
				Tools:        []string{"Read", "Grep"},
				BlockedTools: []string{"Bash", "Write"},
			},
		},
	}

	out, err := Compile(config)
	require.NoError(t, err)
	assert.Contains(t, out.Files["agents/readonly.md"], "tools_blocked: [Bash, Write]")
}

func TestCompile_EmptyAgents(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Project: ast.ProjectConfig{Name: "empty-project"},
	}
	out, err := Compile(config)
	require.NoError(t, err)
	assert.Empty(t, out.Files)
}

func TestCompileAgentMarkdown_PathTraversalPrevented(t *testing.T) {
	// An agent id containing path separators should be cleaned safely.
	config := &ast.XcaffoldConfig{
		Project: ast.ProjectConfig{Name: "traversal-test"},
		Agents: map[string]ast.AgentConfig{
			"../evil": {Description: "Malicious agent."},
		},
	}
	out, err := Compile(config)
	require.NoError(t, err)
	for path := range out.Files {
		assert.NotContains(t, path, "..", "output path must not contain traversal sequences")
	}
}
