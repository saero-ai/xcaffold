package renderer_test

import (
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRenderer is a minimal TargetRenderer for testing.
type mockRenderer struct {
	target    string
	outputDir string
}

func (m *mockRenderer) Target() string    { return m.target }
func (m *mockRenderer) OutputDir() string { return m.outputDir }
func (m *mockRenderer) Render(files map[string]string) *output.Output {
	return &output.Output{Files: files}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := renderer.NewRegistry()
	mock := &mockRenderer{target: "claude", outputDir: ".claude"}

	reg.Register(mock)

	got, err := reg.Get("claude")
	require.NoError(t, err)
	assert.Equal(t, "claude", got.Target())
	assert.Equal(t, ".claude", got.OutputDir())
}

func TestRegistry_GetUnknownTarget(t *testing.T) {
	reg := renderer.NewRegistry()

	_, err := reg.Get("unknown-target")
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "unknown-target"),
		"error message should contain the target name, got: %s", err.Error())
}

func TestRegistry_TargetsSorted(t *testing.T) {
	reg := renderer.NewRegistry()
	reg.Register(&mockRenderer{target: "gemini", outputDir: ".gemini"})
	reg.Register(&mockRenderer{target: "claude", outputDir: ".claude"})
	reg.Register(&mockRenderer{target: "cursor", outputDir: ".cursor"})

	targets := reg.Targets()
	assert.Equal(t, []string{"claude", "cursor", "gemini"}, targets)
}

func TestRegistry_RenderPassthrough(t *testing.T) {
	reg := renderer.NewRegistry()
	reg.Register(&mockRenderer{target: "claude", outputDir: ".claude"})

	tr, err := reg.Get("claude")
	require.NoError(t, err)

	input := map[string]string{"CLAUDE.md": "hello"}
	out := tr.Render(input)
	assert.Equal(t, input, out.Files)
}
