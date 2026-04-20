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

func TestGraph_BlueprintFlag_MutualExclusion_WithGlobal(t *testing.T) {
	graphBlueprintFlag = "my-blueprint"
	globalFlag = true
	defer func() {
		graphBlueprintFlag = ""
		globalFlag = false
	}()

	err := runGraph(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--blueprint cannot be used with --global")
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
		Hooks: map[string]ast.NamedHookConfig{
			"default": {
				Name: "default",
				Events: ast.HookConfig{
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
			},
		},
		ResourceScope: ast.ResourceScope{
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

// TestBuildGraph_PolicyNodes verifies that policy entries appear as graph nodes
// with the correct kind, ID, label, and meta fields.
func TestBuildGraph_PolicyNodes(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Policies: map[string]ast.PolicyConfig{
				"no-bash": {
					Name:     "No Bash",
					Severity: "error",
					Target:   "agent",
				},
				"require-memory": {
					Name:     "Require Memory",
					Severity: "warn",
					Target:   "agent",
				},
			},
		},
	}

	g := buildGraph(config)

	policyNodes := map[string]graphNode{}
	for _, n := range g.Nodes {
		if n.Kind == kindPolicy {
			policyNodes[n.ID] = n
		}
	}

	require.Len(t, policyNodes, 2, "expected two policy nodes")

	noBash, ok := policyNodes["policy:no-bash"]
	require.True(t, ok, "expected policy:no-bash node")
	assert.Equal(t, "no-bash", noBash.Label)
	assert.Equal(t, "error", noBash.Meta["severity"])
	assert.Equal(t, "agent", noBash.Meta["target"])

	reqMem, ok := policyNodes["policy:require-memory"]
	require.True(t, ok, "expected policy:require-memory node")
	assert.Equal(t, "require-memory", reqMem.Label)
	assert.Equal(t, "warn", reqMem.Meta["severity"])
	assert.Equal(t, "agent", reqMem.Meta["target"])
}

// TestRenderTerminalHeader_PolicyCount verifies that policy nodes contribute to
// the header summary count when present.
func TestRenderTerminalHeader_PolicyCount(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Policies: map[string]ast.PolicyConfig{
				"p1": {Severity: "error", Target: "agent"},
				"p2": {Severity: "warn", Target: "skill"},
			},
		},
	}

	g := buildGraph(config)
	header := renderTerminalHeader(g)

	assert.Contains(t, header, "2 policies", "header must report policy count")
}

// TestRenderDOT_PolicyColor verifies that policy nodes are rendered with a
// distinct color in DOT output.
func TestRenderDOT_PolicyColor(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Policies: map[string]ast.PolicyConfig{
				"audit-log": {Severity: "error", Target: "agent"},
			},
		},
	}

	g := buildGraph(config)
	dot := renderDOT(g)

	assert.Contains(t, dot, "policy_audit_log", "DOT must include sanitised policy node ID")
	assert.Contains(t, dot, "#E67E22", "DOT must use orange fill for policy nodes")
}
