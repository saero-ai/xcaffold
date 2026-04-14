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
		"maxTurns:",
		"tools:",
		"disallowedTools:",
		"readonly:",
		"permissionMode:",
		"disableModelInvocation:",
		"userInvocable:",
		"background:",
		"isolation:",
		"memory:",
		"color:",
		"initialPrompt:",
		"skills:",
		"rules:",
		"mcp:",
		"mcpServers:",
		"hooks:",
		"targets:",
		"instructions:",
		"instructions_file:",
	}

	for _, field := range requiredFields {
		require.Contains(t, content, field, "agent reference missing field: %s", field)
	}
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
		"permissionMode:",
		"background:",
		"memory:",
		"skills:",
		"targets:",
		"instructions:",
	}

	lastIdx := -1
	for _, key := range orderedKeys {
		idx := strings.Index(content, key)
		require.NotEqual(t, -1, idx, "key %q not found", key)
		require.Greater(t, idx, lastIdx, "key %q out of order", key)
		lastIdx = idx
	}
}
