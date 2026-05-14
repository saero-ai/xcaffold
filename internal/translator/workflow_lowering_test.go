package translator

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/stretchr/testify/require"
)

// --- lowerOrchestratorSkill ---

func TestLowerOrchestratorSkill(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name: "feature-lifecycle",
		Steps: []ast.WorkflowStep{
			{Name: "design", Skill: "brainstorming", Description: "Run brainstorming."},
			{Name: "review", Skill: "claude-cli-review", Description: "Review the spec."},
			{Name: "plan", Skill: "writing-plans", Description: "Write the plan."},
		},
	}
	primitives, notes := lowerOrchestratorSkill(wf, "feature-lifecycle", "claude")

	require.Len(t, primitives, 1)
	require.Equal(t, "skill", primitives[0].Kind)
	require.Equal(t, "feature-lifecycle", primitives[0].ID)
	require.Contains(t, primitives[0].Content, "brainstorming")
	require.Contains(t, primitives[0].Content, "claude-cli-review")
	require.Contains(t, primitives[0].Content, "writing-plans")

	require.Len(t, notes, 1)
	require.Equal(t, renderer.CodeWorkflowChainedToOrchestrator, notes[0].Code)
}

func TestLowerOrchestratorSkill_MixedSteps_EmitsNote(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name: "mixed-wf",
		Steps: []ast.WorkflowStep{
			{Name: "design", Skill: "brainstorming"},
			{Name: "custom", Instructions: "Do something inline."},
		},
	}
	primitives, notes := lowerOrchestratorSkill(wf, "mixed-wf", "claude")

	// main skill + sub-skill for instructions-only step
	require.GreaterOrEqual(t, len(primitives), 2)

	main := primitives[0]
	require.Equal(t, "skill", main.Kind)
	require.Contains(t, main.Content, "brainstorming")

	// Sub-skill carries the inline instructions
	var subSkill *TargetPrimitive
	for i := range primitives {
		if primitives[i].ID != "mixed-wf" {
			subSkill = &primitives[i]
			break
		}
	}
	require.NotNil(t, subSkill)
	require.Contains(t, subSkill.Content, "Do something inline.")

	hasMixedNote := false
	for _, n := range notes {
		if n.Code == renderer.CodeWorkflowMixedSteps {
			hasMixedNote = true
		}
	}
	require.True(t, hasMixedNote, "expected CodeWorkflowMixedSteps note for mixed steps")
}

// --- lowerBasicSkill ---

func TestLowerBasicSkill(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name: "coding-standards",
		Steps: []ast.WorkflowStep{
			{Name: "lint", Instructions: "Run linting before commit."},
			{Name: "test", Instructions: "Run tests before push."},
		},
	}
	primitives, notes := lowerBasicSkill(wf, "coding-standards", "claude")

	require.Len(t, primitives, 1)
	require.Equal(t, "skill", primitives[0].Kind)
	require.Equal(t, "coding-standards", primitives[0].ID)
	require.Contains(t, primitives[0].Content, "## lint")
	require.Contains(t, primitives[0].Content, "Run linting before commit.")
	require.Contains(t, primitives[0].Content, "## test")
	require.Contains(t, primitives[0].Content, "Run tests before push.")

	hasSimpleNote := false
	for _, n := range notes {
		if n.Code == renderer.CodeWorkflowBasicToSections {
			hasSimpleNote = true
		}
	}
	require.True(t, hasSimpleNote)
}

// --- buildWorkflowRule ---

func TestBuildWorkflowRule_ActivationAlways(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name:        "mandatory-wf",
		Description: "Always active.",
		Activation:  &ast.Activation{Mode: "always"},
	}
	rule := buildWorkflowRule(wf, "mandatory-wf", "claude")

	require.NotNil(t, rule)
	require.Equal(t, "rule", rule.Kind)
	require.Equal(t, "mandatory-wf-workflow", rule.ID)
	require.Contains(t, rule.Content, "mandatory-wf")
	require.Contains(t, rule.Content, "always active")
}

func TestBuildWorkflowRule_ActivationPaths(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name:       "go-workflow",
		Activation: &ast.Activation{Mode: "paths", Paths: []string{"*.go", "*.mod"}},
	}
	rule := buildWorkflowRule(wf, "go-workflow", "claude")

	require.NotNil(t, rule)
	require.Equal(t, "rule", rule.Kind)
	require.Contains(t, rule.Content, "*.go")
}

func TestBuildWorkflowRule_NoActivation_NoRule(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name:  "manual-wf",
		Steps: []ast.WorkflowStep{{Name: "step1", Instructions: "Do things manually."}},
	}
	rule := buildWorkflowRule(wf, "manual-wf", "claude")
	require.Nil(t, rule)
}
