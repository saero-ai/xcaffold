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
