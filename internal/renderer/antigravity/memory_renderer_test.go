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
					Name:         "user-role",
					Type:         "user",
					Description:  "Developer role.",
					Instructions: "Robert is the founder.",
				},
			},
		},
	}

	out, notes, err := r.Compile(config, dir)
	require.NoError(t, err)
	require.Empty(t, notes, "native mapping emits no notes")
	require.Contains(t, out.Files, "knowledge/user-role.md")
	content := out.Files["knowledge/user-role.md"]
	require.Contains(t, content, "title: user-role")
	require.Contains(t, content, "type: user")
	require.Contains(t, content, "- user")
	require.Contains(t, content, "- preferences")
	require.Contains(t, content, "Robert is the founder.")
}

func TestCompileMemory_Antigravity_TypeTagDerivation(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer()

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"ref":  {Name: "ref", Type: "reference", Instructions: "x"},
				"fb":   {Name: "fb", Type: "feedback", Instructions: "y"},
				"proj": {Name: "proj", Type: "project", Instructions: "z"},
			},
		},
	}

	out, _, err := r.Compile(config, dir)
	require.NoError(t, err)
	require.Contains(t, out.Files["knowledge/ref.md"], "- reference")
	require.Contains(t, out.Files["knowledge/ref.md"], "- docs")
	require.Contains(t, out.Files["knowledge/fb.md"], "- feedback")
	require.Contains(t, out.Files["knowledge/proj.md"], "- project")
	require.Contains(t, out.Files["knowledge/proj.md"], "- context")
}

func TestCompileMemory_Antigravity_ProviderKiTagsOverride(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer()

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"custom": {
					Name:         "custom",
					Type:         "user",
					Instructions: "x",
					Targets: map[string]ast.TargetOverride{
						"antigravity": {
							Provider: map[string]any{
								"ki-tags": []interface{}{"a", "b", "c"},
							},
						},
					},
				},
			},
		},
	}

	out, _, err := r.Compile(config, dir)
	require.NoError(t, err)
	content := out.Files["knowledge/custom.md"]
	require.Contains(t, content, "- a")
	require.Contains(t, content, "- b")
	require.Contains(t, content, "- c")
	// Default user tags must NOT appear when provider override is specified.
	require.NotContains(t, content, "- preferences")
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

func TestCompileMemory_Antigravity_FallbackTag_UnknownType(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer()

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"misc": {Name: "misc", Type: "custom-unknown", Instructions: "x"},
			},
		},
	}

	out, _, err := r.Compile(config, dir)
	require.NoError(t, err)
	require.Contains(t, out.Files["knowledge/misc.md"], "- memory")
}

func TestCompileMemory_Antigravity_InvalidName_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer()

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"../escape": {Instructions: "x"},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid entry name")
}
