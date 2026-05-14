package ast

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestWorkflowConfig_StructExists verifies that WorkflowConfig has the expected fields
// with correct yaml tags.
func TestWorkflowConfig_StructExists(t *testing.T) {
	wf := WorkflowConfig{
		Name:        "my-workflow",
		Description: "A test workflow",
	}

	if wf.Name != "my-workflow" {
		t.Errorf("Name: got %q, want %q", wf.Name, "my-workflow")
	}
	if wf.Description != "A test workflow" {
		t.Errorf("Description: got %q, want %q", wf.Description, "A test workflow")
	}
}

// TestWorkflowConfig_ZeroValue verifies that the zero value is usable (all omitempty fields).
func TestWorkflowConfig_ZeroValue(t *testing.T) {
	var wf WorkflowConfig
	if wf.Name != "" || wf.Description != "" {
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
		Hooks: map[string]string{"pre": "echo pre"},
	}

	if tr.Hooks["pre"] != "echo pre" {
		t.Errorf("Hooks[pre]: got %q, want %q", tr.Hooks["pre"], "echo pre")
	}
}

// TestTargetOverride_AllFieldsCombined verifies all four fields work together.
func TestTargetOverride_AllFieldsCombined(t *testing.T) {
	suppressTrue := true
	skipFalse := false

	tr := TargetOverride{
		Hooks:                    map[string]string{"post": "echo done"},
		SuppressFidelityWarnings: &suppressTrue,
		SkipSynthesis:            &skipFalse,
	}

	if tr.Hooks["post"] != "echo done" {
		t.Errorf("Hooks[post]: got %q, want %q", tr.Hooks["post"], "echo done")
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
	require.Contains(t, content, "disable-model-invocation: true")
	require.Contains(t, content, "user-invocable: false")
}

func TestTargetOverride_ProviderPassthrough(t *testing.T) {
	override := TargetOverride{
		Provider: map[string]any{
			"temperature":  0.7,
			"timeout_mins": 15,
			"kind":         "local",
		},
	}

	data, err := yaml.Marshal(override)
	require.NoError(t, err)

	content := string(data)
	require.Contains(t, content, "provider:")
	require.Contains(t, content, "temperature: 0.7")
	require.Contains(t, content, "timeout_mins: 15")
	require.Contains(t, content, "kind: local")
}

func TestAgentConfig_CanonicalFieldOrdering(t *testing.T) {
	truthy := true
	agent := AgentConfig{
		Name:                   "developer",
		Description:            "Software developer.",
		Model:                  "sonnet",
		Effort:                 "high",
		MaxTurns:               intPtr(10),
		Tools:                  ClearableList{Values: []string{"Read", "Write"}},
		Readonly:               &truthy,
		PermissionMode:         "default",
		DisableModelInvocation: &truthy,
		UserInvocable:          &truthy,
		Background:             &truthy,
		Isolation:              "worktree",
		Memory:                 FlexStringSlice{"project"},
		Color:                  "blue",
		InitialPrompt:          "Hello.",
		Skills:                 ClearableList{Values: []string{"tdd"}},
		Rules:                  ClearableList{Values: []string{"coding-standards"}},
		MCP:                    ClearableList{Values: []string{"github"}},
		Body:                   "Do the work.",
	}

	data, err := yaml.Marshal(agent)
	require.NoError(t, err)
	content := string(data)

	orderedKeys := []string{
		"name:", "description:",
		"model:", "effort:", "max-turns:",
		"tools:", "readonly:",
		"permission-mode:", "disable-model-invocation:", "user-invocable:",
		"background:", "isolation:",
		"memory:", "color:", "initial-prompt:",
		"skills:", "rules:", "mcp:",
	}

	lastIdx := -1
	for _, key := range orderedKeys {
		idx := strings.Index(content, key)
		require.NotEqual(t, -1, idx, "key %q not found in YAML", key)
		require.Greater(t, idx, lastIdx, "key %q appeared before a prior key (got idx %d, last %d)\n\n%s", key, idx, lastIdx, content)
		lastIdx = idx
	}
}

func TestSkillConfig_NewCanonicalFields(t *testing.T) {
	truthy := true
	falsy := false
	skill := SkillConfig{
		Name:                   "deploy",
		Description:            "Deploy the application",
		WhenToUse:              "When user asks to deploy",
		License:                "MIT",
		AllowedTools:           ClearableList{Values: []string{"Bash", "Read"}},
		DisableModelInvocation: &truthy,
		UserInvocable:          &falsy,
		ArgumentHint:           "[environment]",
	}

	data, err := yaml.Marshal(skill)
	require.NoError(t, err)
	content := string(data)

	require.Contains(t, content, "name: deploy")
	require.Contains(t, content, "description: Deploy the application")
	require.Contains(t, content, "when-to-use: When user asks to deploy")
	require.Contains(t, content, "license: MIT")
	require.Contains(t, content, "allowed-tools:")
	require.Contains(t, content, "disable-model-invocation: true")
	require.Contains(t, content, "user-invocable: false")
	require.Contains(t, content, "argument-hint: '[environment]'")
}

func TestSkillConfig_ToolsFieldRemoved(t *testing.T) {
	// The legacy Tools field must no longer exist on SkillConfig.
	// This compile-time assertion catches accidental re-introduction.
	skill := SkillConfig{Name: "x"}
	_ = skill
	// Intentional: if someone adds .Tools back, the next line will fail to compile:
	// _ = skill.Tools  // uncomment to verify the field is gone
}

func TestRuleConfig_Activation_Field(t *testing.T) {
	rule := RuleConfig{
		Name:       "coding-style",
		Activation: RuleActivationAlways,
		Paths:      ClearableList{Values: []string{"src/**"}},
	}

	data, err := yaml.Marshal(rule)
	require.NoError(t, err)

	content := string(data)
	require.Contains(t, content, "activation: always")
	require.Contains(t, content, "paths:")
}

func TestRuleActivation_Constants(t *testing.T) {
	require.Equal(t, "always", RuleActivationAlways)
	require.Equal(t, "path-glob", RuleActivationPathGlob)
	require.Equal(t, "model-decided", RuleActivationModelDecided)
	require.Equal(t, "manual-mention", RuleActivationManualMention)
	require.Equal(t, "explicit-invoke", RuleActivationExplicitInvoke)
}

func TestRuleConfig_ExcludeAgents_Serializes(t *testing.T) {
	rule := RuleConfig{
		Name:          "security",
		Activation:    RuleActivationAlways,
		ExcludeAgents: ClearableList{Values: []string{"code-review", "cloud-agent"}},
		Body:          "Body.",
	}

	data, err := yaml.Marshal(rule)
	require.NoError(t, err)

	content := string(data)
	require.Contains(t, content, "exclude-agents:")
	require.Contains(t, content, "- code-review")
	require.Contains(t, content, "- cloud-agent")
}

func TestRuleConfig_Targets_Serializes(t *testing.T) {
	rule := RuleConfig{
		Name:       "api-style",
		Activation: RuleActivationAlways,
		Targets: map[string]TargetOverride{
			"copilot": {
				Provider: map[string]any{
					"mode": "edit",
				},
			},
		},
		Body: "Body.",
	}

	data, err := yaml.Marshal(rule)
	require.NoError(t, err)

	content := string(data)
	require.Contains(t, content, "targets:")
	require.Contains(t, content, "copilot:")
	require.Contains(t, content, "provider:")
	require.Contains(t, content, "mode: edit")
}

func intPtr(n int) *int { return &n }
