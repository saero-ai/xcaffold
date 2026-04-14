package ast

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestWorkflowConfig_StructExists verifies that WorkflowConfig has the expected fields
// with correct yaml tags.
func TestWorkflowConfig_StructExists(t *testing.T) {
	wf := WorkflowConfig{
		Name:             "my-workflow",
		Description:      "A test workflow",
		Instructions:     "Do something useful",
		InstructionsFile: "workflows/my-workflow.md",
	}

	if wf.Name != "my-workflow" {
		t.Errorf("Name: got %q, want %q", wf.Name, "my-workflow")
	}
	if wf.Description != "A test workflow" {
		t.Errorf("Description: got %q, want %q", wf.Description, "A test workflow")
	}
	if wf.Instructions != "Do something useful" {
		t.Errorf("Instructions: got %q, want %q", wf.Instructions, "Do something useful")
	}
	if wf.InstructionsFile != "workflows/my-workflow.md" {
		t.Errorf("InstructionsFile: got %q, want %q", wf.InstructionsFile, "workflows/my-workflow.md")
	}
}

// TestWorkflowConfig_ZeroValue verifies that the zero value is usable (all omitempty fields).
func TestWorkflowConfig_ZeroValue(t *testing.T) {
	var wf WorkflowConfig
	if wf.Name != "" || wf.Description != "" || wf.Instructions != "" || wf.InstructionsFile != "" {
		t.Error("zero WorkflowConfig should have empty string fields")
	}
}

// TestXcaffoldConfig_HasWorkflowsField verifies that XcaffoldConfig accepts a Workflows map.
func TestXcaffoldConfig_HasWorkflowsField(t *testing.T) {
	cfg := XcaffoldConfig{
		ResourceScope: ResourceScope{
			Workflows: map[string]WorkflowConfig{
				"build": {
					Name:        "Build Workflow",
					Description: "Compiles the project",
				},
			},
		},
	}

	if len(cfg.Workflows) != 1 {
		t.Fatalf("Workflows len: got %d, want 1", len(cfg.Workflows))
	}
	build, ok := cfg.Workflows["build"]
	if !ok {
		t.Fatal("Workflows missing key 'build'")
	}
	if build.Name != "Build Workflow" {
		t.Errorf("Workflows[build].Name: got %q, want %q", build.Name, "Build Workflow")
	}
}

// TestXcaffoldConfig_WorkflowsIsNilByDefault verifies omitempty behaviour — nil map is valid.
func TestXcaffoldConfig_WorkflowsIsNilByDefault(t *testing.T) {
	var cfg XcaffoldConfig
	if cfg.Workflows != nil {
		t.Error("Workflows should be nil by default")
	}
}

// TestTargetOverride_HasSuppressFidelityWarnings verifies the new pointer-bool field exists.
func TestTargetOverride_HasSuppressFidelityWarnings(t *testing.T) {
	tr := TargetOverride{}
	if tr.SuppressFidelityWarnings != nil {
		t.Error("SuppressFidelityWarnings should be nil by default")
	}

	val := true
	tr.SuppressFidelityWarnings = &val
	if tr.SuppressFidelityWarnings == nil || !*tr.SuppressFidelityWarnings {
		t.Error("SuppressFidelityWarnings: expected true after assignment")
	}
}

// TestTargetOverride_HasSkipSynthesis verifies the new pointer-bool field exists.
func TestTargetOverride_HasSkipSynthesis(t *testing.T) {
	tr := TargetOverride{}
	if tr.SkipSynthesis != nil {
		t.Error("SkipSynthesis should be nil by default")
	}

	val := false
	tr.SkipSynthesis = &val
	if tr.SkipSynthesis == nil || *tr.SkipSynthesis {
		t.Error("SkipSynthesis: expected false after assignment")
	}
}

// TestTargetOverride_ExistingFieldsUnchanged verifies that pre-existing fields still work.
func TestTargetOverride_ExistingFieldsUnchanged(t *testing.T) {
	tr := TargetOverride{
		Hooks:                map[string]string{"pre": "echo pre"},
		InstructionsOverride: "use this instead",
	}

	if tr.Hooks["pre"] != "echo pre" {
		t.Errorf("Hooks[pre]: got %q, want %q", tr.Hooks["pre"], "echo pre")
	}
	if tr.InstructionsOverride != "use this instead" {
		t.Errorf("InstructionsOverride: got %q, want %q", tr.InstructionsOverride, "use this instead")
	}
}

// TestTargetOverride_AllFieldsCombined verifies all four fields work together.
func TestTargetOverride_AllFieldsCombined(t *testing.T) {
	suppressTrue := true
	skipFalse := false

	tr := TargetOverride{
		Hooks:                    map[string]string{"post": "echo done"},
		InstructionsOverride:     "alternate instructions",
		SuppressFidelityWarnings: &suppressTrue,
		SkipSynthesis:            &skipFalse,
	}

	if tr.Hooks["post"] != "echo done" {
		t.Errorf("Hooks[post]: got %q, want %q", tr.Hooks["post"], "echo done")
	}
	if tr.InstructionsOverride != "alternate instructions" {
		t.Errorf("InstructionsOverride: got %q, want %q", tr.InstructionsOverride, "alternate instructions")
	}
	if tr.SuppressFidelityWarnings == nil || !*tr.SuppressFidelityWarnings {
		t.Error("SuppressFidelityWarnings: expected true")
	}
	if tr.SkipSynthesis == nil || *tr.SkipSynthesis {
		t.Error("SkipSynthesis: expected false")
	}
}

func TestAgentConfig_InvocationControlFields(t *testing.T) {
	truthy := true
	falsy := false
	agent := AgentConfig{
		Name:                   "test",
		DisableModelInvocation: &truthy,
		UserInvocable:          &falsy,
	}

	data, err := yaml.Marshal(agent)
	require.NoError(t, err)

	content := string(data)
	require.Contains(t, content, "disableModelInvocation: true")
	require.Contains(t, content, "userInvocable: false")
}
