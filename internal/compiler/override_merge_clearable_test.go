package compiler

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestMerge_ClearableList_Clear(t *testing.T) {
	base := ast.AgentConfig{
		Description: "test agent",
		Tools:       ast.ClearableList{Values: []string{"Read", "Write"}},
	}
	override := ast.AgentConfig{
		Tools: ast.ClearableList{Cleared: true},
	}
	result := mergeAgentConfig(base, override)
	require.True(t, result.Tools.Cleared, "Tools should be cleared")
	require.Nil(t, result.Tools.Values, "Cleared tools should have nil values")
}

func TestMerge_ClearableList_Replace(t *testing.T) {
	base := ast.AgentConfig{
		Description: "test agent",
		Tools:       ast.ClearableList{Values: []string{"Read", "Write"}},
	}
	override := ast.AgentConfig{
		Tools: ast.ClearableList{Values: []string{"Bash", "Grep"}},
	}
	result := mergeAgentConfig(base, override)
	require.False(t, result.Tools.Cleared)
	require.Equal(t, []string{"Bash", "Grep"}, result.Tools.Values)
}

func TestMerge_ClearableList_Inherit(t *testing.T) {
	base := ast.AgentConfig{
		Description: "test agent",
		Tools:       ast.ClearableList{Values: []string{"Read", "Write"}},
	}
	override := ast.AgentConfig{
		Tools: ast.ClearableList{},
	}
	result := mergeAgentConfig(base, override)
	require.False(t, result.Tools.Cleared)
	require.Equal(t, []string{"Read", "Write"}, result.Tools.Values)
}

func TestMerge_ClearableList_Clear_AllAgentFields(t *testing.T) {
	base := ast.AgentConfig{
		Description:     "test agent",
		Tools:           ast.ClearableList{Values: []string{"Read"}},
		DisallowedTools: ast.ClearableList{Values: []string{"Write"}},
		Skills:          ast.ClearableList{Values: []string{"tdd"}},
		Rules:           ast.ClearableList{Values: []string{"secure"}},
		MCP:             ast.ClearableList{Values: []string{"server1"}},
		Assertions:      ast.ClearableList{Values: []string{"assert1"}},
	}
	override := ast.AgentConfig{
		Tools:           ast.ClearableList{Cleared: true},
		DisallowedTools: ast.ClearableList{Cleared: true},
		Skills:          ast.ClearableList{Cleared: true},
		Rules:           ast.ClearableList{Cleared: true},
		MCP:             ast.ClearableList{Cleared: true},
		Assertions:      ast.ClearableList{Cleared: true},
	}
	result := mergeAgentConfig(base, override)
	require.True(t, result.Tools.Cleared)
	require.True(t, result.DisallowedTools.Cleared)
	require.True(t, result.Skills.Cleared)
	require.True(t, result.Rules.Cleared)
	require.True(t, result.MCP.Cleared)
	require.True(t, result.Assertions.Cleared)
}

func TestMerge_ClearableList_Clear_SkillFields(t *testing.T) {
	base := ast.SkillConfig{
		AllowedTools: ast.ClearableList{Values: []string{"Read"}},
		Artifacts:    []string{"artifact.txt"},
	}
	override := ast.SkillConfig{
		AllowedTools: ast.ClearableList{Cleared: true},
	}
	result := mergeSkillConfig(base, override)
	require.True(t, result.AllowedTools.Cleared, "AllowedTools should be cleared")
	require.Equal(t, []string{"artifact.txt"}, result.Artifacts, "Artifacts should be inherited")
}

func TestMerge_ClearableList_Clear_RuleFields(t *testing.T) {
	base := ast.RuleConfig{
		Paths:         ast.ClearableList{Values: []string{"*.go"}},
		ExcludeAgents: ast.ClearableList{Values: []string{"code-review"}},
	}
	override := ast.RuleConfig{
		Paths:         ast.ClearableList{Cleared: true},
		ExcludeAgents: ast.ClearableList{},
	}
	result := mergeRuleConfig(base, override)
	require.True(t, result.Paths.Cleared, "Paths should be cleared")
	require.Equal(t, []string{"code-review"}, result.ExcludeAgents.Values, "ExcludeAgents should be inherited")
}

func TestMerge_ClearableList_Replace_DoesNotShareSlice(t *testing.T) {
	base := ast.AgentConfig{Description: "test agent"}
	override := ast.AgentConfig{
		Tools: ast.ClearableList{Values: []string{"Read", "Write"}},
	}
	result := mergeAgentConfig(base, override)
	override.Tools.Values[0] = "CHANGED"
	require.Equal(t, "Read", result.Tools.Values[0], "result should be independent copy")
}

// TestClearableList_NullOnValueType_IsInherit documents that yaml.v3 does NOT
// call UnmarshalYAML for a null value when the field is a value-type ClearableList
// (not a pointer). The field is left at its zero value, which has the same
// semantics as absent — i.e. inherit from base. Only `[]` (empty sequence) sets
// Cleared=true.
func TestClearableList_NullOnValueType_IsInherit(t *testing.T) {
	input := `---
name: test-agent
tools: ~
skills:
  - tdd
`
	var agent ast.AgentConfig
	err := yaml.Unmarshal([]byte(input), &agent)
	require.NoError(t, err)
	// null on a value-type ClearableList field produces the zero value (inherit semantics).
	require.False(t, agent.Tools.Cleared)
	require.Nil(t, agent.Tools.Values)
	require.True(t, agent.Tools.IsEmpty())
	// A non-null populated field works normally.
	require.Equal(t, []string{"tdd"}, agent.Skills.Values)
}
