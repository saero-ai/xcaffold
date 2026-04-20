package blueprint

import (
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── ResolveBlueprintExtends ─────────────────────────────────────────────────

func TestResolveBlueprintExtends_SimpleInheritance(t *testing.T) {
	blueprints := map[string]ast.BlueprintConfig{
		"parent": {
			Name:   "parent",
			Agents: []string{"agent-a"},
			Skills: []string{"skill-x"},
			Rules:  []string{"rule-1"},
		},
		"child": {
			Name:    "child",
			Extends: "parent",
			Agents:  []string{"agent-b"},
		},
	}

	err := ResolveBlueprintExtends(blueprints)
	require.NoError(t, err)

	child := blueprints["child"]
	assert.Equal(t, []string{"agent-a", "agent-b"}, child.Agents)
	assert.Equal(t, []string{"skill-x"}, child.Skills)
	assert.Equal(t, []string{"rule-1"}, child.Rules)
}

func TestResolveBlueprintExtends_ThreeLevelChain(t *testing.T) {
	// a → b → c  (a extends b which extends c)
	blueprints := map[string]ast.BlueprintConfig{
		"c": {
			Name:   "c",
			Agents: []string{"agent-c"},
			Skills: []string{"skill-c"},
		},
		"b": {
			Name:    "b",
			Extends: "c",
			Rules:   []string{"rule-b"},
		},
		"a": {
			Name:    "a",
			Extends: "b",
			MCP:     []string{"mcp-a"},
		},
	}

	err := ResolveBlueprintExtends(blueprints)
	require.NoError(t, err)

	a := blueprints["a"]
	assert.Equal(t, []string{"agent-c"}, a.Agents)
	assert.Equal(t, []string{"skill-c"}, a.Skills)
	assert.Equal(t, []string{"rule-b"}, a.Rules)
	assert.Equal(t, []string{"mcp-a"}, a.MCP)
}

func TestResolveBlueprintExtends_SetUnionDeduplication(t *testing.T) {
	blueprints := map[string]ast.BlueprintConfig{
		"base": {
			Name:   "base",
			Agents: []string{"agent-shared", "agent-base"},
			Skills: []string{"skill-shared"},
		},
		"derived": {
			Name:    "derived",
			Extends: "base",
			Agents:  []string{"agent-shared", "agent-derived"},
			Skills:  []string{"skill-shared", "skill-extra"},
		},
	}

	err := ResolveBlueprintExtends(blueprints)
	require.NoError(t, err)

	derived := blueprints["derived"]
	// agent-shared appears in both; must appear once, parent order first
	assert.Equal(t, []string{"agent-shared", "agent-base", "agent-derived"}, derived.Agents)
	assert.Equal(t, []string{"skill-shared", "skill-extra"}, derived.Skills)
}

func TestResolveBlueprintExtends_CycleDetection(t *testing.T) {
	blueprints := map[string]ast.BlueprintConfig{
		"a": {Name: "a", Extends: "b"},
		"b": {Name: "b", Extends: "a"},
	}

	err := ResolveBlueprintExtends(blueprints)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "circular"), "expected 'circular' in error: %s", err)
}

func TestResolveBlueprintExtends_SelfCycle(t *testing.T) {
	blueprints := map[string]ast.BlueprintConfig{
		"a": {Name: "a", Extends: "a"},
	}

	err := ResolveBlueprintExtends(blueprints)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "circular"), "expected 'circular' in error: %s", err)
}

func TestResolveBlueprintExtends_MaxDepthExceeded(t *testing.T) {
	// Chain of 6: a→b→c→d→e→f  (depth 5 when resolving a)
	blueprints := map[string]ast.BlueprintConfig{
		"f": {Name: "f"},
		"e": {Name: "e", Extends: "f"},
		"d": {Name: "d", Extends: "e"},
		"c": {Name: "c", Extends: "d"},
		"b": {Name: "b", Extends: "c"},
		"a": {Name: "a", Extends: "b"},
	}

	err := ResolveBlueprintExtends(blueprints)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "maximum depth"), "expected 'maximum depth' in error: %s", err)
}

func TestResolveBlueprintExtends_MaxDepthExact_Succeeds(t *testing.T) {
	// Chain of 5: a→b→c→d→e  (depth 4 when resolving a, within limit)
	blueprints := map[string]ast.BlueprintConfig{
		"e": {Name: "e", Agents: []string{"agent-e"}},
		"d": {Name: "d", Extends: "e"},
		"c": {Name: "c", Extends: "d"},
		"b": {Name: "b", Extends: "c"},
		"a": {Name: "a", Extends: "b"},
	}

	err := ResolveBlueprintExtends(blueprints)
	require.NoError(t, err)
	assert.Equal(t, []string{"agent-e"}, blueprints["a"].Agents)
}

func TestResolveBlueprintExtends_NonExistentParent(t *testing.T) {
	blueprints := map[string]ast.BlueprintConfig{
		"child": {Name: "child", Extends: "missing-parent"},
	}

	err := ResolveBlueprintExtends(blueprints)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "missing-parent"), "expected parent name in error: %s", err)
}

func TestResolveBlueprintExtends_NoExtends_NoOp(t *testing.T) {
	blueprints := map[string]ast.BlueprintConfig{
		"standalone": {
			Name:   "standalone",
			Agents: []string{"agent-x"},
		},
	}

	err := ResolveBlueprintExtends(blueprints)
	require.NoError(t, err)
	assert.Equal(t, []string{"agent-x"}, blueprints["standalone"].Agents)
}

// ── ResolveTransitiveDeps ───────────────────────────────────────────────────

func TestResolveTransitiveDeps_AgentPullsSkillsRulesMCP(t *testing.T) {
	scope := &ast.ResourceScope{
		Agents: map[string]ast.AgentConfig{
			"backend": {
				Skills: []string{"skill-db", "skill-http"},
				Rules:  []string{"rule-security"},
				MCP:    []string{"mcp-postgres"},
			},
		},
	}

	p := &ast.BlueprintConfig{
		Name:   "test",
		Agents: []string{"backend"},
	}

	ResolveTransitiveDeps(p, scope)

	assert.Equal(t, []string{"skill-db", "skill-http"}, p.Skills)
	assert.Equal(t, []string{"rule-security"}, p.Rules)
	assert.Equal(t, []string{"mcp-postgres"}, p.MCP)
}

func TestResolveTransitiveDeps_ExplicitSkillsNotOverridden(t *testing.T) {
	scope := &ast.ResourceScope{
		Agents: map[string]ast.AgentConfig{
			"backend": {
				Skills: []string{"skill-db"},
				Rules:  []string{"rule-a"},
				MCP:    []string{"mcp-x"},
			},
		},
	}

	// Blueprint already has explicit skills — those must not be replaced
	p := &ast.BlueprintConfig{
		Name:   "test",
		Agents: []string{"backend"},
		Skills: []string{"my-explicit-skill"},
	}

	ResolveTransitiveDeps(p, scope)

	// Skills unchanged (explicit override)
	assert.Equal(t, []string{"my-explicit-skill"}, p.Skills)
	// Rules and MCP still auto-resolved
	assert.Equal(t, []string{"rule-a"}, p.Rules)
	assert.Equal(t, []string{"mcp-x"}, p.MCP)
}

func TestResolveTransitiveDeps_NoAgents_NoOp(t *testing.T) {
	scope := &ast.ResourceScope{
		Agents: map[string]ast.AgentConfig{
			"backend": {Skills: []string{"skill-db"}},
		},
	}

	p := &ast.BlueprintConfig{
		Name:   "test",
		Agents: nil, // no agents selected
	}

	ResolveTransitiveDeps(p, scope)

	assert.Empty(t, p.Skills)
	assert.Empty(t, p.Rules)
	assert.Empty(t, p.MCP)
}

func TestResolveTransitiveDeps_NilScope_NoOp(t *testing.T) {
	p := &ast.BlueprintConfig{
		Name:   "test",
		Agents: []string{"backend"},
	}

	// Must not panic
	ResolveTransitiveDeps(p, nil)

	assert.Empty(t, p.Skills)
}

func TestResolveTransitiveDeps_AgentNotInScope_Skipped(t *testing.T) {
	scope := &ast.ResourceScope{
		Agents: map[string]ast.AgentConfig{},
	}

	p := &ast.BlueprintConfig{
		Name:   "test",
		Agents: []string{"ghost-agent"},
	}

	// Must not panic
	ResolveTransitiveDeps(p, scope)

	assert.Empty(t, p.Skills)
	assert.Empty(t, p.Rules)
	assert.Empty(t, p.MCP)
}

func TestResolveTransitiveDeps_MultipleAgents_Union(t *testing.T) {
	scope := &ast.ResourceScope{
		Agents: map[string]ast.AgentConfig{
			"frontend": {
				Skills: []string{"skill-react", "skill-shared"},
				Rules:  []string{"rule-ui"},
			},
			"backend": {
				Skills: []string{"skill-db", "skill-shared"},
				Rules:  []string{"rule-security"},
				MCP:    []string{"mcp-postgres"},
			},
		},
	}

	p := &ast.BlueprintConfig{
		Name:   "test",
		Agents: []string{"frontend", "backend"},
	}

	ResolveTransitiveDeps(p, scope)

	// skill-shared appears in both agents, must appear once
	assert.Equal(t, 3, len(p.Skills), "expected 3 unique skills, got: %v", p.Skills)
	for _, s := range []string{"skill-react", "skill-shared", "skill-db"} {
		assert.Contains(t, p.Skills, s)
	}
	assert.Equal(t, 2, len(p.Rules))
	assert.Equal(t, []string{"mcp-postgres"}, p.MCP)
}

func TestResolveTransitiveDeps_AllExplicit_NoAutoResolve(t *testing.T) {
	scope := &ast.ResourceScope{
		Agents: map[string]ast.AgentConfig{
			"backend": {
				Skills: []string{"skill-db"},
				Rules:  []string{"rule-a"},
				MCP:    []string{"mcp-x"},
			},
		},
	}

	// All three lists are explicitly set — nothing should be auto-added
	p := &ast.BlueprintConfig{
		Name:   "test",
		Agents: []string{"backend"},
		Skills: []string{"my-skill"},
		Rules:  []string{"my-rule"},
		MCP:    []string{"my-mcp"},
	}

	ResolveTransitiveDeps(p, scope)

	assert.Equal(t, []string{"my-skill"}, p.Skills)
	assert.Equal(t, []string{"my-rule"}, p.Rules)
	assert.Equal(t, []string{"my-mcp"}, p.MCP)
}
