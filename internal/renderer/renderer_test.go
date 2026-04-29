package renderer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
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

// ── ValidateContextUniqueness ─────────────────────────────────────────────────

func TestValidateContextUniqueness_SingleContext_OK(t *testing.T) {
	contexts := map[string]ast.ContextConfig{
		"main": {Name: "main", Body: "hello"},
	}
	err := ValidateContextUniqueness(contexts, []string{"claude"})
	require.NoError(t, err)
}

func TestValidateContextUniqueness_MultipleNoOverlap_OK(t *testing.T) {
	contexts := map[string]ast.ContextConfig{
		"claude-ctx": {Name: "claude-ctx", Body: "for claude", Targets: []string{"claude"}},
		"gemini-ctx": {Name: "gemini-ctx", Body: "for gemini", Targets: []string{"gemini"}},
	}
	err := ValidateContextUniqueness(contexts, []string{"claude", "gemini"})
	require.NoError(t, err)
}

func TestValidateContextUniqueness_MultipleOverlap_NoDefault_Error(t *testing.T) {
	contexts := map[string]ast.ContextConfig{
		"ctx-a": {Name: "ctx-a", Body: "a", Targets: []string{"claude"}},
		"ctx-b": {Name: "ctx-b", Body: "b", Targets: []string{"claude"}},
	}
	err := ValidateContextUniqueness(contexts, []string{"claude"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"claude"`)
	assert.Contains(t, err.Error(), "default")
}

func TestValidateContextUniqueness_MultipleOverlap_WithDefault_OK(t *testing.T) {
	contexts := map[string]ast.ContextConfig{
		"ctx-a": {Name: "ctx-a", Body: "a", Targets: []string{"claude"}},
		"ctx-b": {Name: "ctx-b", Body: "b", Targets: []string{"claude"}, Default: true},
	}
	err := ValidateContextUniqueness(contexts, []string{"claude"})
	require.NoError(t, err)
}

func TestValidateContextUniqueness_MultipleDefaults_Error(t *testing.T) {
	contexts := map[string]ast.ContextConfig{
		"ctx-a": {Name: "ctx-a", Body: "a", Targets: []string{"claude"}, Default: true},
		"ctx-b": {Name: "ctx-b", Body: "b", Targets: []string{"claude"}, Default: true},
	}
	err := ValidateContextUniqueness(contexts, []string{"claude"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple contexts marked as default")
	assert.Contains(t, err.Error(), `"claude"`)
}

// ── ResolveContextBody ────────────────────────────────────────────────────────

func TestResolveContextBody_MultipleMatchSelectsDefault(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Contexts: map[string]ast.ContextConfig{
				"ctx-main":  {Name: "ctx-main", Body: "main body", Targets: []string{"claude"}},
				"ctx-extra": {Name: "ctx-extra", Body: "extra body", Targets: []string{"claude"}, Default: true},
			},
		},
	}
	got := ResolveContextBody(config, "claude")
	assert.Equal(t, "extra body", got)
}
