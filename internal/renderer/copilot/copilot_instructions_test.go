package copilot_test

import (
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/renderer/copilot"
	"github.com/stretchr/testify/require"
)

func TestCopilotRenderer_ProjectInstructions_FlatSingleton(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:         "test-project",
			Instructions: "Root content.",
			InstructionsScopes: []ast.InstructionsScope{
				{Path: "packages/worker", Instructions: "Worker content.", MergeStrategy: "concat"},
			},
		},
	}
	out, _, err := r.Compile(config, "")
	require.NoError(t, err)
	root := out.Files[".github/copilot-instructions.md"]
	require.NotEmpty(t, root)
	require.Contains(t, root, "Root content.")
	require.Contains(t, root, `<!-- xcaffold:scope path="packages/worker"`)
	require.Contains(t, root, "Worker content.")
	require.Contains(t, root, `<!-- xcaffold:/scope -->`)
}

func TestCopilotRenderer_ProjectInstructions_FidelityNote(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:         "test",
			Instructions: "Root.",
			InstructionsScopes: []ast.InstructionsScope{
				{Path: "packages/worker", Instructions: "Worker.", MergeStrategy: "concat"},
			},
		},
	}
	_, notes, err := r.Compile(config, "")
	require.NoError(t, err)
	require.NotEmpty(t, notes)
	var found *renderer.FidelityNote
	for i := range notes {
		if notes[i].Code == "INSTRUCTIONS_FLATTENED" {
			found = &notes[i]
			break
		}
	}
	require.NotNil(t, found, "expected INSTRUCTIONS_FLATTENED note")
	require.Equal(t, renderer.LevelInfo, found.Level)
}

func TestCopilotRenderer_ProjectInstructions_NoNestedOutput(t *testing.T) {
	// Copilot flat-mode renderer must NEVER emit nested AGENTS.md.
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:         "test",
			Instructions: "Root.",
			InstructionsScopes: []ast.InstructionsScope{
				{Path: "packages/worker", Instructions: "Worker.", MergeStrategy: "concat"},
			},
		},
	}
	out, _, err := r.Compile(config, "")
	require.NoError(t, err)
	for path := range out.Files {
		require.NotEqual(t, "AGENTS.md", path, "copilot renderer must not emit AGENTS.md")
		require.False(t, strings.HasSuffix(path, "/AGENTS.md"),
			"copilot renderer must not emit nested AGENTS.md; got %s", path)
	}
}
