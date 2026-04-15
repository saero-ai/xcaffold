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
