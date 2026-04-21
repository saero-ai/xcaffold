package shared

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
)

func TestLowerWorkflows_EmptyWorkflows(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	got, notes := LowerWorkflows(config, "gemini")
	if got != config {
		t.Error("expected same pointer returned when no workflows present")
	}
	if len(notes) != 0 {
		t.Errorf("expected no fidelity notes, got %d", len(notes))
	}
}

func TestLowerWorkflows_DoesNotMutateInput(t *testing.T) {
	wf := ast.WorkflowConfig{
		Name:         "my-workflow",
		Description:  "A workflow",
		Instructions: "Do the thing.",
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
	existing := ast.RuleConfig{Instructions: "existing rule"}
	config := &ast.XcaffoldConfig{}
	config.ResourceScope.Rules = map[string]ast.RuleConfig{"keep-me": existing}
	config.ResourceScope.Workflows = map[string]ast.WorkflowConfig{
		"wf1": {Name: "wf1", Instructions: "workflow body"},
	}

	out, _ := LowerWorkflows(config, "gemini")
	if _, ok := out.ResourceScope.Rules["keep-me"]; !ok {
		t.Error("LowerWorkflows must preserve pre-existing rules")
	}
}
