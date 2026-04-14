package policy

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluate_NoPolicies_NoViolations(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	out := &output.Output{Files: map[string]string{}}
	violations := Evaluate(nil, config, out)
	assert.Empty(t, violations)
}

func TestEvaluate_SeverityOff_Skipped(t *testing.T) {
	policies := map[string]ast.PolicyConfig{
		"disabled": {
			Name:     "disabled",
			Severity: SeverityOff,
			Target:   "agent",
		},
	}
	config := &ast.XcaffoldConfig{}
	out := &output.Output{Files: map[string]string{}}
	violations := Evaluate(policies, config, out)
	assert.Empty(t, violations)
}

func TestMergeBuiltins_AllBuiltinsLoad(t *testing.T) {
	merged := mergeBuiltins(nil)
	require.Len(t, merged, 4)
	assert.Contains(t, merged, "path-safety")
	assert.Contains(t, merged, "agent-has-description")
	assert.Contains(t, merged, "settings-schema")
	assert.Contains(t, merged, "no-empty-skills")
}

func TestMergeBuiltins_UserOverridesBuiltin(t *testing.T) {
	userPolicies := map[string]ast.PolicyConfig{
		"path-safety": {
			Name:     "path-safety",
			Severity: SeverityOff,
			Target:   "output",
		},
	}
	merged := mergeBuiltins(userPolicies)
	assert.Equal(t, SeverityOff, merged["path-safety"].Severity)
}

func TestMergeBuiltins_UserAddsNew(t *testing.T) {
	userPolicies := map[string]ast.PolicyConfig{
		"custom": {
			Name:     "custom",
			Severity: SeverityError,
			Target:   "agent",
		},
	}
	merged := mergeBuiltins(userPolicies)
	assert.Len(t, merged, 5) // 4 built-in + 1 custom
	assert.Contains(t, merged, "custom")
}

func TestEvaluate_AgentPolicy_Violation(t *testing.T) {
	isPresent := true
	policies := map[string]ast.PolicyConfig{
		// disable built-in that would also fire on a missing description
		"agent-has-description": {
			Name:     "agent-has-description",
			Severity: SeverityOff,
			Target:   "agent",
		},
		"needs-desc": {
			Name:     "needs-desc",
			Severity: SeverityError,
			Target:   "agent",
			Require:  []ast.PolicyRequire{{Field: "description", IsPresent: &isPresent}},
		},
	}
	config := &ast.XcaffoldConfig{}
	config.Agents = map[string]ast.AgentConfig{
		"dev": {Name: "dev"}, // no description
	}
	out := &output.Output{Files: map[string]string{}}
	violations := Evaluate(policies, config, out)
	require.Len(t, violations, 1)
	assert.Equal(t, "needs-desc", violations[0].PolicyName)
	assert.Equal(t, SeverityError, violations[0].Severity)
}

func TestEvaluate_OutputDenyPolicy_Violation(t *testing.T) {
	policies := map[string]ast.PolicyConfig{
		// disable built-in that would also flag ".." traversal
		"path-safety": {
			Name:     "path-safety",
			Severity: SeverityOff,
			Target:   "output",
		},
		"no-traversal": {
			Name:     "no-traversal",
			Severity: SeverityError,
			Target:   "output",
			Deny:     []ast.PolicyDeny{{PathContains: ".."}},
		},
	}
	config := &ast.XcaffoldConfig{}
	out := &output.Output{Files: map[string]string{
		"agents/../../../etc/passwd": "evil",
	}}
	violations := Evaluate(policies, config, out)
	require.Len(t, violations, 1)
	assert.Contains(t, violations[0].Message, `forbidden string ".."`)
}
