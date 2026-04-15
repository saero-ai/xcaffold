package templates

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderSkillReference_ContainsAllCanonicalFields(t *testing.T) {
	out := RenderSkillReference()

	// Section headers (descriptive — never "Group N:")
	require.Contains(t, out, "── Identity")
	require.Contains(t, out, "── Tool Access")
	require.Contains(t, out, "── Permissions & Invocation Control")
	require.Contains(t, out, "── Composition")
	require.Contains(t, out, "── Multi-Target")
	require.Contains(t, out, "── Instructions")

	// Identity fields
	require.Contains(t, out, "name:")
	require.Contains(t, out, "description:")
	require.Contains(t, out, "when-to-use:")
	require.Contains(t, out, "license:")

	// Tool access (canonical name, not legacy "tools:")
	require.Contains(t, out, "allowed-tools:")
	require.NotContains(t, out, "\ntools:")

	// Invocation control
	require.Contains(t, out, "disable-model-invocation:")
	require.Contains(t, out, "user-invocable:")
	require.Contains(t, out, "argument-hint:")

	// Composition
	require.Contains(t, out, "references:")
	require.Contains(t, out, "scripts:")
	require.Contains(t, out, "assets:")

	// Multi-target with provider passthrough
	require.Contains(t, out, "targets:")
	require.Contains(t, out, "provider:")

	// Instructions LAST
	idxInstructions := strings.Index(out, "instructions:")
	idxTargets := strings.Index(out, "targets:")
	require.True(t, idxInstructions > idxTargets, "instructions must appear after targets")

	// Header annotation about not being parsed
	require.Contains(t, out, "NOT parsed")
}

func TestRenderSkillReference_NoGroupNumbersInOutput(t *testing.T) {
	out := RenderSkillReference()
	for _, forbidden := range []string{"Group 1:", "Group 3:", "Group 4:", "Group 7:", "Group 9:", "Group 10:"} {
		require.NotContains(t, out, forbidden, "user-facing reference must not expose internal group numbering")
	}
}
