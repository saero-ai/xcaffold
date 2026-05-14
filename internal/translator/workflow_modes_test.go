package translator_test

// Tests T-19 through T-35: new Basic/Orchestrator mode logic.
// These tests cover InferWorkflowMode, lowerBasicSkill, lowerOrchestratorSkill,
// buildWorkflowRule (Activation-based), and the DefaultChanged/MixedSteps notes.

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/translator"
	"github.com/stretchr/testify/require"
)

// T-19: All steps have instructions only → ModeBasic
func TestInferWorkflowMode_AllInstructions_Basic(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name: "all-instructions",
		Steps: []ast.WorkflowStep{
			{Name: "step1", Instructions: "Do step 1."},
			{Name: "step2", Instructions: "Do step 2."},
		},
	}
	mode := translator.InferWorkflowMode(wf)
	require.Equal(t, translator.ModeBasic, mode)
}

// T-20: One step has skill → ModeOrchestrator
func TestInferWorkflowMode_AnySkillRef_Orchestrator(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name: "any-skill",
		Steps: []ast.WorkflowStep{
			{Name: "step1", Instructions: "Do step 1."},
			{Name: "step2", Skill: "some-skill"},
		},
	}
	mode := translator.InferWorkflowMode(wf)
	require.Equal(t, translator.ModeOrchestrator, mode)
}

// T-21: Mix of skill-ref and instructions → ModeOrchestrator
func TestInferWorkflowMode_MixedSteps_Orchestrator(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name: "mixed",
		Steps: []ast.WorkflowStep{
			{Name: "design", Skill: "brainstorming"},
			{Name: "custom", Instructions: "Run custom logic."},
		},
	}
	mode := translator.InferWorkflowMode(wf)
	require.Equal(t, translator.ModeOrchestrator, mode)
}

// T-22: 1 step, instructions only → ModeBasic
func TestInferWorkflowMode_SingleStep_Instructions(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name:  "single",
		Steps: []ast.WorkflowStep{{Name: "step1", Instructions: "Do it."}},
	}
	mode := translator.InferWorkflowMode(wf)
	require.Equal(t, translator.ModeBasic, mode)
}

// T-23: All steps with skill refs → ModeOrchestrator
func TestInferWorkflowMode_AllSkillRefs(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name: "all-skills",
		Steps: []ast.WorkflowStep{
			{Name: "step1", Skill: "skill-a"},
			{Name: "step2", Skill: "skill-b"},
		},
	}
	mode := translator.InferWorkflowMode(wf)
	require.Equal(t, translator.ModeOrchestrator, mode)
}

// T-24: Basic mode renders ## sections in one skill
func TestTranslate_BasicMode_SingleSkillWithSections(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name: "basic-wf",
		Steps: []ast.WorkflowStep{
			{Name: "setup", Instructions: "Install dependencies."},
			{Name: "build", Instructions: "Run make build."},
		},
	}
	primitives, _ := translator.TranslateWorkflow(wf, "claude")

	require.NotEmpty(t, primitives)
	require.Equal(t, "skill", primitives[0].Kind)
	require.Equal(t, "basic-wf", primitives[0].ID)
	require.Contains(t, primitives[0].Content, "## setup")
	require.Contains(t, primitives[0].Content, "Install dependencies.")
	require.Contains(t, primitives[0].Content, "## build")
	require.Contains(t, primitives[0].Content, "Run make build.")
}

// T-25: Orchestrator mode renders main skill with /skill invocations
func TestTranslate_OrchestratorMode_MainSkill(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name: "orch-wf",
		Steps: []ast.WorkflowStep{
			{Name: "design", Skill: "brainstorming", Description: "Run design."},
			{Name: "review", Skill: "code-review"},
		},
	}
	primitives, _ := translator.TranslateWorkflow(wf, "claude")

	skillPrimitives := filterKind(primitives, "skill")
	require.NotEmpty(t, skillPrimitives)
	main := skillPrimitives[0]
	require.Equal(t, "orch-wf", main.ID)
	require.Contains(t, main.Content, "/brainstorming")
	require.Contains(t, main.Content, "/code-review")
}

// T-26: Instructions-only step in orchestrator becomes separate skill primitive
func TestTranslate_OrchestratorMode_InlineStepBecomesSkill(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name: "orch-inline",
		Steps: []ast.WorkflowStep{
			{Name: "design", Skill: "brainstorming"},
			{Name: "custom-step", Instructions: "Do custom work."},
		},
	}
	primitives, _ := translator.TranslateWorkflow(wf, "claude")

	skills := filterKind(primitives, "skill")
	// Main skill + sub-skill for custom-step
	require.GreaterOrEqual(t, len(skills), 2)

	var subSkill *translator.TargetPrimitive
	for i := range skills {
		if skills[i].ID != "orch-inline" {
			subSkill = &skills[i]
			break
		}
	}
	require.NotNil(t, subSkill, "expected a sub-skill for the instructions-only step")
	require.Contains(t, subSkill.Content, "Do custom work.")
}

// T-27: Skill ref step shows "Invoke `/skill-name`"
func TestTranslate_OrchestratorMode_SkillRefInvocation(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name: "ref-wf",
		Steps: []ast.WorkflowStep{
			{Name: "step1", Skill: "my-skill"},
		},
	}
	primitives, _ := translator.TranslateWorkflow(wf, "claude")

	skills := filterKind(primitives, "skill")
	require.NotEmpty(t, skills)
	require.Contains(t, skills[0].Content, "Invoke the `/my-skill` skill.")
}

// T-28: Step with both skill and instructions renders both
func TestTranslate_OrchestratorMode_SkillPlusInstructions(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name: "mixed-skill",
		Steps: []ast.WorkflowStep{
			{Name: "do-it", Skill: "some-skill", Instructions: "Extra context here."},
		},
	}
	primitives, _ := translator.TranslateWorkflow(wf, "claude")

	skills := filterKind(primitives, "skill")
	require.NotEmpty(t, skills)
	main := skills[0]
	require.Contains(t, main.Content, "Invoke the `/some-skill` skill.")
	require.Contains(t, main.Content, "Extra context here.")
}

// T-29: Activation nil → zero rule primitives
func TestTranslate_NoActivation_NoRule(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name: "no-activation",
		Steps: []ast.WorkflowStep{
			{Name: "step1", Instructions: "Do something."},
		},
	}
	primitives, _ := translator.TranslateWorkflow(wf, "claude")

	rules := filterKind(primitives, "rule")
	require.Empty(t, rules, "no rule should be emitted without activation")
}

// T-30: activation: always → 1 rule primitive
func TestTranslate_ActivationAlways_EmitsRule(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name:       "always-wf",
		Activation: &ast.Activation{Mode: "always"},
		Steps: []ast.WorkflowStep{
			{Name: "step1", Instructions: "Do it."},
		},
	}
	primitives, _ := translator.TranslateWorkflow(wf, "claude")

	rules := filterKind(primitives, "rule")
	require.Len(t, rules, 1, "expected 1 rule primitive for activation: always")
	require.Equal(t, "always-wf-workflow", rules[0].ID)
	require.Contains(t, rules[0].Content, "always active")
}

// T-31: activation: paths → rule with paths
func TestTranslate_ActivationPaths_EmitsRule(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name:       "paths-wf",
		Activation: &ast.Activation{Mode: "paths", Paths: []string{"*.go", "*.mod"}},
		Steps: []ast.WorkflowStep{
			{Name: "step1", Instructions: "Do it."},
		},
	}
	primitives, _ := translator.TranslateWorkflow(wf, "claude")

	rules := filterKind(primitives, "rule")
	require.Len(t, rules, 1, "expected 1 rule primitive for activation: paths")
	require.Contains(t, rules[0].Content, "*.go")
	require.Contains(t, rules[0].Content, "*.mod")
}

// T-32: No activation + basic mode → DefaultChanged note
func TestTranslate_BasicMode_EmitsDefaultChanged(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name: "default-changed",
		Steps: []ast.WorkflowStep{
			{Name: "step1", Instructions: "Do it."},
		},
	}
	_, notes := translator.TranslateWorkflow(wf, "claude")

	hasDefaultChanged := false
	for _, n := range notes {
		if n.Code == renderer.CodeWorkflowDefaultChanged {
			hasDefaultChanged = true
		}
	}
	require.True(t, hasDefaultChanged, "expected CodeWorkflowDefaultChanged when no activation in basic mode")
}

// T-33: Mixed steps in orchestrator → MixedSteps note
func TestTranslate_OrchestratorMode_EmitsMixedSteps(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name: "mixed-orch",
		Steps: []ast.WorkflowStep{
			{Name: "design", Skill: "brainstorming"},
			{Name: "custom", Instructions: "Inline instructions."},
		},
	}
	_, notes := translator.TranslateWorkflow(wf, "claude")

	hasMixed := false
	for _, n := range notes {
		if n.Code == renderer.CodeWorkflowMixedSteps {
			hasMixed = true
		}
	}
	require.True(t, hasMixed, "expected CodeWorkflowMixedSteps for mixed orchestrator steps")
}

// T-34: lowering-strategy override still works
func TestTranslate_LoweringStrategyOverride_Bypasses(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name: "override-wf",
		Steps: []ast.WorkflowStep{
			{Name: "step1", Instructions: "Do it."},
			{Name: "step2", Instructions: "Do more."},
		},
		Targets: map[string]ast.TargetOverride{
			"claude": {Provider: map[string]any{"lowering-strategy": "rule-plus-skill"}},
		},
	}
	primitives, notes := translator.TranslateWorkflow(wf, "claude")

	rules := filterKind(primitives, "rule")
	require.NotEmpty(t, rules, "explicit rule-plus-skill should still emit a rule")
	require.Len(t, notes, 1)
	require.Equal(t, renderer.CodeWorkflowLoweredToRulePlusSkill, notes[0].Code)
}

// T-35: rule-plus-skill explicit strategy still works
func TestTranslate_ExplicitRulePlusSkill_StillWorks(t *testing.T) {
	wf := &ast.WorkflowConfig{
		Name: "rps-wf",
		Steps: []ast.WorkflowStep{
			{Name: "analyze", Instructions: "Analyze code."},
			{Name: "report", Instructions: "Write report."},
		},
		Targets: map[string]ast.TargetOverride{
			"claude": {Provider: map[string]any{"lowering-strategy": "rule-plus-skill"}},
		},
	}
	primitives, _ := translator.TranslateWorkflow(wf, "claude")

	skills := filterKind(primitives, "skill")
	require.Len(t, skills, 2, "rule-plus-skill should emit one skill per step")
}

// filterKind filters a slice of TargetPrimitive by kind.
func filterKind(primitives []translator.TargetPrimitive, kind string) []translator.TargetPrimitive {
	var out []translator.TargetPrimitive
	for _, p := range primitives {
		if p.Kind == kind {
			out = append(out, p)
		}
	}
	return out
}
