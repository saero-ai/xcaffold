package gemini_test

import (
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	geminimp "github.com/saero-ai/xcaffold/providers/gemini"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saero-ai/xcaffold/internal/importer"
)

// --- Classify tests ---

func TestGeminiClassify_AgentPattern(t *testing.T) {
	imp := geminimp.NewImporter()
	kind, layout := imp.Classify("agents/assistant.md", false)
	assert.Equal(t, importer.KindAgent, kind)
	assert.Equal(t, importer.FlatFile, layout)
}

func TestGeminiClassify_SkillPattern_DirectoryPerEntry(t *testing.T) {
	// Gemini skills use directory-per-entry layout
	imp := geminimp.NewImporter()
	kind, layout := imp.Classify("skills/search/SKILL.md", false)
	assert.Equal(t, importer.KindSkill, kind)
	assert.Equal(t, importer.DirectoryPerEntry, layout)
}

func TestGeminiClassify_SkillReferences(t *testing.T) {
	imp := geminimp.NewImporter()
	// Skill asset files in references/ subdirectory
	kind, layout := imp.Classify("skills/search/references/example.md", false)
	assert.Equal(t, importer.KindSkillAsset, kind)
	assert.Equal(t, importer.DirectoryPerEntry, layout)
}

func TestGeminiClassify_SkillScripts(t *testing.T) {
	imp := geminimp.NewImporter()
	// Skill asset files in scripts/ subdirectory
	kind, layout := imp.Classify("skills/search/scripts/setup.sh", false)
	assert.Equal(t, importer.KindSkillAsset, kind)
	assert.Equal(t, importer.DirectoryPerEntry, layout)
}

func TestGeminiClassify_SkillAssets(t *testing.T) {
	imp := geminimp.NewImporter()
	// Skill asset files in assets/ subdirectory (Gemini-native)
	kind, layout := imp.Classify("skills/search/assets/icon.png", false)
	assert.Equal(t, importer.KindSkillAsset, kind)
	assert.Equal(t, importer.DirectoryPerEntry, layout)
}

func TestGeminiClassify_RulePattern(t *testing.T) {
	imp := geminimp.NewImporter()
	kind, layout := imp.Classify("rules/style.md", false)
	assert.Equal(t, importer.KindRule, kind)
	assert.Equal(t, importer.FlatFile, layout)
}

func TestGeminiClassify_SettingsFile(t *testing.T) {
	imp := geminimp.NewImporter()
	kind, layout := imp.Classify("settings.json", false)
	assert.Equal(t, importer.KindSettings, kind)
	assert.Equal(t, importer.EmbeddedJSONKey, layout)
}

func TestGeminiClassify_UnknownFile(t *testing.T) {
	imp := geminimp.NewImporter()
	kind, layout := imp.Classify("unknown-file", false)
	assert.Equal(t, importer.KindUnknown, kind)
	assert.Equal(t, importer.LayoutUnknown, layout)
}

func TestGeminiClassify_SkillFlatFileNotMatched(t *testing.T) {
	// Gemini flat file patterns no longer match — only directory-per-entry skills/* / SKILL.md
	imp := geminimp.NewImporter()
	kind, _ := imp.Classify("skills/search.md", false)
	assert.Equal(t, importer.KindUnknown, kind)
}

func TestGeminiImporter_Provider(t *testing.T) {
	assert.Equal(t, "gemini", geminimp.NewImporter().Provider())
}

func TestGeminiImporter_InputDir(t *testing.T) {
	assert.Equal(t, ".gemini", geminimp.NewImporter().InputDir())
}

// --- Extract tests ---

func TestGeminiExtract_AgentFrontmatter(t *testing.T) {
	data := []byte("---\nname: Assistant Agent\ndescription: General helper\nmodel: gemini-2.5-pro\n---\n\nYou are a helpful assistant.\n")
	config := &ast.XcaffoldConfig{}
	imp := geminimp.NewImporter()
	err := imp.Extract("agents/assistant.md", data, config)
	require.NoError(t, err)
	agent, ok := config.Agents["assistant"]
	require.True(t, ok, "expected agent 'assistant' in config")
	assert.Equal(t, "Assistant Agent", agent.Name)
	assert.Equal(t, "General helper", agent.Description)
	assert.Equal(t, "gemini-2.5-pro", agent.Model)
	assert.Contains(t, agent.Body, "You are a helpful assistant.")
	assert.Equal(t, "gemini", agent.SourceProvider)
}

func TestGeminiExtract_SkillFrontmatter(t *testing.T) {
	// Gemini skill: id is the directory name, SKILL.md contains frontmatter
	data := []byte("---\nname: web-search\ndescription: Search the web\nallowed-tools:\n  - WebSearch\n---\n\nUse for external lookups.\n")
	config := &ast.XcaffoldConfig{}
	imp := geminimp.NewImporter()
	err := imp.Extract("skills/search/SKILL.md", data, config)
	require.NoError(t, err)
	skill, ok := config.Skills["search"]
	require.True(t, ok, "expected skill 'search' in config")
	assert.Equal(t, "web-search", skill.Name)
	assert.Equal(t, []string{"WebSearch"}, skill.AllowedTools)
	assert.Contains(t, skill.Body, "Use for external lookups.")
	assert.Equal(t, "gemini", skill.SourceProvider)
}

func TestGeminiExtract_RuleFrontmatter(t *testing.T) {
	data := []byte("---\ndescription: Style guide\n---\n\nUse gofmt. Keep functions small.\n")
	config := &ast.XcaffoldConfig{}
	imp := geminimp.NewImporter()
	err := imp.Extract("rules/style.md", data, config)
	require.NoError(t, err)
	rule, ok := config.Rules["style"]
	require.True(t, ok, "expected rule 'style' in config")
	assert.Equal(t, "Style guide", rule.Description)
	assert.Contains(t, rule.Body, "Use gofmt.")
	assert.Equal(t, "gemini", rule.SourceProvider)
}

func TestGeminiExtract_SettingsDecomposition(t *testing.T) {
	data := []byte(`{
		"model": "gemini-2.5-pro",
		"hooks": {
			"PreToolUse": [
				{"matcher": "Bash", "hooks": [{"type": "command", "command": "echo pre"}]}
			]
		},
		"mcpServers": {
			"search-tool": {"type": "stdio", "command": "search-mcp"}
		}
	}`)
	config := &ast.XcaffoldConfig{}
	imp := geminimp.NewImporter()
	err := imp.Extract("settings.json", data, config)
	require.NoError(t, err)
	// Hooks extracted
	var effectiveHooks ast.HookConfig
	if dh, ok := config.Hooks["default"]; ok {
		effectiveHooks = dh.Events
	}
	assert.NotNil(t, effectiveHooks["PreToolUse"])
	assert.Len(t, effectiveHooks["PreToolUse"], 1)
	// MCP from settings.json mcpServers extracted
	_, ok := config.MCP["search-tool"]
	require.True(t, ok, "expected mcp 'search-tool' from settings.json")
	assert.Equal(t, "gemini", config.MCP["search-tool"].SourceProvider)
	// Settings model extracted
	assert.Equal(t, "gemini-2.5-pro", config.Settings["default"].Model)
	assert.Equal(t, "gemini", config.Settings["default"].SourceProvider)
}

func TestGeminiExtract_UnknownKindReturnsError(t *testing.T) {
	imp := geminimp.NewImporter()
	config := &ast.XcaffoldConfig{}
	err := imp.Extract("unknown-file", []byte("data"), config)
	require.Error(t, err)
}

// --- Import (full workspace) tests ---

func TestGeminiExtract_ExtrasCollection(t *testing.T) {
	imp := geminimp.NewImporter()
	config := &ast.XcaffoldConfig{}
	inputDir := filepath.Join("testdata", "input")
	err := imp.Import(inputDir, config)
	require.NoError(t, err)
	// All known files should be classified
	assert.NotEmpty(t, config.Agents)
	assert.NotEmpty(t, config.Skills)
	assert.NotEmpty(t, config.Rules)
	// unknown-file is not classified — should appear in ProviderExtras
	extras, hasProvider := config.ProviderExtras["gemini"]
	require.True(t, hasProvider, "expected ProviderExtras to have 'gemini' entry")
	_, hasUnknown := extras["unknown-file"]
	assert.True(t, hasUnknown, "expected 'unknown-file' in ProviderExtras")
}

func TestGeminiImporter_FullWorkspace(t *testing.T) {
	imp := geminimp.NewImporter()
	config := &ast.XcaffoldConfig{}
	inputDir := filepath.Join("testdata", "input")
	err := imp.Import(inputDir, config)
	require.NoError(t, err)

	// Agents
	assistant, ok := config.Agents["assistant"]
	require.True(t, ok, "expected agent 'assistant'")
	assert.Equal(t, "Assistant Agent", assistant.Name)
	assert.Equal(t, "gemini", assistant.SourceProvider)
	assert.NotEmpty(t, assistant.Body)

	// Skills (flat file — id is filename stem)
	search, ok := config.Skills["search"]
	require.True(t, ok, "expected skill 'search'")
	assert.Equal(t, "web-search", search.Name)
	assert.Equal(t, "gemini", search.SourceProvider)

	// Rules
	style, ok := config.Rules["style"]
	require.True(t, ok, "expected rule 'style'")
	assert.Equal(t, "gemini", style.SourceProvider)

	// Hooks from settings.json
	assert.NotEmpty(t, config.Hooks["default"].Events["PreToolUse"])

	// MCP from settings.json mcpServers
	_, ok = config.MCP["search-tool"]
	require.True(t, ok, "expected mcp server 'search-tool' from settings.json")
	assert.Equal(t, "gemini", config.MCP["search-tool"].SourceProvider)

	// Settings
	assert.Equal(t, "gemini-2.5-pro", config.Settings["default"].Model)

	// Unknown file goes to extras
	assert.NotEmpty(t, config.ProviderExtras["gemini"])
}
