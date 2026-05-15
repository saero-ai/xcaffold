package main

import (
	"bytes"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/stretchr/testify/assert"
)

// TestPrintFidelityNotes_Suppressed verifies the suppression path: callers
// pre-filter with renderer.FilterNotes before passing to printFidelityNotes,
// so printFidelityNotes receives an already-filtered (empty) slice.
func TestPrintFidelityNotes_Suppressed(t *testing.T) {
	var buf bytes.Buffer

	notes := []renderer.FidelityNote{
		{
			Level:    renderer.LevelWarning,
			Target:   "cursor",
			Kind:     "agent",
			Resource: "reviewer",
			Field:    "permission-mode",
			Code:     renderer.CodeFieldUnsupported,
			Reason:   "permission-mode unsupported",
		},
	}

	suppressed := map[string]bool{"reviewer": true}
	filtered := renderer.FilterNotes(notes, suppressed)
	// When verbose=false, warning notes should not be printed (even if not suppressed)
	printed := printFidelityNotes(&buf, filtered, false)
	assert.Equal(t, 0, printed, "empty filtered notes must not be printed")
	assert.Empty(t, buf.String())
}

func TestPrintFidelityNotes_VerboseMode_PrintsWarnings(t *testing.T) {
	var buf bytes.Buffer

	notes := []renderer.FidelityNote{
		{
			Level:    renderer.LevelWarning,
			Target:   "cursor",
			Kind:     "agent",
			Resource: "analyst",
			Field:    "isolation",
			Code:     renderer.CodeFieldUnsupported,
			Reason:   "isolation unsupported",
		},
	}

	// When verbose=true, warnings should be printed
	printed := printFidelityNotes(&buf, notes, true)
	assert.Equal(t, 1, printed)
	assert.Contains(t, buf.String(), "WARNING")
}

func TestPrintFidelityNotes_VerboseFalse_FiltersWarnings(t *testing.T) {
	var buf bytes.Buffer

	notes := []renderer.FidelityNote{
		{
			Level:    renderer.LevelWarning,
			Target:   "cursor",
			Kind:     "settings",
			Resource: "global",
			Field:    "permissions",
			Code:     renderer.CodeSettingsFieldUnsupported,
			Reason:   "permissions dropped",
		},
	}

	// When verbose=false, warnings should be filtered out
	printed := printFidelityNotes(&buf, notes, false)
	assert.Equal(t, 0, printed)
	assert.Empty(t, buf.String())
}

func TestBuildSuppressedResourcesMap_PicksUpAgentOverride(t *testing.T) {
	suppress := true
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"quiet": {
					Description: "test agent",
					Targets: map[string]ast.TargetOverride{
						"cursor": {SuppressFidelityWarnings: &suppress},
					},
				},
				"loud": {
					Description: "test agent",
					Targets: map[string]ast.TargetOverride{
						"cursor": {SuppressFidelityWarnings: nil},
					},
				},
			},
		},
	}

	got := buildSuppressedResourcesMap(config, "cursor")
	assert.True(t, got["quiet"])
	assert.False(t, got["loud"])
}

func TestBuildSuppressedResourcesMap_PicksUpSkillOverride(t *testing.T) {
	suppress := true
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"quiet-skill": {
					Targets: map[string]ast.TargetOverride{
						"cursor": {SuppressFidelityWarnings: &suppress},
					},
				},
				"loud-skill": {
					Targets: map[string]ast.TargetOverride{
						"cursor": {SuppressFidelityWarnings: nil},
					},
				},
			},
		},
	}

	got := buildSuppressedResourcesMap(config, "cursor")
	assert.True(t, got["quiet-skill"])
	assert.False(t, got["loud-skill"])
}

func TestBuildSuppressedResourcesMap_PicksUpRuleOverride(t *testing.T) {
	suppress := true
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"quiet-rule": {
					Targets: map[string]ast.TargetOverride{
						"antigravity": {SuppressFidelityWarnings: &suppress},
					},
				},
			},
		},
	}

	got := buildSuppressedResourcesMap(config, "antigravity")
	assert.True(t, got["quiet-rule"])
}

func TestBuildSuppressedResourcesMap_PicksUpWorkflowOverride(t *testing.T) {
	suppress := true
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Workflows: map[string]ast.WorkflowConfig{
				"quiet-wf": {
					Targets: map[string]ast.TargetOverride{
						"cursor": {SuppressFidelityWarnings: &suppress},
					},
				},
			},
		},
	}

	got := buildSuppressedResourcesMap(config, "cursor")
	assert.True(t, got["quiet-wf"])
}

func TestBuildSuppressedResourcesMap_MixedResourceKinds(t *testing.T) {
	suppress := true
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"agent-a": {
					Description: "test agent",
					Targets: map[string]ast.TargetOverride{
						"cursor": {SuppressFidelityWarnings: &suppress},
					},
				},
			},
			Skills: map[string]ast.SkillConfig{
				"skill-b": {
					Targets: map[string]ast.TargetOverride{
						"cursor": {SuppressFidelityWarnings: &suppress},
					},
				},
			},
			Rules: map[string]ast.RuleConfig{
				"rule-c": {
					Targets: map[string]ast.TargetOverride{
						"cursor": {},
					},
				},
			},
		},
	}

	got := buildSuppressedResourcesMap(config, "cursor")
	assert.True(t, got["agent-a"])
	assert.True(t, got["skill-b"])
	assert.False(t, got["rule-c"])
}
