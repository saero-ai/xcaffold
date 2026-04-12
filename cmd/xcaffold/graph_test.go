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
