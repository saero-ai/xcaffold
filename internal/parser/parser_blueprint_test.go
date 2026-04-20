package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlueprint_ParsesBasicDocument(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte("kind: project\nversion: \"1.0\"\nname: test-project\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agent.xcf"), []byte("kind: agent\nversion: \"1.0\"\nname: developer\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "skill.xcf"), []byte("kind: skill\nversion: \"1.0\"\nname: tdd\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "rule.xcf"), []byte("kind: rule\nversion: \"1.0\"\nname: testing\n"), 0600))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "xcf", "blueprints"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "xcf", "blueprints", "backend.xcf"), []byte(`kind: blueprint
version: "1.0"
name: backend
description: Backend engineering
agents:
  - developer
skills:
  - tdd
rules:
  - testing
`), 0600))
	cfg, err := ParseDirectory(dir)
	require.NoError(t, err)
	require.Contains(t, cfg.Blueprints, "backend")
	assert.Equal(t, "Backend engineering", cfg.Blueprints["backend"].Description)
	assert.Equal(t, []string{"developer"}, cfg.Blueprints["backend"].Agents)
	assert.Equal(t, []string{"tdd"}, cfg.Blueprints["backend"].Skills)
	assert.Equal(t, []string{"testing"}, cfg.Blueprints["backend"].Rules)
}

func TestBlueprint_ActiveField(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte("kind: project\nversion: \"1.0\"\nname: test-project\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bp.xcf"), []byte("kind: blueprint\nversion: \"1.0\"\nname: active-bp\nactive: true\n"), 0600))
	cfg, err := ParseDirectory(dir)
	require.NoError(t, err)
	require.Contains(t, cfg.Blueprints, "active-bp")
	assert.True(t, cfg.Blueprints["active-bp"].Active)
}

func TestBlueprint_UnknownField_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte("kind: project\nversion: \"1.0\"\nname: x\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bad-bp.xcf"), []byte("kind: blueprint\nversion: \"1.0\"\nname: myblueprint\nunknown-field: bad\n"), 0600))
	_, err := ParseDirectory(dir)
	require.Error(t, err)
}

func TestBlueprint_DuplicateName_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte("kind: project\nversion: \"1.0\"\nname: x\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "p1.xcf"), []byte("kind: blueprint\nversion: \"1.0\"\nname: backend\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "p2.xcf"), []byte("kind: blueprint\nversion: \"1.0\"\nname: backend\n"), 0600))
	_, err := ParseDirectory(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate blueprint name")
}

func TestBlueprint_EmptyName_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte("kind: project\nversion: \"1.0\"\nname: x\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bp.xcf"), []byte("kind: blueprint\nversion: \"1.0\"\nname: \"\"\n"), 0600))
	_, err := ParseDirectory(dir)
	require.Error(t, err)
}

func TestBlueprint_InvalidNameChars_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte("kind: project\nversion: \"1.0\"\nname: x\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bp.xcf"), []byte("kind: blueprint\nversion: \"1.0\"\nname: \"Backend Engineering\"\n"), 0600))
	_, err := ParseDirectory(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestBlueprint_MultipleActive_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte("kind: project\nversion: \"1.0\"\nname: x\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.xcf"), []byte("kind: blueprint\nversion: \"1.0\"\nname: a\nactive: true\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.xcf"), []byte("kind: blueprint\nversion: \"1.0\"\nname: b\nactive: true\n"), 0600))
	_, err := ParseDirectory(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple blueprints")
}

func TestBlueprint_MissingVersion_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bp.xcf"), []byte("kind: blueprint\nname: myblueprint\n"), 0600))
	_, err := ParseFileExact(filepath.Join(dir, "bp.xcf"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version")
}

func TestBlueprint_ParsedViaParseFileExact(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "backend.xcf")
	require.NoError(t, os.WriteFile(f, []byte(`kind: blueprint
version: "1.0"
name: backend
description: The backend blueprint
workflows:
  - deploy
mcp:
  - github
`), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	require.Contains(t, cfg.Blueprints, "backend")
	bp := cfg.Blueprints["backend"]
	assert.Equal(t, "The backend blueprint", bp.Description)
	assert.Equal(t, []string{"deploy"}, bp.Workflows)
	assert.Equal(t, []string{"github"}, bp.MCP)
}

func TestBlueprint_SingleActive_IsValid(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte("kind: project\nversion: \"1.0\"\nname: x\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.xcf"), []byte("kind: blueprint\nversion: \"1.0\"\nname: a\nactive: true\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.xcf"), []byte("kind: blueprint\nversion: \"1.0\"\nname: b\n"), 0600))
	_, err := ParseDirectory(dir)
	require.NoError(t, err)
}
