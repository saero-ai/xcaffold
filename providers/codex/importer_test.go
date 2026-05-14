package codex

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/importer"
)

// TestClassify_AgentTOML verifies that "agents/my-agent.toml" is classified as KindAgent, FlatFile
func TestClassify_AgentTOML(t *testing.T) {
	c := NewImporter()
	kind, layout := c.Classify("agents/my-agent.toml", false)
	assert.Equal(t, importer.KindAgent, kind, "agents/*.toml should be KindAgent")
	assert.Equal(t, importer.FlatFile, layout, "agents/*.toml should be FlatFile")
}

// TestClassify_HooksJSON verifies that "hooks.json" is classified as KindHook, StandaloneJSON
func TestClassify_HooksJSON(t *testing.T) {
	c := NewImporter()
	kind, layout := c.Classify("hooks.json", false)
	assert.Equal(t, importer.KindHook, kind, "hooks.json should be KindHook")
	assert.Equal(t, importer.StandaloneJSON, layout, "hooks.json should be StandaloneJSON")
}

// TestClassify_HookScript verifies that "hooks/my-hook.sh" is classified as KindHookScript, FlatFile
func TestClassify_HookScript(t *testing.T) {
	c := NewImporter()
	kind, layout := c.Classify("hooks/my-hook.sh", false)
	assert.Equal(t, importer.KindHookScript, kind, "hooks/*.sh should be KindHookScript")
	assert.Equal(t, importer.FlatFile, layout, "hooks/*.sh should be FlatFile")
}

// TestClassify_Unknown verifies that "random.txt" is classified as KindUnknown, LayoutUnknown
func TestClassify_Unknown(t *testing.T) {
	c := NewImporter()
	kind, layout := c.Classify("random.txt", false)
	assert.Equal(t, importer.KindUnknown, kind, "unknown files should be KindUnknown")
	assert.Equal(t, importer.LayoutUnknown, layout, "unknown files should be LayoutUnknown")
}

// TestExtract_Agent_ValidTOML verifies that valid TOML agent data populates config.Agents with correct fields
func TestExtract_Agent_ValidTOML(t *testing.T) {
	validTOML := []byte(`
name = "my-agent"
description = "A test agent"
model = "gpt-4"
developer_instructions = "You are an expert developer."
sandbox_mode = "strict"
effort = "high"
initial_prompt = "Start by analyzing the request."
memory = "long-term"
tools = ["bash", "python"]
disallowed_tools = ["system"]
max_turns = 10
`)
	config := &ast.XcaffoldConfig{}
	err := extractAgent("agents/my-agent.toml", validTOML, config)
	require.NoError(t, err, "should parse valid TOML without error")

	agent, ok := config.Agents["my-agent"]
	require.True(t, ok, "agent 'my-agent' should be present in config")
	assert.Equal(t, "my-agent", agent.Name, "agent Name should match")
	assert.Equal(t, "A test agent", agent.Description, "agent Description should match")
	assert.Equal(t, "gpt-4", agent.Model, "agent Model should match")
	assert.Equal(t, "You are an expert developer.", agent.Body, "agent Body should contain developer_instructions")
	assert.Equal(t, "codex", agent.SourceProvider, "SourceProvider should be 'codex'")
	assert.Equal(t, 2, len(agent.Tools.Values), "agent Tools should have 2 entries")
	assert.Equal(t, "bash", agent.Tools.Values[0])
	assert.Equal(t, "python", agent.Tools.Values[1])
}

// TestExtract_Agent_NameValidation verifies that agent names with invalid patterns are rejected
func TestExtract_Agent_NameValidation(t *testing.T) {
	tests := []struct {
		name      string
		shouldErr bool
		desc      string
	}{
		{"my-agent", false, "valid kebab-case name"},
		{"agent123", false, "valid name with digits"},
		{"../evil", true, "invalid: path traversal"},
		{"My Agent", true, "invalid: uppercase and space"},
		{"agent_1", true, "invalid: underscore"},
		{"agent@home", true, "invalid: special character"},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			tomlData := []byte(`
name = "` + tc.name + `"
description = "Test"
model = "gpt-4"
`)
			config := &ast.XcaffoldConfig{}
			err := extractAgent("agents/test.toml", tomlData, config)
			if tc.shouldErr {
				require.Error(t, err, "should reject invalid name: %s", tc.name)
				assert.Contains(t, err.Error(), "invalid name", "error should mention invalid name")
			} else {
				require.NoError(t, err, "should accept valid name: %s", tc.name)
			}
		})
	}
}

// TestExtract_Agent_SizeGate verifies that agent files exceeding 1 MiB are rejected
func TestExtract_Agent_SizeGate(t *testing.T) {
	// Create data that exceeds 1 MiB (1<<20 = 1048576 bytes)
	oversizedData := make([]byte, (1<<20)+1)
	// Write minimal valid TOML at the start
	copy(oversizedData, []byte(`name = "big"`))

	config := &ast.XcaffoldConfig{}
	err := extractAgent("agents/oversized.toml", oversizedData, config)
	require.Error(t, err, "should reject oversized agent")
	assert.Contains(t, err.Error(), "size limit", "error should mention size limit")
}

// TestExtract_Agent_InvalidTOML verifies that malformed TOML is rejected with a parse error
func TestExtract_Agent_InvalidTOML(t *testing.T) {
	invalidTOML := []byte(`
name = "broken
description = "Missing closing quote"
model = [unclosed array
`)
	config := &ast.XcaffoldConfig{}
	err := extractAgent("agents/broken.toml", invalidTOML, config)
	require.Error(t, err, "should reject invalid TOML")
	assert.Contains(t, err.Error(), "agents/broken.toml", "error should include file path")
}

// TestExtract_Hooks_ValidJSON verifies that valid JSON hooks populate config.Hooks
func TestExtract_Hooks_ValidJSON(t *testing.T) {
	hooksData := []byte(`{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "echo 'pre-tool hook'"
          }
        ]
      }
    ]
  }
}`)
	config := &ast.XcaffoldConfig{}
	err := extractHooks("hooks.json", hooksData, config)
	require.NoError(t, err, "should parse valid hooks JSON without error")

	_, ok := config.Hooks["default"]
	require.True(t, ok, "hooks should be stored under 'default' key")
}

// TestNewImporter_Defaults verifies that NewImporter() returns correct defaults
func TestNewImporter_Defaults(t *testing.T) {
	c := NewImporter()
	assert.Equal(t, "codex", c.Provider(), "Provider should be 'codex'")
	assert.Equal(t, ".codex", c.InputDir(), "InputDir should be '.codex'")
}

// TestExtract_Agent_MissingRequiredFields verifies that missing required fields are handled
func TestExtract_Agent_MissingRequiredFields(t *testing.T) {
	// Agent with only name, missing model
	minimalTOML := []byte(`
name = "minimal-agent"
`)
	config := &ast.XcaffoldConfig{}
	err := extractAgent("agents/minimal.toml", minimalTOML, config)
	require.NoError(t, err, "should allow agent with missing optional fields")

	agent, ok := config.Agents["minimal"]
	require.True(t, ok, "agent should be created")
	assert.Equal(t, "minimal-agent", agent.Name)
	assert.Equal(t, "", agent.Model, "missing model should be empty string")
}

// TestExtract_Agent_ToolsPreserved verifies that Tools and DisallowedTools lists are preserved correctly
func TestExtract_Agent_ToolsPreserved(t *testing.T) {
	tomlData := []byte(`
name = "tool-agent"
description = "Tests tool handling"
model = "gpt-4"
tools = ["bash", "python", "go"]
disallowed_tools = ["system", "network"]
`)
	config := &ast.XcaffoldConfig{}
	err := extractAgent("agents/tool-agent.toml", tomlData, config)
	require.NoError(t, err)

	agent, ok := config.Agents["tool-agent"]
	require.True(t, ok)
	assert.Equal(t, 3, len(agent.Tools.Values), "should have 3 allowed tools")
	assert.Equal(t, 2, len(agent.DisallowedTools.Values), "should have 2 disallowed tools")
	assert.Contains(t, agent.Tools.Values, "python")
	assert.Contains(t, agent.DisallowedTools.Values, "network")
}

// TestClassify_MultipleExtensions verifies classification with various file types
func TestClassify_MultipleExtensions(t *testing.T) {
	c := NewImporter()
	tests := []struct {
		path     string
		expected importer.Kind
		desc     string
	}{
		{"agents/agent-1.toml", importer.KindAgent, "toml agent file"},
		{"agents/agent-2.TOML", importer.KindUnknown, "uppercase TOML extension (exact match only)"},
		{"hooks/setup.sh", importer.KindHookScript, "shell hook script"},
		{"hooks/cleanup.sh", importer.KindHookScript, "shell cleanup script"},
		{"config.yaml", importer.KindUnknown, "unmatched YAML file"},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			kind, _ := c.Classify(tc.path, false)
			assert.Equal(t, tc.expected, kind)
		})
	}
}

// TestExtract_Agent_EffortPreserved verifies that the effort level is preserved in the agent config
func TestExtract_Agent_EffortPreserved(t *testing.T) {
	tomlData := []byte(`
name = "effort-agent"
description = "Tests effort field"
model = "gpt-4"
effort = "extreme"
`)
	config := &ast.XcaffoldConfig{}
	err := extractAgent("agents/effort-agent.toml", tomlData, config)
	require.NoError(t, err)

	agent, ok := config.Agents["effort-agent"]
	require.True(t, ok)
	assert.Equal(t, "extreme", agent.Effort, "effort field should be preserved")
}

// TestExtract_Agent_MaxTurnsAsPtr verifies that max_turns is converted to a pointer (nil if zero)
func TestExtract_Agent_MaxTurnsAsPtr(t *testing.T) {
	tests := []struct {
		toml      string
		expectNil bool
		value     int
		desc      string
	}{
		{`max_turns = 0`, true, 0, "zero max_turns should result in nil pointer"},
		{`max_turns = 10`, false, 10, "non-zero max_turns should be pointer to value"},
		{``, true, 0, "missing max_turns should result in nil pointer"},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			tomlData := []byte(`
name = "maxturns-agent"
model = "gpt-4"
` + tc.toml)
			config := &ast.XcaffoldConfig{}
			err := extractAgent("agents/maxturns.toml", tomlData, config)
			require.NoError(t, err)

			agent, ok := config.Agents["maxturns"]
			require.True(t, ok)
			if tc.expectNil {
				assert.Nil(t, agent.MaxTurns, "MaxTurns should be nil")
			} else {
				require.NotNil(t, agent.MaxTurns, "MaxTurns should not be nil")
				assert.Equal(t, tc.value, *agent.MaxTurns)
			}
		})
	}
}
