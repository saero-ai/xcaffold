package antigravity_test

import (
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/renderer/antigravity"
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

func TestRenderer_Render_Identity(t *testing.T) {
	r := antigravity.New()
	files := map[string]string{
		"rules/my-rule.md": "content",
	}
	out := r.Render(files)
	require.NotNil(t, out)
	assert.Equal(t, files, out.Files)
}

// ─── Rule tests ───────────────────────────────────────────────────────────────

func TestCompile_Rule_OutputPathIsMarkdown(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"my-rule": {
					Description:  "A rule",
					Instructions: "Always format with gofmt.",
				},
			},
		},
	}

	out, _, err := r.Compile(config, "")
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
					Instructions: "Be concise.",
				},
			},
		},
	}

	out, _, err := r.Compile(config, "")
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
					Description:  "My Rule Description",
					Instructions: "Do something important.",
				},
			},
		},
	}

	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/desc-rule.md"]
	require.NotEmpty(t, content)

	// Description becomes a markdown heading, not frontmatter
	assert.Contains(t, content, "# My Rule Description", "description must appear as # heading")
	assert.NotContains(t, content, "description:", "description must not appear as YAML frontmatter key")
}

func TestCompile_Rule_NoPathsOrGlobs(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"path-rule": {
					Description:  "A rule with paths",
					Paths:        []string{"**/*.go"},
					Instructions: "Check Go files.",
				},
			},
		},
	}

	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/path-rule.md"]
	require.NotEmpty(t, content)

	// AG handles activation via UI — no globs or paths in file
	assert.NotContains(t, content, "globs:", "globs must not appear in AG rules")
	assert.NotContains(t, content, "paths:", "paths must not appear in AG rules")
	assert.NotContains(t, content, "alwaysApply:", "alwaysApply must not appear in AG rules")
}

func TestCompile_Rule_BodyContentPreserved(t *testing.T) {
	r := antigravity.New()
	body := "Always run tests before committing.\nUse table-driven tests."
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"test-rule": {
					Instructions: body,
				},
			},
		},
	}

	out, _, err := r.Compile(config, "")
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
					Description:  "Title",
					Instructions: "Body content.",
				},
			},
		},
	}

	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/order-rule.md"]
	headingPos := strings.Index(content, "# Title")
	bodyPos := strings.Index(content, "Body content.")
	assert.True(t, headingPos < bodyPos, "# heading must precede body content")
}

func TestCompile_Rule_NoDescription_NoHeading(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"no-desc-rule": {
					Instructions: "Just the body.",
				},
			},
		},
	}

	out, _, err := r.Compile(config, "")
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
					Instructions: longBody,
				},
			},
		},
	}

	out, _, err := r.Compile(config, "")
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
					Instructions: body,
				},
			},
		},
	}

	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/short-rule.md"]
	assert.False(t, strings.HasPrefix(content, "<!--"), "rules under 12K must not have warning comment")
}

func TestCompile_Rule_EmptyID_ReturnsError(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"   ": {
					Instructions: "Bad rule.",
				},
			},
		},
	}

	_, _, err := r.Compile(config, "")
	assert.Error(t, err)
}

// ─── Skill tests ──────────────────────────────────────────────────────────────

func TestCompile_Skill_OutputAtCorrectPath(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"my-skill": {
					Name:         "My Skill",
					Description:  "A test skill",
					Instructions: "Do the skill thing.",
				},
			},
		},
	}

	out, _, err := r.Compile(config, "")
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
					Name:         "Format Skill",
					Description:  "Formats code",
					Instructions: "Run gofmt first.",
				},
			},
		},
	}

	out, _, err := r.Compile(config, "")
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
					Name:         "Delim Skill",
					Instructions: "Some body.",
				},
			},
		},
	}

	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["skills/delim-skill/SKILL.md"]
	assert.True(t, strings.HasPrefix(content, "---\n"), "skill must start with frontmatter delimiter")
	assert.Contains(t, content, "\n---\n", "skill must have closing frontmatter delimiter")
}

func TestCompile_Skill_CCOnlyFieldsDropped(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"rich-skill": {
					Name:         "Rich Skill",
					Description:  "Has many fields.",
					Instructions: "Do something.",
					AllowedTools: []string{"Bash"},
					References:   []string{"**/*.go"},
					Scripts:      []string{"setup.sh"},
					Assets:       []string{"icon.png"},
				},
			},
		},
	}

	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["skills/rich-skill/SKILL.md"]
	require.NotEmpty(t, content)

	// CC-only and target-specific fields must NOT appear
	for _, dropped := range []string{
		"tools:", "references:", "scripts:", "assets:",
	} {
		assert.NotContains(t, content, dropped, "field %q must be dropped for AG skills", dropped)
	}

	// Only name and description allowed in frontmatter
	assert.Contains(t, content, "name: Rich Skill")
	assert.Contains(t, content, "description: Has many fields.")
}

func TestCompile_Skill_BodyContentPreserved(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"body-skill": {
					Name:         "Body Skill",
					Instructions: "Step 1: do this.\nStep 2: do that.",
				},
			},
		},
	}

	out, _, err := r.Compile(config, "")
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
					Instructions: "Bad skill.",
				},
			},
		},
	}

	_, _, err := r.Compile(config, "")
	assert.Error(t, err)
}

// ─── Silent-skip tests ────────────────────────────────────────────────────────

func TestCompile_AgentsAndHooks_AreNotEmitted(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
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
		},
	}

	out, _, err := r.Compile(config, "")
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

func TestCompile_MCP_EmitsConfig(t *testing.T) {
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

	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	content, ok := out.Files["mcp_config.json"]
	assert.True(t, ok, "expected mcp_config.json in output")
	assert.Contains(t, content, `"command": "npx"`)
	assert.Contains(t, content, `"my-server"`)
	assert.Contains(t, content, `"-y"`)
}

func TestCompile_EmptyConfig_ReturnsEmptyOutput(t *testing.T) {
	r := antigravity.New()
	out, _, err := r.Compile(&ast.XcaffoldConfig{}, "")
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
					Name:         "Commit Changes",
					Description:  "How to commit changes",
					Instructions: "1. Stage files\n2. Commit",
				},
			},
		},
	}

	out, _, err := r.Compile(config, "")
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
					Description:  "Deploy to production",
					Instructions: "Run deploy script.",
				},
			},
		},
	}

	out, _, err := r.Compile(config, "")
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
					Name:         "Build Project",
					Instructions: "Run make build.",
				},
			},
		},
	}

	out, _, err := r.Compile(config, "")
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
					Description:  "Run tests",
					Instructions: "1. Run unit tests\n2. Run integration tests\n3. Check coverage",
				},
			},
		},
	}

	out, _, err := r.Compile(config, "")
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

	out, _, err := r.Compile(config, "")
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
					Instructions: "Bad workflow.",
				},
			},
		},
	}

	_, _, err := r.Compile(config, "")
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

	out, _, err := r.Compile(config, "")
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
		Settings: ast.SettingsConfig{
			Permissions: &ast.PermissionsConfig{Allow: []string{"Read"}},
		},
	}

	_, notes, err := r.Compile(config, "")
	require.NoError(t, err)

	note, ok := findAgNote(notes, renderer.CodeSettingsFieldUnsupported, "permissions")
	require.True(t, ok)
	assert.Equal(t, "antigravity", note.Target)
}

func TestAntigravityRenderer_SandboxSetting_EmitsNote(t *testing.T) {
	r := antigravity.New()
	enabled := true
	config := &ast.XcaffoldConfig{
		Settings: ast.SettingsConfig{
			Sandbox: &ast.SandboxConfig{Enabled: &enabled},
		},
	}

	_, notes, err := r.Compile(config, "")
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
					Instructions:    "Build things.",
					PermissionMode:  "plan",
					DisallowedTools: []string{"Write"},
					Isolation:       "container",
				},
			},
		},
	}

	_, notes, err := r.Compile(config, "")
	require.NoError(t, err)

	_, pm := findAgNote(notes, renderer.CodeAgentSecurityFieldsDropped, "permissionMode")
	_, dt := findAgNote(notes, renderer.CodeAgentSecurityFieldsDropped, "disallowedTools")
	_, iso := findAgNote(notes, renderer.CodeAgentSecurityFieldsDropped, "isolation")
	assert.True(t, pm, "permissionMode note must be emitted")
	assert.True(t, dt, "disallowedTools note must be emitted")
	assert.True(t, iso, "isolation note must be emitted")
}

// Suppression is enforced at the command layer; the renderer returns notes
// regardless of the suppress-fidelity-warnings override.
func TestAntigravityRenderer_SuppressFidelityWarnings_NotesStillReturned(t *testing.T) {
	r := antigravity.New()
	suppress := true
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {
					Instructions:   "Build things.",
					PermissionMode: "plan",
					Isolation:      "container",
					Targets: map[string]ast.TargetOverride{
						"antigravity": {SuppressFidelityWarnings: &suppress},
					},
				},
			},
		},
	}

	_, notes, err := r.Compile(config, "")
	require.NoError(t, err)
	assert.NotEmpty(t, notes, "renderer returns notes; suppression is applied at the command layer")
}

func TestAntigravityRenderer_MCPEnvInterpolation_EmitsNote(t *testing.T) {
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

	_, notes, err := r.Compile(config, "")
	require.NoError(t, err)

	_, ok := findAgNote(notes, renderer.CodeHookInterpolationRequiresEnvSyntax, "mcp.env")
	assert.True(t, ok)
}
