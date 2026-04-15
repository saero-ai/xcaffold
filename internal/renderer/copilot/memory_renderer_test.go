package copilot

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/stretchr/testify/require"
)

func TestCompileMemory_Copilot_NoOp(t *testing.T) {
	r := NewMemoryRenderer()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {Name: "user-role", Type: "user", Instructions: "test"},
			},
		},
	}

	output, notes, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	require.NotNil(t, output)
	require.Empty(t, output.Files, "no files must be written for copilot memory")
	require.Len(t, notes, 1)
	require.Equal(t, renderer.CodeMemoryNoNativeTarget, notes[0].Code)
	require.Contains(t, notes[0].Reason, "Copilot")
	require.Contains(t, notes[0].Mitigation, ".github/copilot-instructions.md")
}

func TestCompileMemory_Copilot_MultipleEntries_OneNotePerEntry(t *testing.T) {
	r := NewMemoryRenderer()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"a": {Name: "a", Type: "user", Instructions: "a"},
				"b": {Name: "b", Type: "project", Instructions: "b"},
			},
		},
	}
	_, notes, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	require.Len(t, notes, 2)
}

func TestCompileMemory_Copilot_EmptyMemory_NoNotes(t *testing.T) {
	r := NewMemoryRenderer()
	config := &ast.XcaffoldConfig{}
	_, notes, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	require.Empty(t, notes)
}
