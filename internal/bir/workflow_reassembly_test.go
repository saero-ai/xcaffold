package bir_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/bir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeWorkflowFixtures creates a .claude/rules/<name>-workflow.md and matching
// .claude/skills/<id>/SKILL.md files under a temp directory to simulate the output
// of translator.TranslateWorkflow / claude renderer.
func writeWorkflowFixtures(t *testing.T, dir, workflowName string, steps []struct{ name, skillID, body string }) {
	t.Helper()

	// Build the provenance rule content (mirrors translator.TranslateWorkflow format).
	ruleBody := "```yaml\nx-xcaffold:\n"
	ruleBody += "  compiled-from: workflow\n"
	ruleBody += "  workflow-name: " + workflowName + "\n"
	ruleBody += "  api-version: workflow/v1\n"
	ruleBody += "  step-order: ["
	for i, s := range steps {
		if i > 0 {
			ruleBody += ", "
		}
		ruleBody += s.name
	}
	ruleBody += "]\n"
	ruleBody += "  step-skills:\n"
	for _, s := range steps {
		ruleBody += "    - " + s.skillID + "\n"
	}
	ruleBody += "```\n\nRun steps in order.\n"

	rulesDir := filepath.Join(dir, ".claude", "rules")
	require.NoError(t, os.MkdirAll(rulesDir, 0o755))
	ruleFile := filepath.Join(rulesDir, workflowName+"-workflow.md")
	require.NoError(t, os.WriteFile(ruleFile, []byte(ruleBody), 0o600))

	// Write each skill file.
	for _, s := range steps {
		skillDir := filepath.Join(dir, ".claude", "skills", s.skillID)
		require.NoError(t, os.MkdirAll(skillDir, 0o755))
		skillContent := "---\nname: " + s.skillID + "\ndescription: test skill\n---\n\n" + s.body + "\n"
		require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o600))
	}
}

func TestReassembleWorkflow_ThreeSteps_ProducesWorkflowConfig(t *testing.T) {
	dir := t.TempDir()
	steps := []struct{ name, skillID, body string }{
		{"analyze", "code-review-01-analyze", "Analyze the code for issues."},
		{"lint", "code-review-02-lint", "Run lint checks."},
		{"summarize", "code-review-03-summarize", "Summarize findings."},
	}
	writeWorkflowFixtures(t, dir, "code-review", steps)

	wf, notes, err := bir.ReassembleWorkflow(dir, "code-review")
	require.NoError(t, err)
	require.NotNil(t, wf)

	assert.Equal(t, "code-review", wf.Name)
	assert.Equal(t, "workflow/v1", wf.ApiVersion)
	require.Len(t, wf.Steps, 3)
	assert.Equal(t, "analyze", wf.Steps[0].Name)
	assert.Equal(t, "lint", wf.Steps[1].Name)
	assert.Equal(t, "summarize", wf.Steps[2].Name)
	assert.Equal(t, "Analyze the code for issues.", wf.Steps[0].Body)
	assert.Equal(t, "Run lint checks.", wf.Steps[1].Body)
	assert.Equal(t, "Summarize findings.", wf.Steps[2].Body)

	// Should produce at least one info note.
	require.NotEmpty(t, notes)
	assert.Equal(t, "info", string(notes[0].Level))
}

func TestReassembleWorkflow_MissingSkill_FallsBackGracefully(t *testing.T) {
	dir := t.TempDir()
	// Write rule file but omit one of the skill files.
	steps := []struct{ name, skillID, body string }{
		{"analyze", "code-review-01-analyze", "Analyze."},
		{"lint", "code-review-02-lint", "Lint."},
	}
	writeWorkflowFixtures(t, dir, "code-review", steps)

	// Delete one skill file to simulate missing.
	missing := filepath.Join(dir, ".claude", "skills", "code-review-02-lint", "SKILL.md")
	require.NoError(t, os.Remove(missing))

	wf, notes, err := bir.ReassembleWorkflow(dir, "code-review")
	require.NoError(t, err)
	assert.Nil(t, wf)
	assert.Empty(t, notes)
}

func TestReassembleWorkflow_NoMarker_ReturnsNil(t *testing.T) {
	dir := t.TempDir()

	// Write a plain rule file with no x-xcaffold marker.
	rulesDir := filepath.Join(dir, ".claude", "rules")
	require.NoError(t, os.MkdirAll(rulesDir, 0o755))
	plainRule := "---\ndescription: plain rule\n---\n\nJust a regular rule with no provenance.\n"
	require.NoError(t, os.WriteFile(filepath.Join(rulesDir, "plain-rule-workflow.md"), []byte(plainRule), 0o600))

	wf, notes, err := bir.ReassembleWorkflow(dir, "plain-rule")
	require.NoError(t, err)
	assert.Nil(t, wf)
	assert.Empty(t, notes)
}

func TestReassembleWorkflow_Idempotent(t *testing.T) {
	dir := t.TempDir()
	steps := []struct{ name, skillID, body string }{
		{"analyze", "idempotent-01-analyze", "Analyze."},
		{"report", "idempotent-02-report", "Report."},
	}
	writeWorkflowFixtures(t, dir, "idempotent", steps)

	wf1, notes1, err1 := bir.ReassembleWorkflow(dir, "idempotent")
	require.NoError(t, err1)
	require.NotNil(t, wf1)

	wf2, notes2, err2 := bir.ReassembleWorkflow(dir, "idempotent")
	require.NoError(t, err2)
	require.NotNil(t, wf2)

	assert.Equal(t, wf1.Name, wf2.Name)
	assert.Equal(t, wf1.ApiVersion, wf2.ApiVersion)
	require.Equal(t, len(wf1.Steps), len(wf2.Steps))
	for i := range wf1.Steps {
		assert.Equal(t, wf1.Steps[i].Name, wf2.Steps[i].Name)
		assert.Equal(t, wf1.Steps[i].Body, wf2.Steps[i].Body)
	}
	assert.Equal(t, len(notes1), len(notes2))
}
