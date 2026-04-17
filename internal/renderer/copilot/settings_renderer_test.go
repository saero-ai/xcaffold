package copilot_test

import (
	"encoding/json"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/renderer/copilot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompile_Copilot_Hooks_BasicEvent verifies that a PreToolUse hook is written
// to .github/hooks/xcaffold-hooks.json with version 1 and the preToolUse event key.
func TestCompile_Copilot_Hooks_BasicEvent(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Hooks: ast.HookConfig{
				"PreToolUse": []ast.HookMatcherGroup{
					{
						Hooks: []ast.HookHandler{
							{Type: "command", Command: "scripts/pre-tool.sh"},
						},
					},
				},
			},
		},
	}

	r := copilot.New()
	out, _, err := r.Compile(cfg, t.TempDir())
	require.NoError(t, err)

	raw, ok := out.Files[".github/hooks/xcaffold-hooks.json"]
	require.True(t, ok, "expected .github/hooks/xcaffold-hooks.json to be emitted")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	version, ok := parsed["version"].(float64)
	require.True(t, ok, "expected version key in hooks JSON")
	assert.Equal(t, float64(1), version)

	hooksSection, ok := parsed["hooks"].(map[string]any)
	require.True(t, ok, "expected hooks key in hooks JSON")
	_, hasPreToolUse := hooksSection["preToolUse"]
	assert.True(t, hasPreToolUse, "PreToolUse must map to preToolUse in Copilot hooks JSON")
}

// TestCompile_Copilot_Hooks_MultipleEvents verifies that PreToolUse, PostToolUse,
// and SessionStart all appear as separate event keys in xcaffold-hooks.json.
func TestCompile_Copilot_Hooks_MultipleEvents(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Hooks: ast.HookConfig{
				"PreToolUse": []ast.HookMatcherGroup{
					{Hooks: []ast.HookHandler{{Type: "command", Command: "pre.sh"}}},
				},
				"PostToolUse": []ast.HookMatcherGroup{
					{Hooks: []ast.HookHandler{{Type: "command", Command: "post.sh"}}},
				},
				"SessionStart": []ast.HookMatcherGroup{
					{Hooks: []ast.HookHandler{{Type: "command", Command: "start.sh"}}},
				},
			},
		},
	}

	r := copilot.New()
	out, _, err := r.Compile(cfg, t.TempDir())
	require.NoError(t, err)

	raw, ok := out.Files[".github/hooks/xcaffold-hooks.json"]
	require.True(t, ok, "expected .github/hooks/xcaffold-hooks.json")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	hooksSection, ok := parsed["hooks"].(map[string]any)
	require.True(t, ok, "expected hooks key")
	assert.Contains(t, hooksSection, "preToolUse")
	assert.Contains(t, hooksSection, "postToolUse")
	assert.Contains(t, hooksSection, "sessionStart")
}

// TestCompile_Copilot_Hooks_UnsupportedEvent verifies that an unsupported event
// (Notification) emits a CodeFieldUnsupported note and does NOT appear in the JSON.
func TestCompile_Copilot_Hooks_UnsupportedEvent(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Hooks: ast.HookConfig{
				"Notification": []ast.HookMatcherGroup{
					{Hooks: []ast.HookHandler{{Type: "command", Command: "notify.sh"}}},
				},
				// A valid event that should still be emitted.
				"PreToolUse": []ast.HookMatcherGroup{
					{Hooks: []ast.HookHandler{{Type: "command", Command: "check.sh"}}},
				},
			},
		},
	}

	r := copilot.New()
	out, notes, err := r.Compile(cfg, t.TempDir())
	require.NoError(t, err)

	unsupportedNotes := filterNotes(notes, renderer.CodeFieldUnsupported)
	fieldSet := make(map[string]bool)
	for _, n := range unsupportedNotes {
		fieldSet[n.Field] = true
	}
	assert.True(t, fieldSet["Notification"], "expected CodeFieldUnsupported for Notification event")

	raw, ok := out.Files[".github/hooks/xcaffold-hooks.json"]
	require.True(t, ok, "expected .github/hooks/xcaffold-hooks.json to be emitted for valid event")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	hooksSection, ok := parsed["hooks"].(map[string]any)
	require.True(t, ok, "expected hooks key")
	assert.Contains(t, hooksSection, "preToolUse", "valid event must be present")
	assert.NotContains(t, hooksSection, "Notification", "unsupported event must not appear in output")
	assert.NotContains(t, hooksSection, "notification", "unsupported event must not appear in output")
}

// TestCompile_Copilot_Hooks_TimeoutConversion verifies that a timeout of 5000ms
// is converted to timeoutSec: 5 in the hook entry.
func TestCompile_Copilot_Hooks_TimeoutConversion(t *testing.T) {
	timeout := 5000
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Hooks: ast.HookConfig{
				"PreToolUse": []ast.HookMatcherGroup{
					{
						Hooks: []ast.HookHandler{
							{Type: "command", Command: "check.sh", Timeout: &timeout},
						},
					},
				},
			},
		},
	}

	r := copilot.New()
	out, _, err := r.Compile(cfg, t.TempDir())
	require.NoError(t, err)

	raw, ok := out.Files[".github/hooks/xcaffold-hooks.json"]
	require.True(t, ok, "expected .github/hooks/xcaffold-hooks.json")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	hooksSection := parsed["hooks"].(map[string]any)
	preToolUse := hooksSection["preToolUse"].([]any)
	require.NotEmpty(t, preToolUse)

	entry := preToolUse[0].(map[string]any)
	timeoutSec, ok := entry["timeoutSec"].(float64)
	require.True(t, ok, "expected timeoutSec in hook entry")
	assert.Equal(t, float64(5), timeoutSec, "5000ms must convert to 5 seconds")
}

// TestCompile_Copilot_MCP_StdioServer verifies that an MCP server with command
// and args is written to .vscode/mcp.json with the servers key.
func TestCompile_Copilot_MCP_StdioServer(t *testing.T) {
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

	r := copilot.New()
	out, _, err := r.Compile(cfg, t.TempDir())
	require.NoError(t, err)

	raw, ok := out.Files[".vscode/mcp.json"]
	require.True(t, ok, "expected .vscode/mcp.json to be emitted")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	serversSection, ok := parsed["servers"].(map[string]any)
	require.True(t, ok, "expected servers key in .vscode/mcp.json")

	serverEntry, ok := serversSection["my-server"].(map[string]any)
	require.True(t, ok, "expected my-server entry in servers")

	assert.Equal(t, "node", serverEntry["command"])
	args, _ := serverEntry["args"].([]any)
	require.Len(t, args, 1)
	assert.Equal(t, "server.js", args[0])
}

// TestCompile_Copilot_MCP_EnvVars verifies that env variables on an MCP server
// are passed through to .vscode/mcp.json.
func TestCompile_Copilot_MCP_EnvVars(t *testing.T) {
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

	r := copilot.New()
	out, _, err := r.Compile(cfg, t.TempDir())
	require.NoError(t, err)

	raw, ok := out.Files[".vscode/mcp.json"]
	require.True(t, ok, "expected .vscode/mcp.json")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	serversSection := parsed["servers"].(map[string]any)
	serverEntry := serversSection["env-server"].(map[string]any)

	envMap, ok := serverEntry["env"].(map[string]any)
	require.True(t, ok, "expected env key in MCP server entry")
	assert.Equal(t, "secret", envMap["API_KEY"])
	assert.Equal(t, "prod", envMap["MODE"])
}

// TestCompile_Copilot_MCP_GlobalConfigNote verifies that MCP compilation emits
// a CodeMCPGlobalConfigOnly info note about the CLI config path.
func TestCompile_Copilot_MCP_GlobalConfigNote(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			MCP: map[string]ast.MCPConfig{
				"my-server": {Command: "node", Args: []string{"index.js"}},
			},
		},
	}

	r := copilot.New()
	_, notes, err := r.Compile(cfg, t.TempDir())
	require.NoError(t, err)

	globalNotes := filterNotes(notes, renderer.CodeMCPGlobalConfigOnly)
	assert.NotEmpty(t, globalNotes, "expected CodeMCPGlobalConfigOnly note for MCP servers")
}

// TestCompile_Copilot_Settings_MergedOutput verifies that hooks and MCP compile
// to SEPARATE files (.github/hooks/xcaffold-hooks.json and .vscode/mcp.json),
// not merged into a single file.
func TestCompile_Copilot_Settings_MergedOutput(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Hooks: ast.HookConfig{
				"PreToolUse": []ast.HookMatcherGroup{
					{Hooks: []ast.HookHandler{{Type: "command", Command: "check.sh"}}},
				},
			},
			MCP: map[string]ast.MCPConfig{
				"my-mcp": {Command: "node", Args: []string{"index.js"}},
			},
		},
	}

	r := copilot.New()
	out, _, err := r.Compile(cfg, t.TempDir())
	require.NoError(t, err)

	_, hasHooks := out.Files[".github/hooks/xcaffold-hooks.json"]
	_, hasMCP := out.Files[".vscode/mcp.json"]
	assert.True(t, hasHooks, "expected .github/hooks/xcaffold-hooks.json")
	assert.True(t, hasMCP, "expected .vscode/mcp.json")
}

// TestCompile_Copilot_Settings_UnsupportedFields verifies that Claude-specific
// settings fields (Permissions, Sandbox, StatusLine) emit CodeSettingsFieldUnsupported.
func TestCompile_Copilot_Settings_UnsupportedFields(t *testing.T) {
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

	r := copilot.New()
	_, notes, err := r.Compile(cfg, t.TempDir())
	require.NoError(t, err)

	settingsNotes := filterNotes(notes, renderer.CodeSettingsFieldUnsupported)
	fieldSet := make(map[string]bool)
	for _, n := range settingsNotes {
		fieldSet[n.Field] = true
	}

	assert.True(t, fieldSet["permissions"], "expected fidelity note for permissions")
	assert.True(t, fieldSet["sandbox"], "expected fidelity note for sandbox")
	assert.True(t, fieldSet["statusLine"], "expected fidelity note for statusLine")
}
