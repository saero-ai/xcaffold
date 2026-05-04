package antigravity

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/require"
)

func TestCompileMemory_Antigravity_WritesKnowledgeItem(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer()

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {
					Name:        "user-role",
					Description: "Developer role.",
					Content:     "Robert is the founder.",
				},
			},
		},
	}

	out, notes, err := r.Compile(config, dir)
	require.NoError(t, err)
	require.Empty(t, notes, "native mapping emits no notes")
	require.Contains(t, out.Files, "knowledge/project_user-role.md")
	content := out.Files["knowledge/project_user-role.md"]
	require.Contains(t, content, "title: user-role")
	// type: field was removed from MemoryConfig; must not appear in output.
	require.NotContains(t, content, "type:")
	// Default tag after type removal is "memory".
	require.Contains(t, content, "- memory")
	require.Contains(t, content, "Robert is the founder.")
}

// TestCompileMemory_Antigravity_DefaultTagAfterTypeRemoval verifies that all
// entries get the generic "memory" tag now that type: has been removed.
func TestCompileMemory_Antigravity_DefaultTagAfterTypeRemoval(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer()

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"ref":  {Name: "ref", Content: "x"},
				"fb":   {Name: "fb", Content: "y"},
				"proj": {Name: "proj", Content: "z"},
			},
		},
	}

	out, _, err := r.Compile(config, dir)
	require.NoError(t, err)
	require.Contains(t, out.Files["knowledge/project_ref.md"], "- memory")
	require.Contains(t, out.Files["knowledge/project_fb.md"], "- memory")
	require.Contains(t, out.Files["knowledge/project_proj.md"], "- memory")
}

func TestCompileMemory_Antigravity_EmptyMemory_NoFiles(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer()

	config := &ast.XcaffoldConfig{}
	out, notes, err := r.Compile(config, dir)
	require.NoError(t, err)
	require.Empty(t, notes)
	require.Empty(t, out.Files)
}

// TestCompileMemory_Antigravity_DefaultTag verifies all entries get "memory" tag.
func TestCompileMemory_Antigravity_DefaultTag(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer()

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"misc": {Name: "misc", Content: "x"},
			},
		},
	}

	out, _, err := r.Compile(config, dir)
	require.NoError(t, err)
	require.Contains(t, out.Files["knowledge/project_misc.md"], "- memory")
}

func TestCompileMemory_Antigravity_InvalidName_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer()

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"../escape": {Content: "x"},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid entry name")
}
