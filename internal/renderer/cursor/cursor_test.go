package cursor_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer/cursor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderer_Target(t *testing.T) {
	r := cursor.New()
	assert.Equal(t, "cursor", r.Target())
}

func TestRenderer_OutputDir(t *testing.T) {
	r := cursor.New()
	assert.Equal(t, ".cursor", r.OutputDir())
}

func TestRenderer_Render_Identity(t *testing.T) {
	r := cursor.New()
	files := map[string]string{
		"rules/foo.mdc": "content",
	}
	out := r.Render(files)
	require.NotNil(t, out)
	assert.Equal(t, files, out.Files)
}

func TestCompile_Rule_WithPaths_OutputExtensionIsMdc(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"my-rule": {
					Description:  "A rule with paths",
					Paths:        []string{"**/*.go"},
					Instructions: "Always format with gofmt.",
				},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	// Output path must use .mdc extension, not .md
	_, hasMdc := out.Files["rules/my-rule.mdc"]
	_, hasMd := out.Files["rules/my-rule.md"]
	assert.True(t, hasMdc, "expected rules/my-rule.mdc in output")
	assert.False(t, hasMd, "rules/my-rule.md must not appear in output")
}

func TestCompile_Rule_WithPaths_FrontmatterHasGlobs(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"go-fmt": {
					Description:  "Go formatting rule",
					Paths:        []string{"**/*.go", "**/*.mod"},
					Instructions: "Run gofmt.",
				},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/go-fmt.mdc"]
	require.NotEmpty(t, content)

	// Must contain globs: not paths:
	assert.Contains(t, content, "globs:", "frontmatter must use globs: key")
	assert.NotContains(t, content, "paths:", "frontmatter must not use paths: key")

	// Values must be preserved
	assert.Contains(t, content, "**/*.go")
	assert.Contains(t, content, "**/*.mod")
}

func TestCompile_Rule_WithoutPaths_HasAlwaysApply(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"global-rule": {
					Description:  "Always active rule",
					Instructions: "Be concise.",
				},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/global-rule.mdc"]
	require.NotEmpty(t, content)

	assert.Contains(t, content, "alwaysApply: true")
	assert.NotContains(t, content, "globs:")
	assert.NotContains(t, content, "paths:")
}

func TestCompile_Rule_BodyContentPreserved(t *testing.T) {
	r := cursor.New()
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

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/test-rule.mdc"]
	assert.Contains(t, content, "Always run tests before committing.")
	assert.Contains(t, content, "Use table-driven tests.")
}

func TestCompile_Rule_DescriptionInFrontmatter(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"desc-rule": {
					Description:  "My rule description",
					Instructions: "Do something.",
				},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/desc-rule.mdc"]
	require.NotEmpty(t, content)

	assert.Contains(t, content, "description: My rule description")
}

func TestCompile_Rule_FrontmatterDelimiters(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"delim-rule": {
					Instructions: "Body text.",
				},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/delim-rule.mdc"]
	assert.True(t, strings.HasPrefix(content, "---\n"), "file must start with frontmatter delimiter")
	// Must have closing ---
	assert.Contains(t, content, "\n---\n")
}

func TestCompile_EmptyConfig_ReturnsEmptyOutput(t *testing.T) {
	r := cursor.New()
	out, err := r.Compile(&ast.XcaffoldConfig{}, "")
	require.NoError(t, err)
	assert.Empty(t, out.Files)
}

func TestCompile_Rule_EmptyID_ReturnsError(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"   ": {
					Instructions: "Bad rule.",
				},
			},
		},
	}

	_, err := r.Compile(config, "")
	assert.Error(t, err)
}

// ─── Agent tests ─────────────────────────────────────────────────────────────

func TestCompile_Agent_OutputAtCorrectPath(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"my-agent": {
					Name:         "My Agent",
					Description:  "A test agent",
					Instructions: "Do something useful.",
				},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	_, ok := out.Files["agents/my-agent.md"]
	assert.True(t, ok, "expected agents/my-agent.md in output")
}

func TestCompile_Agent_BackgroundRenamedToIsBackground(t *testing.T) {
	r := cursor.New()
	bg := true
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"bg-agent": {
					Name:         "Background Agent",
					Instructions: "Run in background.",
					Background:   &bg,
				},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["agents/bg-agent.md"]
	require.NotEmpty(t, content)

	assert.Contains(t, content, "is_background: true", "background: true must become is_background: true")
	// Verify the original bare "background:" key is not present (is_background: is fine)
	assert.NotContains(t, content, "\nbackground:", "original background: key must be dropped")
}

func TestCompile_Agent_BackgroundFalse_NotEmitted(t *testing.T) {
	r := cursor.New()
	bg := false
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"normal-agent": {
					Name:         "Normal Agent",
					Instructions: "Run normally.",
					Background:   &bg,
				},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["agents/normal-agent.md"]
	assert.NotContains(t, content, "is_background:", "false background should not appear")
	assert.NotContains(t, content, "background:", "background key must never appear")
}

func TestCompile_Agent_CCOnlyFieldsDropped(t *testing.T) {
	r := cursor.New()
	bg := true
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"full-agent": {
					Name:            "Full Agent",
					Description:     "Has many fields.",
					Instructions:    "Do work.",
					Model:           "claude-opus-4-5",
					Effort:          "high",
					PermissionMode:  "acceptEdits",
					Isolation:       "worktree",
					Color:           "blue",
					Memory:          "project",
					MaxTurns:        10,
					Tools:           []string{"Bash", "Read"},
					DisallowedTools: []string{"Write"},
					Skills:          []string{"my-skill"},
					InitialPrompt:   "Hello!",
					Background:      &bg,
				},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["agents/full-agent.md"]
	require.NotEmpty(t, content)

	// CC-only fields must NOT appear as standalone frontmatter keys
	for _, dropped := range []string{"effort:", "permissionMode:", "isolation:", "color:", "memory:", "maxTurns:", "tools:", "disallowedTools:", "skills:", "initialPrompt:", "\nbackground:"} {
		assert.NotContains(t, content, dropped, "CC-only field %q must be dropped", dropped)
	}

	// Cursor-compatible fields MUST appear
	assert.Contains(t, content, "name: Full Agent")
	assert.Contains(t, content, "description: Has many fields.")
	assert.NotContains(t, content, "model:", "literal model claude-opus-4-5 is unmapped for cursor — must be omitted")
	assert.Contains(t, content, "is_background: true")
}

func TestCompile_Agent_UnmappedModel_Omitted(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"model-agent": {
					Name:         "Model Agent",
					Model:        "claude-sonnet-4-5",
					Instructions: "Use this model.",
				},
			},
		},
	}

	stderr, restore := captureStderr(t)
	out, err := r.Compile(config, "")
	restore()
	require.NoError(t, err)

	content := out.Files["agents/model-agent.md"]
	assert.NotContains(t, content, "model:", "unmapped literal model must be omitted from Cursor output")

	// Verify warning was emitted
	assert.Contains(t, stderr.String(), "WARNING (cursor):")
	assert.Contains(t, stderr.String(), "unmapped model")
	assert.Contains(t, stderr.String(), "model-agent")
}

func TestCompile_Agent_MappedAlias_EmittedWhenMapped(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"aliased-agent": {
					Name:         "Aliased Agent",
					Model:        "sonnet-4",
					Instructions: "Uses an alias.",
				},
			},
		},
	}

	stderr, restore := captureStderr(t)
	out, err := r.Compile(config, "")
	restore()
	require.NoError(t, err)

	content := out.Files["agents/aliased-agent.md"]
	// sonnet-4 has no cursor mapping, so ResolveModel falls through to literal.
	// IsMappedModel("sonnet-4", "cursor") is false, so the model is omitted.
	assert.NotContains(t, content, "model:", "sonnet-4 has no cursor mapping — must be omitted")

	// Warning should be emitted since there is no cursor mapping for this alias
	assert.Contains(t, stderr.String(), "WARNING (cursor):")
}

func TestCompile_Agent_UnmappedModel_WarningSuppressed(t *testing.T) {
	r := cursor.New()
	suppress := true
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"quiet-agent": {
					Name:         "Quiet Agent",
					Model:        "claude-sonnet-4-5",
					Instructions: "Silence warnings.",
					Targets: map[string]ast.TargetOverride{
						"cursor": {
							SuppressFidelityWarnings: &suppress,
						},
					},
				},
			},
		},
	}

	stderr, restore := captureStderr(t)
	out, err := r.Compile(config, "")
	restore()
	require.NoError(t, err)

	content := out.Files["agents/quiet-agent.md"]
	assert.NotContains(t, content, "model:", "unmapped literal model must be omitted regardless of suppress flag")

	// Warning must NOT appear when SuppressFidelityWarnings is true
	assert.NotContains(t, stderr.String(), "unmapped model", "warning must be suppressed")
}

func TestCompile_Agent_BodyContentPreserved(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"body-agent": {
					Name:         "Body Agent",
					Instructions: "Always write tests.\nNever skip lint.",
				},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["agents/body-agent.md"]
	assert.Contains(t, content, "Always write tests.")
	assert.Contains(t, content, "Never skip lint.")
}

// ─── Skill tests ──────────────────────────────────────────────────────────────

func TestCompile_Skill_OutputAtCorrectPath(t *testing.T) {
	r := cursor.New()
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

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	_, ok := out.Files["skills/my-skill/SKILL.md"]
	assert.True(t, ok, "expected skills/my-skill/SKILL.md in output")
}

func TestCompile_Skill_FrontmatterHasNameAndDescription(t *testing.T) {
	r := cursor.New()
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

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["skills/fmt-skill/SKILL.md"]
	require.NotEmpty(t, content)

	assert.Contains(t, content, "name: Format Skill")
	assert.Contains(t, content, "description: Formats code")
}

func TestCompile_Skill_CCOnlyFieldsDropped(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"rich-skill": {
					Name:         "Rich Skill",
					Description:  "Has many fields.",
					Instructions: "Do something.",
					Tools:        []string{"Bash"},
					References:   []string{"**/*.go"},
					Scripts:      []string{"setup.sh"},
					Assets:       []string{"icon.png"},
				},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["skills/rich-skill/SKILL.md"]
	require.NotEmpty(t, content)

	// CC-only fields must NOT appear
	for _, dropped := range []string{"tools:", "references:", "scripts:", "assets:"} {
		assert.NotContains(t, content, dropped, "CC-only field %q must be dropped", dropped)
	}

	// Cursor-compatible fields MUST appear
	assert.Contains(t, content, "name: Rich Skill")
	assert.Contains(t, content, "description: Has many fields.")
}

func TestCompile_Skill_BodyContentPreserved(t *testing.T) {
	r := cursor.New()
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

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["skills/body-skill/SKILL.md"]
	assert.Contains(t, content, "Step 1: do this.")
	assert.Contains(t, content, "Step 2: do that.")
}

// ─── MCP tests ────────────────────────────────────────────────────────────────

func TestCompile_MCP_EmitsMCPJson(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			MCP: map[string]ast.MCPConfig{
				"my-server": {
					Type:    "stdio",
					Command: "npx",
					Args:    []string{"-y", "@modelcontextprotocol/server-filesystem"},
				},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	_, ok := out.Files["mcp.json"]
	assert.True(t, ok, "expected mcp.json in output")
}

func TestCompile_MCP_HTTPTransport_URLBecomesServerURL(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			MCP: map[string]ast.MCPConfig{
				"remote-server": {
					Type:    "http",
					URL:     "https://mcp.example.com/endpoint",
					Headers: map[string]string{"Authorization": "Bearer tok"},
				},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	raw := out.Files["mcp.json"]
	require.NotEmpty(t, raw)

	var envelope map[string]map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(raw), &envelope))

	servers, ok := envelope["mcpServers"]
	require.True(t, ok, "mcp.json must have mcpServers key")

	entry, ok := servers["remote-server"].(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "https://mcp.example.com/endpoint", entry["serverUrl"], "url must be renamed to serverUrl")
	assert.Nil(t, entry["url"], "original url field must not appear")
	assert.Nil(t, entry["type"], "type field must be omitted — Cursor infers transport")
}

func TestCompile_MCP_StdioTransport_CommandArgsEnvPreserved(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			MCP: map[string]ast.MCPConfig{
				"stdio-server": {
					Type:    "stdio",
					Command: "python3",
					Args:    []string{"-m", "my_mcp_server"},
					Env:     map[string]string{"MY_VAR": "value"},
				},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	raw := out.Files["mcp.json"]
	var envelope map[string]map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(raw), &envelope))

	entry := envelope["mcpServers"]["stdio-server"].(map[string]interface{})

	assert.Equal(t, "python3", entry["command"])
	assert.Nil(t, entry["type"], "type field must be omitted")

	args, ok := entry["args"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, []interface{}{"-m", "my_mcp_server"}, args)

	env, ok := entry["env"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "value", env["MY_VAR"])
}

func TestCompile_MCP_TypeFieldNotInOutput(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			MCP: map[string]ast.MCPConfig{
				"typed-server": {
					Type: "sse",
					URL:  "https://sse.example.com",
				},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	raw := out.Files["mcp.json"]
	// The type field must not appear anywhere in the JSON
	assert.NotContains(t, raw, `"type"`, "type field must be omitted from mcp.json")
}

func TestCompile_MCP_MCPServersEnvelope(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			MCP: map[string]ast.MCPConfig{
				"server-a": {Command: "tool-a"},
				"server-b": {Command: "tool-b"},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	raw := out.Files["mcp.json"]
	assert.Contains(t, raw, `"mcpServers"`, "output must be wrapped in mcpServers envelope")

	var envelope map[string]map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(raw), &envelope))
	assert.Contains(t, envelope["mcpServers"], "server-a")
	assert.Contains(t, envelope["mcpServers"], "server-b")
}

func TestCompile_MCP_EmptyMCPMap_NoMCPJsonEmitted(t *testing.T) {
	r := cursor.New()
	out, err := r.Compile(&ast.XcaffoldConfig{}, "")
	require.NoError(t, err)
	_, ok := out.Files["mcp.json"]
	assert.False(t, ok, "mcp.json must not be emitted when no MCP servers are defined")
}

// ─── Hook tests (Gap 1-3) ────────────────────────────────────────────────────

func TestCompile_Hooks_ProducesHooksJson(t *testing.T) {
	r := cursor.New()
	timeout := 5000
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Hooks: ast.HookConfig{
				"PreToolUse": {
					{
						Matcher: "Bash",
						Hooks: []ast.HookHandler{
							{Type: "command", Command: "echo lint", Timeout: &timeout},
						},
					},
				},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	_, ok := out.Files["hooks.json"]
	assert.True(t, ok, "hooks.json must be emitted when Hooks are defined")
}

func TestCompile_Hooks_EmptyHooksNoOutput(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Hooks: ast.HookConfig{},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	_, ok := out.Files["hooks.json"]
	assert.False(t, ok, "hooks.json must not be emitted when Hooks map is empty")
}

func TestCompile_Hooks_EventNamesCamelCase(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Hooks: ast.HookConfig{
				"PreToolUse": {
					{
						Matcher: "Bash",
						Hooks: []ast.HookHandler{
							{Type: "command", Command: "echo pre"},
						},
					},
				},
				"PostToolUse": {
					{
						Matcher: "Write",
						Hooks: []ast.HookHandler{
							{Type: "command", Command: "echo post"},
						},
					},
				},
				"SessionStart": {
					{
						Hooks: []ast.HookHandler{
							{Type: "command", Command: "echo session"},
						},
					},
				},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	raw := out.Files["hooks.json"]
	require.NotEmpty(t, raw)

	// PascalCase → camelCase
	assert.Contains(t, raw, `"preToolUse"`, "PreToolUse must be converted to preToolUse")
	assert.Contains(t, raw, `"postToolUse"`, "PostToolUse must be converted to postToolUse")
	assert.Contains(t, raw, `"sessionStart"`, "SessionStart must be converted to sessionStart")

	// Original PascalCase must NOT appear as JSON keys
	assert.NotContains(t, raw, `"PreToolUse"`, "PascalCase key must not appear")
	assert.NotContains(t, raw, `"PostToolUse"`, "PascalCase key must not appear")
	assert.NotContains(t, raw, `"SessionStart"`, "PascalCase key must not appear")
}

func TestCompile_Hooks_FlatStructure_NoNestedHooksArray(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Hooks: ast.HookConfig{
				"PreToolUse": {
					{
						Matcher: "Bash",
						Hooks: []ast.HookHandler{
							{Type: "command", Command: "echo lint"},
							{Type: "command", Command: "echo test"},
						},
					},
				},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	raw := out.Files["hooks.json"]
	require.NotEmpty(t, raw)

	// Parse and verify flat structure
	var parsed map[string][]map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	handlers, ok := parsed["preToolUse"]
	require.True(t, ok)
	assert.Len(t, handlers, 2, "both handlers should be at top level, not nested")

	// Each handler should have command and matcher inline, NOT a nested "hooks" array
	for _, h := range handlers {
		assert.NotContains(t, h, "hooks", "handler must not contain nested 'hooks' array")
		assert.Equal(t, "Bash", h["matcher"], "matcher must be injected inline")
		assert.Equal(t, "command", h["type"])
	}
}

func TestCompile_Hooks_MatcherInjectedInline(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Hooks: ast.HookConfig{
				"PostToolUse": {
					{
						Matcher: "Edit",
						Hooks: []ast.HookHandler{
							{Type: "command", Command: "echo edited"},
						},
					},
				},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	var parsed map[string][]map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out.Files["hooks.json"]), &parsed))

	handlers := parsed["postToolUse"]
	require.Len(t, handlers, 1)
	assert.Equal(t, "Edit", handlers[0]["matcher"])
}

func TestCompile_Hooks_EmptyMatcher_NotInjected(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Hooks: ast.HookConfig{
				"SessionStart": {
					{
						// No matcher — should not appear on handler
						Hooks: []ast.HookHandler{
							{Type: "command", Command: "echo hi"},
						},
					},
				},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	var parsed map[string][]map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out.Files["hooks.json"]), &parsed))

	handlers := parsed["sessionStart"]
	require.Len(t, handlers, 1)
	_, hasMatcher := handlers[0]["matcher"]
	assert.False(t, hasMatcher, "empty matcher must not be injected as key")
}

// ─── AlwaysApply edge cases (Issue #4) ───────────────────────────────────────

func TestCompile_Rule_AlwaysApplyExplicitFalse_EmitsFalse(t *testing.T) {
	r := cursor.New()
	f := false
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"opt-out-rule": {
					Description:  "Explicit opt-out",
					Instructions: "Not always.",
					AlwaysApply:  &f,
				},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/opt-out-rule.mdc"]
	assert.Contains(t, content, "alwaysApply: false", "explicit false must be emitted")
}

func TestCompile_Rule_PathsWithAlwaysApplyTrue_BothEmitted(t *testing.T) {
	r := cursor.New()
	t2 := true
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"combo-rule": {
					Description:  "Has both",
					Paths:        []string{"**/*.ts"},
					Instructions: "Lint TypeScript.",
					AlwaysApply:  &t2,
				},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/combo-rule.mdc"]
	assert.Contains(t, content, "globs:", "paths must be emitted as globs")
	assert.Contains(t, content, "alwaysApply: true", "alwaysApply must also be emitted")
}

// ─── Readonly tests (Issue #5) ───────────────────────────────────────────────

func TestCompile_Agent_Readonly_EmitsReadonlyTrue(t *testing.T) {
	r := cursor.New()
	ro := true
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"ro-agent": {
					Name:         "Readonly Agent",
					Instructions: "Only read files.",
					Readonly:     &ro,
				},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["agents/ro-agent.md"]
	assert.Contains(t, content, "readonly: true", "readonly: true must be emitted for Cursor")
}

func TestCompile_Agent_ReadonlyFalse_NotEmitted(t *testing.T) {
	r := cursor.New()
	ro := false
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"rw-agent": {
					Name:         "ReadWrite Agent",
					Instructions: "Full access.",
					Readonly:     &ro,
				},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["agents/rw-agent.md"]
	assert.NotContains(t, content, "readonly:", "readonly false must not be emitted")
}

func TestCompile_Agent_WithDoubleQuotes_ProperlyEscapes(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"quoted": {
					Name:        `Agent with "quotes"`,
					Description: `A "very specal" agent`,
				},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["agents/quoted.md"]
	assert.Contains(t, content, `name: "Agent with \"quotes\""`)
	assert.Contains(t, content, `description: "A \"very specal\" agent"`)
	assert.NotContains(t, content, `\\\"`, "Must not double-escape quotes")
}

// ─── Security fidelity warning tests ─────────────────────────────────────────

func TestCursorRenderer_PermissionsFidelityWarning_Settings(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		Settings: ast.SettingsConfig{
			Permissions: &ast.PermissionsConfig{
				Allow: []string{"Read"},
			},
		},
	}

	stderr, restore := captureStderr(t)
	_, err := r.Compile(config, "")
	restore()
	require.NoError(t, err)

	output := stderr.String()
	assert.Contains(t, output, "WARNING (cursor):")
	assert.Contains(t, output, "settings.permissions dropped")
}

func TestCursorRenderer_SandboxFidelityWarning_Settings(t *testing.T) {
	r := cursor.New()
	enabled := true
	config := &ast.XcaffoldConfig{
		Settings: ast.SettingsConfig{
			Sandbox: &ast.SandboxConfig{
				Enabled: &enabled,
			},
		},
	}

	stderr, restore := captureStderr(t)
	_, err := r.Compile(config, "")
	restore()
	require.NoError(t, err)

	output := stderr.String()
	assert.Contains(t, output, "WARNING (cursor):")
	assert.Contains(t, output, "settings.sandbox dropped")
}

func TestCursorRenderer_PermissionModeFidelityWarning_Agent(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {
					Instructions:   "Build things.",
					PermissionMode: "plan",
				},
			},
		},
	}

	stderr, restore := captureStderr(t)
	_, err := r.Compile(config, "")
	restore()
	require.NoError(t, err)

	output := stderr.String()
	assert.Contains(t, output, "WARNING (cursor):")
	assert.Contains(t, output, "permissionMode")
	assert.Contains(t, output, "dropped")
}

func TestCursorRenderer_DisallowedToolsFidelityWarning_Agent(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {
					Instructions:    "Build things.",
					DisallowedTools: []string{"Write"},
				},
			},
		},
	}

	stderr, restore := captureStderr(t)
	_, err := r.Compile(config, "")
	restore()
	require.NoError(t, err)

	output := stderr.String()
	assert.Contains(t, output, "WARNING (cursor):")
	assert.Contains(t, output, "disallowedTools dropped")
}

func TestCursorRenderer_IsolationFidelityWarning_Agent(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {
					Instructions: "Build things.",
					Isolation:    "container",
				},
			},
		},
	}

	stderr, restore := captureStderr(t)
	_, err := r.Compile(config, "")
	restore()
	require.NoError(t, err)

	output := stderr.String()
	assert.Contains(t, output, "WARNING (cursor):")
	assert.Contains(t, output, "isolation")
	assert.Contains(t, output, "dropped")
}

func TestCursorRenderer_SuppressFidelityWarnings_SkipsAgentWarnings(t *testing.T) {
	r := cursor.New()
	suppress := true
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {
					Instructions:   "Build things.",
					PermissionMode: "plan",
					Isolation:      "container",
					Targets: map[string]ast.TargetOverride{
						"cursor": {
							SuppressFidelityWarnings: &suppress,
						},
					},
				},
			},
		},
	}

	stderr, restore := captureStderr(t)
	_, err := r.Compile(config, "")
	restore()
	require.NoError(t, err)

	output := stderr.String()
	// Per-agent warnings must be suppressed
	assert.NotContains(t, output, "permissionMode")
	assert.NotContains(t, output, "isolation")
}

// captureStderr temporarily redirects os.Stderr to a buffer for testing.
// Returns the buffer and a restore function that the caller must defer.
func captureStderr(t *testing.T) (*strings.Builder, func()) {
	t.Helper()
	old := os.Stderr
	r2, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	var buf strings.Builder
	done := make(chan struct{})
	go func() {
		defer close(done)
		b := make([]byte, 4096)
		for {
			n, readErr := r2.Read(b)
			if n > 0 {
				buf.Write(b[:n])
			}
			if readErr != nil {
				return
			}
		}
	}()

	return &buf, func() {
		w.Close()
		<-done
		r2.Close()
		os.Stderr = old
	}
}
