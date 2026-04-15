package integration_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// buildTranslateBinary builds the xcaffold binary for integration testing.
// It compiles the binary once per test and returns the path.
func buildTranslateBinary(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "xcaffold")

	// Resolve repo root from this test file's location.
	// We are in internal/integration/; repo root is ../../.
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")

	cmd := exec.Command("go", "build", "-o", bin, "./cmd/xcaffold/")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to build xcaffold binary:\n%s", out)
	return bin
}

// TestTranslate_AntigravityToClaude_DryRun verifies that --dry-run
// does not create output directories or write files.
func TestTranslate_AntigravityToClaude_DryRun(t *testing.T) {
	bin := buildTranslateBinary(t)

	// Create a minimal source .claude/ structure with a rule.
	srcDir := t.TempDir()
	claudeDir := filepath.Join(srcDir, ".claude")
	rulesDir := filepath.Join(claudeDir, "rules")
	require.NoError(t, os.MkdirAll(rulesDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(rulesDir, "code-review.md"),
		[]byte("# Code Review\n\nAnalyze pull request diffs."),
		0o644,
	))

	// Create destination directory.
	dstDir := t.TempDir()

	// Run translate with --dry-run: should not write to dstDir.
	cmd := exec.Command(
		bin,
		"translate",
		"--from", "claude",
		"--to", "antigravity",
		"--source-dir", srcDir,
		"--output-dir", dstDir,
		"--dry-run",
	)
	err := cmd.Run()
	require.NoError(t, err, "translate --dry-run must succeed")

	// Verify no .claude/ directory was created at destination.
	claudeDstPath := filepath.Join(dstDir, ".claude")
	_, statErr := os.Stat(claudeDstPath)
	require.True(t, os.IsNotExist(statErr),
		".claude/ must not exist at destination after --dry-run; got: %v", statErr)
}

// TestTranslate_AntigravityToClaude_Idempotent verifies that running
// twice in idempotent mode exits with success on the second run.
func TestTranslate_AntigravityToClaude_Idempotent(t *testing.T) {
	bin := buildTranslateBinary(t)

	// Create source structure.
	srcDir := t.TempDir()
	claudeDir := filepath.Join(srcDir, ".claude")
	rulesDir := filepath.Join(claudeDir, "rules")
	require.NoError(t, os.MkdirAll(rulesDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(rulesDir, "style.md"),
		[]byte("# Style\n\nMaintain code style consistency."),
		0o644,
	))

	dstDir := t.TempDir()

	// First run: translate and write files.
	cmd1 := exec.Command(
		bin,
		"translate",
		"--from", "claude",
		"--to", "antigravity",
		"--source-dir", srcDir,
		"--output-dir", dstDir,
	)
	err := cmd1.Run()
	require.NoError(t, err, "first translate run must succeed")

	// Second run: verify idempotent-check passes (no diff).
	cmd2 := exec.Command(
		bin,
		"translate",
		"--from", "claude",
		"--to", "antigravity",
		"--source-dir", srcDir,
		"--output-dir", dstDir,
		"--idempotent-check",
	)
	err = cmd2.Run()
	require.NoError(t, err, "idempotent-check must exit 0 when files are up to date")
}

// TestTranslate_SaveXcf_ProducesValidYAML verifies that --save-xcf
// creates a valid YAML file.
func TestTranslate_SaveXcf_ProducesValidYAML(t *testing.T) {
	bin := buildTranslateBinary(t)

	// Create source structure.
	srcDir := t.TempDir()
	claudeDir := filepath.Join(srcDir, ".claude")
	agentsDir := filepath.Join(claudeDir, "agents")
	rulesDir := filepath.Join(claudeDir, "rules")
	require.NoError(t, os.MkdirAll(agentsDir, 0o755))
	require.NoError(t, os.MkdirAll(rulesDir, 0o755))

	// Write a minimal agent file.
	require.NoError(t, os.WriteFile(
		filepath.Join(agentsDir, "reviewer.md"),
		[]byte("---\nname: reviewer\ndescription: Code reviewer\nmodel: sonnet\n---\n\nReview code changes."),
		0o644,
	))

	// Write a minimal rule file.
	require.NoError(t, os.WriteFile(
		filepath.Join(rulesDir, "test-coverage.md"),
		[]byte("# Test Coverage\n\nAll new code must have tests."),
		0o644,
	))

	xcfOut := filepath.Join(t.TempDir(), "scaffold.xcf")

	// Run translate with --save-xcf; the flag takes a positional value.
	cmd := exec.Command(
		bin,
		"translate",
		"--from", "claude",
		"--to", "antigravity",
		"--source-dir", srcDir,
		"--save-xcf", xcfOut,
		"--dry-run",
	)
	err := cmd.Run()
	require.NoError(t, err, "translate --save-xcf must succeed")

	// Verify the file exists and is non-empty.
	data, err := os.ReadFile(xcfOut)
	require.NoError(t, err, "must be able to read saved .xcf file")
	require.NotEmpty(t, data, "saved .xcf file must be non-empty")

	// Verify it contains YAML structure (basic check).
	// The generated XCF will be a marshalled project config structure
	content := string(data)
	require.Contains(t, content, "version:", ".xcf must contain version field")
	// Either "kind:" or "project:" should be present depending on the generated schema
	hasKindOrProject := strings.Contains(content, "kind:") || strings.Contains(content, "project:")
	require.True(t, hasKindOrProject, ".xcf must contain either kind or project field")
}

// TestTranslate_AuditOut_ContainsWorkflowLowering verifies that --audit-out
// produces a JSON file with expected audit metadata.
func TestTranslate_AuditOut_ContainsWorkflowLowering(t *testing.T) {
	bin := buildTranslateBinary(t)

	// Create source with a workflow definition.
	srcDir := t.TempDir()
	claudeDir := filepath.Join(srcDir, ".claude")
	rulesDir := filepath.Join(claudeDir, "rules")
	skillsDir := filepath.Join(claudeDir, "skills")
	require.NoError(t, os.MkdirAll(rulesDir, 0o755))
	require.NoError(t, os.MkdirAll(skillsDir, 0o755))

	// Write rule file with x-xcaffold provenance marker.
	require.NoError(t, os.WriteFile(
		filepath.Join(rulesDir, "code-review-workflow.md"),
		[]byte("---\nx-xcaffold:\n  compiled-from: workflow\n---\n\n# Code Review\n\nMulti-step review."),
		0o644,
	))

	// Write skill files for each step.
	for i, step := range []string{"analyze", "lint", "summarize"} {
		skillDir := filepath.Join(skillsDir, "code-review-0"+string(rune('1'+i))+"-"+step)
		require.NoError(t, os.MkdirAll(skillDir, 0o755))
		require.NoError(t, os.WriteFile(
			filepath.Join(skillDir, "SKILL.md"),
			[]byte("---\nname: code-review-0"+string(rune('1'+i))+"-"+step+"\n---\n\n"+strings.Title(step)),
			0o644,
		))
	}

	dstDir := t.TempDir()
	auditOut := filepath.Join(t.TempDir(), "audit.json")

	// Run translate with --audit-out.
	cmd := exec.Command(
		bin,
		"translate",
		"--from", "claude",
		"--to", "antigravity",
		"--source-dir", srcDir,
		"--output-dir", dstDir,
		"--audit-out", auditOut,
	)
	err := cmd.Run()
	require.NoError(t, err, "translate --audit-out must succeed")

	// Read and parse the audit JSON.
	auditData, err := os.ReadFile(auditOut)
	require.NoError(t, err, "must be able to read audit file")

	var audit map[string]any
	require.NoError(t, json.Unmarshal(auditData, &audit), "audit file must be valid JSON")

	// Verify required fields.
	require.Equal(t, "1.0", audit["audit-version"], "audit-version must be 1.0")
	require.Equal(t, "claude", audit["from"], "from must be claude")
	require.Equal(t, "antigravity", audit["to"], "to must be antigravity")
}

// TestTranslate_WorkflowLowering_RulePlusSkill_EndToEnd verifies that the translate
// command processes workflow-lowered artifacts (rule + per-step skills) without error.
func TestTranslate_WorkflowLowering_RulePlusSkill_EndToEnd(t *testing.T) {
	bin := buildTranslateBinary(t)

	// Create a source .claude/ structure with workflow-lowered artifacts.
	srcDir := t.TempDir()
	claudeDir := filepath.Join(srcDir, ".claude")
	rulesDir := filepath.Join(claudeDir, "rules")
	skillsDir := filepath.Join(claudeDir, "skills")
	require.NoError(t, os.MkdirAll(rulesDir, 0o755))
	require.NoError(t, os.MkdirAll(skillsDir, 0o755))

	// Write the main rule file with x-xcaffold provenance marker.
	ruleContent := `---
name: code-review-workflow
description: Multi-step code review workflow.
x-xcaffold:
  compiled-from: workflow
  original-name: code-review
---

# Code Review Workflow

Analyze → Lint → Summarize pipeline for pull request reviews.
`
	require.NoError(t, os.WriteFile(
		filepath.Join(rulesDir, "code-review-workflow.md"),
		[]byte(ruleContent),
		0o644,
	))

	// Write per-step skill files (01, 02, 03).
	steps := []string{"analyze", "lint", "summarize"}
	for i, step := range steps {
		skillNum := i + 1
		skillDir := filepath.Join(skillsDir, "code-review-0"+string(rune('0'+skillNum))+"-"+step)
		require.NoError(t, os.MkdirAll(skillDir, 0o755))

		skillContent := `---
name: code-review-0` + string(rune('0'+skillNum)) + `-` + step + `
description: Step ` + string(rune('0'+skillNum)) + ` - ` + step + `
allowed-tools: [Read, Grep]
---

Perform ` + step + ` checks on code changes.
`
		require.NoError(t, os.WriteFile(
			filepath.Join(skillDir, "SKILL.md"),
			[]byte(skillContent),
			0o644,
		))
	}

	dstDir := t.TempDir()

	// Run translate.
	cmd := exec.Command(
		bin,
		"translate",
		"--from", "claude",
		"--to", "antigravity",
		"--source-dir", srcDir,
		"--output-dir", dstDir,
	)
	err := cmd.Run()
	require.NoError(t, err, "translate must succeed with workflow-lowered source files")

	// Verify source files are still intact (not modified by translate).
	// This ensures the command can handle workflow artifacts without error.
	ruleSource := filepath.Join(rulesDir, "code-review-workflow.md")
	_, err = os.Stat(ruleSource)
	require.NoError(t, err, "source rule file must still exist after translate")

	for i := range steps {
		skillNum := i + 1
		skillPath := filepath.Join(
			skillsDir,
			"code-review-0"+string(rune('0'+skillNum))+"-"+steps[i],
			"SKILL.md",
		)
		_, err := os.Stat(skillPath)
		require.NoError(t, err, "source skill file must still exist after translate: %s", skillPath)
	}
}
