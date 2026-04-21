package cursor_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/renderer/cursor"
	"github.com/stretchr/testify/require"
)

// TestCursorRenderer_ProjectInstructions_PreFlattensConcat verifies that a
// concat-tagged scope is pre-flattened into the child AGENTS.md (root + child).
func TestCursorRenderer_ProjectInstructions_PreFlattensConcat(t *testing.T) {
	r := cursor.New()
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
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)
	// Pre-flattening: worker AGENTS.md must contain root + worker content.
	worker, ok := out.Files["packages/worker/AGENTS.md"]
	require.True(t, ok, "packages/worker/AGENTS.md must be present in output")
	require.Contains(t, worker, "Root context.")
	require.Contains(t, worker, "Worker context.")
}

// TestCursorRenderer_ProjectInstructions_FidelityNoteForPreFlatten verifies that
// a concat-tagged scope emits a INSTRUCTIONS_CLOSEST_WINS_FORCED_CONCAT warning note.
func TestCursorRenderer_ProjectInstructions_FidelityNoteForPreFlatten(t *testing.T) {
	r := cursor.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:         "test-project",
			Instructions: "Root.",
			InstructionsScopes: []ast.InstructionsScope{
				{Path: "packages/worker", Instructions: "Worker.", MergeStrategy: "concat"},
			},
		},
	}
	_, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)
	require.NotEmpty(t, notes)

	var found renderer.FidelityNote
	var foundIt bool
	for _, n := range notes {
		if n.Code == renderer.CodeInstructionsClosestWinsForcedConcat {
			found = n
			foundIt = true
			break
		}
	}
	require.True(t, foundIt, "expected INSTRUCTIONS_CLOSEST_WINS_FORCED_CONCAT note")
	require.Equal(t, renderer.LevelWarning, found.Level)
	require.Contains(t, found.Reason, "pre-flattened")
}

// TestCursorRenderer_ProjectInstructions_InlinesImports verifies that InstructionsImports
// are inlined into AGENTS.md (no @-import lines) and that an INSTRUCTIONS_IMPORT_INLINED
// info note is emitted.
func TestCursorRenderer_ProjectInstructions_InlinesImports(t *testing.T) {
	// Create a temp dir with an import file.
	dir := t.TempDir()
	importContent := "# Shared guidelines\nAlways be helpful."
	require.NoError(t, os.WriteFile(filepath.Join(dir, "shared.md"), []byte(importContent), 0600))

	r := cursor.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:                "test-project",
			Instructions:        "Root instructions.",
			InstructionsImports: []string{"shared.md"},
		},
	}

	out, notes, err := renderer.Orchestrate(r, config, dir)
	require.NoError(t, err)

	rootMD, ok := out.Files["AGENTS.md"]
	require.True(t, ok, "AGENTS.md must be present in output")

	// Must NOT contain raw @-import syntax — Cursor does not support it.
	require.False(t, strings.Contains(rootMD, "@shared.md"), "AGENTS.md must not contain @-import lines")

	// Must contain the inlined content.
	require.Contains(t, rootMD, "Shared guidelines")

	// Must emit an INSTRUCTIONS_IMPORT_INLINED info note.
	var found renderer.FidelityNote
	var foundIt bool
	for _, n := range notes {
		if n.Code == renderer.CodeInstructionsImportInlined {
			found = n
			foundIt = true
			break
		}
	}
	require.True(t, foundIt, "expected INSTRUCTIONS_IMPORT_INLINED note")
	require.Equal(t, renderer.LevelInfo, found.Level)
}
