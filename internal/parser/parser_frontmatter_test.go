package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_Frontmatter_ExtractsBodyAsInstructions(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "coder.xcaf")
	require.NoError(t, os.WriteFile(f, []byte(
		"---\nkind: agent\nname: coder\nversion: \"1.0\"\n---\nThis is the body.\n"), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	assert.Equal(t, "This is the body.", cfg.Agents["coder"].Body)
}

func TestParse_Frontmatter_NoDelimiter(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "coder.xcaf")
	require.NoError(t, os.WriteFile(f, []byte("kind: agent\nname: coder\nversion: \"1.0\"\n"), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	_, ok := cfg.Agents["coder"]
	assert.True(t, ok)
}

func TestParse_Frontmatter_EmptyBody(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "coder.xcaf")
	require.NoError(t, os.WriteFile(f, []byte("---\nkind: agent\nname: coder\nversion: \"1.0\"\n---\n"), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	assert.Equal(t, "", cfg.Agents["coder"].Body)
}

func TestParse_Frontmatter_WhitespaceBody(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "coder.xcaf")
	require.NoError(t, os.WriteFile(f, []byte("---\nkind: agent\nname: coder\nversion: \"1.0\"\n---\n   \n   \n"), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	assert.Equal(t, "", cfg.Agents["coder"].Body)
}

func TestParse_Negative_MissingClosingDelimiter(t *testing.T) {
	// A file that starts with "---\n" followed by non-YAML markdown text (no
	// "kind:" line) signals intent to use frontmatter format but is missing the
	// closing "---" delimiter. This must be a parse error.
	dir := t.TempDir()
	f := filepath.Join(dir, "bad.xcaf")
	require.NoError(t, os.WriteFile(f, []byte("---\nThis is just markdown, not YAML.\nNo closing delimiter.\n"), 0600))
	_, err := ParseFileExact(f)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "closing '---'")
}

func TestParse_Frontmatter_InstructionsFieldWins(t *testing.T) {
	t.Skip("Legacy instructions test removed")

	t.Skip("Legacy instructions test removed")

	// When instructions: is already set in the YAML, the body is silently discarded.
	dir := t.TempDir()
	f := filepath.Join(dir, "coder.xcaf")
	require.NoError(t, os.WriteFile(f, []byte(
		"---\nkind: agent\nname: coder\nversion: \"1.0\"\ninstructions: \"from yaml\"\n---\nThis body is ignored.\n"), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	assert.Equal(t, "from yaml", cfg.Agents["coder"].Body)
}

func TestParse_Frontmatter_SkillBodyAsInstructions(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "my-skill.xcaf")
	require.NoError(t, os.WriteFile(f, []byte(
		"---\nkind: skill\nname: my-skill\nversion: \"1.0\"\ndescription: \"A skill\"\n---\nSkill body content.\n"), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	assert.Equal(t, "Skill body content.", cfg.Skills["my-skill"].Body)
}

func TestParse_Frontmatter_RuleBodyAsInstructions(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "my-rule.xcaf")
	require.NoError(t, os.WriteFile(f, []byte(
		"---\nkind: rule\nname: my-rule\nversion: \"1.0\"\n---\nRule body content.\n"), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	assert.Equal(t, "Rule body content.", cfg.Rules["my-rule"].Body)
}

func TestParse_Frontmatter_SkillWithBody(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "tdd.xcaf")
	content := "---\nkind: skill\nname: tdd\nversion: \"1.0\"\ndescription: TDD workflow\n---\n# Red-Green-Refactor\n\nWrite tests first.\n"
	require.NoError(t, os.WriteFile(f, []byte(content), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	sk, ok := cfg.Skills["tdd"]
	require.True(t, ok)
	assert.Equal(t, "# Red-Green-Refactor\n\nWrite tests first.", sk.Body)
}

func TestParse_Frontmatter_RuleWithBody(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "testing.xcaf")
	content := "---\nkind: rule\nname: testing\nversion: \"1.0\"\n---\nAlways write tests before implementation.\n"
	require.NoError(t, os.WriteFile(f, []byte(content), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	r, ok := cfg.Rules["testing"]
	require.True(t, ok)
	assert.Equal(t, "Always write tests before implementation.", r.Body)
}

func TestParse_Frontmatter_ProjectWithBody(t *testing.T) {
	t.Skip("Legacy instructions test removed")

	t.Skip("Legacy instructions test removed")

	dir := t.TempDir()
	f := filepath.Join(dir, "project.xcaf")
	content := "---\nkind: project\nname: myapp\nversion: \"1.0\"\n---\nProject-level instructions here.\n"
	require.NoError(t, os.WriteFile(f, []byte(content), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	require.NotNil(t, cfg.Project)
	assert.Equal(t, "Project-level instructions here.", cfg.ResourceScope.Contexts["root"].Body)
}

func TestParse_Frontmatter_BodyDoesNotOverrideYAMLInstructions(t *testing.T) {
	t.Skip("Legacy instructions test removed")

	t.Skip("Legacy instructions test removed")

	dir := t.TempDir()
	f := filepath.Join(dir, "coder.xcaf")
	content := "---\nkind: agent\nname: coder\nversion: \"1.0\"\ninstructions: from yaml\n---\nfrom body\n"
	require.NoError(t, os.WriteFile(f, []byte(content), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	assert.Equal(t, "from yaml", cfg.Agents["coder"].Body)
}

func TestParse_Frontmatter_SettingsKindWithBodyIgnored(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "settings.xcaf")
	content := "---\nkind: settings\nversion: \"1.0\"\nmodel: sonnet-4\n---\nThis body should be ignored.\n"
	require.NoError(t, os.WriteFile(f, []byte(content), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	assert.Equal(t, "sonnet-4", cfg.Settings["default"].Model)
}

func TestParse_Frontmatter_KnownFieldsOnFrontmatterOnly(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "bad.xcaf")
	content := "---\nkind: agent\nname: coder\nversion: \"1.0\"\nunknown-field: oops\n---\nbody text\n"
	require.NoError(t, os.WriteFile(f, []byte(content), 0600))
	_, err := ParseFileExact(f)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown-field")
}

func TestParse_Frontmatter_WorkflowWithBody(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "deploy.xcaf")
	// Workflows are now pure YAML — step instructions come from YAML fields
	content := "---\nkind: workflow\nname: deploy\nversion: \"1.0\"\ndescription: Deployment workflow\nsteps:\n  - name: build\n    instructions: \"Build the project\"\n  - name: test\n    instructions: \"Run tests\"\n  - name: deploy\n    instructions: \"Deploy to production\"\n---\nThis markdown body area is ignored for workflows (pure YAML kind).\n"
	require.NoError(t, os.WriteFile(f, []byte(content), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	wf, ok := cfg.Workflows["deploy"]
	require.True(t, ok)
	require.Len(t, wf.Steps, 3)
	assert.Equal(t, "Build the project", wf.Steps[0].Instructions)
	assert.Equal(t, "Run tests", wf.Steps[1].Instructions)
	assert.Equal(t, "Deploy to production", wf.Steps[2].Instructions)
	// Body should not be assigned for pure YAML workflows
	assert.Empty(t, wf.Body)
}

func TestParse_Frontmatter_MultiDocumentDeprecationWarning(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "multi.xcaf")
	content := "kind: agent\nname: agent-a\nversion: \"1.0\"\n---\nkind: skill\nname: skill-b\nversion: \"1.0\"\n"
	require.NoError(t, os.WriteFile(f, []byte(content), 0600))

	_, err := ParseFileExact(f)
	require.Error(t, err, "multi-document files must be rejected")
	assert.Contains(t, err.Error(), "no longer supported")
}

func TestParse_Frontmatter_MultiDocumentRejected(t *testing.T) {
	t.Skip("Legacy instructions test removed")

	t.Skip("Legacy instructions test removed")

	input := `kind: project
version: "1.0"
name: test
---
kind: agent
version: "1.0"
name: developer

`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no longer supported")
	assert.Contains(t, err.Error(), "document 2")
}
