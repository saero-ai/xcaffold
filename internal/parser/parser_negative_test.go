package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_NegativeKind_Config(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "bad.xcaf")
	require.NoError(t, os.WriteFile(f, []byte("kind: config\nname: old\nversion: \"1.0\"\n"), 0600))
	_, err := ParseFileExact(f)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "kind \"config\" has been removed")
}

func TestParse_NegativeKind_Unknown(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "bad.xcaf")
	require.NoError(t, os.WriteFile(f, []byte("kind: foobar\nname: x\nversion: \"1.0\"\n"), 0600))
	_, err := ParseFileExact(f)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "foobar")
}

func TestParse_NegativeKind_Empty(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "bad.xcaf")
	require.NoError(t, os.WriteFile(f, []byte("name: x\nversion: \"1.0\"\n"), 0600))
	_, err := ParseFileExact(f)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "kind field is required")
}

func TestParse_Negative_FrontmatterUnknownField(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "bad.xcaf")
	content := "---\nkind: skill\nname: tdd\nversion: \"1.0\"\nbogus-key: oops\n---\nbody\n"
	require.NoError(t, os.WriteFile(f, []byte(content), 0600))
	_, err := ParseFileExact(f)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogus-key")
}

func TestParseDirectory_MultiFileProject(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcaf"),
		[]byte("kind: project\nname: multifile\nversion: \"1.0\"\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "dev.xcaf"),
		[]byte("kind: agent\nname: developer\nversion: \"1.0\"\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tdd.xcaf"),
		[]byte("kind: skill\nname: tdd\nversion: \"1.0\"\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "testing.xcaf"),
		[]byte("kind: rule\nname: testing\nversion: \"1.0\"\n"), 0600))
	cfg, err := ParseDirectory(dir)
	require.NoError(t, err)
	assert.NotNil(t, cfg.Agents["developer"])
	assert.NotNil(t, cfg.Skills["tdd"])
	assert.NotNil(t, cfg.Rules["testing"])
}

func TestParseDirectory_FilterExcludesOutputDirs(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcaf"),
		[]byte("kind: project\nname: myproject\nversion: \"1.0\"\n"), 0600))
	claudeDir := filepath.Join(dir, ".claude")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "should-skip.xcaf"),
		[]byte("kind: agent\nname: phantom\nversion: \"1.0\"\n"), 0600))
	cfg, err := ParseDirectory(dir)
	require.NoError(t, err)
	_, found := cfg.Agents["phantom"]
	assert.False(t, found, ".claude/ must be excluded from the scan")
}

func TestParseDirectory_FrontmatterMultiFileWithBodies(t *testing.T) {
	dir := t.TempDir()
	// ProjectConfig no longer supports Body (moved to kind:context).
	// Test that agent bodies are still parsed correctly.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcaf"),
		[]byte("---\nkind: project\nname: my-app\nversion: \"1.0\"\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "context.xcaf"),
		[]byte("---\nkind: context\nname: root\nversion: \"1.0\"\n---\nProject instructions.\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "dev.xcaf"),
		[]byte("---\nkind: agent\nname: developer\nversion: \"1.0\"\n---\nYou are a developer.\n"), 0600))
	cfg, err := ParseDirectory(dir)
	require.NoError(t, err)
	assert.Equal(t, "Project instructions.", cfg.ResourceScope.Contexts["root"].Body)
	assert.Equal(t, "You are a developer.", cfg.Agents["developer"].Body)
}
