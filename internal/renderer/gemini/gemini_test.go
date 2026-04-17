package gemini

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompile_Gemini_Target(t *testing.T) {
	r := New()
	assert.Equal(t, "gemini", r.Target())
}

func TestCompile_Gemini_OutputDir(t *testing.T) {
	r := New()
	assert.Equal(t, ".gemini", r.OutputDir())
}

func TestCompile_Gemini_EmptyConfig(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{}
	out, notes, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Empty(t, out.Files)
	assert.Empty(t, notes)
}

func TestCompile_Gemini_Render(t *testing.T) {
	r := New()
	files := map[string]string{
		"GEMINI.md":           "# Project\n",
		".gemini/rules/go.md": "Use gofmt.\n",
	}
	out := r.Render(files)
	require.NotNil(t, out)
	assert.Equal(t, files, out.Files)
}

// Task 8 rule tests.

func TestCompile_Gemini_Rules_AlwaysActivation(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"go-style": {
					Description:  "Go style guide",
					Instructions: "Use gofmt.",
					Activation:   ast.RuleActivationAlways,
				},
			},
		},
	}
	out, notes, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)

	// Rule file must exist.
	ruleContent, ok := out.Files[".gemini/rules/go-style.md"]
	assert.True(t, ok, "expected .gemini/rules/go-style.md")
	assert.Contains(t, ruleContent, "Use gofmt.")

	// GEMINI.md must contain @-import.
	geminiContent := out.Files["GEMINI.md"]
	assert.Contains(t, geminiContent, "@.gemini/rules/go-style.md")

	// No fidelity notes for always activation.
	assert.Empty(t, notes)
}

func TestCompile_Gemini_Rules_PathGlob(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"api-style": {
					Description:  "API style guide",
					Instructions: "Follow REST conventions.",
					Activation:   ast.RuleActivationPathGlob,
					Paths:        []string{"api/**"},
				},
			},
		},
	}
	out, notes, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)

	_, ok := out.Files[".gemini/rules/api-style.md"]
	assert.True(t, ok, "expected .gemini/rules/api-style.md")

	geminiContent := out.Files["GEMINI.md"]
	assert.Contains(t, geminiContent, "@.gemini/rules/api-style.md")

	// path-glob is supported — no fidelity note.
	assert.Empty(t, notes)
}

func TestCompile_Gemini_Rules_UnsupportedActivation(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"secret-rule": {
					Description:  "Manual rule",
					Instructions: "Only on demand.",
					Activation:   ast.RuleActivationManualMention,
				},
			},
		},
	}
	out, notes, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)

	// Rule still written.
	_, ok := out.Files[".gemini/rules/secret-rule.md"]
	assert.True(t, ok, "expected .gemini/rules/secret-rule.md even for unsupported activation")

	// Fidelity note must be emitted.
	require.Len(t, notes, 1)
	assert.Equal(t, renderer.CodeRuleActivationUnsupported, notes[0].Code)
	assert.Equal(t, renderer.LevelWarning, notes[0].Level)
	assert.Equal(t, "secret-rule", notes[0].Resource)
}

func TestCompile_Gemini_Rules_NoProjectInstructions(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"lint": {
					Instructions: "Run golangci-lint.",
				},
			},
		},
	}
	out, notes, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	assert.Empty(t, notes)

	// No project — no GEMINI.md content except from rules.
	geminiContent := out.Files["GEMINI.md"]
	assert.Contains(t, geminiContent, "@.gemini/rules/lint.md")
}
