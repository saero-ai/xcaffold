package shared

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
)

func TestLowerWorkflows_EmptyWorkflows(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	got, directFiles, notes := LowerWorkflows(config, "gemini")
	if got != config {
		t.Error("expected same pointer returned when no workflows present")
	}
	if len(directFiles) != 0 {
		t.Errorf("expected no direct files, got %d", len(directFiles))
	}
	if len(notes) != 0 {
		t.Errorf("expected no fidelity notes, got %d", len(notes))
	}
}

func TestLowerWorkflows_DoesNotMutateInput(t *testing.T) {
	wf := ast.WorkflowConfig{
		Name:        "my-workflow",
		Description: "A workflow",
		Body:        "Do the thing.",
	}
	config := &ast.XcaffoldConfig{}
	config.ResourceScope.Workflows = map[string]ast.WorkflowConfig{
		"my-workflow": wf,
	}

	original := len(config.ResourceScope.Rules)
	LowerWorkflows(config, "gemini")
	if len(config.ResourceScope.Rules) != original {
		t.Error("LowerWorkflows must not mutate the input config")
	}
}

func TestLowerWorkflows_PreservesExistingRules(t *testing.T) {
	existing := ast.RuleConfig{Body: "existing rule"}
	config := &ast.XcaffoldConfig{}
	config.ResourceScope.Rules = map[string]ast.RuleConfig{"keep-me": existing}
	config.ResourceScope.Workflows = map[string]ast.WorkflowConfig{
		"wf1": {Name: "wf1", Body: "workflow body"},
	}

	out, _, _ := LowerWorkflows(config, "gemini")
	if _, ok := out.ResourceScope.Rules["keep-me"]; !ok {
		t.Error("LowerWorkflows must preserve pre-existing rules")
	}
}

func TestLowerWorkflows_CustomCommandPrimitive(t *testing.T) {
	wf := ast.WorkflowConfig{
		Name: "my-cmd",
		Steps: []ast.WorkflowStep{
			{Name: "step-one", Body: "do the thing"},
		},
		Targets: map[string]ast.TargetOverride{
			"gemini": {
				Provider: map[string]interface{}{
					"lowering-strategy": "custom-command",
				},
			},
		},
	}
	config := &ast.XcaffoldConfig{}
	config.ResourceScope.Workflows = map[string]ast.WorkflowConfig{
		"my-cmd": wf,
	}

	_, directFiles, notes := LowerWorkflows(config, "gemini")
	if len(notes) == 0 {
		t.Error("expected at least one fidelity note")
	}
	expectedPath := ".gemini/commands/my-cmd.md"
	if _, ok := directFiles[expectedPath]; !ok {
		t.Errorf("expected direct file at %q, got keys: %v", expectedPath, mapKeys(directFiles))
	}
}

func TestLowerWorkflows_PromptFilePrimitive(t *testing.T) {
	wf := ast.WorkflowConfig{
		Name: "my-prompt",
		Steps: []ast.WorkflowStep{
			{Name: "step-one", Body: "some prompt body"},
		},
		Targets: map[string]ast.TargetOverride{
			"copilot": {
				Provider: map[string]interface{}{
					"lowering-strategy": "prompt-file",
				},
			},
		},
	}
	config := &ast.XcaffoldConfig{}
	config.ResourceScope.Workflows = map[string]ast.WorkflowConfig{
		"my-prompt": wf,
	}

	_, directFiles, notes := LowerWorkflows(config, "copilot")
	if len(notes) == 0 {
		t.Error("expected at least one fidelity note")
	}
	expectedPath := ".github/prompts/my-prompt.prompt.md"
	if _, ok := directFiles[expectedPath]; !ok {
		t.Errorf("expected direct file at %q, got keys: %v", expectedPath, mapKeys(directFiles))
	}
}

func TestLowerWorkflows_RoutedMode(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.ResourceScope.Workflows = map[string]ast.WorkflowConfig{
		"router": {
			Name: "router",
			Body: "# Router content\nDo analysis.",
		},
	}
	merged, directFiles, notes := LowerWorkflows(config, "claude")

	// Routed mode produces a single skill, no direct files.
	if len(directFiles) != 0 {
		t.Errorf("expected no direct files for routed mode, got %d", len(directFiles))
	}
	if _, ok := merged.ResourceScope.Skills["router"]; !ok {
		t.Error("expected skill 'router' in merged output")
	}
	if len(notes) == 0 {
		t.Error("expected at least one fidelity note")
	}
	var hasRoutedNote bool
	for _, n := range notes {
		if n.Code == renderer.CodeWorkflowRoutedToSingleSkill {
			hasRoutedNote = true
		}
	}
	if !hasRoutedNote {
		t.Errorf("expected CodeWorkflowRoutedToSingleSkill note; got: %v", notes)
	}
}

func TestLowerWorkflows_ChainedMode(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.ResourceScope.Workflows = map[string]ast.WorkflowConfig{
		"lifecycle": {
			Name: "lifecycle",
			Steps: []ast.WorkflowStep{
				{Name: "design", Skill: "brainstorming"},
				{Name: "plan", Skill: "writing-plans"},
			},
		},
	}
	merged, _, notes := LowerWorkflows(config, "claude")

	if _, ok := merged.ResourceScope.Skills["lifecycle"]; !ok {
		t.Error("expected skill 'lifecycle' in merged output for chained mode")
	}
	var hasChainedNote bool
	for _, n := range notes {
		if n.Code == renderer.CodeWorkflowChainedToOrchestrator {
			hasChainedNote = true
		}
	}
	if !hasChainedNote {
		t.Errorf("expected CodeWorkflowChainedToOrchestrator note; got: %v", notes)
	}
}

func TestLowerWorkflows_AlwaysApply_EmitsRule(t *testing.T) {
	alwaysApply := true
	config := &ast.XcaffoldConfig{}
	config.ResourceScope.Workflows = map[string]ast.WorkflowConfig{
		"mandatory": {
			Name:        "mandatory",
			AlwaysApply: &alwaysApply,
			Steps: []ast.WorkflowStep{
				{Name: "lint", Body: "Run lint."},
			},
		},
	}
	merged, _, _ := LowerWorkflows(config, "claude")

	if _, ok := merged.ResourceScope.Rules["mandatory-workflow"]; !ok {
		ruleKeys := make([]string, 0)
		for k := range merged.ResourceScope.Rules {
			ruleKeys = append(ruleKeys, k)
		}
		t.Errorf("expected rule 'mandatory-workflow' for always-apply workflow; got rules: %v", ruleKeys)
	}
	if _, ok := merged.ResourceScope.Skills["mandatory"]; !ok {
		skillKeys := make([]string, 0)
		for k := range merged.ResourceScope.Skills {
			skillKeys = append(skillKeys, k)
		}
		t.Errorf("expected skill 'mandatory' for always-apply workflow; got skills: %v", skillKeys)
	}
}

// mapKeys returns the keys of m as a slice for test error reporting.
func mapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func TestLowerWorkflows_WithArtifacts_PreservesArtifacts(t *testing.T) {
	// Verify that workflows with Artifacts []string preserve those artifact
	// declarations so the provider renderer can discover and copy them later.
	config := &ast.XcaffoldConfig{}
	config.ResourceScope.Workflows = map[string]ast.WorkflowConfig{
		"my-workflow": {
			Name:        "my-workflow",
			Description: "Workflow with artifacts",
			Body:        "Do the thing.",
			Artifacts:   []string{"references", "examples"},
		},
	}

	out, _, _ := LowerWorkflows(config, "claude")

	// After lowering, the workflow should still appear in the output config
	// with its Artifacts field intact (for the renderer to process later).
	outWf, ok := out.ResourceScope.Workflows["my-workflow"]
	if !ok {
		t.Error("expected workflow 'my-workflow' to be preserved after lowering")
		return
	}

	if len(outWf.Artifacts) != 2 {
		t.Errorf("expected 2 artifacts preserved, got %d: %v", len(outWf.Artifacts), outWf.Artifacts)
	}
	if outWf.Artifacts[0] != "references" || outWf.Artifacts[1] != "examples" {
		t.Errorf("expected artifacts [references examples], got %v", outWf.Artifacts)
	}
}
