package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/bir"
	"github.com/saero-ai/xcaffold/internal/compiler"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/stretchr/testify/require"
)

// threeStepCodeReview returns a WorkflowConfig with inline step instructions
// so tests run without relying on external file resolution.
func threeStepCodeReview() *ast.XcaffoldConfig {
	return &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Workflows: map[string]ast.WorkflowConfig{
				"code-review": {
					ApiVersion:  "workflow/v1",
					Name:        "code-review",
					Description: "Multi-step pull request review procedure.",
					Steps: []ast.WorkflowStep{
						{Name: "analyze", Instructions: "Read the diff and summarize changed modules."},
						{Name: "lint", Instructions: "Check style violations in changed files."},
						{Name: "summarize", Instructions: "Write the final review comment."},
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
	out, notes, err := compiler.Compile(config, t.TempDir(), "claude", "")
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
// emits a native workflow file and does not create a rules/ lowering artifact.
// The promote-rules-to-workflows target override triggers the native path and
// causes the CodeWorkflowLoweredToNative fidelity note to be emitted.
func TestCompile_Workflow_AntigravityNative(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Workflows: map[string]ast.WorkflowConfig{
				"code-review": {
					ApiVersion:  "workflow/v1",
					Name:        "code-review",
					Description: "Multi-step pull request review procedure.",
					Steps: []ast.WorkflowStep{
						{Name: "analyze", Instructions: "Read the diff and summarize changed modules."},
						{Name: "lint", Instructions: "Check style violations in changed files."},
						{Name: "summarize", Instructions: "Write the final review comment."},
					},
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
	out, notes, err := compiler.Compile(config, t.TempDir(), "antigravity", "")
	require.NoError(t, err)

	// Native workflow file must exist.
	wfContent, ok := out.Files["workflows/code-review.md"]
	require.True(t, ok, "expected workflows/code-review.md in output; got keys: %v", fileKeys(out.Files))
	require.NotEmpty(t, wfContent)

	// No rules lowering artifact should exist.
	_, hasRule := out.Files["rules/code-review.md"]
	require.False(t, hasRule, "antigravity target must not emit a rules/ lowering artifact")

	// At least one fidelity note with LevelInfo and CodeWorkflowLoweredToNative.
	var found bool
	for _, n := range notes {
		if n.Code == renderer.CodeWorkflowLoweredToNative && n.Level == renderer.LevelInfo {
			found = true
			break
		}
	}
	require.True(t, found,
		"expected a LevelInfo note with CodeWorkflowLoweredToNative; got notes: %v", notes)
}

// TestRoundTrip_Workflow_ProvenanceReassembly verifies that a compiled workflow
// can be reconstructed from the provenance marker embedded in the rule file.
func TestRoundTrip_Workflow_ProvenanceReassembly(t *testing.T) {
	tmp := t.TempDir()
	config := threeStepCodeReview()

	// Step 1: Compile.
	out, _, err := compiler.Compile(config, tmp, "claude", "")
	require.NoError(t, err)

	// Step 2: Write output to disk under .claude/ (mimicking xcaffold apply).
	for path, content := range out.Files {
		full := filepath.Join(tmp, ".claude", path)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o600))
	}

	// Step 3: Reassemble from provenance marker.
	wf, _, err := bir.ReassembleWorkflow(tmp, "code-review")
	require.NoError(t, err)
	require.NotNil(t, wf, "round-trip must produce a workflow")

	require.Equal(t, "workflow/v1", wf.ApiVersion)
	require.Len(t, wf.Steps, 3)
	require.Equal(t, "analyze", wf.Steps[0].Name)
	require.Equal(t, "lint", wf.Steps[1].Name)
	require.Equal(t, "summarize", wf.Steps[2].Name)
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

// fileKeys returns the sorted list of file keys for error output.
func fileKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
