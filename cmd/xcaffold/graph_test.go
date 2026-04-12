package main

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraphAll_MutualExclusion_WithGlobal(t *testing.T) {
	graphAll = true
	globalFlag = true
	defer func() {
		graphAll = false
		globalFlag = false
	}()

	err := runGraph(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestGraphAll_MutualExclusion_WithProject(t *testing.T) {
	graphAll = true
	graphProject = "some-project"
	defer func() {
		graphAll = false
		graphProject = ""
	}()

	err := runGraph(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

// TestBuildGraph_HooksAndWorkflows verifies that hooks and workflows appear as
// graph nodes with the correct kind and labels.
func TestBuildGraph_HooksAndWorkflows(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Hooks: ast.HookConfig{
				"PreToolUse": {
					{
						Matcher: "Bash",
						Hooks:   []ast.HookHandler{{Type: "command", Command: "echo pre"}},
					},
				},
				"Stop": {
					{
						Hooks: []ast.HookHandler{{Type: "command", Command: "echo stop"}},
					},
				},
			},
			Workflows: map[string]ast.WorkflowConfig{
				"deploy": {
					Description: "Run deployment pipeline",
				},
				"release": {},
			},
		},
	}

	g := buildGraph(config)

	// Collect hook and workflow nodes.
	hookIDs := map[string]bool{}
	workflowIDs := map[string]string{} // id -> label
	for _, n := range g.Nodes {
		switch n.Kind {
		case "hook":
			hookIDs[n.ID] = true
		case "workflow":
			workflowIDs[n.ID] = n.Label
		}
	}

	// Both hook events must appear.
	require.True(t, hookIDs["hook:PreToolUse"], "expected hook:PreToolUse node")
	require.True(t, hookIDs["hook:Stop"], "expected hook:Stop node")
	assert.Len(t, hookIDs, 2)

	// Workflows: description used as label when present, id otherwise.
	require.Contains(t, workflowIDs, "workflow:deploy")
	assert.Equal(t, "Run deployment pipeline", workflowIDs["workflow:deploy"])
	require.Contains(t, workflowIDs, "workflow:release")
	assert.Equal(t, "release", workflowIDs["workflow:release"])
	assert.Len(t, workflowIDs, 2)
}

// TestBuildGraph_ExcludesInheritedAgents verifies that inherited resources
// (from an extends: global config) do not appear in a project-scope graph
// after StripInherited is applied.
func TestBuildGraph_ExcludesInheritedAgents(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"local-agent": {
					Description: "local only",
					Inherited:   false,
				},
				"inherited-agent": {
					Description: "comes from global base",
					Inherited:   true,
				},
			},
		},
	}

	// Before stripping, both agents appear in the graph.
	gBefore := buildGraph(config)
	var agentNodesBefore []string
	for _, n := range gBefore.Nodes {
		if n.Kind == kindAgent {
			agentNodesBefore = append(agentNodesBefore, n.ID)
		}
	}
	require.Len(t, agentNodesBefore, 2, "expected both agents before stripping")

	// Simulate what parseGraphData does for non-global scopes.
	config.StripInherited()
	gAfter := buildGraph(config)

	var agentNodesAfter []string
	for _, n := range gAfter.Nodes {
		if n.Kind == kindAgent {
			agentNodesAfter = append(agentNodesAfter, n.ID)
		}
	}

	require.Len(t, agentNodesAfter, 1, "expected only local agent after stripping inherited")
	assert.Equal(t, "agent:local-agent", agentNodesAfter[0])
}
