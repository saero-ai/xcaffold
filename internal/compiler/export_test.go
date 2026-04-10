package compiler

import (
	"encoding/json"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
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
				"deploy": {Description: "Deploy skill", Instructions: "Deploy it."},
			},
		},
	}

	compiled, err := Compile(config, "", "")
	require.NoError(t, err)

	exported, err := ExportPlugin(config, compiled)
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
		Settings: ast.SettingsConfig{
			Model: "claude-sonnet-4-6",
		},
	}

	compiled, err := Compile(config, "", "")
	require.NoError(t, err)

	exported, err := ExportPlugin(config, compiled)
	require.NoError(t, err)

	_, hasSettings := exported.Files["settings.json"]
	assert.False(t, hasSettings, "settings.json should not be in plugin export")
}

func TestExportPlugin_RemapsHooks(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "test"},
		ResourceScope: ast.ResourceScope{
			Hooks: ast.HookConfig{
				"SessionStart": []ast.HookMatcherGroup{
					{Hooks: []ast.HookHandler{{Type: "command", Command: "echo hi"}}},
				},
			},
		},
	}

	compiled, err := Compile(config, "", "")
	require.NoError(t, err)

	exported, err := ExportPlugin(config, compiled)
	require.NoError(t, err)

	_, hasOldHooks := exported.Files["hooks.json"]
	assert.False(t, hasOldHooks, "hooks.json at root should not exist")

	_, hasNewHooks := exported.Files["hooks/hooks.json"]
	assert.True(t, hasNewHooks, "hooks should be at hooks/hooks.json")
}
