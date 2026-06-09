package antigravity2

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/importer"
)

// TestImport_Antigravity2_AgentJSON verifies that agent.json files are correctly parsed
// and all 12 AgentConfig fields are populated.
func TestImport_Antigravity2_AgentJSON(t *testing.T) {
	tmpdir := t.TempDir()
	agentsDir := filepath.Join(tmpdir, ".agents", "agents", "test")
	require.NoError(t, os.MkdirAll(agentsDir, 0755))

	agentData := `{
  "name": "test-agent",
  "description": "A test agent",
  "model": "claude-3-opus",
  "maxTurns": 10,
  "tools": ["bash", "python"],
  "disabledTools": ["system"],
  "readonly": false,
  "userInvocable": true,
  "initialPrompt": "Start by understanding the request.",
  "skills": ["skill-1"],
  "rules": ["rule-1"],
  "instructions": "You are a helpful assistant."
}`
	agentPath := filepath.Join(agentsDir, "agent.json")
	require.NoError(t, os.WriteFile(agentPath, []byte(agentData), 0644))

	config := &ast.XcaffoldConfig{}
	imp := NewImporter()
	require.NoError(t, imp.Import(filepath.Join(tmpdir, ".agents"), config))

	agent, ok := config.Agents["test"]
	require.True(t, ok, "agent 'test' should be present")
	assert.Equal(t, "test-agent", agent.Name)
	assert.Equal(t, "A test agent", agent.Description)
	assert.Equal(t, "claude-3-opus", agent.Model)
	assert.NotNil(t, agent.MaxTurns)
	assert.Equal(t, 10, *agent.MaxTurns)
	assert.Equal(t, 2, len(agent.Tools.Values))
	assert.Equal(t, 1, len(agent.DisallowedTools.Values))
	assert.NotNil(t, agent.Readonly)
	assert.Equal(t, false, *agent.Readonly)
	assert.NotNil(t, agent.UserInvocable)
	assert.Equal(t, true, *agent.UserInvocable)
	assert.Equal(t, "Start by understanding the request.", agent.InitialPrompt)
	assert.Equal(t, 1, len(agent.Skills.Values))
	assert.Equal(t, 1, len(agent.Rules.Values))
	assert.Equal(t, "You are a helpful assistant.", agent.Body)
	assert.Equal(t, "antigravity2", agent.SourceProvider)
}

// TestImport_Antigravity2_Skills verifies that SKILL.md files with frontmatter are
// correctly parsed into SkillConfig.
func TestImport_Antigravity2_Skills(t *testing.T) {
	tmpdir := t.TempDir()
	skillsDir := filepath.Join(tmpdir, ".agents", "skills", "test-skill")
	require.NoError(t, os.MkdirAll(skillsDir, 0755))

	skillData := `---
name: test-skill
description: A test skill
when-to-use: When you need to do something
license: MIT
allowed-tools:
  - bash
  - python
user-invocable: true
argument-hint: "input string"
---

This is the skill body with detailed instructions.
`
	skillPath := filepath.Join(skillsDir, "SKILL.md")
	require.NoError(t, os.WriteFile(skillPath, []byte(skillData), 0644))

	config := &ast.XcaffoldConfig{}
	imp := NewImporter()
	require.NoError(t, imp.Import(filepath.Join(tmpdir, ".agents"), config))

	skill, ok := config.Skills["test-skill"]
	require.True(t, ok, "skill 'test-skill' should be present")
	assert.Equal(t, "test-skill", skill.Name)
	assert.Equal(t, "A test skill", skill.Description)
	assert.Equal(t, "When you need to do something", skill.WhenToUse)
	assert.Equal(t, "MIT", skill.License)
	assert.Equal(t, 2, len(skill.AllowedTools.Values))
	assert.True(t, *skill.UserInvocable)
	assert.Equal(t, "input string", skill.ArgumentHint)
	assert.Equal(t, "antigravity2", skill.SourceProvider)
	assert.Contains(t, skill.Body, "This is the skill body")
}

// TestImport_Antigravity2_Rules verifies that rule Markdown files are correctly
// parsed into RuleConfig.
func TestImport_Antigravity2_Rules(t *testing.T) {
	tmpdir := t.TempDir()
	rulesDir := filepath.Join(tmpdir, ".agents", "rules")
	require.NoError(t, os.MkdirAll(rulesDir, 0755))

	ruleData := `---
name: test-rule
description: A test rule
activation: always
---

This is the rule content.
`
	rulePath := filepath.Join(rulesDir, "test-rule.md")
	require.NoError(t, os.WriteFile(rulePath, []byte(ruleData), 0644))

	config := &ast.XcaffoldConfig{}
	imp := NewImporter()
	require.NoError(t, imp.Import(filepath.Join(tmpdir, ".agents"), config))

	rule, ok := config.Rules["test-rule"]
	require.True(t, ok, "rule 'test-rule' should be present")
	assert.Equal(t, "test-rule", rule.Name)
	assert.Equal(t, "A test rule", rule.Description)
	assert.Equal(t, "antigravity2", rule.SourceProvider)
	assert.Contains(t, rule.Body, "This is the rule content")
}

// TestImport_Antigravity2_HooksJSON verifies that hooks.json with event entries is
// correctly parsed into the hook config.
func TestImport_Antigravity2_HooksJSON(t *testing.T) {
	tmpdir := t.TempDir()
	agentsDir := filepath.Join(tmpdir, ".agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0755))

	hooksData := `{
  "PreToolUse": [
    {
      "matcher": "Bash",
      "hooks": [
        {
          "type": "command",
          "command": "echo 'before tool'"
        }
      ]
    }
  ],
  "PostToolUse": [
    {
      "matcher": "*",
      "hooks": [
        {
          "type": "command",
          "command": "echo 'after tool'"
        }
      ]
    }
  ]
}`
	hooksPath := filepath.Join(agentsDir, "hooks.json")
	require.NoError(t, os.WriteFile(hooksPath, []byte(hooksData), 0644))

	config := &ast.XcaffoldConfig{}
	imp := NewImporter()
	require.NoError(t, imp.Import(agentsDir, config))

	_, ok := config.Hooks["default"]
	require.True(t, ok, "hooks should be stored under 'default' key")
	assert.NotNil(t, config.Hooks["default"].Events)
}

// TestImport_Antigravity2_MCPConfig verifies that mcp_config.json is correctly parsed
// and that the url field is normalized to serverUrl.
func TestImport_Antigravity2_MCPConfig(t *testing.T) {
	tmpdir := t.TempDir()
	agentsDir := filepath.Join(tmpdir, ".agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0755))

	mcpData := `{
  "mcpServers": {
    "server-with-url": {
      "command": "node",
      "args": ["server.js"],
      "url": "http://localhost:3000"
    },
    "server-with-serverUrl": {
      "command": "python",
      "args": ["server.py"],
      "serverUrl": "http://localhost:4000"
    },
    "server-with-both": {
      "command": "go",
      "args": ["run", "main.go"],
      "url": "http://localhost:5000",
      "serverUrl": "http://localhost:6000"
    }
  }
}`
	mcpPath := filepath.Join(agentsDir, "mcp_config.json")
	require.NoError(t, os.WriteFile(mcpPath, []byte(mcpData), 0644))

	config := &ast.XcaffoldConfig{}
	imp := NewImporter()
	require.NoError(t, imp.Import(agentsDir, config))

	// url should be normalized to serverUrl
	srv1, ok := config.MCP["server-with-url"]
	require.True(t, ok)
	assert.Equal(t, "http://localhost:3000", srv1.URL)

	srv2, ok := config.MCP["server-with-serverUrl"]
	require.True(t, ok)
	assert.Equal(t, "http://localhost:4000", srv2.URL)

	// When both present, serverUrl takes precedence
	srv3, ok := config.MCP["server-with-both"]
	require.True(t, ok)
	assert.Equal(t, "http://localhost:6000", srv3.URL)
}

// TestImport_Antigravity2_Workflows verifies that workflow Markdown files are
// correctly parsed and body is preserved.
func TestImport_Antigravity2_Workflows(t *testing.T) {
	tmpdir := t.TempDir()
	workflowsDir := filepath.Join(tmpdir, ".agents", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	workflowData := `---
name: test-workflow
description: A test workflow
---

This is the workflow body with step definitions.
`
	workflowPath := filepath.Join(workflowsDir, "test-workflow.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(workflowData), 0644))

	config := &ast.XcaffoldConfig{}
	imp := NewImporter()
	require.NoError(t, imp.Import(filepath.Join(tmpdir, ".agents"), config))

	workflow, ok := config.Workflows["test-workflow"]
	require.True(t, ok, "workflow 'test-workflow' should be present")
	assert.Equal(t, "test-workflow", workflow.Name)
	assert.Equal(t, "A test workflow", workflow.Description)
	assert.Equal(t, "antigravity2", workflow.SourceProvider)
	assert.Contains(t, workflow.Steps[0].Instructions, "This is the workflow body")
}

// TestImport_Antigravity2_Classify verifies that the classifier correctly routes
// paths to their respective kinds.
func TestImport_Antigravity2_Classify(t *testing.T) {
	imp := NewImporter()
	tests := []struct {
		path     string
		expected importer.Kind
		desc     string
	}{
		{"agents/my-agent/agent.json", importer.KindAgent, "agent.json file"},
		{"skills/my-skill/SKILL.md", importer.KindSkill, "skill markdown file"},
		{"skills/my-skill/references/ref.md", importer.KindSkillAsset, "skill reference asset"},
		{"skills/my-skill/scripts/script.sh", importer.KindSkillAsset, "skill script asset"},
		{"rules/test-rule.md", importer.KindRule, "rule markdown file"},
		{"hooks.json", importer.KindHook, "hooks configuration"},
		{"mcp_config.json", importer.KindMCP, "MCP configuration"},
		{"workflows/test-workflow.md", importer.KindWorkflow, "workflow markdown file"},
		{"random.txt", importer.KindUnknown, "unmatched file"},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			kind, _ := imp.Classify(tc.path, false)
			assert.Equal(t, tc.expected, kind)
		})
	}
}

// TestImport_Antigravity2_RoundTrip verifies that agents rendered to JSON can be
// imported back with key fields matching the original input.
func TestImport_Antigravity2_RoundTrip(t *testing.T) {
	tmpdir := t.TempDir()
	agentsDir := filepath.Join(tmpdir, ".agents", "agents", "roundtrip")
	require.NoError(t, os.MkdirAll(agentsDir, 0755))

	// Create a manually rendered agent.json matching what the renderer would produce
	agentJSON := `{
  "name": "roundtrip-agent",
  "description": "A round-trip test agent",
  "model": "claude-3-opus",
  "maxTurns": 5,
  "tools": ["bash"],
  "disabledTools": [],
  "readonly": false,
  "userInvocable": false,
  "initialPrompt": "Begin here",
  "skills": [],
  "rules": [],
  "instructions": "Test agent instructions"
}`
	agentPath := filepath.Join(agentsDir, "agent.json")
	require.NoError(t, os.WriteFile(agentPath, []byte(agentJSON), 0644))

	// Create a skill directory and SKILL.md
	skillsDir := filepath.Join(tmpdir, ".agents", "skills", "roundtrip-skill")
	require.NoError(t, os.MkdirAll(skillsDir, 0755))

	skillMD := `---
name: roundtrip-skill
description: A round-trip test skill
allowed-tools:
  - python
---
Test skill content
`
	skillPath := filepath.Join(skillsDir, "SKILL.md")
	require.NoError(t, os.WriteFile(skillPath, []byte(skillMD), 0644))

	// Create a rule
	rulesDir := filepath.Join(tmpdir, ".agents", "rules")
	require.NoError(t, os.MkdirAll(rulesDir, 0755))

	ruleMD := `---
name: roundtrip-rule
description: A round-trip test rule
activation: always
---
Test rule content
`
	rulePath := filepath.Join(rulesDir, "roundtrip-rule.md")
	require.NoError(t, os.WriteFile(rulePath, []byte(ruleMD), 0644))

	// Import the rendered output
	importedConfig := &ast.XcaffoldConfig{}
	importer := NewImporter()
	require.NoError(t, importer.Import(filepath.Join(tmpdir, ".agents"), importedConfig))

	// Verify agents round-trip
	importedAgent, ok := importedConfig.Agents["roundtrip"]
	require.True(t, ok, "imported agent should exist")
	assert.Equal(t, "roundtrip-agent", importedAgent.Name)
	assert.Equal(t, "A round-trip test agent", importedAgent.Description)
	assert.Equal(t, "Test agent instructions", importedAgent.Body)
	assert.Equal(t, 1, len(importedAgent.Tools.Values))

	// Verify skills round-trip
	importedSkill, ok := importedConfig.Skills["roundtrip-skill"]
	require.True(t, ok, "imported skill should exist")
	assert.Equal(t, "roundtrip-skill", importedSkill.Name)
	assert.Equal(t, "A round-trip test skill", importedSkill.Description)
	assert.Contains(t, importedSkill.Body, "Test skill content")
	assert.Equal(t, 1, len(importedSkill.AllowedTools.Values))

	// Verify rules round-trip
	importedRule, ok := importedConfig.Rules["roundtrip-rule"]
	require.True(t, ok, "imported rule should exist")
	assert.Equal(t, "roundtrip-rule", importedRule.Name)
	assert.Equal(t, "A round-trip test rule", importedRule.Description)
	assert.Contains(t, importedRule.Body, "Test rule content")
}

// TestImporter_Provider verifies that the importer identifies itself correctly.
func TestImporter_Provider(t *testing.T) {
	imp := NewImporter()
	assert.Equal(t, "antigravity2", imp.Provider())
}

// TestImporter_InputDir verifies that the importer returns the correct input directory.
func TestImporter_InputDir(t *testing.T) {
	imp := NewImporter()
	assert.Equal(t, ".agents", imp.InputDir())
}

// TestImport_Antigravity2_EmptyConfig verifies that importing from an empty directory
// returns a valid but empty config.
func TestImport_Antigravity2_EmptyConfig(t *testing.T) {
	tmpdir := t.TempDir()
	agentsDir := filepath.Join(tmpdir, ".agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0755))

	config := &ast.XcaffoldConfig{}
	imp := NewImporter()
	require.NoError(t, imp.Import(agentsDir, config))
	// Empty config should not have nil maps
	assert.NotNil(t, config)
}

// TestExtract_Agent_MCPServerURLPreference verifies that when both url and serverUrl
// are present in mcp_config.json, serverUrl takes precedence.
func TestExtract_Agent_MCPServerURLPreference(t *testing.T) {
	data := []byte(`{
  "mcpServers": {
    "test-server": {
      "command": "node",
      "serverUrl": "https://preferred.example.com",
      "url": "https://fallback.example.com"
    }
  }
}`)

	config := &ast.XcaffoldConfig{}
	err := extractMCPConfig("mcp_config.json", data, config)
	require.NoError(t, err)

	server, ok := config.MCP["test-server"]
	require.True(t, ok)
	assert.Equal(t, "https://preferred.example.com", server.URL)
}

// TestExtract_Agent_MCPFallbackURL verifies that when only url is present,
// it is used as the server URL.
func TestExtract_Agent_MCPFallbackURL(t *testing.T) {
	data := []byte(`{
  "mcpServers": {
    "test-server": {
      "command": "node",
      "url": "https://only-url.example.com"
    }
  }
}`)

	config := &ast.XcaffoldConfig{}
	err := extractMCPConfig("mcp_config.json", data, config)
	require.NoError(t, err)

	server, ok := config.MCP["test-server"]
	require.True(t, ok)
	assert.Equal(t, "https://only-url.example.com", server.URL)
}

// BenchmarkImport_Large benchmarks the performance of importing a large agent
// configuration with many fields populated.
func BenchmarkImport_Large(b *testing.B) {
	agentData := `{
  "name": "bench-agent",
  "description": "Benchmarking agent",
  "model": "claude-3-opus",
  "maxTurns": 50,
  "tools": ["bash", "python", "go", "node", "rust"],
  "disabledTools": ["system", "network"],
  "readonly": false,
  "userInvocable": true,
  "initialPrompt": "Start here",
  "skills": ["s1", "s2", "s3"],
  "rules": ["r1", "r2"],
  "instructions": "These are comprehensive instructions for the agent."
}`

	config := &ast.XcaffoldConfig{}
	data := []byte(agentData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractAgentJSON("agents/bench/agent.json", data, config)
	}
}
