package gemini

import (
	"os"
	"path/filepath"
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
					Description:  "Go style guide",
					Instructions: "Use gofmt.",
					Activation:   ast.RuleActivationAlways,
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

	// No fidelity notes for always activation.
	assert.Empty(t, notes)
}

func TestCompile_Gemini_Rule_DescriptionAsProse(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"prose-rule": {
					Description:  "This is a prose description.",
					Instructions: "Rule content here.",
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
					Description:  "API style guide",
					Instructions: "Follow REST conventions.",
					Activation:   ast.RuleActivationPathGlob,
					Paths:        []string{"api/**"},
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

	// path-glob is supported — no fidelity note.
	assert.Empty(t, notes)
}

func TestCompile_Gemini_Rules_UnsupportedActivation(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"secret-rule": {
					Description:  "Manual rule",
					Instructions: "Only on demand.",
					Activation:   ast.RuleActivationManualMention,
				},
			},
		},
	}
	out, notes, err := renderer.Orchestrate(r, config, t.TempDir())
	require.NoError(t, err)

	// Rule still written.
	_, ok := out.Files["rules/secret-rule.md"]
	assert.True(t, ok, "expected rules/secret-rule.md (relative to OutputDir) even for unsupported activation")

	// Fidelity note must be emitted.
	require.Len(t, notes, 1)
	assert.Equal(t, renderer.CodeRuleActivationUnsupported, notes[0].Code)
	assert.Equal(t, renderer.LevelWarning, notes[0].Level)
	assert.Equal(t, "secret-rule", notes[0].Resource)
}

func TestCompile_Gemini_Rules_NoProjectInstructions(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"lint": {
					Instructions: "Run golangci-lint.",
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

func TestCompile_Gemini_FullConfig_InstructionsAndRules(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "style.md"), []byte("Style guide content."), 0o644)
	require.NoError(t, err)

	r := New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:                "integration-test",
			Instructions:        "Root project instructions.",
			InstructionsImports: []string{"./docs/contributing.md"},
			InstructionsScopes: []ast.InstructionsScope{
				{
					Path:         "packages/api",
					Instructions: "API scope instructions.",
				},
			},
		},
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"code-style": {
					Description:      "Code style.",
					InstructionsFile: "style.md",
				},
				"testing": {
					Description:  "Testing rules.",
					Instructions: "Always write tests.",
				},
			},
		},
	}
	out, notes, err := renderer.Orchestrate(r, config, tmpDir)
	require.NoError(t, err)
	assert.Empty(t, notes, "no fidelity notes expected for supported activations")

	root := out.RootFiles["GEMINI.md"]
	assert.Contains(t, root, "Root project instructions.")
	assert.Contains(t, root, "@./docs/contributing.md")
	assert.Contains(t, root, "Always apply @.gemini/rules/code-style.md")
	assert.Contains(t, root, "Always apply @.gemini/rules/testing.md")

	assert.Contains(t, out.RootFiles, "packages/api/GEMINI.md")
	assert.Contains(t, out.RootFiles["packages/api/GEMINI.md"], "API scope instructions.")

	assert.Contains(t, out.Files["rules/code-style.md"], "Style guide content.")
	assert.Contains(t, out.Files["rules/testing.md"], "Always write tests.")
}

func TestCompile_Gemini_PathTraversal(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"../evil": {
					Description:  "Malicious rule.",
					Instructions: "Bad content.",
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

func intPtr(i int) *int { return &i }

// TestCompile_Gemini_OutputPathsRelativeToOutputDir verifies that all keys in
// out.Files are relative to OutputDir() — no ".gemini/" prefix — so that the
// apply command's join of outputDir+relPath produces the correct final path
// (e.g. ".gemini/rules/foo.md", not ".gemini/.gemini/rules/foo.md").
// @-import lines in GEMINI.md must still use the project-relative ".gemini/"
// prefix so that the Gemini CLI can locate the rule files.
func TestCompile_Gemini_OutputPathsRelativeToOutputDir(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:         "path-test",
			Instructions: "Root instructions.",
		},
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"style": {Description: "Style.", Instructions: "Use gofmt."},
			},
			Skills: map[string]ast.SkillConfig{
				"tdd": {Name: "tdd", Description: "TDD workflow.", Instructions: "Test first."},
			},
			Agents: map[string]ast.AgentConfig{
				"helper": {Name: "helper", Description: "Helper agent.", Instructions: "You help."},
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

func TestCompile_Gemini_FullParity_AllKinds(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "agent-body.md"), []byte("Agent system prompt."), 0o644))

	r := New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:                "full-parity-test",
			Instructions:        "Root project instructions.",
			InstructionsImports: []string{"./docs/contributing.md"},
			InstructionsScopes: []ast.InstructionsScope{
				{Path: "packages/api", Instructions: "API scope."},
			},
		},
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"code-style": {Description: "Style rules.", Instructions: "Use gofmt."},
			},
			Skills: map[string]ast.SkillConfig{
				"tdd": {Name: "tdd", Description: "TDD workflow.", Instructions: "Test first."},
			},
			Agents: map[string]ast.AgentConfig{
				"helper": {
					Name: "helper", Description: "Helper agent.",
					Tools: []string{"read_file"}, Model: "opus-4",
					InstructionsFile: "agent-body.md",
				},
			},
			MCP: map[string]ast.MCPConfig{
				"github": {Command: "docker", Args: []string{"run", "-i", "ghcr.io/github/github-mcp-server"}},
			},
		},
		Hooks: map[string]ast.NamedHookConfig{
			"default": {
				Name: "default",
				Events: ast.HookConfig{
					"PreToolExecution": {
						{Matcher: "write_file", Hooks: []ast.HookHandler{
							{Type: "command", Command: "./hooks/check.sh", Timeout: intPtr(5000)},
						}},
					},
				},
			},
		},
	}

	out, notes, err := renderer.Orchestrate(r, config, tmpDir)
	require.NoError(t, err)

	// Instructions
	assert.Contains(t, out.RootFiles, "GEMINI.md")
	assert.Contains(t, out.RootFiles["GEMINI.md"], "Root project instructions.")
	assert.Contains(t, out.RootFiles["GEMINI.md"], "@./docs/contributing.md")
	assert.Contains(t, out.RootFiles, "packages/api/GEMINI.md")

	// Rules — key relative to OutputDir; @-import uses project-relative path.
	assert.Contains(t, out.Files, "rules/code-style.md")
	assert.Contains(t, out.RootFiles["GEMINI.md"], "Always apply @.gemini/rules/code-style.md")

	// Skills — key relative to OutputDir.
	assert.Contains(t, out.Files, "skills/tdd/SKILL.md")
	assert.Contains(t, out.Files["skills/tdd/SKILL.md"], "name: tdd")

	// Agents — key relative to OutputDir.
	assert.Contains(t, out.Files, "agents/helper.md")
	assert.Contains(t, out.Files["agents/helper.md"], "name: helper")
	assert.Contains(t, out.Files["agents/helper.md"], "Agent system prompt.")

	// Settings (hooks + MCP) — key relative to OutputDir.
	assert.Contains(t, out.Files, "settings.json")
	settingsJSON := out.Files["settings.json"]
	assert.Contains(t, settingsJSON, "BeforeTool")
	assert.Contains(t, settingsJSON, "github")

	// Should have zero fidelity notes for this supported config
	// (no unsupported fields used)
	assert.Empty(t, notes, "full-parity config with only supported fields should produce zero fidelity notes")
}

func TestCompileAgents_Gemini_ClaudeToolsDropped(t *testing.T) {
	r := New()
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"tester": {
					Name: "tester", Tools: []string{"Read", "Edit", "Bash"},
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
					Name: "tester", Tools: []string{"*"},
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
						{Name: "build", Instructions: "Run go build."},
						{Name: "test", Instructions: "Run go test."},
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
