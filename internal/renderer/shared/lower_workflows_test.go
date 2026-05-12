package shared

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
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

// mapKeys returns the keys of m as a slice for test error reporting.
func mapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
