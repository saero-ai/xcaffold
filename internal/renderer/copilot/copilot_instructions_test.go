package copilot_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/parser"
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
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)
	root := out.Files["copilot-instructions.md"]
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
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)
	for path := range out.Files {
		require.NotEqual(t, "AGENTS.md", path, "copilot renderer must not emit AGENTS.md")
		require.False(t, strings.HasSuffix(path, "/AGENTS.md"),
			"copilot renderer must not emit nested AGENTS.md; got %s", path)
	}
}

// TestCopilotRenderer_FlatMode_ExplicitFlag verifies that setting
// target-options.copilot.instructions-mode: flat produces flat singleton output
// (identical to the default).
func TestCopilotRenderer_FlatMode_ExplicitFlag(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:         "test",
			Instructions: "Root.",
			TargetOptions: map[string]ast.TargetOverride{
				"copilot": {InstructionsMode: "flat"},
			},
			InstructionsScopes: []ast.InstructionsScope{
				{Path: "packages/worker", Instructions: "Worker.", MergeStrategy: "flat"},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)
	_, hasCopilotInstructions := out.Files["copilot-instructions.md"]
	require.True(t, hasCopilotInstructions, "flat mode must emit .github/copilot-instructions.md")
	for path := range out.Files {
		require.False(t, strings.HasSuffix(path, "/AGENTS.md"),
			"flat mode must not emit AGENTS.md; got %s", path)
	}
}

// TestCopilotRenderer_NestedMode_EmitsNestedDirs verifies that setting
// target-options.copilot.instructions-mode: nested produces per-directory AGENTS.md
// files instead of the flat singleton.
func TestCopilotRenderer_NestedMode_EmitsNestedDirs(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:         "test",
			Instructions: "Root.",
			TargetOptions: map[string]ast.TargetOverride{
				"copilot": {InstructionsMode: "nested"},
			},
			InstructionsScopes: []ast.InstructionsScope{
				{Path: "packages/worker", Instructions: "Worker.", MergeStrategy: "closest-wins"},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)
	// Nested mode must NOT emit flat singleton.
	_, hasFlatFile := out.Files["copilot-instructions.md"]
	require.False(t, hasFlatFile, "nested mode must not emit .github/copilot-instructions.md")
	// Nested mode must emit root AGENTS.md and per-scope AGENTS.md.
	_, hasRoot := out.Files["AGENTS.md"]
	require.True(t, hasRoot, "nested mode must emit root AGENTS.md")
	_, hasScope := out.Files["packages/worker/AGENTS.md"]
	require.True(t, hasScope, "nested mode must emit packages/worker/AGENTS.md")
}

// TestCopilotParser_InstructionsModeInvalidValue verifies that an unknown
// instructions-mode value on a project.target-options.copilot block is a parse error.
func TestCopilotParser_InstructionsModeInvalidValue(t *testing.T) {
	yml := `
kind: project
version: "1.0"
name: test
instructions: "Root."
target-options:
  copilot:
    instructions-mode: sideways
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "project.xcf")
	require.NoError(t, os.WriteFile(path, []byte(yml), 0o600))
	_, err := parser.ParseFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "instructions-mode")
}

func TestCopilotRenderer_ProjectInstructions_XCFExtractsBody(t *testing.T) {
	dir := t.TempDir()
	xcfContent := `kind: project
instructions: |
  This is Copilot root extracted from XCF.`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "root.xcf"), []byte(xcfContent), 0600))

	r := copilot.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:             "test-project",
			InstructionsFile: "root.xcf",
		},
	}

	out, _, err := renderer.Orchestrate(r, config, dir)
	require.NoError(t, err)

	rootMD, ok := out.Files["copilot-instructions.md"]
	require.True(t, ok, "copilot-instructions.md must be present in output")
	require.Contains(t, rootMD, "This is Copilot root extracted from XCF.", "Must extract body")
	require.NotContains(t, rootMD, "kind: project", "Must strip XCF frontmatter")
}

func TestRenderProjectInstructions_Copilot_ScopeFiles_WithApplyTo(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:         "test-project",
			Instructions: "Root",
			InstructionsScopes: []ast.InstructionsScope{
				{
					Path:         "dummy_platform",
					Instructions: "Dummy scope content.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	scopeFile, ok := out.Files["instructions/dummy_platform.instructions.md"]
	require.True(t, ok, "instructions/dummy_platform.instructions.md must be present")
	require.Contains(t, scopeFile, "applyTo: \"dummy_platform/**\"", "Must contain applyTo frontmatter")
	require.Contains(t, scopeFile, "Dummy scope content.", "Must contain extracted content")
}
