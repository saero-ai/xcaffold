package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/importer"
)

func TestParseFrontmatter_EmptyFrontmatter(t *testing.T) {
	input := []byte("---\n---\n\n# Rule: Worktree Creation\n\nThis is the body.")
	var front struct {
		Name string `yaml:"name"`
	}
	body, err := importer.ParseFrontmatterLenient(input, &front)
	require.NoError(t, err)
	assert.Equal(t, "", front.Name, "empty frontmatter should produce zero-value struct")
	assert.Equal(t, "# Rule: Worktree Creation\n\nThis is the body.", body)
	assert.NotContains(t, body, "---", "body must not contain frontmatter delimiters")
}

func TestParseFrontmatter_NormalFrontmatter(t *testing.T) {
	input := []byte("---\nname: developer\ndescription: Dev agent\n---\nYou are a developer.")
	var front struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	body, err := importer.ParseFrontmatterLenient(input, &front)
	require.NoError(t, err)
	assert.Equal(t, "developer", front.Name)
	assert.Equal(t, "Dev agent", front.Description)
	assert.Equal(t, "You are a developer.", body)
}

func TestParseFrontmatter_NoFrontmatter(t *testing.T) {
	input := []byte("# Just markdown\n\nNo frontmatter here.")
	var front struct {
		Name string `yaml:"name"`
	}
	body, err := importer.ParseFrontmatterLenient(input, &front)
	require.NoError(t, err)
	assert.Equal(t, "", front.Name)
	assert.Equal(t, "# Just markdown\n\nNo frontmatter here.", body)
}

func TestExtractMemory_SkipsMemoryMd(t *testing.T) {
	config := &ast.XcaffoldConfig{}

	// MEMORY.md at root of agent-memory/
	require.NoError(t, extractMemory("agent-memory/MEMORY.md", []byte("index"), config))
	assert.Empty(t, config.Memory, "MEMORY.md at root should be skipped")

	// MEMORY.md nested under an agent dir
	require.NoError(t, extractMemory("agent-memory/dev/MEMORY.md", []byte("index"), config))
	assert.Empty(t, config.Memory, "MEMORY.md under agent dir should be skipped")

	// Regular .md should NOT be skipped
	require.NoError(t, extractMemory("agent-memory/dev/real.md", []byte("content"), config))
	assert.Contains(t, config.Memory, "dev/real", "regular memory file should be imported")
}

func TestClaudeExtract_AgentHooks_Preserved(t *testing.T) {
	data := []byte(`---
name: my-agent
description: Agent with hooks
hooks:
  PreToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: echo pre-tool
---
Agent body.
`)
	config := &ast.XcaffoldConfig{}
	err := extractAgent("agents/my-agent.md", data, config)
	require.NoError(t, err)

	agent, ok := config.Agents["my-agent"]
	require.True(t, ok, "agent 'my-agent' should be present")

	require.NotNil(t, agent.Hooks, "hooks should not be nil")
	preToolGroups, ok := agent.Hooks["PreToolUse"]
	require.True(t, ok, "hooks should contain PreToolUse event")
	require.Len(t, preToolGroups, 1, "expected one matcher group")
	assert.Equal(t, "Bash", preToolGroups[0].Matcher)
	require.Len(t, preToolGroups[0].Hooks, 1)
	assert.Equal(t, "command", preToolGroups[0].Hooks[0].Type)
	assert.Equal(t, "echo pre-tool", preToolGroups[0].Hooks[0].Command)
}

func TestClaudeExtract_AgentMCPServers_Preserved(t *testing.T) {
	data := []byte(`---
name: my-agent
description: Agent with MCP servers
mcp-servers:
  my-server:
    type: stdio
    command: /usr/local/bin/my-mcp
    args:
      - --flag
---
Agent body.
`)
	config := &ast.XcaffoldConfig{}
	err := extractAgent("agents/my-agent.md", data, config)
	require.NoError(t, err)

	agent, ok := config.Agents["my-agent"]
	require.True(t, ok, "agent 'my-agent' should be present")

	require.NotNil(t, agent.MCPServers, "mcp-servers should not be nil")
	srv, ok := agent.MCPServers["my-server"]
	require.True(t, ok, "mcp-servers should contain 'my-server'")
	assert.Equal(t, "stdio", srv.Type)
	assert.Equal(t, "/usr/local/bin/my-mcp", srv.Command)
	require.Len(t, srv.Args, 1)
	assert.Equal(t, "--flag", srv.Args[0])
}

func TestImport_Memory_OnlyProjectAgents(t *testing.T) {
	c := NewImporter()
	dir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(dir, "agents"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "agent-memory/a"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "agent-memory/b"), 0755))

	agentContent := []byte("---\nname: Agent A\n---\nHello")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agents/a.md"), agentContent, 0644))

	memAContent := []byte("---\ntype: user\n---\nMemory A")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agent-memory/a/context.md"), memAContent, 0644))

	memBContent := []byte("---\ntype: project\n---\nMemory B")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agent-memory/b/context.md"), memBContent, 0644))

	config := &ast.XcaffoldConfig{}
	err := c.Import(dir, config)
	require.NoError(t, err)

	_, okA := config.Memory["a/context"]
	assert.True(t, okA, "memory a/context should be kept since agent 'a' exists")

	_, okB := config.Memory["b/context"]
	assert.True(t, okB, "memory b/context should be kept even when agent 'b' has no agent definition")

	foundWarning := false
	for _, w := range c.Warnings {
		if strings.Contains(w, `memory for agent "b" has no matching agent definition`) {
			foundWarning = true
			break
		}
	}
	assert.True(t, foundWarning, "expected a warning about missing agent definition for agent b")
}
