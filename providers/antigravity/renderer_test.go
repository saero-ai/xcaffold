package antigravity_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/providers/antigravity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func findAgNote(notes []renderer.FidelityNote, code, field string) (renderer.FidelityNote, bool) {
	for _, n := range notes {
		if n.Code == code && (field == "" || n.Field == field) {
			return n, true
		}
	}
	return renderer.FidelityNote{}, false
}

// ─── Target / OutputDir / Render ─────────────────────────────────────────────

func TestRenderer_Target(t *testing.T) {
	r := antigravity.New()
	assert.Equal(t, "antigravity", r.Target())
}

func TestRenderer_OutputDir(t *testing.T) {
	r := antigravity.New()
	assert.Equal(t, ".agents", r.OutputDir())
}

// ─── Rule tests ───────────────────────────────────────────────────────────────

func TestCompile_Rule_OutputPathIsMarkdown(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"my-rule": {
					Description: "A rule",
					Body:        "Always format with gofmt.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	_, ok := out.Files["rules/my-rule.md"]
	assert.True(t, ok, "expected rules/my-rule.md in output")
}

func TestCompile_Rule_NoFrontmatterDelimiters(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"plain-rule": {
					Body: "Be concise.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/plain-rule.md"]
	require.NotEmpty(t, content)

	// AG rules must NOT have YAML frontmatter delimiters
	assert.False(t, strings.HasPrefix(content, "---"), "AG rules must not start with --- frontmatter delimiter")
	assert.NotContains(t, content, "---", "AG rules must contain no --- delimiters")
}

func TestCompile_Rule_DescriptionAsHeading(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"desc-rule": {
					Description: "My Rule Description",
					Body:        "Do something important.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/desc-rule.md"]
	require.NotEmpty(t, content)

	// Description must appear ONLY in YAML frontmatter, not as a # heading.
	assert.True(t, strings.HasPrefix(content, "---\n"), "AG rule with description must start with frontmatter delimiter")
	assert.Contains(t, content, "description: My Rule Description", "description must appear as YAML frontmatter field")
	assert.NotContains(t, content, "# My Rule Description", "description must NOT appear as a markdown heading")
}

func TestCompile_Rule_NoPathsOrGlobs(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"path-rule": {
					Description: "A rule with paths",
					Paths:       []string{"**/*.go"},
					Body:        "Check Go files.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/path-rule.md"]
	require.NotEmpty(t, content)

	// PathGlob activation must emit trigger + globs in frontmatter.
	assert.Contains(t, content, "trigger: glob", "PathGlob activation must emit trigger: glob")
	assert.Contains(t, content, "globs: **/*.go", "PathGlob activation must emit globs field")
	// No JSON array format — comma-separated string only.
	assert.NotContains(t, content, "[\"**/*.go\"]", "globs must not be a JSON array")
	// No old-style HTML comments.
	assert.NotContains(t, content, "<!-- xcaffold:paths", "old HTML-comment paths must not appear")
	assert.NotContains(t, content, "alwaysApply:", "alwaysApply must not appear in AG rules")
}

func TestCompile_Rule_BodyContentPreserved(t *testing.T) {
	r := antigravity.New()
	body := "Always run tests before committing.\nUse table-driven tests."
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"test-rule": {
					Body: body,
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/test-rule.md"]
	assert.Contains(t, content, "Always run tests before committing.")
	assert.Contains(t, content, "Use table-driven tests.")
}

func TestCompile_Rule_DescriptionHeadingPrecedesBody(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"order-rule": {
					Description: "Title",
					Body:        "Body content.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/order-rule.md"]
	// Frontmatter block must precede body content.
	frontmatterEnd := strings.Index(content, "---\n\n")
	bodyPos := strings.Index(content, "Body content.")
	require.True(t, frontmatterEnd >= 0, "must have closing frontmatter delimiter")
	assert.True(t, frontmatterEnd < bodyPos, "frontmatter block must precede body content")
	// Description in frontmatter, not as heading.
	assert.Contains(t, content, "description: Title")
	assert.NotContains(t, content, "# Title")
}

func TestCompile_Rule_NoDescription_NoHeading(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"no-desc-rule": {
					Body: "Just the body.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/no-desc-rule.md"]
	assert.False(t, strings.HasPrefix(content, "#"), "no heading when description is absent")
}

func TestCompile_Rule_12KCharacterLimitWarning(t *testing.T) {
	r := antigravity.New()
	// Build a body longer than 12,000 characters
	longBody := strings.Repeat("a", 12001)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"long-rule": {
					Body: longBody,
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/long-rule.md"]
	// A warning comment must be prepended
	assert.True(t, strings.HasPrefix(content, "<!--"), "long rule must begin with warning HTML comment")
	assert.Contains(t, content, "12000", "warning must mention the 12000-char limit")
}

func TestCompile_Rule_Under12K_NoWarning(t *testing.T) {
	r := antigravity.New()
	body := strings.Repeat("b", 11999)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"short-rule": {
					Body: body,
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/short-rule.md"]
	assert.NotContains(t, content, "WARNING:", "rules under 12K must not have 12K warning comment")
}

func TestCompile_Rule_EmptyID_ReturnsError(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"   ": {
					Body: "Bad rule.",
				},
			},
		},
	}

	_, _, err := renderer.Orchestrate(r, config, "")
	assert.Error(t, err)
}

func TestCompileRules_Antigravity_AlwaysOn_NoHtmlComments(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"policy": {
					Description: "Enforces security policy.",
					Body:        "Always check permissions.",
					Activation:  ast.RuleActivationAlways,
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/policy.md"]
	require.NotEmpty(t, content)

	// Must have YAML frontmatter with description.
	assert.True(t, strings.HasPrefix(content, "---\n"), "AlwaysOn rule with description must start with frontmatter")
	assert.Contains(t, content, "description: Enforces security policy.")
	// Must NOT have trigger field — AlwaysOn is the default.
	assert.NotContains(t, content, "trigger:")
	// Must NOT have any HTML comment activation markers.
	assert.NotContains(t, content, "<!-- xcaffold:activation")
	assert.NotContains(t, content, "<!-- xcaffold:")
}

func TestCompileRules_Antigravity_AlwaysOn_NoTriggerField(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"always-rule": {
					Description: "Always active.",
					Body:        "Be consistent.",
					Activation:  ast.RuleActivationAlways,
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/always-rule.md"]
	assert.NotContains(t, content, "trigger:", "AlwaysOn activation must not emit trigger field")
}

func TestCompileRules_Antigravity_AlwaysOn_EmptyDescription_NoFrontmatter(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"bare-rule": {
					Body:       "Just the body. No description.",
					Activation: ast.RuleActivationAlways,
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/bare-rule.md"]
	require.NotEmpty(t, content)

	// No description + AlwaysOn → needsFrontmatter=false → no --- block.
	assert.False(t, strings.HasPrefix(content, "---"), "AlwaysOn with empty description must not emit frontmatter")
	assert.NotContains(t, content, "---", "no frontmatter delimiters when needsFrontmatter=false")
	assert.Contains(t, content, "Just the body.")
}

func TestCompileRules_Antigravity_Glob_EmitsTriggerAndGlobs(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"go-files": {
					Description: "Go-specific rule.",
					Body:        "Use gofmt.",
					Activation:  ast.RuleActivationPathGlob,
					Paths:       []string{"xcaffold/**", "xcaffold/internal/**"},
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/go-files.md"]
	require.NotEmpty(t, content)

	assert.Contains(t, content, "trigger: glob\n", "PathGlob activation must emit 'trigger: glob'")
	assert.Contains(t, content, "globs: xcaffold/**,xcaffold/internal/**\n", "paths must be comma-joined in globs field")
	// No old HTML comment format.
	assert.NotContains(t, content, "<!-- xcaffold:activation Glob -->")
	assert.NotContains(t, content, "<!-- xcaffold:paths")
}

func TestCompileRules_Antigravity_ModelDecided_EmitsTrigger(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"smart-rule": {
					Description: "Applied when relevant.",
					Body:        "Be context-aware.",
					Activation:  ast.RuleActivationModelDecided,
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/smart-rule.md"]
	assert.Contains(t, content, "trigger: model_decision\n", "ModelDecided activation must emit 'trigger: model_decision'")
	assert.NotContains(t, content, "trigger: glob", "ModelDecided must not emit glob trigger")
}

func TestCompileRules_Antigravity_ManualMention_FidelityNote(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"manual-rule": {
					Description: "Only when mentioned.",
					Body:        "Only on explicit request.",
					Activation:  ast.RuleActivationManualMention,
				},
			},
		},
	}

	_, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	var found bool
	for _, n := range notes {
		if n.Code == renderer.CodeRuleActivationUnsupported && n.Resource == "manual-rule" {
			found = true
			assert.Equal(t, renderer.LevelWarning, n.Level)
			assert.Equal(t, "antigravity", n.Target)
			assert.Equal(t, "rule", n.Kind)
		}
	}
	assert.True(t, found, "ManualMention must emit CodeRuleActivationUnsupported fidelity note")
}

func TestCompileRules_Antigravity_ExplicitInvoke_FidelityNote(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"invoke-rule": {
					Body:       "Explicit invocation only.",
					Activation: ast.RuleActivationExplicitInvoke,
				},
			},
		},
	}

	_, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	var found bool
	for _, n := range notes {
		if n.Code == renderer.CodeRuleActivationUnsupported && n.Resource == "invoke-rule" {
			found = true
		}
	}
	assert.True(t, found, "ExplicitInvoke must emit CodeRuleActivationUnsupported fidelity note")
}

// ─── Skill tests ──────────────────────────────────────────────────────────────

func TestCompile_Skill_OutputAtCorrectPath(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"my-skill": {
					Name:        "My Skill",
					Description: "A test skill",
					Body:        "Do the skill thing.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	_, ok := out.Files["skills/my-skill/SKILL.md"]
	assert.True(t, ok, "expected skills/my-skill/SKILL.md in output")
}

func TestCompile_Skill_FrontmatterHasNameAndDescription(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"fmt-skill": {
					Name:        "Format Skill",
					Description: "Formats code",
					Body:        "Run gofmt first.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["skills/fmt-skill/SKILL.md"]
	require.NotEmpty(t, content)

	assert.Contains(t, content, "name: Format Skill")
	assert.Contains(t, content, "description: Formats code")
}

func TestCompile_Skill_FrontmatterDelimitersPresent(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"delim-skill": {
					Name: "Delim Skill",
					Body: "Some body.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["skills/delim-skill/SKILL.md"]
	assert.True(t, strings.HasPrefix(content, "---\n"), "skill must start with frontmatter delimiter")
	assert.Contains(t, content, "\n---\n", "skill must have closing frontmatter delimiter")
}

func TestCompile_Skill_CCOnlyFieldsDropped(t *testing.T) {
	// Create actual files so CompileSkillSubdir can read them.
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "refs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "refs", "guide.go"), []byte("// guide"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "setup.sh"), []byte("#!/bin/sh"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "icon.png"), []byte("PNG"), 0o644))

	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"rich-skill": {
					Name:         "Rich Skill",
					Description:  "Has many fields.",
					Body:         "Do something.",
					AllowedTools: []string{"Bash"},
					References:   []string{"refs/guide.go"},
					Scripts:      []string{"setup.sh"},
					Assets:       []string{"icon.png"},
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, tmpDir)
	require.NoError(t, err)

	content := out.Files["skills/rich-skill/SKILL.md"]
	require.NotEmpty(t, content)

	// CC-only and target-specific fields must NOT appear in SKILL.md frontmatter.
	for _, dropped := range []string{
		"tools:", "references:", "scripts:", "assets:",
	} {
		assert.NotContains(t, content, dropped, "field %q must be dropped for AG SKILL.md frontmatter", dropped)
	}

	// Only name and description allowed in frontmatter
	assert.Contains(t, content, "name: Rich Skill")
	assert.Contains(t, content, "description: Has many fields.")
}

func TestCompile_Skill_References_CompiledToExamples(t *testing.T) {
	// Antigravity now compiles references/ → examples/ instead of dropping them.
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "refs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "refs", "doc.md"), []byte("# Doc"), 0o644))

	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"test-skill": {
					Name:        "test-skill",
					Description: "A skill with references",
					Body:        "Do things.",
					References:  []string{"refs/doc.md"},
				},
			},
		},
	}
	files, notes, err := renderer.Orchestrate(r, config, tmpDir)
	require.NoError(t, err)

	// References are compiled to examples/ (name translation)
	_, ok := files.Files["skills/test-skill/examples/doc.md"]
	assert.True(t, ok, "expected references compiled to examples/ subdirectory")

	// No drop note should be emitted
	for _, n := range notes {
		assert.NotEqual(t, renderer.CodeSkillReferencesDropped, n.Code, "references must not produce a drop note")
	}
}

func TestCompile_Skill_BodyContentPreserved(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"body-skill": {
					Name: "Body Skill",
					Body: "Step 1: do this.\nStep 2: do that.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["skills/body-skill/SKILL.md"]
	assert.Contains(t, content, "Step 1: do this.")
	assert.Contains(t, content, "Step 2: do that.")
}

func TestCompile_Skill_EmptyID_ReturnsError(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"   ": {
					Body: "Bad skill.",
				},
			},
		},
	}

	_, _, err := renderer.Orchestrate(r, config, "")
	assert.Error(t, err)
}

// ─── Silent-skip tests ────────────────────────────────────────────────────────

func TestCompile_AgentsAndHooks_AreNotEmitted(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		Hooks: map[string]ast.NamedHookConfig{
			"default": {
				Name: "default",
				Events: ast.HookConfig{
					"PreToolUse": {
						{
							Matcher: "Bash",
							Hooks: []ast.HookHandler{
								{Type: "command", Command: "echo hi"},
							},
						},
					},
				},
			},
		},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"my-agent": {
					Name: "My Agent",
					Body: "Do work.",
				},
			},
			Rules: map[string]ast.RuleConfig{
				"a-rule": {
					Body: "A rule body.",
				},
			},
			Skills: map[string]ast.SkillConfig{
				"a-skill": {
					Name: "A Skill",
					Body: "A skill body.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	// Rules and skills must be emitted
	assert.Contains(t, out.Files, "rules/a-rule.md", "rules must be compiled")
	assert.Contains(t, out.Files, "skills/a-skill/SKILL.md", "skills must be compiled")

	// Agents and hooks must be silently skipped
	for path := range out.Files {
		assert.False(t, strings.HasPrefix(path, "agents/"), "agents must not be emitted for AG target")
		assert.NotEqual(t, "hooks.json", path, "hooks must not be emitted for AG target")
	}
}

// TestCompile_MCP_NoProjectLocalFile verifies that Antigravity does NOT emit
// mcp_config.json into the project output directory. Antigravity reads MCP
// config from the global user path (~/.gemini/antigravity/mcp_config.json)
// only; a project-local file is silently ignored by the tool.
func TestCompile_MCP_NoProjectLocalFile(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			MCP: map[string]ast.MCPConfig{
				"my-server": {
					Command: "npx",
					Args:    []string{"-y", "my-mcp", "--test"},
					Env: map[string]string{
						"DB": "pg",
					},
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	_, ok := out.Files["mcp_config.json"]
	assert.False(t, ok, "mcp_config.json must NOT be written; Antigravity only reads MCP config from ~/.gemini/antigravity/mcp_config.json")
}

// TestCompile_MCP_EmitsGlobalConfigOnlyNote verifies that a MCP_GLOBAL_CONFIG_ONLY
// fidelity note is emitted when MCP servers are declared. The note directs the
// user to configure MCP via the Antigravity UI or by editing the global config file.
func TestCompile_MCP_EmitsGlobalConfigOnlyNote(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			MCP: map[string]ast.MCPConfig{
				"my-server": {
					Command: "npx",
					Args:    []string{"-y", "my-mcp"},
				},
			},
		},
	}

	_, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	note, ok := findAgNote(notes, renderer.CodeMCPGlobalConfigOnly, "mcp")
	require.True(t, ok, "expected a MCP_GLOBAL_CONFIG_ONLY fidelity note")
	assert.Equal(t, "antigravity", note.Target)
}

// TestCompile_MCP_HTTPServer_NoProjectLocalFile verifies that HTTP-transport MCP
// servers also do not produce a project-local mcp_config.json for Antigravity.
func TestCompile_MCP_HTTPServer_NoProjectLocalFile(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			MCP: map[string]ast.MCPConfig{
				"http-server": {URL: "https://mcp.example.com/v1"},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	_, ok := out.Files["mcp_config.json"]
	assert.False(t, ok, "mcp_config.json must NOT be written for HTTP-transport servers either")
}

func TestCompile_EmptyConfig_ReturnsEmptyOutput(t *testing.T) {
	r := antigravity.New()
	out, _, err := renderer.Orchestrate(r, &ast.XcaffoldConfig{}, "")
	require.NoError(t, err)
	assert.Empty(t, out.Files)
}

// ─── Workflow tests (Gap 6) ──────────────────────────────────────────────────

func TestCompile_Workflow_OutputAtCorrectPath(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Workflows: map[string]ast.WorkflowConfig{
				"commit-changes": {
					Name:        "Commit Changes",
					Description: "How to commit changes",
					Body:        "1. Stage files\n2. Commit",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	_, ok := out.Files["workflows/commit-changes.md"]
	assert.True(t, ok, "expected workflows/commit-changes.md in output")
}

func TestCompile_Workflow_FrontmatterContainsDescription(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Workflows: map[string]ast.WorkflowConfig{
				"deploy": {
					Description: "Deploy to production",
					Body:        "Run deploy script.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["workflows/deploy.md"]
	require.NotEmpty(t, content)

	assert.True(t, strings.HasPrefix(content, "---\n"), "workflow must start with frontmatter")
	assert.Contains(t, content, "description: Deploy to production")
}

func TestCompile_Workflow_NameFallbackToDescription(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Workflows: map[string]ast.WorkflowConfig{
				"build": {
					Name: "Build Project",
					Body: "Run make build.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["workflows/build.md"]
	assert.Contains(t, content, "description: Build Project", "name should fall back to description")
}

func TestCompile_Workflow_BodyPreserved(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Workflows: map[string]ast.WorkflowConfig{
				"test": {
					Description: "Run tests",
					Body:        "1. Run unit tests\n2. Run integration tests\n3. Check coverage",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["workflows/test.md"]
	assert.Contains(t, content, "1. Run unit tests")
	assert.Contains(t, content, "2. Run integration tests")
	assert.Contains(t, content, "3. Check coverage")
}

func TestCompile_Workflow_EmptyWorkflowsNoOutput(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Workflows: map[string]ast.WorkflowConfig{},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	for path := range out.Files {
		assert.False(t, strings.HasPrefix(path, "workflows/"), "empty workflows map must not produce output")
	}
}

func TestCompile_Workflow_EmptyID_ReturnsError(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Workflows: map[string]ast.WorkflowConfig{
				"   ": {
					Body: "Bad workflow.",
				},
			},
		},
	}

	_, _, err := renderer.Orchestrate(r, config, "")
	assert.Error(t, err)
}

func TestCompile_Skill_WithDoubleQuotes_ProperlyEscapes(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"quoted": {
					Name:        `Skill with "quotes"`,
					Description: `A "very specal" skill`,
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["skills/quoted/SKILL.md"]
	assert.Contains(t, content, `name: "Skill with \"quotes\""`)
	assert.Contains(t, content, `description: "A \"very specal\" skill"`)
	assert.NotContains(t, content, `\\\"`, "Must not double-escape quotes")
}

// ─── Fidelity note tests ──────────────────────────────────────────────────────

func TestAntigravityRenderer_PermissionsSetting_EmitsNote(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		Settings: map[string]ast.SettingsConfig{"default": {
			Permissions: &ast.PermissionsConfig{Allow: []string{"Read"}},
		}},
	}

	_, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	note, ok := findAgNote(notes, renderer.CodeSettingsFieldUnsupported, "permissions")
	require.True(t, ok)
	assert.Equal(t, "antigravity", note.Target)
}

func TestAntigravityRenderer_SandboxSetting_EmitsNote(t *testing.T) {
	r := antigravity.New()
	enabled := true
	config := &ast.XcaffoldConfig{
		Settings: map[string]ast.SettingsConfig{"default": {
			Sandbox: &ast.SandboxConfig{Enabled: &enabled},
		}},
	}

	_, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	_, ok := findAgNote(notes, renderer.CodeSettingsFieldUnsupported, "sandbox")
	assert.True(t, ok)
}

func TestAntigravityRenderer_AgentSecurityFields_EmitsPerFieldNotes(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {
					Body:            "Build things.",
					PermissionMode:  "plan",
					DisallowedTools: ast.ClearableList{Values: []string{"Write"}},
					Isolation:       "container",
				},
			},
		},
	}

	_, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	// Security field checks are now centralized in the orchestrator via
	// CheckFieldSupport, which emits FIELD_UNSUPPORTED with YAML key names.
	_, pm := findAgNote(notes, renderer.CodeFieldUnsupported, "permission-mode")
	_, dt := findAgNote(notes, renderer.CodeFieldUnsupported, "disallowed-tools")
	_, iso := findAgNote(notes, renderer.CodeFieldUnsupported, "isolation")
	assert.True(t, pm, "permission-mode note must be emitted")
	assert.True(t, dt, "disallowed-tools note must be emitted")
	assert.True(t, iso, "isolation note must be emitted")
}

// The RENDERER_KIND_UNSUPPORTED note for the dropped agent is always emitted
// by CompileAgents regardless of suppress-fidelity-warnings.
func TestAntigravityRenderer_SuppressFidelityWarnings_NotesStillReturned(t *testing.T) {
	r := antigravity.New()
	suppress := true
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {
					Body:           "Build things.",
					PermissionMode: "plan",
					Isolation:      "container",
					Targets: map[string]ast.TargetOverride{
						"antigravity": {SuppressFidelityWarnings: &suppress},
					},
				},
			},
		},
	}

	_, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)
	assert.NotEmpty(t, notes, "renderer returns notes; suppression is applied at the command layer")
}

// TestAntigravityRenderer_MCPDeclared_EmitsGlobalConfigOnlyNote verifies that
// any MCP declaration — regardless of env var interpolation or transport type —
// produces a MCP_GLOBAL_CONFIG_ONLY note rather than an interpolation warning.
// Antigravity reads MCP config from a global path only, so project-local
// compilation is not performed and per-entry env inspection is skipped.
func TestAntigravityRenderer_MCPDeclared_EmitsGlobalConfigOnlyNote(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			MCP: map[string]ast.MCPConfig{
				"my-server": {
					Command: "node",
					Env:     map[string]string{"TOKEN": "${MY_TOKEN}"},
				},
			},
		},
	}

	_, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	_, ok := findAgNote(notes, renderer.CodeMCPGlobalConfigOnly, "mcp")
	assert.True(t, ok, "expected MCP_GLOBAL_CONFIG_ONLY note for MCP declarations on Antigravity target")
}

func TestAntigravityRenderer_SkillScripts_CompiledToScripts(t *testing.T) {
	// Antigravity now compiles scripts/ instead of dropping them.
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "scripts"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "scripts", "install.sh"), []byte("#!/bin/sh\necho hi"), 0o755))

	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"setup": {
					Description: "Env setup.",
					Scripts:     []string{"scripts/install.sh"},
				},
			},
		},
	}
	files, notes, err := renderer.Orchestrate(r, config, tmpDir)
	require.NoError(t, err)

	_, ok := files.Files["skills/setup/scripts/install.sh"]
	assert.True(t, ok, "expected scripts compiled to scripts/ subdirectory")

	for _, n := range notes {
		assert.NotEqual(t, renderer.CodeSkillScriptsDropped, n.Code, "scripts must not produce a drop note")
	}
}

func TestAntigravityRenderer_SkillAssets_CompiledToResources(t *testing.T) {
	// Antigravity now compiles assets/ → resources/ instead of dropping them.
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "assets"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "assets", "logo.svg"), []byte("<svg/>"), 0o644))

	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"branding": {
					Description: "Brand assets.",
					Assets:      []string{"assets/logo.svg"},
				},
			},
		},
	}
	files, notes, err := renderer.Orchestrate(r, config, tmpDir)
	require.NoError(t, err)

	_, ok := files.Files["skills/branding/resources/logo.svg"]
	assert.True(t, ok, "expected assets compiled to resources/ subdirectory")

	for _, n := range notes {
		assert.NotEqual(t, renderer.CodeSkillAssetsDropped, n.Code, "assets must not produce a drop note")
	}
}

// ─── Activation provenance comment tests ─────────────────────────────────────

func TestCompileAntigravityRule_Activation_AlwaysOn(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"security": {
					Description: "Security checklist.",
					Activation:  ast.RuleActivationAlways,
					Body:        "Follow OWASP.",
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/security.md"]
	// AlwaysOn has description -> should have frontmatter with description but NO trigger.
	require.NotContains(t, content, "trigger:")
	require.NotContains(t, content, "<!-- xcaffold:activation")
}

func TestCompileAntigravityRule_Activation_Manual(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"commit-style": {
					Activation: ast.RuleActivationManualMention,
					Body:       "Body.",
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/commit-style.md"]
	// ManualMention has no description -> should have NO frontmatter.
	require.NotContains(t, content, "---")
	require.NotContains(t, content, "<!-- xcaffold:activation")
}

func TestCompileAntigravityRule_Activation_Glob_WithPaths(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"api-style": {
					Activation: ast.RuleActivationPathGlob,
					Paths:      []string{"src/**", "packages/api/**"},
					Body:       "REST conventions.",
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/api-style.md"]
	// Glob should have trigger: glob and globs: src/**,packages/api/**
	require.Contains(t, content, "trigger: glob")
	require.Contains(t, content, "globs: src/**,packages/api/**")
	require.NotContains(t, content, "<!-- xcaffold:activation")
}

func TestCompile_Agents_EmitsKindUnsupported(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"test-agent": {
					Name:        "test-agent",
					Description: "A test agent",
					Body:        "Do things.",
				},
			},
		},
	}
	r := antigravity.New()
	_, notes, err := renderer.Orchestrate(r, config, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, n := range notes {
		if n.Code == renderer.CodeRendererKindUnsupported && n.Resource == "test-agent" {
			found = true
		}
	}
	if !found {
		t.Error("expected RENDERER_KIND_UNSUPPORTED fidelity note for agent in antigravity")
	}
}

// ─── Skill subdirectory compilation tests ────────────────────────────────────

func TestCompile_SkillWithSubdirs_Antigravity(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "assets", "TEMPLATE.md"), []byte("# Template"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "refs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "refs", "guide.md"), []byte("# Guide"), 0o644); err != nil {
		t.Fatal(err)
	}

	skills := map[string]ast.SkillConfig{
		"my-skill": {
			Description: "test",
			Body:        "Do the thing.",
			Assets:      []string{"assets/TEMPLATE.md"},
			References:  []string{"refs/guide.md"},
		},
	}

	r := antigravity.New()
	files, notes, err := r.CompileSkills(skills, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Antigravity: assets → resources
	if _, ok := files["skills/my-skill/resources/TEMPLATE.md"]; !ok {
		keys := make([]string, 0, len(files))
		for k := range files {
			keys = append(keys, k)
		}
		t.Errorf("expected assets mapped to resources/, got keys: %v", keys)
	}
	// Antigravity: references → examples
	if _, ok := files["skills/my-skill/examples/guide.md"]; !ok {
		keys := make([]string, 0, len(files))
		for k := range files {
			keys = append(keys, k)
		}
		t.Errorf("expected references mapped to examples/, got keys: %v", keys)
	}
	// Should NOT have "dropped" fidelity notes for assets/references anymore
	for _, n := range notes {
		if n.Code == renderer.CodeSkillAssetsDropped || n.Code == renderer.CodeSkillReferencesDropped {
			t.Errorf("unexpected drop note: %v", n)
		}
	}
}

func TestCompileAntigravityRule_NoProvenance_ExistingBehaviorPreserved(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"legacy": {
					Description: "Legacy rule.",
					Body:        "No activation field set.",
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/legacy.md"]
	// No explicit activation → AlwaysOn (the default) → description in frontmatter only.
	assert.True(t, strings.HasPrefix(content, "---\n"), "AlwaysOn rule with description must start with frontmatter")
	assert.Contains(t, content, "description: Legacy rule.", "description must appear as YAML frontmatter field")
	assert.NotContains(t, content, "<!-- xcaffold:activation AlwaysOn -->", "HTML comment activation must be removed")
	assert.NotContains(t, content, "# Legacy rule.", "description must not appear as # heading")
}

// ─── Project instructions tests ───────────────────────────────────────────────

func TestCompile_ProjectInstructions_EmitsGeminiMdAsRootFile(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name: "test-project",
		},
		ResourceScope: ast.ResourceScope{
			Contexts: map[string]ast.ContextConfig{
				"root": {Body: "You are a test project."},
			},
		},
	}

	outputDirFiles, rootFiles, notes, err := r.CompileProjectInstructions(config, "/tmp/test")
	if err != nil {
		t.Fatal(err)
	}

	// Root files should contain GEMINI.md
	content, ok := rootFiles["GEMINI.md"]
	if !ok {
		t.Fatal("expected GEMINI.md in rootFiles, got none")
	}
	if !strings.Contains(content, "You are a test project.") {
		t.Errorf("GEMINI.md content missing project instructions, got: %s", content)
	}

	// Output-dir files should NOT contain the old path
	if _, ok := outputDirFiles["rules/project-instructions.md"]; ok {
		t.Error("rules/project-instructions.md should no longer be in outputDirFiles")
	}

	// Should not have error-level notes
	for _, n := range notes {
		if n.Level == renderer.LevelError {
			t.Errorf("unexpected error note: %s", n.Reason)
		}
	}
}
