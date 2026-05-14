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

// WorkflowMode describes the composition pattern inferred from a workflow's structure.
type WorkflowMode string

const (
	ModeRouted  WorkflowMode = "routed"
	ModeChained WorkflowMode = "chained"
	ModeSimple  WorkflowMode = "simple"
)

// InferWorkflowMode determines the rendering mode from a workflow's structure.
// Body-only workflows are routed (single skill). Steps with skill refs are
// chained (orchestrator). Steps with inline bodies are simple (sections).
func InferWorkflowMode(wf *ast.WorkflowConfig) WorkflowMode {
	if len(wf.Steps) == 0 && wf.Body != "" {
		return ModeRouted
	}
	for _, step := range wf.Steps {
		if step.Skill != "" {
			return ModeChained
		}
	}
	return ModeSimple
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

	// 1. Explicit lowering-strategy in targets takes precedence.
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

	// 2. Native workflow promotion for providers that support it.
	if override, ok := wf.Targets[target]; ok {
		if v, ok := override.Provider["promote-rules-to-workflows"]; ok {
			if promote, _ := v.(bool); promote {
				return lowerNativeWorkflow(wf, name, target)
			}
		}
	}

	// 3. Structure-based inference (new default path).
	mode := InferWorkflowMode(wf)

	var primitives []TargetPrimitive
	var notes []renderer.FidelityNote

	// Warn when body is present alongside steps (body will be ignored).
	if wf.Body != "" && len(wf.Steps) > 0 {
		notes = append(notes, renderer.FidelityNote{
			Level:      renderer.LevelWarning,
			Target:     target,
			Kind:       "workflow",
			Resource:   name,
			Code:       renderer.CodeWorkflowBodyIgnored,
			Reason:     fmt.Sprintf("workflow %q has both body and steps; body is ignored in %s mode", name, mode),
			Mitigation: "Move body content into a step, or remove steps to use routed mode.",
		})
	}

	switch mode {
	case ModeRouted:
		p, n := lowerRoutedSkill(wf, name, target)
		primitives, notes = p, append(notes, n...)
	case ModeChained:
		p, n := lowerChainedSkill(wf, name, target)
		primitives, notes = p, append(notes, n...)
	case ModeSimple:
		p, n := lowerSimpleSkill(wf, name, target)
		primitives, notes = p, append(notes, n...)

		// Migration warning: this workflow would have been rule+skill before.
		if wf.AlwaysApply == nil && wf.Paths.Len() == 0 && len(wf.Steps) > 0 {
			notes = append(notes, renderer.FidelityNote{
				Level:      renderer.LevelWarning,
				Target:     target,
				Kind:       "workflow",
				Resource:   name,
				Code:       renderer.CodeWorkflowDefaultChanged,
				Reason:     fmt.Sprintf("workflow %q rendered as single skill (new default); previously would have been rule+skill", name),
				Mitigation: "Add always-apply: true to restore ambient rule injection.",
			})
		}
	}

	// 4. Add activation rule if always-apply or paths is set.
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
		body := step.Body
		if body == "" {
			body = step.Description
		}
		primitives = append(primitives, TargetPrimitive{
			Kind:    "skill",
			ID:      skillIDs[i],
			Content: body,
		})
	}

	note := renderer.FidelityNote{
		Level:      renderer.LevelWarning,
		Target:     target,
		Kind:       "workflow",
		Resource:   name,
		Code:       renderer.CodeWorkflowLoweredToRulePlusSkill,
		Reason:     fmt.Sprintf("workflow %q lowered to rule+skill; %s has no native workflow primitive", name, target),
		Mitigation: "Accept the rule-plus-skill lowering or add a target with a native workflow primitive.",
	}
	return primitives, []renderer.FidelityNote{note}
}

// lowerNativeWorkflow emits a single workflow primitive with step bodies
// concatenated under ## <step-name> headers.
func lowerNativeWorkflow(wf *ast.WorkflowConfig, name, target string) ([]TargetPrimitive, []renderer.FidelityNote) {
	var body strings.Builder
	for _, step := range wf.Steps {
		fmt.Fprintf(&body, "## %s\n\n", step.Name)
		if step.Body != "" {
			body.WriteString(step.Body)
			body.WriteString("\n\n")
		}
	}

	p := TargetPrimitive{
		Kind:    "workflow",
		ID:      name,
		Content: body.String(),
	}

	note := renderer.FidelityNote{
		Level:    renderer.LevelInfo,
		Target:   target,
		Kind:     "workflow",
		Resource: name,
		Code:     renderer.CodeWorkflowLoweredToNative,
		Reason:   fmt.Sprintf("workflow %q emitted as native workflow for %s", name, target),
	}
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

	// Step bodies concatenated.
	for _, step := range wf.Steps {
		if step.Body != "" {
			content.WriteString(step.Body)
			content.WriteString("\n\n")
		}
	}

	p := TargetPrimitive{
		Kind:    "prompt-file",
		ID:      name,
		Path:    path,
		Content: content.String(),
	}

	note := renderer.FidelityNote{
		Level:    renderer.LevelInfo,
		Target:   target,
		Kind:     "workflow",
		Resource: name,
		Code:     renderer.CodeWorkflowLoweredToPromptFile,
		Reason:   fmt.Sprintf("workflow %q lowered to prompt file at %s", name, path),
	}
	return []TargetPrimitive{p}, []renderer.FidelityNote{note}
}

// lowerRoutedSkill emits a single skill primitive from a body-only workflow.
// Used when the workflow has no steps — it is treated as a routing skill.
func lowerRoutedSkill(wf *ast.WorkflowConfig, name, target string) ([]TargetPrimitive, []renderer.FidelityNote) {
	p := TargetPrimitive{
		Kind:    "skill",
		ID:      name,
		Content: wf.Body,
	}
	note := renderer.FidelityNote{
		Level:    renderer.LevelInfo,
		Target:   target,
		Kind:     "workflow",
		Resource: name,
		Code:     renderer.CodeWorkflowRoutedToSingleSkill,
		Reason:   fmt.Sprintf("workflow %q rendered as single routing skill for %s", name, target),
	}
	return []TargetPrimitive{p}, []renderer.FidelityNote{note}
}

// lowerChainedSkill emits a single orchestrator skill that references sub-skills
// by name for each step. Mixed steps (both skill refs and inline bodies) emit an
// additional CodeWorkflowMixedSteps note.
func lowerChainedSkill(wf *ast.WorkflowConfig, name, target string) ([]TargetPrimitive, []renderer.FidelityNote) {
	var body strings.Builder
	fmt.Fprintf(&body, "# %s\n\n", name)

	hasMixed := false
	for i, step := range wf.Steps {
		fmt.Fprintf(&body, "## %d. %s\n\n", i+1, step.Name)
		if step.Description != "" {
			body.WriteString(step.Description)
			body.WriteString("\n\n")
		}
		if step.Skill != "" {
			fmt.Fprintf(&body, "Invoke the `/%s` skill.\n\n", step.Skill)
			if step.Body != "" {
				hasMixed = true
				body.WriteString(step.Body)
				body.WriteString("\n\n")
			}
		} else {
			hasMixed = true
			if step.Body != "" {
				body.WriteString(step.Body)
				body.WriteString("\n\n")
			}
		}
	}

	p := TargetPrimitive{Kind: "skill", ID: name, Content: body.String()}

	notes := []renderer.FidelityNote{{
		Level:    renderer.LevelInfo,
		Target:   target,
		Kind:     "workflow",
		Resource: name,
		Code:     renderer.CodeWorkflowChainedToOrchestrator,
		Reason:   fmt.Sprintf("workflow %q rendered as orchestrator skill chaining sub-skills for %s", name, target),
	}}
	if hasMixed {
		notes = append(notes, renderer.FidelityNote{
			Level:    renderer.LevelInfo,
			Target:   target,
			Kind:     "workflow",
			Resource: name,
			Code:     renderer.CodeWorkflowMixedSteps,
			Reason:   fmt.Sprintf("workflow %q has mixed skill-ref and inline-body steps", name),
		})
	}
	return []TargetPrimitive{p}, notes
}

// lowerSimpleSkill emits a single skill primitive whose body is the concatenation
// of each step's body under a ## <step-name> heading.
func lowerSimpleSkill(wf *ast.WorkflowConfig, name, target string) ([]TargetPrimitive, []renderer.FidelityNote) {
	var body strings.Builder
	for _, step := range wf.Steps {
		fmt.Fprintf(&body, "## %s\n\n", step.Name)
		if step.Body != "" {
			body.WriteString(step.Body)
			body.WriteString("\n\n")
		}
	}

	p := TargetPrimitive{Kind: "skill", ID: name, Content: body.String()}

	notes := []renderer.FidelityNote{{
		Level:    renderer.LevelInfo,
		Target:   target,
		Kind:     "workflow",
		Resource: name,
		Code:     renderer.CodeWorkflowSimpleToSections,
		Reason:   fmt.Sprintf("workflow %q rendered as single skill with step sections for %s", name, target),
	}}
	return []TargetPrimitive{p}, notes
}

// buildWorkflowRule constructs an optional rule primitive that activates the
// workflow automatically. Returns nil when neither always-apply nor paths is set.
func buildWorkflowRule(wf *ast.WorkflowConfig, name, target string) *TargetPrimitive {
	if (wf.AlwaysApply == nil || !*wf.AlwaysApply) && wf.Paths.Len() == 0 {
		return nil
	}

	var ruleBody strings.Builder
	ruleBody.WriteString("```yaml\n")
	ruleBody.WriteString("x-xcaffold:\n")
	fmt.Fprintf(&ruleBody, "  compiled-from: workflow\n")
	fmt.Fprintf(&ruleBody, "  workflow-name: %s\n", name)
	ruleBody.WriteString("```\n\n")

	if wf.AlwaysApply != nil && *wf.AlwaysApply {
		fmt.Fprintf(&ruleBody, "This workflow (%s) is always active. Invoke `/%s` to begin.\n", name, name)
	} else if wf.Paths.Len() > 0 {
		fmt.Fprintf(&ruleBody, "This workflow (%s) activates for paths: %s. Invoke `/%s` to begin.\n",
			name, strings.Join(wf.Paths.Values, ", "), name)
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
		if step.Body != "" {
			content.WriteString(step.Body)
			content.WriteString("\n\n")
		}
	}

	p := TargetPrimitive{
		Kind:    "custom-command",
		ID:      name,
		Path:    path,
		Content: content.String(),
	}

	note := renderer.FidelityNote{
		Level:    renderer.LevelInfo,
		Target:   target,
		Kind:     "workflow",
		Resource: name,
		Code:     renderer.CodeWorkflowLoweredToCustomCommand,
		Reason:   fmt.Sprintf("workflow %q lowered to custom command at %s", name, path),
	}
	return []TargetPrimitive{p}, []renderer.FidelityNote{note}
}
