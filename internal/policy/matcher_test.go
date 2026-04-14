package policy

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/assert"
)

func TestMatch_NilMatch_MatchesAll(t *testing.T) {
	acc := newAgentAccessor(ast.AgentConfig{Name: "dev"})
	assert.True(t, matchResource(nil, "dev", acc))
}

func TestMatch_HasTool_Matches(t *testing.T) {
	match := &ast.PolicyMatch{HasTool: "Bash"}
	acc := newAgentAccessor(ast.AgentConfig{Tools: []string{"Bash", "Read"}})
	assert.True(t, matchResource(match, "dev", acc))
}

func TestMatch_HasTool_NoMatch(t *testing.T) {
	match := &ast.PolicyMatch{HasTool: "Bash"}
	acc := newAgentAccessor(ast.AgentConfig{Tools: []string{"Read", "Write"}})
	assert.False(t, matchResource(match, "dev", acc))
}

func TestMatch_HasField_Present(t *testing.T) {
	match := &ast.PolicyMatch{HasField: "description"}
	acc := newAgentAccessor(ast.AgentConfig{Description: "A dev agent"})
	assert.True(t, matchResource(match, "dev", acc))
}

func TestMatch_HasField_Empty(t *testing.T) {
	match := &ast.PolicyMatch{HasField: "description"}
	acc := newAgentAccessor(ast.AgentConfig{})
	assert.False(t, matchResource(match, "dev", acc))
}

func TestMatch_NameMatches_Glob(t *testing.T) {
	match := &ast.PolicyMatch{NameMatches: "deploy*"}
	acc := newAgentAccessor(ast.AgentConfig{})
	assert.True(t, matchResource(match, "deployer", acc))
	assert.False(t, matchResource(match, "reviewer", acc))
}

func TestMatch_AllConditionsAND(t *testing.T) {
	match := &ast.PolicyMatch{HasTool: "Bash", HasField: "description"}
	// Has Bash but no description -> false
	acc := newAgentAccessor(ast.AgentConfig{Tools: []string{"Bash"}})
	assert.False(t, matchResource(match, "dev", acc))
	// Has both -> true
	acc2 := newAgentAccessor(ast.AgentConfig{Tools: []string{"Bash"}, Description: "Has both"})
	assert.True(t, matchResource(match, "dev", acc2))
}
