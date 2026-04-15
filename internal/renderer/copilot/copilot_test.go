package copilot_test

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer/copilot"
	"github.com/stretchr/testify/require"
)

func TestCompileCopilotRule_Activation_Always(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"security": {
					Description:  "Security guide.",
					Activation:   ast.RuleActivationAlways,
					Instructions: "Follow OWASP.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files[".github/instructions/security.instructions.md"]
	require.Contains(t, content, `applyTo: "**"`)
}

func TestCompileCopilotRule_Activation_PathGlob(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"api-style": {
					Activation:   ast.RuleActivationPathGlob,
					Paths:        []string{"src/api/**", "packages/api/**"},
					Instructions: "REST conventions.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files[".github/instructions/api-style.instructions.md"]
	require.Contains(t, content, `applyTo: "src/api/**, packages/api/**"`)
}

func TestCompileCopilotRule_ExcludeAgents_Single(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"pr-review": {
					Activation:    ast.RuleActivationAlways,
					ExcludeAgents: []string{"code-review"},
					Instructions:  "Review standards.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files[".github/instructions/pr-review.instructions.md"]
	require.Contains(t, content, "excludeAgent:")
	require.Contains(t, content, "code-review")
}

func TestCompileCopilotRule_ExcludeAgents_Multiple(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"security": {
					Activation:    ast.RuleActivationAlways,
					ExcludeAgents: []string{"code-review", "cloud-agent"},
					Instructions:  "Security.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files[".github/instructions/security.instructions.md"]
	require.Contains(t, content, "code-review")
	require.Contains(t, content, "cloud-agent")
}

func TestCompileCopilotRule_OutputPath(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"my-rule": {
					Activation:   ast.RuleActivationAlways,
					Instructions: "Body.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	_, ok := out.Files[".github/instructions/my-rule.instructions.md"]
	require.True(t, ok, "output path must be .github/instructions/<id>.instructions.md")
}

func TestCompileCopilotRule_Activation_ManualMention_FidelityNote(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"manual-rule": {
					Activation:   ast.RuleActivationManualMention,
					Instructions: "Only when mentioned.",
				},
			},
		},
	}
	out, notes, err := r.Compile(config, "")
	require.NoError(t, err)

	// Degraded to "**" with a fidelity note
	content := out.Files[".github/instructions/manual-rule.instructions.md"]
	require.Contains(t, content, `applyTo: "**"`)
	require.NotEmpty(t, notes, "expected a fidelity note for manual-mention activation")
}
