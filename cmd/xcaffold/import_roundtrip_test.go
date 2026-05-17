package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/stretchr/testify/require"
)

// TestImport_RoundTrip_SkillAllowedTools verifies that a skill with allowed-tools
// can be imported from a provider-specific directory and maintains fidelity.
func TestImport_RoundTrip_SkillAllowedTools(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()

	orig, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(orig) }()
	require.NoError(t, os.Chdir(tmp))

	// Create .claude/skills/ with a skill that has allowed-tools
	claudeSkillsDir := filepath.Join(tmp, ".claude", "skills", "test-skill")
	require.NoError(t, os.MkdirAll(claudeSkillsDir, 0755))

	// Write a SKILL.md with a YAML frontmatter that includes allowed-tools
	skillContent := `---
name: test-skill
description: Test skill with allowed-tools
allowed-tools:
  - Bash
  - Read
  - Write
when-to-use: This is a test skill.
---

# Test Skill

This is a test skill for verifying allowed-tools round-trip.
`
	require.NoError(t, os.WriteFile(
		filepath.Join(claudeSkillsDir, "SKILL.md"),
		[]byte(skillContent),
		0600,
	))

	// Run importScope to import from .claude/
	err = importScope(".claude", "project.xcaf", "project", "claude")
	require.NoError(t, err)

	// Verify project.xcaf was created
	require.FileExists(t, filepath.Join(tmp, "project.xcaf"))

	// Parse the imported config from the directory (which includes split xcaf/ files)
	importedConfig, parseErr := parser.ParseDirectory(".", parser.WithSkipGlobal())
	require.NoError(t, parseErr)
	require.NotNil(t, importedConfig)

	// Verify skill was imported with allowed-tools preserved
	require.Len(t, importedConfig.Skills, 1, "should have 1 skill")
	skill, ok := importedConfig.Skills["test-skill"]
	require.True(t, ok, "test-skill should be in config")
	require.Equal(t, []string{"Bash", "Read", "Write"}, skill.AllowedTools.Values)
}

// TestImport_RoundTrip_AgentMCPServers verifies that an agent with mcpServers
// can be imported from a provider-specific directory and maintains fidelity.
func TestImport_RoundTrip_AgentMCPServers(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()

	orig, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(orig) }()
	require.NoError(t, os.Chdir(tmp))

	// Create .claude/agents/ with an agent that has mcpServers
	claudeAgentsDir := filepath.Join(tmp, ".claude", "agents")
	require.NoError(t, os.MkdirAll(claudeAgentsDir, 0755))

	// Write an agent.md with mcpServers in YAML frontmatter
	agentContent := `---
name: test-agent
description: Test agent with mcpServers
model: claude-opus-4-5
mcpServers:
  my-server:
    command: node
    args:
      - server.js
---

# Test Agent

This is a test agent for verifying mcpServers round-trip.
`
	require.NoError(t, os.WriteFile(
		filepath.Join(claudeAgentsDir, "test-agent.md"),
		[]byte(agentContent),
		0600,
	))

	// Run importScope to import from .claude/
	err = importScope(".claude", "project.xcaf", "project", "claude")
	require.NoError(t, err)

	// Verify project.xcaf was created
	require.FileExists(t, filepath.Join(tmp, "project.xcaf"))

	// Parse the imported config from the directory (which includes split xcaf/ files)
	importedConfig, parseErr := parser.ParseDirectory(".", parser.WithSkipGlobal())
	require.NoError(t, parseErr)
	require.NotNil(t, importedConfig)

	// Verify agent was imported with mcpServers preserved
	require.Len(t, importedConfig.Agents, 1, "should have 1 agent")
	agent, ok := importedConfig.Agents["test-agent"]
	require.True(t, ok, "test-agent should be in config")
	require.Len(t, agent.MCPServers, 1, "should have 1 mcpServer")
	require.Contains(t, agent.MCPServers, "my-server", "my-server should be in mcpServers")
}

// TestImport_RoundTrip_CompleteProject verifies that a project with both
// skills and agents can be imported without loss of fidelity.
func TestImport_RoundTrip_CompleteProject(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()

	orig, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(orig) }()
	require.NoError(t, os.Chdir(tmp))

	// Create .claude/skills/ with a skill
	claudeSkillsDir := filepath.Join(tmp, ".claude", "skills", "test-skill")
	require.NoError(t, os.MkdirAll(claudeSkillsDir, 0755))
	skillContent := `---
name: test-skill
description: Test skill
allowed-tools:
  - Bash
  - Read
when-to-use: Use for testing.
---

# Test Skill
`
	require.NoError(t, os.WriteFile(
		filepath.Join(claudeSkillsDir, "SKILL.md"),
		[]byte(skillContent),
		0600,
	))

	// Create .claude/agents/ with an agent
	claudeAgentsDir := filepath.Join(tmp, ".claude", "agents")
	require.NoError(t, os.MkdirAll(claudeAgentsDir, 0755))
	agentContent := `---
name: test-agent
description: Test agent
model: claude-sonnet-4-5
mcpServers:
  my-server:
    command: node
    args:
      - server.js
---

# Test Agent
`
	require.NoError(t, os.WriteFile(
		filepath.Join(claudeAgentsDir, "test-agent.md"),
		[]byte(agentContent),
		0600,
	))

	// Create .claude/settings.json with global config
	require.NoError(t, os.WriteFile(
		filepath.Join(tmp, ".claude", "settings.json"),
		[]byte("{}"),
		0600,
	))

	// Run importScope to import from .claude/
	err = importScope(".claude", "project.xcaf", "project", "claude")
	require.NoError(t, err)

	// Verify project.xcaf was created
	require.FileExists(t, filepath.Join(tmp, "project.xcaf"))

	// Parse the imported config from the directory (which includes split xcaf/ files)
	importedConfig, parseErr := parser.ParseDirectory(".", parser.WithSkipGlobal())
	require.NoError(t, parseErr)
	require.NotNil(t, importedConfig)

	// Verify both skill and agent were imported
	require.Len(t, importedConfig.Skills, 1, "should have 1 skill")
	require.Len(t, importedConfig.Agents, 1, "should have 1 agent")

	// Verify skill preserved allowed-tools
	skill, ok := importedConfig.Skills["test-skill"]
	require.True(t, ok)
	require.Equal(t, []string{"Bash", "Read"}, skill.AllowedTools.Values)

	// Verify agent preserved mcpServers
	agent, ok := importedConfig.Agents["test-agent"]
	require.True(t, ok)
	require.Len(t, agent.MCPServers, 1)
	require.Contains(t, agent.MCPServers, "my-server")
}

// TestImport_RoundTrip_MultipleTools verifies various allowed-tools combinations
// are preserved during import.
func TestImport_RoundTrip_MultipleTools(t *testing.T) {
	testCases := []struct {
		name  string
		tools []string
	}{
		{"single-tool", []string{"Bash"}},
		{"multiple-tools", []string{"Bash", "Read", "Write", "Edit"}},
		{"many-tools", []string{"Bash", "Read", "Write", "Edit", "Grep", "Glob"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("XCAFFOLD_HOME", t.TempDir())
			tmp := t.TempDir()

			orig, err := os.Getwd()
			require.NoError(t, err)
			defer func() { _ = os.Chdir(orig) }()
			require.NoError(t, os.Chdir(tmp))

			// Create skill with specified tools
			claudeSkillsDir := filepath.Join(tmp, ".claude", "skills", "test-skill")
			require.NoError(t, os.MkdirAll(claudeSkillsDir, 0755))

			toolsYAML := "  - " + tc.tools[0]
			for _, tool := range tc.tools[1:] {
				toolsYAML += "\n  - " + tool
			}

			skillContent := `---
name: test-skill
description: Test skill
allowed-tools:
` + toolsYAML + `
---

# Test Skill
`
			require.NoError(t, os.WriteFile(
				filepath.Join(claudeSkillsDir, "SKILL.md"),
				[]byte(skillContent),
				0600,
			))

			// Import
			err = importScope(".claude", "project.xcaf", "project", "claude")
			require.NoError(t, err)

			// Verify
			importedConfig, parseErr := parser.ParseDirectory(".", parser.WithSkipGlobal())
			require.NoError(t, parseErr)
			require.Len(t, importedConfig.Skills, 1)
			skill, ok := importedConfig.Skills["test-skill"]
			require.True(t, ok)
			require.Equal(t, tc.tools, skill.AllowedTools.Values, tc.name)
		})
	}
}
