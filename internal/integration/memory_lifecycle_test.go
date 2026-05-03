package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/compiler"
	"github.com/saero-ai/xcaffold/internal/renderer"
	antigravity "github.com/saero-ai/xcaffold/providers/antigravity"
	"github.com/saero-ai/xcaffold/providers/claude"
	copilot "github.com/saero-ai/xcaffold/providers/copilot"
	"github.com/saero-ai/xcaffold/providers/cursor"
	gemini "github.com/saero-ai/xcaffold/providers/gemini"
	"github.com/stretchr/testify/require"
)

func TestIntegration_Memory_Claude_AlwaysOverwrites(t *testing.T) {
	dir := t.TempDir()

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {Name: "user-role", Description: "User role", Content: "Robert is the founder."},
			},
		},
	}

	// First apply: file absent, must write + record a seed.
	r := claude.NewMemoryRenderer(dir)
	_, notes1, err := r.Compile(config, dir)
	require.NoError(t, err)
	require.Empty(t, notes1, "first apply must not emit fidelity notes")
	require.Len(t, r.Seeds(), 1)

	// Second apply on a fresh renderer: must still write and record a seed.
	r2 := claude.NewMemoryRenderer(dir)
	_, _, err = r2.Compile(config, dir)
	require.NoError(t, err)
	require.Len(t, r2.Seeds(), 1, "apply always overwrites, must record seed")
}

func TestIntegration_Memory_MultipleEntries_AllSeeded(t *testing.T) {
	// Verifies that all memory entries are seeded. Memory is convention-based
	// (filesystem scan), so there is no inherited-vs-local distinction.
	dir := t.TempDir()
	r := claude.NewMemoryRenderer(dir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"entry-a": {
					Name:        "entry-a",
					Description: "Entry A",
					Content:     "Content A.",
				},
				"entry-b": {
					Name:        "entry-b",
					Description: "Entry B",
					Content:     "Content B.",
				},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.NoError(t, err)

	// Both entries have no AgentRef so they land in "default" — one seed record
	// per agent group, not per entry.
	seeds := r.Seeds()
	require.Len(t, seeds, 1, "one seed per agent group (default)")

	// Both entries must be present in the per-agent MEMORY.md index.
	memPath := filepath.Join(dir, "default", "MEMORY.md")
	require.FileExists(t, memPath)
	data, err := os.ReadFile(memPath)
	require.NoError(t, err)
	require.Contains(t, string(data), "[entry-a]")
	require.Contains(t, string(data), "[entry-b]")
}

func TestIntegration_Memory_Gemini_AppendWithMarkers(t *testing.T) {
	dir := t.TempDir()
	r := gemini.NewMemoryRenderer(dir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {Name: "user-role", Content: "Robert is the founder."},
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
				"user-role": {Name: "user-role", Content: "test"},
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
				"user-role": {Name: "user-role", Content: "test"},
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
				"user-role": {Name: "user-role", Description: "Developer role.", Content: "Robert is the founder."},
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
					Name:        "user-role",
					Description: "Developer role.",
					Content:     "Robert is the founder.",
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

func TestIntegration_Memory_ConventionDiscovery_EndToEnd(t *testing.T) {
	// Set up convention-based memory files on disk.
	baseDir := t.TempDir()
	memDir := filepath.Join(baseDir, "xcf", "agents", "dev", "memory")
	require.NoError(t, os.MkdirAll(memDir, 0o755))

	// Write a memory file with frontmatter.
	require.NoError(t, os.WriteFile(
		filepath.Join(memDir, "orm-decision.md"),
		[]byte("---\nname: ORM Decision\ndescription: Always use Drizzle\n---\nWe chose Drizzle ORM.\n"),
		0o644,
	))
	// Write a memory file without frontmatter (name derived from filename).
	require.NoError(t, os.WriteFile(
		filepath.Join(memDir, "api-patterns.md"),
		[]byte("All endpoints use JSON:API format.\n"),
		0o644,
	))

	// Step 1: Discover memory entries.
	discovered := compiler.DiscoverAgentMemory(baseDir)
	require.Len(t, discovered, 2, "should discover 2 memory entries")

	// Step 2: Feed to Claude renderer.
	outputDir := t.TempDir()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: discovered,
		},
	}
	r := claude.NewMemoryRenderer(outputDir)
	_, _, err := r.Compile(config, baseDir)
	require.NoError(t, err)

	// Step 3: Verify output.
	// MEMORY.md should use link-list format.
	indexPath := filepath.Join(outputDir, "dev", "MEMORY.md")
	require.FileExists(t, indexPath)
	indexData, err := os.ReadFile(indexPath)
	require.NoError(t, err)
	indexContent := string(indexData)
	require.Contains(t, indexContent, "- [ORM Decision](orm-decision.md)")
	require.Contains(t, indexContent, "api-patterns")
	require.NotContains(t, indexContent, "## ", "must use link-list, not ## headings")

	// Individual files should exist.
	require.FileExists(t, filepath.Join(outputDir, "dev", "orm-decision.md"))
	require.FileExists(t, filepath.Join(outputDir, "dev", "api-patterns.md"))

	// Verify individual file content.
	ormData, err := os.ReadFile(filepath.Join(outputDir, "dev", "orm-decision.md"))
	require.NoError(t, err)
	require.Contains(t, string(ormData), "We chose Drizzle ORM.")
}
