package claude_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	claudeimp "github.com/saero-ai/xcaffold/internal/importer/claude"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saero-ai/xcaffold/internal/importer"
)

// --- Classify tests ---

func TestClaudeClassify_AgentPattern(t *testing.T) {
	imp := claudeimp.New()
	kind, layout := imp.Classify("agents/ceo.md", false)
	assert.Equal(t, importer.KindAgent, kind)
	assert.Equal(t, importer.FlatFile, layout)
}

func TestClaudeClassify_SkillPattern(t *testing.T) {
	imp := claudeimp.New()
	kind, layout := imp.Classify("skills/tdd/SKILL.md", false)
	assert.Equal(t, importer.KindSkill, kind)
	assert.Equal(t, importer.DirectoryPerEntry, layout)
}

func TestClaudeClassify_RulePattern(t *testing.T) {
	imp := claudeimp.New()
	kind, layout := imp.Classify("rules/security.md", false)
	assert.Equal(t, importer.KindRule, kind)
	assert.Equal(t, importer.FlatFile, layout)
}

func TestClaudeClassify_SettingsFile(t *testing.T) {
	imp := claudeimp.New()
	// settings.json returns KindSettings (first match in mapping table)
	kind, layout := imp.Classify("settings.json", false)
	assert.Equal(t, importer.KindSettings, kind)
	assert.Equal(t, importer.EmbeddedJSONKey, layout)
}

func TestClaudeClassify_McpJson(t *testing.T) {
	imp := claudeimp.New()
	kind, layout := imp.Classify("mcp.json", false)
	assert.Equal(t, importer.KindMCP, kind)
	assert.Equal(t, importer.StandaloneJSON, layout)
}

func TestClaudeClassify_UnknownFile(t *testing.T) {
	imp := claudeimp.New()
	kind, layout := imp.Classify("statusline", false)
	assert.Equal(t, importer.KindUnknown, kind)
	assert.Equal(t, importer.LayoutUnknown, layout)
}

func TestClaudeClassify_MemoryFile(t *testing.T) {
	imp := claudeimp.New()
	kind, layout := imp.Classify("agent-memory/user.md", false)
	assert.Equal(t, importer.KindMemory, kind)
	assert.Equal(t, importer.FlatFile, layout)
}

func TestClaudeClassify_MemoryNestedFile(t *testing.T) {
	imp := claudeimp.New()
	kind, layout := imp.Classify("agent-memory/go-cli-developer/context.md", false)
	assert.Equal(t, importer.KindMemory, kind)
	assert.Equal(t, importer.FlatFile, layout)
}

func TestClaudeClassify_SkillReferenceFile(t *testing.T) {
	imp := claudeimp.New()
	kind, layout := imp.Classify("skills/tdd/references/schema.sql", false)
	assert.Equal(t, importer.KindSkillAsset, kind)
	assert.Equal(t, importer.DirectoryPerEntry, layout)
}

func TestClaudeClassify_SkillScriptFile(t *testing.T) {
	imp := claudeimp.New()
	kind, layout := imp.Classify("skills/tdd/scripts/run.sh", false)
	assert.Equal(t, importer.KindSkillAsset, kind)
	assert.Equal(t, importer.DirectoryPerEntry, layout)
}

func TestClaudeClassify_SkillAssetFile(t *testing.T) {
	imp := claudeimp.New()
	kind, layout := imp.Classify("skills/tdd/assets/template.md", false)
	assert.Equal(t, importer.KindSkillAsset, kind)
	assert.Equal(t, importer.DirectoryPerEntry, layout)
}

func TestClaudeClassify_SkillNestedReferenceFile(t *testing.T) {
	imp := claudeimp.New()
	kind, _ := imp.Classify("skills/tdd/references/subdir/deep.txt", false)
	assert.Equal(t, importer.KindSkillAsset, kind)
}

func TestClaudeExtract_SkillCompanionFilePopulatesReferences(t *testing.T) {
	// Pre-populate the skill so the companion extractor can find it.
	config := &ast.XcaffoldConfig{}
	config.Skills = map[string]ast.SkillConfig{
		"tdd": {Name: "tdd-driven-development", SourceProvider: "claude"},
	}
	imp := claudeimp.New()
	err := imp.Extract("skills/tdd/references/schema.sql", []byte("CREATE TABLE t (id INT);"), config)
	require.NoError(t, err)
	skill := config.Skills["tdd"]
	assert.Contains(t, skill.References, "references/schema.sql")
}

func TestClaudeExtract_SkillCompanionFilePopulatesScripts(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Skills = map[string]ast.SkillConfig{
		"tdd": {Name: "tdd-driven-development", SourceProvider: "claude"},
	}
	imp := claudeimp.New()
	err := imp.Extract("skills/tdd/scripts/run.sh", []byte("#!/bin/bash\necho ok"), config)
	require.NoError(t, err)
	skill := config.Skills["tdd"]
	assert.Contains(t, skill.Scripts, "scripts/run.sh")
}

func TestClaudeExtract_SkillCompanionFilePopulatesAssets(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Skills = map[string]ast.SkillConfig{
		"tdd": {Name: "tdd-driven-development", SourceProvider: "claude"},
	}
	imp := claudeimp.New()
	err := imp.Extract("skills/tdd/assets/template.md", []byte("# Template"), config)
	require.NoError(t, err)
	skill := config.Skills["tdd"]
	assert.Contains(t, skill.Assets, "assets/template.md")
}

func TestClaudeImport_SkillCompanionFilesNotInExtras(t *testing.T) {
	// After import, companion files must NOT appear in ProviderExtras.
	imp := claudeimp.New()
	config := &ast.XcaffoldConfig{}
	inputDir := filepath.Join("testdata", "input")
	err := imp.Import(inputDir, config)
	require.NoError(t, err)
	extras := config.ProviderExtras["claude"]
	for k := range extras {
		assert.False(t, strings.HasPrefix(k, "skills/tdd/references/"),
			"skill reference file %q must not appear in ProviderExtras", k)
		assert.False(t, strings.HasPrefix(k, "skills/tdd/scripts/"),
			"skill script file %q must not appear in ProviderExtras", k)
		assert.False(t, strings.HasPrefix(k, "skills/tdd/assets/"),
			"skill asset file %q must not appear in ProviderExtras", k)
	}
}

func TestClaudeImporter_Provider(t *testing.T) {
	assert.Equal(t, "claude", claudeimp.New().Provider())
}

func TestClaudeImporter_InputDir(t *testing.T) {
	assert.Equal(t, ".claude", claudeimp.New().InputDir())
}

// --- Extract tests ---

func TestClaudeExtract_AgentFrontmatter(t *testing.T) {
	data := []byte("---\nname: CEO\ndescription: Chief Executive\nmodel: claude-opus-4-5\n---\n\nSystem prompt body.\n")
	config := &ast.XcaffoldConfig{}
	imp := claudeimp.New()
	err := imp.Extract("agents/ceo.md", data, config)
	require.NoError(t, err)
	agent, ok := config.Agents["ceo"]
	require.True(t, ok, "expected agent 'ceo' in config")
	assert.Equal(t, "CEO", agent.Name)
	assert.Equal(t, "Chief Executive", agent.Description)
	assert.Equal(t, "claude-opus-4-5", agent.Model)
	assert.Contains(t, agent.Instructions, "System prompt body.")
	assert.Equal(t, "claude", agent.SourceProvider)
}

func TestClaudeExtract_RuleFrontmatter(t *testing.T) {
	data := []byte("---\ndescription: Testing standards\n---\n\nAlways write tests first.\n")
	config := &ast.XcaffoldConfig{}
	imp := claudeimp.New()
	err := imp.Extract("rules/testing.md", data, config)
	require.NoError(t, err)
	rule, ok := config.Rules["testing"]
	require.True(t, ok, "expected rule 'testing' in config")
	assert.Equal(t, "Testing standards", rule.Description)
	assert.Contains(t, rule.Instructions, "Always write tests first.")
	assert.Equal(t, "claude", rule.SourceProvider)
}

func TestClaudeExtract_McpStandaloneJSON(t *testing.T) {
	data := []byte(`{"mcpServers":{"github":{"type":"stdio","command":"gh-mcp","args":["serve"]}}}`)
	config := &ast.XcaffoldConfig{}
	imp := claudeimp.New()
	err := imp.Extract("mcp.json", data, config)
	require.NoError(t, err)
	mc, ok := config.MCP["github"]
	require.True(t, ok, "expected mcp server 'github'")
	assert.Equal(t, "stdio", mc.Type)
	assert.Equal(t, "gh-mcp", mc.Command)
	assert.Equal(t, "claude", mc.SourceProvider)
}

func TestClaudeExtract_SettingsDecomposition(t *testing.T) {
	data := []byte(`{
		"model": "claude-opus-4-5",
		"hooks": {
			"PreToolUse": [
				{"matcher": "Bash", "hooks": [{"type": "command", "command": "echo pre"}]}
			]
		},
		"mcpServers": {
			"local-tool": {"type": "stdio", "command": "my-tool"}
		}
	}`)
	config := &ast.XcaffoldConfig{}
	imp := claudeimp.New()
	err := imp.Extract("settings.json", data, config)
	require.NoError(t, err)
	// Hooks extracted
	assert.NotNil(t, config.Hooks["PreToolUse"])
	assert.Len(t, config.Hooks["PreToolUse"], 1)
	// MCP from settings.json mcpServers extracted
	_, ok := config.MCP["local-tool"]
	require.True(t, ok, "expected mcp 'local-tool' from settings.json")
	// Settings model extracted
	assert.Equal(t, "claude-opus-4-5", config.Settings.Model)
	assert.Equal(t, "claude", config.Settings.SourceProvider)
}

func TestClaudeExtract_ExtrasCollection(t *testing.T) {
	imp := claudeimp.New()
	config := &ast.XcaffoldConfig{}
	inputDir := filepath.Join("testdata", "input")
	err := imp.Import(inputDir, config)
	require.NoError(t, err)
	// All known files should be classified
	assert.NotEmpty(t, config.Agents)
	assert.NotEmpty(t, config.Skills)
	assert.NotEmpty(t, config.Rules)
	// statusline is unknown — should appear in ProviderExtras
	extras, hasProvider := config.ProviderExtras["claude"]
	require.True(t, hasProvider, "expected ProviderExtras to have 'claude' entry")
	_, hasStatusline := extras["statusline"]
	assert.True(t, hasStatusline, "expected 'statusline' in ProviderExtras")
}

// --- Nested rules tests ---

func TestClaudeClassify_NestedRulePattern(t *testing.T) {
	imp := claudeimp.New()
	kind, layout := imp.Classify("rules/cli/testing-framework.md", false)
	assert.Equal(t, importer.KindRule, kind, "nested rule should classify as KindRule")
	assert.Equal(t, importer.FlatFile, layout)
}

func TestClaudeClassify_NestedRuleDeepPattern(t *testing.T) {
	imp := claudeimp.New()
	kind, layout := imp.Classify("rules/platform/infrastructure.md", false)
	assert.Equal(t, importer.KindRule, kind, "two-level nested rule should classify as KindRule")
	assert.Equal(t, importer.FlatFile, layout)
}

func TestClaudeExtract_NestedRuleID(t *testing.T) {
	// A rule at rules/cli/testing-framework.md should get ID "cli/testing-framework",
	// not just "testing-framework" (which would collide with rules/testing-framework.md).
	data := []byte("---\ndescription: CLI testing standards\n---\n\nAlways write table-driven tests.\n")
	config := &ast.XcaffoldConfig{}
	imp := claudeimp.New()
	err := imp.Extract("rules/cli/testing-framework.md", data, config)
	require.NoError(t, err)
	_, flat := config.Rules["testing-framework"]
	assert.False(t, flat, "flat ID 'testing-framework' should NOT be used for nested rules")
	rule, nested := config.Rules["cli/testing-framework"]
	require.True(t, nested, "expected rule ID 'cli/testing-framework' for nested rule")
	assert.Equal(t, "CLI testing standards", rule.Description)
	assert.Equal(t, "claude", rule.SourceProvider)
}

// --- Bad frontmatter resilience tests ---

func TestClaudeParseFrontmatter_BadYAMLFallback(t *testing.T) {
	// YAML with a backtick-quoted value containing ': ' triggers "mapping values
	// are not allowed in this context". parseFrontmatter skips the unparseable
	// metadata and returns the body after the closing ---. The rule is registered
	// with zero-value metadata but valid instructions.
	data := []byte("---\ndescription: Do NOT use `isolation: worktree` in agents\n---\n\nBody text.\n")
	config := &ast.XcaffoldConfig{}
	imp := claudeimp.New()
	err := imp.Extract("rules/bad-frontmatter.md", data, config)
	require.NoError(t, err, "Extract must not fail — body is valid even if metadata is unparseable")
	rule, ok := config.Rules["bad-frontmatter"]
	require.True(t, ok, "rule must be registered with body content")
	assert.Equal(t, "Body text.", rule.Instructions)
	assert.Empty(t, rule.Description, "metadata should be zero-valued when frontmatter fails")
}

func TestClaudeImport_BadFrontmatterContinues(t *testing.T) {
	// Import must not abort when one file has bad YAML frontmatter.
	// All other files in the workspace must still be processed.
	imp := claudeimp.New()
	config := &ast.XcaffoldConfig{}
	inputDir := filepath.Join("testdata", "input")
	err := imp.Import(inputDir, config)
	require.NoError(t, err, "Import must not return error for bad-frontmatter.md")
	// skills must have been discovered (walked AFTER rules/ alphabetically).
	_, ok := config.Skills["tdd"]
	assert.True(t, ok, "skills must be imported even when a rule file has bad frontmatter")
	// The bad-frontmatter file should be registered as a rule with body content.
	rule, inRules := config.Rules["bad-frontmatter"]
	assert.True(t, inRules, "bad-frontmatter must be registered as a rule with body content")
	assert.NotEmpty(t, rule.Instructions)
}

func TestClaudeImport_NestedRulesDiscovered(t *testing.T) {
	// Import must discover rules in subdirectories (e.g. rules/cli/*.md).
	imp := claudeimp.New()
	config := &ast.XcaffoldConfig{}
	inputDir := filepath.Join("testdata", "input")
	err := imp.Import(inputDir, config)
	require.NoError(t, err)
	_, ok := config.Rules["cli/testing-framework"]
	assert.True(t, ok, "nested rule rules/cli/testing-framework.md must be imported with ID 'cli/testing-framework'")
}

// --- Full workspace golden test ---

func TestClaudeImporter_FullWorkspace(t *testing.T) {
	imp := claudeimp.New()
	config := &ast.XcaffoldConfig{}
	inputDir := filepath.Join("testdata", "input")
	err := imp.Import(inputDir, config)
	require.NoError(t, err)

	// Agents
	ceo, ok := config.Agents["backend"]
	require.True(t, ok, "expected agent 'backend'")
	assert.Equal(t, "Backend Agent", ceo.Name)
	assert.Equal(t, "claude", ceo.SourceProvider)
	assert.NotEmpty(t, ceo.Instructions)

	// Skills
	tdd, ok := config.Skills["tdd"]
	require.True(t, ok, "expected skill 'tdd'")
	assert.Equal(t, "tdd-driven-development", tdd.Name)
	assert.Equal(t, "claude", tdd.SourceProvider)

	// Rules — flat and nested
	sec, ok := config.Rules["security"]
	require.True(t, ok, "expected rule 'security'")
	assert.Equal(t, "claude", sec.SourceProvider)
	_, ok = config.Rules["cli/testing-framework"]
	assert.True(t, ok, "expected nested rule 'cli/testing-framework'")

	// Hooks from settings.json
	assert.NotEmpty(t, config.Hooks["PreToolUse"])

	// MCP from mcp.json
	_, ok = config.MCP["github"]
	require.True(t, ok, "expected mcp server 'github' from mcp.json")

	// Memory
	assert.NotEmpty(t, config.Memory)

	// Settings
	assert.Equal(t, "claude-opus-4-5", config.Settings.Model)

	// Unknown file goes to extras
	assert.NotEmpty(t, config.ProviderExtras["claude"])
}
