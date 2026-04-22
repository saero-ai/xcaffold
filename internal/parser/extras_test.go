package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimalScaffoldXCF is the smallest valid project.xcf content for test dirs.
const minimalScaffoldXCF = `kind: project
version: "1.0"
name: "test-project"
`

// writeExtrasFile creates the directory hierarchy and writes content to
// <dir>/xcf/extras/<provider>/<relpath>.
func writeExtrasFile(t *testing.T, dir, provider, relpath string, content []byte) {
	t.Helper()
	full := filepath.Join(dir, "xcf", "extras", provider, filepath.FromSlash(relpath))
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, content, 0o644))
}

func TestLoadExtras_PopulatesProviderExtras(t *testing.T) {
	dir := t.TempDir()

	// Write a minimal project.xcf so ParseDirectory succeeds.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte(minimalScaffoldXCF), 0o644))

	wantData := []byte("#!/bin/sh\necho hello\n")
	writeExtrasFile(t, dir, "claude", "hooks/pre-commit.sh", wantData)

	cfg, err := ParseDirectory(dir)
	require.NoError(t, err)

	require.NotNil(t, cfg.ProviderExtras, "ProviderExtras must be populated")
	require.Contains(t, cfg.ProviderExtras, "claude")
	assert.Equal(t, wantData, cfg.ProviderExtras["claude"]["hooks/pre-commit.sh"])
}

func TestLoadExtras_NoExtrasDir_NoError(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte(minimalScaffoldXCF), 0o644))

	// No xcf/extras/ directory written — must succeed with empty/nil ProviderExtras.
	cfg, err := ParseDirectory(dir)
	require.NoError(t, err)
	assert.Empty(t, cfg.ProviderExtras)
}

func TestLoadExtras_MultipleProviders(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte(minimalScaffoldXCF), 0o644))

	claudeData := []byte("claude hook content")
	cursorData := []byte("cursor rule content")
	writeExtrasFile(t, dir, "claude", "hooks/pre-commit.sh", claudeData)
	writeExtrasFile(t, dir, "cursor", "rules/style.md", cursorData)

	cfg, err := ParseDirectory(dir)
	require.NoError(t, err)

	require.NotNil(t, cfg.ProviderExtras)
	assert.Equal(t, claudeData, cfg.ProviderExtras["claude"]["hooks/pre-commit.sh"])
	assert.Equal(t, cursorData, cfg.ProviderExtras["cursor"]["rules/style.md"])
}

func TestLoadExtras_NestedPaths(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte(minimalScaffoldXCF), 0o644))

	deepData := []byte("deep nested content")
	writeExtrasFile(t, dir, "claude", "hooks/nested/deep.sh", deepData)

	cfg, err := ParseDirectory(dir)
	require.NoError(t, err)

	require.NotNil(t, cfg.ProviderExtras)
	assert.Equal(t, deepData, cfg.ProviderExtras["claude"]["hooks/nested/deep.sh"])
}

// writeProviderFile creates the directory hierarchy and writes content to
// <dir>/xcf/provider/<provider>/<relpath>.
func writeProviderFile(t *testing.T, dir, provider, relpath string, content []byte) {
	t.Helper()
	full := filepath.Join(dir, "xcf", "provider", provider, filepath.FromSlash(relpath))
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, content, 0o644))
}

func TestLoadExtras_ReadsFromProviderDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte(minimalScaffoldXCF), 0o644))

	wantData := []byte("provider hook content")
	writeProviderFile(t, dir, "claude", "hooks/pre-commit.sh", wantData)

	cfg, err := ParseDirectory(dir)
	require.NoError(t, err)

	require.NotNil(t, cfg.ProviderExtras, "ProviderExtras must be populated from xcf/provider/")
	require.Contains(t, cfg.ProviderExtras, "claude")
	assert.Equal(t, wantData, cfg.ProviderExtras["claude"]["hooks/pre-commit.sh"])
}

func TestLoadExtras_FileWithoutProviderSubdir_Skipped(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte(minimalScaffoldXCF), 0o644))

	// Write file directly under xcf/extras/ with no provider subdirectory
	orphanDir := filepath.Join(dir, "xcf", "extras")
	require.NoError(t, os.MkdirAll(orphanDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(orphanDir, "orphan.sh"), []byte("#!/bin/sh"), 0o644))

	cfg, err := ParseDirectory(dir)
	require.NoError(t, err)
	assert.Empty(t, cfg.ProviderExtras, "file without provider subdirectory should be skipped")
}

func TestLoadExtras_PrefersProviderOverExtras(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte(minimalScaffoldXCF), 0o644))

	providerData := []byte("provider version wins")
	extrasData := []byte("extras version loses")

	writeProviderFile(t, dir, "claude", "hooks/pre-commit.sh", providerData)
	writeExtrasFile(t, dir, "claude", "hooks/pre-commit.sh", extrasData)

	cfg, err := ParseDirectory(dir)
	require.NoError(t, err)

	require.NotNil(t, cfg.ProviderExtras)
	assert.Equal(t, providerData, cfg.ProviderExtras["claude"]["hooks/pre-commit.sh"],
		"xcf/provider/ content must win over xcf/extras/ when the same relpath exists in both")
}
