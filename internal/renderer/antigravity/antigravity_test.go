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
	assert.NotContains(t, content, "WARNING:", "rules under 12K must not have 12K warning comment")
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

	out, _, err := r.Compile(config, "")
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

	_, notes, err := r.Compile(config, "")
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
	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	_, ok := out.Files["mcp_config.json"]
	assert.False(t, ok, "mcp_config.json must NOT be written for HTTP-transport servers either")
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

	_, notes, err := r.Compile(config, "")
	require.NoError(t, err)

	_, ok := findAgNote(notes, renderer.CodeMCPGlobalConfigOnly, "mcp")
	assert.True(t, ok, "expected MCP_GLOBAL_CONFIG_ONLY note for MCP declarations on Antigravity target")
}

func TestAntigravityRenderer_SkillScripts_EmitsNote(t *testing.T) {
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
	_, notes, err := r.Compile(config, "")
	require.NoError(t, err)
	note, ok := findAgNote(notes, renderer.CodeSkillScriptsDropped, "scripts")
	require.True(t, ok)
	assert.Equal(t, "setup", note.Resource)
}

func TestAntigravityRenderer_SkillAssets_EmitsNote(t *testing.T) {
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
	_, notes, err := r.Compile(config, "")
	require.NoError(t, err)
	_, ok := findAgNote(notes, renderer.CodeSkillAssetsDropped, "assets")
	assert.True(t, ok)
}

// ─── Activation provenance comment tests ─────────────────────────────────────

func TestCompileAntigravityRule_Activation_AlwaysOn(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"security": {
					Description:  "Security checklist.",
					Activation:   ast.RuleActivationAlways,
					Instructions: "Follow OWASP.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/security.md"]
	require.Contains(t, content, "<!-- xcaffold:activation AlwaysOn -->")
}

func TestCompileAntigravityRule_Activation_Manual(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"commit-style": {
					Activation:   ast.RuleActivationManualMention,
					Instructions: "Body.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/commit-style.md"]
	require.Contains(t, content, "<!-- xcaffold:activation Manual -->")
}

func TestCompileAntigravityRule_Activation_Glob_WithPaths(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"api-style": {
					Activation:   ast.RuleActivationPathGlob,
					Paths:        []string{"src/**", "packages/api/**"},
					Instructions: "REST conventions.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/api-style.md"]
	require.Contains(t, content, "<!-- xcaffold:activation Glob -->")
	require.Contains(t, content, `<!-- xcaffold:paths ["src/**","packages/api/**"] -->`)
}

func TestCompileAntigravityRule_NoProvenance_ExistingBehaviorPreserved(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"legacy": {
					Description:  "Legacy rule.",
					Instructions: "No activation field set.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/legacy.md"]
	// No explicit activation → AlwaysOn (the default) is still emitted.
	require.Contains(t, content, "<!-- xcaffold:activation AlwaysOn -->")
	// Description must appear as heading.
	require.Contains(t, content, "# Legacy rule.")
}
