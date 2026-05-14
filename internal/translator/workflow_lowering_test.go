package translator

import (
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/stretchr/testify/require"
)

// --- lowerRoutedSkill ---

func TestLowerRoutedSkill(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name:        "my-router",
		Description: "Routes to different behaviors.",
		Body:        "# Router\n## Step 1\nDo analysis.\n## Step 2\nDo review.",
	}
	primitives, notes := lowerRoutedSkill(wf, "my-router", "claude")

	require.Len(t, primitives, 1)
	require.Equal(t, "skill", primitives[0].Kind)
	require.Equal(t, "my-router", primitives[0].ID)
	require.Contains(t, primitives[0].Content, "# Router")
	require.Contains(t, primitives[0].Content, "Do analysis.")

	require.Len(t, notes, 1)
	require.Equal(t, renderer.CodeWorkflowRoutedToSingleSkill, notes[0].Code)
}

// --- lowerChainedSkill ---

func TestLowerChainedSkill(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name: "feature-lifecycle",
		Steps: []ast.WorkflowStep{
			{Name: "design", Skill: "brainstorming", Description: "Run brainstorming."},
			{Name: "review", Skill: "claude-cli-review", Description: "Review the spec."},
			{Name: "plan", Skill: "writing-plans", Description: "Write the plan."},
		},
	}
	primitives, notes := lowerChainedSkill(wf, "feature-lifecycle", "claude")

	require.Len(t, primitives, 1)
	require.Equal(t, "skill", primitives[0].Kind)
	require.Equal(t, "feature-lifecycle", primitives[0].ID)
	require.Contains(t, primitives[0].Content, "brainstorming")
	require.Contains(t, primitives[0].Content, "claude-cli-review")
	require.Contains(t, primitives[0].Content, "writing-plans")

	require.Len(t, notes, 1)
	require.Equal(t, renderer.CodeWorkflowChainedToOrchestrator, notes[0].Code)
}

func TestLowerChainedSkill_MixedSteps_EmitsNote(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name: "mixed-wf",
		Steps: []ast.WorkflowStep{
			{Name: "design", Skill: "brainstorming"},
			{Name: "custom", Body: "Do something inline."},
		},
	}
	primitives, notes := lowerChainedSkill(wf, "mixed-wf", "claude")

	require.Len(t, primitives, 1)
	require.Equal(t, "skill", primitives[0].Kind)
	require.Contains(t, primitives[0].Content, "brainstorming")
	require.Contains(t, primitives[0].Content, "Do something inline.")

	hasMixedNote := false
	for _, n := range notes {
		if n.Code == renderer.CodeWorkflowMixedSteps {
			hasMixedNote = true
		}
	}
	require.True(t, hasMixedNote, "expected CodeWorkflowMixedSteps note for mixed steps")
}

// --- lowerSimpleSkill ---

func TestLowerSimpleSkill(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name: "coding-standards",
		Steps: []ast.WorkflowStep{
			{Name: "lint", Body: "Run linting before commit."},
			{Name: "test", Body: "Run tests before push."},
		},
	}
	primitives, notes := lowerSimpleSkill(wf, "coding-standards", "claude")

	require.Len(t, primitives, 1)
	require.Equal(t, "skill", primitives[0].Kind)
	require.Equal(t, "coding-standards", primitives[0].ID)
	require.Contains(t, primitives[0].Content, "## lint")
	require.Contains(t, primitives[0].Content, "Run linting before commit.")
	require.Contains(t, primitives[0].Content, "## test")
	require.Contains(t, primitives[0].Content, "Run tests before push.")

	hasSimpleNote := false
	for _, n := range notes {
		if n.Code == renderer.CodeWorkflowSimpleToSections {
			hasSimpleNote = true
		}
	}
	require.True(t, hasSimpleNote)
}

// --- buildWorkflowRule ---

func TestBuildWorkflowRule_AlwaysApply(t *testing.T) {
	alwaysApply := true
	wf := &ast.WorkflowConfig{
		Name:        "mandatory-wf",
		Description: "Always active.",
		AlwaysApply: &alwaysApply,
	}
	rule := buildWorkflowRule(wf, "mandatory-wf", "claude")

	require.NotNil(t, rule)
	require.Equal(t, "rule", rule.Kind)
	require.Equal(t, "mandatory-wf-workflow", rule.ID)
	require.Contains(t, rule.Content, "mandatory-wf")
	require.Contains(t, strings.ToLower(rule.Content), "always active")
}

func TestBuildWorkflowRule_Paths(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name:  "go-workflow",
		Paths: ast.ClearableList{Values: []string{"*.go", "*.mod"}},
	}
	rule := buildWorkflowRule(wf, "go-workflow", "claude")

	require.NotNil(t, rule)
	require.Equal(t, "rule", rule.Kind)
	require.Contains(t, rule.Content, "*.go")
}

func TestBuildWorkflowRule_NoModifier_NoRule(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name: "manual-wf",
		Body: "Do things manually.",
	}
	rule := buildWorkflowRule(wf, "manual-wf", "claude")
	require.Nil(t, rule)
}
