package renderer_test

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/stretchr/testify/assert"
)

func TestBuildRuleDescriptionFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		caps        renderer.CapabilitySet
		description string
		expected    string
	}{
		{
			name:        "frontmatter supported and desc provided",
			caps:        renderer.CapabilitySet{RuleEncoding: renderer.RuleEncodingCapabilities{Description: "frontmatter"}},
			description: "Test rule",
			expected:    "description: Test rule\n",
		},
		{
			name:        "frontmatter supported but desc empty",
			caps:        renderer.CapabilitySet{RuleEncoding: renderer.RuleEncodingCapabilities{Description: "frontmatter"}},
			description: "",
			expected:    "",
		},
		{
			name:        "omit supported and desc provided",
			caps:        renderer.CapabilitySet{RuleEncoding: renderer.RuleEncodingCapabilities{Description: "omit"}},
			description: "Test rule",
			expected:    "",
		},
		{
			name:        "prose supported and desc provided",
			caps:        renderer.CapabilitySet{RuleEncoding: renderer.RuleEncodingCapabilities{Description: "prose"}},
			description: "Test rule",
			expected:    "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rule := ast.RuleConfig{Description: tc.description}
			actual := renderer.BuildRuleDescriptionFrontmatter(rule, tc.caps)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestBuildRuleProsePrefix(t *testing.T) {
	tests := []struct {
		name        string
		caps        renderer.CapabilitySet
		description string
		expected    string
	}{
		{
			name:        "prose supported and desc provided",
			caps:        renderer.CapabilitySet{RuleEncoding: renderer.RuleEncodingCapabilities{Description: "prose"}},
			description: "Test rule",
			expected:    "Test rule\n\n",
		},
		{
			name:        "frontmatter supported and desc provided",
			caps:        renderer.CapabilitySet{RuleEncoding: renderer.RuleEncodingCapabilities{Description: "frontmatter"}},
			description: "Test rule",
			expected:    "",
		},
		{
			name:        "prose supported but desc empty",
			caps:        renderer.CapabilitySet{RuleEncoding: renderer.RuleEncodingCapabilities{Description: "prose"}},
			description: "",
			expected:    "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rule := ast.RuleConfig{Description: tc.description}
			actual := renderer.BuildRuleProsePrefix(rule, tc.caps)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestValidateRuleActivation(t *testing.T) {
	rule := ast.RuleConfig{Activation: string(ast.RuleActivationManualMention)}

	capsOmit := renderer.CapabilitySet{RuleEncoding: renderer.RuleEncodingCapabilities{Activation: "omit"}}
	capsFront := renderer.CapabilitySet{
		RuleEncoding:    renderer.RuleEncodingCapabilities{Activation: "frontmatter"},
		RuleActivations: []string{ast.RuleActivationAlways, ast.RuleActivationPathGlob},
	}

	t.Run("omit capability without supported activation returns false", func(t *testing.T) {
		valid := renderer.ValidateRuleActivation(rule, capsOmit)
		assert.False(t, valid, "expected false because manual-mention is not in capsOmit supported list")
	})

	t.Run("unsupported activation returns false", func(t *testing.T) {
		valid := renderer.ValidateRuleActivation(rule, capsFront)
		assert.False(t, valid, "expected false when activation is unsupported")
	})

	t.Run("supported activation returns true", func(t *testing.T) {
		rule.Activation = string(ast.RuleActivationAlways)
		valid := renderer.ValidateRuleActivation(rule, capsFront)
		assert.True(t, valid, "expected true when activation is supported")
	})
}
