// Package importer_test contains round-trip tests that import a provider
// directory and compile the resulting AST back to the same provider,
// verifying that the same resources are present in the output.
package importer_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/compiler"
	antigravityimp "github.com/saero-ai/xcaffold/providers/antigravity"
	claudeimp "github.com/saero-ai/xcaffold/providers/claude"
	copilotimp "github.com/saero-ai/xcaffold/providers/copilot"
	cursorimp "github.com/saero-ai/xcaffold/providers/cursor"
	geminimp "github.com/saero-ai/xcaffold/providers/gemini"
)

// testdataDir returns the absolute path to a provider's testdata/input directory.
// It resolves relative to this test file's location so tests are location-independent.
// Consolidated providers (claude, cursor, gemini, copilot, antigravity) live in providers/<provider>/; others remain under internal/importer/<provider>/.
func testdataDir(provider string) string {
	_, file, _, _ := runtime.Caller(0)
	base := filepath.Dir(file)
	switch provider {
	case "claude":
		return filepath.Join(base, "..", "..", "providers", "claude", "testdata", "input")
	case "cursor":
		return filepath.Join(base, "..", "..", "providers", "cursor", "testdata", "input")
	case "gemini":
		return filepath.Join(base, "..", "..", "providers", "gemini", "testdata", "input")
	case "copilot":
		return filepath.Join(base, "..", "..", "providers", "copilot", "testdata", "input")
	case "antigravity":
		return filepath.Join(base, "..", "..", "providers", "antigravity", "testdata", "input")
	}
	return filepath.Join(base, provider, "testdata", "input")
}

// TestRoundTrip_ClaudeImportCompile imports a Claude .claude/ directory tree
// and compiles the AST back to the Claude target, then verifies that each
// imported resource appears in the compilation output.
func TestRoundTrip_ClaudeImportCompile(t *testing.T) {
	// --- Red: import phase ---
	imp := claudeimp.NewImporter()
	config := &ast.XcaffoldConfig{}
	err := imp.Import(testdataDir("claude"), config)
	require.NoError(t, err, "Import must not fail on well-formed testdata")

	// Sanity-check that the importer populated the expected resources.
	require.Contains(t, config.Agents, "backend", "agent 'backend' must be imported")
	require.Contains(t, config.Rules, "security", "rule 'security' must be imported")
	require.Contains(t, config.Skills, "tdd", "skill 'tdd' must be imported")

	// --- Green: compile phase ---
	out, _, err := compiler.Compile(config, ".", "claude", "")
	require.NoError(t, err, "Compile must not fail on a valid AST")

	// --- Verify: same resource keys are present in the output ---
	assert.Contains(t, out.Files, filepath.Clean("agents/backend.md"),
		"compiled output must contain agents/backend.md")
	assert.Contains(t, out.Files, filepath.Clean("rules/security.md"),
		"compiled output must contain rules/security.md")
	assert.Contains(t, out.Files, filepath.Clean("skills/tdd/SKILL.md"),
		"compiled output must contain skills/tdd/SKILL.md")

	// Verify field round-trip at the semantic level (not byte-identical).
	backendContent := out.Files[filepath.Clean("agents/backend.md")]
	assert.Contains(t, backendContent, "Backend Agent",
		"compiled agent must preserve the name field")
	assert.Contains(t, backendContent, "claude-opus-4-5",
		"compiled agent must preserve the model field")

	securityContent := out.Files[filepath.Clean("rules/security.md")]
	assert.Contains(t, securityContent, "Security standards",
		"compiled rule must preserve the description field")

	skillContent := out.Files[filepath.Clean("skills/tdd/SKILL.md")]
	assert.Contains(t, skillContent, "tdd-driven-development",
		"compiled skill must preserve the name field")
}

// TestRoundTrip_ClaudeImportCompile_AgentFields verifies that agent frontmatter
// fields (tools, description, instructions body) survive the round-trip.
func TestRoundTrip_ClaudeImportCompile_AgentFields(t *testing.T) {
	imp := claudeimp.NewImporter()
	config := &ast.XcaffoldConfig{}
	require.NoError(t, imp.Import(testdataDir("claude"), config))

	agent, ok := config.Agents["backend"]
	require.True(t, ok, "agent 'backend' must be present after import")

	assert.Equal(t, "Backend Agent", agent.Name)
	assert.Equal(t, "claude-opus-4-5", agent.Model)
	assert.NotEmpty(t, agent.Tools, "tools must be imported")
	assert.NotEmpty(t, agent.Body, "instructions body must be imported")

	out, _, err := compiler.Compile(config, ".", "claude", "")
	require.NoError(t, err)

	content := out.Files[filepath.Clean("agents/backend.md")]
	assert.Contains(t, content, "Read", "compiled agent must list at least one tool")
	assert.Contains(t, content, "You are a backend Go developer",
		"compiled agent must preserve instruction body")
}

// TestRoundTrip_ClaudeImportCompile_SkillFields verifies that skill frontmatter
// fields survive the round-trip.
func TestRoundTrip_ClaudeImportCompile_SkillFields(t *testing.T) {
	imp := claudeimp.NewImporter()
	config := &ast.XcaffoldConfig{}
	require.NoError(t, imp.Import(testdataDir("claude"), config))

	skill, ok := config.Skills["tdd"]
	require.True(t, ok, "skill 'tdd' must be present after import")

	assert.Equal(t, "tdd-driven-development", skill.Name)
	assert.NotEmpty(t, skill.AllowedTools, "allowed-tools must be imported")
	assert.NotEmpty(t, skill.Body, "instructions body must be imported")

	out, _, err := compiler.Compile(config, ".", "claude", "")
	require.NoError(t, err)

	content := out.Files[filepath.Clean("skills/tdd/SKILL.md")]
	assert.Contains(t, content, "Red-Green-Refactor",
		"compiled skill must preserve instruction body")
}

// TestRoundTrip_CursorImportCompile imports a Cursor .cursor/ directory tree
// and compiles the AST back to the Cursor target, then verifies that each
// imported resource appears in the compilation output.
func TestRoundTrip_CursorImportCompile(t *testing.T) {
	// --- Red: import phase ---
	imp := cursorimp.NewImporter()
	config := &ast.XcaffoldConfig{}
	err := imp.Import(testdataDir("cursor"), config)
	require.NoError(t, err, "Import must not fail on well-formed testdata")

	// Sanity-check that the importer populated the expected resources.
	require.Contains(t, config.Agents, "reviewer", "agent 'reviewer' must be imported")
	require.Contains(t, config.Rules, "formatting", "rule 'formatting' must be imported")
	require.Contains(t, config.Skills, "code-review", "skill 'code-review' must be imported")

	// --- Green: compile phase ---
	out, _, err := compiler.Compile(config, ".", "cursor", "")
	require.NoError(t, err, "Compile must not fail on a valid AST")

	// --- Verify: same resource keys are present in the output ---
	// Cursor rules use .mdc extension; agents are .md; skills are SKILL.md.
	assert.Contains(t, out.Files, filepath.Clean("rules/formatting.mdc"),
		"compiled output must contain rules/formatting.mdc")
	assert.Contains(t, out.Files, filepath.Clean("agents/reviewer.md"),
		"compiled output must contain agents/reviewer.md")
	assert.Contains(t, out.Files, filepath.Clean("skills/code-review/SKILL.md"),
		"compiled output must contain skills/code-review/SKILL.md")

	// Verify field round-trip at the semantic level.
	// Note: Cursor is a lossy target — tools: and unmapped model strings are
	// dropped by the renderer with fidelity notes. We assert what Cursor does preserve.
	reviewerContent := out.Files[filepath.Clean("agents/reviewer.md")]
	assert.Contains(t, reviewerContent, "Code Reviewer",
		"compiled agent must preserve the name field")
	assert.Contains(t, reviewerContent, "Reviews pull requests",
		"compiled agent must preserve the description field")

	formattingContent := out.Files[filepath.Clean("rules/formatting.mdc")]
	assert.Contains(t, formattingContent, "Code formatting standards",
		"compiled rule must preserve the description field")

	skillContent := out.Files[filepath.Clean("skills/code-review/SKILL.md")]
	assert.Contains(t, skillContent, "code-review",
		"compiled skill must preserve the name field")
}

// TestRoundTrip_CursorImportCompile_AgentFields verifies that Cursor agent
// fields survive the round-trip at the level of fidelity Cursor supports.
// Cursor does not emit tools: (no tool-gating concept) and drops unmapped
// model strings, so only name, description, and the instruction body are checked.
func TestRoundTrip_CursorImportCompile_AgentFields(t *testing.T) {
	imp := cursorimp.NewImporter()
	config := &ast.XcaffoldConfig{}
	require.NoError(t, imp.Import(testdataDir("cursor"), config))

	agent, ok := config.Agents["reviewer"]
	require.True(t, ok, "agent 'reviewer' must be present after import")

	// Verify AST-level fields were imported correctly.
	assert.Equal(t, "Code Reviewer", agent.Name)
	assert.Equal(t, "claude-opus-4-5", agent.Model)
	assert.NotEmpty(t, agent.Tools, "tools must be imported into the AST")
	assert.NotEmpty(t, agent.Body, "instructions body must be imported")

	out, _, err := compiler.Compile(config, ".", "cursor", "")
	require.NoError(t, err)

	// Cursor emits name, description, and body — but not tools: or unmapped models.
	content := out.Files[filepath.Clean("agents/reviewer.md")]
	assert.Contains(t, content, "Code Reviewer",
		"compiled agent must preserve the name field")
	assert.Contains(t, content, "You are a code reviewer",
		"compiled agent must preserve instruction body")
}

// TestRoundTrip_CursorImportCompile_RuleExtension verifies that Cursor rules
// are emitted with the .mdc extension (not .md) after a round-trip.
func TestRoundTrip_CursorImportCompile_RuleExtension(t *testing.T) {
	imp := cursorimp.NewImporter()
	config := &ast.XcaffoldConfig{}
	require.NoError(t, imp.Import(testdataDir("cursor"), config))

	out, _, err := compiler.Compile(config, ".", "cursor", "")
	require.NoError(t, err)

	// The rule must appear as .mdc, not .md.
	assert.Contains(t, out.Files, filepath.Clean("rules/formatting.mdc"),
		"cursor rules must use .mdc extension")
	assert.NotContains(t, out.Files, filepath.Clean("rules/formatting.md"),
		"cursor rules must NOT use .md extension")
}

// TestRoundTrip_CopilotImportCompile imports a Copilot .github/ directory tree
// and compiles the AST back to the Copilot target, verifying each imported
// resource appears in the compilation output.
func TestRoundTrip_CopilotImportCompile(t *testing.T) {
	// --- Red: import phase ---
	imp := copilotimp.NewImporter()
	config := &ast.XcaffoldConfig{}
	err := imp.Import(testdataDir("copilot"), config)
	require.NoError(t, err, "Import must not fail on well-formed testdata")

	// Sanity-check that the importer populated the expected resources.
	require.Contains(t, config.Agents, "auditor", "agent 'auditor' must be imported")
	require.Contains(t, config.Rules, "security", "rule 'security' must be imported")
	require.Contains(t, config.Skills, "review", "skill 'review' must be imported")

	// --- Green: compile phase ---
	out, _, err := compiler.Compile(config, ".", "copilot", "")
	require.NoError(t, err, "Compile must not fail on a valid AST")

	// --- Verify: same resource keys are present in the output ---
	assert.Contains(t, out.Files, filepath.Clean("agents/auditor.agent.md"),
		"compiled output must contain agents/auditor.agent.md")
	assert.Contains(t, out.Files, filepath.Clean("instructions/security.instructions.md"),
		"compiled output must contain instructions/security.instructions.md")
	assert.Contains(t, out.Files, filepath.Clean("skills/review/SKILL.md"),
		"compiled output must contain skills/review/SKILL.md")

	// Verify field round-trip at the semantic level.
	auditorContent := out.Files[filepath.Clean("agents/auditor.agent.md")]
	assert.Contains(t, auditorContent, "Code Auditor",
		"compiled agent must preserve the name field")
}

// TestRoundTrip_AntigravityImportCompile imports an Antigravity .agents/ directory tree
// and compiles the AST back to the Antigravity target, verifying each imported
// resource appears in the compilation output.
func TestRoundTrip_AntigravityImportCompile(t *testing.T) {
	// --- Red: import phase ---
	imp := antigravityimp.NewImporter()
	config := &ast.XcaffoldConfig{}
	err := imp.Import(testdataDir("antigravity"), config)
	require.NoError(t, err, "Import must not fail on well-formed testdata")

	// Sanity-check that the importer populated the expected resources.
	require.Contains(t, config.Rules, "safety", "rule 'safety' must be imported")
	require.Contains(t, config.Skills, "search", "skill 'search' must be imported")

	// --- Green: compile phase ---
	out, _, err := compiler.Compile(config, ".", "antigravity", "")
	require.NoError(t, err, "Compile must not fail on a valid AST")

	// --- Verify: same resource keys are present in the output ---
	assert.Contains(t, out.Files, filepath.Clean("rules/safety.md"),
		"compiled output must contain rules/safety.md")
	assert.Contains(t, out.Files, filepath.Clean("skills/search/SKILL.md"),
		"compiled output must contain skills/search/SKILL.md")
}

// TestRoundTrip_GeminiImportCompile imports a Gemini .gemini/ directory tree
// and compiles the AST back to the Gemini target, verifying each imported
// resource appears in the compilation output.
func TestRoundTrip_GeminiImportCompile(t *testing.T) {
	// --- Red: import phase ---
	imp := geminimp.NewImporter()
	config := &ast.XcaffoldConfig{}
	err := imp.Import(testdataDir("gemini"), config)
	require.NoError(t, err, "Import must not fail on well-formed testdata")

	// Sanity-check that the importer populated the expected resources.
	require.Contains(t, config.Agents, "assistant", "agent 'assistant' must be imported")
	require.Contains(t, config.Rules, "style", "rule 'style' must be imported")
	require.Contains(t, config.Skills, "search", "skill 'search' must be imported")

	// --- Green: compile phase ---
	out, _, err := compiler.Compile(config, ".", "gemini", "")
	require.NoError(t, err, "Compile must not fail on a valid AST")

	// --- Verify: same resource keys are present in the output ---
	assert.Contains(t, out.Files, filepath.Clean("agents/assistant.md"),
		"compiled output must contain agents/assistant.md")
	assert.Contains(t, out.Files, filepath.Clean("rules/style.md"),
		"compiled output must contain rules/style.md")
	assert.Contains(t, out.Files, filepath.Clean("skills/search/SKILL.md"),
		"compiled output must contain skills/search/SKILL.md")

	// Verify field round-trip at the semantic level.
	assistantContent := out.Files[filepath.Clean("agents/assistant.md")]
	assert.Contains(t, assistantContent, "Assistant Agent",
		"compiled agent must preserve the name field")
	assert.Contains(t, assistantContent, "gemini-2.5-pro",
		"compiled agent must preserve the model field")

	skillContent := out.Files[filepath.Clean("skills/search/SKILL.md")]
	assert.Contains(t, skillContent, "web-search",
		"compiled skill must preserve the name field")
}
