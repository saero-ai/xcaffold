package compiler

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
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

	out, _, err := Compile(config, "", "")
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

	out, _, err := Compile(config, "", "")
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

	out, _, err := Compile(config, "", "")
	require.NoError(t, err)
	assert.Contains(t, out.Files["agents/readonly.md"], "disallowed-tools: [Bash, Write]")
}

func TestCompile_EmptyAgents(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "empty-project"},
	}
	out, _, err := Compile(config, "", "")
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

	out, _, err := Compile(config, "", "")
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

	settingsJSON, hasSettings := out.Files["settings.json"]
	require.True(t, hasSettings, "expected settings.json to be compiled")
	assert.Contains(t, settingsJSON, `"PreToolUse"`)
	assert.Contains(t, settingsJSON, `"make test"`)
	assert.Contains(t, settingsJSON, `"command"`)

	// MCP (which was previously in settings.json)
	mcpContent, ok := out.Files["mcp.json"]
	require.True(t, ok, "expected mcp.json to be compiled")
	assert.Contains(t, mcpContent, `"db"`)
	assert.Contains(t, mcpContent, `"npx"`)
	assert.Contains(t, mcpContent, `"sqlite"`)
}

func TestCompile_CursorTarget_Supported(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"test": {Description: "Test", Instructions: "Test rule."},
			},
		},
	}
	out, _, err := Compile(config, "", "cursor")
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
	out, _, err := Compile(config, "", "cursor")
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
	out, _, err := Compile(config, "", "antigravity")
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
	out, _, err := Compile(config, "", "antigravity")
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
	out, _, err := Compile(config, "", "antigravity")
	require.NoError(t, err)
	// Only rule should appear, not agent
	assert.Len(t, out.Files, 1)
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
	out, _, err := Compile(config, "", "")
	require.NoError(t, err)
	for path := range out.Files {
		assert.NotContains(t, path, "..", "output path must not contain traversal sequences")
	}
}

func TestCompile_ResolveAttributes_SkillToolsInherited(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"tdd": {
					Description:  "TDD workflow",
					AllowedTools: []string{"Bash", "Read", "Write"},
					Instructions: "Follow TDD",
				},
			},
			Agents: map[string]ast.AgentConfig{
				"developer": {
					Description:  "Dev agent",
					Model:        "sonnet",
					Tools:        []string{"${skill.tdd.allowed-tools}"},
					Skills:       []string{"tdd"},
					Instructions: "You are a developer",
				},
			},
		},
	}

	output, _, err := Compile(config, t.TempDir(), "claude")
	require.NoError(t, err)

	// The compiled agent output should have the resolved tools, not the ${...} reference
	agentContent := output.Files["agents/developer.md"]
	assert.Contains(t, agentContent, "Bash")
	assert.Contains(t, agentContent, "Read")
	assert.Contains(t, agentContent, "Write")
	assert.NotContains(t, agentContent, "${skill.tdd.allowed-tools}")
}

func TestCompile_ResolveAttributes_NoRefsPassthrough(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"developer": {
					Description:  "Dev agent",
					Model:        "sonnet",
					Tools:        []string{"Bash", "Read"},
					Instructions: "You are a developer",
				},
			},
		},
	}

	output, _, err := Compile(config, t.TempDir(), "claude")
	require.NoError(t, err)
	assert.Contains(t, output.Files["agents/developer.md"], "Dev agent")
}

// Plan A4: ensure notes emitted by a target renderer are threaded through
// compiler.Compile's return values.
func TestOutputDir_AllTargets(t *testing.T) {
	tests := []struct {
		target string
		want   string
	}{
		{"", ".claude"},
		{"claude", ".claude"},
		{"cursor", ".cursor"},
		{"antigravity", ".agents"},
		{"copilot", ".github/instructions"},
		{"unknown", ".claude"},
	}
	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			got := OutputDir(tt.target)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCompile_FidelityNotes_Propagated_FromCursor(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"reviewer": {
					Description:    "Code review agent.",
					PermissionMode: "acceptEdits",
				},
			},
		},
	}

	_, notes, err := Compile(config, t.TempDir(), "cursor")
	require.NoError(t, err)
	require.NotEmpty(t, notes, "cursor compile with permissionMode must produce fidelity notes")

	var found bool
	for _, n := range notes {
		if n.Code == renderer.CodeAgentSecurityFieldsDropped && n.Resource == "reviewer" {
			found = true
			assert.Equal(t, renderer.LevelWarning, n.Level)
			assert.Equal(t, "cursor", n.Target)
			assert.Equal(t, "agent", n.Kind)
		}
	}
	assert.True(t, found, "AGENT_SECURITY_FIELDS_DROPPED note must be in the returned slice")
}
