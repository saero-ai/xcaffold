package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/compiler"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/stretchr/testify/require"
)

// threeStepCodeReview returns a WorkflowConfig with inline step instructions
// and an explicit rule-plus-skill lowering strategy for Claude so tests run
// without relying on external file resolution or default-behavior assumptions.
func threeStepCodeReview() *ast.XcaffoldConfig {
	return &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Workflows: map[string]ast.WorkflowConfig{
				"code-review": {

					Name:        "code-review",
					Description: "Multi-step pull request review procedure.",
					Steps: []ast.WorkflowStep{
						{Name: "analyze", Instructions: "Read the diff and summarize changed modules."},
						{Name: "lint", Instructions: "Check style violations in changed files."},
						{Name: "summarize", Instructions: "Write the final review comment."},
					},
					Targets: map[string]ast.TargetOverride{
						"claude": {Provider: map[string]any{"lowering-strategy": "rule-plus-skill"}},
					},
				},
			},
		},
	}
}

// TestCompile_Workflow_RulePlusSkillLowering verifies that the Claude renderer
// lowers a multi-step workflow to a rule + per-step skills and emits the
// correct fidelity note.
func TestCompile_Workflow_RulePlusSkillLowering(t *testing.T) {
	config := threeStepCodeReview()
	out, notes, err := compiler.Compile(config, t.TempDir(), compiler.CompileOpts{Target: "claude"})
	require.NoError(t, err)

	// Rule file must exist at the canonical lowered path.
	ruleContent, ok := out.Files["rules/code-review-workflow.md"]
	require.True(t, ok, "expected rules/code-review-workflow.md in output; got keys: %v", fileKeys(out.Files))
	require.NotEmpty(t, ruleContent)

	// Rule body must contain x-xcaffold: provenance marker.
	require.Contains(t, ruleContent, "x-xcaffold:")
	require.Contains(t, ruleContent, "compiled-from: workflow")

	// Per-step skill files must exist.
	require.Contains(t, out.Files, "skills/code-review-01-analyze/SKILL.md",
		"step skill 01-analyze missing; got keys: %v", fileKeys(out.Files))
	require.Contains(t, out.Files, "skills/code-review-02-lint/SKILL.md",
		"step skill 02-lint missing; got keys: %v", fileKeys(out.Files))
	require.Contains(t, out.Files, "skills/code-review-03-summarize/SKILL.md",
		"step skill 03-summarize missing; got keys: %v", fileKeys(out.Files))

	// At least one fidelity note must carry the lowering code at Warning level.
	var found bool
	for _, n := range notes {
		if n.Code == renderer.CodeWorkflowLoweredToRulePlusSkill && n.Level == renderer.LevelWarning {
			found = true
			break
		}
	}
	require.True(t, found,
		"expected a LevelWarning note with CodeWorkflowLoweredToRulePlusSkill; got notes: %v", notes)
}

// TestCompile_Workflow_AntigravityNative verifies that the Antigravity renderer
// always emits a native workflow file regardless of target overrides, and does
// not create a rules/ lowering artifact. promote-rules-to-workflows is a no-op
// because Antigravity renders natively on every path.
func TestCompile_Workflow_AntigravityNative(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Workflows: map[string]ast.WorkflowConfig{
				"code-review": {

					Name:        "code-review",
					Description: "Multi-step pull request review procedure.",
					Steps: []ast.WorkflowStep{
						{Name: "analyze", Instructions: "Read the diff and summarize changed modules."},
						{Name: "lint", Instructions: "Check style violations in changed files."},
						{Name: "summarize", Instructions: "Write the final review comment."},
					},
					// promote-rules-to-workflows is a no-op; kept here for
					// backward-compat verification that old configs still compile.
					Targets: map[string]ast.TargetOverride{
						"antigravity": {
							Provider: map[string]any{
								"promote-rules-to-workflows": true,
							},
						},
					},
				},
			},
		},
	}
	out, _, err := compiler.Compile(config, t.TempDir(), compiler.CompileOpts{Target: "antigravity"})
	require.NoError(t, err)

	// Native workflow file must exist.
	wfContent, ok := out.Files["workflows/code-review.md"]
	require.True(t, ok, "expected workflows/code-review.md in output; got keys: %v", fileKeys(out.Files))
	require.NotEmpty(t, wfContent)

	// Workflow must use step-based rendering (## headings for each step).
	require.Contains(t, wfContent, "## analyze")
	require.Contains(t, wfContent, "## lint")
	require.Contains(t, wfContent, "## summarize")

	// No rules lowering artifact should exist.
	_, hasRule := out.Files["rules/code-review.md"]
	require.False(t, hasRule, "antigravity target must not emit a rules/ lowering artifact")
}

// TestRealData_Workflow_Fixtures verifies the sidecar markdown files used as
// workflow step bodies exist and are non-empty.
func TestRealData_Workflow_Fixtures(t *testing.T) {
	// Resolve relative to this package (internal/integration/) — go test sets
	// cwd to the package directory.
	fixtureDir := filepath.Join("..", "..", "testing", "workflows", "code-review")

	files := []string{
		"01-analyze.md",
		"02-lint.md",
		"03-summarize.md",
	}

	for _, name := range files {
		path := filepath.Join(fixtureDir, name)
		data, err := os.ReadFile(path)
		require.NoError(t, err, "fixture file must exist: %s", path)
		require.NotEmpty(t, data, "fixture file must be non-empty: %s", path)
	}
}

// TestCompile_Workflow_DefaultSimpleMode verifies that a workflow without
// an explicit lowering-strategy uses the new default behavior: structure-based
// inference produces a single skill file (simple mode) rather than a rule.
// This test exercises the new default-behavior path and verifies the migration
// fidelity notes are emitted.
func TestCompile_Workflow_DefaultSimpleMode(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Workflows: map[string]ast.WorkflowConfig{
				"code-review": {

					Name:        "code-review",
					Description: "Multi-step pull request review procedure.",
					Steps: []ast.WorkflowStep{
						{Name: "analyze", Instructions: "Read the diff."},
						{Name: "lint", Instructions: "Check style."},
						{Name: "summarize", Instructions: "Write the review."},
					},
					// No Targets, no lowering-strategy — new default applies.
				},
			},
		},
	}
	out, notes, err := compiler.Compile(config, t.TempDir(), compiler.CompileOpts{Target: "claude"})
	require.NoError(t, err)

	// New default: single skill file (NOT per-step micro-skills).
	_, hasSkill := out.Files["skills/code-review/SKILL.md"]
	require.True(t, hasSkill, "expected skills/code-review/SKILL.md for simple mode; got keys: %v", fileKeys(out.Files))

	// No rule file should exist (no always-apply or paths set).
	_, hasRule := out.Files["rules/code-review-workflow.md"]
	require.False(t, hasRule, "simple mode without always-apply should NOT emit a rule; got keys: %v", fileKeys(out.Files))

	// Should have CodeWorkflowBasicToSections note.
	var hasSimpleNote bool
	for _, n := range notes {
		if n.Code == renderer.CodeWorkflowBasicToSections {
			hasSimpleNote = true
		}
	}
	require.True(t, hasSimpleNote, "expected CodeWorkflowBasicToSections note; got: %v", notes)

	// Should have CodeWorkflowDefaultChanged migration warning.
	var hasMigrationNote bool
	for _, n := range notes {
		if n.Code == renderer.CodeWorkflowDefaultChanged {
			hasMigrationNote = true
		}
	}
	require.True(t, hasMigrationNote, "expected CodeWorkflowDefaultChanged migration note; got: %v", notes)
}

// fileKeys returns the sorted list of file keys for error output.
func fileKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
