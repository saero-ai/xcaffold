package translator_test

import (
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/translator"
	"github.com/stretchr/testify/require"
)

func threeStepWorkflow() *ast.WorkflowConfig {
	return &ast.WorkflowConfig{
		ApiVersion:  "workflow/v1",
		Name:        "code-review",
		Description: "Multi-step PR review procedure.",
		Steps: []ast.WorkflowStep{
			{Name: "analyze", Instructions: "Read the diff."},
			{Name: "lint", Instructions: "Check style."},
			{Name: "summarize", Instructions: "Write the review comment."},
		},
	}
}

func TestTranslateWorkflow_Claude_RulePlusSkill_EmitsRule(t *testing.T) {
	primitives, notes := translator.TranslateWorkflow(threeStepWorkflow(), "claude")

	var rulePrimitive *translator.TargetPrimitive
	for i := range primitives {
		if primitives[i].Kind == "rule" {
			rulePrimitive = &primitives[i]
			break
		}
	}
	require.NotNil(t, rulePrimitive, "expected a rule primitive for Claude Tier 2")
	require.Contains(t, rulePrimitive.Content, "x-xcaffold:")
	require.Contains(t, rulePrimitive.Content, "compiled-from: workflow")
	require.Contains(t, rulePrimitive.Content, "workflow-name: code-review")
	require.Contains(t, rulePrimitive.Content, "step-order:")

	require.Len(t, notes, 1)
	require.Equal(t, renderer.LevelWarning, notes[0].Level)
	require.Equal(t, renderer.CodeWorkflowLoweredToRulePlusSkill, notes[0].Code)
}

func TestTranslateWorkflow_Claude_RulePlusSkill_EmitsSkillsPerStep(t *testing.T) {
	primitives, _ := translator.TranslateWorkflow(threeStepWorkflow(), "claude")

	var skills []translator.TargetPrimitive
	for _, p := range primitives {
		if p.Kind == "skill" {
			skills = append(skills, p)
		}
	}
	require.Len(t, skills, 3)

	// Verify naming: code-review-01-analyze, code-review-02-lint, code-review-03-summarize
	require.Contains(t, skills[0].ID, "code-review-01-analyze")
	require.Contains(t, skills[1].ID, "code-review-02-lint")
	require.Contains(t, skills[2].ID, "code-review-03-summarize")
}

func TestTranslateWorkflow_Antigravity_NativeEmit(t *testing.T) {
	wf := threeStepWorkflow()
	wf.Targets = map[string]ast.TargetOverride{
		"antigravity": {Provider: map[string]any{"promote-rules-to-workflows": true}},
	}

	primitives, notes := translator.TranslateWorkflow(wf, "antigravity")

	require.Len(t, primitives, 1)
	require.Equal(t, "workflow", primitives[0].Kind)
	require.Contains(t, primitives[0].Content, "## analyze")
	require.Contains(t, primitives[0].Content, "## lint")

	require.Len(t, notes, 1)
	require.Equal(t, renderer.LevelInfo, notes[0].Level)
	require.Equal(t, renderer.CodeWorkflowLoweredToNative, notes[0].Code)
}

func TestTranslateWorkflow_Copilot_PromptFile(t *testing.T) {
	wf := threeStepWorkflow()
	wf.Targets = map[string]ast.TargetOverride{
		"copilot": {Provider: map[string]any{"lowering-strategy": "prompt-file"}},
	}

	primitives, notes := translator.TranslateWorkflow(wf, "copilot")

	require.Len(t, primitives, 1)
	require.Equal(t, "prompt-file", primitives[0].Kind)
	require.True(t, strings.HasSuffix(primitives[0].Path, ".prompt.md"))
	require.Contains(t, primitives[0].Content, "mode: agent")
	require.Contains(t, primitives[0].Content, "compiled-from: workflow")

	require.Len(t, notes, 1)
	require.Equal(t, renderer.LevelInfo, notes[0].Level)
	require.Equal(t, renderer.CodeWorkflowLoweredToPromptFile, notes[0].Code)
}

func TestTranslateWorkflow_Gemini_CustomCommand(t *testing.T) {
	wf := threeStepWorkflow()
	wf.Targets = map[string]ast.TargetOverride{
		"gemini": {Provider: map[string]any{"lowering-strategy": "custom-command"}},
	}

	primitives, notes := translator.TranslateWorkflow(wf, "gemini")

	require.Len(t, primitives, 1)
	require.Equal(t, "custom-command", primitives[0].Kind)
	require.True(t, strings.HasSuffix(primitives[0].Path, ".md"))
	require.Contains(t, primitives[0].Path, ".gemini/commands/")

	require.Len(t, notes, 1)
	require.Equal(t, renderer.LevelInfo, notes[0].Level)
	require.Equal(t, renderer.CodeWorkflowLoweredToCustomCommand, notes[0].Code)
}

func TestTranslateWorkflow_NoNativeTarget_EmitsErrorNote(t *testing.T) {
	wf := threeStepWorkflow()
	// No targets configured — no strategy specified.

	_, notes := translator.TranslateWorkflow(wf, "cursor")

	require.NotEmpty(t, notes)
	// At minimum a warning (strict mode would be error).
	require.Equal(t, renderer.CodeWorkflowNoNativeTarget, notes[len(notes)-1].Code)
}

func TestTranslateWorkflow_ProvenanceMarker_StepSkillsList(t *testing.T) {
	primitives, _ := translator.TranslateWorkflow(threeStepWorkflow(), "claude")

	var ruleContent string
	for _, p := range primitives {
		if p.Kind == "rule" {
			ruleContent = p.Content
			break
		}
	}

	require.Contains(t, ruleContent, "step-skills:")
	require.Contains(t, ruleContent, "code-review-01-analyze")
	require.Contains(t, ruleContent, "code-review-02-lint")
	require.Contains(t, ruleContent, "code-review-03-summarize")
}
