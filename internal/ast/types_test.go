package ast

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestProjectConfig_InstructionsFields_RoundTrip(t *testing.T) {
	config := ProjectConfig{
		Name:         "payments-monorepo",
		Instructions: "Use pnpm. PostgreSQL 16.",
		InstructionsImports: []string{
			"xcf/instructions/style-guide.md",
		},
		InstructionsScopes: []InstructionsScope{
			{
				Path:             "packages/worker",
				InstructionsFile: "xcf/instructions/scopes/packages-worker.md",
				MergeStrategy:    "concat",
				SourceProvider:   "claude",
				SourceFilename:   "CLAUDE.md",
			},
		},
	}

	data, err := yaml.Marshal(config)
	require.NoError(t, err)
	content := string(data)
	require.Contains(t, content, "instructions:")
	require.Contains(t, content, "instructions-imports:")
	require.Contains(t, content, "instructions-scopes:")
	require.Contains(t, content, "merge-strategy: concat")
	require.Contains(t, content, "source-provider: claude")

	var out ProjectConfig
	require.NoError(t, yaml.Unmarshal(data, &out))
	require.Equal(t, "Use pnpm. PostgreSQL 16.", out.Instructions)
	require.Len(t, out.InstructionsImports, 1)
	require.Len(t, out.InstructionsScopes, 1)
	require.Equal(t, "packages/worker", out.InstructionsScopes[0].Path)
	require.Equal(t, "concat", out.InstructionsScopes[0].MergeStrategy)
}

func TestInstructionsScope_WithVariants_RoundTrip(t *testing.T) {
	scope := InstructionsScope{
		Path:          "packages/shared",
		MergeStrategy: "concat",
		Variants: map[string]InstructionsVariant{
			"claude": {
				InstructionsFile: "xcf/instructions/scopes/packages-shared.claude.md",
				SourceFilename:   "CLAUDE.md",
			},
			"cursor": {
				InstructionsFile: "xcf/instructions/scopes/packages-shared.cursor.md",
				SourceFilename:   "AGENTS.md",
			},
		},
		Reconciliation: &ReconciliationConfig{
			Strategy:       "per-target",
			LastReconciled: "2026-04-15T15:30:00Z",
			Notes:          "claude variant has security section absent from cursor variant",
		},
	}

	data, err := yaml.Marshal(scope)
	require.NoError(t, err)
	content := string(data)
	require.Contains(t, content, "variants:")
	require.Contains(t, content, "reconciliation:")
	require.Contains(t, content, "strategy: per-target")

	var out InstructionsScope
	require.NoError(t, yaml.Unmarshal(data, &out))
	require.Len(t, out.Variants, 2)
	require.NotNil(t, out.Reconciliation)
	require.Equal(t, "per-target", out.Reconciliation.Strategy)
}

func TestProjectConfig_MutualExclusivity_BothSet_IsNotParseable(t *testing.T) {
	// Mutual exclusivity is enforced at parser level, not yaml.Unmarshal level.
	// This test verifies the struct can hold both (parser rejects it separately).
	config := ProjectConfig{
		Name:             "test",
		Instructions:     "inline",
		InstructionsFile: "xcf/instructions/root.md",
	}
	_, err := yaml.Marshal(config)
	require.NoError(t, err) // struct allows it; parser rejects it
}

// TestProjectConfig_AdvisoryRefFields verifies that the six advisory ref
// fields on ProjectConfig are accessible at the Go level and are intentionally
// excluded from YAML marshaling.
//
// Design invariant: ref fields use yaml:"-" because ProjectConfig also embeds
// ResourceScope inline, which defines the same YAML keys (agents, skills, etc.)
// as map types. Exposing both at the same yaml key would cause yaml.v3 to panic
// with a "duplicated key" error. The parser uses a separate projectDocFields
// struct (without ResourceScope) to decode the ref lists, then assigns them to
// these fields post-decode.
func TestProjectConfig_AdvisoryRefFields(t *testing.T) {
	pc := &ProjectConfig{
		Name:         "my-proj",
		AgentRefs:    []string{"developer"},
		SkillRefs:    []string{"tdd"},
		RuleRefs:     []string{"testing"},
		WorkflowRefs: []string{"ci"},
		MCPRefs:      []string{"github"},
		PolicyRefs:   []string{"require-model"},
	}

	require.Equal(t, []string{"developer"}, pc.AgentRefs)
	require.Equal(t, []string{"tdd"}, pc.SkillRefs)
	require.Equal(t, []string{"testing"}, pc.RuleRefs)
	require.Equal(t, []string{"ci"}, pc.WorkflowRefs)
	require.Equal(t, []string{"github"}, pc.MCPRefs)
	require.Equal(t, []string{"require-model"}, pc.PolicyRefs)

	// Ref fields are intentionally excluded from YAML output (yaml:"-").
	// They are populated by the parser, not decoded from YAML, to avoid
	// a duplicate-key collision with ResourceScope's map-typed fields.
	data, err := yaml.Marshal(pc)
	require.NoError(t, err)
	content := string(data)
	// "agents:" in the output would come from ResourceScope.Agents (map), not AgentRefs.
	// Since ResourceScope.Agents is nil here, "agents:" should not appear at all.
	require.NotContains(t, content, "agents:")
	require.NotContains(t, content, "skills:")
	require.NotContains(t, content, "rules:")
	require.NotContains(t, content, "workflows:")
	require.NotContains(t, content, "mcp:")
	require.NotContains(t, content, "policies:")
}

// TestProjectConfig_AdvisoryRefFields_DoNotConflictWithResourceScope verifies
// that setting ref fields alongside ResourceScope.Agents (the map form) does
// not panic or produce unexpected YAML output.
func TestProjectConfig_AdvisoryRefFields_DoNotConflictWithResourceScope(t *testing.T) {
	pc := &ProjectConfig{
		Name:      "my-proj",
		AgentRefs: []string{"developer"},
		ResourceScope: ResourceScope{
			Agents: map[string]AgentConfig{
				"developer": {Name: "developer", Model: "claude-sonnet-4-5"},
			},
		},
	}

	// Marshaling must not panic. yaml.v3 only panics on duplicate keys when
	// both the outer struct field and the inlined embedded field expose the
	// same yaml key name. AgentRefs uses yaml:"-", so there is no collision.
	require.NotPanics(t, func() {
		data, err := yaml.Marshal(pc)
		require.NoError(t, err)
		// The Agents map (from ResourceScope) is the one that appears in YAML.
		require.Contains(t, string(data), "agents:")
		require.Contains(t, string(data), "developer:")
	})
}

// --- BlueprintConfig tests ---

func TestBlueprintConfig_Fields_Exist(t *testing.T) {
	p := BlueprintConfig{
		Name:        "backend",
		Description: "Backend engineering",
		Extends:     "base",
		Active:      true,
		Agents:      []string{"developer"},
		Skills:      []string{"tdd"},
		Rules:       []string{"testing"},
		Workflows:   []string{"commit-changes"},
		MCP:         []string{"database-tools"},
		Policies:    []string{"security"},
		Memory:      []string{"arch-decisions"},
		Settings:    "default",
		Hooks:       "ci",
		Inherited:   false,
	}
	require.Equal(t, "backend", p.Name)
	require.True(t, p.Active)
}

func TestXcaffoldConfig_BlueprintsMap_Exists(t *testing.T) {
	cfg := &XcaffoldConfig{
		Blueprints: map[string]BlueprintConfig{
			"backend": {Name: "backend"},
		},
	}
	require.Len(t, cfg.Blueprints, 1)
}

func TestProjectConfig_BlueprintRefs_Exists(t *testing.T) {
	pc := &ProjectConfig{
		BlueprintRefs: []string{"backend", "frontend"},
	}
	require.Len(t, pc.BlueprintRefs, 2)
}
