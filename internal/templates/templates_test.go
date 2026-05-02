package templates

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

// --- RenderXaffOverrideXCF format tests ---

func TestRenderXaffOverrideXCF_StartsWithFrontmatter(t *testing.T) {
	out := RenderXaffOverrideXCF("claude")
	if !strings.HasPrefix(out, "---\n") {
		t.Errorf("override must start with --- delimiter, got: %.40s", out)
	}
	if strings.Contains(out, "# agent.claude") {
		t.Error("generated manifest must not contain header comments")
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
