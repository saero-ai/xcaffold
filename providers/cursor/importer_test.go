package cursor_test

import (
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	cursorimp "github.com/saero-ai/xcaffold/providers/cursor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saero-ai/xcaffold/internal/importer"
)

// --- Classify tests ---

func TestCursorClassify_AgentPattern(t *testing.T) {
	imp := cursorimp.NewImporter()
	kind, layout := imp.Classify("agents/reviewer.md", false)
	assert.Equal(t, importer.KindAgent, kind)
	assert.Equal(t, importer.FlatFile, layout)
}

func TestCursorClassify_SkillPattern(t *testing.T) {
	imp := cursorimp.NewImporter()
	kind, layout := imp.Classify("skills/code-review/SKILL.md", false)
	assert.Equal(t, importer.KindSkill, kind)
	assert.Equal(t, importer.DirectoryPerEntry, layout)
}

func TestCursorClassify_RulePattern(t *testing.T) {
	imp := cursorimp.NewImporter()
	kind, layout := imp.Classify("rules/formatting.mdc", false)
	assert.Equal(t, importer.KindRule, kind)
	assert.Equal(t, importer.FlatFile, layout)
}

func TestCursorClassify_RuleMdNotMatched(t *testing.T) {
	// Cursor rules use .mdc extension; plain .md should not match
	imp := cursorimp.NewImporter()
	kind, _ := imp.Classify("rules/formatting.md", false)
	assert.Equal(t, importer.KindUnknown, kind)
}

func TestCursorClassify_McpJson(t *testing.T) {
	imp := cursorimp.NewImporter()
	kind, layout := imp.Classify("mcp.json", false)
	assert.Equal(t, importer.KindMCP, kind)
	assert.Equal(t, importer.StandaloneJSON, layout)
}

func TestCursorClassify_HooksJson(t *testing.T) {
	imp := cursorimp.NewImporter()
	kind, layout := imp.Classify("hooks.json", false)
	assert.Equal(t, importer.KindHook, kind)
	assert.Equal(t, importer.StandaloneJSON, layout)
}

func TestCursorClassify_UnknownFile(t *testing.T) {
	imp := cursorimp.NewImporter()
	kind, layout := imp.Classify("some-unknown-file", false)
	assert.Equal(t, importer.KindUnknown, kind)
	assert.Equal(t, importer.LayoutUnknown, layout)
}

func TestCursorClassify_SkillReferenceFile(t *testing.T) {
	imp := cursorimp.NewImporter()
	kind, layout := imp.Classify("skills/code-review/references/guide.md", false)
	assert.Equal(t, importer.KindSkillAsset, kind)
	assert.Equal(t, importer.DirectoryPerEntry, layout)
}

func TestCursorClassify_SkillScriptFile(t *testing.T) {
	imp := cursorimp.NewImporter()
	kind, layout := imp.Classify("skills/code-review/scripts/helper.sh", false)
	assert.Equal(t, importer.KindSkillAsset, kind)
	assert.Equal(t, importer.DirectoryPerEntry, layout)
}

func TestCursorClassify_SkillAssetFile(t *testing.T) {
	imp := cursorimp.NewImporter()
	kind, layout := imp.Classify("skills/code-review/assets/icon.svg", false)
	assert.Equal(t, importer.KindSkillAsset, kind)
	assert.Equal(t, importer.DirectoryPerEntry, layout)
}

func TestCursorExtract_SkillCompanionFilePopulatesArtifacts(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Skills = map[string]ast.SkillConfig{
		"code-review": {Name: "code-review", SourceProvider: "cursor"},
	}
	imp := cursorimp.NewImporter()
	err := imp.Extract("skills/code-review/references/guide.md", []byte("# Guide"), config)
	require.NoError(t, err)
	skill := config.Skills["code-review"]
	assert.Contains(t, skill.Artifacts, "references")
}

func TestCursorImporter_Provider(t *testing.T) {
	assert.Equal(t, "cursor", cursorimp.NewImporter().Provider())
}

func TestCursorImporter_InputDir(t *testing.T) {
	assert.Equal(t, ".cursor", cursorimp.NewImporter().InputDir())
}

// --- Extract tests ---

func TestCursorExtract_AgentFrontmatter(t *testing.T) {
	data := []byte("---\nname: Code Reviewer\ndescription: Reviews PRs\nmodel: claude-opus-4-5\n---\n\nReview body.\n")
	config := &ast.XcaffoldConfig{}
	imp := cursorimp.NewImporter()
	err := imp.Extract("agents/reviewer.md", data, config)
	require.NoError(t, err)
	agent, ok := config.Agents["reviewer"]
	require.True(t, ok, "expected agent 'reviewer' in config")
	assert.Equal(t, "Code Reviewer", agent.Name)
	assert.Equal(t, "Reviews PRs", agent.Description)
	assert.Equal(t, "claude-opus-4-5", agent.Model)
	assert.Contains(t, agent.Body, "Review body.")
	assert.Equal(t, "cursor", agent.SourceProvider)
}

func TestCursorExtract_RuleFrontmatterWithGlobs(t *testing.T) {
	data := []byte("---\ndescription: Formatting standards\nglobs:\n  - \"**/*.go\"\n  - \"**/*.ts\"\n---\n\nUse gofmt for Go files.\n")
	config := &ast.XcaffoldConfig{}
	imp := cursorimp.NewImporter()
	err := imp.Extract("rules/formatting.mdc", data, config)
	require.NoError(t, err)
	rule, ok := config.Rules["formatting"]
	require.True(t, ok, "expected rule 'formatting' in config")
	assert.Equal(t, "Formatting standards", rule.Description)
	assert.Contains(t, rule.Body, "Use gofmt for Go files.")
	assert.Equal(t, ast.ClearableList{Values: []string{"**/*.go", "**/*.ts"}}, rule.Paths)
	assert.Equal(t, "cursor", rule.SourceProvider)
}

func TestCursorExtract_RuleFrontmatterWithoutGlobs(t *testing.T) {
	data := []byte("---\ndescription: No globs rule\n---\n\nRule body.\n")
	config := &ast.XcaffoldConfig{}
	imp := cursorimp.NewImporter()
	err := imp.Extract("rules/no-globs.mdc", data, config)
	require.NoError(t, err)
	rule, ok := config.Rules["no-globs"]
	require.True(t, ok, "expected rule 'no-globs' in config")
	assert.Empty(t, rule.Paths.Values)
	assert.Equal(t, "cursor", rule.SourceProvider)
}

func TestCursorExtract_SkillFrontmatter(t *testing.T) {
	data := []byte("---\nname: code-review\ndescription: Code review workflow\nallowed-tools:\n  - Read\n  - Grep\n---\n\nReview code systematically.\n")
	config := &ast.XcaffoldConfig{}
	imp := cursorimp.NewImporter()
	err := imp.Extract("skills/code-review/SKILL.md", data, config)
	require.NoError(t, err)
	skill, ok := config.Skills["code-review"]
	require.True(t, ok, "expected skill 'code-review' in config")
	assert.Equal(t, "code-review", skill.Name)
	assert.Contains(t, skill.Body, "Review code systematically.")
	assert.Equal(t, "cursor", skill.SourceProvider)
}

func TestCursorExtract_McpStandaloneJSON(t *testing.T) {
	data := []byte(`{"mcpServers":{"github":{"type":"stdio","command":"gh-mcp","args":["serve"]}}}`)
	config := &ast.XcaffoldConfig{}
	imp := cursorimp.NewImporter()
	err := imp.Extract("mcp.json", data, config)
	require.NoError(t, err)
	mc, ok := config.MCP["github"]
	require.True(t, ok, "expected mcp server 'github'")
	assert.Equal(t, "stdio", mc.Type)
	assert.Equal(t, "gh-mcp", mc.Command)
	assert.Equal(t, "cursor", mc.SourceProvider)
}

func TestCursorExtract_HooksStandaloneJSON(t *testing.T) {
	data := []byte(`{
		"PreToolUse": [
			{"matcher": "Bash", "hooks": [{"type": "command", "command": "echo pre"}]}
		]
	}`)
	config := &ast.XcaffoldConfig{}
	imp := cursorimp.NewImporter()
	err := imp.Extract("hooks.json", data, config)
	require.NoError(t, err)
	var effectiveHooks ast.HookConfig
	if dh, ok2 := config.Hooks["default"]; ok2 {
		effectiveHooks = dh.Events
	}
	hooks, ok := effectiveHooks["PreToolUse"]
	require.True(t, ok, "expected PreToolUse hooks")
	require.Len(t, hooks, 1)
	assert.Equal(t, "Bash", hooks[0].Matcher)
}

func TestCursorExtract_ExtrasCollection(t *testing.T) {
	imp := cursorimp.NewImporter()
	config := &ast.XcaffoldConfig{}
	inputDir := filepath.Join("testdata", "input")
	err := imp.Import(inputDir, config)
	require.NoError(t, err)
	// All known files should be classified
	assert.NotEmpty(t, config.Agents)
	assert.NotEmpty(t, config.Skills)
	assert.NotEmpty(t, config.Rules)
	// Unknown files are skipped by RunImport; only extraction errors appear in ProviderExtras
	// Verify that there are no errors (all files parsed successfully)
	assert.True(t, len(imp.GetWarnings()) == 0, "expected no extraction warnings")
}

// --- Full workspace golden test ---

func TestCursorImporter_FullWorkspace(t *testing.T) {
	imp := cursorimp.NewImporter()
	config := &ast.XcaffoldConfig{}
	inputDir := filepath.Join("testdata", "input")
	err := imp.Import(inputDir, config)
	require.NoError(t, err)

	// Agents
	reviewer, ok := config.Agents["reviewer"]
	require.True(t, ok, "expected agent 'reviewer'")
	assert.Equal(t, "Code Reviewer", reviewer.Name)
	assert.Equal(t, "cursor", reviewer.SourceProvider)
	assert.NotEmpty(t, reviewer.Body)

	// Skills
	cr, ok := config.Skills["code-review"]
	require.True(t, ok, "expected skill 'code-review'")
	assert.Equal(t, "code-review", cr.Name)
	assert.Equal(t, "cursor", cr.SourceProvider)

	// Rules
	fmt, ok := config.Rules["formatting"]
	require.True(t, ok, "expected rule 'formatting'")
	assert.Equal(t, "cursor", fmt.SourceProvider)
	// globs: field maps to Paths
	assert.Equal(t, ast.ClearableList{Values: []string{"**/*.go", "**/*.ts"}}, fmt.Paths)

	// Hooks from hooks.json (standalone)
	assert.NotEmpty(t, config.Hooks["default"].Events["PreToolUse"])

	// MCP from mcp.json
	_, ok = config.MCP["github"]
	require.True(t, ok, "expected mcp server 'github' from mcp.json")

	// Unknown files are skipped by RunImport; verify no extraction errors
	assert.True(t, len(imp.GetWarnings()) == 0, "expected no extraction warnings")
}
