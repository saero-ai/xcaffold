package renderer_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/renderer"
	antigravity "github.com/saero-ai/xcaffold/providers/antigravity"
	"github.com/saero-ai/xcaffold/providers/claude"
	copilot "github.com/saero-ai/xcaffold/providers/copilot"
	"github.com/saero-ai/xcaffold/providers/cursor"
	gemini "github.com/saero-ai/xcaffold/providers/gemini"
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
	skillArtifactDirs   map[string]string
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
			skillArtifactDirs: map[string]string{
				"references": "references",
				"scripts":    "scripts",
				"assets":     "assets",
				"examples":   "",
			},
			ruleActivations: []string{"always", "path-glob"},
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
			skillArtifactDirs: map[string]string{
				"references": "references",
				"scripts":    "scripts",
				"assets":     "assets",
				"examples":   "references",
			},
			ruleActivations: []string{"always", "path-glob", "manual-mention"},
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
			skillArtifactDirs: map[string]string{
				"references": "references",
				"scripts":    "scripts",
				"assets":     "assets",
				"examples":   "references",
			},
			ruleActivations: []string{"always", "path-glob"},
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
			skillArtifactDirs: map[string]string{
				"references": "references",
				"scripts":    "scripts",
				"assets":     "assets",
				"examples":   "examples",
			},
			ruleActivations: []string{"always", "path-glob"},
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
			memory:              false, // deferred — native format not yet implemented
			projectInstructions: true,
			skillArtifactDirs: map[string]string{
				"references": "examples",
				"scripts":    "scripts",
				"assets":     "resources",
				"examples":   "examples",
			},
			ruleActivations: []string{"always", "path-glob", "model-decided"},
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
			assert.Equal(t, exp.skillArtifactDirs, caps.SkillArtifactDirs, "SkillArtifactDirs")
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

// referenceDataModelRecord mirrors the per-record shape in models.json.
type referenceDataModelRecord struct {
	Provider string `json:"provider"`
	ModelID  string `json:"model_id"`
}

// referenceDataModelsDB mirrors the top-level shape of models.json.
type referenceDataModelsDB struct {
	Records []referenceDataModelRecord `json:"records"`
}

// referenceDataDir returns the path to the provider reference database directory.
// It walks up from this source file to find the Go module root (go.mod), then
// goes one level further to reach the monorepo root where docs/ lives. Falls
// back to the XCAFFOLD_REFERENCE_DATA_DIR environment variable if set.
//
// This approach handles both the main working tree and any worktree checkout
// without hard-coding a fixed level count.
func referenceDataDir(t *testing.T) string {
	t.Helper()
	if dir := os.Getenv("XCAFFOLD_REFERENCE_DATA_DIR"); dir != "" {
		return dir
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Skip("cannot determine source file path; skipping reference data binding test")
	}
	// Walk up from the source file's directory until we find go.mod (the Go
	// module root, i.e. xcaffold/). The monorepo root (xcaffold-project/) is
	// one level above that, and docs/ lives there.
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			// dir is now xcaffold/ — parent is the monorepo root.
			root := filepath.Dir(dir)
			candidate := filepath.Join(root, "docs", "agentic", "data", "ground_truth", "db")
			if _, err := os.Stat(candidate); err != nil {
				t.Skipf("provider reference database not found at %s; skipping binding test", candidate)
			}
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without finding go.mod.
			t.Skip("could not locate go.mod walking up from source file; skipping reference data binding test")
		}
		dir = parent
	}
}

// TestResolveModel_ReferenceDataBinding verifies that every model ID produced by
// ResolveModel for a known alias actually exists in the verified provider reference
// models database. If the reference data files are absent (e.g. in CI without the
// full monorepo checkout), the test is skipped rather than failed.
func TestResolveModel_ReferenceDataBinding(t *testing.T) {
	dir := referenceDataDir(t)
	dbPath := filepath.Join(dir, "models.json")

	data, err := os.ReadFile(dbPath)
	if os.IsNotExist(err) {
		t.Skipf("provider reference database not found at %s; skipping binding test", dbPath)
	}
	require.NoError(t, err, "reading models.json")

	var db referenceDataModelsDB
	require.NoError(t, json.Unmarshal(data, &db), "parsing models.json")

	// providerTargetName maps the provider "provider" display name to the
	// xcaffold target name used in ResolveModel.
	providerTargetName := map[string]string{
		"Claude Code":    "claude",
		"Gemini CLI":     "gemini",
		"GitHub Copilot": "copilot",
		"Cursor":         "cursor",
		"Codex":          "codex",
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
	knownAliases := []string{"balanced", "flagship", "fast"}
	targets := []string{"claude", "gemini", "copilot", "cursor", "codex"}

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
					t.Skipf("no reference data records for target %q", target)
				}
				assert.True(t, known[resolved],
					"ResolveModel(%q, %q) = %q is not a known model ID in the provider reference database for %q",
					alias, target, resolved, target)
			})
		}
	}
}

func TestProviderFeatures_SkillFrontmatter_ReferenceData(t *testing.T) {
	dir := referenceDataDir(t)
	dbPath := filepath.Join(dir, "skills.json")

	data, err := os.ReadFile(dbPath)
	if os.IsNotExist(err) {
		t.Skipf("provider reference database not found at %s; skipping", dbPath)
	}
	require.NoError(t, err)

	var db struct {
		Records []struct {
			Category string `json:"category"`
			Aspect   string `json:"aspect"`
			Provider string `json:"provider"`
			Value    string `json:"value"`
		} `json:"records"`
	}
	require.NoError(t, json.Unmarshal(data, &db))

	providerFields := make(map[string]map[string]bool)
	for _, rec := range db.Records {
		if rec.Category != "Frontmatter" {
			continue
		}
		if providerFields[rec.Provider] == nil {
			providerFields[rec.Provider] = make(map[string]bool)
		}
		supported := !strings.Contains(strings.ToLower(rec.Value), "not supported") &&
			!strings.Contains(strings.ToLower(rec.Value), "no native")
		providerFields[rec.Provider][rec.Aspect] = supported
	}

	for provider, fields := range providerFields {
		provider, fields := provider, fields
		t.Run(provider, func(t *testing.T) {
			assert.NotEmpty(t, fields, "expected frontmatter field data for %s", provider)
		})
	}
}

func TestProviderFeatures_Capabilities_ReferenceDataConsistency(t *testing.T) {
	dir := referenceDataDir(t)

	resourceFiles := []string{"agents.json", "skills.json", "rules.json", "hooks.json", "memory.json"}

	providerTargetName := map[string]string{
		"Claude Code":    "claude",
		"Gemini CLI":     "gemini",
		"GitHub Copilot": "copilot",
		"Cursor":         "cursor",
		"Antigravity":    "antigravity",
	}

	providerSupport := make(map[string]map[string]bool)

	for _, fname := range resourceFiles {
		data, err := os.ReadFile(filepath.Join(dir, fname))
		if os.IsNotExist(err) {
			continue
		}
		require.NoError(t, err)

		var db struct {
			Records []struct {
				Category string `json:"category"`
				Aspect   string `json:"aspect"`
				Provider string `json:"provider"`
				Value    string `json:"value"`
			} `json:"records"`
		}
		require.NoError(t, json.Unmarshal(data, &db))

		resource := strings.TrimSuffix(fname, ".json")
		for _, rec := range db.Records {
			if rec.Category == "Architecture" && rec.Aspect == "Native Support" {
				target, ok := providerTargetName[rec.Provider]
				if !ok {
					continue
				}
				if providerSupport[target] == nil {
					providerSupport[target] = make(map[string]bool)
				}
				supported := !strings.Contains(strings.ToLower(rec.Value), "not supported") &&
					!strings.Contains(strings.ToLower(rec.Value), "no native")
				providerSupport[target][resource] = supported
			}
		}
	}

	renderers := map[string]renderer.TargetRenderer{
		"claude":      claude.New(),
		"cursor":      cursor.New(),
		"gemini":      gemini.New(),
		"copilot":     copilot.New(),
		"antigravity": antigravity.New(),
	}

	for target, r := range renderers {
		target, r := target, r
		caps := r.Capabilities()
		gtSupport, ok := providerSupport[target]
		if !ok {
			continue
		}
		t.Run(target, func(t *testing.T) {
			if v, exists := gtSupport["agents"]; exists {
				assert.Equal(t, v, caps.Agents, "Agents capability mismatch for %s", target)
			}
			if v, exists := gtSupport["skills"]; exists {
				assert.Equal(t, v, caps.Skills, "Skills capability mismatch for %s", target)
			}
			if v, exists := gtSupport["rules"]; exists {
				assert.Equal(t, v, caps.Rules, "Rules capability mismatch for %s", target)
			}
			if v, exists := gtSupport["hooks"]; exists {
				assert.Equal(t, v, caps.Hooks, "Hooks capability mismatch for %s", target)
			}
			if v, exists := gtSupport["memory"]; exists {
				assert.Equal(t, v, caps.Memory, "Memory capability mismatch for %s", target)
			}
		})
	}
}
