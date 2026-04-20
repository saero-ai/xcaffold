package parser

import (
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParse_UnknownFieldRejected verifies that strict YAML mode rejects unknown fields.
func TestParse_UnknownFieldRejected(t *testing.T) {
	yaml := `kind: global
version: "1.0"
agents:
  dev:
    description: "dev"
unknown_field: "this should not be allowed"
`
	_, err := Parse(strings.NewReader(yaml))
	require.Error(t, err, "unknown top-level fields must be rejected in strict mode")
}

// TestParse_AgentIDWithDotDot verifies that ".." agent IDs are rejected.
func TestParse_AgentIDWithDotDot(t *testing.T) {
	yaml := `kind: global
version: "1.0"
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
	yaml := `kind: global
version: "1.0"
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
	yaml := `kind: global
version: "1.0"
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
	yaml := `kind: project
version: "1.0"
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
	yaml := `kind: project
version: "1.0"
name: "no-agents-project"
`
	cfg, err := Parse(strings.NewReader(yaml))
	require.NoError(t, err, "missing agents block should not be an error")
	require.NotNil(t, cfg)
	assert.Empty(t, cfg.Agents)
}

// TestParse_AllOptionalBlocks verifies that a config with all optional blocks populated parses fully.
func TestParse_AllOptionalBlocks(t *testing.T) {
	yaml := `---
kind: project
version: "1.0"
name: "full-project"
description: "All blocks populated"
test:
  claude-path: "/usr/local/bin/claude"
  judge-model: "claude-3-5-haiku-20241022"
---
kind: global
version: "1.0"
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
  PreToolUse:
    - matcher: Bash
      hooks:
        - type: command
          command: echo hello
mcp:
  my-server:
    command: "npx"
    args: ["-y", "my-mcp-server"]
`
	cfg, err := Parse(strings.NewReader(yaml))
	require.NoError(t, err, "fully populated config should parse without errors")
	require.NotNil(t, cfg)
	assert.Contains(t, cfg.Agents, "my-agent")
	assert.Contains(t, cfg.Skills, "my-skill")
	assert.Contains(t, cfg.Rules, "my-rule")
	assert.Contains(t, cfg.Hooks["default"].Events, "PreToolUse")
	assert.Contains(t, cfg.MCP, "my-server")
	require.NotNil(t, cfg.Project)
	assert.Equal(t, "/usr/local/bin/claude", cfg.Project.Test.ClaudePath)
	assert.Equal(t, "claude-3-5-haiku-20241022", cfg.Project.Test.JudgeModel)
}

// TestParse_MissingVersion verifies that a missing version field causes an error.
func TestParse_MissingVersion(t *testing.T) {
	yaml := `kind: project
name: "my-project"
`
	_, err := Parse(strings.NewReader(yaml))
	require.Error(t, err, "missing version must be rejected")
	assert.Contains(t, err.Error(), "version is required")
}

// TestParse_WhitespaceOnlyProjectName verifies that a whitespace-only project name is rejected.
func TestParse_WhitespaceOnlyProjectName(t *testing.T) {
	yaml := `kind: project
version: "1.0"
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
	yaml := `kind: global
version: "1.0"
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
	yaml := `kind: global
version: "1.0"
extends: "base.xcf"
`
	cfg, err := Parse(strings.NewReader(yaml))
	require.NoError(t, err, "with extends set, empty project.name should be allowed")
	require.NotNil(t, cfg)
	assert.Equal(t, "base.xcf", cfg.Extends)
	assert.Nil(t, cfg.Project)
}

// TestParse_SkillIDWithPathTraversal tests whether the parser catches path traversal in skill IDs.
// BUG PROBE: validate() only checks agent IDs — skill IDs are NOT validated.
// This test is expected to PASS (no error) if the parser does NOT validate skill IDs,
// revealing a security gap.
func TestParse_SkillIDWithPathTraversal(t *testing.T) {
	yaml := `kind: global
version: "1.0"
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
	yaml := `kind: global
version: "1.0"
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

// TestParse_HookEventKey_ValidStructure tests that hooks parse with the new 3-level structure.
func TestParse_HookEventKey_ValidStructure(t *testing.T) {
	yaml := `kind: global
version: "1.0"
hooks:
  PostToolUse:
    - matcher: "Write"
      hooks:
        - type: command
          command: "npx prettier --write $FILE"
`
	cfg, err := Parse(strings.NewReader(yaml))
	require.NoError(t, err, "valid hooks structure should parse without errors")
	require.NotNil(t, cfg)
	var effectiveHooks ast.HookConfig
	if dh, ok := cfg.Hooks["default"]; ok {
		effectiveHooks = dh.Events
	}
	assert.Contains(t, effectiveHooks, "PostToolUse")
	assert.Len(t, effectiveHooks["PostToolUse"], 1)
	assert.Equal(t, "Write", effectiveHooks["PostToolUse"][0].Matcher)
}

// TestParse_InstructionsAndFileSet_ReturnsError verifies that setting both
// instructions: and instructions-file: on the same agent is a parse error.
func TestParse_InstructionsAndFileSet_ReturnsError(t *testing.T) {
	yaml := `kind: global
version: "1.0"
agents:
  ambiguous:
    instructions: "Inline instructions."
    instructions-file: "agents/ambiguous.md"
`
	_, err := Parse(strings.NewReader(yaml))
	require.Error(t, err, "both instructions and instructions-file set must be rejected")
	assert.Contains(t, err.Error(), "mutually exclusive")
}

// TestParse_InstructionsFile_AbsolutePath_Rejected verifies that an absolute path
// in instructions-file is rejected at parse time.
func TestParse_InstructionsFile_AbsolutePath_Rejected(t *testing.T) {
	yaml := `kind: global
version: "1.0"
agents:
  cto:
    instructions-file: "/etc/passwd"
`
	_, err := Parse(strings.NewReader(yaml))
	require.Error(t, err, "absolute instructions-file path must be rejected at parse time")
	assert.Contains(t, err.Error(), "relative path")
}

// TestParse_InstructionsFile_PathTraversal_Rejected verifies that a path
// traversal attempt in instructions-file is rejected at parse time.
func TestParse_InstructionsFile_PathTraversal_Rejected(t *testing.T) {
	yaml := `kind: global
version: "1.0"
agents:
  cto:
    instructions-file: "../outside/cto.md"
`
	_, err := Parse(strings.NewReader(yaml))
	require.Error(t, err, "instructions-file with path traversal must be rejected at parse time")
	assert.Contains(t, err.Error(), "instructions-file")
}

// TestParse_SkillInstructionsFile_Valid verifies skills accept instructions-file.
func TestParse_SkillInstructionsFile_Valid(t *testing.T) {
	yaml := `kind: global
version: "1.0"
skills:
  flutter-integration:
    description: "Flutter integration skill"
    instructions-file: "skills/flutter-integration/SKILL.md"
    references:
      - "skills/flutter-integration/references/advanced-patterns.md"
`
	config, err := Parse(strings.NewReader(yaml))
	require.NoError(t, err, "skill with instructions-file and references should be accepted")
	skill, ok := config.Skills["flutter-integration"]
	require.True(t, ok)
	assert.Equal(t, "skills/flutter-integration/SKILL.md", skill.InstructionsFile)
	assert.Len(t, skill.References, 1)
}

// TestParse_InstructionsFile_CircularReference_ClaudeDir verifies that instructions-file
// pointing into .claude/ is rejected as a circular dependency.
func TestParse_InstructionsFile_CircularReference_ClaudeDir(t *testing.T) {
	yaml := `kind: global
version: "1.0"
agents:
  dev:
    instructions-file: .claude/agents/dev.md
`
	_, err := Parse(strings.NewReader(yaml))
	require.Error(t, err, "instructions-file pointing to .claude/ must be rejected")
	assert.Contains(t, err.Error(), "circular dependency")
	assert.Contains(t, err.Error(), ".claude/")
}

// TestParse_InstructionsFile_CircularReference_CursorDir verifies that instructions-file
// pointing into .cursor/ is rejected as a circular dependency.
func TestParse_InstructionsFile_CircularReference_CursorDir(t *testing.T) {
	yaml := `kind: global
version: "1.0"
skills:
  deploy:
    instructions-file: .cursor/rules/deploy.mdc
`
	_, err := Parse(strings.NewReader(yaml))
	require.Error(t, err, "instructions-file pointing to .cursor/ must be rejected")
	assert.Contains(t, err.Error(), "circular dependency")
	assert.Contains(t, err.Error(), ".cursor/")
}

// TestParse_InstructionsFile_CircularReference_AgentsDir verifies that instructions-file
// pointing into .agents/ (Antigravity output) is rejected.
func TestParse_InstructionsFile_CircularReference_AgentsDir(t *testing.T) {
	yaml := `kind: global
version: "1.0"
agents:
  reviewer:
    instructions-file: .agents/agents/reviewer.md
`
	_, err := Parse(strings.NewReader(yaml))
	require.Error(t, err, "instructions-file pointing to .agents/ must be rejected")
	assert.Contains(t, err.Error(), "circular dependency")
	assert.Contains(t, err.Error(), ".agents/")
}

// TestParse_InstructionsFile_ValidRelativePath_Allowed ensures legitimate relative paths
// are NOT rejected by the circular reference check.
func TestParse_InstructionsFile_ValidRelativePath_Allowed(t *testing.T) {
	yaml := `kind: global
version: "1.0"
agents:
  dev:
    instructions-file: docs/agents/dev.md
`
	cfg, err := Parse(strings.NewReader(yaml))
	require.NoError(t, err, "valid relative path must not be rejected")
	assert.Equal(t, "docs/agents/dev.md", cfg.Agents["dev"].InstructionsFile)
}

// TestParse_InstructionsFile_CircularReference_AntigravityDir verifies that instructions-file
// pointing into .antigravity/ is rejected.
func TestParse_InstructionsFile_CircularReference_AntigravityDir(t *testing.T) {
	yaml := `kind: global
version: "1.0"
rules:
  security:
    instructions-file: .antigravity/rules/security.md
`
	_, err := Parse(strings.NewReader(yaml))
	require.Error(t, err, "instructions-file pointing to .antigravity/ must be rejected")
	assert.Contains(t, err.Error(), "circular dependency")
	assert.Contains(t, err.Error(), ".antigravity/")
}

// TestParsePartial_GlobalScope_AllowsAbsoluteInstructionsFile verifies that
// parsePartial called with withGlobalScope() accepts absolute instructions-file
// paths — global configs legitimately reference files like ~/.claude/agents/*.md.
func TestParsePartial_GlobalScope_AllowsAbsoluteInstructionsFile(t *testing.T) {
	xcf := `kind: global
version: "1.0"
agents:
  ceo:
    description: "Global CEO agent"
    instructions-file: "/Users/testuser/.claude/agents/ceo.md"
`
	cfg, err := parsePartial(strings.NewReader(xcf), withGlobalScope())
	require.NoError(t, err, "global scope must accept absolute instructions-file path")
	assert.Equal(t, "/Users/testuser/.claude/agents/ceo.md", cfg.Agents["ceo"].InstructionsFile)
}

// TestParsePartial_ProjectScope_RejectsAbsoluteInstructionsFile verifies that
// parsePartial without withGlobalScope() still rejects absolute paths —
// project-scoped configs must not read arbitrary absolute files.
func TestParsePartial_ProjectScope_RejectsAbsoluteInstructionsFile(t *testing.T) {
	xcf := `kind: global
version: "1.0"
agents:
  ceo:
    description: "Project CEO agent"
    instructions-file: "/etc/passwd"
`
	_, err := parsePartial(strings.NewReader(xcf))
	require.Error(t, err, "project scope must reject absolute instructions-file path")
	assert.Contains(t, err.Error(), "relative path")
}

// TestParsePartial_GlobalScope_StillRejectsPathTraversal verifies that
// withGlobalScope() does not relax the path-traversal guard — ".." remains
// invalid even in global scope.
func TestParsePartial_GlobalScope_StillRejectsPathTraversal(t *testing.T) {
	xcf := `kind: global
version: "1.0"
agents:
  ceo:
    description: "Global CEO agent"
    instructions-file: "../../etc/passwd"
`
	_, err := parsePartial(strings.NewReader(xcf), withGlobalScope())
	require.Error(t, err, "global scope must still reject path traversal")
	assert.Contains(t, err.Error(), "instructions-file")
}

// TestParseDirectoryRaw_GlobalScope_AllowsAbsoluteInstructionsFile verifies that
// parseDirectoryRaw propagates withGlobalScope() into parseFileExact so that an
// XCF file inside ~/.xcaffold/ with an absolute instructions-file parses without error.
func TestParseDirectoryRaw_GlobalScope_AllowsAbsoluteInstructionsFile(t *testing.T) {
	dir := t.TempDir()
	writeTestXCF(t, dir, "agents.xcf", `kind: global
version: "1.0"
agents:
  ceo:
    description: "Global CEO agent"
    instructions-file: "/Users/testuser/.claude/agents/ceo.md"
`)

	cfg, err := parseDirectoryRaw(dir, withGlobalScope())
	require.NoError(t, err, "parseDirectoryRaw with global scope must accept absolute instructions-file")
	assert.Equal(t, "/Users/testuser/.claude/agents/ceo.md", cfg.Agents["ceo"].InstructionsFile)
}

// TestParseDirectoryRaw_ProjectScope_RejectsAbsoluteInstructionsFile verifies that
// parseDirectoryRaw without withGlobalScope() still rejects absolute paths.
func TestParseDirectoryRaw_ProjectScope_RejectsAbsoluteInstructionsFile(t *testing.T) {
	dir := t.TempDir()
	writeTestXCF(t, dir, "agents.xcf", `kind: global
version: "1.0"
agents:
  ceo:
    description: "Project CEO agent"
    instructions-file: "/etc/passwd"
`)

	_, err := parseDirectoryRaw(dir)
	require.Error(t, err, "parseDirectoryRaw without global scope must reject absolute instructions-file")
	assert.Contains(t, err.Error(), "relative path")
}
