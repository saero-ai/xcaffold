package gemini

import (
	"encoding/json"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompile_Gemini_Hooks_BasicEvent verifies that a hook on PreToolExecution
// maps to Gemini's BeforeTool event and is written to .gemini/settings.json.
func TestCompile_Gemini_Hooks_BasicEvent(t *testing.T) {
	timeout := 5000
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Hooks: ast.HookConfig{
				"PreToolExecution": []ast.HookMatcherGroup{
					{
						Matcher: "write_file|replace",
						Hooks: []ast.HookHandler{
							{
								Type:    "command",
								Command: "scripts/security-check.sh",
								Timeout: &timeout,
							},
						},
					},
				},
			},
		},
	}

	r := New()
	out, _, err := r.Compile(cfg, t.TempDir())
	require.NoError(t, err)

	raw, ok := out.Files[".gemini/settings.json"]
	require.True(t, ok, "expected .gemini/settings.json to be emitted")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	hooksSection, ok := parsed["hooks"].(map[string]any)
	require.True(t, ok, "expected hooks key in settings.json")

	_, hasBefore := hooksSection["BeforeTool"]
	assert.True(t, hasBefore, "PreToolExecution must map to BeforeTool in Gemini settings")
}

// TestCompile_Gemini_Hooks_PostToolExecution verifies that PostToolExecution
// maps to Gemini's AfterTool event.
func TestCompile_Gemini_Hooks_PostToolExecution(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Hooks: ast.HookConfig{
				"PostToolExecution": []ast.HookMatcherGroup{
					{
						Hooks: []ast.HookHandler{
							{Type: "command", Command: "scripts/post.sh"},
						},
					},
				},
			},
		},
	}

	r := New()
	out, _, err := r.Compile(cfg, t.TempDir())
	require.NoError(t, err)

	raw, ok := out.Files[".gemini/settings.json"]
	require.True(t, ok, "expected .gemini/settings.json to be emitted")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	hooksSection, ok := parsed["hooks"].(map[string]any)
	require.True(t, ok, "expected hooks key")
	_, hasAfter := hooksSection["AfterTool"]
	assert.True(t, hasAfter, "PostToolExecution must map to AfterTool in Gemini settings")
}

// TestCompile_Gemini_MCP_StdioServer verifies that an MCP server with command
// and args is written to mcpServers in .gemini/settings.json.
func TestCompile_Gemini_MCP_StdioServer(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			MCP: map[string]ast.MCPConfig{
				"my-server": {
					Command: "node",
					Args:    []string{"server.js"},
				},
			},
		},
	}

	r := New()
	out, _, err := r.Compile(cfg, t.TempDir())
	require.NoError(t, err)

	raw, ok := out.Files[".gemini/settings.json"]
	require.True(t, ok, "expected .gemini/settings.json")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	mcpSection, ok := parsed["mcpServers"].(map[string]any)
	require.True(t, ok, "expected mcpServers key")

	serverEntry, ok := mcpSection["my-server"].(map[string]any)
	require.True(t, ok, "expected my-server entry")

	assert.Equal(t, "node", serverEntry["command"])
	args, _ := serverEntry["args"].([]any)
	require.Len(t, args, 1)
	assert.Equal(t, "server.js", args[0])
}

// TestCompile_Gemini_MCP_WithEnv verifies that env vars on an MCP server are preserved.
func TestCompile_Gemini_MCP_WithEnv(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			MCP: map[string]ast.MCPConfig{
				"env-server": {
					Command: "python",
					Args:    []string{"-m", "myserver"},
					Env: map[string]string{
						"API_KEY": "secret",
						"MODE":    "prod",
					},
				},
			},
		},
	}

	r := New()
	out, _, err := r.Compile(cfg, t.TempDir())
	require.NoError(t, err)

	raw, ok := out.Files[".gemini/settings.json"]
	require.True(t, ok, "expected .gemini/settings.json")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	mcpSection := parsed["mcpServers"].(map[string]any)
	serverEntry := mcpSection["env-server"].(map[string]any)

	envMap, ok := serverEntry["env"].(map[string]any)
	require.True(t, ok, "expected env key in MCP server entry")
	assert.Equal(t, "secret", envMap["API_KEY"])
	assert.Equal(t, "prod", envMap["MODE"])
}

// TestCompile_Gemini_Settings_MergedOutput verifies that both hooks and MCP
// produce a single .gemini/settings.json containing both keys.
func TestCompile_Gemini_Settings_MergedOutput(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Hooks: ast.HookConfig{
				"PreToolExecution": []ast.HookMatcherGroup{
					{Hooks: []ast.HookHandler{{Type: "command", Command: "check.sh"}}},
				},
			},
			MCP: map[string]ast.MCPConfig{
				"my-mcp": {Command: "node", Args: []string{"index.js"}},
			},
		},
	}

	r := New()
	out, _, err := r.Compile(cfg, t.TempDir())
	require.NoError(t, err)

	raw, ok := out.Files[".gemini/settings.json"]
	require.True(t, ok, "expected single .gemini/settings.json")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	_, hasHooks := parsed["hooks"]
	_, hasMCP := parsed["mcpServers"]
	assert.True(t, hasHooks, "expected hooks key in merged settings.json")
	assert.True(t, hasMCP, "expected mcpServers key in merged settings.json")
}

// TestCompile_Gemini_Settings_EmptyHooksAndMCP verifies that no settings.json
// is emitted when there are no hooks and no MCP servers.
func TestCompile_Gemini_Settings_EmptyHooksAndMCP(t *testing.T) {
	cfg := &ast.XcaffoldConfig{}

	r := New()
	out, _, err := r.Compile(cfg, t.TempDir())
	require.NoError(t, err)

	_, ok := out.Files[".gemini/settings.json"]
	assert.False(t, ok, "settings.json must not be emitted when there are no hooks or MCP servers")
}

// TestCompile_Gemini_Hooks_UnsupportedEvent verifies that a hook on a
// Claude-specific event (e.g. SubagentStop, PreCompact) emits a
// CodeFieldUnsupported fidelity note and is NOT written to settings.json.
func TestCompile_Gemini_Hooks_UnsupportedEvent(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Hooks: ast.HookConfig{
				"SubagentStop": []ast.HookMatcherGroup{
					{
						Hooks: []ast.HookHandler{
							{Type: "command", Command: "scripts/cleanup.sh"},
						},
					},
				},
				"PreCompact": []ast.HookMatcherGroup{
					{
						Hooks: []ast.HookHandler{
							{Type: "command", Command: "scripts/pre-compact.sh"},
						},
					},
				},
				// A valid event that should still be emitted.
				"PreToolExecution": []ast.HookMatcherGroup{
					{
						Hooks: []ast.HookHandler{
							{Type: "command", Command: "scripts/check.sh"},
						},
					},
				},
			},
		},
	}

	r := New()
	out, notes, err := r.Compile(cfg, t.TempDir())
	require.NoError(t, err)

	// At least two CodeFieldUnsupported notes must be emitted (one per bad event).
	unsupportedFields := make(map[string]bool)
	for _, n := range notes {
		if n.Code == renderer.CodeFieldUnsupported {
			unsupportedFields[n.Field] = true
		}
	}
	assert.True(t, unsupportedFields["SubagentStop"], "expected CodeFieldUnsupported for SubagentStop")
	assert.True(t, unsupportedFields["PreCompact"], "expected CodeFieldUnsupported for PreCompact")

	// settings.json must be present (because PreToolExecution is valid).
	raw, ok := out.Files[".gemini/settings.json"]
	require.True(t, ok, "expected .gemini/settings.json to be emitted for the valid event")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	hooksSection, ok := parsed["hooks"].(map[string]any)
	require.True(t, ok, "expected hooks key in settings.json")

	// Valid event must be present.
	assert.Contains(t, hooksSection, "BeforeTool", "PreToolExecution must map to BeforeTool")

	// Unsupported events must NOT appear in output.
	assert.NotContains(t, hooksSection, "SubagentStop", "SubagentStop must not appear in settings.json")
	assert.NotContains(t, hooksSection, "PreCompact", "PreCompact must not appear in settings.json")
}

// TestCompile_Gemini_Settings_UnsupportedSettingsFields verifies that
// Claude-specific SettingsConfig fields emit CodeSettingsFieldUnsupported notes.
func TestCompile_Gemini_Settings_UnsupportedSettingsFields(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		Settings: ast.SettingsConfig{
			Permissions: &ast.PermissionsConfig{
				Allow: []string{"Bash(*)"},
			},
			Sandbox: &ast.SandboxConfig{},
			StatusLine: &ast.StatusLineConfig{
				Type:    "command",
				Command: "echo ok",
			},
		},
	}

	r := New()
	_, notes, err := r.Compile(cfg, t.TempDir())
	require.NoError(t, err)

	codeSet := make(map[string]bool)
	for _, n := range notes {
		if n.Code == renderer.CodeSettingsFieldUnsupported {
			codeSet[n.Field] = true
		}
	}

	assert.True(t, codeSet["permissions"], "expected fidelity note for permissions")
	assert.True(t, codeSet["sandbox"], "expected fidelity note for sandbox")
	assert.True(t, codeSet["statusLine"], "expected fidelity note for statusLine")
}
