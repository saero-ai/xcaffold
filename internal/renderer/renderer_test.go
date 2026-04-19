package renderer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveInstructionsContent_InlinePreferred(t *testing.T) {
	dir := t.TempDir()
	// Write a file that would be read if inline is not preferred.
	filePath := filepath.Join(dir, "instructions.xcf")
	require.NoError(t, os.WriteFile(filePath, []byte("kind: instructions\ninstructions: from file\n"), 0600))

	got := ResolveInstructionsContent("inline content", "instructions.xcf", dir)
	assert.Equal(t, "inline content", got)
}

func TestResolveInstructionsContent_MdFile_ReturnsRawContent(t *testing.T) {
	dir := t.TempDir()
	content := "# My Rules\n\nSome instructions here.\n"
	filePath := filepath.Join(dir, "RULES.md")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0600))

	got := ResolveInstructionsContent("", "RULES.md", dir)
	assert.Equal(t, content, got)
}

func TestResolveInstructionsContent_XCFFile_ExtractsInstructions(t *testing.T) {
	dir := t.TempDir()
	xcfContent := `kind: instructions
version: "1.0"
name: root
instructions: |
  # Actual instructions content
  Some rules and guidelines...
`
	filePath := filepath.Join(dir, "root.xcf")
	require.NoError(t, os.WriteFile(filePath, []byte(xcfContent), 0600))

	got := ResolveInstructionsContent("", "root.xcf", dir)
	assert.Equal(t, "# Actual instructions content\nSome rules and guidelines...\n", got)
}

func TestResolveInstructionsContent_XCFFile_MalformedYAML_FallsBack(t *testing.T) {
	dir := t.TempDir()
	bad := ":\t: bad yaml\n\x00\x01\x02"
	filePath := filepath.Join(dir, "broken.xcf")
	require.NoError(t, os.WriteFile(filePath, []byte(bad), 0600))

	got := ResolveInstructionsContent("", "broken.xcf", dir)
	assert.Equal(t, bad, got)
}

func TestResolveInstructionsContent_XCFFile_NoInstructionsField_FallsBack(t *testing.T) {
	dir := t.TempDir()
	xcfContent := `kind: instructions
version: "1.0"
name: root
`
	filePath := filepath.Join(dir, "empty.xcf")
	require.NoError(t, os.WriteFile(filePath, []byte(xcfContent), 0600))

	got := ResolveInstructionsContent("", "empty.xcf", dir)
	assert.Equal(t, xcfContent, got)
}

func TestResolveInstructionsContent_MissingFile_ReturnsEmpty(t *testing.T) {
	got := ResolveInstructionsContent("", "nonexistent.xcf", t.TempDir())
	assert.Equal(t, "", got)
}

func TestResolveInstructionsContent_EmptyInlineAndFile_ReturnsEmpty(t *testing.T) {
	got := ResolveInstructionsContent("", "", "/any/dir")
	assert.Equal(t, "", got)
}
