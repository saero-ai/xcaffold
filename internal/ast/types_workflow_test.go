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
