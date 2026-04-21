package renderer_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/renderer/antigravity"
	"github.com/saero-ai/xcaffold/internal/renderer/claude"
	"github.com/saero-ai/xcaffold/internal/renderer/copilot"
	"github.com/saero-ai/xcaffold/internal/renderer/cursor"
	"github.com/saero-ai/xcaffold/internal/renderer/gemini"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// groundTruthModelRecord mirrors the per-record shape in models.json.
type groundTruthModelRecord struct {
	Provider string `json:"provider"`
	ModelID  string `json:"model_id"`
}

// groundTruthModelsDB mirrors the top-level shape of models.json.
type groundTruthModelsDB struct {
	Records []groundTruthModelRecord `json:"records"`
}

// groundTruthDir returns the path to the ground truth database directory.
// It walks up from this source file to find the project root, then descends
// into docs/agentic/data/ground_truth/db/. Falls back to the
// XCAFFOLD_GROUND_TRUTH_DIR environment variable if set.
func groundTruthDir(t *testing.T) string {
	t.Helper()
	if dir := os.Getenv("XCAFFOLD_GROUND_TRUTH_DIR"); dir != "" {
		return dir
	}
	// __FILE__ is internal/renderer/provider_features_test.go inside the worktree.
	// We climb 5 levels to reach the monorepo root where docs/ lives.
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Skip("cannot determine source file path; skipping ground truth binding test")
	}
	// file -> internal/renderer -> worktree root -> .worktrees -> xcaffold root -> project root
	root := filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "..")
	return filepath.Join(root, "docs", "agentic", "data", "ground_truth", "db")
}

// TestResolveModel_GroundTruthBinding verifies that every model ID produced by
// ResolveModel for a known alias actually exists in the verified ground truth
// models database. If the ground truth files are absent (e.g. in CI without the
// full monorepo checkout), the test is skipped rather than failed.
func TestResolveModel_GroundTruthBinding(t *testing.T) {
	dir := groundTruthDir(t)
	dbPath := filepath.Join(dir, "models.json")

	data, err := os.ReadFile(dbPath)
	if os.IsNotExist(err) {
		t.Skipf("ground truth database not found at %s; skipping binding test", dbPath)
	}
	require.NoError(t, err, "reading models.json")

	var db groundTruthModelsDB
	require.NoError(t, json.Unmarshal(data, &db), "parsing models.json")

	// providerTargetName maps the ground truth "provider" display name to the
	// xcaffold target name used in ResolveModel.
	providerTargetName := map[string]string{
		"Claude Code":    "claude",
		"Gemini CLI":     "gemini",
		"GitHub Copilot": "copilot",
		"Cursor":         "cursor",
	}
	// Antigravity has no model field; its records are not in models.json.

	// Build a set of valid model IDs per xcaffold target name.
	validModels := make(map[string]map[string]bool)
	for _, rec := range db.Records {
		target, ok := providerTargetName[rec.Provider]
		if !ok {
			continue // provider not mapped to an xcaffold target
		}
		if validModels[target] == nil {
			validModels[target] = make(map[string]bool)
		}
		validModels[target][rec.ModelID] = true
	}

	// knownAliases are the xcaffold cross-provider model aliases understood by
	// ResolveModel. They are defined in internal/renderer/models.go.
	knownAliases := []string{"sonnet-4", "opus-4", "haiku-3.5"}
	targets := []string{"claude", "gemini", "copilot", "cursor"}

	for _, alias := range knownAliases {
		for _, target := range targets {
			alias, target := alias, target
			t.Run(alias+"/"+target, func(t *testing.T) {
				resolved, ok := renderer.ResolveModel(alias, target)
				if !ok {
					t.Skipf("ResolveModel(%q, %q) returned ok=false; target may not support models", alias, target)
				}
				known, hasTarget := validModels[target]
				if !hasTarget {
					t.Skipf("no ground truth records for target %q", target)
				}
				assert.True(t, known[resolved],
					"ResolveModel(%q, %q) = %q is not a known model ID in the ground truth database for provider %q",
					alias, target, resolved, target)
			})
		}
	}
}
