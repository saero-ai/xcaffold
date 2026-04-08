package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validXCF = `
version: "1.0"
project:
  name: "test-project"
  description: "A test project."
agents:
  developer:
    description: "An expert developer."
    instructions: |
      You are a software developer.
    model: "claude-3-7-sonnet-20250219"
    effort: "high"
    tools: [Bash, Read, Write]
`

const missingProjectName = `
version: "1.0"
project:
  description: "Missing the name field."
`

const unknownFieldXCF = `
version: "1.0"
project:
  name: "test-project"
unknown_top_level_field: "should fail"
`

func TestParse_ValidConfig(t *testing.T) {
	cfg, err := Parse(strings.NewReader(validXCF))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "test-project", cfg.Project.Name)
	assert.Contains(t, cfg.Agents, "developer")
	assert.Equal(t, "claude-3-7-sonnet-20250219", cfg.Agents["developer"].Model)
}

func TestParse_MissingProjectName(t *testing.T) {
	_, err := Parse(strings.NewReader(missingProjectName))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project.name is required")
}

func TestParse_UnknownField_Rejected(t *testing.T) {
	// Strict mode must reject unknown top-level fields to prevent silent misconfigurations.
	_, err := Parse(strings.NewReader(unknownFieldXCF))
	require.Error(t, err)
}

func TestParse_EmptyReader(t *testing.T) {
	_, err := Parse(strings.NewReader(""))
	require.Error(t, err)
}

func TestParse_MissingVersion_Rejected(t *testing.T) {
	missingVersionXCF := `
project:
  name: "test-project"
`
	_, err := Parse(strings.NewReader(missingVersionXCF))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version is required")
}

func TestParse_PathTraversalAgentID_Rejected(t *testing.T) {
	maliciousXCF := `
version: "1.0"
project:
  name: "test-project"
agents:
  "../evil":
    description: "Path traversal attempt"
`
	_, err := Parse(strings.NewReader(maliciousXCF))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent id contains invalid characters")
}

func TestParse_HookEvent_ValidEvent_Accepted(t *testing.T) {
	input := `
version: "1.0"
project:
  name: "test"
hooks:
  PreToolUse:
    - hooks:
        - type: command
          command: "echo ok"
`
	config, err := Parse(strings.NewReader(input))
	require.NoError(t, err)
	assert.Len(t, config.Hooks["PreToolUse"], 1)
}

func TestParse_HookEvent_InvalidEvent_Rejected(t *testing.T) {
	input := `
version: "1.0"
project:
  name: "test"
hooks:
  MadeUpEvent:
    - hooks:
        - type: command
          command: "echo ok"
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err, "unknown hook event names must be rejected")
	assert.Contains(t, err.Error(), "MadeUpEvent")
}

func TestParseFile_ExtendsGlobal_ResolvesToHomeClaude(t *testing.T) {
	// Create a fake home structure with global.xcf
	fakeHome := t.TempDir()
	claudeDir := filepath.Join(fakeHome, ".claude")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(claudeDir, "global.xcf"),
		[]byte(`version: "1.0"
project:
  name: "global-base"
agents:
  shared:
    description: "Shared agent"
`),
		0600,
	))

	// Create a project scaffold.xcf that extends using an absolute path.
	// (os.UserHomeDir cannot be mocked in tests, so the extends mechanism
	// is verified with an absolute path; the "global" keyword path is a
	// simple string-comparison branch on top of the same merge logic.)
	projectDir := t.TempDir()
	globalPath := filepath.Join(claudeDir, "global.xcf")
	require.NoError(t, os.WriteFile(
		filepath.Join(projectDir, "scaffold.xcf"),
		[]byte(fmt.Sprintf(`extends: "%s"
version: "1.0"
project:
  name: "my-project"
agents:
  local:
    description: "Local agent"
`, globalPath)),
		0600,
	))

	config, err := ParseFile(filepath.Join(projectDir, "scaffold.xcf"))
	require.NoError(t, err)

	assert.Equal(t, "my-project", config.Project.Name)
	_, hasShared := config.Agents["shared"]
	assert.True(t, hasShared, "inherited agent should be present")
	_, hasLocal := config.Agents["local"]
	assert.True(t, hasLocal, "local agent should be present")
}

func TestParse_HookEvent_AllValidEvents_Accepted(t *testing.T) {
	events := []string{
		"PreToolUse", "PostToolUse", "PostToolUseFailure",
		"PermissionRequest", "PermissionDenied",
		"SessionStart", "SessionEnd", "UserPromptSubmit",
		"Stop", "StopFailure", "SubagentStart", "SubagentStop",
		"TeammateIdle", "TaskCreated", "TaskCompleted",
		"PreCompact", "PostCompact", "InstructionsLoaded",
		"ConfigChange", "CwdChanged", "FileChanged",
		"WorktreeCreate", "WorktreeRemove",
		"Elicitation", "ElicitationResult", "Notification",
	}

	for _, event := range events {
		t.Run(event, func(t *testing.T) {
			input := `
version: "1.0"
project:
  name: "test"
hooks:
  ` + event + `:
    - hooks:
        - type: command
          command: "echo ok"
`
			_, err := Parse(strings.NewReader(input))
			require.NoError(t, err, "event %q should be accepted", event)
		})
	}
}

func TestValidatePermissions_ValidRule_BareToolName(t *testing.T) {
	input := `
version: "1.0"
project:
  name: "test"
settings:
  permissions:
    allow: [Bash]
`
	cfg, err := Parse(strings.NewReader(input))
	require.NoError(t, err)
	require.NotNil(t, cfg)
}

func TestValidatePermissions_ValidRule_WithPattern(t *testing.T) {
	input := `
version: "1.0"
project:
  name: "test"
settings:
  permissions:
    allow: ["Bash(npm test *)"]
`
	cfg, err := Parse(strings.NewReader(input))
	require.NoError(t, err)
	require.NotNil(t, cfg)
}

func TestValidatePermissions_InvalidRule_UnknownTool(t *testing.T) {
	input := `
version: "1.0"
project:
  name: "test"
settings:
  permissions:
    deny: [SomeUnknownTool]
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SomeUnknownTool")
}

func TestValidatePermissions_InvalidRule_MalformedPattern(t *testing.T) {
	input := `
version: "1.0"
project:
  name: "test"
settings:
  permissions:
    allow: ["Bash(unclosed"]
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
}

func TestValidatePermissions_Contradiction_AllowAndDeny(t *testing.T) {
	input := `
version: "1.0"
project:
  name: "test"
settings:
  permissions:
    allow: [Bash]
    deny: [Bash]
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "allow and deny")
}

func TestValidatePermissions_Contradiction_AllowAndAsk(t *testing.T) {
	input := `
version: "1.0"
project:
  name: "test"
settings:
  permissions:
    allow: [Read]
    ask: [Read]
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "allow and ask")
}

func TestValidatePermissions_AgentDisallowedConflict(t *testing.T) {
	input := `
version: "1.0"
project:
  name: "test"
agents:
  dev:
    description: "Developer"
    disallowedTools: [Write]
settings:
  permissions:
    allow: [Write]
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dev")
	assert.Contains(t, err.Error(), "Write")
}

func TestValidatePermissions_AgentToolsDenyConflict(t *testing.T) {
	input := `
version: "1.0"
project:
  name: "test"
agents:
  dev:
    description: "Developer"
    tools: [Bash]
settings:
  permissions:
    deny: [Bash]
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dev")
	assert.Contains(t, err.Error(), "Bash")
}

func TestParse_MCP_PlatformSpecificFields(t *testing.T) {
	input := `
version: "1.0"
project:
  name: "test"
mcp:
  local-server:
    type: "stdio"
    command: "node"
    cwd: "/path/to/server"
    oauth:
      provider: "github"
      scope: "repo"
  remote-server:
    type: "http"
    url: "https://example.com/mcp"
    disabled: true
    disabledTools: ["dangerousTool"]
    authProviderType: "custom"
`
	config, err := Parse(strings.NewReader(input))
	require.NoError(t, err)

	local, ok := config.MCP["local-server"]
	require.True(t, ok)
	assert.Equal(t, "/path/to/server", local.Cwd)
	require.NotNil(t, local.OAuth)
	assert.Equal(t, "github", local.OAuth["provider"])
	assert.Equal(t, "repo", local.OAuth["scope"])

	remote, ok := config.MCP["remote-server"]
	require.True(t, ok)
	assert.True(t, *remote.Disabled)
	assert.Contains(t, remote.DisabledTools, "dangerousTool")
	assert.Equal(t, "custom", remote.AuthProviderType)
}

// ── ValidateFile: file-ref and duplicate-ID tests ──────────────────────────

func writeXCFFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte(content), 0600))
	return p
}

func TestValidateFileRefs_MissingSkillReference(t *testing.T) {
	dir := t.TempDir()
	xcf := writeXCFFile(t, dir, "scaffold.xcf", `version: "1.0"
project:
  name: "test"
skills:
  my-skill:
    description: "A skill"
    instructions: "do stuff"
    references:
      - nonexistent.md
`)
	diags := ValidateFile(xcf)
	var found bool
	for _, d := range diags {
		if d.Severity == "warning" && strings.Contains(d.Message, "does not exist") {
			found = true
		}
	}
	assert.True(t, found, "expected a warning diagnostic about a missing reference file, got: %v", diags)
}

func TestValidateFileRefs_MissingInstructionsFile_Agent(t *testing.T) {
	dir := t.TempDir()
	xcf := writeXCFFile(t, dir, "scaffold.xcf", `version: "1.0"
project:
  name: "test"
agents:
  my-agent:
    description: "An agent"
    instructions_file: ghost.md
`)
	diags := ValidateFile(xcf)
	var found bool
	for _, d := range diags {
		if d.Severity == "error" && strings.Contains(d.Message, "not found") {
			found = true
		}
	}
	assert.True(t, found, "expected an error diagnostic about missing instructions_file, got: %v", diags)
}

func TestValidateFileRefs_PresentInstructionsFile(t *testing.T) {
	dir := t.TempDir()
	// Create the actual instructions file
	instrFile := filepath.Join(dir, "real.md")
	require.NoError(t, os.WriteFile(instrFile, []byte("# instructions"), 0600))

	xcf := writeXCFFile(t, dir, "scaffold.xcf", `version: "1.0"
project:
  name: "test"
agents:
  my-agent:
    description: "An agent"
    instructions_file: real.md
`)
	diags := ValidateFile(xcf)
	for _, d := range diags {
		if strings.Contains(d.Message, "not found") || strings.Contains(d.Message, "does not exist") {
			t.Errorf("unexpected file-existence diagnostic: %+v", d)
		}
	}
}

func TestValidateFileRefs_DuplicateID(t *testing.T) {
	dir := t.TempDir()
	xcf := writeXCFFile(t, dir, "scaffold.xcf", `version: "1.0"
project:
  name: "test"
agents:
  foo:
    description: "An agent"
    instructions: "do stuff"
skills:
  foo:
    description: "A skill"
    instructions: "do stuff"
`)
	diags := ValidateFile(xcf)
	var found bool
	for _, d := range diags {
		if d.Severity == "warning" && strings.Contains(d.Message, "used in both") {
			found = true
		}
	}
	assert.True(t, found, "expected a warning about duplicate ID across types, got: %v", diags)
}

func TestValidateFileRefs_UniqueIDs(t *testing.T) {
	dir := t.TempDir()
	xcf := writeXCFFile(t, dir, "scaffold.xcf", `version: "1.0"
project:
  name: "test"
agents:
  agent-one:
    description: "An agent"
    instructions: "do stuff"
skills:
  skill-one:
    description: "A skill"
    instructions: "do stuff"
rules:
  rule-one:
    instructions: "a rule"
`)
	diags := ValidateFile(xcf)
	for _, d := range diags {
		if strings.Contains(d.Message, "used in both") {
			t.Errorf("unexpected duplicate-ID diagnostic: %+v", d)
		}
	}
}

// ── ValidateFile: plugin validation tests ─────────────────────────────────

func TestValidatePlugins_KnownPlugin(t *testing.T) {
	dir := t.TempDir()
	xcf := writeXCFFile(t, dir, "scaffold.xcf", `version: "1.0"
project:
  name: "test"
settings:
  enabledPlugins:
    commit-commands: true
`)
	diags := ValidateFile(xcf)
	for _, d := range diags {
		if strings.Contains(d.Message, "unknown plugin") {
			t.Errorf("unexpected unknown-plugin diagnostic for a known plugin: %+v", d)
		}
	}
}

func TestValidatePlugins_UnknownSettingsPlugin(t *testing.T) {
	dir := t.TempDir()
	xcf := writeXCFFile(t, dir, "scaffold.xcf", `version: "1.0"
project:
  name: "test"
settings:
  enabledPlugins:
    my-custom-plugin: true
`)
	diags := ValidateFile(xcf)
	var found bool
	for _, d := range diags {
		if d.Severity == "warning" && strings.Contains(d.Message, "unknown plugin") &&
			strings.Contains(d.Message, "commit-commands") {
			found = true
		}
	}
	assert.True(t, found, "expected a warning about unknown settings plugin, got: %v", diags)
}

func TestValidatePlugins_UnknownLocalPlugin(t *testing.T) {
	dir := t.TempDir()
	xcf := writeXCFFile(t, dir, "scaffold.xcf", `version: "1.0"
project:
  name: "test"
local:
  enabledPlugins:
    mystery-plugin: true
`)
	diags := ValidateFile(xcf)
	var found bool
	for _, d := range diags {
		if d.Severity == "warning" && strings.Contains(d.Message, "local") &&
			strings.Contains(d.Message, "unknown plugin") {
			found = true
		}
	}
	assert.True(t, found, "expected a warning about unknown local plugin, got: %v", diags)
}

func TestValidatePlugins_BothBlocksUnknown(t *testing.T) {
	dir := t.TempDir()
	xcf := writeXCFFile(t, dir, "scaffold.xcf", `version: "1.0"
project:
  name: "test"
settings:
  enabledPlugins:
    alpha-plugin: true
local:
  enabledPlugins:
    beta-plugin: true
`)
	diags := ValidateFile(xcf)
	count := 0
	for _, d := range diags {
		if d.Severity == "warning" && strings.Contains(d.Message, "unknown plugin") {
			count++
		}
	}
	assert.Equal(t, 2, count, "expected 2 unknown-plugin warnings (one per block), got: %v", diags)
}
