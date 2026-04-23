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
	capsOmit := renderer.CapabilitySet{RuleEncoding: renderer.RuleEncodingCapabilities{Activation: "omit"}}
	capsFront := renderer.CapabilitySet{
		RuleEncoding:    renderer.RuleEncodingCapabilities{Activation: "frontmatter"},
		RuleActivations: []string{ast.RuleActivationAlways, ast.RuleActivationPathGlob},
	}

	t.Run("omit handles skipped activation", func(t *testing.T) {
		rule := ast.RuleConfig{Activation: ast.RuleActivationAlways}
		notes := renderer.ValidateRuleActivation(rule, capsOmit, "target", "r1")
		assert.Empty(t, notes)
	})

	t.Run("supported activation yields no notes", func(t *testing.T) {
		rule := ast.RuleConfig{Activation: ast.RuleActivationAlways}
		notes := renderer.ValidateRuleActivation(rule, capsFront, "target", "r1")
		assert.Empty(t, notes)
	})

	t.Run("unsupported activation yields fidelity note", func(t *testing.T) {
		rule := ast.RuleConfig{Activation: ast.RuleActivationModelDecided}
		notes := renderer.ValidateRuleActivation(rule, capsFront, "target", "r1")
		assert.Len(t, notes, 1)
		assert.Equal(t, renderer.CodeRuleActivationUnsupported, notes[0].Code)
	})
}
