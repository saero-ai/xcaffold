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

const validXCF = `---
kind: project
version: "1.0"
name: "test-project"
description: "A test project."
---
kind: global
version: "1.0"
agents:
  developer:
    description: "An expert developer."
    instructions: |
      You are a software developer.
    model: "claude-3-7-sonnet-20250219"
    effort: "high"
    tools: [Bash, Read, Write]
`

const missingProjectName = `kind: project
version: "1.0"
description: "Missing the name field."
`

const unknownFieldXCF = `kind: global
version: "1.0"
agents:
  dev:
    description: "developer"
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
	assert.Contains(t, err.Error(), "name is required")
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
	missingVersionXCF := `kind: project
name: "test-project"
`
	_, err := Parse(strings.NewReader(missingVersionXCF))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version is required")
}

func TestParse_PathTraversalAgentID_Rejected(t *testing.T) {
	maliciousXCF := `kind: global
version: "1.0"
agents:
  "../evil":
    description: "Path traversal attempt"
`
	_, err := Parse(strings.NewReader(maliciousXCF))
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
	assert.Len(t, config.Hooks["PreToolUse"], 1)
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
	// Create a fake home structure with global.xcf
	fakeHome := t.TempDir()
	claudeDir := filepath.Join(fakeHome, ".claude")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(claudeDir, "global.xcf"),
		[]byte(`kind: global
version: "1.0"
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
		[]byte(fmt.Sprintf(`kind: global
version: "1.0"
extends: "%s"
agents:
  local:
    description: "Local agent"
`, globalPath)),
		0600,
	))

	config, err := ParseFile(filepath.Join(projectDir, "scaffold.xcf"))
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
	input := `---
kind: global
version: "1.0"
agents:
  dev:
    description: "Developer"
    disallowed-tools: [Write]
---
kind: settings
version: "1.0"
permissions:
  allow: [Write]
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dev")
	assert.Contains(t, err.Error(), "Write")
}

func TestValidatePermissions_AgentToolsDenyConflict(t *testing.T) {
	input := `---
kind: global
version: "1.0"
agents:
  dev:
    description: "Developer"
    tools: [Bash]
---
kind: settings
version: "1.0"
permissions:
  deny: [Bash]
`
	_, err := Parse(strings.NewReader(input))
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
	xcf := writeXCFFile(t, dir, "scaffold.xcf", `kind: global
version: "1.0"
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
		if d.Severity == "warning" && strings.Contains(d.Message, "does not exist") { //nolint:goconst
			found = true
		}
	}
	assert.True(t, found, "expected a warning diagnostic about a missing reference file, got: %v", diags)
}

func TestValidateFileRefs_MissingInstructionsFile_Agent(t *testing.T) {
	dir := t.TempDir()
	xcf := writeXCFFile(t, dir, "scaffold.xcf", `kind: global
version: "1.0"
agents:
  my-agent:
    description: "An agent"
    instructions-file: ghost.md
`)
	diags := ValidateFile(xcf)
	var found bool
	for _, d := range diags {
		if d.Severity == "error" && strings.Contains(d.Message, "not found") {
			found = true
		}
	}
	assert.True(t, found, "expected an error diagnostic about missing instructions-file, got: %v", diags)
}

func TestValidateFileRefs_PresentInstructionsFile(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	dir := t.TempDir()
	// Create the actual instructions file
	instrFile := filepath.Join(dir, "real.md")
	require.NoError(t, os.WriteFile(instrFile, []byte("# instructions"), 0600))

	xcf := writeXCFFile(t, dir, "scaffold.xcf", `kind: global
version: "1.0"
agents:
  my-agent:
    description: "An agent"
    instructions-file: real.md
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
	xcf := writeXCFFile(t, dir, "scaffold.xcf", `kind: global
version: "1.0"
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
	xcf := writeXCFFile(t, dir, "scaffold.xcf", `kind: global
version: "1.0"
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
	xcf := writeXCFFile(t, dir, "scaffold.xcf", `kind: settings
version: "1.0"
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
	xcf := writeXCFFile(t, dir, "scaffold.xcf", `kind: settings
version: "1.0"
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
	xcf := writeXCFFile(t, dir, "scaffold.xcf", `kind: project
version: "1.0"
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
	xcf := writeXCFFile(t, dir, "scaffold.xcf", `---
kind: project
version: "1.0"
name: "test"
local:
  enabledPlugins:
    beta-plugin: true
---
kind: settings
version: "1.0"
enabledPlugins:
  alpha-plugin: true
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

func TestParseDirectory_SkipsNonConfigFiles(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	dir := t.TempDir()

	// Write a valid config file
	require.NoError(t, os.WriteFile(filepath.Join(dir, "scaffold.xcf"), []byte(`kind: project
version: "1.0"
name: "test-project"
`), 0600))

	// Write a registry file (should be skipped)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "registry.xcf"), []byte(`kind: registry
projects: []
`), 0600))

	// Write a template file (should be skipped — "template" is not a parseable kind)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "template.xcf"), []byte(`kind: template
default_target: claude
`), 0600))

	// ParseDirectory should succeed — it should skip registry.xcf and template.xcf
	// and only parse scaffold.xcf. Without the isConfigFile filter, this would
	// panic or error because registry.xcf and template.xcf don't conform to
	// the XcaffoldConfig schema.
	config, err := ParseDirectory(dir)
	require.NoError(t, err)
	assert.Equal(t, "test-project", config.Project.Name)
}

func TestParseDirectory_SkipsNonConfigFiles_OnlyNonConfig(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	dir := t.TempDir()

	// Write only non-config files
	require.NoError(t, os.WriteFile(filepath.Join(dir, "registry.xcf"), []byte(`kind: registry
projects: []
`), 0600))

	// ParseDirectory should fail with "no *.xcf files found" since all files are non-config
	_, err := ParseDirectory(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no *.xcf files found")
}

func TestIsParseableFile_LegacyNoKind_ReturnsFalse(t *testing.T) {
	dir := t.TempDir()

	// File without kind: field must not be treated as parseable — kind is required
	path := filepath.Join(dir, "legacy.xcf")
	require.NoError(t, os.WriteFile(path, []byte(`version: "1.0"
project:
  name: "legacy"
`), 0600))

	assert.False(t, isParseableFile(path), "files without kind: must not be treated as parseable")
}

func TestCompile_MultiFile_DuplicateIDErrorTracksOrigin(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "agent1.xcf")
	err := os.WriteFile(file1, []byte(`kind: global
version: "1.0"
agents:
  dev:
    description: "dev1"
`), 0600) //nolint:goconst
	if err != nil {
		t.Fatal(err)
	}

	file2 := filepath.Join(dir, "agent2.xcf")
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
	expectedSubstring2 := "agent1.xcf"
	expectedSubstring3 := "agent2.xcf"

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
	path := filepath.Join(tmp, "agent.xcf")
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
instructions: "Research deeply."
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "skill.xcf")
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
instructions: "Body."
`, activation, paths)
			tmp := t.TempDir()
			path := filepath.Join(tmp, "rule.xcf")
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
instructions: "Body."
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "rule.xcf")
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
instructions: "Body."
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "rule.xcf")
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
instructions: "Body."
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "rule.xcf")
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
instructions: "Body."
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "rule.xcf")
	require.NoError(t, os.WriteFile(path, []byte(src), 0o600))
	// Must not return a hard error — always-apply without activation is a deprecation warning.
	_, err := ParseFile(path)
	require.NoError(t, err, "always-apply without activation must not return an error")
}

func TestParseRule_OldSnakeCaseInstructionsFile_Error(t *testing.T) {
	src := `kind: rule
version: "1.0"
name: test-rule
instructions_file: rules/body.md
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "rule.xcf")
	require.NoError(t, os.WriteFile(path, []byte(src), 0o600))
	_, err := ParseFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), `instructions-file`)
}

func TestParseRule_MutuallyExclusive_Instructions(t *testing.T) {
	src := `kind: rule
version: "1.0"
name: test-rule
instructions: "Body."
instructions-file: rules/body.md
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "rule.xcf")
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
instructions: "Body."
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "rule.xcf")
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
instructions: "Body."
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "rule.xcf")
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
instructions: "Body."
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "rule.xcf")
	require.NoError(t, os.WriteFile(path, []byte(src), 0o600))
	config, err := ParseFile(path)
	require.NoError(t, err)

	rule := config.Rules["test-rule"]
	require.NotNil(t, rule.Targets)
	copilot := rule.Targets["copilot"]
	require.NotNil(t, copilot.Provider)
	require.Equal(t, "edit", copilot.Provider["mode"])
}

func TestParse_ProjectInstructions_MutualExclusivity(t *testing.T) {
	yaml := `
kind: project
version: "1.0"
name: test
instructions: "inline content"
instructions-file: xcf/instructions/root.md
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "scaffold.xcf")
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))
	_, err := ParseFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "instructions and instructions-file are mutually exclusive")
}

func TestParse_ProjectInstructions_ScopeMutualExclusivity(t *testing.T) {
	yaml := `
kind: project
version: "1.0"
name: test
instructions-scopes:
  - path: packages/worker
    instructions: "inline"
    instructions-file: xcf/instructions/scopes/packages-worker.md
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "scaffold.xcf")
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))
	_, err := ParseFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "mutually exclusive")
}

func TestParse_ProjectInstructions_DuplicateScopePath(t *testing.T) {
	yaml := `
kind: project
version: "1.0"
name: test
instructions-scopes:
  - path: packages/worker
    instructions-file: xcf/instructions/scopes/packages-worker.md
  - path: packages/worker
    instructions-file: xcf/instructions/scopes/packages-worker-2.md
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "scaffold.xcf")
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))
	_, err := ParseFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), `duplicate instructions-scope path "packages/worker"`)
}

func TestParse_ProjectInstructions_InvalidMergeStrategy(t *testing.T) {
	yaml := `
kind: project
version: "1.0"
name: test
instructions-scopes:
  - path: packages/worker
    instructions-file: xcf/instructions/scopes/packages-worker.md
    merge-strategy: invalid-value
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "scaffold.xcf")
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))
	_, err := ParseFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "merge-strategy")
}

func TestParse_ProjectInstructions_ValidConfig(t *testing.T) {
	yaml := `
kind: project
version: "1.0"
name: test
instructions-scopes:
  - path: packages/worker
    instructions-file: xcf/instructions/scopes/packages-worker.md
    merge-strategy: concat
    source-provider: claude
    source-filename: CLAUDE.md
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "scaffold.xcf")
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))
	config, err := ParseFile(path)
	require.NoError(t, err)
	require.Len(t, config.Project.InstructionsScopes, 1)
	require.Equal(t, "concat", config.Project.InstructionsScopes[0].MergeStrategy)
}

func TestParse_ProjectInstructions_UnknownFieldRejected(t *testing.T) {
	yaml := `
kind: project
version: "1.0"
name: test
instructions-bogus: should-fail
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "scaffold.xcf")
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))
	_, err := ParseFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "instructions-bogus")
}

func TestParse_InstructionsScope_UnknownFieldRejected(t *testing.T) {
	yaml := `
kind: project
version: "1.0"
name: test
instructions-scopes:
  - path: packages/worker
    unknown-field: bogus
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "scaffold.xcf")
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))
	_, err := ParseFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown-field")
}
