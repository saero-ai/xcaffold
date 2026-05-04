package policy

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequire_IsPresent_Empty_Violation(t *testing.T) {
	isPresent := true
	req := ast.PolicyRequire{Field: "description", IsPresent: &isPresent}
	acc := newAgentAccessor(ast.AgentConfig{Name: "dev"})
	v := evaluateRequire("test-policy", "dev", req, acc)
	require.NotNil(t, v)
	assert.Contains(t, v.Message, `field "description" must be present`)
}

func TestRequire_IsPresent_Set_NoViolation(t *testing.T) {
	isPresent := true
	req := ast.PolicyRequire{Field: "description", IsPresent: &isPresent}
	acc := newAgentAccessor(ast.AgentConfig{Name: "dev", Description: "A developer"})
	v := evaluateRequire("test-policy", "dev", req, acc)
	assert.Nil(t, v)
}

func TestRequire_MinLength_Short_Violation(t *testing.T) {
	minLen := 50
	req := ast.PolicyRequire{Field: "description", MinLength: &minLen}
	acc := newAgentAccessor(ast.AgentConfig{Name: "dev", Description: "Short"})
	v := evaluateRequire("test-policy", "dev", req, acc)
	require.NotNil(t, v)
	assert.Contains(t, v.Message, "below minimum 50")
}

func TestRequire_MinLength_Exact_NoViolation(t *testing.T) {
	minLen := 5
	req := ast.PolicyRequire{Field: "description", MinLength: &minLen}
	acc := newAgentAccessor(ast.AgentConfig{Name: "dev", Description: "Hello"})
	v := evaluateRequire("test-policy", "dev", req, acc)
	assert.Nil(t, v)
}

func TestRequire_MaxCount_Over_Violation(t *testing.T) {
	maxCount := 2
	req := ast.PolicyRequire{Field: "tools", MaxCount: &maxCount}
	acc := newAgentAccessor(ast.AgentConfig{Name: "dev", Tools: ast.ClearableList{Values: []string{"Bash", "Read", "Write"}}})
	v := evaluateRequire("test-policy", "dev", req, acc)
	require.NotNil(t, v)
	assert.Contains(t, v.Message, "3 items, maximum is 2")
}

func TestRequire_OneOf_Invalid_Violation(t *testing.T) {
	req := ast.PolicyRequire{Field: "model", OneOf: []string{"sonnet", "opus"}}
	acc := newAgentAccessor(ast.AgentConfig{Name: "dev", Model: "gpt-4o"})
	v := evaluateRequire("test-policy", "dev", req, acc)
	require.NotNil(t, v)
	assert.Contains(t, v.Message, `not in approved list`)
}

func TestRequire_OneOf_Valid_NoViolation(t *testing.T) {
	req := ast.PolicyRequire{Field: "model", OneOf: []string{"sonnet", "opus"}}
	acc := newAgentAccessor(ast.AgentConfig{Name: "dev", Model: "sonnet"})
	v := evaluateRequire("test-policy", "dev", req, acc)
	assert.Nil(t, v)
}

func TestDeny_ContentContains_Found(t *testing.T) {
	deny := ast.PolicyDeny{ContentContains: []string{"TODO"}}
	vs := evaluateDenyOnFile("no-todos", "agents/dev.md", "Some TODO marker here", deny)
	require.Len(t, vs, 1)
	assert.Contains(t, vs[0].Message, `forbidden string "TODO"`)
}

func TestDeny_ContentContains_CaseInsensitive(t *testing.T) {
	deny := ast.PolicyDeny{ContentContains: []string{"fixme"}}
	vs := evaluateDenyOnFile("no-fixme", "agents/dev.md", "Some FIXME marker", deny)
	require.Len(t, vs, 1)
}

func TestDeny_ContentMatches_Regex(t *testing.T) {
	deny := ast.PolicyDeny{ContentMatches: `sk-[a-zA-Z0-9]{20,}`}
	vs := evaluateDenyOnFile("no-keys", "agents/dev.md", "key: sk-abcdefghijklmnopqrstuvwxyz", deny)
	require.Len(t, vs, 1)
	assert.Contains(t, vs[0].Message, "forbidden pattern")
}

func TestDeny_ContentMatches_InvalidRegex(t *testing.T) {
	deny := ast.PolicyDeny{ContentMatches: `[invalid`}
	vs := evaluateDenyOnFile("bad-regex", "agents/dev.md", "content", deny)
	require.Len(t, vs, 1)
	assert.Contains(t, vs[0].Message, "invalid regex")
}

func TestDeny_PathContains_Traversal(t *testing.T) {
	deny := ast.PolicyDeny{PathContains: ".."}
	vs := evaluateDenyOnFile("path-safety", "agents/../../../etc/passwd", "", deny)
	require.Len(t, vs, 1)
	assert.Contains(t, vs[0].Message, `forbidden string ".."`)
}

func TestDeny_MultipleChecks_IndependentViolations(t *testing.T) {
	deny := ast.PolicyDeny{
		ContentContains: []string{"TODO"},
		PathContains:    "..",
	}
	vs := evaluateDenyOnFile("multi", "../bad/path", "has TODO", deny)
	assert.Len(t, vs, 2)
}
