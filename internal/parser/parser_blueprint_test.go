package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlueprint_Targets_ParsedFromYAML(t *testing.T) {
	// Create a temp dir with a blueprint xcaf that has targets
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "xcaf", "blueprints"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcaf"), []byte("kind: project\nversion: \"1.0\"\nname: x\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "xcaf", "blueprints", "test-bp.xcaf"), []byte(`kind: blueprint
version: "1.0"
name: test-bp
targets:
  - claude
  - gemini
agents:
  - my-agent
`), 0o600))

	config, err := ParseDirectory(dir)
	require.NoError(t, err)
	bp, ok := config.Blueprints["test-bp"]
	require.True(t, ok)
	require.Equal(t, []string{"claude", "gemini"}, bp.Targets)
}

func TestBlueprint_Targets_EmptyOmitted(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "xcaf", "blueprints"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcaf"), []byte("kind: project\nversion: \"1.0\"\nname: x\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "xcaf", "blueprints", "test-bp.xcaf"), []byte(`kind: blueprint
version: "1.0"
name: test-bp
agents:
  - my-agent
`), 0o600))

	config, err := ParseDirectory(dir)
	require.NoError(t, err)
	bp, ok := config.Blueprints["test-bp"]
	require.True(t, ok)
	require.Nil(t, bp.Targets)
}

func TestBlueprint_Targets_SingleTarget(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "xcaf", "blueprints"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcaf"), []byte("kind: project\nversion: \"1.0\"\nname: x\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "xcaf", "blueprints", "test-bp.xcaf"), []byte(`kind: blueprint
version: "1.0"
name: test-bp
targets:
  - cursor
agents:
  - my-agent
`), 0o600))

	config, err := ParseDirectory(dir)
	require.NoError(t, err)
	bp, ok := config.Blueprints["test-bp"]
	require.True(t, ok)
	require.Equal(t, []string{"cursor"}, bp.Targets)
}

func TestBlueprint_ParsesBasicDocument(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcaf"), []byte("kind: project\nversion: \"1.0\"\nname: test-project\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agent.xcaf"), []byte("kind: agent\nversion: \"1.0\"\nname: developer\ndescription: \"test agent\"\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "skill.xcaf"), []byte("kind: skill\nversion: \"1.0\"\nname: tdd\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "rule.xcaf"), []byte("kind: rule\nversion: \"1.0\"\nname: testing\n"), 0600))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "xcaf", "blueprints"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "xcaf", "blueprints", "backend.xcaf"), []byte(`kind: blueprint
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

func TestBlueprint_UnknownField_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcaf"), []byte("kind: project\nversion: \"1.0\"\nname: x\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bad-bp.xcaf"), []byte("kind: blueprint\nversion: \"1.0\"\nname: myblueprint\nunknown-field: bad\n"), 0600))
	_, err := ParseDirectory(dir)
	require.Error(t, err)
}

func TestBlueprint_DuplicateName_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcaf"), []byte("kind: project\nversion: \"1.0\"\nname: x\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "p1.xcaf"), []byte("kind: blueprint\nversion: \"1.0\"\nname: backend\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "p2.xcaf"), []byte("kind: blueprint\nversion: \"1.0\"\nname: backend\n"), 0600))
	_, err := ParseDirectory(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate blueprint name")
}

func TestBlueprint_EmptyName_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcaf"), []byte("kind: project\nversion: \"1.0\"\nname: x\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bp.xcaf"), []byte("kind: blueprint\nversion: \"1.0\"\nname: \"\"\n"), 0600))
	_, err := ParseDirectory(dir)
	require.Error(t, err)
}

func TestBlueprint_InvalidNameChars_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcaf"), []byte("kind: project\nversion: \"1.0\"\nname: x\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bp.xcaf"), []byte("kind: blueprint\nversion: \"1.0\"\nname: \"Backend Engineering\"\n"), 0600))
	_, err := ParseDirectory(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestBlueprint_MissingVersion_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bp.xcaf"), []byte("kind: blueprint\nname: myblueprint\n"), 0600))
	_, err := ParseFileExact(filepath.Join(dir, "bp.xcaf"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version")
}

func TestBlueprint_ParsedViaParseFileExact(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "backend.xcaf")
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

func TestBlueprint_FixturesParse(t *testing.T) {
	// Verifies that each blueprint fixture file is structurally valid YAML.
	// ParseFileExact runs validatePartial (IDs, hooks, instructions, activations)
	// but not validateMerged (cross-ref checks), so fixtures may reference resources
	// that do not exist in the fixture directory without failing here.
	fixtureDir := filepath.Join("..", "..", "testing", "fixtures", "blueprints")

	entries, err := os.ReadDir(fixtureDir)
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".xcaf") {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			path := filepath.Join(fixtureDir, entry.Name())
			_, err := ParseFileExact(path)
			require.NoError(t, err, "fixture %s should parse without error", entry.Name())
		})
	}
}
