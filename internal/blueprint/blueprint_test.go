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
			Agents: ast.ClearableList{Values: []string{"agent-a"}},
			Skills: ast.ClearableList{Values: []string{"skill-x"}},
			Rules:  ast.ClearableList{Values: []string{"rule-1"}},
		},
		"child": {
			Name:    "child",
			Extends: "parent",
			Agents:  ast.ClearableList{Values: []string{"agent-b"}},
		},
	}

	err := ResolveBlueprintExtends(blueprints)
	require.NoError(t, err)

	child := blueprints["child"]
	assert.Equal(t, []string{"agent-a", "agent-b"}, child.Agents.Values)
	assert.Equal(t, []string{"skill-x"}, child.Skills.Values)
	assert.Equal(t, []string{"rule-1"}, child.Rules.Values)
}

func TestResolveBlueprintExtends_ThreeLevelChain(t *testing.T) {
	// a → b → c  (a extends b which extends c)
	blueprints := map[string]ast.BlueprintConfig{
		"c": {
			Name:   "c",
			Agents: ast.ClearableList{Values: []string{"agent-c"}},
			Skills: ast.ClearableList{Values: []string{"skill-c"}},
		},
		"b": {
			Name:    "b",
			Extends: "c",
			Rules:   ast.ClearableList{Values: []string{"rule-b"}},
		},
		"a": {
			Name:    "a",
			Extends: "b",
			MCP:     ast.ClearableList{Values: []string{"mcp-a"}},
		},
	}

	err := ResolveBlueprintExtends(blueprints)
	require.NoError(t, err)

	a := blueprints["a"]
	assert.Equal(t, []string{"agent-c"}, a.Agents.Values)
	assert.Equal(t, []string{"skill-c"}, a.Skills.Values)
	assert.Equal(t, []string{"rule-b"}, a.Rules.Values)
	assert.Equal(t, []string{"mcp-a"}, a.MCP.Values)
}

func TestResolveBlueprintExtends_SetUnionDeduplication(t *testing.T) {
	blueprints := map[string]ast.BlueprintConfig{
		"base": {
			Name:   "base",
			Agents: ast.ClearableList{Values: []string{"agent-shared", "agent-base"}},
			Skills: ast.ClearableList{Values: []string{"skill-shared"}},
		},
		"derived": {
			Name:    "derived",
			Extends: "base",
			Agents:  ast.ClearableList{Values: []string{"agent-shared", "agent-derived"}},
			Skills:  ast.ClearableList{Values: []string{"skill-shared", "skill-extra"}},
		},
	}

	err := ResolveBlueprintExtends(blueprints)
	require.NoError(t, err)

	derived := blueprints["derived"]
	// agent-shared appears in both; must appear once, parent order first
	assert.Equal(t, []string{"agent-shared", "agent-base", "agent-derived"}, derived.Agents.Values)
	assert.Equal(t, []string{"skill-shared", "skill-extra"}, derived.Skills.Values)
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
		"e": {Name: "e", Agents: ast.ClearableList{Values: []string{"agent-e"}}},
		"d": {Name: "d", Extends: "e"},
		"c": {Name: "c", Extends: "d"},
		"b": {Name: "b", Extends: "c"},
		"a": {Name: "a", Extends: "b"},
	}

	err := ResolveBlueprintExtends(blueprints)
	require.NoError(t, err)
	assert.Equal(t, []string{"agent-e"}, blueprints["a"].Agents.Values)
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
			Agents: ast.ClearableList{Values: []string{"agent-x"}},
		},
	}

	err := ResolveBlueprintExtends(blueprints)
	require.NoError(t, err)
	assert.Equal(t, []string{"agent-x"}, blueprints["standalone"].Agents.Values)
}

// ── ResolveTransitiveDeps ───────────────────────────────────────────────────

func TestResolveTransitiveDeps_AgentPullsSkillsRulesMCP(t *testing.T) {
	scope := &ast.ResourceScope{
		Agents: map[string]ast.AgentConfig{
			"backend": {
				Skills: ast.ClearableList{Values: []string{"skill-db", "skill-http"}},
				Rules:  ast.ClearableList{Values: []string{"rule-security"}},
				MCP:    ast.ClearableList{Values: []string{"mcp-postgres"}},
			},
		},
	}

	p := &ast.BlueprintConfig{
		Name:   "test",
		Agents: ast.ClearableList{Values: []string{"backend"}},
	}

	err := ResolveTransitiveDeps(p, scope)
	require.NoError(t, err)

	assert.Equal(t, []string{"skill-db", "skill-http"}, p.Skills.Values)
	assert.Equal(t, []string{"rule-security"}, p.Rules.Values)
	assert.Equal(t, []string{"mcp-postgres"}, p.MCP.Values)
}

func TestResolveTransitiveDeps_ExplicitSkillsMergedWithAgentDeps(t *testing.T) {
	scope := &ast.ResourceScope{
		Agents: map[string]ast.AgentConfig{
			"backend": {
				Skills: ast.ClearableList{Values: []string{"skill-db"}},
				Rules:  ast.ClearableList{Values: []string{"rule-a"}},
				MCP:    ast.ClearableList{Values: []string{"mcp-x"}},
			},
		},
	}

	// Blueprint has explicit skills with NO overlap — should merge with agent deps
	p := &ast.BlueprintConfig{
		Name:   "test",
		Agents: ast.ClearableList{Values: []string{"backend"}},
		Skills: ast.ClearableList{Values: []string{"my-explicit-skill"}},
	}

	err := ResolveTransitiveDeps(p, scope)
	require.NoError(t, err)

	// Explicit skill + agent skill merged
	assert.Equal(t, []string{"my-explicit-skill", "skill-db"}, p.Skills.Values)
	// Rules and MCP auto-resolved
	assert.Equal(t, []string{"rule-a"}, p.Rules.Values)
	assert.Equal(t, []string{"mcp-x"}, p.MCP.Values)
}

func TestResolveTransitiveDeps_NoAgents_NoOp(t *testing.T) {
	scope := &ast.ResourceScope{
		Agents: map[string]ast.AgentConfig{
			"backend": {Skills: ast.ClearableList{Values: []string{"skill-db"}}},
		},
	}

	p := &ast.BlueprintConfig{
		Name: "test",
		// Agents zero-value (no agents selected)
	}

	err := ResolveTransitiveDeps(p, scope)
	require.NoError(t, err)

	assert.Empty(t, p.Skills)
	assert.Empty(t, p.Rules)
	assert.Empty(t, p.MCP)
}

func TestResolveTransitiveDeps_NilScope_NoOp(t *testing.T) {
	p := &ast.BlueprintConfig{
		Name:   "test",
		Agents: ast.ClearableList{Values: []string{"backend"}},
	}

	// Must not panic
	err := ResolveTransitiveDeps(p, nil)
	require.NoError(t, err)

	assert.Empty(t, p.Skills)
}

func TestResolveTransitiveDeps_AgentNotInScope_Skipped(t *testing.T) {
	scope := &ast.ResourceScope{
		Agents: map[string]ast.AgentConfig{},
	}

	p := &ast.BlueprintConfig{
		Name:   "test",
		Agents: ast.ClearableList{Values: []string{"ghost-agent"}},
	}

	// Must not panic
	err := ResolveTransitiveDeps(p, scope)
	require.NoError(t, err)

	assert.Empty(t, p.Skills)
	assert.Empty(t, p.Rules)
	assert.Empty(t, p.MCP)
}

func TestResolveTransitiveDeps_MultipleAgents_Union(t *testing.T) {
	scope := &ast.ResourceScope{
		Agents: map[string]ast.AgentConfig{
			"frontend": {
				Skills: ast.ClearableList{Values: []string{"skill-react", "skill-shared"}},
				Rules:  ast.ClearableList{Values: []string{"rule-ui"}},
			},
			"backend": {
				Skills: ast.ClearableList{Values: []string{"skill-db", "skill-shared"}},
				Rules:  ast.ClearableList{Values: []string{"rule-security"}},
				MCP:    ast.ClearableList{Values: []string{"mcp-postgres"}},
			},
		},
	}

	p := &ast.BlueprintConfig{
		Name:   "test",
		Agents: ast.ClearableList{Values: []string{"frontend", "backend"}},
	}

	err := ResolveTransitiveDeps(p, scope)
	require.NoError(t, err)

	// skill-shared appears in both agents, must appear once
	assert.Equal(t, 3, len(p.Skills.Values), "expected 3 unique skills, got: %v", p.Skills.Values)
	for _, s := range []string{"skill-react", "skill-shared", "skill-db"} {
		assert.Contains(t, p.Skills.Values, s)
	}
	assert.Equal(t, 2, len(p.Rules.Values))
	assert.Equal(t, []string{"mcp-postgres"}, p.MCP.Values)
}

func TestResolveTransitiveDeps_AllExplicitMergedWithAgentDeps(t *testing.T) {
	scope := &ast.ResourceScope{
		Agents: map[string]ast.AgentConfig{
			"backend": {
				Skills: ast.ClearableList{Values: []string{"skill-db"}},
				Rules:  ast.ClearableList{Values: []string{"rule-a"}},
				MCP:    ast.ClearableList{Values: []string{"mcp-x"}},
			},
		},
	}

	// All three lists are explicitly set with NO overlaps — should merge with agent deps
	p := &ast.BlueprintConfig{
		Name:   "test",
		Agents: ast.ClearableList{Values: []string{"backend"}},
		Skills: ast.ClearableList{Values: []string{"my-skill"}},
		Rules:  ast.ClearableList{Values: []string{"my-rule"}},
		MCP:    ast.ClearableList{Values: []string{"my-mcp"}},
	}

	err := ResolveTransitiveDeps(p, scope)
	require.NoError(t, err)

	assert.Equal(t, []string{"my-skill", "skill-db"}, p.Skills.Values)
	assert.Equal(t, []string{"my-rule", "rule-a"}, p.Rules.Values)
	assert.Equal(t, []string{"my-mcp", "mcp-x"}, p.MCP.Values)
}

func TestResolveTransitiveDeps_DuplicateSkill_ReturnsError(t *testing.T) {
	scope := &ast.ResourceScope{
		Agents: map[string]ast.AgentConfig{
			"developer": {
				Skills: ast.ClearableList{Values: []string{"tdd", "schema-design"}},
			},
		},
	}

	p := &ast.BlueprintConfig{
		Name:   "test",
		Agents: ast.ClearableList{Values: []string{"developer"}},
		Skills: ast.ClearableList{Values: []string{"tdd", "security-audit"}},
	}

	err := ResolveTransitiveDeps(p, scope)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tdd")
	assert.Contains(t, err.Error(), "developer")
}

func TestResolveTransitiveDeps_DuplicateRule_ReturnsError(t *testing.T) {
	scope := &ast.ResourceScope{
		Agents: map[string]ast.AgentConfig{
			"developer": {
				Rules: ast.ClearableList{Values: []string{"secure-code"}},
			},
		},
	}

	p := &ast.BlueprintConfig{
		Name:   "test",
		Agents: ast.ClearableList{Values: []string{"developer"}},
		Rules:  ast.ClearableList{Values: []string{"secure-code", "extra-rule"}},
	}

	err := ResolveTransitiveDeps(p, scope)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "secure-code")
	assert.Contains(t, err.Error(), "developer")
}

func TestResolveTransitiveDeps_DuplicateMCP_ReturnsError(t *testing.T) {
	scope := &ast.ResourceScope{
		Agents: map[string]ast.AgentConfig{
			"developer": {
				MCP: ast.ClearableList{Values: []string{"database-tools"}},
			},
		},
	}

	p := &ast.BlueprintConfig{
		Name:   "test",
		Agents: ast.ClearableList{Values: []string{"developer"}},
		MCP:    ast.ClearableList{Values: []string{"database-tools"}},
	}

	err := ResolveTransitiveDeps(p, scope)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database-tools")
	assert.Contains(t, err.Error(), "developer")
}

// ── ValidateBlueprintRefs ───────────────────────────────────────────────────

func TestValidateBlueprintRefs_AllExist_NoErrors(t *testing.T) {
	blueprints := map[string]ast.BlueprintConfig{
		"backend": {Name: "backend", Agents: ast.ClearableList{Values: []string{"developer"}}, Skills: ast.ClearableList{Values: []string{"tdd"}}},
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
		"backend": {Name: "backend", Agents: ast.ClearableList{Values: []string{"missing-agent"}}},
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
		"bad": {Name: "bad", Agents: ast.ClearableList{Values: []string{"ghost"}}, Skills: ast.ClearableList{Values: []string{"ghost"}}, Rules: ast.ClearableList{Values: []string{"ghost"}}},
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
		"bp": {Name: "bp", Agents: ast.ClearableList{Values: []string{"someone"}}},
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
			Agents:    ast.ClearableList{Values: []string{"missing-agent"}},
			Skills:    ast.ClearableList{Values: []string{"missing-skill"}},
			Rules:     ast.ClearableList{Values: []string{"missing-rule"}},
			Workflows: ast.ClearableList{Values: []string{"missing-workflow"}},
			MCP:       ast.ClearableList{Values: []string{"missing-mcp"}},
			Policies:  ast.ClearableList{Values: []string{"missing-policy"}},
			Memory:    ast.ClearableList{Values: []string{"missing-memory"}},
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
			"backend": {Name: "backend", Agents: ast.ClearableList{Values: []string{"developer"}}, Skills: ast.ClearableList{Values: []string{"tdd"}}},
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
			"backend": {Name: "backend", Agents: ast.ClearableList{Values: []string{"developer"}}},
		},
	}
	_, err := ApplyBlueprint(cfg, "backend")
	require.NoError(t, err)
	require.Contains(t, cfg.Agents, "designer")
}

// ── BlueprintHash ───────────────────────────────────────────────────────────

func TestBlueprintHash_StableForSameRefs(t *testing.T) {
	p := ast.BlueprintConfig{Name: "backend", Agents: ast.ClearableList{Values: []string{"dev", "dba"}}, Skills: ast.ClearableList{Values: []string{"tdd"}}}
	h1 := BlueprintHash(p)
	h2 := BlueprintHash(p)
	require.Equal(t, h1, h2)
	require.True(t, strings.HasPrefix(h1, "sha256:"))
}

func TestBlueprintHash_ChangesWhenRefsChange(t *testing.T) {
	p1 := ast.BlueprintConfig{Agents: ast.ClearableList{Values: []string{"dev"}}}
	p2 := ast.BlueprintConfig{Agents: ast.ClearableList{Values: []string{"dev", "dba"}}}
	require.NotEqual(t, BlueprintHash(p1), BlueprintHash(p2))
}

func TestBlueprintHash_OrderIndependent(t *testing.T) {
	p1 := ast.BlueprintConfig{Agents: ast.ClearableList{Values: []string{"a", "b"}}}
	p2 := ast.BlueprintConfig{Agents: ast.ClearableList{Values: []string{"b", "a"}}}
	require.Equal(t, BlueprintHash(p1), BlueprintHash(p2))
}

func TestApplyBlueprint_NamedSettings_SelectsOnly(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		Settings: map[string]ast.SettingsConfig{
			"default":    {Model: "balanced"},
			"restricted": {Model: "haiku"},
		},
		Blueprints: map[string]ast.BlueprintConfig{
			"locked": {Name: "locked", Settings: "restricted"},
		},
	}
	filtered, err := ApplyBlueprint(cfg, "locked")
	require.NoError(t, err)
	require.Len(t, filtered.Settings, 1)
	require.Contains(t, filtered.Settings, "restricted")
}

func TestApplyBlueprint_NamedSettings_MissingKey_Error(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		Settings: map[string]ast.SettingsConfig{"default": {Model: "balanced"}},
		Blueprints: map[string]ast.BlueprintConfig{
			"bad": {Name: "bad", Settings: "nonexistent"},
		},
	}
	_, err := ApplyBlueprint(cfg, "bad")
	require.Error(t, err)
	require.Contains(t, err.Error(), "nonexistent")
}

func TestApplyBlueprint_NamedHooks_SelectsOnly(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		Hooks: map[string]ast.NamedHookConfig{
			"ci":   {Name: "ci"},
			"test": {Name: "test"},
		},
		Blueprints: map[string]ast.BlueprintConfig{
			"backend": {Name: "backend", Hooks: "ci"},
		},
	}
	filtered, err := ApplyBlueprint(cfg, "backend")
	require.NoError(t, err)
	require.Contains(t, filtered.Hooks, "ci")
	require.NotContains(t, filtered.Hooks, "test")
}

func TestApplyBlueprint_OmittedSettings_IncludesAll(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		Settings: map[string]ast.SettingsConfig{
			"default": {Model: "balanced"},
			"other":   {Model: "haiku"},
		},
		Blueprints: map[string]ast.BlueprintConfig{
			"backend": {Name: "backend"},
		},
	}
	filtered, err := ApplyBlueprint(cfg, "backend")
	require.NoError(t, err)
	require.Len(t, filtered.Settings, 2)
}

func TestApplyBlueprint_EmptyRefList_ReturnsNilMap(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{"developer": {}},
			Skills: map[string]ast.SkillConfig{"tdd": {}},
		},
		Blueprints: map[string]ast.BlueprintConfig{
			"minimal": {Name: "minimal", Agents: ast.ClearableList{Values: []string{"developer"}}},
		},
	}
	filtered, err := ApplyBlueprint(cfg, "minimal")
	require.NoError(t, err)
	require.Contains(t, filtered.Agents, "developer")
	require.Nil(t, filtered.Skills) // not listed in blueprint = nil
}

func TestApplyBlueprint_FiltersContexts(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Contexts: map[string]ast.ContextConfig{
				"main":       {Name: "main", Body: "shared context"},
				"claude-dev": {Name: "claude-dev", Targets: []string{"claude"}, Body: "claude specific"},
				"gemini-dev": {Name: "gemini-dev", Targets: []string{"gemini"}, Body: "gemini specific"},
			},
		},
		Blueprints: map[string]ast.BlueprintConfig{
			"backend": {Name: "backend", Contexts: ast.ClearableList{Values: []string{"main", "claude-dev"}}},
		},
	}
	filtered, err := ApplyBlueprint(config, "backend")
	require.NoError(t, err)
	require.Len(t, filtered.Contexts, 2)
	require.Contains(t, filtered.Contexts, "main")
	require.Contains(t, filtered.Contexts, "claude-dev")
	require.NotContains(t, filtered.Contexts, "gemini-dev")
}

func TestValidateBlueprintRefs_MissingContext(t *testing.T) {
	blueprints := map[string]ast.BlueprintConfig{
		"test": {Name: "test", Contexts: ast.ClearableList{Values: []string{"nonexistent"}}},
	}
	scope := &ast.ResourceScope{
		Contexts: map[string]ast.ContextConfig{"main": {}},
	}
	errs := ValidateBlueprintRefs(blueprints, scope)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "nonexistent")
}

func TestResolveBlueprintExtends_ChildClearsParentField(t *testing.T) {
	blueprints := map[string]ast.BlueprintConfig{
		"parent": {
			Name:   "parent",
			Agents: ast.ClearableList{Values: []string{"agent-a", "agent-b"}},
			Skills: ast.ClearableList{Values: []string{"skill-x"}},
		},
		"child": {
			Name:    "child",
			Extends: "parent",
			Agents:  ast.ClearableList{Cleared: true},
			Skills:  ast.ClearableList{},
		},
	}

	err := ResolveBlueprintExtends(blueprints)
	require.NoError(t, err)

	child := blueprints["child"]
	assert.Empty(t, child.Agents.Values, "child cleared agents, should be empty")
	assert.True(t, child.Agents.Cleared, "Cleared flag should be preserved")
	assert.Equal(t, []string{"skill-x"}, child.Skills.Values, "skills inherited from parent")
}
