package main

import (
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInitWizard_GeneratesMultiKindFormat verifies that buildXCFContent emits
// a multi-kind scaffold with separate kind: config and kind: agent documents.
func TestInitWizard_GeneratesMultiKindFormat(t *testing.T) {
	ans := wizardAnswers{
		name:      "test-project",
		desc:      "",
		target:    "claude",
		wantAgent: true,
	}
	content := buildXCFContent(ans)

	// Must contain kind: config document
	assert.Contains(t, content, "kind: config")
	assert.Contains(t, content, `name: "test-project"`)

	// Must contain kind: agent document
	assert.Contains(t, content, "kind: agent")
	assert.Contains(t, content, "name: developer")

	// Must have --- separator
	assert.Contains(t, content, "---")

	// Must NOT contain nested agents: under project:
	assert.NotContains(t, content, "  agents:")

	// Must be parseable as a valid XcaffoldConfig
	config, err := parser.Parse(strings.NewReader(content))
	require.NoError(t, err)
	assert.Equal(t, "test-project", config.Project.Name)
	assert.Contains(t, config.Agents, "developer")
}

// TestInitWizard_GeneratesMultiKindFormat_NoAgent verifies that when wantAgent
// is false, only the kind: config document is emitted (no separator, no agent).
func TestInitWizard_GeneratesMultiKindFormat_NoAgent(t *testing.T) {
	ans := wizardAnswers{
		name:      "test-project",
		desc:      "",
		target:    "claude",
		wantAgent: false,
	}
	content := buildXCFContent(ans)
	assert.Contains(t, content, "kind: config")
	assert.NotContains(t, content, "kind: agent")
}
