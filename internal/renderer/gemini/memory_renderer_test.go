package gemini

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/require"
)

func TestCompileMemory_Gemini_Append(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)

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

	_, notes, err := r.Compile(config, dir)
	require.NoError(t, err)
	require.NotEmpty(t, notes, "Gemini flattening must emit a fidelity note per entry")

	data, err := os.ReadFile(filepath.Join(dir, "GEMINI.md"))
	require.NoError(t, err)
	content := string(data)
	require.Contains(t, content, "## Gemini Added Memories")
	require.Contains(t, content, `xcaffold:memory name="user-role"`)
	require.Contains(t, content, "Robert is the founder.")
	require.Contains(t, content, "xcaffold:/memory")
}

func TestCompileMemory_Gemini_MarkerIdempotent(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {
					Name:         "user-role",
					Type:         "user",
					Instructions: "First version.",
				},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.NoError(t, err)
	_, _, err = r.Compile(config, dir)
	require.NoError(t, err)

	data, _ := os.ReadFile(filepath.Join(dir, "GEMINI.md"))
	count := strings.Count(string(data), `xcaffold:memory name="user-role"`)
	require.Equal(t, 1, count, "memory block must appear exactly once after two compiles")
}

func TestCompileMemory_Gemini_ProvenanceMarker(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"arch": {
					Name:         "arch",
					Type:         "reference",
					Description:  "Architecture context.",
					Instructions: "One-way compiler.",
				},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.NoError(t, err)

	data, _ := os.ReadFile(filepath.Join(dir, "GEMINI.md"))
	content := string(data)
	require.Contains(t, content, `type="reference"`)
	require.Contains(t, content, `seeded-at="`)
}

func TestCompileMemory_Gemini_FidelityNoteCode(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"entry1": {Name: "entry1", Type: "user", Instructions: "a"},
				"entry2": {Name: "entry2", Type: "project", Instructions: "b"},
			},
		},
	}

	_, notes, err := r.Compile(config, dir)
	require.NoError(t, err)
	require.Len(t, notes, 2, "one note per memory entry")
	for _, n := range notes {
		require.Equal(t, "MEMORY_PARTIAL_FIDELITY", n.Code)
	}
}

func TestCompileMemory_Gemini_ReplacesStaleBlock(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)

	// First compile with old content
	config1 := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {Name: "user-role", Type: "user", Instructions: "Old body."},
			},
		},
	}
	_, _, err := r.Compile(config1, dir)
	require.NoError(t, err)

	// Second compile with new content
	config2 := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {Name: "user-role", Type: "user", Instructions: "New body."},
			},
		},
	}
	_, _, err = r.Compile(config2, dir)
	require.NoError(t, err)

	data, _ := os.ReadFile(filepath.Join(dir, "GEMINI.md"))
	content := string(data)
	require.NotContains(t, content, "Old body.", "stale body must be removed")
	require.Contains(t, content, "New body.", "new body must be written")
}
