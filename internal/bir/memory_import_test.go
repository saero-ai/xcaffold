package bir

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportMemory_Claude_DiscoverFiles(t *testing.T) {
	memDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(memDir, "user-role.md"), []byte("content"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(memDir, "project-context.md"), []byte("content"), 0o600))

	entries, err := DiscoverClaudeMemory(memDir)
	require.NoError(t, err)
	require.Len(t, entries, 2)
}

func TestImportMemory_Claude_ParseFrontmatter(t *testing.T) {
	memDir := t.TempDir()
	content := `---
type: user
description: Developer role.
---
Robert is the founder.
`
	require.NoError(t, os.WriteFile(filepath.Join(memDir, "user-role.md"), []byte(content), 0o600))

	entries, err := DiscoverClaudeMemory(memDir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "user", entries[0].Type)
	require.Equal(t, "Developer role.", entries[0].Description)
	require.Contains(t, entries[0].Body, "Robert is the founder.")
}

func TestImportMemory_Claude_DeriveKey(t *testing.T) {
	require.Equal(t, "user-role", DeriveMemoryKey("User Role.md"))
	require.Equal(t, "architecture-decisions", DeriveMemoryKey("architecture_decisions.md"))
	require.Equal(t, "my-file", DeriveMemoryKey("my--file.md"))            // collapse multiple hyphens
	require.Equal(t, "trimmed", DeriveMemoryKey("--trimmed--.md"))         // trim leading/trailing
	require.Equal(t, "section-three", DeriveMemoryKey("section.three.md")) // dots to hyphens
}

func TestImportMemory_Claude_WritesSidecar(t *testing.T) {
	tmp := t.TempDir()
	sidecarDir := filepath.Join(tmp, "xcf", "memory")
	require.NoError(t, os.MkdirAll(sidecarDir, 0o755))

	entry := MemoryImportEntry{
		Key:         "user-role",
		Type:        "user",
		Description: "Developer role.",
		Body:        "Robert is the founder.",
	}

	err := WriteSidecar(sidecarDir, entry)
	require.NoError(t, err)
	path := filepath.Join(sidecarDir, "user-role.md")
	require.FileExists(t, path)

	data, _ := os.ReadFile(path)
	content := string(data)
	require.Contains(t, content, "type: user")
	require.Contains(t, content, "Robert is the founder.")
}

func TestImportMemory_Conflict_NoForce(t *testing.T) {
	existing := map[string]bool{"user-role": true}
	entry := MemoryImportEntry{Key: "user-role"}

	skipped, warning := HandleConflict(existing, entry, false)
	require.True(t, skipped)
	require.Contains(t, warning, "user-role")
}

func TestImportMemory_Conflict_Force(t *testing.T) {
	existing := map[string]bool{"user-role": true}
	entry := MemoryImportEntry{Key: "user-role"}

	skipped, _ := HandleConflict(existing, entry, true)
	require.False(t, skipped)
}

func TestImportMemory_Gemini_ExtractBlocks(t *testing.T) {
	geminiMd := `# GEMINI.md
## Gemini Added Memories

<!-- xcaffold:memory name="user-role" type="user" seeded-at="2026-04-15T00:00:00Z" -->
**user-role** (user): Developer role.

Robert is the founder.
<!-- xcaffold:/memory -->
`
	blocks, err := ExtractGeminiMemoryBlocks(geminiMd)
	require.NoError(t, err)
	require.Len(t, blocks, 1)
	require.Equal(t, "user-role", blocks[0].Key)
	require.Equal(t, "user", blocks[0].Type)
	require.Contains(t, blocks[0].Body, "Robert is the founder.")
}

func TestImportMemory_Gemini_MultipleBlocks(t *testing.T) {
	geminiMd := `## Gemini Added Memories

<!-- xcaffold:memory name="one" type="user" seeded-at="2026-04-15T00:00:00Z" -->
**one** (user): first

body-one
<!-- xcaffold:/memory -->

<!-- xcaffold:memory name="two" type="project" seeded-at="2026-04-15T00:00:01Z" -->
**two** (project): second

body-two
<!-- xcaffold:/memory -->
`
	blocks, err := ExtractGeminiMemoryBlocks(geminiMd)
	require.NoError(t, err)
	require.Len(t, blocks, 2)
	require.Equal(t, "one", blocks[0].Key)
	require.Equal(t, "two", blocks[1].Key)
}

func TestImportMemory_Plan_NoDiskWrite(t *testing.T) {
	tmp := t.TempDir()
	memDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(memDir, "user-role.md"),
		[]byte("Robert is the founder."), 0o600))

	opts := ImportOpts{PlanOnly: true, SidecarDir: filepath.Join(tmp, "xcf", "memory")}
	summary, err := ImportClaudeMemory(memDir, opts)
	require.NoError(t, err)
	require.Equal(t, 1, summary.WouldImport)
	require.Equal(t, 0, summary.Imported)

	_, statErr := os.Stat(filepath.Join(tmp, "xcf", "memory", "user-role.md"))
	require.True(t, os.IsNotExist(statErr), "plan mode must not write files")
}

func TestImportMemory_RoundTrip_Claude(t *testing.T) {
	memDir := t.TempDir()
	sidecarDir := filepath.Join(t.TempDir(), "xcf", "memory")
	content := `---
type: user
description: Developer role.
---
Robert is the founder.
`
	require.NoError(t, os.WriteFile(filepath.Join(memDir, "user-role.md"), []byte(content), 0o600))

	summary, err := ImportClaudeMemory(memDir, ImportOpts{SidecarDir: sidecarDir})
	require.NoError(t, err)
	require.Equal(t, 1, summary.Imported)
	require.FileExists(t, filepath.Join(sidecarDir, "user-role.md"))
}
