package renderer_test

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/renderer/claude"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOrchestrate_FallbackToCompile verifies that when a renderer only implements
// TargetRenderer (not ResourceRenderer), Orchestrate falls back to Compile().
func TestOrchestrate_FallbackToCompile(t *testing.T) {
	r := claude.New()
	config := &ast.XcaffoldConfig{
		Version: "1",
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"tester": {
					Name:         "Tester",
					Description:  "A test agent",
					Instructions: "Do the test.",
				},
			},
		},
	}

	out, notes, err := renderer.Orchestrate(r, config, t.TempDir())
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.NotEmpty(t, out.Files, "fallback Compile() should produce at least one file")
	_ = notes
}

// TestOrchestrate_FallbackToCompile_EmptyConfig verifies the fallback path
// returns a non-nil output even when config has no resources.
func TestOrchestrate_FallbackToCompile_EmptyConfig(t *testing.T) {
	r := claude.New()
	config := &ast.XcaffoldConfig{
		Version: "1",
	}

	out, notes, err := renderer.Orchestrate(r, config, t.TempDir())
	require.NoError(t, err)
	require.NotNil(t, out)
	_ = notes
}

// TestOrchestrate_SignatureCompiles verifies the function signature matches
// the expected contract: (TargetRenderer, *ast.XcaffoldConfig, string) → (*output.Output, []FidelityNote, error).
// This is a compile-time check encoded as a runtime smoke test.
func TestOrchestrate_SignatureCompiles(t *testing.T) {
	r := claude.New()
	config := &ast.XcaffoldConfig{Version: "1"}

	out, _, err := renderer.Orchestrate(r, config, ".")
	require.NoError(t, err)
	assert.NotNil(t, out)
}

// mockLegacyRenderer implements only TargetRenderer, not ResourceRenderer.
// Used to verify the orchestrator dispatches to Compile() for legacy renderers.
type mockLegacyRenderer struct {
	compileCallCount int
	returnFiles      map[string]string
}

func (m *mockLegacyRenderer) Target() string    { return "mock" }
func (m *mockLegacyRenderer) OutputDir() string { return ".mock" }
func (m *mockLegacyRenderer) Render(files map[string]string) *output.Output {
	return &output.Output{Files: files}
}
func (m *mockLegacyRenderer) Compile(_ *ast.XcaffoldConfig, _ string) (*output.Output, []renderer.FidelityNote, error) {
	m.compileCallCount++
	files := m.returnFiles
	if files == nil {
		files = map[string]string{"mock.txt": "mock content"}
	}
	return &output.Output{Files: files}, nil, nil
}

// TestOrchestrate_LegacyRenderer_CallsCompile verifies that Orchestrate dispatches
// to Compile() exactly once when the renderer does not implement ResourceRenderer.
func TestOrchestrate_LegacyRenderer_CallsCompile(t *testing.T) {
	m := &mockLegacyRenderer{}
	config := &ast.XcaffoldConfig{Version: "1"}

	out, _, err := renderer.Orchestrate(m, config, ".")
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, 1, m.compileCallCount, "Compile should be called exactly once for legacy renderers")
	assert.Equal(t, "mock content", out.Files["mock.txt"])
}
