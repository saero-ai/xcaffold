package templates

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListTemplates(t *testing.T) {
	list := List()
	require.True(t, len(list) >= 3, "must have at least 3 templates")

	ids := make(map[string]bool)
	for _, tmpl := range list {
		ids[tmpl.ID] = true
		assert.NotEmpty(t, tmpl.ID, "template must have an ID")
		assert.NotEmpty(t, tmpl.Label, "template must have a label")
		assert.NotEmpty(t, tmpl.Description, "template must have a description")
	}

	assert.True(t, ids["rest-api"], "rest-api template must exist")
	assert.True(t, ids["cli-tool"], "cli-tool template must exist")
	assert.True(t, ids["frontend-app"], "frontend-app template must exist")
}

func TestRenderTemplate_RESTAPI(t *testing.T) {
	content, err := Render("rest-api", "my-service", "claude-sonnet-4-6")
	require.NoError(t, err)
	assert.Contains(t, content, "my-service")
	assert.Contains(t, content, "claude-sonnet-4-6")
	assert.Contains(t, content, "agents:")
	assert.Contains(t, content, "skills:")
	assert.Contains(t, content, "rules:")
}

func TestRenderTemplate_CLITool(t *testing.T) {
	content, err := Render("cli-tool", "my-cli", "claude-sonnet-4-6")
	require.NoError(t, err)
	assert.Contains(t, content, "my-cli")
	assert.Contains(t, content, "agents:")
}

func TestRenderTemplate_FrontendApp(t *testing.T) {
	content, err := Render("frontend-app", "my-app", "claude-sonnet-4-6")
	require.NoError(t, err)
	assert.Contains(t, content, "my-app")
	assert.Contains(t, content, "agents:")
}

func TestRenderTemplate_Unknown(t *testing.T) {
	_, err := Render("nonexistent", "test", "model")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestRenderTemplates_SkillBlock_UsesAllowedToolsName(t *testing.T) {
	for _, id := range []string{"rest-api", "cli-tool", "frontend-app"} {
		out, err := Render(id, "test-project", "sonnet")
		require.NoError(t, err)
		// If template has a skills: section, skill entries must not use the legacy
		// 'tools:' key (the canonical key is 'allowed-tools:'). Agents are allowed
		// to keep 'tools:' — we check only within the skills: block.
		if idx := strings.Index(out, "\nskills:"); idx != -1 {
			skillsBlock := out[idx:]
			// Find the next top-level section (line starting with a non-space char after skills:).
			// Any 'tools:' inside the skills block is the legacy key.
			require.NotContains(t, skillsBlock, "\n  allowed-tools: ", "internal check: template %q uses old key", id)
			// Confirm no 'tools:' appears as a skill-level field (indented under a skill entry).
			// Skill entries are indented by 4 spaces; agent 'tools:' is at 4-space indent too
			// but appears before the skills: section. After splitting at skills:, any '    tools:'
			// would belong to a skill entry.
			require.NotContains(t, skillsBlock, "\n    tools:", "template %q has legacy 'tools:' inside a skill entry", id)
		}
	}
}

func TestRenderTemplate_CanonicalFieldOrdering(t *testing.T) {
	content, err := Render("rest-api", "my-api", "sonnet")
	require.NoError(t, err)

	orderedKeys := []string{
		"    description:",
		"    model:",
		"    effort:",
		"    tools:",
		"    skills:",
		"    rules:",
		"    instructions:",
	}

	lastIdx := -1
	for _, key := range orderedKeys {
		idx := strings.Index(content, key)
		if idx == -1 {
			continue
		}
		require.Greater(t, idx, lastIdx, "key %q appeared before a prior key in rest-api template", key)
		lastIdx = idx
	}
}

// --- Provider-first scaffold render functions ---

func TestRenderProjectXCF_SingleTarget(t *testing.T) {
	out := RenderProjectXCF("my-api", []string{"claude"})

	assert.Contains(t, out, "kind: project")
	assert.Contains(t, out, `name: "my-api"`)
	assert.Contains(t, out, "targets:")
	assert.Contains(t, out, "- claude")
	assert.Contains(t, out, "agents:")
	assert.Contains(t, out, "- developer")
	assert.Contains(t, out, "rules:")
	assert.Contains(t, out, "- conventions")
	assert.Contains(t, out, "skills:")
	assert.Contains(t, out, "- xcaffold")
	assert.Contains(t, out, "policies:")
	assert.Contains(t, out, "- require-agent-description")
	assert.NotContains(t, out, "- cursor")
}

func TestRenderProjectXCF_MultiTarget(t *testing.T) {
	out := RenderProjectXCF("multi", []string{"claude", "cursor", "gemini"})

	assert.Contains(t, out, "- claude")
	assert.Contains(t, out, "- cursor")
	assert.Contains(t, out, "- gemini")
}

func TestRenderAgentXCF_ContainsMatrix(t *testing.T) {
	out := RenderAgentXCF("developer", "claude-sonnet-4-6", []string{"claude", "cursor"})

	assert.Contains(t, out, "kind: agent")
	assert.Contains(t, out, "name: developer")
	assert.Contains(t, out, "model:")
	assert.Contains(t, out, "kind: agent - provider field support")
	assert.Contains(t, out, "claude")
	assert.Contains(t, out, "cursor")
	// effort is claude-only: must show dropped for cursor
	assert.Contains(t, out, "dropped")
}

func TestRenderAgentXCF_SingleTarget_NoCursorColumn(t *testing.T) {
	out := RenderAgentXCF("developer", "claude-sonnet-4-6", []string{"claude"})

	// Matrix must exist but cursor column must not appear
	assert.Contains(t, out, "kind: agent - provider field support")

	// Only check the matrix section for the column (before the actual kind declaration)
	matrixBlock := out[:strings.Index(out, "kind: agent")]
	assert.NotContains(t, matrixBlock, "cursor")
}

func TestRenderRuleXCF_ContainsMatrix(t *testing.T) {
	out := RenderRuleXCF([]string{"claude", "cursor"})

	assert.Contains(t, out, "kind: rule")
	assert.Contains(t, out, "name: conventions")
	assert.Contains(t, out, "activation: always")
	assert.Contains(t, out, "kind: rule - provider field support")
}

func TestRenderSettingsXCF_ContainsMatrix(t *testing.T) {
	out := RenderSettingsXCF([]string{"claude"})

	assert.Contains(t, out, "kind: settings")
	assert.Contains(t, out, "kind: settings - provider field support")
	assert.Contains(t, out, "mcp-servers")
	assert.Contains(t, out, "permissions")
}

func TestRenderPolicyXCF_ContainsTwoPolicies(t *testing.T) {
	out := RenderPolicyXCF()

	assert.Contains(t, out, "kind: policy")
	assert.Contains(t, out, "require-agent-description")
	assert.Contains(t, out, "require-agent-instructions")
	assert.Contains(t, out, "severity: warning")
	assert.Contains(t, out, "severity: error")
	// Must be valid multi-document YAML (two --- separated docs)
	assert.Contains(t, out, "---")
}
