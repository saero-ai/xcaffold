package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/bir"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/renderer/antigravity"
	"github.com/saero-ai/xcaffold/internal/renderer/claude"
	"github.com/saero-ai/xcaffold/internal/renderer/copilot"
	"github.com/saero-ai/xcaffold/internal/renderer/cursor"
	"github.com/saero-ai/xcaffold/internal/renderer/gemini"
	"github.com/stretchr/testify/require"
)

func TestIntegration_Memory_Claude_SeedOnce_WritesThenSkips(t *testing.T) {
	dir := t.TempDir()
	r := claude.NewMemoryRenderer(dir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {Name: "user-role", Instructions: "Robert is the founder."},
			},
		},
	}

	// First apply: file absent, must write + record a seed.
	_, notes1, err := r.Compile(config, dir)
	require.NoError(t, err)
	require.Empty(t, notes1, "first apply must not emit fidelity notes")
	require.Len(t, r.Seeds(), 1)

	// Second apply on a fresh renderer (simulate a new run): file present, must skip.
	r2 := claude.NewMemoryRenderer(dir)
	_, notes2, err := r2.Compile(config, dir)
	require.NoError(t, err)
	require.NotEmpty(t, notes2, "second apply must emit a MEMORY_SEED_SKIPPED fidelity note")
	require.Empty(t, r2.Seeds(), "second apply must not record a new seed")
}

// TestIntegration_Memory_Claude_SeedOnce_ReseedOverrides verifies that
// WithReseed(true) overwrites existing seed-once files (now the only lifecycle).
func TestIntegration_Memory_Claude_SeedOnce_ReseedOverrides(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "project_user-role.md")
	require.NoError(t, os.WriteFile(targetPath, []byte("existing content"), 0o600))

	r := claude.NewMemoryRenderer(dir).WithReseed(true)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {Name: "user-role", Instructions: "Authoritative xcf content."},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.NoError(t, err)

	data, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	require.Contains(t, string(data), "Authoritative xcf content.")
}

func TestIntegration_Memory_ImportClaudeApply_RoundTrip(t *testing.T) {
	// Source: mock Claude project memory directory.
	providerDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(providerDir, "project_user-role.md"),
		[]byte("---\ndescription: Developer role.\n---\nRobert is the founder.\n"),
		0o600,
	))

	// Step 1: import to sidecar dir.
	sidecarDir := filepath.Join(t.TempDir(), "xcf", "memory")
	summary, err := bir.ImportClaudeMemory(providerDir, bir.ImportOpts{SidecarDir: sidecarDir})
	require.NoError(t, err)
	require.Equal(t, 1, summary.Imported)

	sidecarPath := filepath.Join(sidecarDir, "user-role.md")
	require.FileExists(t, sidecarPath)

	// Step 2: apply (seed) to a fresh target directory.
	applyDir := t.TempDir()
	r := claude.NewMemoryRenderer(applyDir)

	// Read the sidecar content and use it inline for the simplified round-trip test.
	sidecarRaw, err := os.ReadFile(sidecarPath)
	require.NoError(t, err)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {
					Name:         "user-role",
					Description:  "Developer role.",
					Instructions: string(sidecarRaw),
				},
			},
		},
	}

	_, _, err = r.Compile(config, applyDir)
	require.NoError(t, err)
	require.Len(t, r.Seeds(), 1)
	require.FileExists(t, filepath.Join(applyDir, "project_user-role.md"))
}

func TestIntegration_Memory_InheritedEntry_NotSeeded(t *testing.T) {
	// Verifies that inherited memory entries are stripped before the seed pass
	// and therefore do not produce seed records or on-disk files.
	dir := t.TempDir()
	r := claude.NewMemoryRenderer(dir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				// A locally owned entry: must be seeded.
				"local-entry": {
					Name:         "local-entry",
					Instructions: "Local content owned by this project.",
				},
				// An inherited entry: must NOT be seeded.
				"inherited-entry": {
					Name:         "inherited-entry",
					Instructions: "Inherited from global base.",
					Inherited:    true,
				},
			},
		},
	}

	// Strip inherited entries as the compiler would before calling the renderer.
	config.StripInherited()

	_, _, err := r.Compile(config, dir)
	require.NoError(t, err)

	// Only the local entry must produce a seed record.
	seeds := r.Seeds()
	require.Len(t, seeds, 1, "only the local entry must be seeded")
	require.Equal(t, "local-entry", seeds[0].Name)

	// The local file must exist on disk.
	require.FileExists(t, filepath.Join(dir, "project_local-entry.md"))

	// The inherited file must NOT exist on disk.
	inheritedPath := filepath.Join(dir, "project_inherited-entry.md")
	_, statErr := os.Stat(inheritedPath)
	require.True(t, os.IsNotExist(statErr), "inherited entry must not be written to disk")
}

func TestIntegration_Memory_Gemini_AppendWithMarkers(t *testing.T) {
	dir := t.TempDir()
	r := gemini.NewMemoryRenderer(dir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {Name: "user-role", Instructions: "Robert is the founder."},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "GEMINI.md"))
	require.NoError(t, err)
	content := string(data)
	require.Contains(t, content, `xcaffold:memory name="user-role"`)
	require.Contains(t, content, "Robert is the founder.")
	// type= attribute removed from markers.
	require.NotContains(t, content, `type="`)
}

func TestIntegration_Memory_Cursor_EmitsNoNativeTargetNote(t *testing.T) {
	r := cursor.NewMemoryRenderer()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {Name: "user-role", Instructions: "test"},
			},
		},
	}
	out, notes, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	require.Empty(t, out.Files)
	require.Len(t, notes, 1)
	require.Equal(t, renderer.CodeMemoryNoNativeTarget, notes[0].Code)
}

func TestIntegration_Memory_Copilot_EmitsNoNativeTargetNote(t *testing.T) {
	r := copilot.NewMemoryRenderer()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {Name: "user-role", Instructions: "test"},
			},
		},
	}
	out, notes, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	require.Empty(t, out.Files)
	require.Len(t, notes, 1)
	require.Equal(t, renderer.CodeMemoryNoNativeTarget, notes[0].Code)
}

func TestIntegration_Memory_Antigravity_WritesKnowledgeItems(t *testing.T) {
	r := antigravity.NewMemoryRenderer()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {Name: "user-role", Description: "Developer role.", Instructions: "Robert is the founder."},
			},
		},
	}
	out, notes, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	require.Empty(t, notes)
	require.Contains(t, out.Files, "knowledge/project_user-role.md")
	content := out.Files["knowledge/project_user-role.md"]
	// type: removed from MemoryConfig; all entries get generic "memory" tag.
	require.NotContains(t, content, "type:")
	require.Contains(t, content, "- memory")
	require.Contains(t, content, "Robert is the founder.")
}

func TestIntegration_Memory_Gemini_AppendsToGeminiMD(t *testing.T) {
	dir := t.TempDir()
	r := gemini.NewMemoryRenderer(dir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {
					Name:         "user-role",
					Description:  "Developer role.",
					Instructions: "Robert is the founder.",
				},
			},
		},
	}

	_, notes, err := r.Compile(config, dir)
	require.NoError(t, err)

	// Gemini emits MEMORY_PARTIAL_FIDELITY notes (one per entry).
	require.NotEmpty(t, notes)
	require.Equal(t, renderer.CodeMemoryPartialFidelity, notes[0].Code)

	data, err := os.ReadFile(filepath.Join(dir, "GEMINI.md"))
	require.NoError(t, err)
	content := string(data)

	// The "## Gemini Added Memories" section must exist.
	require.Contains(t, content, "## Gemini Added Memories")

	// The seeded entry must be present under the section.
	require.Contains(t, content, `xcaffold:memory name="user-role"`)
	require.Contains(t, content, "Robert is the founder.")
}

func TestIntegration_Memory_GeminiImportExtract_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	r := gemini.NewMemoryRenderer(dir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {Name: "user-role", Description: "Dev role.", Instructions: "Body."},
			},
		},
	}
	_, _, err := r.Compile(config, dir)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "GEMINI.md"))
	require.NoError(t, err)

	blocks, err := bir.ExtractGeminiMemoryBlocks(string(data))
	require.NoError(t, err)
	require.Len(t, blocks, 1)
	require.Equal(t, "user-role", blocks[0].Key)
	// Type field removed from MemoryConfig; blocks no longer carry it.
	require.Equal(t, "", blocks[0].Type)
	require.Contains(t, blocks[0].Body, "Body.")
}
