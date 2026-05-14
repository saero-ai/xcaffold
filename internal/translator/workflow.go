package translator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
)

// slugRe matches any run of non-alphanumeric characters to collapse into a dash.
var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// workflowNoteSpec holds the identifying fields for a workflow fidelity note.
type workflowNoteSpec struct {
	target string
	name   string
	code   string
	reason string
}

// newWorkflowNote constructs a FidelityNote with Kind pre-set to "workflow".
func newWorkflowNote(level renderer.FidelityLevel, spec workflowNoteSpec) renderer.FidelityNote {
	return renderer.FidelityNote{
		Level:    level,
		Target:   spec.target,
		Kind:     "workflow",
		Resource: spec.name,
		Code:     spec.code,
		Reason:   spec.reason,
	}
}

// WorkflowMode describes the composition pattern inferred from a workflow's structure.
type WorkflowMode string

const (
	ModeBasic        WorkflowMode = "basic"
	ModeOrchestrator WorkflowMode = "orchestrator"
)

// InferWorkflowMode determines the rendering mode from a workflow's structure.
// Steps with any skill reference → Orchestrator. All-instructions steps → Basic.
func InferWorkflowMode(wf *ast.WorkflowConfig) WorkflowMode {
	for _, step := range wf.Steps {
		if step.Skill != "" {
			return ModeOrchestrator
		}
	}
	return ModeBasic
}

// slugify lowercases s, replaces non-alphanumeric runs with a single dash, and
// trims leading/trailing dashes.
func slugify(s string) string {
	s = strings.ToLower(s)
	s = slugRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

// stepSkillID produces the canonical step-skill identifier:
// <workflow-name>-<NN zero-padded>-<step-name-slugified>
func stepSkillID(workflowName string, i int, stepName string) string {
	return fmt.Sprintf("%s-%02d-%s", workflowName, i+1, slugify(stepName))
}

// loweringStrategy returns the lowering-strategy string from the workflow's
// target override provider map. Returns "" when not set.
func loweringStrategy(wf *ast.WorkflowConfig, target string) string {
	override, ok := wf.Targets[target]
	if !ok {
		return ""
	}
	v, ok := override.Provider["lowering-strategy"]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// resolveExplicitStrategy checks for an explicit lowering-strategy or native
// workflow promotion in the target overrides. Returns the primitives and notes
// if a match is found, or nil if structure-based inference should be used.
func resolveExplicitStrategy(wf *ast.WorkflowConfig, name, target string) ([]TargetPrimitive, []renderer.FidelityNote) {
	strategy := loweringStrategy(wf, target)
	if strategy != "" {
		switch strategy {
		case "prompt-file":
			return lowerPromptFile(wf, name, target)
		case "custom-command":
			return lowerCustomCommand(wf, name, target)
		case "rule-plus-skill":
			return lowerRulePlusSkill(wf, name, target)
		}
	}

	if override, ok := wf.Targets[target]; ok {
		if v, ok := override.Provider["promote-rules-to-workflows"]; ok {
			if promote, _ := v.(bool); promote {
				return lowerNativeWorkflow(wf, name, target)
			}
		}
	}

	return nil, nil
}

// lowerByMode infers the workflow mode from its structure and dispatches to the
// appropriate lowering function. Returns primitives and notes including any
// migration warnings.
func lowerByMode(wf *ast.WorkflowConfig, name, target string) ([]TargetPrimitive, []renderer.FidelityNote) {
	mode := InferWorkflowMode(wf)
	var notes []renderer.FidelityNote
	var primitives []TargetPrimitive

	switch mode {
	case ModeOrchestrator:
		p, n := lowerOrchestratorSkill(wf, name, target)
		primitives, notes = p, append(notes, n...)
	case ModeBasic:
		p, n := lowerBasicSkill(wf, name, target)
		primitives, notes = p, append(notes, n...)

		if wf.Activation == nil {
			n := newWorkflowNote(renderer.LevelWarning, workflowNoteSpec{
				target: target, name: name,
				code:   renderer.CodeWorkflowDefaultChanged,
				reason: fmt.Sprintf("workflow %q rendered as single skill (new default); previously would have been rule+skill", name),
			})
			n.Mitigation = "Add activation: always to restore ambient rule injection."
			notes = append(notes, n)
		}
	}

	return primitives, notes
}

// TranslateWorkflow lowers a WorkflowConfig into one or more TargetPrimitives
// for the named target platform. The strategy is determined in this order:
//
//  1. Explicit lowering-strategy in targets takes precedence.
//  2. Target override promote-rules-to-workflows: true → native workflow.
//  3. Structure-based inference: body-only → routed, skill-ref steps → chained,
//     inline-body steps → simple.
//  4. Activation rule appended when always-apply or paths is set.
func TranslateWorkflow(wf *ast.WorkflowConfig, target string) ([]TargetPrimitive, []renderer.FidelityNote) {
	name := wf.Name
	if name == "" {
		name = "unnamed"
	}

	if primitives, notes := resolveExplicitStrategy(wf, name, target); primitives != nil {
		return primitives, notes
	}

	primitives, notes := lowerByMode(wf, name, target)

	if rule := buildWorkflowRule(wf, name, target); rule != nil {
		primitives = append(primitives, *rule)
	}

	return primitives, notes
}

// buildRulePlusSkillRule constructs the rule primitive (provenance block +
// invocation instruction) from the pre-computed step names and skill IDs.
func buildRulePlusSkillRule(name string, stepNames, skillIDs []string) TargetPrimitive {
	var marker strings.Builder
	marker.WriteString("x-xcaffold:\n")
	fmt.Fprintf(&marker, "  compiled-from: workflow\n")
	fmt.Fprintf(&marker, "  workflow-name: %s\n", name)
	fmt.Fprintf(&marker, "  api-version: workflow/v1\n")
	marker.WriteString("  step-order: [")
	marker.WriteString(strings.Join(stepNames, ", "))
	marker.WriteString("]\n")
	marker.WriteString("  step-skills:\n")
	for _, sid := range skillIDs {
		fmt.Fprintf(&marker, "    - %s\n", sid)
	}

	var ruleBody strings.Builder
	ruleBody.WriteString("```yaml\n")
	ruleBody.WriteString(marker.String())
	ruleBody.WriteString("```\n\n")
	fmt.Fprintf(&ruleBody, "Run steps in order: %s.\n", strings.Join(skillIDs, " → "))

	return TargetPrimitive{Kind: "rule", ID: name + "-workflow", Content: ruleBody.String()}
}

// lowerRulePlusSkill emits one rule with a provenance marker plus one skill per step.
func lowerRulePlusSkill(wf *ast.WorkflowConfig, name, target string) ([]TargetPrimitive, []renderer.FidelityNote) {
	stepNames := make([]string, len(wf.Steps))
	skillIDs := make([]string, len(wf.Steps))
	for i, step := range wf.Steps {
		stepNames[i] = step.Name
		skillIDs[i] = stepSkillID(name, i, step.Name)
	}

	primitives := []TargetPrimitive{buildRulePlusSkillRule(name, stepNames, skillIDs)}

	for i, step := range wf.Steps {
		body := step.Instructions
		if body == "" {
			body = step.Description
		}
		primitives = append(primitives, TargetPrimitive{
			Kind:    "skill",
			ID:      skillIDs[i],
			Content: body,
		})
	}

	note := newWorkflowNote(renderer.LevelWarning, workflowNoteSpec{
		target: target, name: name,
		code:   renderer.CodeWorkflowLoweredToRulePlusSkill,
		reason: fmt.Sprintf("workflow %q lowered to rule+skill; %s has no native workflow primitive", name, target),
	})
	note.Mitigation = "Accept the rule-plus-skill lowering or add a target with a native workflow primitive."
	return primitives, []renderer.FidelityNote{note}
}

// lowerNativeWorkflow emits a single workflow primitive with step instructions
// concatenated under ## <step-name> headers.
func lowerNativeWorkflow(wf *ast.WorkflowConfig, name, target string) ([]TargetPrimitive, []renderer.FidelityNote) {
	var body strings.Builder
	for _, step := range wf.Steps {
		fmt.Fprintf(&body, "## %s\n\n", step.Name)
		if step.Instructions != "" {
			body.WriteString(step.Instructions)
			body.WriteString("\n\n")
		}
	}

	p := TargetPrimitive{
		Kind:    "workflow",
		ID:      name,
		Content: body.String(),
	}

	note := newWorkflowNote(renderer.LevelInfo, workflowNoteSpec{
		target: target, name: name,
		code:   renderer.CodeWorkflowLoweredToNative,
		reason: fmt.Sprintf("workflow %q emitted as native workflow for %s", name, target),
	})
	return []TargetPrimitive{p}, []renderer.FidelityNote{note}
}

// lowerPromptFile emits a single prompt-file primitive for copilot.
func lowerPromptFile(wf *ast.WorkflowConfig, name, target string) ([]TargetPrimitive, []renderer.FidelityNote) {
	path := fmt.Sprintf(".github/prompts/%s.prompt.md", name)

	// Frontmatter with xcaffold provenance.
	var content strings.Builder
	content.WriteString("---\n")
	content.WriteString("mode: agent\n")
	content.WriteString("x-xcaffold:\n")
	fmt.Fprintf(&content, "  compiled-from: workflow\n")
	fmt.Fprintf(&content, "  workflow-name: %s\n", name)
	fmt.Fprintf(&content, "  api-version: workflow/v1\n")
	content.WriteString("---\n\n")

	// Step instructions concatenated.
	for _, step := range wf.Steps {
		if step.Instructions != "" {
			content.WriteString(step.Instructions)
			content.WriteString("\n\n")
		}
	}

	p := TargetPrimitive{
		Kind:    "prompt-file",
		ID:      name,
		Path:    path,
		Content: content.String(),
	}

	note := newWorkflowNote(renderer.LevelInfo, workflowNoteSpec{
		target: target, name: name,
		code:   renderer.CodeWorkflowLoweredToPromptFile,
		reason: fmt.Sprintf("workflow %q lowered to prompt file at %s", name, path),
	})
	return []TargetPrimitive{p}, []renderer.FidelityNote{note}
}

// lowerOrchestratorSkill emits a main orchestrator skill that references sub-skills
// by name for each step. Steps with only instructions become separate skill
// primitives. Mixed steps (both skill refs and inline instructions) emit a
// CodeWorkflowMixedSteps note.
func lowerOrchestratorSkill(wf *ast.WorkflowConfig, name, target string) ([]TargetPrimitive, []renderer.FidelityNote) {
	var mainBody strings.Builder
	var subSkills []TargetPrimitive
	fmt.Fprintf(&mainBody, "# %s\n\n", name)

	hasMixed := false
	for i, step := range wf.Steps {
		fmt.Fprintf(&mainBody, "## %d. %s\n\n", i+1, step.Name)
		if step.Description != "" {
			mainBody.WriteString(step.Description)
			mainBody.WriteString("\n\n")
		}
		if step.Skill != "" {
			fmt.Fprintf(&mainBody, "Invoke the `/%s` skill.\n\n", step.Skill)
			if step.Instructions != "" {
				hasMixed = true
				mainBody.WriteString(step.Instructions)
				mainBody.WriteString("\n\n")
			}
		} else if step.Instructions != "" {
			hasMixed = true
			subID := stepSkillID(name, i, step.Name)
			subSkills = append(subSkills, TargetPrimitive{
				Kind:    "skill",
				ID:      subID,
				Content: step.Instructions,
			})
			fmt.Fprintf(&mainBody, "Invoke the `/%s` skill.\n\n", subID)
		}
	}

	primitives := []TargetPrimitive{{Kind: "skill", ID: name, Content: mainBody.String()}}
	primitives = append(primitives, subSkills...)

	notes := []renderer.FidelityNote{
		newWorkflowNote(renderer.LevelInfo, workflowNoteSpec{
			target: target, name: name,
			code:   renderer.CodeWorkflowChainedToOrchestrator,
			reason: fmt.Sprintf("workflow %q rendered as orchestrator skill chaining sub-skills for %s", name, target),
		}),
	}
	if hasMixed {
		notes = append(notes, newWorkflowNote(renderer.LevelInfo, workflowNoteSpec{
			target: target, name: name,
			code:   renderer.CodeWorkflowMixedSteps,
			reason: fmt.Sprintf("workflow %q has mixed skill-ref and inline-instructions steps", name),
		}))
	}
	return primitives, notes
}

// lowerBasicSkill emits a single skill primitive whose body is the concatenation
// of each step's instructions under a ## <step-name> heading.
func lowerBasicSkill(wf *ast.WorkflowConfig, name, target string) ([]TargetPrimitive, []renderer.FidelityNote) {
	var body strings.Builder
	for _, step := range wf.Steps {
		fmt.Fprintf(&body, "## %s\n\n", step.Name)
		if step.Instructions != "" {
			body.WriteString(step.Instructions)
			body.WriteString("\n\n")
		}
	}

	p := TargetPrimitive{Kind: "skill", ID: name, Content: body.String()}

	notes := []renderer.FidelityNote{
		newWorkflowNote(renderer.LevelInfo, workflowNoteSpec{
			target: target, name: name,
			code:   renderer.CodeWorkflowBasicToSections,
			reason: fmt.Sprintf("workflow %q rendered as single skill with step sections for %s", name, target),
		}),
	}
	return []TargetPrimitive{p}, notes
}

// buildWorkflowRule constructs an optional rule primitive that activates the
// workflow automatically. Returns nil when activation is not set.
func buildWorkflowRule(wf *ast.WorkflowConfig, name, target string) *TargetPrimitive {
	if wf.Activation == nil {
		return nil
	}

	var ruleBody strings.Builder
	ruleBody.WriteString("```yaml\n")
	ruleBody.WriteString("x-xcaffold:\n")
	fmt.Fprintf(&ruleBody, "  compiled-from: workflow\n")
	fmt.Fprintf(&ruleBody, "  workflow-name: %s\n", name)
	ruleBody.WriteString("```\n\n")

	switch wf.Activation.Mode {
	case "always":
		fmt.Fprintf(&ruleBody, "This workflow (%s) is always active. Invoke `/%s` to begin.\n", name, name)
	case "paths":
		fmt.Fprintf(&ruleBody, "This workflow (%s) activates for paths: %s. Invoke `/%s` to begin.\n",
			name, strings.Join(wf.Activation.Paths, ", "), name)
	}

	return &TargetPrimitive{
		Kind:    "rule",
		ID:      name + "-workflow",
		Content: ruleBody.String(),
	}
}

// lowerCustomCommand emits a single custom-command primitive for gemini.
func lowerCustomCommand(wf *ast.WorkflowConfig, name, target string) ([]TargetPrimitive, []renderer.FidelityNote) {
	path := fmt.Sprintf(".gemini/commands/%s.md", name)

	var content strings.Builder
	for _, step := range wf.Steps {
		if step.Instructions != "" {
			content.WriteString(step.Instructions)
			content.WriteString("\n\n")
		}
	}

	p := TargetPrimitive{
		Kind:    "custom-command",
		ID:      name,
		Path:    path,
		Content: content.String(),
	}

	note := newWorkflowNote(renderer.LevelInfo, workflowNoteSpec{
		target: target, name: name,
		code:   renderer.CodeWorkflowLoweredToCustomCommand,
		reason: fmt.Sprintf("workflow %q lowered to custom command at %s", name, path),
	})
	return []TargetPrimitive{p}, []renderer.FidelityNote{note}
}
