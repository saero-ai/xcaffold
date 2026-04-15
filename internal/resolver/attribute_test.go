package resolver

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveAttributes_StringField(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Skills = map[string]ast.SkillConfig{
		"tdd": {Description: "Test-driven development"},
	}
	config.Agents = map[string]ast.AgentConfig{
		"developer": {Description: "${skill.tdd.description}"},
	}
	err := ResolveAttributes(config)
	require.NoError(t, err)
	assert.Equal(t, "Test-driven development", config.Agents["developer"].Description)
}

func TestResolveAttributes_StringSliceField(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Skills = map[string]ast.SkillConfig{
		"tdd": {AllowedTools: []string{"Bash", "Read", "Write"}},
	}
	config.Agents = map[string]ast.AgentConfig{
		"developer": {Tools: []string{"${skill.tdd.allowed-tools}"}},
	}
	err := ResolveAttributes(config)
	require.NoError(t, err)
	assert.Equal(t, []string{"Bash", "Read", "Write"}, config.Agents["developer"].Tools)
}

func TestResolveAttributes_StringInterpolation(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Skills = map[string]ast.SkillConfig{
		"tdd": {Description: "TDD"},
	}
	config.Agents = map[string]ast.AgentConfig{
		"developer": {Description: "Developer using ${skill.tdd.description} workflow"},
	}
	err := ResolveAttributes(config)
	require.NoError(t, err)
	assert.Equal(t, "Developer using TDD workflow", config.Agents["developer"].Description)
}

func TestResolveAttributes_MissingResource_Error(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Agents = map[string]ast.AgentConfig{
		"developer": {Tools: []string{"${skill.nonexistent.tools}"}},
	}
	err := ResolveAttributes(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestResolveAttributes_MissingField_Error(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Skills = map[string]ast.SkillConfig{
		"tdd": {Description: "TDD"},
	}
	config.Agents = map[string]ast.AgentConfig{
		"developer": {Description: "${skill.tdd.nonexistent}"},
	}
	err := ResolveAttributes(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestResolveAttributes_CircularReference_Error(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Agents = map[string]ast.AgentConfig{
		"dev": {Description: "${agent.ops.description}"},
		"ops": {Description: "${agent.dev.description}"},
	}
	err := ResolveAttributes(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular")
}

func TestResolveAttributes_NoReferences_Passthrough(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Agents = map[string]ast.AgentConfig{
		"developer": {Description: "Plain text, no refs", Model: "sonnet"},
	}
	err := ResolveAttributes(config)
	require.NoError(t, err)
	assert.Equal(t, "Plain text, no refs", config.Agents["developer"].Description)
	assert.Equal(t, "sonnet", config.Agents["developer"].Model)
}
