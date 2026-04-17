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
//  1. Antigravity with promote-rules-to-workflows: true → native workflow primitive.
//  2. Target provider lowering-strategy: prompt-file   → .github/prompts/*.prompt.md
//  3. Target provider lowering-strategy: custom-command → .gemini/commands/*.md
//  4. Claude / Cursor (default or lowering-strategy: rule-plus-skill)
//     → one rule primitive (with provenance marker) + one skill per step.
//  5. No determinable strategy → empty primitives + CodeWorkflowNoNativeTarget note.
func TranslateWorkflow(wf *ast.WorkflowConfig, target string) ([]TargetPrimitive, []renderer.FidelityNote) {
	name := wf.Name
	if name == "" {
		name = "unnamed"
	}

	// Antigravity: check for native workflow promotion.
	if target == "antigravity" {
		if override, ok := wf.Targets["antigravity"]; ok {
			if v, ok := override.Provider["promote-rules-to-workflows"]; ok {
				if promote, _ := v.(bool); promote {
					return lowerAntigravityNative(wf, name, target)
				}
			}
		}
	}

	strategy := loweringStrategy(wf, target)

	switch {
	case strategy == "prompt-file":
		return lowerPromptFile(wf, name, target)
	case strategy == "custom-command":
		return lowerCustomCommand(wf, name, target)
	case strategy == "rule-plus-skill":
		return lowerRulePlusSkill(wf, name, target)
	case target == "claude", target == "gemini", target == "copilot":
		// Claude, Gemini, and Copilot default to rule-plus-skill when no strategy is set.
		return lowerRulePlusSkill(wf, name, target)
	default:
		note := renderer.NewNote(
			renderer.LevelWarning,
			target,
			"workflow",
			name,
			"",
			renderer.CodeWorkflowNoNativeTarget,
			fmt.Sprintf("workflow %q: no native lowering strategy for target %q", name, target),
			"Add a targets.<target>.provider.lowering-strategy to the workflow config.",
		)
		return nil, []renderer.FidelityNote{note}
	}
}

// lowerRulePlusSkill emits one rule with a provenance marker plus one skill per step.
func lowerRulePlusSkill(wf *ast.WorkflowConfig, name, target string) ([]TargetPrimitive, []renderer.FidelityNote) {
	stepNames := make([]string, len(wf.Steps))
	skillIDs := make([]string, len(wf.Steps))
	for i, step := range wf.Steps {
		stepNames[i] = step.Name
		skillIDs[i] = stepSkillID(name, i, step.Name)
	}

	// Build provenance marker as inline YAML block.
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

	// Rule body: provenance block + invocation instruction.
	var ruleBody strings.Builder
	ruleBody.WriteString("```yaml\n")
	ruleBody.WriteString(marker.String())
	ruleBody.WriteString("```\n\n")
	fmt.Fprintf(&ruleBody, "Run steps in order: %s.\n", strings.Join(skillIDs, " → "))

	rulePrimitive := TargetPrimitive{
		Kind:    "rule",
		ID:      name + "-workflow",
		Content: ruleBody.String(),
	}

	primitives := []TargetPrimitive{rulePrimitive}

	// One skill per step.
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

	note := renderer.NewNote(
		renderer.LevelWarning,
		target,
		"workflow",
		name,
		"",
		renderer.CodeWorkflowLoweredToRulePlusSkill,
		fmt.Sprintf("workflow %q lowered to rule+skill; %s has no native workflow primitive", name, target),
		"Accept the rule-plus-skill lowering or add a target with a native workflow primitive.",
	)
	return primitives, []renderer.FidelityNote{note}
}

// lowerAntigravityNative emits a single workflow primitive with step bodies
// concatenated under ## <step-name> headers.
func lowerAntigravityNative(wf *ast.WorkflowConfig, name, target string) ([]TargetPrimitive, []renderer.FidelityNote) {
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

	note := renderer.NewNote(
		renderer.LevelInfo,
		target,
		"workflow",
		name,
		"",
		renderer.CodeWorkflowLoweredToNative,
		fmt.Sprintf("workflow %q emitted as native antigravity workflow", name),
		"",
	)
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

	note := renderer.NewNote(
		renderer.LevelInfo,
		target,
		"workflow",
		name,
		"",
		renderer.CodeWorkflowLoweredToPromptFile,
		fmt.Sprintf("workflow %q lowered to prompt file at %s", name, path),
		"",
	)
	return []TargetPrimitive{p}, []renderer.FidelityNote{note}
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

	note := renderer.NewNote(
		renderer.LevelInfo,
		target,
		"workflow",
		name,
		"",
		renderer.CodeWorkflowLoweredToCustomCommand,
		fmt.Sprintf("workflow %q lowered to custom command at %s", name, path),
		"",
	)
	return []TargetPrimitive{p}, []renderer.FidelityNote{note}
}
