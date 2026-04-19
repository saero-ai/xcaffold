package claude_test

import (
	"path/filepath"
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

func TestClaudeClassify_SkillAssetNotRoot(t *testing.T) {
	// skills/tdd/references/schema.sql is inside a skill dir — not a root-level pattern
	// Skill subdirectory files are handled by Extract(), not Classify() root-level patterns.
	imp := claudeimp.New()
	kind, _ := imp.Classify("skills/tdd/references/schema.sql", false)
	assert.Equal(t, importer.KindUnknown, kind)
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

	// Rules
	sec, ok := config.Rules["security"]
	require.True(t, ok, "expected rule 'security'")
	assert.Equal(t, "claude", sec.SourceProvider)

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
