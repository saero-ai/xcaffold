package ast

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestWorkflowConfig_ApiVersion_RoundTrip(t *testing.T) {
	wf := WorkflowConfig{
		ApiVersion:  "workflow/v1",
		Name:        "code-review",
		Description: "Multi-step PR review procedure.",
	}

	data, err := yaml.Marshal(wf)
	require.NoError(t, err)
	content := string(data)
	require.Contains(t, content, "api-version: workflow/v1")
	require.Contains(t, content, "name: code-review")
}

func TestWorkflowConfig_Steps_RoundTrip(t *testing.T) {
	wf := WorkflowConfig{
		ApiVersion: "workflow/v1",
		Name:       "deploy-pipeline",
		Steps: []WorkflowStep{
			{Name: "preflight", Description: "Check preconditions.", Instructions: "Verify infra is ready."},
			{Name: "build", InstructionsFile: "xcf/workflows/deploy-pipeline/02-build.md"},
		},
	}

	data, err := yaml.Marshal(wf)
	require.NoError(t, err)
	content := string(data)
	require.Contains(t, content, "name: preflight")
	require.Contains(t, content, "name: build")
	require.Contains(t, content, "instructions-file: xcf/workflows/deploy-pipeline/02-build.md")
}

func TestWorkflowConfig_Targets_RoundTrip(t *testing.T) {
	wf := WorkflowConfig{
		ApiVersion: "workflow/v1",
		Name:       "code-review",
		Targets: map[string]TargetOverride{
			"claude": {
				Provider: map[string]any{"lowering-strategy": "rule-plus-skill"},
			},
			"copilot": {
				Provider: map[string]any{"lowering-strategy": "prompt-file"},
			},
		},
	}

	data, err := yaml.Marshal(wf)
	require.NoError(t, err)
	content := string(data)
	require.Contains(t, content, "lowering-strategy: rule-plus-skill")
	require.Contains(t, content, "lowering-strategy: prompt-file")
}

func TestWorkflowStep_MutualExclusion_BothSet_IsDetectableAtParseTime(t *testing.T) {
	step := WorkflowStep{
		Name:             "analyze",
		Instructions:     "Inline body.",
		InstructionsFile: "xcf/workflows/foo/01-analyze.md",
	}
	data, err := yaml.Marshal(step)
	require.NoError(t, err)
	content := string(data)
	require.Contains(t, content, "instructions: Inline body.")
	require.Contains(t, content, "instructions-file:")
}
