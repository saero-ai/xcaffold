package importer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/importer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFrontmatter_Strict_ValidYAML(t *testing.T) {
	input := []byte("---\nname: test\n---\nBody content.")
	var front struct {
		Name string `yaml:"name"`
	}
	body, err := importer.ParseFrontmatter(input, &front)
	require.NoError(t, err)
	assert.Equal(t, "test", front.Name)
	assert.Equal(t, "Body content.", body)
}

func TestParseFrontmatter_Strict_MalformedYAML(t *testing.T) {
	input := []byte("---\ninvalid: :\n---\nBody content.")
	var front struct {
		Name string `yaml:"name"`
	}
	_, err := importer.ParseFrontmatter(input, &front)
	assert.Error(t, err, "strict variant must return error on malformed YAML")
	assert.Contains(t, err.Error(), "frontmatter:")
}

func TestParseFrontmatter_Strict_NoFrontmatter(t *testing.T) {
	input := []byte("Just plain content.")
	var front struct{}
	body, err := importer.ParseFrontmatter(input, &front)
	require.NoError(t, err)
	assert.Equal(t, "Just plain content.", body)
}

func TestParseFrontmatterLenient_MalformedYAML(t *testing.T) {
	input := []byte("---\ninvalid: :\n---\nBody content.")
	var front struct {
		Name string `yaml:"name"`
	}
	body, err := importer.ParseFrontmatterLenient(input, &front)
	require.NoError(t, err, "lenient variant must not return error on malformed YAML")
	assert.Equal(t, "Body content.", body)
	assert.Equal(t, "", front.Name)
}

func TestMatchGlob_ExactMatch(t *testing.T) {
	assert.True(t, importer.MatchGlob("agents/dev.md", "agents/dev.md"))
}

func TestMatchGlob_SingleWildcard(t *testing.T) {
	assert.True(t, importer.MatchGlob("agents/*.md", "agents/dev.md"))
	assert.False(t, importer.MatchGlob("agents/*.md", "agents/sub/dev.md"))
}

func TestMatchGlob_DoubleWildcard(t *testing.T) {
	assert.True(t, importer.MatchGlob("**/*.md", "agents/dev.md"))
	assert.True(t, importer.MatchGlob("**/*.md", "agents/sub/dev.md"))
	assert.False(t, importer.MatchGlob("**/*.md", "agents/dev.txt"))
}

func TestMatchGlob_NoMatch(t *testing.T) {
	assert.False(t, importer.MatchGlob("agents/*.md", "rules/sec.md"))
}

func TestAppendUnique_NewElement(t *testing.T) {
	result := importer.AppendUnique([]string{"a", "b"}, "c")
	assert.Equal(t, []string{"a", "b", "c"}, result)
}

func TestAppendUnique_Duplicate(t *testing.T) {
	result := importer.AppendUnique([]string{"a", "b"}, "b")
	assert.Equal(t, []string{"a", "b"}, result)
}

func TestAppendUnique_EmptySlice(t *testing.T) {
	result := importer.AppendUnique(nil, "a")
	assert.Equal(t, []string{"a"}, result)
}

func TestWalkProviderDir_CallsVisitorForEachFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.md"), []byte("alpha"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub", "b.md"), []byte("beta"), 0o644))

	var visited []string
	err := importer.WalkProviderDir(dir, func(rel string, data []byte) error {
		visited = append(visited, rel)
		return nil
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"a.md", "sub/b.md"}, visited)
}

func TestWalkProviderDir_EmptyDir_NoError(t *testing.T) {
	dir := t.TempDir()
	var count int
	err := importer.WalkProviderDir(dir, func(rel string, data []byte) error {
		count++
		return nil
	})
	require.NoError(t, err)
	assert.Zero(t, count)
}

func TestWalkProviderDir_ReadsFileContent(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.md"), []byte("hello world"), 0o644))

	var content []byte
	err := importer.WalkProviderDir(dir, func(rel string, data []byte) error {
		content = data
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(content))
}
