package bir_test

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/bir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyze_EmptyConfig_ProducesEmptyUnits(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	ir, err := bir.Analyze(config, "/tmp")
	require.NoError(t, err)
	assert.Empty(t, ir.Units)
}

func TestAnalyze_SingleAgent_ProducesSourceAgentUnit(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"alpha": {Instructions: "do agent things"},
			},
		},
	}
	ir, err := bir.Analyze(config, "/tmp")
	require.NoError(t, err)
	require.Len(t, ir.Units, 1)
	assert.Equal(t, "alpha", ir.Units[0].ID)
	assert.Equal(t, bir.SourceAgent, ir.Units[0].SourceKind)
	assert.Equal(t, "do agent things", ir.Units[0].ResolvedBody)
}

func TestAnalyze_SingleSkill_ProducesSourceSkillUnit(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"beta": {Instructions: "do skill things"},
			},
		},
	}
	ir, err := bir.Analyze(config, "/tmp")
	require.NoError(t, err)
	require.Len(t, ir.Units, 1)
	assert.Equal(t, "beta", ir.Units[0].ID)
	assert.Equal(t, bir.SourceSkill, ir.Units[0].SourceKind)
	assert.Equal(t, "do skill things", ir.Units[0].ResolvedBody)
}

func TestAnalyze_SingleRule_ProducesSourceRuleUnit(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"gamma": {Instructions: "do rule things"},
			},
		},
	}
	ir, err := bir.Analyze(config, "/tmp")
	require.NoError(t, err)
	require.Len(t, ir.Units, 1)
	assert.Equal(t, "gamma", ir.Units[0].ID)
	assert.Equal(t, bir.SourceRule, ir.Units[0].SourceKind)
	assert.Equal(t, "do rule things", ir.Units[0].ResolvedBody)
}

func TestAnalyze_SingleWorkflow_ProducesSourceWorkflowUnit(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Workflows: map[string]ast.WorkflowConfig{
				"delta": {Instructions: "do workflow things"},
			},
		},
	}
	ir, err := bir.Analyze(config, "/tmp")
	require.NoError(t, err)
	require.Len(t, ir.Units, 1)
	assert.Equal(t, "delta", ir.Units[0].ID)
	assert.Equal(t, bir.SourceWorkflow, ir.Units[0].SourceKind)
	assert.Equal(t, "do workflow things", ir.Units[0].ResolvedBody)
}

func TestAnalyze_MultipleTypes_ProducesCorrectCount(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"a1": {Instructions: "agent one"},
				"a2": {Instructions: "agent two"},
			},
			Skills: map[string]ast.SkillConfig{
				"s1": {Instructions: "skill one"},
			},
			Rules: map[string]ast.RuleConfig{
				"r1": {Instructions: "rule one"},
			},
			Workflows: map[string]ast.WorkflowConfig{
				"w1": {Instructions: "workflow one"},
			},
		},
	}
	ir, err := bir.Analyze(config, "/tmp")
	require.NoError(t, err)
	assert.Len(t, ir.Units, 5)
}

func TestAnalyze_ResolvedBodyMatchesInstructions(t *testing.T) {
	const body = "detailed instructions body"
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"x": {Instructions: body},
			},
		},
	}
	ir, err := bir.Analyze(config, "/tmp")
	require.NoError(t, err)
	require.Len(t, ir.Units, 1)
	assert.Equal(t, body, ir.Units[0].ResolvedBody)
}
