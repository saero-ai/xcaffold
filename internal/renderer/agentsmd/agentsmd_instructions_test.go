package agentsmd_test

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/renderer/agentsmd"
	"github.com/stretchr/testify/require"
)

// TestAgentsmdRenderer_ProjectInstructions_NestedFiles verifies that the agentsmd
// renderer emits a root AGENTS.md and a scoped AGENTS.md for each InstructionsScope.
func TestAgentsmdRenderer_ProjectInstructions_NestedFiles(t *testing.T) {
	r := agentsmd.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:         "test",
			Instructions: "Root context.",
			InstructionsScopes: []ast.InstructionsScope{
				{Path: "packages/worker", Instructions: "Worker context.", MergeStrategy: "closest-wins"},
			},
		},
	}
	out, _, err := r.Compile(config, "")
	require.NoError(t, err)
	require.Contains(t, out.Files, "AGENTS.md")
	require.Contains(t, out.Files, "packages/worker/AGENTS.md")
	require.Contains(t, out.Files["AGENTS.md"], "Root context.")
	require.Contains(t, out.Files["packages/worker/AGENTS.md"], "Worker context.")
}

// TestAgentsmdRenderer_ProjectInstructions_ConcatScopeIsPreFlattened verifies that
// concat-tagged scopes are pre-flattened (root + scope) into the child AGENTS.md
// and that an INSTRUCTIONS_CLOSEST_WINS_FORCED_CONCAT warning note is emitted.
func TestAgentsmdRenderer_ProjectInstructions_ConcatScopeIsPreFlattened(t *testing.T) {
	r := agentsmd.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:         "test",
			Instructions: "Root context.",
			InstructionsScopes: []ast.InstructionsScope{
				{Path: "packages/worker", Instructions: "Worker context.", MergeStrategy: "concat"},
			},
		},
	}
	out, notes, err := r.Compile(config, "")
	require.NoError(t, err)
	worker := out.Files["packages/worker/AGENTS.md"]
	require.Contains(t, worker, "Root context.")
	require.Contains(t, worker, "Worker context.")
	require.NotEmpty(t, notes)
	var found *renderer.FidelityNote
	for i := range notes {
		if notes[i].Code == "INSTRUCTIONS_CLOSEST_WINS_FORCED_CONCAT" {
			found = &notes[i]
			break
		}
	}
	require.NotNil(t, found, "expected INSTRUCTIONS_CLOSEST_WINS_FORCED_CONCAT note")
	require.Equal(t, renderer.LevelWarning, found.Level)
}

// TestAgentsmdRenderer_ProjectInstructions_MergesWithRuleAggregatedRoot verifies that
// renderProjectInstructions appends project.instructions to the rule-aggregated AGENTS.md
// instead of overwriting it. Both rule content and project instructions must appear.
func TestAgentsmdRenderer_ProjectInstructions_MergesWithRuleAggregatedRoot(t *testing.T) {
	r := agentsmd.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"security-policy": {Instructions: "Never expose secrets."},
			},
		},
		Project: &ast.ProjectConfig{
			Name:         "test",
			Instructions: "Project-level guidance here.",
		},
	}
	out, _, err := r.Compile(config, "")
	require.NoError(t, err)
	root := out.Files["AGENTS.md"]
	require.Contains(t, root, "Never expose secrets.", "rule content must not be discarded")
	require.Contains(t, root, "Project-level guidance here.", "project instructions must be present")
}

// TestAgentsmdRenderer_ProjectInstructions_NeverEmitsCopilotFlatFile verifies that
// the agentsmd nested-tree renderer never emits the Copilot flat file path.
func TestAgentsmdRenderer_ProjectInstructions_NeverEmitsCopilotFlatFile(t *testing.T) {
	// agentsmd is the nested-tree renderer; it must NEVER emit .github/copilot-instructions.md.
	r := agentsmd.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "test", Instructions: "Root."},
	}
	out, _, err := r.Compile(config, "")
	require.NoError(t, err)
	_, hasFlat := out.Files[".github/copilot-instructions.md"]
	require.False(t, hasFlat, "agentsmd must not emit Copilot flat file")
}
