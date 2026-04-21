package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/renderer/claude"
)

const skillFixtureDir = "/Volumes/Projects/xcaffold-project/xcaffold-test/.claude/skills"

func TestSkillSchema_RealData_RoundTrip_CanonicalOrder(t *testing.T) {
	if _, err := os.Stat(skillFixtureDir); os.IsNotExist(err) {
		t.Skip("skill fixtures not available — skipping")
	}

	entries, err := os.ReadDir(skillFixtureDir)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(entries), 3, "need at least 3 fixtures for meaningful round-trip")

	r := claude.New()
	tested := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillPath := filepath.Join(skillFixtureDir, e.Name(), "SKILL.md")
		raw, err := os.ReadFile(skillPath)
		if err != nil {
			continue
		}

		// Build a synthetic SkillConfig from the fixture's frontmatter (minimal)
		// — full bidirectional parse of SKILL.md is out of scope; we only need
		// to confirm the renderer handles real-world body content.
		skill := ast.SkillConfig{
			Name:         e.Name(),
			Description:  extractField(string(raw), "description"),
			Instructions: string(raw),
		}

		config := &ast.XcaffoldConfig{
			ResourceScope: ast.ResourceScope{
				Skills: map[string]ast.SkillConfig{e.Name(): skill},
			},
		}
		out, _, err := renderer.Orchestrate(r, config, "")
		require.NoError(t, err, "compile failed for fixture %s", e.Name())

		md := out.Files["skills/"+e.Name()+"/SKILL.md"]
		require.NotEmpty(t, md, "no output for fixture %s", e.Name())

		// Canonical field order: name appears before description in frontmatter
		idxName := strings.Index(md, "name:")
		idxDesc := strings.Index(md, "description:")
		if idxName != -1 && idxDesc != -1 {
			require.True(t, idxName < idxDesc, "name must appear before description in fixture %s", e.Name())
		}

		tested++
		if tested >= 5 {
			break
		}
	}
	require.GreaterOrEqual(t, tested, 3, "must validate at least 3 fixtures")
}

func TestSkillSchema_RealData_ProviderIsolation(t *testing.T) {
	r := claude.New()
	skill := ast.SkillConfig{
		Name:        "iso",
		Description: "isolation check",
		Targets: map[string]ast.TargetOverride{
			"cursor": {Provider: map[string]any{"compatibility": "cursor >= 2.4"}},
			"claude": {Provider: map[string]any{"context": "fork", "agent": "Explore"}},
		},
		Instructions: "body",
	}
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{"iso": skill},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)
	md := out.Files["skills/iso/SKILL.md"]
	require.Contains(t, md, "context: fork")
	require.Contains(t, md, "agent: Explore")
	require.NotContains(t, md, "compatibility")
	require.NotContains(t, md, "cursor >= 2.4")
}

// extractField pulls a top-level YAML scalar from a SKILL.md frontmatter block.
// Returns "" if not found. This is intentionally minimal — a full YAML parse
// would require importing yaml.v3, which the integration package keeps optional.
func extractField(src, key string) string {
	prefix := key + ": "
	for _, line := range strings.Split(src, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), prefix))
		}
	}
	return ""
}
