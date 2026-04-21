// Package renderer_test contains cross-provider invariant tests.
// These tests compile the same fixture through all supported renderers and assert
// properties that must hold regardless of which provider is targeted.
package renderer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/renderer/antigravity"
	"github.com/saero-ai/xcaffold/internal/renderer/claude"
	"github.com/saero-ai/xcaffold/internal/renderer/copilot"
	"github.com/saero-ai/xcaffold/internal/renderer/cursor"
	"github.com/saero-ai/xcaffold/internal/renderer/gemini"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// allRenderers returns one instance of every supported target renderer.
func allRenderers() []renderer.TargetRenderer {
	return []renderer.TargetRenderer{
		claude.New(),
		cursor.New(),
		gemini.New(),
		copilot.New(),
		antigravity.New(),
	}
}

// crossProviderFixture returns a minimal but non-trivial XcaffoldConfig that
// exercises agents, skills with references, and rules with instructions-file.
// The returned baseDir is a temp directory containing the referenced files.
func crossProviderFixture(t *testing.T) (*ast.XcaffoldConfig, string) {
	t.Helper()
	baseDir := t.TempDir()

	// Create the instructions file for the rule.
	require.NoError(t, os.MkdirAll(filepath.Join(baseDir, "rules"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(baseDir, "rules", "test-rule.md"),
		[]byte("# Rule Content\nFollow this rule."),
		0o644,
	))

	// Create a reference file for the skill.
	require.NoError(t, os.MkdirAll(filepath.Join(baseDir, "refs"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(baseDir, "refs", "doc.md"),
		[]byte("# Reference Doc"),
		0o644,
	))

	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"test-agent": {
					Name:         "test-agent",
					Description:  "A cross-provider test agent",
					Model:        "sonnet-4",
					Instructions: "Agent instructions here.",
				},
			},
			Skills: map[string]ast.SkillConfig{
				"test-skill": {
					Name:         "test-skill",
					Description:  "A skill with references",
					Instructions: "Skill body.",
					References:   []string{"refs/doc.md"},
				},
			},
			Rules: map[string]ast.RuleConfig{
				"test-rule": {
					Description:      "A rule with instructions-file",
					InstructionsFile: "rules/test-rule.md",
					Activation:       ast.RuleActivationAlways,
				},
			},
		},
	}
	return cfg, baseDir
}

// allCodesSet returns the set of all known fidelity codes for O(1) lookup.
func allCodesSet() map[string]bool {
	m := make(map[string]bool)
	for _, c := range renderer.AllCodes() {
		m[c] = true
	}
	return m
}

// TestCrossProvider_RenderOrNote asserts that every agent either produces at
// least one output file or causes a RENDERER_KIND_UNSUPPORTED fidelity note.
// A renderer that silently drops an agent without either is a regression.
func TestCrossProvider_RenderOrNote(t *testing.T) {
	for _, r := range allRenderers() {
		r := r
		t.Run(r.Target(), func(t *testing.T) {
			cfg, baseDir := crossProviderFixture(t)
			out, notes, err := r.Compile(cfg, baseDir)
			require.NoError(t, err)

			if len(out.Files) == 0 {
				// Acceptable only if the renderer emitted RENDERER_KIND_UNSUPPORTED.
				found := false
				for _, n := range notes {
					if n.Code == renderer.CodeRendererKindUnsupported {
						found = true
						break
					}
				}
				assert.True(t, found,
					"renderer %q produced no output files and no RENDERER_KIND_UNSUPPORTED note", r.Target())
			}
		})
	}
}

// TestCrossProvider_NoRawModelAliases asserts that no provider emits a raw
// xcaffold model alias (e.g. "sonnet-4") as a literal model: value in output.
// Aliases must be resolved to provider-specific identifiers before output.
func TestCrossProvider_NoRawModelAliases(t *testing.T) {
	rawAliases := []string{"sonnet-4", "opus-4", "haiku-3.5"}

	for _, r := range allRenderers() {
		r := r
		t.Run(r.Target(), func(t *testing.T) {
			cfg, baseDir := crossProviderFixture(t)
			out, _, err := r.Compile(cfg, baseDir)
			require.NoError(t, err)

			for path, content := range out.Files {
				for _, alias := range rawAliases {
					// Match "model: <alias>" patterns only — avoid false positives in
					// prose or comments that might reference the alias name.
					pattern := "model: " + alias
					assert.NotContains(t, content, pattern,
						"renderer %q emitted raw alias %q in file %q", r.Target(), alias, path)
				}
			}
		})
	}
}

// TestCrossProvider_NoClaudeEnvVars asserts that non-Claude renderers do not
// emit $CLAUDE_PROJECT_DIR or other Claude-specific environment variables in
// their output. These are implementation details of the Claude provider and
// must not leak into other targets.
func TestCrossProvider_NoClaudeEnvVars(t *testing.T) {
	claudeVars := []string{"$CLAUDE_PROJECT_DIR", "$CLAUDE_"}

	for _, r := range allRenderers() {
		r := r
		if r.Target() == "claude" {
			continue // Claude is allowed to reference its own env vars.
		}
		t.Run(r.Target(), func(t *testing.T) {
			cfg, baseDir := crossProviderFixture(t)
			out, _, err := r.Compile(cfg, baseDir)
			require.NoError(t, err)

			for path, content := range out.Files {
				for _, v := range claudeVars {
					assert.NotContains(t, content, v,
						"renderer %q leaked Claude env var %q in file %q", r.Target(), v, path)
				}
			}
		})
	}
}

// TestCrossProvider_SkillReferences asserts that every skill with References
// either produces reference sub-files in the output or emits a
// SKILL_REFERENCES_DROPPED fidelity note. A renderer that silently discards
// references without a note is a regression.
func TestCrossProvider_SkillReferences(t *testing.T) {
	for _, r := range allRenderers() {
		r := r
		t.Run(r.Target(), func(t *testing.T) {
			cfg, baseDir := crossProviderFixture(t)
			out, notes, err := r.Compile(cfg, baseDir)
			require.NoError(t, err)

			// Check whether any output file path looks like a references sub-file.
			hasRefFile := false
			for path := range out.Files {
				if strings.Contains(path, "references/") || strings.Contains(path, "/references") {
					hasRefFile = true
					break
				}
			}

			if hasRefFile {
				return // Provider copied references to output — invariant satisfied.
			}

			// Provider did not produce reference files; it must have emitted a note.
			found := false
			for _, n := range notes {
				if n.Code == renderer.CodeSkillReferencesDropped {
					found = true
					break
				}
			}
			assert.True(t, found,
				"renderer %q has a skill with References but produced neither reference files nor a SKILL_REFERENCES_DROPPED note",
				r.Target())
		})
	}
}

// TestCrossProvider_RuleInstructionsFile asserts that every rule using
// InstructionsFile produces non-empty body content in the compiled output.
// An empty rule body (only frontmatter) indicates the file was not resolved.
func TestCrossProvider_RuleInstructionsFile(t *testing.T) {
	for _, r := range allRenderers() {
		r := r
		t.Run(r.Target(), func(t *testing.T) {
			cfg, baseDir := crossProviderFixture(t)
			out, _, err := r.Compile(cfg, baseDir)
			require.NoError(t, err)

			// Find output files that correspond to the rule (contains "test-rule").
			for path, content := range out.Files {
				if !strings.Contains(path, "test-rule") {
					continue
				}
				// The file must contain the rule body text from rules/test-rule.md.
				// If it's only frontmatter (---\n...\n---\n) the content is empty.
				assert.Contains(t, content, "Rule Content",
					"renderer %q rule file %q has empty body; InstructionsFile was not resolved",
					r.Target(), path)
			}
		})
	}
}

// TestCrossProvider_FidelityCodesValid asserts that every fidelity note code
// emitted by any renderer appears in the AllCodes() catalog. An unregistered
// code indicates a renderer was updated without registering the new code.
func TestCrossProvider_FidelityCodesValid(t *testing.T) {
	known := allCodesSet()

	for _, r := range allRenderers() {
		r := r
		t.Run(r.Target(), func(t *testing.T) {
			cfg, baseDir := crossProviderFixture(t)
			_, notes, err := r.Compile(cfg, baseDir)
			require.NoError(t, err)

			for _, n := range notes {
				assert.True(t, known[n.Code],
					"renderer %q emitted unregistered fidelity code %q", r.Target(), n.Code)
			}
		})
	}
}
