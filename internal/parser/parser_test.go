package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	_ "github.com/saero-ai/xcaffold/providers/antigravity"
	_ "github.com/saero-ai/xcaffold/providers/claude"
	_ "github.com/saero-ai/xcaffold/providers/copilot"
	_ "github.com/saero-ai/xcaffold/providers/cursor"
	_ "github.com/saero-ai/xcaffold/providers/gemini"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validXCAF = `---
kind: global
version: "1.0"
agents:
  developer:
    description: "An expert developer."
    model: "claude-3-7-sonnet-20250219"
    effort: "high"
    tools: [Bash, Read, Write]
---
You are a software developer.
`

const missingProjectName = `kind: project
version: "1.0"
description: "Missing the name field."
`

const unknownFieldXCAF = `kind: global
version: "1.0"
agents:
  dev:
    description: "developer"
unknown_top_level_field: "should fail"
`

func TestParse_ValidConfig(t *testing.T) {
	cfg, err := Parse(strings.NewReader(validXCAF))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Contains(t, cfg.Agents, "developer")
	assert.Equal(t, "claude-3-7-sonnet-20250219", cfg.Agents["developer"].Model)

	// Add new test for ProjectConfig.AllowedEnvVars
	projectConfigYAML := `---
kind: project
version: "1.0"
name: "test-project-with-env"
allowed-env-vars:
  - "MY_ENV_VAR"
  - "ANOTHER_ENV_VAR"
`
	projectCfg, err := Parse(strings.NewReader(projectConfigYAML))
	require.NoError(t, err)
	require.NotNil(t, projectCfg)

	require.NotNil(t, projectCfg.Project)
	assert.Contains(t, projectCfg.Project.AllowedEnvVars, "MY_ENV_VAR")
	assert.Contains(t, projectCfg.Project.AllowedEnvVars, "ANOTHER_ENV_VAR")
	assert.Len(t, projectCfg.Project.AllowedEnvVars, 2)
}

func TestParse_MissingProjectName(t *testing.T) {
	_, err := Parse(strings.NewReader(missingProjectName))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestParse_UnknownField_Rejected(t *testing.T) {
	// Strict mode must reject unknown top-level fields to prevent silent misconfigurations.
	_, err := Parse(strings.NewReader(unknownFieldXCAF))
	require.Error(t, err)
}

func TestParse_EmptyReader(t *testing.T) {
	_, err := Parse(strings.NewReader(""))
	require.Error(t, err)
}

func TestParse_MissingVersion_Rejected(t *testing.T) {
	missingVersionXCAF := `kind: project
name: "test-project"
`
	_, err := Parse(strings.NewReader(missingVersionXCAF))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version is required")
}

func TestParse_PathTraversalAgentID_Rejected(t *testing.T) {
	maliciousXCAF := `kind: global
version: "1.0"
agents:
  "../evil":
    description: "Path traversal attempt"
`
	_, err := Parse(strings.NewReader(maliciousXCAF))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent id contains invalid characters")
}

func TestParse_HookEvent_ValidEvent_Accepted(t *testing.T) {
	input := `kind: global
version: "1.0"
hooks:
  PreToolUse:
    - hooks:
        - type: command
          command: "echo ok"
`
	config, err := Parse(strings.NewReader(input))
	require.NoError(t, err)
	assert.Len(t, config.Hooks["default"].Events["PreToolUse"], 1)
}

func TestParse_HookEvent_InvalidEvent_Rejected(t *testing.T) {
	input := `kind: global
version: "1.0"
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
	// Create a fake home structure with global.xcaf
	fakeHome := t.TempDir()
	claudeDir := filepath.Join(fakeHome, ".claude")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(claudeDir, "global.xcaf"),
		[]byte(`kind: global
version: "1.0"
agents:
  shared:
    description: "Shared agent"
`),
		0600,
	))

	// Create a project project.xcaf that extends using an absolute path.
	// (os.UserHomeDir cannot be mocked in tests, so the extends mechanism
	// is verified with an absolute path; the "global" keyword path is a
	// simple string-comparison branch on top of the same merge logic.)
	projectDir := t.TempDir()
	globalPath := filepath.Join(claudeDir, "global.xcaf")
	require.NoError(t, os.WriteFile(
		filepath.Join(projectDir, "project.xcaf"),
		[]byte(fmt.Sprintf(`kind: global
version: "1.0"
extends: "%s"
agents:
  local:
    description: "Local agent"
`, globalPath)),
		0600,
	))

	config, err := ParseFile(filepath.Join(projectDir, "project.xcaf"))
	require.NoError(t, err)

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
			input := `kind: global
version: "1.0"
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
	input := `kind: settings
version: "1.0"
permissions:
  allow: [Bash]
`
	cfg, err := Parse(strings.NewReader(input))
	require.NoError(t, err)
	require.NotNil(t, cfg)
}

func TestValidatePermissions_ValidRule_WithPattern(t *testing.T) {
	input := `kind: settings
version: "1.0"
permissions:
  allow: ["Bash(npm test *)"]
`
	cfg, err := Parse(strings.NewReader(input))
	require.NoError(t, err)
	require.NotNil(t, cfg)
}

func TestValidatePermissions_InvalidRule_UnknownTool(t *testing.T) {
	input := `kind: settings
version: "1.0"
permissions:
  deny: [SomeUnknownTool]
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SomeUnknownTool")
}

func TestValidatePermissions_InvalidRule_MalformedPattern(t *testing.T) {
	input := `kind: settings
version: "1.0"
permissions:
  allow: ["Bash(unclosed"]
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
}

func TestValidatePermissions_Contradiction_AllowAndDeny(t *testing.T) {
	input := `kind: settings
version: "1.0"
permissions:
  allow: [Bash]
  deny: [Bash]
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "allow and deny")
}

func TestValidatePermissions_Contradiction_AllowAndAsk(t *testing.T) {
	input := `kind: settings
version: "1.0"
permissions:
  allow: [Read]
  ask: [Read]
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "allow and ask")
}

func TestValidatePermissions_AgentDisallowedConflict(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "global.xcaf"), []byte(`kind: global
version: "1.0"
agents:
  dev:
    description: "Developer"
    disallowed-tools: [Write]
`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "settings.xcaf"), []byte(`kind: settings
version: "1.0"
permissions:
  allow: [Write]
`), 0600))
	_, err := ParseDirectory(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dev")
	assert.Contains(t, err.Error(), "Write")
}

func TestValidatePermissions_AgentToolsDenyConflict(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "global.xcaf"), []byte(`kind: global
version: "1.0"
agents:
  dev:
    description: "Developer"
    tools: [Bash]
`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "settings.xcaf"), []byte(`kind: settings
version: "1.0"
permissions:
  deny: [Bash]
`), 0600))
	_, err := ParseDirectory(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dev")
	assert.Contains(t, err.Error(), "Bash")
}

func TestParse_MCP_PlatformSpecificFields(t *testing.T) {
	input := `kind: global
version: "1.0"
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
    disabled-tools: ["dangerousTool"]
    auth-provider-type: "custom"
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
	assert.Equal(t, "custom", remote.AuthProviderType) // Field name in struct is PascalCase
}

// ── ValidateFile: file-ref and duplicate-ID tests ──────────────────────────

func writeXCAFFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte(content), 0600))
	return p
}

func TestValidateFileRefs_MissingInstructionsFile_Agent(t *testing.T) {
	t.Skip("Legacy instructions test removed")

	dir := t.TempDir()
	xcaf := writeXCAFFile(t, dir, "project.xcaf", `kind: global
version: "1.0"
agents:
  my-agent:
    description: "An agent"

`)
	diags := ValidateFile(xcaf)
	var found bool
	for _, d := range diags {
		if d.Severity == "error" && strings.Contains(d.Message, "not found") {
			found = true
		}
	}
	assert.True(t, found, "expected an error diagnostic about missing instructions-file, got: %v", diags)
}

func TestValidateFileRefs_PresentInstructionsFile(t *testing.T) {
	t.Skip("Legacy instructions test removed")
	dir := t.TempDir()
	// Create the actual instructions file
	instrFile := filepath.Join(dir, "real.md")
	require.NoError(t, os.WriteFile(instrFile, []byte("# instructions"), 0600))

	xcaf := writeXCAFFile(t, dir, "project.xcaf", `kind: global
version: "1.0"
agents:
  my-agent:
    description: "An agent"

`)
	diags := ValidateFile(xcaf)
	for _, d := range diags {
		if strings.Contains(d.Message, "not found") || strings.Contains(d.Message, "does not exist") {
			t.Errorf("unexpected file-existence diagnostic: %+v", d)
		}
	}
}

func TestValidateFileRefs_DuplicateID(t *testing.T) {
	t.Skip("Legacy instructions test removed")

	dir := t.TempDir()
	xcaf := writeXCAFFile(t, dir, "project.xcaf", `kind: global
version: "1.0"
agents:
  foo:
    description: "An agent"

skills:
  foo:
    description: "A skill"

`)
	diags := ValidateFile(xcaf)
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
	xcaf := writeXCAFFile(t, dir, "project.xcaf", `kind: global
version: "1.0"
agents:
  agent-one:
    description: "An agent"

skills:
  skill-one:
    description: "A skill"

rules:
  rule-one:

`)
	diags := ValidateFile(xcaf)
	for _, d := range diags {
		if strings.Contains(d.Message, "used in both") {
			t.Errorf("unexpected duplicate-ID diagnostic: %+v", d)
		}
	}
}

// ── ValidateFile: plugin validation tests ─────────────────────────────────

func TestValidatePlugins_KnownPlugin(t *testing.T) {
	dir := t.TempDir()
	xcaf := writeXCAFFile(t, dir, "project.xcaf", `kind: settings
version: "1.0"
enabled-plugins:
  commit-commands: true
`)
	diags := ValidateFile(xcaf)
	for _, d := range diags {
		if strings.Contains(d.Message, "unknown plugin") {
			t.Errorf("unexpected unknown-plugin diagnostic for a known plugin: %+v", d)
		}
	}
}

func TestValidatePlugins_UnknownSettingsPlugin(t *testing.T) {
	dir := t.TempDir()
	xcaf := writeXCAFFile(t, dir, "project.xcaf", `kind: settings
version: "1.0"
enabled-plugins:
  my-custom-plugin: true
`)
	diags := ValidateFile(xcaf)
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
	// Local plugin validation now happens at kind:settings level, not kind:project.
	// This test verifies that kind:settings validates unknown plugins.
	xcaf := writeXCAFFile(t, dir, "settings.xcaf", `kind: settings
version: "1.0"
enabled-plugins:
  mystery-plugin: true
`)
	diags := ValidateFile(xcaf)
	var found bool
	for _, d := range diags {
		if d.Severity == "warning" && strings.Contains(d.Message, "unknown plugin") {
			found = true
		}
	}
	assert.True(t, found, "expected a warning about unknown plugin, got: %v", diags)
}

func TestValidatePlugins_BothBlocksUnknown(t *testing.T) {
	dir := t.TempDir()
	// Two separate kind:settings files with different unknown plugins.
	writeXCAFFile(t, dir, "settings1.xcaf", `kind: settings
version: "1.0"
enabled-plugins:
  beta-plugin: true
`)
	writeXCAFFile(t, dir, "settings2.xcaf", `kind: settings
version: "1.0"
enabled-plugins:
  alpha-plugin: true
`)
	// ValidateFile operates on a single file; run it on each file separately.
	diagsSettings1 := ValidateFile(filepath.Join(dir, "settings1.xcaf"))
	diagsSettings2 := ValidateFile(filepath.Join(dir, "settings2.xcaf"))
	diags := append(diagsSettings1, diagsSettings2...)
	count := 0
	for _, d := range diags {
		if d.Severity == "warning" && strings.Contains(d.Message, "unknown plugin") {
			count++
		}
	}
	assert.Equal(t, 2, count, "expected 2 unknown-plugin warnings (one per settings file), got: %v", diags)
}

func TestParseDirectory_SkipsNonConfigFiles(t *testing.T) {
	dir := t.TempDir()

	// Write a valid config file
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcaf"), []byte(`kind: project
version: "1.0"
name: "test-project"
`), 0600))

	// Write a registry file (should be skipped)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "registry.xcaf"), []byte(`kind: registry
projects: []
`), 0600))

	// Write a template file (should be skipped — "template" is not a parseable kind)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "template.xcaf"), []byte(`kind: template
default_target: claude
`), 0600))

	// ParseDirectory should succeed — it should skip registry.xcaf and template.xcaf
	// and only parse project.xcaf. Without the isConfigFile filter, this would
	// panic or error because registry.xcaf and template.xcaf don't conform to
	// the XcaffoldConfig schema.
	config, err := ParseDirectory(dir)
	require.NoError(t, err)
	assert.Equal(t, "test-project", config.Project.Name)
}

func TestParseDirectory_SkipsNonConfigFiles_OnlyNonConfig(t *testing.T) {
	dir := t.TempDir()

	// Write only non-config files
	require.NoError(t, os.WriteFile(filepath.Join(dir, "registry.xcaf"), []byte(`kind: registry
projects: []
`), 0600))

	// ParseDirectory should fail with "no *.xcaf files found" since all files are non-config
	_, err := ParseDirectory(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no *.xcaf files found")
}

func TestIsParseableFile_LegacyNoKind_ReturnsFalse(t *testing.T) {
	dir := t.TempDir()

	// File without kind: field must not be treated as parseable — kind is required
	path := filepath.Join(dir, "legacy.xcaf")
	require.NoError(t, os.WriteFile(path, []byte(`version: "1.0"
project:
  name: "legacy"
`), 0600))

	assert.False(t, isParseableFile(path), "files without kind: must not be treated as parseable")
}

func TestCompile_MultiFile_DuplicateIDErrorTracksOrigin(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "agent1.xcaf")
	err := os.WriteFile(file1, []byte(`kind: global
version: "1.0"
agents:
  dev:
    description: "dev1"
`), 0600) //nolint:goconst
	if err != nil {
		t.Fatal(err)
	}

	file2 := filepath.Join(dir, "agent2.xcaf")
	err = os.WriteFile(file2, []byte(`kind: global
version: "1.0"
agents:
  dev:
    description: "dev2"
`), 0600) //nolint:goconst
	if err != nil {
		t.Fatal(err)
	}

	_, err = ParseDirectory(dir)
	if err == nil {
		t.Fatal("expected error for duplicate agent ID")
	}

	expectedSubstring1 := "duplicate agent ID \"dev\" found in"
	expectedSubstring2 := "agent1.xcaf"
	expectedSubstring3 := "agent2.xcaf"

	errMsg := err.Error()
	// For simplicity, we just check if it contains the duplicate key strings and filenames.
	if !strings.Contains(errMsg, expectedSubstring1) || !strings.Contains(errMsg, expectedSubstring2) || !strings.Contains(errMsg, expectedSubstring3) {
		t.Errorf("error did not contain exact file origin info, got: %v", errMsg)
	}
}

func TestParse_TargetOverride_ProviderPassthrough(t *testing.T) {
	yaml := `
kind: agent
version: "1.0"
name: researcher
model: sonnet
targets:
  gemini:
    provider:
      temperature: 0.7
      timeout_mins: 15
      kind: local
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "agent.xcaf")
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	config, err := ParseFile(path)
	require.NoError(t, err)

	agent := config.Agents["researcher"]
	gemini := agent.Targets["gemini"]
	require.NotNil(t, gemini.Provider)
	require.Equal(t, 0.7, gemini.Provider["temperature"])
	require.Equal(t, 15, gemini.Provider["timeout_mins"])
	require.Equal(t, "local", gemini.Provider["kind"])
}

func TestParse_Skill_TargetsClaudeProviderPassthrough(t *testing.T) {
	t.Skip("Legacy instructions test removed")

	yamlSrc := `
kind: skill
version: "1.0"
name: deep-research
description: Research deeply
targets:
  claude:
    provider:
      context: fork
      agent: Explore
      model: sonnet

`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "skill.xcaf")
	require.NoError(t, os.WriteFile(path, []byte(yamlSrc), 0o600))

	config, err := ParseFile(path)
	require.NoError(t, err)

	skill := config.Skills["deep-research"]
	claude := skill.Targets["claude"]
	require.NotNil(t, claude.Provider)
	require.Equal(t, "fork", claude.Provider["context"])
	require.Equal(t, "Explore", claude.Provider["agent"])
	require.Equal(t, "sonnet", claude.Provider["model"])
}

func TestParseRule_Activation_Valid(t *testing.T) {
	for _, activation := range []string{"always", "path-glob", "model-decided", "manual-mention", "explicit-invoke"} {
		t.Run(activation, func(t *testing.T) {
			paths := ""
			if activation == "path-glob" {
				paths = "\npaths:\n  - src/**"
			}
			src := fmt.Sprintf(`kind: rule
version: "1.0"
name: test-rule
activation: %s%s

`, activation, paths)
			tmp := t.TempDir()
			path := filepath.Join(tmp, "rule.xcaf")
			require.NoError(t, os.WriteFile(path, []byte(src), 0o600))
			_, err := ParseFile(path)
			require.NoError(t, err, "activation %q must be valid", activation)
		})
	}
}

func TestParseRule_Activation_Invalid(t *testing.T) {
	src := `kind: rule
version: "1.0"
name: test-rule
activation: on-demand

`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "rule.xcaf")
	require.NoError(t, os.WriteFile(path, []byte(src), 0o600))
	_, err := ParseFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be one of")
}

func TestParseRule_PathsRequiredForPathGlob(t *testing.T) {
	src := `kind: rule
version: "1.0"
name: test-rule
activation: path-glob

`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "rule.xcaf")
	require.NoError(t, os.WriteFile(path, []byte(src), 0o600))
	_, err := ParseFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "requires at least one path")
}

func TestParseRule_PathsMustBeEmptyForAlways(t *testing.T) {
	src := `kind: rule
version: "1.0"
name: test-rule
activation: always
paths:
  - src/**

`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "rule.xcaf")
	require.NoError(t, os.WriteFile(path, []byte(src), 0o600))
	_, err := ParseFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), `paths must be empty when activation is "always"`)
}

func TestParseRule_LegacyAlwaysApply_Deprecation(t *testing.T) {
	src := `kind: rule
version: "1.0"
name: test-rule
always-apply: true

`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "rule.xcaf")
	require.NoError(t, os.WriteFile(path, []byte(src), 0o600))
	// Must not return a hard error — always-apply without activation is a deprecation warning.
	_, err := ParseFile(path)
	require.NoError(t, err, "always-apply without activation must not return an error")
}

func TestParseRule_OldSnakeCaseInstructionsFile_Error(t *testing.T) {
	t.Skip("Legacy instructions test removed")

	src := `kind: rule
version: "1.0"
name: test-rule
instructions_file: rules/body.md
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "rule.xcaf")
	require.NoError(t, os.WriteFile(path, []byte(src), 0o600))
	_, err := ParseFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), `instructions-file`)
}

func TestParseRule_MutuallyExclusive_Instructions(t *testing.T) {
	t.Skip("Legacy instructions test removed")

	src := `kind: rule
version: "1.0"
name: test-rule


`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "rule.xcaf")
	require.NoError(t, os.WriteFile(path, []byte(src), 0o600))
	_, err := ParseFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "mutually exclusive")
}

func TestParseRule_ExcludeAgents_Invalid(t *testing.T) {
	src := `kind: rule
version: "1.0"
name: test-rule
activation: always
exclude-agents:
  - ci-bot

`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "rule.xcaf")
	require.NoError(t, os.WriteFile(path, []byte(src), 0o600))
	_, err := ParseFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be one of: code-review, cloud-agent")
}

func TestParseRule_ExcludeAgents_Valid(t *testing.T) {
	src := `kind: rule
version: "1.0"
name: test-rule
activation: always
exclude-agents:
  - code-review
  - cloud-agent

`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "rule.xcaf")
	require.NoError(t, os.WriteFile(path, []byte(src), 0o600))
	_, err := ParseFile(path)
	require.NoError(t, err)
}

func TestParseRule_Targets_ProviderPassthrough(t *testing.T) {
	src := `kind: rule
version: "1.0"
name: test-rule
activation: always
targets:
  copilot:
    provider:
      mode: edit

`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "rule.xcaf")
	require.NoError(t, os.WriteFile(path, []byte(src), 0o600))
	config, err := ParseFile(path)
	require.NoError(t, err)

	rule := config.Rules["test-rule"]
	require.NotNil(t, rule.Targets)
	copilot := rule.Targets["copilot"]
	require.NotNil(t, copilot.Provider)
	require.Equal(t, "edit", copilot.Provider["mode"])
}

func TestParseContext_Basic(t *testing.T) {
	src := `---
kind: context
version: "1.0"
name: coding-standards
description: "My standards"
targets: [claude, cursor]
---
# Rules
1. Do this
`
	config, err := Parse(strings.NewReader(src))
	require.NoError(t, err)

	ctx, ok := config.Contexts["coding-standards"]
	require.True(t, ok)
	assert.Equal(t, "coding-standards", ctx.Name)
	assert.Equal(t, "My standards", ctx.Description)
	assert.Equal(t, []string{"claude", "cursor"}, ctx.Targets)
	assert.Equal(t, "# Rules\n1. Do this", ctx.Body)
}

func TestParseContext_NoTargets(t *testing.T) {
	src := `---
kind: context
version: "1.0"
name: universal
---
# Body
`
	config, err := Parse(strings.NewReader(src))
	require.NoError(t, err)

	ctx, ok := config.Contexts["universal"]
	require.True(t, ok)
	assert.Empty(t, ctx.Targets)
	assert.Equal(t, "# Body", ctx.Body)
}

func TestParseContext_MultipleTargets(t *testing.T) {
	src := `---
kind: context
version: "1.0"
name: subset
targets:
  - gemini
  - copilot
---
test
`
	config, err := Parse(strings.NewReader(src))
	require.NoError(t, err)

	ctx, ok := config.Contexts["subset"]
	require.True(t, ok)
	assert.Equal(t, []string{"gemini", "copilot"}, ctx.Targets)
}

func TestParseSkill_BodyAssignment(t *testing.T) {
	src := `---
kind: skill
version: "1.0"
name: my-skill
---
# Skill steps
Do x, y, z.
`
	config, err := Parse(strings.NewReader(src))
	require.NoError(t, err)

	skill, ok := config.Skills["my-skill"]
	require.True(t, ok)
	assert.Equal(t, "# Skill steps\nDo x, y, z.", skill.Body)
}

func TestParseRule_BodyAssignment(t *testing.T) {
	src := `---
kind: rule
version: "1.0"
name: my-rule
activation: always
---
# Rule
Never do bad things.
`
	config, err := Parse(strings.NewReader(src))
	require.NoError(t, err)

	rule, ok := config.Rules["my-rule"]
	require.True(t, ok)
	assert.Equal(t, "# Rule\nNever do bad things.", rule.Body)
}

func TestCompile_MultiFile_DuplicateContextIDErrorTracksOrigin(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "ctx1.xcaf")
	err := os.WriteFile(file1, []byte(`kind: context
version: "1.0"
name: global
`), 0600)
	require.NoError(t, err)

	file2 := filepath.Join(dir, "ctx2.xcaf")
	err = os.WriteFile(file2, []byte(`kind: context
version: "1.0"
name: global
`), 0600)
	require.NoError(t, err)

	_, err = ParseDirectory(dir)
	require.Error(t, err)

	errMsg := err.Error()
	assert.Contains(t, errMsg, "duplicate context ID \"global\" found in")
	assert.Contains(t, errMsg, "ctx1.xcaf")
	assert.Contains(t, errMsg, "ctx2.xcaf")
}

func TestParse_AgentMemory_AcceptsList(t *testing.T) {
	input := `
kind: agent
version: "1.0"
name: developer
memory:
  - user-prefs
  - project-context
`
	cfg, err := Parse(strings.NewReader(input))
	require.NoError(t, err)
	agent := cfg.Agents["developer"]
	require.Equal(t, 2, len(agent.Memory), "expected 2 memory refs, got %d", len(agent.Memory))
	assert.Equal(t, "user-prefs", agent.Memory[0])
	assert.Equal(t, "project-context", agent.Memory[1])
}

func TestParse_AgentMemory_AcceptsScalarBackwardCompat(t *testing.T) {
	input := `
kind: agent
version: "1.0"
name: developer
memory: user-prefs
`
	cfg, err := Parse(strings.NewReader(input))
	require.NoError(t, err)
	agent := cfg.Agents["developer"]
	require.Equal(t, 1, len(agent.Memory), "expected 1 memory ref, got %d", len(agent.Memory))
	assert.Equal(t, "user-prefs", agent.Memory[0])
}

func TestParse_MemoryKind_ParsesDocument(t *testing.T) {
	input := `kind: memory
version: "1.0"
name: user-prefs
description: "User preferences and context"
`
	cfg, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mem, ok := cfg.Memory["user-prefs"]
	if !ok {
		t.Fatal("expected memory entry 'user-prefs' in config")
	}
	if mem.Description != "User preferences and context" {
		t.Fatalf("unexpected description: %s", mem.Description)
	}
}

func TestParse_Skill_ArtifactsField(t *testing.T) {
	input := `---
kind: skill
version: "1.0"
name: my-skill
artifacts: [references, scripts, custom-data]
---
Do things.
`
	config, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	skill := config.Skills["my-skill"]
	if len(skill.Artifacts) != 3 {
		t.Fatalf("expected 3 artifacts, got %d", len(skill.Artifacts))
	}
	if skill.Artifacts[0] != "references" {
		t.Errorf("artifacts[0] = %q, want %q", skill.Artifacts[0], "references")
	}
	if skill.Artifacts[1] != "scripts" {
		t.Errorf("artifacts[1] = %q, want %q", skill.Artifacts[1], "scripts")
	}
	if skill.Artifacts[2] != "custom-data" {
		t.Errorf("artifacts[2] = %q, want %q", skill.Artifacts[2], "custom-data")
	}
}

// TestParse_Skill_LegacyFieldsMigrateToArtifacts has been removed because
// the legacy fields (references, scripts, assets, examples) have been removed
// from the SkillConfig struct. All subdirectories are now managed via the
// artifacts field alone.
//
// func TestParse_Skill_LegacyFieldsMigrateToArtifacts(t *testing.T) {
//   (test removed)
// }

func TestParse_Agent_RejectsWhenField(t *testing.T) {
	input := `---
kind: agent
version: "1.0"
name: test
description: "Test agent"
when: "some condition"
---
Test.
`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Error("expected error for removed 'when' field")
	}
}

func TestParse_Agent_RejectsModeField(t *testing.T) {
	input := `---
kind: agent
version: "1.0"
name: test
description: "Test agent"
mode: "interactive"
---
Test.
`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Error("expected error for removed 'mode' field")
	}
}

func TestParse_Hooks_ArtifactsField(t *testing.T) {
	input := `kind: hooks
version: "1.0"
name: enforce
artifacts: [scripts]
events:
  PreToolUse:
    - hooks:
        - type: command
          command: bash scripts/enforce-standards.sh
`
	config, err := Parse(strings.NewReader(input))
	require.NoError(t, err)
	hook := config.Hooks["enforce"]
	require.NotNil(t, hook)
	require.Len(t, hook.Artifacts, 1)
	assert.Equal(t, "scripts", hook.Artifacts[0])
}

func TestParseOverrideFile_Hooks_DecodesAndStores(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, "xcaf", "hooks", "pre-commit")
	require.NoError(t, os.MkdirAll(hooksDir, 0o755))

	content := "---\nname: pre-commit\nartifacts:\n  - lint.sh\n  - format.sh\n---\n"
	require.NoError(t, os.WriteFile(filepath.Join(hooksDir, "hooks.claude.xcaf"), []byte(content), 0o644))

	config := &ast.XcaffoldConfig{}
	entry := overrideFileEntry{
		Path:     filepath.Join(hooksDir, "hooks.claude.xcaf"),
		Kind:     "hooks",
		Provider: "claude",
	}

	err := parseOverrideFile(entry, config, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, config.Overrides)

	cfg, ok := config.Overrides.GetHooks("pre-commit", "claude")
	require.True(t, ok)
	assert.Equal(t, "pre-commit", cfg.Name)
	assert.Equal(t, []string{"lint.sh", "format.sh"}, cfg.Artifacts)
}

func TestParseOverrideFile_Settings_DecodesAndStores(t *testing.T) {
	dir := t.TempDir()
	settingsDir := filepath.Join(dir, "xcaf", "settings", "default")
	require.NoError(t, os.MkdirAll(settingsDir, 0o755))

	content := "---\nname: default\nauto-memory-enabled: true\n---\n"
	require.NoError(t, os.WriteFile(filepath.Join(settingsDir, "settings.gemini.xcaf"), []byte(content), 0o644))

	config := &ast.XcaffoldConfig{}
	entry := overrideFileEntry{
		Path:     filepath.Join(settingsDir, "settings.gemini.xcaf"),
		Kind:     "settings",
		Provider: "gemini",
	}

	err := parseOverrideFile(entry, config, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, config.Overrides)

	cfg, ok := config.Overrides.GetSettings("default", "gemini")
	require.True(t, ok)
	assert.Equal(t, "default", cfg.Name)
	require.NotNil(t, cfg.AutoMemoryEnabled)
	assert.True(t, *cfg.AutoMemoryEnabled)
}

func TestParseOverrideFile_Policy_DecodesAndStores(t *testing.T) {
	dir := t.TempDir()
	policyDir := filepath.Join(dir, "xcaf", "policies", "require-desc")
	require.NoError(t, os.MkdirAll(policyDir, 0o755))

	content := "---\nname: require-desc\nseverity: error\ntarget: agent\n---\n"
	require.NoError(t, os.WriteFile(filepath.Join(policyDir, "policy.cursor.xcaf"), []byte(content), 0o644))

	config := &ast.XcaffoldConfig{}
	entry := overrideFileEntry{
		Path:     filepath.Join(policyDir, "policy.cursor.xcaf"),
		Kind:     "policy",
		Provider: "cursor",
	}

	err := parseOverrideFile(entry, config, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, config.Overrides)

	cfg, ok := config.Overrides.GetPolicy("require-desc", "cursor")
	require.True(t, ok)
	assert.Equal(t, "require-desc", cfg.Name)
	assert.Equal(t, "error", cfg.Severity)
	assert.Equal(t, "agent", cfg.Target)
}

func TestParseOverrideFile_Template_DecodesAndStoresWithBody(t *testing.T) {
	dir := t.TempDir()
	tmplDir := filepath.Join(dir, "xcaf", "templates", "scaffold")
	require.NoError(t, os.MkdirAll(tmplDir, 0o755))

	content := "---\nname: scaffold\ndescription: Project scaffold template\ndefault-target: claude\n---\nThis is the template body content.\nIt supports multiple lines."
	require.NoError(t, os.WriteFile(filepath.Join(tmplDir, "template.copilot.xcaf"), []byte(content), 0o644))

	config := &ast.XcaffoldConfig{}
	entry := overrideFileEntry{
		Path:     filepath.Join(tmplDir, "template.copilot.xcaf"),
		Kind:     "template",
		Provider: "copilot",
	}

	err := parseOverrideFile(entry, config, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, config.Overrides)

	cfg, ok := config.Overrides.GetTemplate("scaffold", "copilot")
	require.True(t, ok)
	assert.Equal(t, "scaffold", cfg.Name)
	assert.Equal(t, "Project scaffold template", cfg.Description)
	assert.Equal(t, "claude", cfg.DefaultTarget)
	assert.Contains(t, cfg.Body, "This is the template body content.")
	assert.Contains(t, cfg.Body, "It supports multiple lines.")
}

func TestParseOverrideFile_Memory_NoOp(t *testing.T) {
	dir := t.TempDir()
	memDir := filepath.Join(dir, "xcaf", "agents", "dev", "memory")
	require.NoError(t, os.MkdirAll(memDir, 0o755))

	content := "---\nname: context\n---\nSome memory content that should be ignored for overrides."
	require.NoError(t, os.WriteFile(filepath.Join(memDir, "memory.claude.xcaf"), []byte(content), 0o644))

	config := &ast.XcaffoldConfig{}
	entry := overrideFileEntry{
		Path:     filepath.Join(memDir, "memory.claude.xcaf"),
		Kind:     "memory",
		Provider: "claude",
	}

	err := parseOverrideFile(entry, config, nil, nil)
	require.NoError(t, err)
	assert.Nil(t, config.Overrides)
}

func TestParseOverrideFile_UnsupportedKind_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	badDir := filepath.Join(dir, "xcaf", "bad", "thing")
	require.NoError(t, os.MkdirAll(badDir, 0o755))

	content := "---\nname: thing\n---\n"
	fpath := filepath.Join(badDir, "bogus.claude.xcaf")
	require.NoError(t, os.WriteFile(fpath, []byte(content), 0o644))

	config := &ast.XcaffoldConfig{}
	entry := overrideFileEntry{
		Path:     fpath,
		Kind:     "bogus",
		Provider: "claude",
	}

	err := parseOverrideFile(entry, config, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported kind")
	assert.Contains(t, err.Error(), "bogus")
}

func TestValidateOverrideBasesExist_Hooks_WithBase_Passes(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Hooks: map[string]ast.NamedHookConfig{
			"pre-commit": {Name: "pre-commit"},
		},
		Overrides: &ast.ResourceOverrides{},
	}
	config.Overrides.AddHooks("pre-commit", "claude", ast.NamedHookConfig{Name: "pre-commit"})

	err := validateOverrideBasesExist(config)
	require.NoError(t, err)
}

func TestValidateOverrideBasesExist_Hooks_WithoutBase_Fails(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Hooks:     map[string]ast.NamedHookConfig{},
		Overrides: &ast.ResourceOverrides{},
	}
	config.Overrides.AddHooks("missing-hook", "claude", ast.NamedHookConfig{Name: "missing-hook"})

	err := validateOverrideBasesExist(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hooks")
	assert.Contains(t, err.Error(), "missing-hook")
	assert.Contains(t, err.Error(), "no base resource")
}

func TestValidateOverrideBasesExist_Settings_WithBase_Passes(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Settings:  map[string]ast.SettingsConfig{"default": {Name: "default"}},
		Overrides: &ast.ResourceOverrides{},
	}
	config.Overrides.AddSettings("default", "claude", ast.SettingsConfig{Name: "default"})

	err := validateOverrideBasesExist(config)
	require.NoError(t, err)
}

func TestValidateOverrideBasesExist_Settings_WithoutBase_Fails(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Settings:  map[string]ast.SettingsConfig{},
		Overrides: &ast.ResourceOverrides{},
	}
	config.Overrides.AddSettings("missing-settings", "claude", ast.SettingsConfig{Name: "missing-settings"})

	err := validateOverrideBasesExist(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "settings")
	assert.Contains(t, err.Error(), "missing-settings")
	assert.Contains(t, err.Error(), "no base resource")
}

func TestValidateOverrideBasesExist_Policy_WithBase_Passes(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Overrides: &ast.ResourceOverrides{},
	}
	config.Policies = map[string]ast.PolicyConfig{"require-desc": {Name: "require-desc"}}
	config.Overrides.AddPolicy("require-desc", "claude", ast.PolicyConfig{Name: "require-desc"})

	err := validateOverrideBasesExist(config)
	require.NoError(t, err)
}

func TestValidateOverrideBasesExist_Policy_WithoutBase_Fails(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Overrides: &ast.ResourceOverrides{},
	}
	config.Policies = map[string]ast.PolicyConfig{}
	config.Overrides.AddPolicy("missing-policy", "claude", ast.PolicyConfig{Name: "missing-policy"})

	err := validateOverrideBasesExist(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "policy")
	assert.Contains(t, err.Error(), "missing-policy")
	assert.Contains(t, err.Error(), "no base resource")
}

func TestValidateOverrideBasesExist_Template_WithBase_Passes(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Overrides: &ast.ResourceOverrides{},
	}
	config.Templates = map[string]ast.TemplateConfig{"scaffold": {Name: "scaffold"}}
	config.Overrides.AddTemplate("scaffold", "claude", ast.TemplateConfig{Name: "scaffold"})

	err := validateOverrideBasesExist(config)
	require.NoError(t, err)
}

func TestValidateOverrideBasesExist_Template_WithoutBase_Fails(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Overrides: &ast.ResourceOverrides{},
	}
	config.Templates = map[string]ast.TemplateConfig{}
	config.Overrides.AddTemplate("missing-tpl", "claude", ast.TemplateConfig{Name: "missing-tpl"})

	err := validateOverrideBasesExist(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "template")
	assert.Contains(t, err.Error(), "missing-tpl")
	assert.Contains(t, err.Error(), "no base resource")
}

func TestValidateCrossReferencesAsList_UnresolvedSkill_ReturnsWarning(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"worker": {Name: "worker", Skills: ast.ClearableList{Values: []string{"nonexistent-skill"}}},
			},
			Skills: map[string]ast.SkillConfig{},
		},
	}

	issues := validateCrossReferencesAsList(cfg)
	require.Len(t, issues, 1)
	assert.Contains(t, issues[0].Message, "nonexistent-skill")
	assert.Contains(t, issues[0].Message, "not found in project scope")
	assert.Equal(t, "skill", issues[0].ResourceType)
}

func TestValidateCrossReferencesAsList_UnresolvedRule_ReturnsWarning(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"reviewer": {Name: "reviewer", Rules: ast.ClearableList{Values: []string{"missing-rule"}}},
			},
			Rules: map[string]ast.RuleConfig{},
		},
	}

	issues := validateCrossReferencesAsList(cfg)
	require.Len(t, issues, 1)
	assert.Contains(t, issues[0].Message, "missing-rule")
	assert.Contains(t, issues[0].Message, "not found in project scope")
	assert.Equal(t, "rule", issues[0].ResourceType)
}

func TestValidateCrossReferencesAsList_UnresolvedMCP_ReturnsWarning(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"helper": {Name: "helper", MCP: ast.ClearableList{Values: []string{"missing-mcp"}}},
			},
			MCP: map[string]ast.MCPConfig{},
		},
	}

	issues := validateCrossReferencesAsList(cfg)
	require.Len(t, issues, 1)
	assert.Contains(t, issues[0].Message, "missing-mcp")
	assert.Contains(t, issues[0].Message, "not found in project scope")
	assert.Equal(t, "mcp", issues[0].ResourceType)
}

func TestValidateCrossReferencesAsList_AllResolved_NoIssues(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"worker": {
					Name:   "worker",
					Skills: ast.ClearableList{Values: []string{"tdd"}},
					Rules:  ast.ClearableList{Values: []string{"style"}},
					MCP:    ast.ClearableList{Values: []string{"postgres"}},
				},
			},
			Skills: map[string]ast.SkillConfig{"tdd": {Name: "tdd"}},
			Rules:  map[string]ast.RuleConfig{"style": {Name: "style"}},
			MCP:    map[string]ast.MCPConfig{"postgres": {}},
		},
	}

	issues := validateCrossReferencesAsList(cfg)
	require.Empty(t, issues)
}

func TestParseDirectory_SkipGlobal_DoesNotLoadGlobalBase(t *testing.T) {
	dir := t.TempDir()
	xcafDir := filepath.Join(dir, "xcaf")
	os.MkdirAll(xcafDir, 0755)

	projectXcaf := filepath.Join(xcafDir, "project.xcaf")
	os.WriteFile(projectXcaf, []byte("---\nkind: project\nversion: \"1.0\"\nname: test-project\n---\n"), 0644)

	agentDir := filepath.Join(xcafDir, "agents", "worker")
	os.MkdirAll(agentDir, 0755)
	os.WriteFile(filepath.Join(agentDir, "agent.xcaf"), []byte("---\nkind: agent\nversion: \"1.0\"\nname: worker\ndescription: \"Test agent\"\ntools: [Read]\n---\nDo work.\n"), 0644)

	cfg, err := ParseDirectory(dir, WithSkipGlobal())
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Contains(t, cfg.Agents, "worker")
}

func TestMain(m *testing.M) {
	tmpHome, err := os.MkdirTemp("", "parser-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpHome)
	os.Setenv("HOME", tmpHome)
	os.Exit(m.Run())
}
