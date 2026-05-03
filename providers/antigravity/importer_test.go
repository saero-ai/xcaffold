package antigravity_test

import (
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	antimp "github.com/saero-ai/xcaffold/providers/antigravity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saero-ai/xcaffold/internal/importer"
)

// --- Classify tests ---

func TestAntigravityClassify_AgentInPrompts(t *testing.T) {
	imp := antimp.NewImporter()
	kind, layout := imp.Classify("prompts/explorer.md", false)
	assert.Equal(t, importer.KindAgent, kind)
	assert.Equal(t, importer.FlatFile, layout)
}

func TestAntigravityClassify_SkillPattern_DirectoryPerEntry(t *testing.T) {
	imp := antimp.NewImporter()
	// Antigravity skills use directory-per-entry layout
	kind, layout := imp.Classify("skills/search/SKILL.md", false)
	assert.Equal(t, importer.KindSkill, kind)
	assert.Equal(t, importer.DirectoryPerEntry, layout)
}

func TestAntigravityClassify_SkillReferences(t *testing.T) {
	imp := antimp.NewImporter()
	// Skill asset files in references/ subdirectory
	kind, layout := imp.Classify("skills/search/references/example.md", false)
	assert.Equal(t, importer.KindSkillAsset, kind)
	assert.Equal(t, importer.DirectoryPerEntry, layout)
}

func TestAntigravityClassify_SkillScripts(t *testing.T) {
	imp := antimp.NewImporter()
	// Skill asset files in scripts/ subdirectory
	kind, layout := imp.Classify("skills/search/scripts/setup.sh", false)
	assert.Equal(t, importer.KindSkillAsset, kind)
	assert.Equal(t, importer.DirectoryPerEntry, layout)
}

func TestAntigravityClassify_SkillExamples(t *testing.T) {
	imp := antimp.NewImporter()
	// Skill asset files in examples/ subdirectory (Antigravity-native)
	kind, layout := imp.Classify("skills/search/examples/usage.md", false)
	assert.Equal(t, importer.KindSkillAsset, kind)
	assert.Equal(t, importer.DirectoryPerEntry, layout)
}

func TestAntigravityClassify_RulePattern(t *testing.T) {
	imp := antimp.NewImporter()
	kind, layout := imp.Classify("rules/safety.md", false)
	assert.Equal(t, importer.KindRule, kind)
	assert.Equal(t, importer.FlatFile, layout)
}

func TestAntigravityClassify_WorkflowPattern(t *testing.T) {
	imp := antimp.NewImporter()
	kind, layout := imp.Classify("workflows/weekly-audit.md", false)
	assert.Equal(t, importer.KindWorkflow, kind)
	assert.Equal(t, importer.FlatFile, layout)
}

func TestAntigravityClassify_MCPConfig(t *testing.T) {
	imp := antimp.NewImporter()
	kind, layout := imp.Classify("mcp_config.json", false)
	assert.Equal(t, importer.KindMCP, kind)
	assert.Equal(t, importer.StandaloneJSON, layout)
}

func TestAntigravityClassify_UnknownFile(t *testing.T) {
	imp := antimp.NewImporter()
	kind, layout := imp.Classify("unknown-file", false)
	assert.Equal(t, importer.KindUnknown, kind)
	assert.Equal(t, importer.LayoutUnknown, layout)
}

func TestAntigravityClassify_AgentsDir_NotMatched(t *testing.T) {
	// Antigravity does NOT have an agents/ directory — prompts/ is the agent directory.
	imp := antimp.NewImporter()
	kind, _ := imp.Classify("agents/ceo.md", false)
	assert.Equal(t, importer.KindUnknown, kind)
}

func TestAntigravityImporter_Provider(t *testing.T) {
	assert.Equal(t, "antigravity", antimp.NewImporter().Provider())
}

func TestAntigravityImporter_InputDir(t *testing.T) {
	assert.Equal(t, ".agents", antimp.NewImporter().InputDir())
}

// --- Extract tests ---

func TestAntigravityExtract_AgentFromPrompts(t *testing.T) {
	data := []byte("---\nname: Explorer Agent\ndescription: Navigates codebases\nmodel: claude-opus-4-5\n---\n\nExplore the codebase thoroughly.\n")
	config := &ast.XcaffoldConfig{}
	imp := antimp.NewImporter()
	err := imp.Extract("prompts/explorer.md", data, config)
	require.NoError(t, err)

	// Agent must land in config.Agents, keyed by filename stem.
	agent, ok := config.Agents["explorer"]
	require.True(t, ok, "expected agent 'explorer' in config.Agents — prompts/*.md must map to KindAgent")
	assert.Equal(t, "Explorer Agent", agent.Name)
	assert.Equal(t, "Navigates codebases", agent.Description)
	assert.Equal(t, "claude-opus-4-5", agent.Model)
	assert.Contains(t, agent.Body, "Explore the codebase thoroughly.")
	assert.Equal(t, "antigravity", agent.SourceProvider)
}

func TestAntigravityExtract_Skill(t *testing.T) {
	data := []byte("---\nname: search\ndescription: Deep search\nallowed-tools:\n  - Grep\n---\n\nSearch across files.\n")
	config := &ast.XcaffoldConfig{}
	imp := antimp.NewImporter()
	err := imp.Extract("skills/search/SKILL.md", data, config)
	require.NoError(t, err)

	skill, ok := config.Skills["search"]
	require.True(t, ok, "expected skill 'search'")
	assert.Equal(t, "search", skill.Name)
	assert.Equal(t, "Deep search", skill.Description)
	assert.Equal(t, []string{"Grep"}, skill.AllowedTools)
	assert.Contains(t, skill.Body, "Search across files.")
	assert.Equal(t, "antigravity", skill.SourceProvider)
}

func TestAntigravityExtract_Rule(t *testing.T) {
	data := []byte("---\ndescription: Safety constraints\n---\n\nNever delete without confirmation.\n")
	config := &ast.XcaffoldConfig{}
	imp := antimp.NewImporter()
	err := imp.Extract("rules/safety.md", data, config)
	require.NoError(t, err)

	rule, ok := config.Rules["safety"]
	require.True(t, ok, "expected rule 'safety'")
	assert.Equal(t, "Safety constraints", rule.Description)
	assert.Contains(t, rule.Body, "Never delete without confirmation.")
	assert.Equal(t, "antigravity", rule.SourceProvider)
}

func TestAntigravityExtract_Workflow(t *testing.T) {
	data := []byte("---\nname: weekly-audit\ndescription: Weekly audit workflow\n---\n\nRun analysis and produce a report.\n")
	config := &ast.XcaffoldConfig{}
	imp := antimp.NewImporter()
	err := imp.Extract("workflows/weekly-audit.md", data, config)
	require.NoError(t, err)

	wf, ok := config.Workflows["weekly-audit"]
	require.True(t, ok, "expected workflow 'weekly-audit'")
	assert.Equal(t, "weekly-audit", wf.Name)
	assert.Equal(t, "Weekly audit workflow", wf.Description)
	assert.Contains(t, wf.Body, "Run analysis and produce a report.")
	assert.Equal(t, "antigravity", wf.SourceProvider)
}

func TestAntigravityExtract_MCPConfig(t *testing.T) {
	data := []byte(`{"mcpServers":{"filesystem":{"type":"stdio","command":"mcp-filesystem","args":["--root","."]}}}`)
	config := &ast.XcaffoldConfig{}
	imp := antimp.NewImporter()
	err := imp.Extract("mcp_config.json", data, config)
	require.NoError(t, err)

	mc, ok := config.MCP["filesystem"]
	require.True(t, ok, "expected mcp server 'filesystem'")
	assert.Equal(t, "stdio", mc.Type)
	assert.Equal(t, "mcp-filesystem", mc.Command)
	assert.Equal(t, "antigravity", mc.SourceProvider)
}

func TestAntigravityExtract_UnknownKindErrors(t *testing.T) {
	imp := antimp.NewImporter()
	config := &ast.XcaffoldConfig{}
	err := imp.Extract("unknown-file", []byte("content"), config)
	require.Error(t, err)
}

// --- Full workspace golden test ---

func TestAntigravityImporter_FullWorkspace(t *testing.T) {
	imp := antimp.NewImporter()
	config := &ast.XcaffoldConfig{}
	inputDir := filepath.Join("testdata", "input")
	err := imp.Import(inputDir, config)
	require.NoError(t, err)

	// Agent from prompts/ directory
	agent, ok := config.Agents["explorer"]
	require.True(t, ok, "expected agent 'explorer' from prompts/explorer.md")
	assert.Equal(t, "Explorer Agent", agent.Name)
	assert.Equal(t, "antigravity", agent.SourceProvider)
	assert.NotEmpty(t, agent.Body)

	// Skill
	skill, ok := config.Skills["search"]
	require.True(t, ok, "expected skill 'search'")
	assert.Equal(t, "antigravity", skill.SourceProvider)

	// Rule
	rule, ok := config.Rules["safety"]
	require.True(t, ok, "expected rule 'safety'")
	assert.Equal(t, "antigravity", rule.SourceProvider)

	// Workflow
	wf, ok := config.Workflows["weekly-audit"]
	require.True(t, ok, "expected workflow 'weekly-audit'")
	assert.Equal(t, "antigravity", wf.SourceProvider)

	// MCP from mcp_config.json
	_, ok = config.MCP["filesystem"]
	require.True(t, ok, "expected mcp server 'filesystem' from mcp_config.json")

	// Unknown file goes to extras
	extras, hasProvider := config.ProviderExtras["antigravity"]
	require.True(t, hasProvider, "expected ProviderExtras to have 'antigravity' entry")
	_, hasUnknown := extras["unknown-file"]
	assert.True(t, hasUnknown, "expected 'unknown-file' in ProviderExtras")
}

// --- Extras collection test ---

func TestAntigravityImporter_ExtrasCollection(t *testing.T) {
	imp := antimp.NewImporter()
	config := &ast.XcaffoldConfig{}
	inputDir := filepath.Join("testdata", "input")
	err := imp.Import(inputDir, config)
	require.NoError(t, err)

	assert.NotEmpty(t, config.Agents)
	assert.NotEmpty(t, config.Skills)
	assert.NotEmpty(t, config.Rules)
	assert.NotEmpty(t, config.Workflows)

	extras, hasProvider := config.ProviderExtras["antigravity"]
	require.True(t, hasProvider)
	_, hasUnknown := extras["unknown-file"]
	assert.True(t, hasUnknown)
}
