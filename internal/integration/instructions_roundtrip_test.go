package integration

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/renderer"
	antigravityRenderer "github.com/saero-ai/xcaffold/internal/renderer/antigravity"
	claudeRenderer "github.com/saero-ai/xcaffold/internal/renderer/claude"
	cursorRenderer "github.com/saero-ai/xcaffold/internal/renderer/cursor"
	"github.com/stretchr/testify/require"
)

// TestInstructionsRoundTrip_ClaudeToCursor verifies that concat scopes imported from Claude
// are pre-flattened when compiled to a closest-wins target (Cursor).
func TestInstructionsRoundTrip_ClaudeToCursor(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")

	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:         "roundtrip-test",
			Instructions: "Root context.",
			InstructionsScopes: []ast.InstructionsScope{
				{
					Path:           "packages/worker",
					Instructions:   "Worker context.",
					MergeStrategy:  "concat",
					SourceProvider: "claude",
					SourceFilename: "CLAUDE.md",
				},
			},
		},
	}

	cr := cursorRenderer.New()
	out, notes, err := renderer.Orchestrate(cr, config, "")
	require.NoError(t, err)

	// Pre-flatten: worker AGENTS.md must contain BOTH root and worker content.
	workerFile := out.Files["../packages/worker/AGENTS.md"]
	require.Contains(t, workerFile, "Root context.")
	require.Contains(t, workerFile, "Worker context.")

	// Fidelity note must be present with Level=Warning and the pre-flatten code.
	require.NotEmpty(t, notes)
	var found bool
	for _, note := range notes {
		if note.Code == renderer.CodeInstructionsClosestWinsForcedConcat {
			found = true
			require.Equal(t, renderer.LevelWarning, note.Level)
			require.Contains(t, note.Reason, "pre-flattened")
		}
	}
	require.True(t, found, "pre-flatten fidelity note must be present")
}

// TestInstructionsRoundTrip_CursorToAntigravity verifies provenance markers in flat output.
func TestInstructionsRoundTrip_CursorToAntigravity(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")

	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:         "roundtrip-test",
			Instructions: "Root context.",
			InstructionsScopes: []ast.InstructionsScope{
				{
					Path:           "packages/api",
					Instructions:   "API context.",
					MergeStrategy:  "closest-wins",
					SourceProvider: "cursor",
					SourceFilename: "AGENTS.md",
				},
			},
		},
	}

	ar := antigravityRenderer.New()
	out, _, err := renderer.Orchestrate(ar, config, "")
	require.NoError(t, err)

	// Flat output must contain provenance markers.
	var flatContent string
	for _, content := range out.Files {
		if strings.Contains(content, "xcaffold:scope") {
			flatContent = content
			break
		}
	}
	require.NotEmpty(t, flatContent, "flat output must contain provenance markers")
	require.Contains(t, flatContent, `path="packages/api"`)
	require.Contains(t, flatContent, `merge="closest-wins"`)
}

// TestInstructionsRoundTrip_FlatReimport verifies provenance markers reconstruct scopes on re-import.
// parseProvenanceMarkers lives in package main (cmd/xcaffold/import.go) and cannot be
// imported from this package. This test is skipped until parseProvenanceMarkers is exported
// to an internal package accessible from integration tests.
func TestInstructionsRoundTrip_FlatReimport(t *testing.T) {
	t.Skip("parseProvenanceMarkers is defined in package main (cmd/xcaffold/import.go) and " +
		"is not importable from the integration package; extract to an internal package to enable this test")
}

// TestInstructionsIntegration_FixtureParsesAndCompiles verifies the fixture file
// parses and compiles through the Claude renderer without error.
func TestInstructionsIntegration_FixtureParsesAndCompiles(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")

	fixturePath := filepath.Join("..", "..", "testing", "fixtures", "instructions-scopes.xcf")
	config, err := parser.ParseFile(fixturePath)
	require.NoError(t, err)

	r := claudeRenderer.New()
	_, _, err = renderer.Orchestrate(r, config, filepath.Dir(fixturePath))
	require.NoError(t, err)
}
