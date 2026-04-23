package gemini

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompile_Gemini_Instructions_RootOnly(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:         "my-project",
			Instructions: "Build with go build.",
		},
	}
	out, notes, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	assert.Empty(t, notes)

	content, ok := out.Files["../GEMINI.md"]
	require.True(t, ok, "expected ../GEMINI.md to be produced")
	assert.Contains(t, content, "Build with go build.")
}

func TestCompile_Gemini_Instructions_WithImports(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:                "imported-project",
			Instructions:        "Main instructions.",
			InstructionsImports: []string{"./rules/style.md"},
		},
	}
	out, notes, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	assert.Empty(t, notes)

	content, ok := out.Files["../GEMINI.md"]
	require.True(t, ok, "expected ../GEMINI.md")
	assert.Contains(t, content, "Main instructions.")
	assert.Contains(t, content, "@./rules/style.md")
}

func TestCompile_Gemini_Instructions_WithScopes(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:         "scoped-project",
			Instructions: "Root instructions.",
			InstructionsScopes: []ast.InstructionsScope{
				{
					Path:         "packages/api",
					Instructions: "API-specific instructions.",
				},
			},
		},
	}
	out, notes, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	assert.Empty(t, notes)

	// Root GEMINI.md.
	rootContent, ok := out.Files["../GEMINI.md"]
	require.True(t, ok, "expected ../GEMINI.md")
	assert.Contains(t, rootContent, "Root instructions.")

	// Scope GEMINI.md.
	scopeContent, ok := out.Files["../packages/api/GEMINI.md"]
	require.True(t, ok, "expected ../packages/api/GEMINI.md")
	assert.Contains(t, scopeContent, "API-specific instructions.")
}

func TestCompile_Gemini_Instructions_FromFile(t *testing.T) {
	tmpDir := t.TempDir()
	instrFile := filepath.Join(tmpDir, "instructions.md")
	require.NoError(t, os.WriteFile(instrFile, []byte("File-sourced instructions.\n"), 0o600))

	r := New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:             "file-project",
			InstructionsFile: "instructions.md",
		},
	}
	out, notes, err := r.Compile(config, tmpDir)
	require.NoError(t, err)
	assert.Empty(t, notes)

	content, ok := out.Files["../GEMINI.md"]
	require.True(t, ok, "expected ../GEMINI.md")
	assert.Contains(t, content, "File-sourced instructions.")
}

func TestCompile_Gemini_Instructions_ScopeVariant(t *testing.T) {
	tmpDir := t.TempDir()
	variantFile := filepath.Join(tmpDir, "gemini-specific.md")
	require.NoError(t, os.WriteFile(variantFile, []byte("Gemini-specific scope content.\n"), 0o600))

	r := New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:         "variant-project",
			Instructions: "default content",
			InstructionsScopes: []ast.InstructionsScope{
				{
					Path:         "packages/backend",
					Instructions: "default scope instructions",
					Variants: map[string]ast.InstructionsVariant{
						"gemini": {
							InstructionsFile: "gemini-specific.md",
						},
					},
				},
			},
		},
	}
	out, _, err := r.Compile(config, tmpDir)
	require.NoError(t, err)

	scopeContent, ok := out.Files["../packages/backend/GEMINI.md"]
	require.True(t, ok, "expected ../packages/backend/GEMINI.md to be produced")
	assert.Contains(t, scopeContent, "Gemini-specific scope content.")
	assert.NotContains(t, scopeContent, "default scope instructions")
}

func TestCompile_Gemini_Instructions_NoProject(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{}
	out, notes, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	assert.Empty(t, notes)

	_, ok := out.Files["../GEMINI.md"]
	assert.False(t, ok, "no project config should produce no GEMINI.md")
}
