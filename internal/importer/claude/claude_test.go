package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/importer"
)

func TestParseFrontmatter_EmptyFrontmatter(t *testing.T) {
	input := []byte("---\n---\n\n# Rule: Worktree Creation\n\nThis is the body.")
	var front struct {
		Name string `yaml:"name"`
	}
	body, err := importer.ParseFrontmatterLenient(input, &front)
	require.NoError(t, err)
	assert.Equal(t, "", front.Name, "empty frontmatter should produce zero-value struct")
	assert.Equal(t, "# Rule: Worktree Creation\n\nThis is the body.", body)
	assert.NotContains(t, body, "---", "body must not contain frontmatter delimiters")
}

func TestParseFrontmatter_NormalFrontmatter(t *testing.T) {
	input := []byte("---\nname: developer\ndescription: Dev agent\n---\nYou are a developer.")
	var front struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	body, err := importer.ParseFrontmatterLenient(input, &front)
	require.NoError(t, err)
	assert.Equal(t, "developer", front.Name)
	assert.Equal(t, "Dev agent", front.Description)
	assert.Equal(t, "You are a developer.", body)
}

func TestParseFrontmatter_NoFrontmatter(t *testing.T) {
	input := []byte("# Just markdown\n\nNo frontmatter here.")
	var front struct {
		Name string `yaml:"name"`
	}
	body, err := importer.ParseFrontmatterLenient(input, &front)
	require.NoError(t, err)
	assert.Equal(t, "", front.Name)
	assert.Equal(t, "# Just markdown\n\nNo frontmatter here.", body)
}

func TestImport_Memory_OnlyProjectAgents(t *testing.T) {
	c := New()
	dir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(dir, "agents"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "agent-memory/a"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "agent-memory/b"), 0755))

	agentContent := []byte("---\nname: Agent A\n---\nHello")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agents/a.md"), agentContent, 0644))

	memAContent := []byte("---\ntype: user\n---\nMemory A")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agent-memory/a/context.md"), memAContent, 0644))

	memBContent := []byte("---\ntype: project\n---\nMemory B")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agent-memory/b/context.md"), memBContent, 0644))

	config := &ast.XcaffoldConfig{}
	err := c.Import(dir, config)
	require.NoError(t, err)

	_, okA := config.Memory["a/context"]
	assert.True(t, okA, "memory a/context should be kept since agent 'a' exists")

	_, okB := config.Memory["b/context"]
	assert.False(t, okB, "memory b/context should be dropped since agent 'b' does not exist")

	foundWarning := false
	for _, w := range c.Warnings {
		if strings.Contains(w, "skipped \"agent-memory/b/context\": agent \"b\" not found in xcf/agents") {
			foundWarning = true
			break
		}
	}
	assert.True(t, foundWarning, "expected a warning about skipped memory for agent b")
}
