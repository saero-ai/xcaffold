package ast

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

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
		AgentRefs:    []AgentManifestEntry{{ID: "developer"}},
		SkillRefs:    []string{"tdd"},
		RuleRefs:     []string{"testing"},
		WorkflowRefs: []string{"ci"},
		MCPRefs:      []string{"github"},
		PolicyRefs:   []string{"require-model"},
	}

	require.Equal(t, []AgentManifestEntry{{ID: "developer"}}, pc.AgentRefs)
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
		AgentRefs: []AgentManifestEntry{{ID: "developer"}},
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

// TestSkillConfig_Examples_RoundTrip verifies that the examples: field on
// SkillConfig is accepted by KnownFields(true) and survives a marshal/unmarshal
// round-trip with all values intact.
func TestSkillConfig_Examples_RoundTrip(t *testing.T) {
	input := `examples:
- xcf/skills/tdd/examples/basic.xcf
- xcf/skills/tdd/examples/advanced.xcf
`
	var sc SkillConfig
	dec := yaml.NewDecoder(strings.NewReader(input))
	dec.KnownFields(true)
	require.NoError(t, dec.Decode(&sc), "examples: must be a known field on SkillConfig")

	require.Len(t, sc.Examples, 2)
	require.Equal(t, "xcf/skills/tdd/examples/basic.xcf", sc.Examples[0])
	require.Equal(t, "xcf/skills/tdd/examples/advanced.xcf", sc.Examples[1])

	data, err := yaml.Marshal(sc)
	require.NoError(t, err)
	require.Contains(t, string(data), "examples:")

	var sc2 SkillConfig
	require.NoError(t, yaml.Unmarshal(data, &sc2))
	require.Equal(t, sc.Examples, sc2.Examples)
}
