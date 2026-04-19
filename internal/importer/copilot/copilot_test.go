package copilot_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/importer"
	copilotimp "github.com/saero-ai/xcaffold/internal/importer/copilot"
)

// --- Provider metadata ---

func TestCopilotImporter_Provider(t *testing.T) {
	assert.Equal(t, "copilot", copilotimp.New().Provider())
}

func TestCopilotImporter_InputDir(t *testing.T) {
	assert.Equal(t, ".github", copilotimp.New().InputDir())
}

// --- Classify tests ---

func TestCopilotClassify_AgentPattern(t *testing.T) {
	imp := copilotimp.New()
	kind, layout := imp.Classify("agents/auditor.agent.md", false)
	assert.Equal(t, importer.KindAgent, kind)
	assert.Equal(t, importer.FlatFile, layout)
}

func TestCopilotClassify_SkillPattern(t *testing.T) {
	imp := copilotimp.New()
	kind, layout := imp.Classify("skills/review.md", false)
	assert.Equal(t, importer.KindSkill, kind)
	assert.Equal(t, importer.FlatFile, layout)
}

func TestCopilotClassify_RulePattern(t *testing.T) {
	imp := copilotimp.New()
	kind, layout := imp.Classify("instructions/security.instructions.md", false)
	assert.Equal(t, importer.KindRule, kind)
	assert.Equal(t, importer.FlatFile, layout)
}

func TestCopilotClassify_MCPPattern(t *testing.T) {
	imp := copilotimp.New()
	kind, layout := imp.Classify("copilot/mcp-config.json", false)
	assert.Equal(t, importer.KindMCP, kind)
	assert.Equal(t, importer.StandaloneJSON, layout)
}

func TestCopilotClassify_WorkflowPattern(t *testing.T) {
	imp := copilotimp.New()
	kind, layout := imp.Classify("workflows/copilot-setup-steps.yml", false)
	assert.Equal(t, importer.KindWorkflow, kind)
	assert.Equal(t, importer.FlatFile, layout)
}

// copilot-instructions.md is the orchestrator's job (project instructions).
// The importer MUST classify it as KindUnknown so it ends up in extras.
func TestCopilotClassify_CopilotInstructionsMdIsUnknown(t *testing.T) {
	imp := copilotimp.New()
	kind, layout := imp.Classify("copilot-instructions.md", false)
	assert.Equal(t, importer.KindUnknown, kind)
	assert.Equal(t, importer.LayoutUnknown, layout)
}

func TestCopilotClassify_UnknownFile(t *testing.T) {
	imp := copilotimp.New()
	kind, layout := imp.Classify("unknown-file", false)
	assert.Equal(t, importer.KindUnknown, kind)
	assert.Equal(t, importer.LayoutUnknown, layout)
}

func TestCopilotClassify_NonMatchingRuleExtension(t *testing.T) {
	// A plain .md in instructions/ without .instructions.md double extension is unknown.
	imp := copilotimp.New()
	kind, _ := imp.Classify("instructions/security.md", false)
	assert.Equal(t, importer.KindUnknown, kind)
}

// --- Extract tests ---

func TestCopilotExtract_Agent(t *testing.T) {
	data := []byte("---\nname: Code Auditor\ndescription: Reviews code\nmodel: gpt-4o\n---\n\nYou are an auditor.\n")
	config := &ast.XcaffoldConfig{}
	imp := copilotimp.New()
	err := imp.Extract("agents/auditor.agent.md", data, config)
	require.NoError(t, err)
	agent, ok := config.Agents["auditor"]
	require.True(t, ok, "expected agent 'auditor' in config")
	assert.Equal(t, "Code Auditor", agent.Name)
	assert.Equal(t, "Reviews code", agent.Description)
	assert.Equal(t, "gpt-4o", agent.Model)
	assert.Contains(t, agent.Instructions, "You are an auditor.")
	assert.Equal(t, "copilot", agent.SourceProvider)
}

// TestCopilotExtract_Agent_PlainMd ensures plain .md agent files still parse
// correctly for backward compatibility.
func TestCopilotExtract_Agent_PlainMd(t *testing.T) {
	data := []byte("---\nname: Legacy Agent\ndescription: Old style\nmodel: gpt-4o\n---\n\nLegacy instructions.\n")
	config := &ast.XcaffoldConfig{}
	imp := copilotimp.New()
	err := imp.Extract("agents/legacy.md", data, config)
	require.NoError(t, err)
	agent, ok := config.Agents["legacy"]
	require.True(t, ok, "expected agent 'legacy' in config")
	assert.Equal(t, "Legacy Agent", agent.Name)
	assert.Equal(t, "copilot", agent.SourceProvider)
}

func TestCopilotExtract_Skill(t *testing.T) {
	data := []byte("---\nname: code-review\ndescription: Review code\n---\n\nReview the code carefully.\n")
	config := &ast.XcaffoldConfig{}
	imp := copilotimp.New()
	err := imp.Extract("skills/review.md", data, config)
	require.NoError(t, err)
	skill, ok := config.Skills["review"]
	require.True(t, ok, "expected skill 'review' in config")
	assert.Equal(t, "code-review", skill.Name)
	assert.Equal(t, "copilot", skill.SourceProvider)
}

// Double-extension stripping: "security.instructions.md" → id = "security"
func TestCopilotExtract_Rule_DoubleExtensionStripped(t *testing.T) {
	data := []byte("---\ndescription: Security guidelines\n---\n\nNever hardcode secrets.\n")
	config := &ast.XcaffoldConfig{}
	imp := copilotimp.New()
	err := imp.Extract("instructions/security.instructions.md", data, config)
	require.NoError(t, err)
	rule, ok := config.Rules["security"]
	require.True(t, ok, "expected rule id 'security' (double extension stripped)")
	assert.Equal(t, "Security guidelines", rule.Description)
	assert.Contains(t, rule.Instructions, "Never hardcode secrets.")
	assert.Equal(t, "copilot", rule.SourceProvider)
}

func TestCopilotExtract_MCP(t *testing.T) {
	data := []byte(`{"mcpServers":{"github":{"type":"stdio","command":"gh-mcp","args":["serve"]}}}`)
	config := &ast.XcaffoldConfig{}
	imp := copilotimp.New()
	err := imp.Extract("copilot/mcp-config.json", data, config)
	require.NoError(t, err)
	mc, ok := config.MCP["github"]
	require.True(t, ok, "expected mcp server 'github'")
	assert.Equal(t, "stdio", mc.Type)
	assert.Equal(t, "gh-mcp", mc.Command)
	assert.Equal(t, "copilot", mc.SourceProvider)
}

func TestCopilotExtract_Workflow(t *testing.T) {
	data := []byte("name: Setup\ndescription: Project setup steps\n")
	config := &ast.XcaffoldConfig{}
	imp := copilotimp.New()
	err := imp.Extract("workflows/copilot-setup-steps.yml", data, config)
	require.NoError(t, err)
	wf, ok := config.Workflows["copilot-setup-steps"]
	require.True(t, ok, "expected workflow 'copilot-setup-steps'")
	assert.Equal(t, "copilot", wf.SourceProvider)
}

// --- Full workspace golden test ---

func TestCopilotImporter_FullWorkspace(t *testing.T) {
	imp := copilotimp.New()
	config := &ast.XcaffoldConfig{}
	inputDir := filepath.Join("testdata", "input")
	err := imp.Import(inputDir, config)
	require.NoError(t, err)

	// Agent
	auditor, ok := config.Agents["auditor"]
	require.True(t, ok, "expected agent 'auditor'")
	assert.Equal(t, "Code Auditor", auditor.Name)
	assert.Equal(t, "copilot", auditor.SourceProvider)
	assert.NotEmpty(t, auditor.Instructions)

	// Skill (flat file — id is filename without .md)
	review, ok := config.Skills["review"]
	require.True(t, ok, "expected skill 'review'")
	assert.Equal(t, "code-review", review.Name)
	assert.Equal(t, "copilot", review.SourceProvider)

	// Rule — double extension stripped
	sec, ok := config.Rules["security"]
	require.True(t, ok, "expected rule 'security'")
	assert.Equal(t, "copilot", sec.SourceProvider)

	// MCP from copilot/mcp-config.json
	_, ok = config.MCP["github"]
	require.True(t, ok, "expected mcp server 'github'")

	// copilot-instructions.md is NOT classified — goes to extras
	extras, hasProvider := config.ProviderExtras["copilot"]
	require.True(t, hasProvider, "expected ProviderExtras to have 'copilot' entry")
	_, hasCopilotInstructions := extras["copilot-instructions.md"]
	assert.True(t, hasCopilotInstructions, "expected 'copilot-instructions.md' in extras")

	// unknown-file also goes to extras
	_, hasUnknown := extras["unknown-file"]
	assert.True(t, hasUnknown, "expected 'unknown-file' in extras")
}
