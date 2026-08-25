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

func TestAntigravityClassify_AgentInAgents(t *testing.T) {
	imp := antimp.NewImporter()
	kind, layout := imp.Classify("agents/explorer.md", false)
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
	// Skill asset files in examples/ subdirectory
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

func TestAntigravityClassify_NestedRulePattern(t *testing.T) {
	imp := antimp.NewImporter()
	kind, layout := imp.Classify("rules/frontend/animation.md", false)
	assert.Equal(t, importer.KindRule, kind)
	assert.Equal(t, importer.FlatFile, layout)
}

func TestAntigravityClassify_HooksJSON(t *testing.T) {
	imp := antimp.NewImporter()
	kind, layout := imp.Classify("hooks.json", false)
	assert.Equal(t, importer.KindHook, kind)
	assert.Equal(t, importer.StandaloneJSON, layout)
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

func TestAntigravityImporter_Provider(t *testing.T) {
	assert.Equal(t, "antigravity", antimp.NewImporter().Provider())
}

func TestAntigravityImporter_InputDir(t *testing.T) {
	assert.Equal(t, ".agents", antimp.NewImporter().InputDir())
}

// --- Extract tests ---

func TestAntigravityExtract_AgentFromAgents(t *testing.T) {
	data := []byte("---\nname: Explorer Agent\ndescription: Navigates codebases\nmodel: claude-opus-4-6-thinking\n---\n\nExplore the codebase thoroughly.\n")
	config := &ast.XcaffoldConfig{}
	imp := antimp.NewImporter()
	err := imp.Extract("agents/explorer.md", data, config)
	require.NoError(t, err)

	// Agent must land in config.Agents, keyed by filename stem.
	agent, ok := config.Agents["explorer"]
	require.True(t, ok, "expected agent 'explorer' in config.Agents — agents/*.md must map to KindAgent")
	assert.Equal(t, "Explorer Agent", agent.Name)
	assert.Equal(t, "Navigates codebases", agent.Description)
	assert.Equal(t, "claude-opus-4-6-thinking", agent.Model)
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
	assert.Equal(t, ast.ClearableList{Values: []string{"Grep"}}, skill.AllowedTools)
	assert.Contains(t, skill.Body, "Search across files.")
	assert.Equal(t, "antigravity", skill.SourceProvider)
}

func TestAntigravityExtract_HooksJSON(t *testing.T) {
	data := []byte(`{
		"PreInvocation": [{"type": "command", "command": "echo pre-invoke"}],
		"PostToolUse": [{"type": "command", "command": "echo post-tool"}]
	}`)
	config := &ast.XcaffoldConfig{}
	imp := antimp.NewImporter()
	err := imp.Extract("hooks.json", data, config)
	require.NoError(t, err)

	require.NotNil(t, config.Hooks)
	defaultHook, ok := config.Hooks["default"]
	require.True(t, ok, "expected default hook in config.Hooks")
	assert.Len(t, defaultHook.Events["PreInvocation"], 1)
	assert.Len(t, defaultHook.Events["PostToolUse"], 1)
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

func TestAntigravityExtract_Rule_WithGlobsString(t *testing.T) {
	data := []byte("---\ntrigger: glob\ndescription: Flutter animations\nglobs: **/huppen-app/**/*.dart, **/huppen_ui/**/*.dart\n---\n\nUse spring animations.\n")
	config := &ast.XcaffoldConfig{}
	imp := antimp.NewImporter()
	err := imp.Extract("rules/frontend/animation.md", data, config)
	require.NoError(t, err)

	rule, ok := config.Rules["frontend/animation"]
	require.True(t, ok, "expected nested rule 'frontend/animation'")
	assert.Equal(t, "frontend/animation", rule.Name)
	assert.Equal(t, "Flutter animations", rule.Description)
	assert.Equal(t, ast.RuleActivationPathGlob, rule.Activation)
	assert.Equal(t, ast.ClearableList{Values: []string{"**/huppen-app/**/*.dart", "**/huppen_ui/**/*.dart"}}, rule.Paths)
	assert.Contains(t, rule.Body, "Use spring animations.")
}

func TestAntigravityExtract_Rule_WithGlobsSequence(t *testing.T) {
	data := []byte("---\ntrigger: glob\ndescription: Formatting\nglobs:\n  - \"**/*.go\"\n  - \"**/*.ts\"\n---\n\nUse proper formatters.\n")
	config := &ast.XcaffoldConfig{}
	imp := antimp.NewImporter()
	err := imp.Extract("rules/formatting.md", data, config)
	require.NoError(t, err)

	rule, ok := config.Rules["formatting"]
	require.True(t, ok, "expected rule 'formatting'")
	assert.Equal(t, ast.RuleActivationPathGlob, rule.Activation)
	assert.Equal(t, ast.ClearableList{Values: []string{"**/*.go", "**/*.ts"}}, rule.Paths)
}

func TestAntigravityExtract_Rule_ModelDecision(t *testing.T) {
	data := []byte("---\ntrigger: model_decision\ndescription: API design standards\n---\n\nDesign clean APIs.\n")
	config := &ast.XcaffoldConfig{}
	imp := antimp.NewImporter()
	err := imp.Extract("rules/api-standards.md", data, config)
	require.NoError(t, err)

	rule, ok := config.Rules["api-standards"]
	require.True(t, ok, "expected rule 'api-standards'")
	assert.Equal(t, ast.RuleActivationModelDecided, rule.Activation)
}

func TestAntigravityExtract_Rule_Manual(t *testing.T) {
	data := []byte("---\ntrigger: manual\ndescription: Explicit mention rule\n---\n\nSpecial procedures.\n")
	config := &ast.XcaffoldConfig{}
	imp := antimp.NewImporter()
	err := imp.Extract("rules/special.md", data, config)
	require.NoError(t, err)

	rule, ok := config.Rules["special"]
	require.True(t, ok, "expected rule 'special'")
	assert.Equal(t, ast.RuleActivationManualMention, rule.Activation)
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
	assert.Equal(t, "antigravity", wf.SourceProvider)
}

// TestAntigravityExtract_Workflow_BodyBecomesMainStep verifies that when a workflow
// has no steps in frontmatter but has body content, the importer creates a single
// "main" step with the body as the Instructions field.
func TestAntigravityExtract_Workflow_BodyBecomesMainStep(t *testing.T) {
	data := []byte("---\nname: analyze-code\ndescription: Code analysis workflow\n---\n\nRun linters and formatters on the codebase.\n")
	config := &ast.XcaffoldConfig{}
	imp := antimp.NewImporter()
	err := imp.Extract("workflows/analyze-code.md", data, config)
	require.NoError(t, err)

	wf, ok := config.Workflows["analyze-code"]
	require.True(t, ok, "expected workflow 'analyze-code'")

	// Native Antigravity workflows store content in the markdown body, not YAML steps.
	// When steps are absent, the body must be imported as a single "main" step.
	require.Len(t, wf.Steps, 1, "expected exactly one step created from body")
	assert.Equal(t, "main", wf.Steps[0].Name, "step name must be 'main'")
	assert.Equal(t, "Run linters and formatters on the codebase.", wf.Steps[0].Instructions)
}

// TestAntigravityExtract_Workflow_FrontmatterStepsPreserved verifies that if a workflow
// already has steps defined in YAML frontmatter, those steps are preserved and no
// "main" step is created from the body.
func TestAntigravityExtract_Workflow_FrontmatterStepsPreserved(t *testing.T) {
	data := []byte(`---
name: multi-step-audit
description: Multi-step workflow
steps:
  - name: scan-deps
    description: Scan dependencies
    instructions: Run dependency checker
  - name: lint-code
    description: Lint code
    instructions: Run linter
---

This is body content that should be ignored when steps are in frontmatter.
`)
	config := &ast.XcaffoldConfig{}
	imp := antimp.NewImporter()
	err := imp.Extract("workflows/multi-step-audit.md", data, config)
	require.NoError(t, err)

	wf, ok := config.Workflows["multi-step-audit"]
	require.True(t, ok, "expected workflow 'multi-step-audit'")

	// When steps are defined in frontmatter, they should be preserved as-is.
	// No "main" step should be created from the body.
	require.Len(t, wf.Steps, 2, "expected two steps from frontmatter")
	assert.Equal(t, "scan-deps", wf.Steps[0].Name)
	assert.Equal(t, "lint-code", wf.Steps[1].Name)
}

func TestAntigravityExtract_MCPConfig(t *testing.T) {
	data := []byte(`{"mcpServers":{"filesystem":{"command":"mcp-filesystem","args":["--root","."]}}}`)
	config := &ast.XcaffoldConfig{}
	imp := antimp.NewImporter()
	err := imp.Extract("mcp_config.json", data, config)
	require.NoError(t, err)

	mc, ok := config.MCP["filesystem"]
	require.True(t, ok, "expected mcp server 'filesystem'")
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

	// Agent from agents/ directory
	agent, ok := config.Agents["explorer"]
	require.True(t, ok, "expected agent 'explorer' from agents/explorer.md")
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

	// Unknown files are silently skipped
	assert.Empty(t, config.ProviderExtras["antigravity"])
}
