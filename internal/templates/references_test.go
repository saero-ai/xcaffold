package templates

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderAgentReference_IncludesAllCanonicalFields(t *testing.T) {
	content := RenderAgentReference()

	requiredFields := []string{
		"name:",
		"description:",
		"model:",
		"effort:",
		"max-turns:",
		"tools:",
		"disallowed-tools:",
		"readonly:",
		"permission-mode:",
		"disable-model-invocation:",
		"user-invocable:",
		"background:",
		"isolation:",
		"memory:",
		"color:",
		"initial-prompt:",
		"skills:",
		"rules:",
		"mcp:",
		"mcp-servers:",
		"hooks:",
		"targets:",
		"instructions-file:",
	}

	for _, field := range requiredFields {
		require.Contains(t, content, field, "agent reference missing field: %s", field)
	}
}

func TestRenderAgentReference_FrontmatterFormat(t *testing.T) {
	content := RenderAgentReference()

	// Must NOT use inline YAML instructions key
	require.NotContains(t, content, "instructions: |", "agent reference must not use inline YAML instructions block")

	// Must have a closing --- body separator
	require.Contains(t, content, "---\n", "agent reference must contain closing frontmatter delimiter")

	// Body text must appear after the second ---
	firstDelim := strings.Index(content, "---")
	require.NotEqual(t, -1, firstDelim)
	rest := content[firstDelim+3:]
	secondDelim := strings.Index(rest, "---")
	require.NotEqual(t, -1, secondDelim, "agent reference must have a second --- delimiter")
	body := rest[secondDelim+3:]
	require.True(t, len(strings.TrimSpace(body)) > 0, "body content must appear after closing ---")
}

func TestRenderAgentReference_IncludesProviderExamples(t *testing.T) {
	content := RenderAgentReference()

	require.Contains(t, content, "temperature:")
	require.Contains(t, content, "timeout_mins:")
	require.Contains(t, content, "kind: local")
	require.Contains(t, content, "provider:")
}

func TestRenderAgentReference_HasHeader(t *testing.T) {
	content := RenderAgentReference()
	require.Contains(t, content, "Agent Kind — Full Field Reference")
	require.Contains(t, content, "This file is NOT parsed by xcaffold")
}

func TestRenderAgentReference_FieldOrdering(t *testing.T) {
	content := RenderAgentReference()

	orderedKeys := []string{
		"name:",
		"description:",
		"model:",
		"tools:",
		"permission-mode:",
		"background:",
		"memory:",
		"skills:",
		"targets:",
	}

	lastIdx := -1
	for _, key := range orderedKeys {
		idx := strings.Index(content, key)
		require.NotEqual(t, -1, idx, "key %q not found", key)
		require.Greater(t, idx, lastIdx, "key %q out of order", key)
		lastIdx = idx
	}

	// Body separator must appear after all YAML fields
	firstDelim := strings.Index(content, "---")
	require.NotEqual(t, -1, firstDelim)
	rest := content[firstDelim+3:]
	secondDelim := strings.Index(rest, "---")
	require.NotEqual(t, -1, secondDelim, "closing --- must appear after all YAML fields")
	closingDelimAbsIdx := firstDelim + 3 + secondDelim
	require.Greater(t, closingDelimAbsIdx, lastIdx, "closing --- must appear after targets:")
}
