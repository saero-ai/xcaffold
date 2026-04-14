package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPolicyConfig_YAMLRoundTrip(t *testing.T) {
	isPresent := true
	minLen := 50
	original := PolicyConfig{
		Name:        "require-model",
		Description: "Agents must use approved model",
		Severity:    "error",
		Target:      "agent",
		Match: &PolicyMatch{
			HasTool:     "Bash",
			NameMatches: "deploy*",
		},
		Require: []PolicyRequire{
			{Field: "model", OneOf: []string{"sonnet", "opus"}},
			{Field: "description", IsPresent: &isPresent, MinLength: &minLen},
		},
		Deny: []PolicyDeny{
			{ContentContains: []string{"TODO", "FIXME"}},
			{ContentMatches: "sk-[a-zA-Z0-9]{20,}", PathContains: ".."},
		},
	}

	data, err := yaml.Marshal(original)
	require.NoError(t, err)

	var decoded PolicyConfig
	err = yaml.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.Name, decoded.Name)
	assert.Equal(t, original.Severity, decoded.Severity)
	assert.Equal(t, original.Target, decoded.Target)
	assert.Equal(t, "Bash", decoded.Match.HasTool)
	assert.Equal(t, "deploy*", decoded.Match.NameMatches)
	assert.Len(t, decoded.Require, 2)
	assert.Equal(t, "model", decoded.Require[0].Field)
	assert.Equal(t, []string{"sonnet", "opus"}, decoded.Require[0].OneOf)
	assert.True(t, *decoded.Require[1].IsPresent)
	assert.Equal(t, 50, *decoded.Require[1].MinLength)
	assert.Len(t, decoded.Deny, 2)
	assert.Equal(t, []string{"TODO", "FIXME"}, decoded.Deny[0].ContentContains)
	assert.Equal(t, "sk-[a-zA-Z0-9]{20,}", decoded.Deny[1].ContentMatches)
	assert.Equal(t, "..", decoded.Deny[1].PathContains)
}

func TestPolicyRequire_PointerFields(t *testing.T) {
	input := `field: model
is_present: false
min_length: 0
`
	var req PolicyRequire
	err := yaml.Unmarshal([]byte(input), &req)
	require.NoError(t, err)
	require.NotNil(t, req.IsPresent, "is_present should be non-nil when explicitly set to false")
	assert.False(t, *req.IsPresent)
	require.NotNil(t, req.MinLength, "min_length should be non-nil when explicitly set to 0")
	assert.Equal(t, 0, *req.MinLength)
}

func TestPolicyMatch_AllFieldsOptional(t *testing.T) {
	input := `{}`
	var match PolicyMatch
	err := yaml.Unmarshal([]byte(input), &match)
	require.NoError(t, err)
	assert.Empty(t, match.HasTool)
	assert.Empty(t, match.HasField)
	assert.Empty(t, match.NameMatches)
	assert.Empty(t, match.TargetIncludes)
}
