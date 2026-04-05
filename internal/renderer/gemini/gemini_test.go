package gemini_test

import (
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer/gemini"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Target / OutputDir / Render ─────────────────────────────────────────────

func TestRenderer_Target(t *testing.T) {
	r := gemini.New()
	assert.Equal(t, "gemini", r.Target())
}

func TestRenderer_OutputDir(t *testing.T) {
	r := gemini.New()
	assert.Equal(t, ".agents", r.OutputDir())
}

func TestRenderer_Render_Identity(t *testing.T) {
	r := gemini.New()
	files := map[string]string{
		"rules/my-rule.md": "content",
	}
	out := r.Render(files)
	require.NotNil(t, out)
	assert.Equal(t, files, out.Files)
}

// ─── Rule tests ───────────────────────────────────────────────────────────────

func TestCompile_Rule_OutputPathIsMarkdown(t *testing.T) {
	r := gemini.New()
	config := &ast.XcaffoldConfig{
		Rules: map[string]ast.RuleConfig{
			"my-rule": {
				Description:  "A rule",
				Instructions: "Always format with gofmt.",
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	_, ok := out.Files["rules/my-rule.md"]
	assert.True(t, ok, "expected rules/my-rule.md in output")
}

func TestCompile_Rule_NoFrontmatterDelimiters(t *testing.T) {
	r := gemini.New()
	config := &ast.XcaffoldConfig{
		Rules: map[string]ast.RuleConfig{
			"plain-rule": {
				Instructions: "Be concise.",
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/plain-rule.md"]
	require.NotEmpty(t, content)

	// AG rules must NOT have YAML frontmatter delimiters
	assert.False(t, strings.HasPrefix(content, "---"), "AG rules must not start with --- frontmatter delimiter")
	assert.NotContains(t, content, "---", "AG rules must contain no --- delimiters")
}

func TestCompile_Rule_DescriptionAsHeading(t *testing.T) {
	r := gemini.New()
	config := &ast.XcaffoldConfig{
		Rules: map[string]ast.RuleConfig{
			"desc-rule": {
				Description:  "My Rule Description",
				Instructions: "Do something important.",
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/desc-rule.md"]
	require.NotEmpty(t, content)

	// Description becomes a markdown heading, not frontmatter
	assert.Contains(t, content, "# My Rule Description", "description must appear as # heading")
	assert.NotContains(t, content, "description:", "description must not appear as YAML frontmatter key")
}

func TestCompile_Rule_NoPathsOrGlobs(t *testing.T) {
	r := gemini.New()
	config := &ast.XcaffoldConfig{
		Rules: map[string]ast.RuleConfig{
			"path-rule": {
				Description:  "A rule with paths",
				Paths:        []string{"**/*.go"},
				Instructions: "Check Go files.",
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/path-rule.md"]
	require.NotEmpty(t, content)

	// AG handles activation via UI — no globs or paths in file
	assert.NotContains(t, content, "globs:", "globs must not appear in AG rules")
	assert.NotContains(t, content, "paths:", "paths must not appear in AG rules")
	assert.NotContains(t, content, "alwaysApply:", "alwaysApply must not appear in AG rules")
}

func TestCompile_Rule_BodyContentPreserved(t *testing.T) {
	r := gemini.New()
	body := "Always run tests before committing.\nUse table-driven tests."
	config := &ast.XcaffoldConfig{
		Rules: map[string]ast.RuleConfig{
			"test-rule": {
				Instructions: body,
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/test-rule.md"]
	assert.Contains(t, content, "Always run tests before committing.")
	assert.Contains(t, content, "Use table-driven tests.")
}

func TestCompile_Rule_DescriptionHeadingPrecedesBody(t *testing.T) {
	r := gemini.New()
	config := &ast.XcaffoldConfig{
		Rules: map[string]ast.RuleConfig{
			"order-rule": {
				Description:  "Title",
				Instructions: "Body content.",
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/order-rule.md"]
	headingPos := strings.Index(content, "# Title")
	bodyPos := strings.Index(content, "Body content.")
	assert.True(t, headingPos < bodyPos, "# heading must precede body content")
}

func TestCompile_Rule_NoDescription_NoHeading(t *testing.T) {
	r := gemini.New()
	config := &ast.XcaffoldConfig{
		Rules: map[string]ast.RuleConfig{
			"no-desc-rule": {
				Instructions: "Just the body.",
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/no-desc-rule.md"]
	assert.False(t, strings.HasPrefix(content, "#"), "no heading when description is absent")
}

func TestCompile_Rule_12KCharacterLimitWarning(t *testing.T) {
	r := gemini.New()
	// Build a body longer than 12,000 characters
	longBody := strings.Repeat("a", 12001)
	config := &ast.XcaffoldConfig{
		Rules: map[string]ast.RuleConfig{
			"long-rule": {
				Instructions: longBody,
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/long-rule.md"]
	// A warning comment must be prepended
	assert.True(t, strings.HasPrefix(content, "<!--"), "long rule must begin with warning HTML comment")
	assert.Contains(t, content, "12000", "warning must mention the 12000-char limit")
}

func TestCompile_Rule_Under12K_NoWarning(t *testing.T) {
	r := gemini.New()
	body := strings.Repeat("b", 11999)
	config := &ast.XcaffoldConfig{
		Rules: map[string]ast.RuleConfig{
			"short-rule": {
				Instructions: body,
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/short-rule.md"]
	assert.False(t, strings.HasPrefix(content, "<!--"), "rules under 12K must not have warning comment")
}

func TestCompile_Rule_EmptyID_ReturnsError(t *testing.T) {
	r := gemini.New()
	config := &ast.XcaffoldConfig{
		Rules: map[string]ast.RuleConfig{
			"   ": {
				Instructions: "Bad rule.",
			},
		},
	}

	_, err := r.Compile(config, "")
	assert.Error(t, err)
}

// ─── Skill tests ──────────────────────────────────────────────────────────────

func TestCompile_Skill_OutputAtCorrectPath(t *testing.T) {
	r := gemini.New()
	config := &ast.XcaffoldConfig{
		Skills: map[string]ast.SkillConfig{
			"my-skill": {
				Name:         "My Skill",
				Description:  "A test skill",
				Instructions: "Do the skill thing.",
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	_, ok := out.Files["skills/my-skill/SKILL.md"]
	assert.True(t, ok, "expected skills/my-skill/SKILL.md in output")
}

func TestCompile_Skill_FrontmatterHasNameAndDescription(t *testing.T) {
	r := gemini.New()
	config := &ast.XcaffoldConfig{
		Skills: map[string]ast.SkillConfig{
			"fmt-skill": {
				Name:         "Format Skill",
				Description:  "Formats code",
				Instructions: "Run gofmt first.",
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["skills/fmt-skill/SKILL.md"]
	require.NotEmpty(t, content)

	assert.Contains(t, content, "name: Format Skill")
	assert.Contains(t, content, "description: Formats code")
}

func TestCompile_Skill_FrontmatterDelimitersPresent(t *testing.T) {
	r := gemini.New()
	config := &ast.XcaffoldConfig{
		Skills: map[string]ast.SkillConfig{
			"delim-skill": {
				Name:         "Delim Skill",
				Instructions: "Some body.",
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["skills/delim-skill/SKILL.md"]
	assert.True(t, strings.HasPrefix(content, "---\n"), "skill must start with frontmatter delimiter")
	assert.Contains(t, content, "\n---\n", "skill must have closing frontmatter delimiter")
}

func TestCompile_Skill_CCOnlyFieldsDropped(t *testing.T) {
	r := gemini.New()
	userInvocable := true
	disabled := true
	config := &ast.XcaffoldConfig{
		Skills: map[string]ast.SkillConfig{
			"rich-skill": {
				Name:                   "Rich Skill",
				Description:            "Has many fields.",
				Instructions:           "Do something.",
				DisableModelInvocation: &disabled,
				Tools:                  []string{"Bash"},
				AllowedTools:           []string{"Read"},
				Context:                "fork",
				Agent:                  "my-agent",
				Model:                  "claude-opus-4-5",
				Effort:                 "high",
				Shell:                  "bash",
				ArgumentHint:           "hint text",
				UserInvocable:          &userInvocable,
				Paths:                  []string{"**/*.go"},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["skills/rich-skill/SKILL.md"]
	require.NotEmpty(t, content)

	// CC-only and target-specific fields must NOT appear
	for _, dropped := range []string{
		"tools:", "allowed-tools:", "context:", "agent:", "model:", "effort:",
		"shell:", "argument-hint:", "user-invocable:", "paths:",
		"disable-model-invocation:",
	} {
		assert.NotContains(t, content, dropped, "field %q must be dropped for AG skills", dropped)
	}

	// Only name and description allowed in frontmatter
	assert.Contains(t, content, "name: Rich Skill")
	assert.Contains(t, content, "description: Has many fields.")
}

func TestCompile_Skill_BodyContentPreserved(t *testing.T) {
	r := gemini.New()
	config := &ast.XcaffoldConfig{
		Skills: map[string]ast.SkillConfig{
			"body-skill": {
				Name:         "Body Skill",
				Instructions: "Step 1: do this.\nStep 2: do that.",
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["skills/body-skill/SKILL.md"]
	assert.Contains(t, content, "Step 1: do this.")
	assert.Contains(t, content, "Step 2: do that.")
}

func TestCompile_Skill_EmptyID_ReturnsError(t *testing.T) {
	r := gemini.New()
	config := &ast.XcaffoldConfig{
		Skills: map[string]ast.SkillConfig{
			"   ": {
				Instructions: "Bad skill.",
			},
		},
	}

	_, err := r.Compile(config, "")
	assert.Error(t, err)
}

// ─── Silent-skip tests ────────────────────────────────────────────────────────

func TestCompile_AgentsHooksAndMCP_AreNotEmitted(t *testing.T) {
	r := gemini.New()
	config := &ast.XcaffoldConfig{
		Agents: map[string]ast.AgentConfig{
			"my-agent": {
				Name:         "My Agent",
				Instructions: "Do work.",
			},
		},
		Hooks: ast.HookConfig{
			"PreToolUse": {
				{
					Matcher: "Bash",
					Hooks: []ast.HookHandler{
						{Type: "command", Command: "echo hi"},
					},
				},
			},
		},
		MCP: map[string]ast.MCPConfig{
			"my-server": {
				Type:    "stdio",
				Command: "npx",
			},
		},
		Rules: map[string]ast.RuleConfig{
			"a-rule": {
				Instructions: "A rule body.",
			},
		},
		Skills: map[string]ast.SkillConfig{
			"a-skill": {
				Name:         "A Skill",
				Instructions: "A skill body.",
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	// Rules and skills must be emitted
	assert.Contains(t, out.Files, "rules/a-rule.md", "rules must be compiled")
	assert.Contains(t, out.Files, "skills/a-skill/SKILL.md", "skills must be compiled")

	// Agents, hooks, MCP must be silently skipped
	for path := range out.Files {
		assert.False(t, strings.HasPrefix(path, "agents/"), "agents must not be emitted for AG target")
		assert.NotEqual(t, "mcp.json", path, "mcp.json must not be emitted for AG target")
		assert.NotEqual(t, "hooks.json", path, "hooks must not be emitted for AG target")
	}
}

func TestCompile_EmptyConfig_ReturnsEmptyOutput(t *testing.T) {
	r := gemini.New()
	out, err := r.Compile(&ast.XcaffoldConfig{}, "")
	require.NoError(t, err)
	assert.Empty(t, out.Files)
}
