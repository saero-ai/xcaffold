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
				"user-role": {Name: "user-role", Type: "user", Instructions: "Robert is the founder."},
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

func TestIntegration_Memory_Claude_Tracked_DetectsDrift(t *testing.T) {
	dir := t.TempDir()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"arch": {Name: "arch", Type: "reference", Lifecycle: "tracked", Instructions: "Original xcf content."},
			},
		},
	}

	// First apply: seed the file.
	r1 := claude.NewMemoryRenderer(dir)
	_, _, err := r1.Compile(config, dir)
	require.NoError(t, err)
	require.Len(t, r1.Seeds(), 1)
	priorHash := r1.Seeds()[0].Hash

	// Simulate agent modification.
	targetPath := filepath.Join(dir, "arch.md")
	require.NoError(t, os.WriteFile(targetPath, []byte("agent modified this"), 0o600))

	// Second apply with prior hash: must detect drift.
	r2 := claude.NewMemoryRenderer(dir)
	_, _, err = r2.CompileWithPriorSeeds(config, dir, map[string]string{"arch": priorHash})
	require.Error(t, err)
	require.Contains(t, err.Error(), "memory drift detected")
}

func TestIntegration_Memory_Claude_Reseed_OverridesDrift(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "user-role.md")
	require.NoError(t, os.WriteFile(targetPath, []byte("agent modified this"), 0o600))

	r := claude.NewMemoryRenderer(dir).WithReseed(true)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {Name: "user-role", Type: "user", Instructions: "Authoritative xcf content."},
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
		filepath.Join(providerDir, "user-role.md"),
		[]byte("---\ntype: user\ndescription: Developer role.\n---\nRobert is the founder.\n"),
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
					Type:         "user",
					Description:  "Developer role.",
					Instructions: string(sidecarRaw),
				},
			},
		},
	}

	_, _, err = r.Compile(config, applyDir)
	require.NoError(t, err)
	require.Len(t, r.Seeds(), 1)
	require.FileExists(t, filepath.Join(applyDir, "user-role.md"))
}

func TestIntegration_Memory_Gemini_AppendWithMarkers(t *testing.T) {
	dir := t.TempDir()
	r := gemini.NewMemoryRenderer(dir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {Name: "user-role", Type: "user", Instructions: "Robert is the founder."},
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
}

func TestIntegration_Memory_Cursor_EmitsNoNativeTargetNote(t *testing.T) {
	r := cursor.NewMemoryRenderer()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {Name: "user-role", Type: "user", Instructions: "test"},
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
				"user-role": {Name: "user-role", Type: "user", Instructions: "test"},
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
				"user-role": {Name: "user-role", Type: "user", Description: "Developer role.", Instructions: "Robert is the founder."},
			},
		},
	}
	out, notes, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	require.Empty(t, notes)
	require.Contains(t, out.Files, "knowledge/user-role.md")
	content := out.Files["knowledge/user-role.md"]
	require.Contains(t, content, "type: user")
	require.Contains(t, content, "- user")
	require.Contains(t, content, "Robert is the founder.")
}

func TestIntegration_Memory_GeminiImportExtract_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	r := gemini.NewMemoryRenderer(dir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {Name: "user-role", Type: "user", Description: "Dev role.", Instructions: "Body."},
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
	require.Equal(t, "user", blocks[0].Type)
	require.Contains(t, blocks[0].Body, "Body.")
}
