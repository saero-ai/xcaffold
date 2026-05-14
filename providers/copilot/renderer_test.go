package copilot_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/providers/copilot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileCopilotRule_Activation_Always(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"security": {
					Description: "Security guide.",
					Activation:  ast.RuleActivationAlways,
					Body:        "Follow OWASP.",
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
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
					Activation: ast.RuleActivationPathGlob,
					Paths:      ast.ClearableList{Values: []string{"src/api/**", "packages/api/**"}},
					Body:       "REST conventions.",
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
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
					ExcludeAgents: ast.ClearableList{Values: []string{"code-review"}},
					Body:          "Review standards.",
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
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
					ExcludeAgents: ast.ClearableList{Values: []string{"code-review", "cloud-agent"}},
					Body:          "Security.",
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
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
					Activation: ast.RuleActivationAlways,
					Body:       "Body.",
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
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
					Activation: ast.RuleActivationManualMention,
					Body:       "Only when mentioned.",
				},
			},
		},
	}
	out, notes, err := renderer.Orchestrate(r, config, "")
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
					Targets: map[string]ast.TargetOverride{
						"copilot": {Provider: map[string]any{"lowering-strategy": "rule-plus-skill"}},
					},
				},
			},
		},
	}
	out, notes, err := renderer.Orchestrate(r, cfg, "")
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

// TestCompile_Copilot_Workflows_DefaultSimpleMode verifies that a workflow
// without an explicit lowering-strategy uses the new default behavior: structure-based
// inference produces a single skill file (simple mode) rather than per-step skills
// or a rule. The test exercises the new default-behavior path and verifies the
// migration fidelity notes are emitted.
func TestCompile_Copilot_Workflows_DefaultSimpleMode(t *testing.T) {
	r := copilot.New()
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Workflows: map[string]ast.WorkflowConfig{
				"code-review": {
					Name:        "code-review",
					Description: "Multi-step pull request review procedure.",
					Steps: []ast.WorkflowStep{
						{Name: "analyze", Instructions: "Read the diff."},
						{Name: "lint", Instructions: "Check style."},
						{Name: "summarize", Instructions: "Write the review."},
					},
					// No Targets, no lowering-strategy — new default applies.
				},
			},
		},
	}
	out, notes, err := renderer.Orchestrate(r, cfg, "")
	require.NoError(t, err)

	// New default: single skill file (NOT per-step micro-skills).
	hasSkill := false
	for path := range out.Files {
		if strings.HasPrefix(path, "skills/code-review") && strings.HasSuffix(path, "SKILL.md") {
			hasSkill = true
			break
		}
	}
	require.True(t, hasSkill, "expected skills/code-review/SKILL.md for simple mode; got keys: %v", mapKeys(out.Files))

	// No instruction file should exist (no always-apply or paths set).
	hasInstructions := false
	for path := range out.Files {
		if strings.HasPrefix(path, "instructions/") && strings.Contains(path, "code-review") {
			hasInstructions = true
			break
		}
	}
	require.False(t, hasInstructions, "simple mode without always-apply should NOT emit an instructions file; got keys: %v", mapKeys(out.Files))

	// Should have CodeWorkflowBasicToSections note.
	var hasSimpleNote bool
	for _, n := range notes {
		if n.Code == renderer.CodeWorkflowBasicToSections {
			hasSimpleNote = true
		}
	}
	require.True(t, hasSimpleNote, "expected CodeWorkflowBasicToSections note; got: %v", notes)

	// Should have CodeWorkflowDefaultChanged migration warning.
	var hasMigrationNote bool
	for _, n := range notes {
		if n.Code == renderer.CodeWorkflowDefaultChanged {
			hasMigrationNote = true
		}
	}
	require.True(t, hasMigrationNote, "expected CodeWorkflowDefaultChanged migration note; got: %v", notes)
}

func TestCompile_Copilot_FullConfig_AllKinds(t *testing.T) {
	r := copilot.New()
	timeout := 3000
	cfg := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "test-proj"},
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
			Contexts: map[string]ast.ContextConfig{
				"root": {Body: "Full Copilot integration test"},
			},
			Rules: map[string]ast.RuleConfig{
				"go-style": {
					Description: "Go style guide",
					Body:        "Follow Go conventions",
					Activation:  "always",
				},
			},
			Agents: map[string]ast.AgentConfig{
				"auditor": {
					Name:        "auditor",
					Description: "Security auditor",
					Tools:       ast.ClearableList{Values: []string{"read", "search"}},
				},
			},
			Skills: map[string]ast.SkillConfig{
				"review": {
					Name:         "review",
					Description:  "Code review skill",
					AllowedTools: ast.ClearableList{Values: []string{"shell"}},
					Body:         "Review code carefully.",
				},
			},
			MCP: map[string]ast.MCPConfig{
				"test-mcp": {Command: "npx", Args: []string{"mcp-test"}},
			},
		},
	}
	out, notes, err := renderer.Orchestrate(r, cfg, "")
	require.NoError(t, err)

	// Verify ALL output paths exist.
	assert.Contains(t, out.Files, "copilot-instructions.md", "instructions")
	assert.Contains(t, out.Files, "instructions/go-style.instructions.md", "rule")
	assert.Contains(t, out.Files, "agents/auditor.agent.md", "agent")
	assert.Contains(t, out.Files, "skills/review/SKILL.md", "skill")
	assert.Contains(t, out.Files, "hooks/xcaffold-hooks.json", "hooks")
	assert.Contains(t, out.RootFiles, ".vscode/mcp.json", "MCP must be in rootFiles (project root)")

	// Verify fidelity notes.
	settingsNotes := filterNotes(notes, renderer.CodeSettingsFieldUnsupported)
	assert.NotEmpty(t, settingsNotes, "sandbox should emit unsupported note")

	mcpNotes := filterNotes(notes, renderer.CodeMCPGlobalConfigOnly)
	assert.NotEmpty(t, mcpNotes, "MCP should emit global config note")
}

func TestCompile_Copilot_FullConfig_Session1(t *testing.T) {
	r := copilot.New()
	cfg := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "test-proj"},
		ResourceScope: ast.ResourceScope{
			Contexts: map[string]ast.ContextConfig{
				"root": {Body: "Project instructions for Copilot"},
			},
			Rules: map[string]ast.RuleConfig{
				"style-guide": {
					Description: "Style guide",
					Body:        "Follow the Go style guide",
					Activation:  "always",
				},
			},
			Agents: map[string]ast.AgentConfig{
				"reviewer": {
					Name:        "reviewer",
					Description: "Code reviewer agent",
					Tools:       ast.ClearableList{Values: []string{"read", "search"}},
					Model:       "gpt-4o",
				},
			},
			Skills: map[string]ast.SkillConfig{
				"tdd": {
					Name:         "tdd",
					Description:  "Test-driven development workflow",
					AllowedTools: ast.ClearableList{Values: []string{"shell"}},
					Body:         "Write failing test first.",
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, cfg, "")
	require.NoError(t, err)

	assert.Contains(t, out.Files, "copilot-instructions.md", "instructions")
	assert.Contains(t, out.Files, "instructions/style-guide.instructions.md", "rule")
	assert.Contains(t, out.Files, "agents/reviewer.agent.md", "agent")
	assert.Contains(t, out.Files, "skills/tdd/SKILL.md", "skill")
}

func TestCompileAgents_Copilot_ClaudeToolsDropped(t *testing.T) {
	r := copilot.New()
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"tester": {
					Name: "tester", Tools: ast.ClearableList{Values: []string{"Read", "Write"}},
				},
			},
		},
	}
	out, notes, err := renderer.Orchestrate(r, cfg, "")
	require.NoError(t, err)

	content := out.Files["agents/tester.agent.md"]
	assert.NotContains(t, content, "tools:")

	found := false
	for _, n := range notes {
		if n.Code == renderer.CodeAgentToolsDropped {
			found = true
		}
	}
	assert.True(t, found, "expected CodeAgentToolsDropped note")
}

func TestCompileAgents_Copilot_ClaudeAliasModel_Omitted(t *testing.T) {
	r := copilot.New()
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"tester": {
					Name: "tester", Model: "sonnet",
				},
			},
		},
	}
	out, notes, err := renderer.Orchestrate(r, cfg, "")
	require.NoError(t, err)

	content := out.Files["agents/tester.agent.md"]
	assert.NotContains(t, content, "model:")

	found := false
	for _, n := range notes {
		if n.Code == renderer.CodeAgentModelUnmapped && n.Level == renderer.LevelWarning {
			found = true
		}
	}
	assert.True(t, found, "expected LevelWarning CodeAgentModelUnmapped")
}

func TestCompileRules_Copilot_ClaudeDirPresent_EmitsPassthroughNotes(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".claude"), 0o755))

	r := copilot.New()
	rules := map[string]ast.RuleConfig{
		"adr-governance": {Body: "Follow ADR governance.", Activation: "always"},
	}
	files, notes, err := r.CompileRules(rules, dir)
	require.NoError(t, err)
	assert.Empty(t, files, "no .github/instructions/ files should be written when .claude/ is present")
	require.Len(t, notes, 1)
	assert.Equal(t, renderer.CodeClaudeNativePassthrough, notes[0].Code)
	assert.Equal(t, renderer.LevelInfo, notes[0].Level)
	assert.Equal(t, "adr-governance", notes[0].Resource)
	assert.Contains(t, notes[0].Reason, ".claude/rules/ detected")
}

// TestCompileRules_Copilot_NoClaude_FullTranslation verifies that when no
// .claude/ directory exists, CompileRules writes the instruction file.
func TestCompileRules_Copilot_NoClaude_FullTranslation(t *testing.T) {
	dir := t.TempDir()

	r := copilot.New()
	rules := map[string]ast.RuleConfig{
		"my-rule": {Body: "Follow the rule.", Activation: "always"},
	}
	files, notes, err := r.CompileRules(rules, dir)
	require.NoError(t, err)
	assert.Contains(t, files, "instructions/my-rule.instructions.md",
		"full translation must write .github/instructions/ when .claude/ is absent")
	for _, n := range notes {
		assert.NotEqual(t, renderer.CodeClaudeNativePassthrough, n.Code,
			"no CLAUDE_NATIVE_PASSTHROUGH notes expected when .claude/ is absent")
	}
}

// mapKeys returns the sorted list of file keys for error output.
func mapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
