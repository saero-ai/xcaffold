package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_Reference_BasicParsing(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "guide.xcf")
	content := "---\nkind: reference\nname: api-guide\nversion: \"1.0\"\ndescription: API usage patterns\n---\n# API Guide\nUse REST.\n"
	require.NoError(t, os.WriteFile(f, []byte(content), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	ref, ok := cfg.References["api-guide"]
	require.True(t, ok)
	assert.Equal(t, "api-guide", ref.Name)
	assert.Equal(t, "API usage patterns", ref.Description)
	assert.Equal(t, "# API Guide\nUse REST.", ref.Content)
}

func TestParse_Reference_StoredInResourceScope(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "ref.xcf")
	require.NoError(t, os.WriteFile(f, []byte("kind: reference\nname: my-ref\nversion: \"1.0\"\n"), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	assert.NotNil(t, cfg.References)
	_, ok := cfg.References["my-ref"]
	assert.True(t, ok)
}

func TestParse_Reference_UnknownFieldError(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "bad-ref.xcf")
	content := "kind: reference\nname: bad\nversion: \"1.0\"\nunknown-field: oops\n"
	require.NoError(t, os.WriteFile(f, []byte(content), 0600))
	_, err := ParseFileExact(f)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown-field")
}

func TestParse_Reference_MissingNameError(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "noname.xcf")
	require.NoError(t, os.WriteFile(f, []byte("kind: reference\nversion: \"1.0\"\ndescription: no name here\n"), 0600))
	_, err := ParseFileExact(f)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

func TestParse_Reference_DuplicateNameError(t *testing.T) {
	dir := t.TempDir()
	proj := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(proj, []byte("kind: project\nname: myproject\nversion: \"1.0\"\n"), 0600))
	r1 := filepath.Join(dir, "ref1.xcf")
	r2 := filepath.Join(dir, "ref2.xcf")
	require.NoError(t, os.WriteFile(r1, []byte("kind: reference\nname: guide\nversion: \"1.0\"\n"), 0600))
	require.NoError(t, os.WriteFile(r2, []byte("kind: reference\nname: guide\nversion: \"1.0\"\n"), 0600))
	_, err := ParseDirectory(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "guide")
}

func TestParse_Reference_NoBodyContentIsEmpty(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "ref.xcf")
	require.NoError(t, os.WriteFile(f, []byte("kind: reference\nname: empty-ref\nversion: \"1.0\"\n"), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	ref, ok := cfg.References["empty-ref"]
	require.True(t, ok)
	assert.Equal(t, "", ref.Content)
}

func TestParse_Reference_WithFrontmatterBody(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "guide.xcf")
	content := "---\nkind: reference\nname: coding-guide\nversion: \"1.0\"\n---\n# Coding Standards\n\nFollow clean code principles.\n"
	require.NoError(t, os.WriteFile(f, []byte(content), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	ref, ok := cfg.References["coding-guide"]
	require.True(t, ok)
	assert.Equal(t, "# Coding Standards\n\nFollow clean code principles.", ref.Content)
}
