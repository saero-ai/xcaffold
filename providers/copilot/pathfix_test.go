package copilot_test

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/providers/copilot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCopilotRenderer_OutputDir verifies that OutputDir returns ".github" (not
// ".github/instructions"), so that apply.go can correctly join relative paths.
func TestCopilotRenderer_OutputDir_PathFix(t *testing.T) {
	r := copilot.New()
	assert.Equal(t, ".github", r.OutputDir(),
		"OutputDir must return \".github\", not \".github/instructions\"")
}

// TestCopilotRenderer_RulePaths_PathFix verifies that compiled rule paths are
// relative to OutputDir (i.e. "instructions/<id>.instructions.md", no ".github/" prefix).
func TestCopilotRenderer_RulePaths_PathFix(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"go-style": {
					Activation: ast.RuleActivationAlways,
					Body:       "Follow Go conventions.",
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	_, ok := out.Files["instructions/go-style.instructions.md"]
	assert.True(t, ok, "rule path must be relative: \"instructions/<id>.instructions.md\"")

	_, bad := out.Files[".github/instructions/go-style.instructions.md"]
	assert.False(t, bad, "rule path must NOT include \".github/\" prefix")
}

// TestCopilotRenderer_AgentPaths_PathFix verifies that compiled agent paths are
// relative to OutputDir (i.e. "agents/<id>.agent.md", no ".github/" prefix).
func TestCopilotRenderer_AgentPaths_PathFix(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"my-agent": {
					Name:        "My Agent",
					Description: "A test agent.",
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	_, ok := out.Files["agents/my-agent.agent.md"]
	assert.True(t, ok, "agent path must be relative: \"agents/<id>.agent.md\"")

	_, bad := out.Files[".github/agents/my-agent.agent.md"]
	assert.False(t, bad, "agent path must NOT include \".github/\" prefix")
}

// TestCopilotRenderer_SkillPaths_PathFix verifies that compiled skill paths are
// relative to OutputDir (i.e. "skills/<id>/SKILL.md", no ".github/" prefix).
func TestCopilotRenderer_SkillPaths_PathFix(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"tdd": {
					Name:        "tdd",
					Description: "Test-driven development.",
					Body:        "Write tests first.",
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	_, ok := out.Files["skills/tdd/SKILL.md"]
	assert.True(t, ok, "skill path must be relative: \"skills/<id>/SKILL.md\"")

	_, bad := out.Files[".github/skills/tdd/SKILL.md"]
	assert.False(t, bad, "skill path must NOT include \".github/\" prefix")
}

// TestCopilotRenderer_HookPaths_PathFix verifies that compiled hook paths are
// relative to OutputDir (i.e. "hooks/xcaffold-hooks.json", no ".github/" prefix).
func TestCopilotRenderer_HookPaths_PathFix(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		Hooks: map[string]ast.NamedHookConfig{
			"default": {
				Name: "default",
				Events: ast.HookConfig{
					"PreToolUse": []ast.HookMatcherGroup{
						{Hooks: []ast.HookHandler{{Type: "command", Command: "check.sh"}}},
					},
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	_, ok := out.Files["hooks/xcaffold-hooks.json"]
	assert.True(t, ok, "hook path must be relative: \"hooks/xcaffold-hooks.json\"")

	_, bad := out.Files[".github/hooks/xcaffold-hooks.json"]
	assert.False(t, bad, "hook path must NOT include \".github/\" prefix")
}

// TestCopilotRenderer_Workflow_PromptFile_NoPathDoubling verifies that a workflow
// lowered to a prompt-file primitive is stored with a path relative to OutputDir
// ("prompts/<name>.prompt.md") and NOT with the full provider prefix
// (".github/prompts/<name>.prompt.md"). Without the fix, apply.go would join
// OutputDir + ".github/prompts/my-prompt.prompt.md", producing
// ".github/.github/prompts/my-prompt.prompt.md" on disk.
func TestCopilotRenderer_Workflow_PromptFile_NoPathDoubling(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Workflows: map[string]ast.WorkflowConfig{
				"my-prompt": {
					Name: "my-prompt",
					Steps: []ast.WorkflowStep{
						{Name: "step-one", Body: "Some prompt body."},
					},
					Targets: map[string]ast.TargetOverride{
						"copilot": {
							Provider: map[string]interface{}{
								"lowering-strategy": "prompt-file",
							},
						},
					},
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, t.TempDir())
	require.NoError(t, err)

	// Relative path (correct — apply.go will prepend .github/).
	_, ok := out.Files["prompts/my-prompt.prompt.md"]
	assert.True(t, ok, "workflow prompt-file path must be relative: \"prompts/my-prompt.prompt.md\"")

	// Absolute path with provider prefix (the doubled path — must not appear).
	_, doubled := out.Files[".github/prompts/my-prompt.prompt.md"]
	assert.False(t, doubled, "workflow prompt-file path must NOT include \".github/\" prefix")
}

// TestCopilotRenderer_InstructionPath_PathFix verifies that the copilot-instructions.md
// path is relative to OutputDir (i.e. "copilot-instructions.md", no ".github/" prefix).
func TestCopilotRenderer_InstructionPath_PathFix(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "test-proj"},
		ResourceScope: ast.ResourceScope{
			Contexts: map[string]ast.ContextConfig{
				"root": {Body: "Project-wide instructions."},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	_, ok := out.Files["copilot-instructions.md"]
	assert.True(t, ok, "instructions path must be relative: \"copilot-instructions.md\"")

	_, bad := out.Files[".github/copilot-instructions.md"]
	assert.False(t, bad, "instructions path must NOT include \".github/\" prefix")
}
