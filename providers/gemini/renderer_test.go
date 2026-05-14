package gemini

import (
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompile_Gemini_Target(t *testing.T) {
	r := New()
	assert.Equal(t, "gemini", r.Target())
}

func TestCompile_Gemini_OutputDir(t *testing.T) {
	r := New()
	assert.Equal(t, ".gemini", r.OutputDir())
}

func TestCompile_Gemini_EmptyConfig(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{}
	out, notes, err := renderer.Orchestrate(r, config, t.TempDir())
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Empty(t, out.Files)
	assert.Empty(t, notes)
}

// Task 8 rule tests.

func TestCompile_Gemini_Rules_AlwaysActivation(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"go-style": {
					Description: "Go style guide",
					Body:        "Use gofmt.",
					Activation:  ast.RuleActivationAlways,
				},
			},
		},
	}
	out, notes, err := renderer.Orchestrate(r, config, t.TempDir())
	require.NoError(t, err)

	// Rule file must exist at the OutputDir-relative path.
	ruleContent, ok := out.Files["rules/go-style.md"]
	assert.True(t, ok, "expected rules/go-style.md (relative to OutputDir)")
	assert.Contains(t, ruleContent, "Use gofmt.")

	// GEMINI.md must contain targeted @-import with the project-relative path.
	geminiContent := out.RootFiles["GEMINI.md"]
	assert.Contains(t, geminiContent, "Always apply @.gemini/rules/go-style.md")

	// No activation-related fidelity notes for always activation. FIELD_UNSUPPORTED
	// notes for description/activation are expected since those fields are not
	// natively supported by Gemini.
	for _, n := range notes {
		if n.Code == renderer.CodeRuleActivationUnsupported {
			t.Errorf("unexpected activation fidelity note: %+v", n)
		}
	}
}

func TestCompile_Gemini_Rule_DescriptionAsProse(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"prose-rule": {
					Description: "This is a prose description.",
					Body:        "Rule content here.",
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, t.TempDir())
	require.NoError(t, err)

	content := out.Files["rules/prose-rule.md"]
	// Verify description is emitted as plain prose, NOT as a heading.
	assert.NotContains(t, content, "# This is a prose description.", "description must not be rendered as an H1 heading")
	assert.Contains(t, content, "This is a prose description.\n\n", "description must be rendered as a plain prose paragraph followed by an empty line")
	assert.Contains(t, content, "Rule content here.", "instructions must be preserved")
}

func TestCompile_Gemini_Rules_PathGlob(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"api-style": {
					Description: "API style guide",
					Body:        "Follow REST conventions.",
					Activation:  ast.RuleActivationPathGlob,
					Paths:       ast.ClearableList{Values: []string{"api/**"}},
				},
			},
		},
	}
	out, notes, err := renderer.Orchestrate(r, config, t.TempDir())
	require.NoError(t, err)

	_, ok := out.Files["rules/api-style.md"]
	assert.True(t, ok, "expected rules/api-style.md (relative to OutputDir)")

	geminiContent := out.RootFiles["GEMINI.md"]
	assert.Contains(t, geminiContent, "Apply this rule when accessing api/**:\n@.gemini/rules/api-style.md")

	// path-glob is supported — no activation-related fidelity note.
	// FIELD_UNSUPPORTED notes for description/activation/paths are expected.
	for _, n := range notes {
		if n.Code == renderer.CodeRuleActivationUnsupported {
			t.Errorf("unexpected activation fidelity note: %+v", n)
		}
	}
}

func TestCompile_Gemini_Rules_UnsupportedActivation(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"secret-rule": {
					Description: "Manual rule",
					Body:        "Only on demand.",
					Activation:  ast.RuleActivationManualMention,
				},
			},
		},
	}
	out, notes, err := renderer.Orchestrate(r, config, t.TempDir())
	require.NoError(t, err)

	// Rule still written.
	_, ok := out.Files["rules/secret-rule.md"]
	assert.True(t, ok, "expected rules/secret-rule.md (relative to OutputDir) even for unsupported activation")

	// Activation unsupported fidelity note must be emitted. Additional
	// FIELD_UNSUPPORTED notes for description/activation are expected.
	var found bool
	for _, n := range notes {
		if n.Code == renderer.CodeRuleActivationUnsupported {
			found = true
			assert.Equal(t, renderer.LevelWarning, n.Level)
			assert.Equal(t, "secret-rule", n.Resource)
		}
	}
	assert.True(t, found, "expected RULE_ACTIVATION_UNSUPPORTED fidelity note")
}

func TestCompile_Gemini_Rules_NoProjectInstructions(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"lint": {
					Body: "Run golangci-lint.",
				},
			},
		},
	}
	out, notes, err := renderer.Orchestrate(r, config, t.TempDir())
	require.NoError(t, err)
	assert.Empty(t, notes)

	// No project — GEMINI.md content comes from rules @-imports only.
	geminiContent := out.RootFiles["GEMINI.md"]
	assert.Contains(t, geminiContent, "Always apply @.gemini/rules/lint.md")
}

func TestCompile_Gemini_PathTraversal(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"../evil": {
					Description: "Malicious rule.",
					Body:        "Bad content.",
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "/tmp/test")
	require.NoError(t, err)
	for path := range out.Files {
		if path != "../GEMINI.md" {
			assert.NotContains(t, path, "..", "output path must not contain traversal sequences")
		}
	}
}

// TestCompile_Gemini_OutputPathsRelativeToOutputDir verifies that all keys in
// out.Files are relative to OutputDir() — no ".gemini/" prefix — so that the
// apply command's join of outputDir+relPath produces the correct final path
// (e.g. ".gemini/rules/foo.md", not ".gemini/.gemini/rules/foo.md").
// @-import lines in GEMINI.md must still use the project-relative ".gemini/"
// prefix so that the Gemini CLI can locate the rule files.
func TestCompile_Gemini_OutputPathsRelativeToOutputDir(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "path-test"},
		ResourceScope: ast.ResourceScope{
			Contexts: map[string]ast.ContextConfig{
				"root": {Body: "Root instructions."},
			},
			Rules: map[string]ast.RuleConfig{
				"style": {Description: "Style.", Body: "Use gofmt."},
			},
			Skills: map[string]ast.SkillConfig{
				"tdd": {Name: "tdd", Description: "TDD workflow.", Body: "Test first."},
			},
			Agents: map[string]ast.AgentConfig{
				"helper": {Name: "helper", Description: "Helper agent.", Body: "You help."},
			},
			MCP: map[string]ast.MCPConfig{
				"github": {Command: "docker", Args: []string{"run", "ghcr.io/github/github-mcp-server"}},
			},
		},
		Hooks: map[string]ast.NamedHookConfig{
			"default": {
				Name: "default",
				Events: ast.HookConfig{
					"PreToolExecution": {
						{Matcher: "write_file", Hooks: []ast.HookHandler{
							{Type: "command", Command: "./hooks/check.sh"},
						}},
					},
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, t.TempDir())
	require.NoError(t, err)

	// No out.Files key may have ".gemini/" as a path segment prefix.
	// OutputDir() already returns ".gemini"; the apply layer joins them.
	for path := range out.Files {
		assert.False(t,
			strings.HasPrefix(path, ".gemini/"),
			"out.Files key %q must not start with .gemini/ — paths must be relative to OutputDir()", path,
		)
	}

	// Specific keys that must exist without the prefix.
	assert.Contains(t, out.Files, "rules/style.md", "rule path must be rules/<id>.md")
	assert.Contains(t, out.Files, "skills/tdd/SKILL.md", "skill path must be skills/<id>/SKILL.md")
	assert.Contains(t, out.Files, "agents/helper.md", "agent path must be agents/<id>.md")
	assert.Contains(t, out.Files, "settings.json", "settings path must be settings.json")

	// @-import lines in GEMINI.md must use the project-relative path so the
	// Gemini CLI resolves the file correctly from the project root.
	geminiMD := out.RootFiles["GEMINI.md"]
	assert.Contains(t, geminiMD, "Always apply @.gemini/rules/style.md",
		"@-import lines must use the project-relative .gemini/ prefix")
}

func TestCompileAgents_Gemini_ClaudeToolsDropped(t *testing.T) {
	r := New()
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"tester": {
					Name: "tester", Tools: ast.ClearableList{Values: []string{"Read", "Edit", "Bash"}},
				},
			},
		},
	}
	out, notes, err := renderer.Orchestrate(r, cfg, t.TempDir())
	require.NoError(t, err)

	content := out.Files["agents/tester.md"]
	assert.NotContains(t, content, "tools:")

	found := false
	for _, n := range notes {
		if n.Code == renderer.CodeAgentToolsDropped {
			found = true
		}
	}
	assert.True(t, found, "expected CodeAgentToolsDropped")
}

func TestCompileAgents_Gemini_WildcardTools_Emitted(t *testing.T) {
	r := New()
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"tester": {
					Name: "tester", Tools: ast.ClearableList{Values: []string{"*"}},
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, cfg, t.TempDir())
	require.NoError(t, err)

	content := out.Files["agents/tester.md"]
	assert.Contains(t, content, "tools:\n  - *")
}

func TestCompileAgents_Gemini_ClaudeAliasModel_Omitted(t *testing.T) {
	r := New()
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"tester": {
					Name: "tester", Model: "sonnet",
				},
			},
		},
	}
	out, notes, err := renderer.Orchestrate(r, cfg, t.TempDir())
	require.NoError(t, err)

	content := out.Files["agents/tester.md"]
	assert.NotContains(t, content, "model:")

	found := false
	for _, n := range notes {
		if n.Code == renderer.CodeAgentModelUnmapped && n.Level == renderer.LevelWarning {
			found = true
		}
	}
	assert.True(t, found, "expected LevelWarning CodeAgentModelUnmapped")
}

func TestCompileAgents_Gemini_MappedAlias_Translated(t *testing.T) {
	r := New()
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"tester": {
					Name: "tester", Model: "sonnet-4",
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, cfg, t.TempDir())
	require.NoError(t, err)

	content := out.Files["agents/tester.md"]
	assert.Contains(t, content, "model: gemini-2.5-flash")
}

func TestCompile_Gemini_Workflows_LoweredToRulePlusSkill(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Workflows: map[string]ast.WorkflowConfig{
				"deploy": {
					Name:        "deploy",
					Description: "Deploy to production.",
					Steps: []ast.WorkflowStep{
						{Name: "build", Body: "Run go build."},
						{Name: "test", Body: "Run go test."},
					},
					Targets: map[string]ast.TargetOverride{
						"gemini": {Provider: map[string]any{"lowering-strategy": "rule-plus-skill"}},
					},
				},
			},
		},
	}
	out, notes, err := renderer.Orchestrate(r, config, "/tmp/test")
	require.NoError(t, err)

	// Should have lowered workflow to rules + skills (paths relative to OutputDir).
	hasRule := false
	hasSkill := false
	for path := range out.Files {
		if strings.HasPrefix(path, "rules/") {
			hasRule = true
		}
		if strings.HasPrefix(path, "skills/") {
			hasSkill = true
		}
	}
	assert.True(t, hasRule, "lowered workflow should produce at least one rule file")
	assert.True(t, hasSkill, "lowered workflow should produce at least one skill file")

	// Should have workflow lowering fidelity note
	hasLoweringNote := false
	for _, n := range notes {
		if n.Code == renderer.CodeWorkflowLoweredToRulePlusSkill {
			hasLoweringNote = true
		}
	}
	assert.True(t, hasLoweringNote, "expected CodeWorkflowLoweredToRulePlusSkill fidelity note")
}

// TestCompile_Gemini_Workflows_DefaultSimpleMode verifies that a workflow
// without an explicit lowering-strategy uses the new default behavior: structure-based
// inference produces a single skill file (simple mode) rather than per-step skills
// or a rule. The test exercises the new default-behavior path and verifies the
// migration fidelity notes are emitted.
func TestCompile_Gemini_Workflows_DefaultSimpleMode(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Workflows: map[string]ast.WorkflowConfig{
				"code-review": {
					Name:        "code-review",
					Description: "Multi-step pull request review procedure.",
					Steps: []ast.WorkflowStep{
						{Name: "analyze", Body: "Read the diff."},
						{Name: "lint", Body: "Check style."},
						{Name: "summarize", Body: "Write the review."},
					},
					// No Targets, no lowering-strategy — new default applies.
				},
			},
		},
	}
	out, notes, err := renderer.Orchestrate(r, config, "/tmp/test")
	require.NoError(t, err)

	// New default: single skill file (NOT per-step micro-skills).
	_, hasSkill := out.Files["skills/code-review/SKILL.md"]
	require.True(t, hasSkill, "expected skills/code-review/SKILL.md for simple mode; got keys: %v", mapKeysGemini(out.Files))

	// No rule file should exist (no always-apply or paths set).
	_, hasRule := out.Files["rules/code-review-workflow.md"]
	require.False(t, hasRule, "simple mode without always-apply should NOT emit a rule; got keys: %v", mapKeysGemini(out.Files))

	// Should have CodeWorkflowSimpleToSections note.
	var hasSimpleNote bool
	for _, n := range notes {
		if n.Code == renderer.CodeWorkflowSimpleToSections {
			hasSimpleNote = true
		}
	}
	require.True(t, hasSimpleNote, "expected CodeWorkflowSimpleToSections note; got: %v", notes)

	// Should have CodeWorkflowDefaultChanged migration warning.
	var hasMigrationNote bool
	for _, n := range notes {
		if n.Code == renderer.CodeWorkflowDefaultChanged {
			hasMigrationNote = true
		}
	}
	require.True(t, hasMigrationNote, "expected CodeWorkflowDefaultChanged migration note; got: %v", notes)
}

// TestCompile_Gemini_Workflow_CustomCommand_NoPathDoubling verifies that a workflow
// lowered to a custom-command primitive is stored with a path relative to OutputDir
// ("commands/<name>.md") and NOT with the full provider prefix (".gemini/commands/<name>.md").
// Without the fix, apply.go would join OutputDir + ".gemini/commands/tdd.md", producing
// ".gemini/.gemini/commands/tdd.md" on disk.
func TestCompile_Gemini_Workflow_CustomCommand_NoPathDoubling(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Workflows: map[string]ast.WorkflowConfig{
				"tdd": {
					Name: "tdd",
					Steps: []ast.WorkflowStep{
						{Name: "red", Body: "Write a failing test."},
					},
					Targets: map[string]ast.TargetOverride{
						"gemini": {
							Provider: map[string]interface{}{
								"lowering-strategy": "custom-command",
							},
						},
					},
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, t.TempDir())
	require.NoError(t, err)

	// Relative path (correct — apply.go will prepend .gemini/).
	_, ok := out.Files["commands/tdd.md"]
	assert.True(t, ok, "workflow custom-command path must be relative: \"commands/tdd.md\"")

	// Absolute path with provider prefix (the doubled path — must not appear).
	_, doubled := out.Files[".gemini/commands/tdd.md"]
	assert.False(t, doubled, "workflow custom-command path must NOT include \".gemini/\" prefix")
}

// mapKeysGemini returns the sorted list of file keys for error output.
func mapKeysGemini(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
