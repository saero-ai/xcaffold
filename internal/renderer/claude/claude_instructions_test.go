package claude_test

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/renderer/claude"
	"github.com/stretchr/testify/require"
)

func TestClaudeRenderer_ProjectInstructions_RootFile(t *testing.T) {
	r := claude.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:         "test-project",
			Instructions: "Use pnpm. PostgreSQL 16.",
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)
	require.Contains(t, out.Files, "CLAUDE.md")
	require.Contains(t, out.Files["CLAUDE.md"], "Use pnpm. PostgreSQL 16.")
}

func TestClaudeRenderer_ProjectInstructions_PerScopeFiles(t *testing.T) {
	r := claude.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:         "test-project",
			Instructions: "Root context.",
			InstructionsScopes: []ast.InstructionsScope{
				{
					Path:          "packages/worker",
					Instructions:  "Worker context.",
					MergeStrategy: "concat",
				},
				{
					Path:          "packages/api",
					Instructions:  "API context.",
					MergeStrategy: "concat",
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)
	require.Contains(t, out.Files, "CLAUDE.md")
	require.Contains(t, out.Files, "packages/worker/CLAUDE.md")
	require.Contains(t, out.Files, "packages/api/CLAUDE.md")
	require.Contains(t, out.Files["packages/worker/CLAUDE.md"], "Worker context.")
}

func TestClaudeRenderer_ProjectInstructions_ImportsAsAtLines(t *testing.T) {
	r := claude.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:                "test-project",
			Instructions:        "Root context.",
			InstructionsImports: []string{"xcf/instructions/style-guide.md"},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)
	require.Contains(t, out.Files["CLAUDE.md"], "@xcf/instructions/style-guide.md")
}

func TestClaudeRenderer_ProjectInstructions_ZeroFidelityNotes(t *testing.T) {
	r := claude.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:         "test-project",
			Instructions: "Root context.",
			InstructionsScopes: []ast.InstructionsScope{
				{Path: "packages/worker", Instructions: "Worker.", MergeStrategy: "concat"},
			},
		},
	}
	_, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)
	require.Empty(t, notes, "concat-nested class must return zero fidelity notes")
}
