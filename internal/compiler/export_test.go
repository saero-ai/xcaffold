package compiler

import (
	"encoding/json"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	_ "github.com/saero-ai/xcaffold/providers/all"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportPlugin_GeneratesManifest(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:        "my-plugin",
			Description: "A useful plugin",
			Version:     "1.0.0",
			Author:      "Test Author",
			License:     "MIT",
		},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"helper": {Description: "A helper agent"},
			},
			Skills: map[string]ast.SkillConfig{
				"deploy": {Description: "Deploy skill", Body: "Deploy it."},
			},
		},
	}

	compiled, _, err := Compile(config, "", "claude", "", "")
	require.NoError(t, err)

	exported, err := ExportPlugin(config, compiled, "claude")
	require.NoError(t, err)

	manifestJSON, ok := exported.Files[".claude-plugin/plugin.json"]
	require.True(t, ok, "plugin.json must exist")

	var manifest map[string]any
	require.NoError(t, json.Unmarshal([]byte(manifestJSON), &manifest))
	assert.Equal(t, "my-plugin", manifest["name"])
	assert.Equal(t, "1.0.0", manifest["version"])
	assert.Equal(t, "MIT", manifest["license"])

	_, hasAgent := exported.Files["agents/helper.md"]
	assert.True(t, hasAgent)

	_, hasSkill := exported.Files["skills/deploy/SKILL.md"]
	assert.True(t, hasSkill)
}

func TestExportPlugin_SkipsSettingsJSON(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "test"},
		Settings: map[string]ast.SettingsConfig{"default": {
			Model: "claude-sonnet-4-6",
		}},
	}

	compiled, _, err := Compile(config, "", "claude", "", "")
	require.NoError(t, err)

	exported, err := ExportPlugin(config, compiled, "claude")
	require.NoError(t, err)

	_, hasSettings := exported.Files["settings.json"]
	assert.False(t, hasSettings, "settings.json should not be in plugin export")
}

func TestExportPlugin_RemapsHooks(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "test"},
		Hooks: map[string]ast.NamedHookConfig{
			"default": {
				Name: "default",
				Events: ast.HookConfig{
					"SessionStart": []ast.HookMatcherGroup{
						{Hooks: []ast.HookHandler{{Type: "command", Command: "echo hi"}}},
					},
				},
			},
		},
	}

	compiled, _, err := Compile(config, "", "claude", "", "")
	require.NoError(t, err)

	exported, err := ExportPlugin(config, compiled, "claude")
	require.NoError(t, err)

	_, hasOldHooks := exported.Files["hooks.json"]
	assert.False(t, hasOldHooks, "hooks.json at root should not exist")

	_, hasNewHooks := exported.Files["hooks/hooks.json"]
	assert.True(t, hasNewHooks, "hooks should be at hooks/hooks.json")
}

// TestExportPlugin_TargetClaude verifies the explicit "claude" target writes
// the manifest to .claude-plugin/plugin.json.
func TestExportPlugin_TargetClaude(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "test-plugin", Version: "0.1.0"},
	}

	compiled, _, err := Compile(config, "", "claude", "", "")
	require.NoError(t, err)

	exported, err := ExportPlugin(config, compiled, "claude")
	require.NoError(t, err)

	_, ok := exported.Files[".claude-plugin/plugin.json"]
	assert.True(t, ok, "manifest must be at .claude-plugin/plugin.json for claude target")
}

// TestExportPlugin_EmptyTargetReturnsError verifies that ExportPlugin with
// an empty target returns an error requiring --target to be specified.
func TestExportPlugin_EmptyTargetReturnsError(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "test-plugin"},
	}

	compiled, _, err := Compile(config, "", "claude", "", "")
	require.NoError(t, err)

	_, err = ExportPlugin(config, compiled, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--target is required")
}

// TestExportPlugin_UnsupportedTarget verifies that known targets without plugin
// support return a clear error rather than silently producing incorrect output.
func TestExportPlugin_UnsupportedTarget(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "test-plugin"},
	}

	compiled, _, err := Compile(config, "", "cursor", "", "")
	require.NoError(t, err)

	_, err = ExportPlugin(config, compiled, "cursor")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cursor")
	assert.Contains(t, err.Error(), "does not support plugin export")

	_, err = ExportPlugin(config, compiled, "antigravity")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "antigravity")
	assert.Contains(t, err.Error(), "does not support plugin export")

	_, err = ExportPlugin(config, compiled, "copilot")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "copilot")
	assert.Contains(t, err.Error(), "does not support plugin export")
}

// TestExportPlugin_UnknownTarget verifies that a completely unknown target
// returns an error that names the target.
func TestExportPlugin_UnknownTarget(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "test-plugin"},
	}

	compiled, _, err := Compile(config, "", "claude", "", "")
	require.NoError(t, err)

	_, err = ExportPlugin(config, compiled, "vscode")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "vscode")
	assert.Contains(t, err.Error(), "unknown target")
}
