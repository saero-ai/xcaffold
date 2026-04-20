package copilot_test

import (
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/renderer/copilot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileCopilotRule_Activation_Always(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"security": {
					Description:  "Security guide.",
					Activation:   ast.RuleActivationAlways,
					Instructions: "Follow OWASP.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["instructions/security.instructions.md"]
	require.Contains(t, content, `applyTo: "**"`)
}

func TestCompileCopilotRule_Activation_PathGlob(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"api-style": {
					Activation:   ast.RuleActivationPathGlob,
					Paths:        []string{"src/api/**", "packages/api/**"},
					Instructions: "REST conventions.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["instructions/api-style.instructions.md"]
	require.Contains(t, content, `applyTo: "src/api/**, packages/api/**"`)
}

func TestCompileCopilotRule_ExcludeAgents_Single(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"pr-review": {
					Activation:    ast.RuleActivationAlways,
					ExcludeAgents: []string{"code-review"},
					Instructions:  "Review standards.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["instructions/pr-review.instructions.md"]
	require.Contains(t, content, "excludeAgent:")
	require.Contains(t, content, "code-review")
}

func TestCompileCopilotRule_ExcludeAgents_Multiple(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"security": {
					Activation:    ast.RuleActivationAlways,
					ExcludeAgents: []string{"code-review", "cloud-agent"},
					Instructions:  "Security.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["instructions/security.instructions.md"]
	require.Contains(t, content, "code-review")
	require.Contains(t, content, "cloud-agent")
}

func TestCompileCopilotRule_OutputPath(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"my-rule": {
					Activation:   ast.RuleActivationAlways,
					Instructions: "Body.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	_, ok := out.Files["instructions/my-rule.instructions.md"]
	require.True(t, ok, "output path must be .github/instructions/<id>.instructions.md")
}

func TestCompileCopilotRule_Activation_ManualMention_FidelityNote(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"manual-rule": {
					Activation:   ast.RuleActivationManualMention,
					Instructions: "Only when mentioned.",
				},
			},
		},
	}
	out, notes, err := r.Compile(config, "")
	require.NoError(t, err)

	// Degraded to "**" with a fidelity note
	content := out.Files["instructions/manual-rule.instructions.md"]
	require.Contains(t, content, `applyTo: "**"`)
	require.NotEmpty(t, notes, "expected a fidelity note for manual-mention activation")
}

func TestCompile_Copilot_Workflows_LoweredToRulePlusSkill(t *testing.T) {
	r := copilot.New()
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Workflows: map[string]ast.WorkflowConfig{
				"deploy-flow": {
					Name:        "deploy-flow",
					Description: "Deployment workflow",
					Steps: []ast.WorkflowStep{
						{Name: "step-1", Instructions: "Run tests"},
						{Name: "step-2", Instructions: "Deploy to staging"},
					},
				},
			},
		},
	}
	out, notes, err := r.Compile(cfg, "")
	require.NoError(t, err)

	// Workflow should produce lowered rule and/or skill files.
	hasRule := false
	hasSkill := false
	for path := range out.Files {
		if strings.HasPrefix(path, "instructions/") {
			hasRule = true
		}
		if strings.HasPrefix(path, "skills/") {
			hasSkill = true
		}
	}
	assert.True(t, hasRule || hasSkill, "workflow should lower to at least a rule or skill")

	workflowNotes := filterNotes(notes, renderer.CodeWorkflowLoweredToRulePlusSkill)
	assert.NotEmpty(t, workflowNotes, "workflow lowering should emit fidelity note")
}

func TestCompile_Copilot_FullConfig_AllKinds(t *testing.T) {
	r := copilot.New()
	timeout := 3000
	cfg := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Instructions: "Full Copilot integration test",
		},
		Settings: map[string]ast.SettingsConfig{"default": {
			Sandbox: &ast.SandboxConfig{},
		}},
		Hooks: map[string]ast.NamedHookConfig{
			"default": {
				Name: "default",
				Events: ast.HookConfig{
					"PreToolUse": []ast.HookMatcherGroup{
						{Hooks: []ast.HookHandler{
							{Type: "command", Command: "echo checking", Timeout: &timeout},
						}},
					},
				},
			},
		},
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"go-style": {
					Description:  "Go style guide",
					Instructions: "Follow Go conventions",
					Activation:   "always",
				},
			},
			Agents: map[string]ast.AgentConfig{
				"auditor": {
					Name:        "auditor",
					Description: "Security auditor",
					Tools:       []string{"read", "search"},
				},
			},
			Skills: map[string]ast.SkillConfig{
				"review": {
					Name:         "review",
					Description:  "Code review skill",
					AllowedTools: []string{"shell"},
					Instructions: "Review code carefully.",
				},
			},
			MCP: map[string]ast.MCPConfig{
				"test-mcp": {Command: "npx", Args: []string{"mcp-test"}},
			},
		},
	}
	out, notes, err := r.Compile(cfg, "")
	require.NoError(t, err)

	// Verify ALL output paths exist.
	assert.Contains(t, out.Files, "copilot-instructions.md", "instructions")
	assert.Contains(t, out.Files, "instructions/go-style.instructions.md", "rule")
	assert.Contains(t, out.Files, "agents/auditor.agent.md", "agent")
	assert.Contains(t, out.Files, "skills/review/SKILL.md", "skill")
	assert.Contains(t, out.Files, "hooks/xcaffold-hooks.json", "hooks")
	// MCP is not written to the output map; it requires manual .vscode/mcp.json placement.
	assert.NotContains(t, out.Files, ".vscode/mcp.json", "MCP must not be in output map")

	// Verify fidelity notes.
	settingsNotes := filterNotes(notes, renderer.CodeSettingsFieldUnsupported)
	assert.NotEmpty(t, settingsNotes, "sandbox should emit unsupported note")

	mcpNotes := filterNotes(notes, renderer.CodeMCPGlobalConfigOnly)
	assert.NotEmpty(t, mcpNotes, "MCP should emit global config note")
}

func TestCompile_Copilot_FullConfig_Session1(t *testing.T) {
	r := copilot.New()
	cfg := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Instructions: "Project instructions for Copilot",
		},
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"style-guide": {
					Description:  "Style guide",
					Instructions: "Follow the Go style guide",
					Activation:   "always",
				},
			},
			Agents: map[string]ast.AgentConfig{
				"reviewer": {
					Name:        "reviewer",
					Description: "Code reviewer agent",
					Tools:       []string{"read", "search"},
					Model:       "gpt-4o",
				},
			},
			Skills: map[string]ast.SkillConfig{
				"tdd": {
					Name:         "tdd",
					Description:  "Test-driven development workflow",
					AllowedTools: []string{"shell"},
					Instructions: "Write failing test first.",
				},
			},
		},
	}
	out, _, err := r.Compile(cfg, "")
	require.NoError(t, err)

	assert.Contains(t, out.Files, "copilot-instructions.md", "instructions")
	assert.Contains(t, out.Files, "instructions/style-guide.instructions.md", "rule")
	assert.Contains(t, out.Files, "agents/reviewer.agent.md", "agent")
	assert.Contains(t, out.Files, "skills/tdd/SKILL.md", "skill")
}
