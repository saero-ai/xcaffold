package cursor_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/renderer/cursor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findNote returns the first note with the given code, or a zero value and false.
func findNote(notes []renderer.FidelityNote, code string) (renderer.FidelityNote, bool) {
	for _, n := range notes {
		if n.Code == code {
			return n, true
		}
	}
	return renderer.FidelityNote{}, false
}

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

	out, _, err := r.Compile(config, "")
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

	out, _, err := r.Compile(config, "")
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

	out, _, err := r.Compile(config, "")
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

	out, _, err := r.Compile(config, "")
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

	out, _, err := r.Compile(config, "")
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

	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/delim-rule.mdc"]
	assert.True(t, strings.HasPrefix(content, "---\n"), "file must start with frontmatter delimiter")
	// Must have closing ---
	assert.Contains(t, content, "\n---\n")
}

func TestCompile_EmptyConfig_ReturnsEmptyOutput(t *testing.T) {
	r := cursor.New()
	out, _, err := r.Compile(&ast.XcaffoldConfig{}, "")
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

	_, _, err := r.Compile(config, "")
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

	out, _, err := r.Compile(config, "")
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

	out, _, err := r.Compile(config, "")
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

	out, _, err := r.Compile(config, "")
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

	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["agents/full-agent.md"]
	require.NotEmpty(t, content)

	// CC-only fields must NOT appear as standalone frontmatter keys
	for _, dropped := range []string{"effort:", "permission-mode:", "isolation:", "color:", "memory:", "max-turns:", "tools:", "disallowed-tools:", "skills:", "initial-prompt:", "\nbackground:"} {
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

	out, notes, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["agents/model-agent.md"]
	assert.NotContains(t, content, "model:", "unmapped literal model must be omitted from Cursor output")

	note, ok := findNote(notes, renderer.CodeAgentModelUnmapped)
	require.True(t, ok, "AGENT_MODEL_UNMAPPED note must be emitted")
	assert.Equal(t, "model-agent", note.Resource)
	assert.Equal(t, renderer.LevelWarning, note.Level)
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

	out, notes, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["agents/aliased-agent.md"]
	assert.NotContains(t, content, "model:", "sonnet-4 has no cursor mapping — must be omitted")

	_, ok := findNote(notes, renderer.CodeAgentModelUnmapped)
	assert.True(t, ok, "unmapped alias must emit AGENT_MODEL_UNMAPPED")
}

// Suppression is now enforced at the command layer; renderers return notes
// unconditionally. The command-layer test covers the filter behaviour.
func TestCompile_Agent_UnmappedModel_NoteReturnedRegardlessOfSuppress(t *testing.T) {
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

	out, notes, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["agents/quiet-agent.md"]
	assert.NotContains(t, content, "model:", "unmapped literal model must be omitted regardless of suppress flag")

	_, ok := findNote(notes, renderer.CodeAgentModelUnmapped)
	assert.True(t, ok, "renderer returns the note regardless of suppression; suppression is applied at the command layer")
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

	out, _, err := r.Compile(config, "")
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

	out, _, err := r.Compile(config, "")
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

	out, _, err := r.Compile(config, "")
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

	out, _, err := r.Compile(config, "")
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

	out, _, err := r.Compile(config, "")
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

	out, _, err := r.Compile(config, "")
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

	out, _, err := r.Compile(config, "")
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

	out, _, err := r.Compile(config, "")
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

	out, _, err := r.Compile(config, "")
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
	out, _, err := r.Compile(&ast.XcaffoldConfig{}, "")
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

	out, _, err := r.Compile(config, "")
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

	out, _, err := r.Compile(config, "")
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

	out, _, err := r.Compile(config, "")
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

	out, _, err := r.Compile(config, "")
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

	out, _, err := r.Compile(config, "")
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

	out, _, err := r.Compile(config, "")
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

	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/opt-out-rule.mdc"]
	assert.Contains(t, content, "alwaysApply: false", "explicit false must be emitted")
}

func TestCompile_Rule_PathsWithAlwaysApplyTrue_AlwaysTakesPrecedence(t *testing.T) {
	// When AlwaysApply=true is set explicitly alongside Paths, ResolvedActivation
	// returns "always" (AlwaysApply has higher precedence than path presence).
	// The rule fires on every file — globs: is not emitted.
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

	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/combo-rule.mdc"]
	assert.Contains(t, content, "alwaysApply: true", "AlwaysApply=true takes precedence; rule fires on every file")
	assert.NotContains(t, content, "globs:", "AlwaysApply=true supersedes paths; globs must not be emitted")
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

	out, _, err := r.Compile(config, "")
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

	out, _, err := r.Compile(config, "")
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

	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["agents/quoted.md"]
	assert.Contains(t, content, `name: "Agent with \"quotes\""`)
	assert.Contains(t, content, `description: "A \"very specal\" agent"`)
	assert.NotContains(t, content, `\\\"`, "Must not double-escape quotes")
}

// ─── Fidelity note tests ──────────────────────────────────────────────────────

func TestCursorRenderer_PermissionsSetting_EmitsNote(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		Settings: ast.SettingsConfig{
			Permissions: &ast.PermissionsConfig{Allow: []string{"Read"}},
		},
	}

	_, notes, err := r.Compile(config, "")
	require.NoError(t, err)

	note, ok := findNote(notes, renderer.CodeSettingsFieldUnsupported)
	require.True(t, ok)
	assert.Equal(t, "permissions", note.Field)
	assert.Equal(t, renderer.LevelWarning, note.Level)
}

func TestCursorRenderer_SandboxSetting_EmitsNote(t *testing.T) {
	r := cursor.New()
	enabled := true
	config := &ast.XcaffoldConfig{
		Settings: ast.SettingsConfig{
			Sandbox: &ast.SandboxConfig{Enabled: &enabled},
		},
	}

	_, notes, err := r.Compile(config, "")
	require.NoError(t, err)

	var found bool
	for _, n := range notes {
		if n.Code == renderer.CodeSettingsFieldUnsupported && n.Field == "sandbox" {
			found = true
		}
	}
	assert.True(t, found, "SETTINGS_FIELD_UNSUPPORTED note for sandbox must be emitted")
}

func TestCursorRenderer_AgentPermissionMode_EmitsNote(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {Instructions: "Build things.", PermissionMode: "plan"},
			},
		},
	}

	_, notes, err := r.Compile(config, "")
	require.NoError(t, err)

	var found bool
	for _, n := range notes {
		if n.Code == renderer.CodeAgentSecurityFieldsDropped && n.Field == "permissionMode" {
			found = true
			assert.Equal(t, "dev", n.Resource)
		}
	}
	assert.True(t, found, "AGENT_SECURITY_FIELDS_DROPPED note for permissionMode must be emitted")
}

func TestCursorRenderer_AgentDisallowedTools_EmitsNote(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {Instructions: "Build things.", DisallowedTools: []string{"Write"}},
			},
		},
	}

	_, notes, err := r.Compile(config, "")
	require.NoError(t, err)

	var found bool
	for _, n := range notes {
		if n.Code == renderer.CodeAgentSecurityFieldsDropped && n.Field == "disallowedTools" {
			found = true
		}
	}
	assert.True(t, found)
}

func TestCursorRenderer_AgentIsolation_EmitsNote(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {Instructions: "Build things.", Isolation: "container"},
			},
		},
	}

	_, notes, err := r.Compile(config, "")
	require.NoError(t, err)

	var found bool
	for _, n := range notes {
		if n.Code == renderer.CodeAgentSecurityFieldsDropped && n.Field == "isolation" {
			found = true
		}
	}
	assert.True(t, found)
}

// Suppression lives at the command layer now; the renderer returns notes
// regardless of the suppress-fidelity-warnings override.
func TestCursorRenderer_SuppressFidelityWarnings_NotesStillReturned(t *testing.T) {
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
						"cursor": {SuppressFidelityWarnings: &suppress},
					},
				},
			},
		},
	}

	_, notes, err := r.Compile(config, "")
	require.NoError(t, err)
	assert.NotEmpty(t, notes, "renderer returns notes; suppression is filtered at the command layer")
}

func TestCursorRenderer_SkillScriptsDropped_EmitsNote(t *testing.T) {
	r := cursor.New()
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

	note, ok := findNote(notes, renderer.CodeSkillScriptsDropped)
	require.True(t, ok)
	assert.Equal(t, "setup", note.Resource)
	assert.Equal(t, renderer.LevelWarning, note.Level)
}

// ─── Activation mapping tests ─────────────────────────────────────────────────

func TestCompileCursorRule_Activation_Always(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"security": {
					Activation:   ast.RuleActivationAlways,
					Instructions: "Body.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/security.mdc"]
	require.Contains(t, content, "alwaysApply: true")
	require.NotContains(t, content, "globs:")
}

func TestCompileCursorRule_Activation_PathGlob(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"api-style": {
					Activation:   ast.RuleActivationPathGlob,
					Paths:        []string{"src/**"},
					Instructions: "Body.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/api-style.mdc"]
	require.Contains(t, content, "globs:")
	require.Contains(t, content, "src/**")
	require.NotContains(t, content, "alwaysApply:")
}

func TestCompileCursorRule_Activation_ManualMention(t *testing.T) {
	r := cursor.New()
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

	content := out.Files["rules/commit-style.mdc"]
	require.Contains(t, content, "alwaysApply: false")
	require.NotContains(t, content, "globs:")
}

func TestCompileCursorRule_Activation_ModelDecided_FidelityNote(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"arch-review": {
					Activation:   ast.RuleActivationModelDecided,
					Instructions: "Body.",
				},
			},
		},
	}
	out, notes, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/arch-review.mdc"]
	require.Contains(t, content, "alwaysApply: false")

	require.NotEmpty(t, notes)
	require.Equal(t, renderer.LevelWarning, notes[0].Level)
	require.Contains(t, notes[0].Reason, "model-decided")
}

func TestCompileCursorRule_Activation_ExplicitInvoke_FidelityNote(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"deploy-gate": {
					Activation:   ast.RuleActivationExplicitInvoke,
					Instructions: "Body.",
				},
			},
		},
	}
	out, notes, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/deploy-gate.mdc"]
	require.Contains(t, content, "alwaysApply: false")

	require.NotEmpty(t, notes)
	require.Equal(t, renderer.LevelWarning, notes[0].Level)
	require.Contains(t, notes[0].Reason, "explicit-invoke")
}

func TestCompileCursorRule_LegacyAlwaysApply_True(t *testing.T) {
	truthy := true
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"style": {
					AlwaysApply:  &truthy,
					Instructions: "Body.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/style.mdc"]
	require.Contains(t, content, "alwaysApply: true")
}

func TestCompileCursorRule_LegacyAlwaysApply_False(t *testing.T) {
	falsy := false
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"manual": {
					AlwaysApply:  &falsy,
					Instructions: "Body.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/manual.mdc"]
	require.Contains(t, content, "alwaysApply: false")
}
