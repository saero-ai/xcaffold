package gemini

import (
	"os"
	"path/filepath"
	"strings"
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

func TestCompile_Gemini_FullConfig_InstructionsAndRules(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "style.md"), []byte("Style guide content."), 0o644)
	require.NoError(t, err)

	r := New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:                "integration-test",
			Instructions:        "Root project instructions.",
			InstructionsImports: []string{"./docs/contributing.md"},
			InstructionsScopes: []ast.InstructionsScope{
				{
					Path:         "packages/api",
					Instructions: "API scope instructions.",
				},
			},
		},
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"code-style": {
					Description:      "Code style.",
					InstructionsFile: "style.md",
				},
				"testing": {
					Description:  "Testing rules.",
					Instructions: "Always write tests.",
				},
			},
		},
	}
	out, notes, err := r.Compile(config, tmpDir)
	require.NoError(t, err)
	assert.Empty(t, notes, "no fidelity notes expected for supported activations")

	root := out.Files["GEMINI.md"]
	assert.Contains(t, root, "Root project instructions.")
	assert.Contains(t, root, "@./docs/contributing.md")
	assert.Contains(t, root, "@.gemini/rules/code-style.md")
	assert.Contains(t, root, "@.gemini/rules/testing.md")

	assert.Contains(t, out.Files, "packages/api/GEMINI.md")
	assert.Contains(t, out.Files["packages/api/GEMINI.md"], "API scope instructions.")

	assert.Contains(t, out.Files[".gemini/rules/code-style.md"], "Style guide content.")
	assert.Contains(t, out.Files[".gemini/rules/testing.md"], "Always write tests.")
}

func TestCompile_Gemini_PathTraversal(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"../evil": {
					Description:  "Malicious rule.",
					Instructions: "Bad content.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, "/tmp/test")
	require.NoError(t, err)
	for path := range out.Files {
		assert.NotContains(t, path, "..", "output path must not contain traversal sequences")
	}
}

func TestCompile_Gemini_Workflows_LoweredToRulePlusSkill(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Workflows: map[string]ast.WorkflowConfig{
				"deploy": {
					Name:        "deploy",
					Description: "Deploy to production.",
					Steps: []ast.WorkflowStep{
						{Name: "build", Instructions: "Run go build."},
						{Name: "test", Instructions: "Run go test."},
					},
				},
			},
		},
	}
	out, notes, err := r.Compile(config, "/tmp/test")
	require.NoError(t, err)

	// Should have lowered workflow to rules + skills
	hasRule := false
	hasSkill := false
	for path := range out.Files {
		if strings.Contains(path, ".gemini/rules/") {
			hasRule = true
		}
		if strings.Contains(path, ".gemini/skills/") {
			hasSkill = true
		}
	}
	assert.True(t, hasRule, "lowered workflow should produce at least one rule file")
	assert.True(t, hasSkill, "lowered workflow should produce at least one skill file")

	// Should have workflow lowering fidelity note
	hasLoweringNote := false
	for _, n := range notes {
		if n.Code == renderer.CodeWorkflowLoweredToRulePlusSkill {
			hasLoweringNote = true
		}
	}
	assert.True(t, hasLoweringNote, "expected CodeWorkflowLoweredToRulePlusSkill fidelity note")
}
