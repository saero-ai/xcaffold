package parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_AgentReferencesUndefinedSkill(t *testing.T) {
	yaml := `
version: "1.0"
project:
  name: "test-project"
agents:
  developer:
    description: "A developer agent"
    skills: [git-workflow, code-review]
skills:
  git-workflow:
    description: "Git workflow skill"
`
	_, err := Parse(strings.NewReader(yaml))
	require.Error(t, err, "agent referencing undefined skill must be rejected")
	assert.Contains(t, err.Error(), "code-review")
	assert.Contains(t, err.Error(), "developer")
}

func TestParse_AgentReferencesUndefinedRule(t *testing.T) {
	yaml := `
version: "1.0"
project:
  name: "test-project"
agents:
  backend:
    description: "Backend agent"
    rules: [typescript, security]
rules:
  typescript:
    description: "TypeScript conventions"
`
	_, err := Parse(strings.NewReader(yaml))
	require.Error(t, err, "agent referencing undefined rule must be rejected")
	assert.Contains(t, err.Error(), "security")
	assert.Contains(t, err.Error(), "backend")
}

func TestParse_AgentReferencesUndefinedMCP(t *testing.T) {
	yaml := `
version: "1.0"
project:
  name: "test-project"
agents:
  developer:
    description: "Dev agent"
    mcp: [postgres, slack]
mcp:
  postgres:
    type: stdio
    command: "pg-mcp"
`
	_, err := Parse(strings.NewReader(yaml))
	require.Error(t, err, "agent referencing undefined MCP server must be rejected")
	assert.Contains(t, err.Error(), "slack")
	assert.Contains(t, err.Error(), "developer")
}

func TestParse_ValidCrossReferences(t *testing.T) {
	yaml := `
version: "1.0"
project:
  name: "test-project"
agents:
  developer:
    description: "Dev agent"
    skills: [git-workflow]
    rules: [typescript]
    mcp: [postgres]
skills:
  git-workflow:
    description: "Git workflow"
rules:
  typescript:
    description: "TS rules"
mcp:
  postgres:
    type: stdio
    command: "pg-mcp"
`
	cfg, err := Parse(strings.NewReader(yaml))
	require.NoError(t, err, "valid cross-references should pass")
	require.NotNil(t, cfg)
}

func TestParse_AgentWithNoReferences(t *testing.T) {
	yaml := `
version: "1.0"
project:
  name: "test-project"
agents:
  developer:
    description: "Standalone agent"
`
	cfg, err := Parse(strings.NewReader(yaml))
	require.NoError(t, err, "agent without references should parse")
	require.NotNil(t, cfg)
}

func TestParse_SkillReferencesUndefinedAgent(t *testing.T) {
	yaml := `
version: "1.0"
project:
  name: "test-project"
skills:
  deploy:
    description: "Deploy skill"
    agent: "devops"
`
	_, err := Parse(strings.NewReader(yaml))
	require.Error(t, err, "skill referencing undefined agent must be rejected")
	assert.Contains(t, err.Error(), "devops")
	assert.Contains(t, err.Error(), "deploy")
}
