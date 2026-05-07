package importer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
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

func TestWalkProviderDir_SymlinkToExternalDir(t *testing.T) {
	// Simulate a symlink pointing to a directory outside the walk root,
	// e.g. .claude/skills/shared-skill -> /some/other/path/shared-skill
	root := t.TempDir()
	external := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(root, "skills", "local-skill"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "skills", "local-skill", "SKILL.md"), []byte("local"), 0o644))

	require.NoError(t, os.MkdirAll(filepath.Join(external, "shared-skill"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(external, "shared-skill", "SKILL.md"), []byte("shared"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(external, "shared-skill", "ref.md"), []byte("reference"), 0o644))

	require.NoError(t, os.Symlink(filepath.Join(external, "shared-skill"), filepath.Join(root, "skills", "shared-skill")))

	var visited []string
	contents := make(map[string]string)
	err := importer.WalkProviderDir(root, func(rel string, data []byte) error {
		visited = append(visited, rel)
		contents[rel] = string(data)
		return nil
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"skills/local-skill/SKILL.md",
		"skills/shared-skill/SKILL.md",
		"skills/shared-skill/ref.md",
	}, visited)
	assert.Equal(t, "shared", contents["skills/shared-skill/SKILL.md"])
	assert.Equal(t, "reference", contents["skills/shared-skill/ref.md"])
}

func TestExtractHookScript(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	err := importer.ExtractHookScript("hooks/test.sh", []byte("echo hello"), config)
	if err != nil {
		t.Fatalf("ExtractHookScript failed: %v", err)
	}

	if config.ProviderExtras == nil || config.ProviderExtras["xcaf"] == nil {
		t.Fatalf("ProviderExtras[\"xcaf\"] not initialized")
	}

	data, ok := config.ProviderExtras["xcaf"]["hooks/test.sh"]
	if !ok {
		t.Fatalf("hook script not found in ProviderExtras")
	}

	if string(data) != "echo hello" {
		t.Errorf("expected 'echo hello', got %s", string(data))
	}
}

func TestDefaultExtractRule_ParsesFrontmatter(t *testing.T) {
	input := []byte("---\nname: secure-coding\ndescription: Security standards\nalways-apply: true\n---\nNo secrets in plaintext.")
	config := &ast.XcaffoldConfig{}

	err := importer.DefaultExtractRule("rules/secure-coding.md", input, "test-provider", config)
	require.NoError(t, err)

	rule, ok := config.Rules["secure-coding"]
	require.True(t, ok, "rule secure-coding not found")
	assert.Equal(t, "secure-coding", rule.Name)
	assert.Equal(t, "Security standards", rule.Description)
	assert.True(t, *rule.AlwaysApply)
	assert.Equal(t, "No secrets in plaintext.", rule.Body)
	assert.Equal(t, "test-provider", rule.SourceProvider)
}

func TestDefaultExtractRule_NestedPaths(t *testing.T) {
	input := []byte("---\nname: cli-testing\n---\nTest rules for CLI.")
	config := &ast.XcaffoldConfig{}

	err := importer.DefaultExtractRule("rules/cli/testing.md", input, "test-provider", config)
	require.NoError(t, err)

	rule, ok := config.Rules["cli/testing"]
	require.True(t, ok, "rule cli/testing not found")
	assert.Equal(t, "cli-testing", rule.Name)
}

func TestDefaultExtractRule_WithPaths(t *testing.T) {
	input := []byte("---\nname: test-rule\npaths:\n  - \"*.go\"\n  - \"**/*.md\"\n---\nBody.")
	config := &ast.XcaffoldConfig{}

	err := importer.DefaultExtractRule("rules/test-rule.md", input, "test-provider", config)
	require.NoError(t, err)

	rule, ok := config.Rules["test-rule"]
	require.True(t, ok)
	assert.Equal(t, []string{"*.go", "**/*.md"}, rule.Paths.Values)
}

func TestDefaultExtractSkillAsset_References(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Skills = make(map[string]ast.SkillConfig)
	config.Skills["tdd"] = ast.SkillConfig{Name: "tdd"}

	err := importer.DefaultExtractSkillAsset("skills/tdd/references/guide.md", []byte("guide"), config)
	require.NoError(t, err)

	skill := config.Skills["tdd"]
	assert.Equal(t, []string{"xcf/skills/tdd/references/guide.md"}, skill.References.Values)
}

func TestDefaultExtractSkillAsset_Scripts(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Skills = make(map[string]ast.SkillConfig)
	config.Skills["tdd"] = ast.SkillConfig{Name: "tdd"}

	err := importer.DefaultExtractSkillAsset("skills/tdd/scripts/helper.sh", []byte("script"), config)
	require.NoError(t, err)

	skill := config.Skills["tdd"]
	assert.Equal(t, []string{"xcf/skills/tdd/scripts/helper.sh"}, skill.Scripts.Values)
}

func TestDefaultExtractSkillAsset_Assets(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Skills = make(map[string]ast.SkillConfig)
	config.Skills["tdd"] = ast.SkillConfig{Name: "tdd"}

	err := importer.DefaultExtractSkillAsset("skills/tdd/assets/data.json", []byte("{}"), config)
	require.NoError(t, err)

	skill := config.Skills["tdd"]
	assert.Equal(t, []string{"xcf/skills/tdd/assets/data.json"}, skill.Assets.Values)
}

func TestDefaultExtractSkillAsset_AppendsUnique(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Skills = make(map[string]ast.SkillConfig)
	config.Skills["tdd"] = ast.SkillConfig{
		Name:       "tdd",
		References: ast.ClearableList{Values: []string{"xcf/skills/tdd/references/existing.md"}},
	}
	err := importer.DefaultExtractSkillAsset("skills/tdd/references/new.md", []byte("new"), config)
	require.NoError(t, err)

	skill := config.Skills["tdd"]
	assert.Equal(t, []string{"xcf/skills/tdd/references/existing.md", "xcf/skills/tdd/references/new.md"}, skill.References.Values)
}

func TestDefaultExtractSkillAsset_InvalidPath(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	err := importer.DefaultExtractSkillAsset("skills/tdd", []byte(""), config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too short")
}

func TestDefaultExtractSkillAsset_UnknownSubdir(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Skills = make(map[string]ast.SkillConfig)
	config.Skills["tdd"] = ast.SkillConfig{Name: "tdd"}

	err := importer.DefaultExtractSkillAsset("skills/tdd/unknown/file.txt", []byte(""), config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown subdirectory")
}

func TestDefaultExtractHookScript_DelegatesToExtractHookScript(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	err := importer.DefaultExtractHookScript("hooks/test.sh", []byte("echo test"), config)
	require.NoError(t, err)

	data, ok := config.ProviderExtras["xcaf"]["hooks/test.sh"]
	require.True(t, ok)
	assert.Equal(t, "echo test", string(data))
}
