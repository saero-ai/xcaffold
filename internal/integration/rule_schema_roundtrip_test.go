package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/renderer/claude"
	"github.com/saero-ai/xcaffold/internal/renderer/cursor"
	"github.com/stretchr/testify/require"
)

const ruleFixtureBase = "testdata/rule-schema"

func TestRuleSchema_RoundTrip_ClaudeToCursor(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	xcfPath := filepath.Join(ruleFixtureBase, "input", "path-glob-rule.xcf")
	config, err := parser.ParseFile(xcfPath)
	require.NoError(t, err)

	cr := cursor.New()
	out, _, err := renderer.Orchestrate(cr, config, filepath.Dir(xcfPath))
	require.NoError(t, err)

	ruleKey := "rules/path-glob-rule.mdc"
	content, ok := out.Files[ruleKey]
	require.True(t, ok, "expected output file %s not found", ruleKey)
	require.Contains(t, content, "globs:")
	require.NotContains(t, content, "alwaysApply:")
}

func TestRuleSchema_LegacyAlwaysApply_NormalizedToActivation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	legacyPath := filepath.Join(ruleFixtureBase, "input", "legacy-always-apply-true.xcf")
	activationPath := filepath.Join(ruleFixtureBase, "input", "always-rule.xcf")

	legacyConfig, err := parser.ParseFile(legacyPath)
	require.NoError(t, err)

	activationConfig, err := parser.ParseFile(activationPath)
	require.NoError(t, err)

	cr := claude.New()

	legacyOut, _, err := renderer.Orchestrate(cr, legacyConfig, filepath.Dir(legacyPath))
	require.NoError(t, err)

	activationOut, _, err := renderer.Orchestrate(cr, activationConfig, filepath.Dir(activationPath))
	require.NoError(t, err)

	for key, legacyContent := range legacyOut.Files {
		activationContent, ok := activationOut.Files[key]
		if !ok {
			continue
		}
		require.Equal(t, activationContent, legacyContent,
			"legacy always-apply output must match explicit activation: always output for %s", key)
	}
}

var realDataRulePath = os.Getenv("XCAFFOLD_TEST_FIXTURES")

func init() {
	if realDataRulePath == "" {
		realDataRulePath = filepath.Join(os.Getenv("HOME"), ".xcaffold", "test-fixtures")
	}
}

func TestRealData_Rules_Fixture_Exists(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	claudeRules := filepath.Join(realDataRulePath, ".claude", "rules")
	if _, err := os.Stat(claudeRules); os.IsNotExist(err) {
		t.Skipf("fixture %s not present; skipping real-data rule tests", claudeRules)
	}
	entries, err := os.ReadDir(claudeRules)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(entries), 5, "expected at least 5 real rule fixtures")
}

func TestRealData_AllClaudeRules_ParseAndCompile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	claudeRules := filepath.Join(realDataRulePath, ".claude", "rules")
	if _, err := os.Stat(claudeRules); os.IsNotExist(err) {
		t.Skipf("fixture %s not present; skipping", claudeRules)
	}

	entries, err := os.ReadDir(claudeRules)
	require.NoError(t, err)

	cr := claude.New()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".md")
		ruleFile := filepath.Join(claudeRules, entry.Name())

		t.Run(id, func(t *testing.T) {
			// instructions-file must be a relative path; copy the fixture into a temp dir.
			tmp := t.TempDir()
			fixtureData, err := os.ReadFile(ruleFile)
			require.NoError(t, err, "read fixture for %s", id)
			localName := entry.Name()
			require.NoError(t, os.WriteFile(filepath.Join(tmp, localName), fixtureData, 0o600))

			src := `kind: rule
version: "1.0"
name: ` + id + `
activation: always
instructions-file: "` + localName + `"
`
			xcfPath := filepath.Join(tmp, "rule.xcf")
			require.NoError(t, os.WriteFile(xcfPath, []byte(src), 0o600))

			config, err := parser.ParseFile(xcfPath)
			require.NoError(t, err, "parse failed for %s", id)

			out, _, err := renderer.Orchestrate(cr, config, tmp)
			require.NoError(t, err, "compile failed for %s", id)
			require.NotEmpty(t, out.Files["rules/"+id+".md"], "no output for %s", id)
		})
	}
}
