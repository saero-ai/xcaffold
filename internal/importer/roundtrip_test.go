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
	claudeimp "github.com/saero-ai/xcaffold/internal/importer/claude"
	cursorimp "github.com/saero-ai/xcaffold/internal/importer/cursor"
)

// testdataDir returns the absolute path to a provider's testdata/input directory.
// It resolves relative to this test file's location so tests are location-independent.
func testdataDir(provider string) string {
	_, file, _, _ := runtime.Caller(0)
	base := filepath.Dir(file)
	return filepath.Join(base, provider, "testdata", "input")
}

// TestRoundTrip_ClaudeImportCompile imports a Claude .claude/ directory tree
// and compiles the AST back to the Claude target, then verifies that each
// imported resource appears in the compilation output.
func TestRoundTrip_ClaudeImportCompile(t *testing.T) {
	// --- Red: import phase ---
	imp := claudeimp.New()
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
	imp := claudeimp.New()
	config := &ast.XcaffoldConfig{}
	require.NoError(t, imp.Import(testdataDir("claude"), config))

	agent, ok := config.Agents["backend"]
	require.True(t, ok, "agent 'backend' must be present after import")

	assert.Equal(t, "Backend Agent", agent.Name)
	assert.Equal(t, "claude-opus-4-5", agent.Model)
	assert.NotEmpty(t, agent.Tools, "tools must be imported")
	assert.NotEmpty(t, agent.Instructions, "instructions body must be imported")

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
	imp := claudeimp.New()
	config := &ast.XcaffoldConfig{}
	require.NoError(t, imp.Import(testdataDir("claude"), config))

	skill, ok := config.Skills["tdd"]
	require.True(t, ok, "skill 'tdd' must be present after import")

	assert.Equal(t, "tdd-driven-development", skill.Name)
	assert.NotEmpty(t, skill.AllowedTools, "allowed-tools must be imported")
	assert.NotEmpty(t, skill.Instructions, "instructions body must be imported")

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
	imp := cursorimp.New()
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
	imp := cursorimp.New()
	config := &ast.XcaffoldConfig{}
	require.NoError(t, imp.Import(testdataDir("cursor"), config))

	agent, ok := config.Agents["reviewer"]
	require.True(t, ok, "agent 'reviewer' must be present after import")

	// Verify AST-level fields were imported correctly.
	assert.Equal(t, "Code Reviewer", agent.Name)
	assert.Equal(t, "claude-opus-4-5", agent.Model)
	assert.NotEmpty(t, agent.Tools, "tools must be imported into the AST")
	assert.NotEmpty(t, agent.Instructions, "instructions body must be imported")

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
	imp := cursorimp.New()
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
