package main

import (
	"bytes"
	"io"
	"os"
	"strings"
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

// TestRenderDOT_PolicyColor verifies that policy nodes are rendered with a
// distinct color in DOT output.

func captureGraphStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	f()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestGraph_TreeAlignment_NonLastBlock(t *testing.T) {
	dir := t.TempDir()
	memDir := dir + "/xcf/agents/test-agent/memory"
	os.MkdirAll(memDir, 0755)
	os.WriteFile(memDir+"/note.md", []byte("x"), 0644)

	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"test-agent": {
					Skills: []string{"skill-a"},
				},
			},
			Skills: map[string]ast.SkillConfig{
				"skill-a": {},
			},
		},
	}

	out := captureGraphStdout(func() { renderAgentTree(cfg, dir) })

	assert.Contains(t, out, "├── skills", "skills should use ├── when memory follows")
	assert.Contains(t, out, "└── memory", "memory should use └── as last block")

	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimRight(line, " \t")
		if trimmed == "" {
			continue
		}
		if len(trimmed) > 2 && (trimmed[2] == 0xe2 || trimmed[2] == '|') {
			assert.Equal(t, "  ", trimmed[:2], "tree connector must be at column 2: %q", trimmed)
		}
	}
}

func TestGraph_TreeAlignment_LastBlock(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"solo": {Skills: []string{"s1"}},
			},
			Skills: map[string]ast.SkillConfig{"s1": {}},
		},
	}

	dir := t.TempDir()
	out := captureGraphStdout(func() { renderAgentTree(cfg, dir) })

	assert.Contains(t, out, "└── skills", "sole block should use └──")
	assert.NotContains(t, out, "├── skills")
}

func TestGraph_Header_OmitsZeroMCP(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "myproj"},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{"a": {}},
		},
	}

	g := buildGraph(cfg)
	header := renderTerminalHeader(g)
	assert.NotContains(t, header, "mcp", "zero MCP should be omitted from header")
	assert.Contains(t, header, "1 agent")
}

func TestGraph_Header_PluralizeMCP(t *testing.T) {
	cfgOne := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "p"},
		ResourceScope: ast.ResourceScope{
			MCP: map[string]ast.MCPConfig{"s1": {Type: "stdio"}},
		},
	}
	cfgMany := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "p"},
		ResourceScope: ast.ResourceScope{
			MCP: map[string]ast.MCPConfig{"s1": {Type: "stdio"}, "s2": {Type: "stdio"}},
		},
	}

	h1 := renderTerminalHeader(buildGraph(cfgOne))
	assert.Contains(t, h1, "1 mcp server")
	assert.NotContains(t, h1, "mcp servers")

	h2 := renderTerminalHeader(buildGraph(cfgMany))
	assert.Contains(t, h2, "2 mcp servers")
}

func TestFilterAgentIfRequested_FiltersToSingleAgent(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"agent-a": {Skills: []string{"s1"}},
				"agent-b": {},
			},
		},
	}

	graphAgent = "agent-a"
	defer func() { graphAgent = "" }()

	err := filterAgentIfRequested(cfg)
	require.NoError(t, err)
	assert.Len(t, cfg.Agents, 1)
	assert.Contains(t, cfg.Agents, "agent-a")
}

func TestFilterAgentIfRequested_ErrorOnNotFound(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"agent-a": {},
			},
		},
	}

	graphAgent = "nonexistent"
	defer func() { graphAgent = "" }()

	err := filterAgentIfRequested(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
	assert.Contains(t, err.Error(), "agent-a")
}

func TestFilterAgentIfRequested_NoopWhenEmpty(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"agent-a": {},
				"agent-b": {},
			},
		},
	}

	graphAgent = ""
	err := filterAgentIfRequested(cfg)
	require.NoError(t, err)
	assert.Len(t, cfg.Agents, 2)
}
