package gemini

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/stretchr/testify/require"
)

func TestCompileMemory_Gemini_Append(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)

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
	// type= attribute must not appear after type field removal.
	require.NotContains(t, content, `type="`)
}

func TestCompileMemory_Gemini_MarkerIdempotent(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {
					Name:    "user-role",
					Content: "First version.",
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

	headerCount := strings.Count(string(data), "## Gemini Added Memories")
	require.Equal(t, 1, headerCount, "section header must appear exactly once after two compiles")
}

func TestCompileMemory_Gemini_ProvenanceMarker(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"arch": {
					Name:        "arch",
					Description: "Architecture context.",
					Content:     "One-way compiler.",
				},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.NoError(t, err)

	data, _ := os.ReadFile(filepath.Join(dir, "GEMINI.md"))
	content := string(data)
	// type= attribute removed; only name= and seeded-at= remain.
	require.Contains(t, content, `seeded-at="`)
	require.NotContains(t, content, `type="`)
}

func TestCompileMemory_Gemini_FidelityNoteCode(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"entry1": {Name: "entry1", Content: "a"},
				"entry2": {Name: "entry2", Content: "b"},
			},
		},
	}

	_, notes, err := r.Compile(config, dir)
	require.NoError(t, err)
	require.Len(t, notes, 2, "one note per memory entry")
	for _, n := range notes {
		require.Equal(t, renderer.CodeMemoryPartialFidelity, n.Code)
	}
}

func TestCompileMemory_Gemini_EmptyBody_EmitsBothNotes(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"empty-entry": {
					Name: "empty-entry",
					// No instructions or instructions-file — body will be empty.
				},
			},
		},
	}

	_, notes, err := r.Compile(config, dir)
	require.NoError(t, err)
	require.Len(t, notes, 2, "empty-body entry must emit exactly 2 notes")

	codes := make([]string, 0, 2)
	for _, n := range notes {
		codes = append(codes, n.Code)
	}
	require.Contains(t, codes, "MEMORY_PARTIAL_FIDELITY", "must emit MEMORY_PARTIAL_FIDELITY info note")
	require.Contains(t, codes, "MEMORY_BODY_EMPTY", "must emit MEMORY_BODY_EMPTY warning note")

	for _, n := range notes {
		if n.Code == renderer.CodeMemoryPartialFidelity {
			require.Equal(t, renderer.LevelInfo, n.Level)
		}
		if n.Code == renderer.CodeMemoryBodyEmpty {
			require.Equal(t, renderer.LevelWarning, n.Level)
		}
	}
}

func TestCompileMemory_Gemini_ReplacesStaleBlock(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)

	// First compile with old content
	config1 := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {Name: "user-role", Content: "Old body."},
			},
		},
	}
	_, _, err := r.Compile(config1, dir)
	require.NoError(t, err)

	// Second compile with new content
	config2 := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {Name: "user-role", Content: "New body."},
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
