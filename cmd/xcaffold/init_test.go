package main

import (
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInitWizard_GeneratesMultiKindFormat verifies that buildXCFContent emits
// a multi-kind scaffold with a kind: project document and a kind: agent document.
func TestInitWizard_GeneratesMultiKindFormat(t *testing.T) {
	ans := wizardAnswers{
		name:      "test-project",
		desc:      "",
		target:    "claude",
		wantAgent: true,
	}
	content := buildXCFContent(ans)

	// Must contain kind: project (not kind: config)
	assert.Contains(t, content, "kind: project")
	assert.NotContains(t, content, "kind: config")

	// name must be at top level, not nested under project:
	assert.Contains(t, content, `name: "test-project"`)
	assert.NotContains(t, content, "project:")

	// Must declare targets
	assert.Contains(t, content, "targets:")
	assert.Contains(t, content, "- claude")

	// Must contain kind: agent document
	assert.Contains(t, content, "kind: agent")
	assert.Contains(t, content, "name: developer")

	// Must have --- separator between documents
	assert.Contains(t, content, "---")

	// Must be parseable as a valid XcaffoldConfig
	config, err := parser.Parse(strings.NewReader(content))
	require.NoError(t, err)
	require.NotNil(t, config.Project)
	assert.Equal(t, "test-project", config.Project.Name)
	assert.Equal(t, []string{"claude"}, config.Project.Targets)
	assert.Contains(t, config.Agents, "developer")
}

// TestInitWizard_GeneratesMultiKindFormat_NoAgent verifies that when wantAgent
// is false, only the kind: project document is emitted (no separator, no agent).
func TestInitWizard_GeneratesMultiKindFormat_NoAgent(t *testing.T) {
	ans := wizardAnswers{
		name:      "test-project",
		desc:      "",
		target:    "claude",
		wantAgent: false,
	}
	content := buildXCFContent(ans)
	assert.Contains(t, content, "kind: project")
	assert.NotContains(t, content, "kind: config")
	assert.NotContains(t, content, "kind: agent")
	assert.Contains(t, content, "targets:")
}

// TestInitWizard_GeneratesMultiKindFormat_TargetCursor verifies that the
// targets field reflects the chosen target when it is not "claude".
func TestInitWizard_GeneratesMultiKindFormat_TargetCursor(t *testing.T) {
	ans := wizardAnswers{
		name:      "cursor-project",
		desc:      "",
		target:    "cursor",
		wantAgent: false,
	}
	content := buildXCFContent(ans)
	assert.Contains(t, content, "kind: project")
	assert.Contains(t, content, "- cursor")

	// Must round-trip through the parser
	config, err := parser.Parse(strings.NewReader(content))
	require.NoError(t, err)
	require.NotNil(t, config.Project)
	assert.Equal(t, []string{"cursor"}, config.Project.Targets)
}
