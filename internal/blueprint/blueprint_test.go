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

// ── ValidateBlueprintRefs ───────────────────────────────────────────────────

func TestValidateBlueprintRefs_AllExist_NoErrors(t *testing.T) {
	blueprints := map[string]ast.BlueprintConfig{
		"backend": {Name: "backend", Agents: []string{"developer"}, Skills: []string{"tdd"}},
	}
	scope := &ast.ResourceScope{
		Agents: map[string]ast.AgentConfig{"developer": {}},
		Skills: map[string]ast.SkillConfig{"tdd": {}},
	}
	errs := ValidateBlueprintRefs(blueprints, scope)
	require.Empty(t, errs)
}

func TestValidateBlueprintRefs_MissingAgent(t *testing.T) {
	blueprints := map[string]ast.BlueprintConfig{
		"backend": {Name: "backend", Agents: []string{"missing-agent"}},
	}
	scope := &ast.ResourceScope{
		Agents: map[string]ast.AgentConfig{"developer": {}},
	}
	errs := ValidateBlueprintRefs(blueprints, scope)
	require.Len(t, errs, 1)
	require.Contains(t, errs[0].Error(), "missing-agent")
	require.Contains(t, errs[0].Error(), "backend")
}

func TestValidateBlueprintRefs_MultipleErrors(t *testing.T) {
	blueprints := map[string]ast.BlueprintConfig{
		"bad": {Name: "bad", Agents: []string{"ghost"}, Skills: []string{"ghost"}, Rules: []string{"ghost"}},
	}
	scope := &ast.ResourceScope{
		Agents: map[string]ast.AgentConfig{},
		Skills: map[string]ast.SkillConfig{},
		Rules:  map[string]ast.RuleConfig{},
	}
	errs := ValidateBlueprintRefs(blueprints, scope)
	require.Len(t, errs, 3)
}

func TestValidateBlueprintRefs_EmptyBlueprint_NoErrors(t *testing.T) {
	blueprints := map[string]ast.BlueprintConfig{
		"empty": {Name: "empty"},
	}
	scope := &ast.ResourceScope{}
	errs := ValidateBlueprintRefs(blueprints, scope)
	require.Empty(t, errs)
}

func TestValidateBlueprintRefs_NilMapsInScope_NoErrors(t *testing.T) {
	blueprints := map[string]ast.BlueprintConfig{
		"bp": {Name: "bp"},
	}
	// nil scope maps — must not panic
	scope := &ast.ResourceScope{}
	errs := ValidateBlueprintRefs(blueprints, scope)
	require.Empty(t, errs)
}

func TestValidateBlueprintRefs_NilScope_NoErrors(t *testing.T) {
	blueprints := map[string]ast.BlueprintConfig{
		"bp": {Name: "bp", Agents: []string{"someone"}},
	}
	// nil scope — all refs missing but must not panic; returns errors for missing refs
	errs := ValidateBlueprintRefs(blueprints, nil)
	require.Len(t, errs, 1)
	require.Contains(t, errs[0].Error(), "someone")
}

func TestValidateBlueprintRefs_EmptyBlueprintsMap_NoErrors(t *testing.T) {
	errs := ValidateBlueprintRefs(map[string]ast.BlueprintConfig{}, &ast.ResourceScope{})
	require.Empty(t, errs)
}

func TestValidateBlueprintRefs_AllResourceTypes(t *testing.T) {
	blueprints := map[string]ast.BlueprintConfig{
		"full": {
			Name:      "full",
			Agents:    []string{"missing-agent"},
			Skills:    []string{"missing-skill"},
			Rules:     []string{"missing-rule"},
			Workflows: []string{"missing-workflow"},
			MCP:       []string{"missing-mcp"},
			Policies:  []string{"missing-policy"},
			Memory:    []string{"missing-memory"},
		},
	}
	scope := &ast.ResourceScope{}
	errs := ValidateBlueprintRefs(blueprints, scope)
	require.Len(t, errs, 7)
}

// ── ApplyBlueprint ──────────────────────────────────────────────────────────

func TestApplyBlueprint_FiltersAgents(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"developer": {Name: "Developer"},
				"designer":  {Name: "Designer"},
			},
			Skills: map[string]ast.SkillConfig{
				"tdd":   {},
				"figma": {},
			},
		},
		Blueprints: map[string]ast.BlueprintConfig{
			"backend": {Name: "backend", Agents: []string{"developer"}, Skills: []string{"tdd"}},
		},
	}
	filtered, err := ApplyBlueprint(cfg, "backend")
	require.NoError(t, err)
	require.Contains(t, filtered.Agents, "developer")
	require.NotContains(t, filtered.Agents, "designer")
	require.Contains(t, filtered.Skills, "tdd")
	require.NotContains(t, filtered.Skills, "figma")
}

func TestApplyBlueprint_EmptyName_ReturnsUnmodified(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{"developer": {}},
		},
	}
	result, err := ApplyBlueprint(cfg, "")
	require.NoError(t, err)
	require.Same(t, cfg, result)
}

func TestApplyBlueprint_UnknownBlueprint_ReturnsError(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		Blueprints: map[string]ast.BlueprintConfig{
			"backend": {Name: "backend"},
		},
	}
	_, err := ApplyBlueprint(cfg, "unknown")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown")
	require.Contains(t, err.Error(), "backend")
}

func TestApplyBlueprint_DoesNotModifyInput(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"developer": {},
				"designer":  {},
			},
		},
		Blueprints: map[string]ast.BlueprintConfig{
			"backend": {Name: "backend", Agents: []string{"developer"}},
		},
	}
	_, err := ApplyBlueprint(cfg, "backend")
	require.NoError(t, err)
	require.Contains(t, cfg.Agents, "designer")
}

// ── BlueprintHash ───────────────────────────────────────────────────────────

func TestBlueprintHash_StableForSameRefs(t *testing.T) {
	p := ast.BlueprintConfig{Name: "backend", Agents: []string{"dev", "dba"}, Skills: []string{"tdd"}}
	h1 := BlueprintHash(p)
	h2 := BlueprintHash(p)
	require.Equal(t, h1, h2)
	require.True(t, strings.HasPrefix(h1, "sha256:"))
}

func TestBlueprintHash_ChangesWhenRefsChange(t *testing.T) {
	p1 := ast.BlueprintConfig{Agents: []string{"dev"}}
	p2 := ast.BlueprintConfig{Agents: []string{"dev", "dba"}}
	require.NotEqual(t, BlueprintHash(p1), BlueprintHash(p2))
}

func TestBlueprintHash_OrderIndependent(t *testing.T) {
	p1 := ast.BlueprintConfig{Agents: []string{"a", "b"}}
	p2 := ast.BlueprintConfig{Agents: []string{"b", "a"}}
	require.Equal(t, BlueprintHash(p1), BlueprintHash(p2))
}

func TestApplyBlueprint_EmptyRefList_ReturnsNilMap(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{"developer": {}},
			Skills: map[string]ast.SkillConfig{"tdd": {}},
		},
		Blueprints: map[string]ast.BlueprintConfig{
			"minimal": {Name: "minimal", Agents: []string{"developer"}},
		},
	}
	filtered, err := ApplyBlueprint(cfg, "minimal")
	require.NoError(t, err)
	require.Contains(t, filtered.Agents, "developer")
	require.Nil(t, filtered.Skills) // not listed in blueprint = nil
}
