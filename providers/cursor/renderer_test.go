package cursor_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/providers/cursor"
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

func TestCompile_Rule_WithPaths_OutputExtensionIsMdc(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"my-rule": {
					Description: "A rule with paths",
					Paths:       ast.ClearableList{Values: []string{"**/*.go"}},
					Body:        "Always format with gofmt.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
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
					Description: "Go formatting rule",
					Paths:       ast.ClearableList{Values: []string{"**/*.go", "**/*.mod"}},
					Body:        "Run gofmt.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
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
					Description: "Always active rule",
					Body:        "Be concise.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/global-rule.mdc"]
	require.NotEmpty(t, content)

	assert.Contains(t, content, "always-apply: true")
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
					Body: body,
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
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
					Description: "My rule description",
					Body:        "Do something.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
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
					Body: "Body text.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/delim-rule.mdc"]
	assert.True(t, strings.HasPrefix(content, "---\n"), "file must start with frontmatter delimiter")
	// Must have closing ---
	assert.Contains(t, content, "\n---\n")
}

func TestCompile_EmptyConfig_ReturnsEmptyOutput(t *testing.T) {
	r := cursor.New()
	out, _, err := renderer.Orchestrate(r, &ast.XcaffoldConfig{}, "")
	require.NoError(t, err)
	assert.Empty(t, out.Files)
}

func TestCompile_Rule_EmptyID_ReturnsError(t *testing.T) {
	r := cursor.New()
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

// ─── Agent tests ─────────────────────────────────────────────────────────────

func TestCompile_Agent_OutputAtCorrectPath(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"my-agent": {
					Name:        "My Agent",
					Description: "A test agent",
					Body:        "Do something useful.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
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
					Name:       "Background Agent",
					Body:       "Run in background.",
					Background: &bg,
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
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
					Name:       "Normal Agent",
					Body:       "Run normally.",
					Background: &bg,
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
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
					Body:            "Do work.",
					Model:           "claude-opus-4-5",
					Effort:          "high",
					PermissionMode:  "acceptEdits",
					Isolation:       "worktree",
					Color:           "blue",
					Memory:          ast.FlexStringSlice{"project"},
					MaxTurns:        intPtr(10),
					Tools:           ast.ClearableList{Values: []string{"Bash", "Read"}},
					DisallowedTools: ast.ClearableList{Values: []string{"Write"}},
					Skills:          ast.ClearableList{Values: []string{"my-skill"}},
					InitialPrompt:   "Hello!",
					Background:      &bg,
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
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
					Name:  "Model Agent",
					Model: "claude-sonnet-4-5",
					Body:  "Use this model.",
				},
			},
		},
	}

	out, notes, err := renderer.Orchestrate(r, config, "")
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
					Name:  "Aliased Agent",
					Model: "sonnet-4",
					Body:  "Uses an alias.",
				},
			},
		},
	}

	out, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["agents/aliased-agent.md"]
	assert.Contains(t, content, "model: claude-sonnet-4-5", "sonnet-4 maps to claude-sonnet-4-5 for cursor — must be emitted")

	_, ok := findNote(notes, renderer.CodeAgentModelUnmapped)
	assert.False(t, ok, "mapped alias must not emit AGENT_MODEL_UNMAPPED")
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
					Name:  "Quiet Agent",
					Model: "claude-sonnet-4-5",
					Body:  "Silence warnings.",
					Targets: map[string]ast.TargetOverride{
						"cursor": {
							SuppressFidelityWarnings: &suppress,
						},
					},
				},
			},
		},
	}

	out, notes, err := renderer.Orchestrate(r, config, "")
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
					Name: "Body Agent",
					Body: "Always write tests.\nNever skip lint.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
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
	r := cursor.New()
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

func TestCompile_Skill_CCOnlyFieldsDropped(t *testing.T) {
	r := cursor.New()

	// Provide real files so compileCursorSkillSubdir can read them.
	// Files live at xcaf/skills/<id>/ since paths are skill-dir-relative.
	tmpDir := t.TempDir()
	skillBase := filepath.Join(tmpDir, "xcaf", "skills", "rich-skill")
	require.NoError(t, os.MkdirAll(filepath.Join(skillBase, "references"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillBase, "references", "main.go"), []byte("package main"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(skillBase, "setup.sh"), []byte("#!/bin/bash"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(skillBase, "icon.png"), []byte("\x89PNG"), 0o644))

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"rich-skill": {
					Name:         "Rich Skill",
					Description:  "Has many fields.",
					Body:         "Do something.",
					AllowedTools: ast.ClearableList{Values: []string{"Bash"}},
					Artifacts:    []string{"references"},
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, tmpDir)
	require.NoError(t, err)

	content := out.Files["skills/rich-skill/SKILL.md"]
	require.NotEmpty(t, content)

	// CC-only fields must NOT appear in the SKILL.md frontmatter.
	for _, dropped := range []string{"tools:", "references:", "scripts:", "assets:"} {
		assert.NotContains(t, content, dropped, "CC-only field %q must be dropped from frontmatter", dropped)
	}

	// Cursor-compatible fields MUST appear.
	assert.Contains(t, content, "name: Rich Skill")
	assert.Contains(t, content, "description: Has many fields.")
}

func TestCompile_Skill_BodyContentPreserved(t *testing.T) {
	r := cursor.New()
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

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	_, ok := out.Files["mcp.json"]
	assert.True(t, ok, "expected mcp.json in output")
}

func TestCompile_MCP_HTTPTransport_URLPreserved(t *testing.T) {
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

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	raw := out.Files["mcp.json"]
	require.NotEmpty(t, raw)

	var envelope map[string]map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(raw), &envelope))

	servers, ok := envelope["mcpServers"]
	require.True(t, ok, "mcp.json must have mcpServers key")

	entry, ok := servers["remote-server"].(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "https://mcp.example.com/endpoint", entry["url"], "url field must be preserved as-is")
	assert.Nil(t, entry["serverUrl"], "serverUrl field must not appear for Cursor")
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

	out, _, err := renderer.Orchestrate(r, config, "")
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

	out, _, err := renderer.Orchestrate(r, config, "")
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

	out, _, err := renderer.Orchestrate(r, config, "")
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
	out, _, err := renderer.Orchestrate(r, &ast.XcaffoldConfig{}, "")
	require.NoError(t, err)
	_, ok := out.Files["mcp.json"]
	assert.False(t, ok, "mcp.json must not be emitted when no MCP servers are defined")
}

// ─── Hook tests (Gap 1-3) ────────────────────────────────────────────────────

func TestCompile_Hooks_ProducesHooksJson(t *testing.T) {
	r := cursor.New()
	timeout := 5000
	config := &ast.XcaffoldConfig{
		Hooks: map[string]ast.NamedHookConfig{
			"default": {
				Name: "default",
				Events: ast.HookConfig{
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
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	_, ok := out.Files["hooks.json"]
	assert.True(t, ok, "hooks.json must be emitted when Hooks are defined")
}

func TestCompile_Hooks_EmptyHooksNoOutput(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		Hooks: map[string]ast.NamedHookConfig{},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	_, ok := out.Files["hooks.json"]
	assert.False(t, ok, "hooks.json must not be emitted when Hooks map is empty")
}

func TestCompile_Hooks_EventNamesCamelCase(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		Hooks: map[string]ast.NamedHookConfig{
			"default": {
				Name: "default",
				Events: ast.HookConfig{
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
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
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
		Hooks: map[string]ast.NamedHookConfig{
			"default": {
				Name: "default",
				Events: ast.HookConfig{
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
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	raw := out.Files["hooks.json"]
	require.NotEmpty(t, raw)

	// Parse and verify flat structure
	var envelope map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(raw), &envelope))

	handlersRaw, ok := envelope["preToolUse"]
	require.True(t, ok)

	handlersBytes, _ := json.Marshal(handlersRaw)
	var handlers []map[string]interface{}
	json.Unmarshal(handlersBytes, &handlers)

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
		Hooks: map[string]ast.NamedHookConfig{
			"default": {
				Name: "default",
				Events: ast.HookConfig{
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
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	var envelope map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out.Files["hooks.json"]), &envelope))

	handlersBytes, _ := json.Marshal(envelope["postToolUse"])
	var handlers []map[string]interface{}
	json.Unmarshal(handlersBytes, &handlers)
	require.Len(t, handlers, 1)
	assert.Equal(t, "Edit", handlers[0]["matcher"])
}

func TestCompile_Hooks_EmptyMatcher_NotInjected(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		Hooks: map[string]ast.NamedHookConfig{
			"default": {
				Name: "default",
				Events: ast.HookConfig{
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
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	var envelope map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out.Files["hooks.json"]), &envelope))

	handlersBytes, _ := json.Marshal(envelope["sessionStart"])
	var handlers []map[string]interface{}
	json.Unmarshal(handlersBytes, &handlers)
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
					Description: "Explicit opt-out",
					Body:        "Not always.",
					AlwaysApply: &f,
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/opt-out-rule.mdc"]
	assert.Contains(t, content, "always-apply: false", "explicit false must be emitted")
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
					Description: "Has both",
					Paths:       ast.ClearableList{Values: []string{"**/*.ts"}},
					Body:        "Lint TypeScript.",
					AlwaysApply: &t2,
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/combo-rule.mdc"]
	assert.Contains(t, content, "always-apply: true", "AlwaysApply=true takes precedence; rule fires on every file")
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
					Name:     "Readonly Agent",
					Body:     "Only read files.",
					Readonly: &ro,
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
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
					Name:     "ReadWrite Agent",
					Body:     "Full access.",
					Readonly: &ro,
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
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

	out, _, err := renderer.Orchestrate(r, config, "")
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
		Settings: map[string]ast.SettingsConfig{"default": {
			Permissions: &ast.PermissionsConfig{Allow: []string{"Read"}},
		}},
	}

	_, notes, err := renderer.Orchestrate(r, config, "")
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
		Settings: map[string]ast.SettingsConfig{"default": {
			Sandbox: &ast.SandboxConfig{Enabled: &enabled},
		}},
	}

	_, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	var found bool
	for _, n := range notes {
		if n.Code == renderer.CodeSettingsFieldUnsupported && n.Field == "sandbox" {
			found = true
		}
	}
	assert.True(t, found, "SETTINGS_FIELD_UNSUPPORTED note for sandbox must be emitted")
}

// TestSecurityFieldCheck_Centralized verifies that security fields on cursor
// agents are handled by the orchestrator's two-layer fidelity check.
// Fields with Role:["rendering"] are silently skipped — no FIELD_UNSUPPORTED
// and no AGENT_SECURITY_FIELDS_DROPPED are emitted.
func TestSecurityFieldCheck_Centralized(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {Body: "Build things.", PermissionMode: "plan"},
			},
		},
	}

	_, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	// Must NOT emit the old renderer-inline code.
	for _, n := range notes {
		if n.Code == renderer.CodeAgentSecurityFieldsDropped && n.Field == "permission-mode" {
			t.Error("AGENT_SECURITY_FIELDS_DROPPED must not be emitted; security checks are centralized in the orchestrator")
		}
	}

	// Two-layer fidelity check: permission-mode has Role:["rendering"], so it
	// is silently skipped for cursor — no FIELD_UNSUPPORTED note is emitted.
	for _, n := range notes {
		if n.Code == renderer.CodeFieldUnsupported && n.Field == "permission-mode" {
			t.Errorf("permission-mode has an xcaf role; FIELD_UNSUPPORTED must not be emitted for cursor, got: %s", n.Reason)
		}
	}
}

func TestCursorRenderer_AgentPermissionMode_Silent(t *testing.T) {
	// permission-mode has Role:["rendering"] — silently skipped for cursor.
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {Body: "Build things.", PermissionMode: "plan"},
			},
		},
	}

	_, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	for _, n := range notes {
		if n.Code == renderer.CodeFieldUnsupported && n.Field == "permission-mode" {
			t.Errorf("permission-mode has an xcaf role; FIELD_UNSUPPORTED must not be emitted, got: %s", n.Reason)
		}
	}
}

func TestCursorRenderer_AgentDisallowedTools_Silent(t *testing.T) {
	// disallowed-tools has Role:["rendering"] — silently skipped for cursor.
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {Body: "Build things.", DisallowedTools: ast.ClearableList{Values: []string{"Write"}}},
			},
		},
	}

	_, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	for _, n := range notes {
		if n.Code == renderer.CodeFieldUnsupported && n.Field == "disallowed-tools" {
			t.Errorf("disallowed-tools has an xcaf role; FIELD_UNSUPPORTED must not be emitted, got: %s", n.Reason)
		}
	}
}

func TestCursorRenderer_AgentIsolation_Silent(t *testing.T) {
	// isolation has Role:["rendering"] — silently skipped for cursor.
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {Body: "Build things.", Isolation: "container"},
			},
		},
	}

	_, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	for _, n := range notes {
		if n.Code == renderer.CodeFieldUnsupported && n.Field == "isolation" {
			t.Errorf("isolation has an xcaf role; FIELD_UNSUPPORTED must not be emitted, got: %s", n.Reason)
		}
	}
}

// Suppression is handled by the orchestrator's CheckFieldSupport — when
// suppress-fidelity-warnings is set, FIELD_UNSUPPORTED notes are suppressed.
func TestCursorRenderer_SuppressFidelityWarnings_SecurityFieldsSuppressed(t *testing.T) {
	r := cursor.New()
	suppress := true
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {
					Body:           "Build things.",
					PermissionMode: "plan",
					Isolation:      "container",
					Targets: map[string]ast.TargetOverride{
						"cursor": {SuppressFidelityWarnings: &suppress},
					},
				},
			},
		},
	}

	_, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)
	for _, n := range notes {
		if n.Code == renderer.CodeFieldUnsupported &&
			(n.Field == "permission-mode" || n.Field == "isolation") {
			t.Errorf("security field note %q should be suppressed when suppress-fidelity-warnings is set", n.Field)
		}
	}
}

func TestCursorRenderer_SkillScriptsEmitted(t *testing.T) {
	r := cursor.New()

	tmpDir := t.TempDir()
	// Auto-discovery walks xcaf/skills/<id>/scripts/ — use canonical dir name.
	skillBase := filepath.Join(tmpDir, "xcaf", "skills", "setup")
	scriptPath := filepath.Join(skillBase, "scripts")
	require.NoError(t, os.MkdirAll(scriptPath, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(scriptPath, "install.sh"), []byte("#!/bin/bash\necho hello"), 0o644))

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"setup": {
					Description: "Env setup.",
					Artifacts:   []string{"scripts"},
				},
			},
		},
	}

	out, notes, err := renderer.Orchestrate(r, config, tmpDir)
	require.NoError(t, err)

	// Scripts should be emitted, not dropped.
	content, ok := out.Files["skills/setup/scripts/install.sh"]
	assert.True(t, ok, "script file should be emitted")
	assert.Contains(t, content, "echo hello")

	// No drop warning should exist.
	for _, n := range notes {
		assert.NotEqual(t, renderer.CodeSkillScriptsDropped, n.Code, "should not emit scripts-dropped warning")
	}
}

func TestCursorRenderer_SkillAssetsEmitted(t *testing.T) {
	r := cursor.New()

	tmpDir := t.TempDir()
	// Auto-discovery walks xcaf/skills/<id>/assets/ — use canonical dir name.
	skillBase := filepath.Join(tmpDir, "xcaf", "skills", "gen")
	assetPath := filepath.Join(skillBase, "assets")
	require.NoError(t, os.MkdirAll(assetPath, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(assetPath, "template.txt"), []byte("hello world"), 0o644))

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"gen": {
					Description: "Generator.",
					Artifacts:   []string{"assets"},
				},
			},
		},
	}

	out, notes, err := renderer.Orchestrate(r, config, tmpDir)
	require.NoError(t, err)

	content, ok := out.Files["skills/gen/assets/template.txt"]
	assert.True(t, ok, "asset file should be emitted")
	assert.Equal(t, "hello world", content)

	for _, n := range notes {
		assert.NotEqual(t, renderer.CodeSkillAssetsDropped, n.Code, "should not emit assets-dropped warning")
	}
}

func TestCursorRenderer_SkillReferencesEmitted(t *testing.T) {
	r := cursor.New()

	tmpDir := t.TempDir()
	// Auto-discovery walks xcaf/skills/<id>/references/ — use canonical dir name.
	skillBase := filepath.Join(tmpDir, "xcaf", "skills", "db-setup")
	refPath := filepath.Join(skillBase, "references")
	require.NoError(t, os.MkdirAll(refPath, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(refPath, "schema.sql"), []byte("CREATE TABLE t(id INT);"), 0o644))

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"db-setup": {
					Description: "DB setup.",
					Artifacts:   []string{"references"},
				},
			},
		},
	}

	out, notes, err := renderer.Orchestrate(r, config, tmpDir)
	require.NoError(t, err)

	content, ok := out.Files["skills/db-setup/references/schema.sql"]
	assert.True(t, ok, "reference file should be emitted")
	assert.Contains(t, content, "CREATE TABLE")

	_ = notes
}

func TestCursorRenderer_SkillSubdirPathTraversal(t *testing.T) {
	// With auto-discovery, path traversal vectors via field values are eliminated.
	// Artifact discovery is driven by skill.Artifacts and walks the canonical directory
	// on disk — path traversal is no longer a vector.
	// Verify that a skill with no Artifacts produces only SKILL.md and no error.
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"safe": {
					Description: "No artifacts declared.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, t.TempDir())
	require.NoError(t, err)
	assert.Contains(t, out.Files, "skills/safe/SKILL.md", "SKILL.md should always be emitted")
}

// ─── Activation mapping tests ─────────────────────────────────────────────────

func TestCompileCursorRule_Activation_Always(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"security": {
					Activation: ast.RuleActivationAlways,
					Body:       "Body.",
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/security.mdc"]
	require.Contains(t, content, "always-apply: true")
	require.NotContains(t, content, "globs:")
}

func TestCompileCursorRule_Activation_PathGlob(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"api-style": {
					Activation: ast.RuleActivationPathGlob,
					Paths:      ast.ClearableList{Values: []string{"src/**"}},
					Body:       "Body.",
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/api-style.mdc"]
	require.Contains(t, content, "globs:")
	require.Contains(t, content, "src/**")
	require.NotContains(t, content, "always-apply:")
}

func TestCompileCursorRule_Activation_ManualMention(t *testing.T) {
	r := cursor.New()
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

	content := out.Files["rules/commit-style.mdc"]
	require.Contains(t, content, "always-apply: false")
	require.NotContains(t, content, "globs:")
}

func TestCompileCursorRule_Activation_ModelDecided_FidelityNote(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"arch-review": {
					Activation: ast.RuleActivationModelDecided,
					Body:       "Body.",
				},
			},
		},
	}
	out, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/arch-review.mdc"]
	require.Contains(t, content, "always-apply: false")

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
					Activation: ast.RuleActivationExplicitInvoke,
					Body:       "Body.",
				},
			},
		},
	}
	out, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/deploy-gate.mdc"]
	require.Contains(t, content, "always-apply: false")

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
					AlwaysApply: &truthy,
					Body:        "Body.",
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/style.mdc"]
	require.Contains(t, content, "always-apply: true")
}

func TestCompileCursorRule_LegacyAlwaysApply_False(t *testing.T) {
	falsy := false
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"manual": {
					AlwaysApply: &falsy,
					Body:        "Body.",
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/manual.mdc"]
	require.Contains(t, content, "always-apply: false")
}

func TestCompile_SkillWithExamples_Cursor(t *testing.T) {
	tmpDir := t.TempDir()
	// Auto-discovery walks xcaf/skills/<id>/examples/ — use canonical dir name.
	skillBase := filepath.Join(tmpDir, "xcaf", "skills", "my-skill")
	require.NoError(t, os.MkdirAll(filepath.Join(skillBase, "examples"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillBase, "examples", "sample.md"), []byte("# Sample"), 0o644))

	skills := map[string]ast.SkillConfig{
		"my-skill": {
			Description: "test",
			Body:        "Do the thing.",
			Artifacts:   []string{"examples"},
		},
	}

	r := cursor.New()
	files, _, err := r.CompileSkills(skills, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Cursor: examples collapse into references/ (via SkillArtifactDirs mapping)
	if _, ok := files["skills/my-skill/references/sample.md"]; !ok {
		t.Errorf("expected examples collapsed into references/, got keys: %v", renderer.SortedKeys(files))
	}
}
