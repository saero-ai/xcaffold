package compiler

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompile_AgentMinimalFields — agent with zero optional fields → frontmatter is just "---\n---\n"
func TestCompile_AgentMinimalFields(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"minimal": {},
			},
		},
	}
	out, _, err := Compile(config, "", "claude", "")
	require.NoError(t, err)

	content, ok := out.Files["agents/minimal.md"]
	require.True(t, ok, "agents/minimal.md should exist in output")
	assert.Equal(t, "---\n---\n", content, "minimal agent should produce only the frontmatter delimiters")
}

// TestCompile_AgentInstructionsTrailingNewlines — instructions ending with "\n\n\n" should be trimmed to one trailing newline
func TestCompile_AgentInstructionsTrailingNewlines(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"trimtest": {
					Body: "Do the thing.\n\n\n",
				},
			},
		},
	}
	out, _, err := Compile(config, "", "claude", "")
	require.NoError(t, err)

	content, ok := out.Files["agents/trimtest.md"]
	require.True(t, ok, "agents/trimtest.md should exist in output")
	// Instructions are TrimRight'd of "\n", then one "\n" appended.
	assert.True(t, strings.HasSuffix(content, "Do the thing.\n"),
		"trailing newlines should be trimmed to exactly one")
	assert.False(t, strings.HasSuffix(content, "Do the thing.\n\n"),
		"should not have more than one trailing newline after instructions")
}

// TestCompile_SkillWithAllFields — all skill fields present in output
func TestCompile_SkillWithAllFields(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"deploy": {
					Description: "Deploy the app",
					Body:        "Run the deploy script.",
				},
			},
		},
	}
	out, _, err := Compile(config, "", "claude", "")
	require.NoError(t, err)

	content, ok := out.Files["skills/deploy/SKILL.md"]
	require.True(t, ok, "skills/deploy/SKILL.md should exist in output")
	assert.Contains(t, content, "description: Deploy the app")
	assert.Contains(t, content, "Run the deploy script.")
}

// TestCompile_SkillEmptyID — whitespace-only ID should return an error
func TestCompile_SkillEmptyID(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"   ": {
					Description: "Blank ID skill",
				},
			},
		},
	}
	_, _, err := Compile(config, "", "claude", "")
	require.Error(t, err, "whitespace-only skill ID should return an error")
	assert.Contains(t, err.Error(), "skill")
}

// TestCompile_RuleEmptyID — empty string ID should return an error
func TestCompile_RuleEmptyID(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"": {
					Body: "Some rule",
				},
			},
		},
	}
	_, _, err := Compile(config, "", "claude", "")
	require.Error(t, err, "empty rule ID should return an error")
	assert.Contains(t, err.Error(), "rule")
}

// TestCompile_RuleWithPaths — paths appear in rule frontmatter
func TestCompile_RuleWithPaths(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"formatting": {
					Paths: ast.ClearableList{Values: []string{"**/*.go", "**/*.ts"}},
					Body:  "Always use gofmt.",
				},
			},
		},
	}
	out, _, err := Compile(config, "", "claude", "")
	require.NoError(t, err)

	content, ok := out.Files["rules/formatting.md"]
	require.True(t, ok, "rules/formatting.md should exist in output")
	assert.Contains(t, content, "paths: [**/*.go, **/*.ts]")
	assert.Contains(t, content, "Always use gofmt.")
}

// TestCompile_HooksJSON — valid JSON output with 3-level nested structure
func TestCompile_HooksJSON(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Hooks: map[string]ast.NamedHookConfig{
			"default": {
				Name: "default",
				Events: ast.HookConfig{
					"PreToolUse": []ast.HookMatcherGroup{
						{
							Matcher: "Bash",
							Hooks: []ast.HookHandler{
								{Type: "command", Command: "make lint"},
							},
						},
					},
				},
			},
		},
	}
	out, _, err := Compile(config, "", "claude", "")
	require.NoError(t, err)

	settingsJSON, hasSettings := out.Files["settings.json"]
	require.True(t, hasSettings)
	assert.Contains(t, settingsJSON, `"hooks"`)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(settingsJSON), &parsed), "settings.json must be valid JSON")

	// Verify the {hooks: ...} envelope
	hooksAny, hasHooks := parsed["hooks"]
	assert.True(t, hasHooks, "hooks.json must contain the 'hooks' wrapper key")

	hooksMap, ok := hooksAny.(map[string]any)
	require.True(t, ok, "hooks value should be an object")

	_, hasEvent := hooksMap["PreToolUse"]
	assert.True(t, hasEvent, "hooks object should contain 'PreToolUse' event key")
}

// TestCompile_MCPSettingsJSON — valid JSON with "mcp" key, env map preserved
func TestCompile_MCPSettingsJSON(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			MCP: map[string]ast.MCPConfig{
				"my-server": {
					Command: "node",
					Args:    []string{"server.js"},
					Env: map[string]string{
						"API_KEY": "secret123",
					},
				},
			},
		},
	}
	out, _, err := Compile(config, "", "claude", "")
	require.NoError(t, err)

	raw, ok := out.Files["mcp.json"]
	require.True(t, ok, "mcp.json should exist in output")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed), "mcp.json must be valid JSON")

	mcpAny, hasMCP := parsed["mcpServers"]
	require.True(t, hasMCP, "mcp.json must contain a top-level 'mcpServers' key")

	mcpMap, ok := mcpAny.(map[string]any)
	require.True(t, ok, "'mcp' value should be an object")

	serverAny, hasServer := mcpMap["my-server"]
	require.True(t, hasServer, "mcp object should contain 'my-server'")

	serverMap, ok := serverAny.(map[string]any)
	require.True(t, ok, "'my-server' value should be an object")

	envAny, hasEnv := serverMap["env"]
	require.True(t, hasEnv, "'my-server' should contain 'env'")

	envMap, ok := envAny.(map[string]any)
	require.True(t, ok, "'env' should be an object")
	assert.Equal(t, "secret123", envMap["API_KEY"], "env value should be preserved")
}

// TestCompile_NoHooksOrMCP_NoJSON — hooks.json and settings.json absent when no hooks/mcp defined
func TestCompile_NoHooksOrMCP_NoJSON(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"agent1": {Description: "An agent"},
			},
		},
	}
	out, _, err := Compile(config, "", "claude", "")
	require.NoError(t, err)

	settingsJSON, hasSettings := out.Files["settings.json"]
	if hasSettings {
		assert.NotContains(t, settingsJSON, `"hooks"`, "settings.json should not contain hooks when no hooks are defined")
	}
	assert.False(t, hasSettings, "settings.json should not exist when no settings are defined")

	_, hasMCP := out.Files["mcp.json"]
	assert.False(t, hasMCP, "mcp.json should not exist when no MCP is defined")
}

// TestCompile_PathTraversalSkillID — "../evil" as skill ID → filepath.Clean prevents ".." in output key
func TestCompile_PathTraversalSkillID(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"../evil": {
					Description: "Malicious skill",
				},
			},
		},
	}
	out, _, err := Compile(config, "", "claude", "")
	// If compilation succeeds, the output key must not start with ".."
	if err == nil {
		for key := range out.Files {
			assert.False(t, strings.HasPrefix(key, ".."),
				"output path %q must not traverse above the output root", key)
		}
	}
	// Either an error or a clean path is acceptable — what is NOT acceptable is
	// a key that starts with "..", which would escape the output directory.
}

// TestCompile_UnicodeInstructions — Japanese + emoji in instructions preserved in output
func TestCompile_UnicodeInstructions(t *testing.T) {
	unicodeInstructions := "日本語のテスト 🎉 これはテストです。"
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"unicode-agent": {
					Body: unicodeInstructions,
				},
			},
		},
	}
	out, _, err := Compile(config, "", "claude", "")
	require.NoError(t, err)

	content, ok := out.Files["agents/unicode-agent.md"]
	require.True(t, ok, "agents/unicode-agent.md should exist in output")
	assert.Contains(t, content, unicodeInstructions,
		"unicode instructions must be preserved verbatim in output")
}

// TestCompile_EmptyConfig — zero agents/skills/rules/hooks/mcp → empty Files map
func TestCompile_EmptyConfig(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	out, _, err := Compile(config, "", "claude", "")
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Empty(t, out.Files, "empty config should produce an empty Files map")
}
