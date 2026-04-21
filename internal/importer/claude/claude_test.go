package claude

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFrontmatter_EmptyFrontmatter(t *testing.T) {
	input := []byte("---\n---\n\n# Rule: Worktree Creation\n\nThis is the body.")
	var front struct {
		Name string `yaml:"name"`
	}
	body, err := parseFrontmatter(input, &front)
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
	body, err := parseFrontmatter(input, &front)
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
	body, err := parseFrontmatter(input, &front)
	require.NoError(t, err)
	assert.Equal(t, "", front.Name)
	assert.Equal(t, "# Just markdown\n\nNo frontmatter here.", body)
}
