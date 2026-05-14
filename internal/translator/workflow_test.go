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
			{Name: "analyze", Body: "Read the diff."},
			{Name: "lint", Body: "Check style."},
			{Name: "summarize", Body: "Write the review comment."},
		},
	}
}

func TestTranslateWorkflow_Claude_RulePlusSkill_EmitsRule(t *testing.T) {
	wf := threeStepWorkflow()
	wf.Targets = map[string]ast.TargetOverride{
		"claude": {Provider: map[string]any{"lowering-strategy": "rule-plus-skill"}},
	}
	primitives, notes := translator.TranslateWorkflow(wf, "claude")

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
	wf := threeStepWorkflow()
	wf.Targets = map[string]ast.TargetOverride{
		"claude": {Provider: map[string]any{"lowering-strategy": "rule-plus-skill"}},
	}
	primitives, _ := translator.TranslateWorkflow(wf, "claude")

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

func TestInferWorkflowMode_BodyOnly(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name: "routing-wf",
		Body: "# Router\n## Step 1\nDo something...",
	}
	mode := translator.InferWorkflowMode(wf)
	require.Equal(t, translator.ModeRouted, mode)
}

func TestInferWorkflowMode_StepsWithSkillRefs(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name: "chain-wf",
		Steps: []ast.WorkflowStep{
			{Name: "design", Skill: "brainstorming"},
			{Name: "review", Skill: "claude-cli-review"},
		},
	}
	mode := translator.InferWorkflowMode(wf)
	require.Equal(t, translator.ModeChained, mode)
}

func TestInferWorkflowMode_StepsWithBodies(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name: "simple-wf",
		Steps: []ast.WorkflowStep{
			{Name: "lint", Body: "Run linter."},
			{Name: "test", Body: "Run tests."},
		},
	}
	mode := translator.InferWorkflowMode(wf)
	require.Equal(t, translator.ModeSimple, mode)
}

func TestInferWorkflowMode_MixedSteps(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name: "mixed-wf",
		Steps: []ast.WorkflowStep{
			{Name: "design", Skill: "brainstorming"},
			{Name: "custom", Body: "Do something custom."},
		},
	}
	mode := translator.InferWorkflowMode(wf)
	require.Equal(t, translator.ModeChained, mode)
}

func TestInferWorkflowMode_BodyAndSteps(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name: "ambiguous-wf",
		Body: "Some body content.",
		Steps: []ast.WorkflowStep{
			{Name: "step1", Body: "Step 1 content."},
		},
	}
	mode := translator.InferWorkflowMode(wf)
	require.Equal(t, translator.ModeSimple, mode)
}

func TestInferWorkflowMode_EmptyWorkflow(t *testing.T) {
	wf := &ast.WorkflowConfig{Name: "empty"}
	mode := translator.InferWorkflowMode(wf)
	require.Equal(t, translator.ModeSimple, mode)
}

func TestTranslateWorkflow_NoStrategy_DefaultsToSimple(t *testing.T) {
	wf := threeStepWorkflow()
	primitives, notes := translator.TranslateWorkflow(wf, "cursor")

	require.NotEmpty(t, primitives)
	require.Equal(t, "skill", primitives[0].Kind)

	hasDefaultChanged := false
	for _, n := range notes {
		if n.Code == renderer.CodeWorkflowDefaultChanged {
			hasDefaultChanged = true
		}
	}
	require.True(t, hasDefaultChanged, "expected CodeWorkflowDefaultChanged migration note")
}

func TestTranslateWorkflow_ProvenanceMarker_StepSkillsList(t *testing.T) {
	wf := threeStepWorkflow()
	wf.Targets = map[string]ast.TargetOverride{
		"claude": {Provider: map[string]any{"lowering-strategy": "rule-plus-skill"}},
	}
	primitives, _ := translator.TranslateWorkflow(wf, "claude")

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

func TestTranslateWorkflow_AlwaysApply_EmitsRule(t *testing.T) {
	alwaysApply := true
	wf := &ast.WorkflowConfig{
		Name:        "mandatory-wf",
		Description: "Always active.",
		AlwaysApply: &alwaysApply,
		Body:        "Do things.",
	}
	primitives, notes := translator.TranslateWorkflow(wf, "claude")

	var hasRule bool
	for _, p := range primitives {
		if p.Kind == "rule" {
			hasRule = true
			require.Equal(t, "mandatory-wf-workflow", p.ID)
			require.Contains(t, p.Content, "mandatory-wf")
		}
	}
	require.True(t, hasRule, "expected a rule primitive for always-apply workflow")
	require.NotEmpty(t, notes)
}

func TestTranslateWorkflow_Paths_EmitsRule(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name:  "go-workflow",
		Paths: ast.ClearableList{Values: []string{"*.go", "*.mod"}},
		Steps: []ast.WorkflowStep{
			{Name: "vet", Body: "Run go vet."},
		},
	}
	primitives, notes := translator.TranslateWorkflow(wf, "claude")

	var hasRule bool
	for _, p := range primitives {
		if p.Kind == "rule" {
			hasRule = true
			require.Contains(t, p.Content, "*.go")
		}
	}
	require.True(t, hasRule, "expected a rule primitive for paths-scoped workflow")
	require.NotEmpty(t, notes)
}

func TestTranslateWorkflow_NoModifier_NoRule(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name: "manual-wf",
		Body: "Do things manually.",
	}
	primitives, _ := translator.TranslateWorkflow(wf, "claude")

	for _, p := range primitives {
		require.NotEqual(t, "rule", p.Kind, "no rule should be emitted without always-apply or paths")
	}
}

func TestTranslateWorkflow_BodyAndSteps_BodyIgnoredWarning(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name: "ambiguous",
		Body: "I should be ignored.",
		Steps: []ast.WorkflowStep{
			{Name: "step1", Body: "Step body."},
		},
	}
	_, notes := translator.TranslateWorkflow(wf, "claude")

	hasBodyIgnored := false
	for _, n := range notes {
		if n.Code == renderer.CodeWorkflowBodyIgnored {
			hasBodyIgnored = true
		}
	}
	require.True(t, hasBodyIgnored, "expected CodeWorkflowBodyIgnored warning")
}
