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
	assert.NotContains(t, out, "policies:")
	assert.NotContains(t, out, "- require-agent-description")
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
	assert.Contains(t, out, "tools: [Read, Write, Edit, Bash, Glob, Grep]")
	assert.Contains(t, out, "skills: [xcaffold]")
	assert.Contains(t, out, "rules: [xcf-conventions]")
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
	assert.Contains(t, out, "kebab-case")
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
	assert.Contains(t, out, "effort: \"high\"")
	assert.Contains(t, out, "tools: [Read, Write, Edit, Bash, Glob, Grep]")
}

func TestRenderAgentXCF_FrontmatterFormat(t *testing.T) {
	out := RenderAgentXCF("developer", "claude-sonnet-4-6", []string{"claude"})

	assert.Contains(t, out, "---\nkind: agent")
	assert.Contains(t, out, "---\nYou are a software developer.")
	assert.NotContains(t, out, "instructions: |")
}

func TestRenderAgentXCF_SingleTarget_NoCursorColumn(t *testing.T) {
	out := RenderAgentXCF("developer", "claude-sonnet-4-6", []string{"claude"})

	assert.Contains(t, out, "---\nkind: agent")
	assert.NotContains(t, out, "cursor")
}

// --- RenderSettingsXCF ---

func TestRenderSettingsXCF_ContainsMatrix(t *testing.T) {
	out := RenderSettingsXCF([]string{"claude"})

	assert.Contains(t, out, "kind: settings")
	assert.Contains(t, out, "version: \"1.0\"")
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

// --- Tests for comment removal ---

func TestRenderXaffAgentXCF_StartsWithFrontmatter(t *testing.T) {
	out := RenderXaffAgentXCF("claude-sonnet-4-6", []string{"claude"})
	if !strings.HasPrefix(out, "---\n") {
		t.Errorf("agent.xcf must start with --- delimiter, got: %.40s", out)
	}
	if strings.Contains(out, "# kind:") {
		t.Error("generated manifest must not contain comment lines starting with '# kind:'")
	}
	if strings.Contains(out, "# model:") {
		t.Error("generated manifest must not contain inline field comments")
	}
}

func TestRenderXaffOverrideXCF_StartsWithFrontmatter(t *testing.T) {
	out := RenderXaffOverrideXCF("claude")
	if !strings.HasPrefix(out, "---\n") {
		t.Errorf("override must start with --- delimiter, got: %.40s", out)
	}
	if strings.Contains(out, "# agent.claude") {
		t.Error("generated manifest must not contain header comments")
	}
}

func TestRenderXcfConventionsRuleXCF_StartsWithFrontmatter(t *testing.T) {
	out := RenderXcfConventionsRuleXCF([]string{"claude"})
	if !strings.HasPrefix(out, "---\n") {
		t.Errorf("rule must start with --- delimiter, got: %.40s", out)
	}
	if strings.Contains(out, "# kind: rule") {
		t.Error("generated manifest must not contain comment lines")
	}
}

func TestRenderProjectXCF_StartsWithKind(t *testing.T) {
	out := RenderProjectXCF("my-project", []string{"claude"})
	if !strings.HasPrefix(out, "kind: project") {
		t.Errorf("project.xcf must start with 'kind: project', got: %.40s", out)
	}
	if strings.Contains(out, "# project.xcf") {
		t.Error("generated project.xcf must not contain header comments")
	}
	if strings.Contains(out, "# Compile") {
		t.Error("generated project.xcf must not contain section comments")
	}
	if strings.Contains(out, "# test:") {
		t.Error("generated project.xcf must not contain commented-out config")
	}
}

func TestRenderSettingsXCF_Minimal(t *testing.T) {
	out := RenderSettingsXCF([]string{"claude"})
	if strings.Contains(out, "# MCP") {
		t.Error("generated settings.xcf must not contain commented examples")
	}
	if !strings.Contains(out, "kind: settings") {
		t.Error("must contain kind: settings")
	}
}

func TestRenderAgentXCF_StartsWithFrontmatter(t *testing.T) {
	out := RenderAgentXCF("dev", "sonnet", []string{"claude"})
	if !strings.HasPrefix(out, "---\n") {
		t.Errorf("must start with --- delimiter, got: %.40s", out)
	}
}

// --- RenderOperatingGuide ---

func TestRenderOperatingGuide_Content(t *testing.T) {
	out := RenderOperatingGuide()

	for _, cmd := range []string{"xcaffold init", "xcaffold apply", "xcaffold validate", "xcaffold status", "xcaffold import"} {
		assert.Contains(t, out, cmd, "operating guide must contain %q", cmd)
	}

	assert.Contains(t, out, "## Starting a new project")
	assert.Contains(t, out, "## Checking compilation state")
	assert.Contains(t, out, "## Importing existing provider config")
}

func TestRenderOperatingGuide_ContainsFlags(t *testing.T) {
	out := RenderOperatingGuide()

	flags := []string{"--target", "--yes", "--dry-run", "--plan"}
	for _, flag := range flags {
		assert.Contains(t, out, flag, "operating guide must document %q flag", flag)
	}
}

// --- RenderAuthoringGuide ---

func TestRenderAuthoringGuide_Content(t *testing.T) {
	out := RenderAuthoringGuide()

	for _, kind := range []string{"kind: agent", "kind: skill", "kind: rule", "kind: workflow", "kind: mcp", "kind: hooks", "kind: memory"} {
		assert.Contains(t, out, kind, "authoring guide must contain %q", kind)
	}

	assert.Contains(t, out, "xcf/agents/", "authoring guide must show directory structure")
	assert.Contains(t, out, "xcf/skills/", "authoring guide must show directory structure")
}

func TestRenderAuthoringGuide_ContainsFieldMatrix(t *testing.T) {
	out := RenderAuthoringGuide()

	assert.Contains(t, out, "description", "authoring guide must document field names")
	assert.Contains(t, out, "allowed-tools", "authoring guide must document allowed-tools field")
}

// --- Xcaffold Skill Body Size ---

func TestRenderXcaffoldSkillXCF_SlimBody(t *testing.T) {
	out := RenderXcaffoldSkillXCF([]string{"claude"})

	parts := strings.SplitN(out, "---\n", 3)
	require.Len(t, parts, 3, "output must have exactly two --- delimiters")
	body := parts[2]
	bodyLines := len(strings.Split(strings.TrimSpace(body), "\n"))

	assert.LessOrEqual(t, bodyLines, 60, "skill body should be slim (<60 lines), got %d", bodyLines)
}

func TestRenderXcaffoldSkillXCF_ReferencesIncludeGuides(t *testing.T) {
	out := RenderXcaffoldSkillXCF([]string{"claude"})

	assert.Contains(t, out, "xcf/skills/xcaffold/references/operating-guide.md", "references must include operating-guide.md")
	assert.Contains(t, out, "xcf/skills/xcaffold/references/authoring-guide.md", "references must include authoring-guide.md")
}
