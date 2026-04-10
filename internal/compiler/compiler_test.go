package compiler

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompile_SingleAgent(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "test-project"},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"developer": {
					Description:  "An expert developer.",
					Instructions: "You are a software developer.\nWrite clean code.\n",
					Model:        "claude-3-7-sonnet-20250219",
					Effort:       "high",
					Tools:        []string{"Bash", "Read", "Write"},
				},
			},
		},
	}

	out, err := Compile(config, "", "")
	require.NoError(t, err)
	require.NotNil(t, out)

	content, ok := out.Files["agents/developer.md"]
	require.True(t, ok, "expected agents/developer.md to be compiled")

	assert.Contains(t, content, "description: An expert developer.")
	assert.Contains(t, content, "model: claude-3-7-sonnet-20250219")
	assert.Contains(t, content, "effort: high")
	assert.Contains(t, content, "tools: [Bash, Read, Write]")
	assert.Contains(t, content, "You are a software developer.")
}

func TestCompile_MultipleAgents(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "multi-agent-project"},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"frontend": {Description: "Frontend specialist."},
				"backend":  {Description: "Backend specialist."},
			},
		},
	}

	out, err := Compile(config, "", "")
	require.NoError(t, err)
	assert.Len(t, out.Files, 2)
	assert.Contains(t, out.Files, "agents/frontend.md")
	assert.Contains(t, out.Files, "agents/backend.md")
}

func TestCompile_AgentWithBlockedTools(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "secure-project"},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"readonly": {
					Description:     "Read-only agent.",
					Tools:           []string{"Read", "Grep"},
					DisallowedTools: []string{"Bash", "Write"},
				},
			},
		},
	}

	out, err := Compile(config, "", "")
	require.NoError(t, err)
	assert.Contains(t, out.Files["agents/readonly.md"], "disallowedTools: [Bash, Write]")
}

func TestCompile_EmptyAgents(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "empty-project"},
	}
	out, err := Compile(config, "", "")
	require.NoError(t, err)
	assert.Empty(t, out.Files)
}

func TestCompile_FullSchema(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "full-project"},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {Description: "A developer."},
			},
			Skills: map[string]ast.SkillConfig{
				"git": {
					Description:  "Git workflows",
					Instructions: "Always use rebase.",
				},
			},
			Rules: map[string]ast.RuleConfig{
				"go": {
					Instructions: "Use gofmt.",
				},
			},
			Hooks: ast.HookConfig{
				"PreToolUse": []ast.HookMatcherGroup{
					{
						Matcher: "Bash",
						Hooks: []ast.HookHandler{
							{Type: "command", Command: "make test"},
						},
					},
				},
			},
			MCP: map[string]ast.MCPConfig{
				"db": {
					Command: "npx",
					Args:    []string{"-y", "sqlite"},
				},
			},
		},
	}

	out, err := Compile(config, "", "")
	require.NoError(t, err)

	// Agents
	assert.Contains(t, out.Files, "agents/dev.md")

	// Skills — now compiled as skills/<id>/SKILL.md directories (Bug 10)
	skillContent, ok := out.Files["skills/git/SKILL.md"]
	require.True(t, ok, "expected skills/git/SKILL.md to be compiled")
	assert.Contains(t, skillContent, "description: Git workflows")
	assert.Contains(t, skillContent, "Always use rebase.")

	// Rules
	ruleContent, ok := out.Files["rules/go.md"]
	require.True(t, ok, "expected rules/go.md to be compiled")
	assert.Contains(t, ruleContent, "Use gofmt.")

	hookContent, ok := out.Files["hooks.json"]
	require.True(t, ok, "expected hooks.json to be compiled")
	assert.Contains(t, hookContent, `"PreToolUse"`)
	assert.Contains(t, hookContent, `"make test"`)
	assert.Contains(t, hookContent, `"command"`)

	// Settings (which should include MCP)
	settingsContent, ok := out.Files["settings.json"]
	require.True(t, ok, "expected settings.json to be compiled")
	assert.Contains(t, settingsContent, `"db"`)
	assert.Contains(t, settingsContent, `"npx"`)
	assert.Contains(t, settingsContent, `"sqlite"`)
}

func TestCompile_CursorTarget_Supported(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"test": {Description: "Test", Instructions: "Test rule."},
			},
		},
	}
	out, err := Compile(config, "", "cursor")
	require.NoError(t, err)
	assert.NotEmpty(t, out.Files)
}

func TestCompile_CursorTarget_RulesUseMdc(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"style": {Description: "Style", Instructions: "Format code."},
			},
		},
	}
	out, err := Compile(config, "", "cursor")
	require.NoError(t, err)
	_, ok := out.Files["rules/style.mdc"]
	assert.True(t, ok, "Cursor rules should use .mdc extension")
}

func TestCompile_AntigravityTarget_Supported(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"test": {Description: "Test", Instructions: "Test rule."},
			},
		},
	}
	out, err := Compile(config, "", "antigravity")
	require.NoError(t, err)
	assert.NotEmpty(t, out.Files)
}

func TestCompile_AntigravityTarget_RulesNoFrontmatter(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"style": {Description: "Style", Instructions: "Format code."},
			},
		},
	}
	out, err := Compile(config, "", "antigravity")
	require.NoError(t, err)
	content, ok := out.Files["rules/style.md"]
	assert.True(t, ok)
	assert.NotContains(t, content, "---")
}

func TestCompile_AntigravityTarget_AgentsExcluded(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"reviewer": {Name: "Reviewer", Instructions: "Review."},
			},
			Rules: map[string]ast.RuleConfig{
				"test": {Instructions: "Test."},
			},
		},
	}
	out, err := Compile(config, "", "antigravity")
	require.NoError(t, err)
	// Only rule should appear, not agent
	assert.Len(t, out.Files, 1)
}

func TestCompile_AgentsMD_Target(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "agentsmd-test"},
	}
	tmpDir := t.TempDir()
	out, err := Compile(config, tmpDir, "agentsmd")
	require.NoError(t, err)
	require.NotNil(t, out)
	_, ok := out.Files["AGENTS.md"]
	assert.True(t, ok, "expected AGENTS.md in compiler output for agentsmd target")
}

func TestOutputDir_AgentsMD(t *testing.T) {
	dir := OutputDir("agentsmd")
	assert.Equal(t, ".", dir)
}

func TestCompileAgentMarkdown_PathTraversalPrevented(t *testing.T) {
	// An agent id containing path separators should be cleaned safely.
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "traversal-test"},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"../evil": {Description: "Malicious agent."},
			},
		},
	}
	out, err := Compile(config, "", "")
	require.NoError(t, err)
	for path := range out.Files {
		assert.NotContains(t, path, "..", "output path must not contain traversal sequences")
	}
}
