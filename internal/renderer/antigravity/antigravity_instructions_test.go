package antigravity_test

import (
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/renderer/antigravity"
	"github.com/stretchr/testify/require"
)

func TestAntigravityRenderer_ProjectInstructions_FlatSingleton(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:         "test-project",
			Instructions: "Root content.",
			InstructionsScopes: []ast.InstructionsScope{
				{Path: "packages/worker", Instructions: "Worker content.", MergeStrategy: "concat"},
				{Path: "packages/api", Instructions: "API content.", MergeStrategy: "closest-wins"},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)
	// Root rules file must contain root + all scopes with provenance markers.
	root := out.RootFiles[antigravity.ProjectContextFile]
	require.Contains(t, root, "Root content.")
	require.Contains(t, root, `<!-- xcaffold:scope path="packages/worker"`)
	require.Contains(t, root, "Worker content.")
	require.Contains(t, root, `<!-- xcaffold:/scope -->`)
	require.Contains(t, root, `<!-- xcaffold:scope path="packages/api"`)
	require.Contains(t, root, "API content.")
}

func TestAntigravityRenderer_ProjectInstructions_ScopeOrderDepthAscAlpha(t *testing.T) {
	// Scopes must appear in depth-ascending then alphabetical order.
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:         "test",
			Instructions: "Root.",
			InstructionsScopes: []ast.InstructionsScope{
				{Path: "packages/worker/src", Instructions: "Deep.", MergeStrategy: "flat"},
				{Path: "packages/api", Instructions: "Shallow.", MergeStrategy: "flat"},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)
	root := out.RootFiles[antigravity.ProjectContextFile]
	apiIdx := strings.Index(root, "packages/api")
	srcIdx := strings.Index(root, "packages/worker/src")
	require.Less(t, apiIdx, srcIdx, "shallower scope must appear before deeper scope")
}

func TestAntigravityRenderer_ProjectInstructions_FidelityNotePerScope(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:         "test",
			Instructions: "Root.",
			InstructionsScopes: []ast.InstructionsScope{
				{Path: "packages/worker", Instructions: "Worker.", MergeStrategy: "concat"},
			},
		},
	}
	_, notes, err := renderer.Orchestrate(r, config, "")
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
