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
	f := filepath.Join(dir, "coder.xcf")
	require.NoError(t, os.WriteFile(f, []byte(
		"---\nkind: agent\nname: coder\nversion: \"1.0\"\n---\nThis is the body.\n"), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	assert.Equal(t, "This is the body.", cfg.Agents["coder"].Instructions)
}

func TestParse_Frontmatter_NoDelimiter(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "coder.xcf")
	require.NoError(t, os.WriteFile(f, []byte("kind: agent\nname: coder\nversion: \"1.0\"\n"), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	_, ok := cfg.Agents["coder"]
	assert.True(t, ok)
}

func TestParse_Frontmatter_EmptyBody(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "coder.xcf")
	require.NoError(t, os.WriteFile(f, []byte("---\nkind: agent\nname: coder\nversion: \"1.0\"\n---\n"), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	assert.Equal(t, "", cfg.Agents["coder"].Instructions)
}

func TestParse_Frontmatter_WhitespaceBody(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "coder.xcf")
	require.NoError(t, os.WriteFile(f, []byte("---\nkind: agent\nname: coder\nversion: \"1.0\"\n---\n   \n   \n"), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	assert.Equal(t, "", cfg.Agents["coder"].Instructions)
}

func TestParse_Negative_MissingClosingDelimiter(t *testing.T) {
	// A file that starts with "---\n" followed by non-YAML markdown text (no
	// "kind:" line) signals intent to use frontmatter format but is missing the
	// closing "---" delimiter. This must be a parse error.
	dir := t.TempDir()
	f := filepath.Join(dir, "bad.xcf")
	require.NoError(t, os.WriteFile(f, []byte("---\nThis is just markdown, not YAML.\nNo closing delimiter.\n"), 0600))
	_, err := ParseFileExact(f)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "closing '---'")
}

func TestParse_Frontmatter_InstructionsFieldWins(t *testing.T) {
	// When instructions: is already set in the YAML, the body is silently discarded.
	dir := t.TempDir()
	f := filepath.Join(dir, "coder.xcf")
	require.NoError(t, os.WriteFile(f, []byte(
		"---\nkind: agent\nname: coder\nversion: \"1.0\"\ninstructions: \"from yaml\"\n---\nThis body is ignored.\n"), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	assert.Equal(t, "from yaml", cfg.Agents["coder"].Instructions)
}

func TestParse_Frontmatter_SkillBodyAsInstructions(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "my-skill.xcf")
	require.NoError(t, os.WriteFile(f, []byte(
		"---\nkind: skill\nname: my-skill\nversion: \"1.0\"\ndescription: \"A skill\"\n---\nSkill body content.\n"), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	assert.Equal(t, "Skill body content.", cfg.Skills["my-skill"].Instructions)
}

func TestParse_Frontmatter_RuleBodyAsInstructions(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "my-rule.xcf")
	require.NoError(t, os.WriteFile(f, []byte(
		"---\nkind: rule\nname: my-rule\nversion: \"1.0\"\n---\nRule body content.\n"), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	assert.Equal(t, "Rule body content.", cfg.Rules["my-rule"].Instructions)
}

func TestParse_Frontmatter_ReferenceBodyAsContent(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "my-ref.xcf")
	require.NoError(t, os.WriteFile(f, []byte(
		"---\nkind: reference\nname: my-ref\nversion: \"1.0\"\n---\nReference body content.\n"), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	assert.Equal(t, "Reference body content.", cfg.References["my-ref"].Content)
}

func TestParse_Frontmatter_SkillWithBody(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "tdd.xcf")
	content := "---\nkind: skill\nname: tdd\nversion: \"1.0\"\ndescription: TDD workflow\n---\n# Red-Green-Refactor\n\nWrite tests first.\n"
	require.NoError(t, os.WriteFile(f, []byte(content), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	sk, ok := cfg.Skills["tdd"]
	require.True(t, ok)
	assert.Equal(t, "# Red-Green-Refactor\n\nWrite tests first.", sk.Instructions)
}

func TestParse_Frontmatter_RuleWithBody(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "testing.xcf")
	content := "---\nkind: rule\nname: testing\nversion: \"1.0\"\n---\nAlways write tests before implementation.\n"
	require.NoError(t, os.WriteFile(f, []byte(content), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	r, ok := cfg.Rules["testing"]
	require.True(t, ok)
	assert.Equal(t, "Always write tests before implementation.", r.Instructions)
}

func TestParse_Frontmatter_ProjectWithBody(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "project.xcf")
	content := "---\nkind: project\nname: myapp\nversion: \"1.0\"\n---\nProject-level instructions here.\n"
	require.NoError(t, os.WriteFile(f, []byte(content), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	require.NotNil(t, cfg.Project)
	assert.Equal(t, "Project-level instructions here.", cfg.Project.Instructions)
}

func TestParse_Frontmatter_ReferenceWithBody(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "guide.xcf")
	content := "---\nkind: reference\nname: tdd-cheatsheet\nversion: \"1.0\"\ndescription: TDD patterns\n---\n## Red-Green-Refactor\n\nWrite tests first.\n"
	require.NoError(t, os.WriteFile(f, []byte(content), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	ref, ok := cfg.References["tdd-cheatsheet"]
	require.True(t, ok)
	assert.Equal(t, "## Red-Green-Refactor\n\nWrite tests first.", ref.Content)
	assert.Equal(t, "TDD patterns", ref.Description)
}

func TestParse_Frontmatter_BodyDoesNotOverrideYAMLInstructions(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "coder.xcf")
	content := "---\nkind: agent\nname: coder\nversion: \"1.0\"\ninstructions: from yaml\n---\nfrom body\n"
	require.NoError(t, os.WriteFile(f, []byte(content), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	assert.Equal(t, "from yaml", cfg.Agents["coder"].Instructions)
}

func TestParse_Frontmatter_SettingsKindWithBodyIgnored(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "settings.xcf")
	content := "---\nkind: settings\nversion: \"1.0\"\nmodel: sonnet-4\n---\nThis body should be ignored.\n"
	require.NoError(t, os.WriteFile(f, []byte(content), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	assert.Equal(t, "sonnet-4", cfg.Settings["default"].Model)
}

func TestParse_Frontmatter_KnownFieldsOnFrontmatterOnly(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "bad.xcf")
	content := "---\nkind: agent\nname: coder\nversion: \"1.0\"\nunknown-field: oops\n---\nbody text\n"
	require.NoError(t, os.WriteFile(f, []byte(content), 0600))
	_, err := ParseFileExact(f)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown-field")
}

func TestParse_Frontmatter_WorkflowWithBody(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "deploy.xcf")
	content := "---\nkind: workflow\nname: deploy\nversion: \"1.0\"\ndescription: Deployment workflow\n---\n1. Build the project.\n2. Run tests.\n3. Deploy.\n"
	require.NoError(t, os.WriteFile(f, []byte(content), 0600))
	cfg, err := ParseFileExact(f)
	require.NoError(t, err)
	wf, ok := cfg.Workflows["deploy"]
	require.True(t, ok)
	assert.Equal(t, "1. Build the project.\n2. Run tests.\n3. Deploy.", wf.Instructions)
}

func TestParse_Frontmatter_MultiDocumentDeprecationWarning(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "multi.xcf")
	content := "kind: agent\nname: agent-a\nversion: \"1.0\"\n---\nkind: skill\nname: skill-b\nversion: \"1.0\"\n"
	require.NoError(t, os.WriteFile(f, []byte(content), 0600))

	_, err := ParseFileExact(f)
	require.Error(t, err, "multi-document files must be rejected")
	assert.Contains(t, err.Error(), "no longer supported")
}

func TestParse_Frontmatter_MultiDocumentRejected(t *testing.T) {
	input := `kind: project
version: "1.0"
name: test
---
kind: agent
version: "1.0"
name: developer
instructions: "Hello."
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no longer supported")
	assert.Contains(t, err.Error(), "document 2")
}
