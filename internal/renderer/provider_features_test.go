package renderer_test

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/renderer/antigravity"
	"github.com/saero-ai/xcaffold/internal/renderer/claude"
	"github.com/saero-ai/xcaffold/internal/renderer/copilot"
	"github.com/saero-ai/xcaffold/internal/renderer/cursor"
	"github.com/saero-ai/xcaffold/internal/renderer/gemini"
	"github.com/stretchr/testify/assert"
)

type capabilityExpectation struct {
	target              string
	renderer            renderer.TargetRenderer
	agents              bool
	skills              bool
	rules               bool
	workflows           bool
	hooks               bool
	settings            bool
	mcp                 bool
	memory              bool
	projectInstructions bool
	modelField          bool
	skillSubdirs        []string
	ruleActivations     []string
}

func TestProviderFeatures_CapabilitySets(t *testing.T) {
	expectations := []capabilityExpectation{
		{
			target:              "claude",
			renderer:            claude.New(),
			agents:              true,
			skills:              true,
			rules:               true,
			workflows:           true,
			hooks:               true,
			settings:            true,
			mcp:                 true,
			memory:              true,
			projectInstructions: true,
			modelField:          true,
			skillSubdirs:        []string{"references", "scripts", "assets"},
			ruleActivations:     []string{"always", "path-glob"},
		},
		{
			target:              "cursor",
			renderer:            cursor.New(),
			agents:              true,
			skills:              true,
			rules:               true,
			workflows:           true,
			hooks:               true,
			settings:            true,
			mcp:                 true,
			memory:              false,
			projectInstructions: true,
			modelField:          false,
			skillSubdirs:        []string{"references", "scripts", "assets"},
			ruleActivations:     []string{"always", "path-glob", "manual-mention"},
		},
		{
			target:              "gemini",
			renderer:            gemini.New(),
			agents:              true,
			skills:              true,
			rules:               true,
			workflows:           true,
			hooks:               true,
			settings:            true,
			mcp:                 true,
			memory:              false,
			projectInstructions: true,
			modelField:          true,
			skillSubdirs:        []string{},
			ruleActivations:     []string{"always", "path-glob"},
		},
		{
			target:              "copilot",
			renderer:            copilot.New(),
			agents:              true,
			skills:              true,
			rules:               true,
			workflows:           true,
			hooks:               true,
			settings:            true,
			mcp:                 true,
			memory:              false,
			projectInstructions: true,
			modelField:          true,
			skillSubdirs:        []string{},
			ruleActivations:     []string{"always", "path-glob"},
		},
		{
			target:              "antigravity",
			renderer:            antigravity.New(),
			agents:              true,
			skills:              true,
			rules:               true,
			workflows:           true,
			hooks:               false,
			settings:            true,
			mcp:                 true,
			memory:              true,
			projectInstructions: true,
			modelField:          false,
			skillSubdirs:        nil,
			ruleActivations:     []string{"always", "path-glob", "manual"},
		},
	}

	for _, exp := range expectations {
		exp := exp
		t.Run(exp.target, func(t *testing.T) {
			caps := exp.renderer.Capabilities()

			assert.Equal(t, exp.agents, caps.Agents, "Agents")
			assert.Equal(t, exp.skills, caps.Skills, "Skills")
			assert.Equal(t, exp.rules, caps.Rules, "Rules")
			assert.Equal(t, exp.workflows, caps.Workflows, "Workflows")
			assert.Equal(t, exp.hooks, caps.Hooks, "Hooks")
			assert.Equal(t, exp.settings, caps.Settings, "Settings")
			assert.Equal(t, exp.mcp, caps.MCP, "MCP")
			assert.Equal(t, exp.memory, caps.Memory, "Memory")
			assert.Equal(t, exp.projectInstructions, caps.ProjectInstructions, "ProjectInstructions")
			assert.Equal(t, exp.modelField, caps.ModelField, "ModelField")
			assert.Equal(t, exp.skillSubdirs, caps.SkillSubdirs, "SkillSubdirs")
			assert.Equal(t, exp.ruleActivations, caps.RuleActivations, "RuleActivations")
		})
	}
}

func TestProviderFeatures_TargetNames(t *testing.T) {
	renderers := map[string]renderer.TargetRenderer{
		"claude":      claude.New(),
		"cursor":      cursor.New(),
		"gemini":      gemini.New(),
		"copilot":     copilot.New(),
		"antigravity": antigravity.New(),
	}

	for expected, r := range renderers {
		assert.Equal(t, expected, r.Target(), "Target() must match the canonical name")
	}
}

func TestProviderFeatures_OutputDirs(t *testing.T) {
	cases := []struct {
		target string
		dir    string
	}{
		{"claude", ".claude"},
		{"cursor", ".cursor"},
		{"gemini", ".gemini"},
		{"copilot", ".github"},
		{"antigravity", ".agents"},
	}

	renderers := map[string]renderer.TargetRenderer{
		"claude":      claude.New(),
		"cursor":      cursor.New(),
		"gemini":      gemini.New(),
		"copilot":     copilot.New(),
		"antigravity": antigravity.New(),
	}

	for _, tc := range cases {
		r := renderers[tc.target]
		assert.Equal(t, tc.dir, r.OutputDir(), "%s OutputDir()", tc.target)
	}
}
