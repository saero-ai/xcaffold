package parser

import (
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
