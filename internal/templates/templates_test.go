package templates

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- RenderProjectXCF ---

func TestRenderProjectXCF_SingleTarget(t *testing.T) {
	out := RenderProjectXCF("my-api", []string{"claude"})

	assert.Contains(t, out, "kind: project")
	assert.Contains(t, out, `name: "my-api"`)
	assert.Contains(t, out, "targets:")
	assert.Contains(t, out, "- claude")
	assert.Contains(t, out, "agents:")
	assert.Contains(t, out, "- xaff")
	assert.Contains(t, out, "rules:")
	assert.Contains(t, out, "- xcf-conventions")
	assert.Contains(t, out, "skills:")
	assert.Contains(t, out, "- xcaffold")
	assert.Contains(t, out, "policies:")
	assert.Contains(t, out, "- require-agent-description")
	assert.NotContains(t, out, "- cursor")
	// must not reference old names
	assert.NotContains(t, out, "- developer")
	assert.NotContains(t, out, "- conventions")
}

func TestRenderProjectXCF_MultiTarget(t *testing.T) {
	out := RenderProjectXCF("multi", []string{"claude", "cursor", "gemini"})

	assert.Contains(t, out, "- claude")
	assert.Contains(t, out, "- cursor")
	assert.Contains(t, out, "- gemini")
}

// --- RenderXaffAgentXCF ---

func TestRenderXaffAgentXCF_ContainsMatrix(t *testing.T) {
	out := RenderXaffAgentXCF("claude-sonnet-4-6", []string{"claude", "cursor"})

	assert.Contains(t, out, "kind: agent")
	assert.Contains(t, out, "name: xaff")
	assert.Contains(t, out, "model:")
	assert.Contains(t, out, "kind: agent - provider field support")
	assert.Contains(t, out, "claude")
	assert.Contains(t, out, "cursor")
	assert.Contains(t, out, "dropped")
}

func TestRenderXaffAgentXCF_FrontmatterFormat(t *testing.T) {
	out := RenderXaffAgentXCF("claude-sonnet-4-6", []string{"claude"})

	assert.Contains(t, out, "---\nkind: agent")
	assert.Contains(t, out, "skills: [xcaffold]")
	assert.Contains(t, out, "rules: [xcf-conventions]")
	assert.NotContains(t, out, "instructions: |")
}

func TestRenderXaffAgentXCF_HasXaffBody(t *testing.T) {
	out := RenderXaffAgentXCF("claude-sonnet-4-6", []string{"claude"})

	assert.Contains(t, out, "You are Xaff")
	assert.Contains(t, out, "xcaffold")
}

// --- RenderXaffOverrideXCF ---

func TestRenderXaffOverrideXCF_Claude(t *testing.T) {
	out := RenderXaffOverrideXCF("claude")

	assert.Contains(t, out, "kind: agent")
	assert.Contains(t, out, "name: xaff")
	assert.Contains(t, out, "effort:")
	assert.Contains(t, out, "permission-mode:")
}

func TestRenderXaffOverrideXCF_Cursor(t *testing.T) {
	out := RenderXaffOverrideXCF("cursor")

	assert.Contains(t, out, "kind: agent")
	assert.Contains(t, out, "name: xaff")
	assert.Contains(t, out, "readonly:")
}

func TestRenderXaffOverrideXCF_AllProviders(t *testing.T) {
	for _, target := range []string{"claude", "cursor", "gemini", "copilot", "antigravity"} {
		out := RenderXaffOverrideXCF(target)
		assert.Contains(t, out, "kind: agent", "override for %s must contain kind: agent", target)
		assert.Contains(t, out, "name: xaff", "override for %s must contain name: xaff", target)
	}
}

// --- RenderXcfConventionsRuleXCF ---

func TestRenderXcfConventionsRuleXCF_ContainsMatrix(t *testing.T) {
	out := RenderXcfConventionsRuleXCF([]string{"claude", "cursor"})

	assert.Contains(t, out, "kind: rule")
	assert.Contains(t, out, "name: xcf-conventions")
	assert.Contains(t, out, "activation: always")
	assert.Contains(t, out, "kind: rule - provider field support")
}

func TestRenderXcfConventionsRuleXCF_FrontmatterFormat(t *testing.T) {
	out := RenderXcfConventionsRuleXCF([]string{"claude"})

	assert.Contains(t, out, "---\nkind: rule")
	assert.Contains(t, out, "kebab-case")
	assert.NotContains(t, out, "instructions: |")
}

// --- RenderAgentXCF (backwards compat) ---

func TestRenderAgentXCF_ContainsMatrix(t *testing.T) {
	out := RenderAgentXCF("developer", "claude-sonnet-4-6", []string{"claude", "cursor"})

	assert.Contains(t, out, "kind: agent")
	assert.Contains(t, out, "name: developer")
	assert.Contains(t, out, "model:")
	assert.Contains(t, out, "kind: agent - provider field support")
	assert.Contains(t, out, "claude")
	assert.Contains(t, out, "cursor")
	assert.Contains(t, out, "dropped")
}

func TestRenderAgentXCF_FrontmatterFormat(t *testing.T) {
	out := RenderAgentXCF("developer", "claude-sonnet-4-6", []string{"claude"})

	assert.Contains(t, out, "---\nkind: agent")
	assert.Contains(t, out, "---\nYou are a software developer.")
	assert.NotContains(t, out, "instructions: |")
}

func TestRenderAgentXCF_SingleTarget_NoCursorColumn(t *testing.T) {
	out := RenderAgentXCF("developer", "claude-sonnet-4-6", []string{"claude"})

	assert.Contains(t, out, "kind: agent - provider field support")

	matrixBlock := out[:strings.Index(out, "kind: agent\n")]
	assert.NotContains(t, matrixBlock, "cursor")
}

// --- RenderSettingsXCF ---

func TestRenderSettingsXCF_ContainsMatrix(t *testing.T) {
	out := RenderSettingsXCF([]string{"claude"})

	assert.Contains(t, out, "kind: settings")
	assert.Contains(t, out, "kind: settings - provider field support")
	assert.Contains(t, out, "mcp-servers")
	assert.Contains(t, out, "permissions")
}

// --- RenderPolicyDescriptionXCF ---

func TestRenderPolicyDescriptionXCF(t *testing.T) {
	out := RenderPolicyDescriptionXCF()

	assert.Contains(t, out, "kind: policy")
	assert.Contains(t, out, "name: require-agent-description")
	assert.Contains(t, out, "severity: warning")
	assert.Contains(t, out, "target: agent")
	assert.Contains(t, out, "is-present: true")
	assert.NotContains(t, out, "\n---\n")
}

// --- RenderPolicyInstructionsXCF ---

func TestRenderPolicyInstructionsXCF(t *testing.T) {
	out := RenderPolicyInstructionsXCF()

	assert.Contains(t, out, "kind: policy")
	assert.Contains(t, out, "name: require-agent-instructions")
	assert.Contains(t, out, "severity: error")
	assert.Contains(t, out, "target: agent")
	assert.Contains(t, out, "min-length: 10")
	assert.NotContains(t, out, "\n---\n")
}

// --- RenderXcaffoldSkillXCF ---

func TestRenderXcaffoldSkillXCF_FrontmatterFormat(t *testing.T) {
	out := RenderXcaffoldSkillXCF([]string{"claude"})

	assert.True(t, strings.HasPrefix(out, "---\n"), "must start with frontmatter delimiter")
	assert.Contains(t, out, "\nkind: skill\n")

	parts := strings.SplitN(out, "---\n", 3)
	require.Len(t, parts, 3, "output must have exactly two --- delimiters")
	frontmatter := parts[1]
	assert.NotContains(t, frontmatter, "instructions: |", "frontmatter must not use legacy block scalar format")

	assert.NotContains(t, out, "\n  # xcaffold")
}
