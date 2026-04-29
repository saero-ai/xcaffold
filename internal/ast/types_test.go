package ast

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestProjectConfig_AdvisoryRefFields verifies that the six advisory ref
// fields on ProjectConfig are accessible at the Go level and are intentionally
// excluded from YAML marshaling.
//
// Design invariant: ref fields use yaml:"-" because ProjectConfig also embeds
// ResourceScope inline, which defines the same YAML keys (agents, skills, etc.)
// as map types. Exposing both at the same yaml key would cause yaml.v3 to panic
// with a "duplicated key" error. The parser uses a separate projectDocFields
// struct (without ResourceScope) to decode the ref lists, then assigns them to
// these fields post-decode.
func TestProjectConfig_AdvisoryRefFields(t *testing.T) {
	pc := &ProjectConfig{
		Name:         "my-proj",
		AgentRefs:    []AgentManifestEntry{{ID: "developer"}},
		SkillRefs:    []string{"tdd"},
		RuleRefs:     []string{"testing"},
		WorkflowRefs: []string{"ci"},
		MCPRefs:      []string{"github"},
		PolicyRefs:   []string{"require-model"},
	}

	require.Equal(t, []AgentManifestEntry{{ID: "developer"}}, pc.AgentRefs)
	require.Equal(t, []string{"tdd"}, pc.SkillRefs)
	require.Equal(t, []string{"testing"}, pc.RuleRefs)
	require.Equal(t, []string{"ci"}, pc.WorkflowRefs)
	require.Equal(t, []string{"github"}, pc.MCPRefs)
	require.Equal(t, []string{"require-model"}, pc.PolicyRefs)

	// Ref fields are intentionally excluded from YAML output (yaml:"-").
	// They are populated by the parser, not decoded from YAML, to avoid
	// a duplicate-key collision with ResourceScope's map-typed fields.
	data, err := yaml.Marshal(pc)
	require.NoError(t, err)
	content := string(data)
	// "agents:" in the output would come from ResourceScope.Agents (map), not AgentRefs.
	// Since ResourceScope.Agents is nil here, "agents:" should not appear at all.
	require.NotContains(t, content, "agents:")
	require.NotContains(t, content, "skills:")
	require.NotContains(t, content, "rules:")
	require.NotContains(t, content, "workflows:")
	require.NotContains(t, content, "mcp:")
	require.NotContains(t, content, "policies:")
}

// TestProjectConfig_AdvisoryRefFields_DoNotConflictWithResourceScope verifies
// that setting ref fields alongside ResourceScope.Agents (the map form) does
// not panic or produce unexpected YAML output.
func TestProjectConfig_AdvisoryRefFields_DoNotConflictWithResourceScope(t *testing.T) {
	pc := &ProjectConfig{
		Name:      "my-proj",
		AgentRefs: []AgentManifestEntry{{ID: "developer"}},
		ResourceScope: ResourceScope{
			Agents: map[string]AgentConfig{
				"developer": {Name: "developer", Model: "claude-sonnet-4-5"},
			},
		},
	}

	// Marshaling must not panic. yaml.v3 only panics on duplicate keys when
	// both the outer struct field and the inlined embedded field expose the
	// same yaml key name. AgentRefs uses yaml:"-", so there is no collision.
	require.NotPanics(t, func() {
		data, err := yaml.Marshal(pc)
		require.NoError(t, err)
		// The Agents map (from ResourceScope) is the one that appears in YAML.
		require.Contains(t, string(data), "agents:")
		require.Contains(t, string(data), "developer:")
	})
}

// --- BlueprintConfig tests ---

func TestBlueprintConfig_Fields_Exist(t *testing.T) {
	p := BlueprintConfig{
		Name:        "backend",
		Description: "Backend engineering",
		Extends:     "base",
		Agents:      []string{"developer"},
		Skills:      []string{"tdd"},
		Rules:       []string{"testing"},
		Workflows:   []string{"commit-changes"},
		MCP:         []string{"database-tools"},
		Policies:    []string{"security"},
		Memory:      []string{"arch-decisions"},
		Settings:    "default",
		Hooks:       "ci",
		Inherited:   false,
	}
	require.Equal(t, "backend", p.Name)
}

func TestXcaffoldConfig_BlueprintsMap_Exists(t *testing.T) {
	cfg := &XcaffoldConfig{
		Blueprints: map[string]BlueprintConfig{
			"backend": {Name: "backend"},
		},
	}
	require.Len(t, cfg.Blueprints, 1)
}

func TestProjectConfig_BlueprintRefs_Exists(t *testing.T) {
	pc := &ProjectConfig{
		BlueprintRefs: []string{"backend", "frontend"},
	}
	require.Len(t, pc.BlueprintRefs, 2)
}

// TestSkillConfig_Examples_RoundTrip verifies that the examples: field on
// SkillConfig is accepted by KnownFields(true) and survives a marshal/unmarshal
// round-trip with all values intact.
func TestSkillConfig_Examples_RoundTrip(t *testing.T) {
	input := `examples:
- xcf/skills/tdd/examples/basic.xcf
- xcf/skills/tdd/examples/advanced.xcf
`
	var sc SkillConfig
	dec := yaml.NewDecoder(strings.NewReader(input))
	dec.KnownFields(true)
	require.NoError(t, dec.Decode(&sc), "examples: must be a known field on SkillConfig")

	require.Len(t, sc.Examples, 2)
	require.Equal(t, "xcf/skills/tdd/examples/basic.xcf", sc.Examples[0])
	require.Equal(t, "xcf/skills/tdd/examples/advanced.xcf", sc.Examples[1])

	data, err := yaml.Marshal(sc)
	require.NoError(t, err)
	require.Contains(t, string(data), "examples:")

	var sc2 SkillConfig
	require.NoError(t, yaml.Unmarshal(data, &sc2))
	require.Equal(t, sc.Examples, sc2.Examples)
}

// TestResourceOverrides_ExcludedFromYAML verifies that XcaffoldConfig.Overrides
// is excluded from YAML serialization via the yaml:"-" and json:"-" tags.
// ResourceOverrides is populated by the importer during filesystem scanning
// and is never written back to YAML.
func TestResourceOverrides_ExcludedFromYAML(t *testing.T) {
	cfg := &XcaffoldConfig{
		Version: "1.0",
		Overrides: &ResourceOverrides{
			Agent:    make(map[string]map[string]AgentConfig),
			Skill:    make(map[string]map[string]SkillConfig),
			Rule:     make(map[string]map[string]RuleConfig),
			Workflow: make(map[string]map[string]WorkflowConfig),
			MCP:      make(map[string]map[string]MCPConfig),
		},
	}
	cfg.Overrides.Agent["my-agent"] = map[string]AgentConfig{
		"claude": {Name: "my-agent", Model: "claude-opus"},
	}

	// Marshal to YAML
	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	content := string(data)

	// Overrides field must not appear in YAML output
	require.NotContains(t, content, "overrides:")
	require.NotContains(t, content, "Overrides:")

	// Unmarshal back: the field must be nil (not populated from YAML)
	var cfg2 XcaffoldConfig
	require.NoError(t, yaml.Unmarshal(data, &cfg2))
	require.Nil(t, cfg2.Overrides)
}

// TestResourceOverrides_AgentProviders_ListsProviders verifies that
// ResourceOverrides.AgentProviders returns sorted provider names for a given agent.
func TestResourceOverrides_AgentProviders_ListsProviders(t *testing.T) {
	ro := &ResourceOverrides{
		Agent: map[string]map[string]AgentConfig{
			"my-agent": {
				"cursor":    {Name: "my-agent", Model: "cursor-model"},
				"anthropic": {Name: "my-agent", Model: "claude-opus"},
				"gemini-ai": {Name: "my-agent", Model: "gemini-2.5-pro"},
			},
		},
	}

	providers := ro.AgentProviders("my-agent")
	require.Len(t, providers, 3)
	require.Equal(t, []string{"anthropic", "cursor", "gemini-ai"}, providers)
}

// TestResourceOverrides_SkillProviders_ListsProviders verifies that
// ResourceOverrides.SkillProviders returns sorted provider names for a given skill.
func TestResourceOverrides_SkillProviders_ListsProviders(t *testing.T) {
	ro := &ResourceOverrides{
		Skill: map[string]map[string]SkillConfig{
			"tdd": {
				"zeta":  {Name: "tdd", AllowedTools: []string{"Read"}},
				"alpha": {Name: "tdd", AllowedTools: []string{"Write"}},
				"gamma": {Name: "tdd", AllowedTools: []string{"Edit"}},
			},
		},
	}

	providers := ro.SkillProviders("tdd")
	require.Len(t, providers, 3)
	require.Equal(t, []string{"alpha", "gamma", "zeta"}, providers)
}

// TestResourceOverrides_AddAgent_StoresAndRetrievesAgentConfigs verifies that
// ResourceOverrides.AddAgent stores and GetAgent retrieves agent configs
// keyed by [name][provider].
func TestResourceOverrides_AddAgent_StoresAndRetrievesAgentConfigs(t *testing.T) {
	ro := &ResourceOverrides{}

	agent1 := AgentConfig{Name: "dev", Model: "claude-opus"}
	ro.AddAgent("dev", "claude", agent1)

	retrieved, ok := ro.GetAgent("dev", "claude")
	require.True(t, ok)
	require.Equal(t, agent1, retrieved)

	_, ok = ro.GetAgent("dev", "cursor")
	require.False(t, ok)
}

// TestResourceOverrides_AddSkill_StoresAndRetrievesSkillConfigs verifies that
// ResourceOverrides.AddSkill stores and GetSkill retrieves skill configs.
func TestResourceOverrides_AddSkill_StoresAndRetrievesSkillConfigs(t *testing.T) {
	ro := &ResourceOverrides{}

	skill1 := SkillConfig{Name: "test", AllowedTools: []string{"Bash"}}
	ro.AddSkill("test", "claude", skill1)

	retrieved, ok := ro.GetSkill("test", "claude")
	require.True(t, ok)
	require.Equal(t, skill1, retrieved)
}

// TestResourceOverrides_AddRule_StoresAndRetrievesRuleConfigs verifies that
// ResourceOverrides.AddRule stores and GetRule retrieves rule configs.
func TestResourceOverrides_AddRule_StoresAndRetrievesRuleConfigs(t *testing.T) {
	ro := &ResourceOverrides{}

	rule1 := RuleConfig{Name: "secure", AlwaysApply: boolPtr(true)}
	ro.AddRule("secure", "cursor", rule1)

	retrieved, ok := ro.GetRule("secure", "cursor")
	require.True(t, ok)
	require.Equal(t, rule1, retrieved)
}

// TestResourceOverrides_AddWorkflow_StoresAndRetrievesWorkflowConfigs verifies that
// ResourceOverrides.AddWorkflow stores and GetWorkflow retrieves workflow configs.
func TestResourceOverrides_AddWorkflow_StoresAndRetrievesWorkflowConfigs(t *testing.T) {
	ro := &ResourceOverrides{}

	wf1 := WorkflowConfig{Name: "ci", ApiVersion: "workflow/v1"}
	ro.AddWorkflow("ci", "gemini", wf1)

	retrieved, ok := ro.GetWorkflow("ci", "gemini")
	require.True(t, ok)
	require.Equal(t, wf1, retrieved)
}

// TestResourceOverrides_AddMCP_StoresAndRetrievesMCPConfigs verifies that
// ResourceOverrides.AddMCP stores and GetMCP retrieves MCP configs.
func TestResourceOverrides_AddMCP_StoresAndRetrievesMCPConfigs(t *testing.T) {
	ro := &ResourceOverrides{}

	mcp1 := MCPConfig{Name: "github", Type: "command"}
	ro.AddMCP("github", "copilot", mcp1)

	retrieved, ok := ro.GetMCP("github", "copilot")
	require.True(t, ok)
	require.Equal(t, mcp1, retrieved)
}

// Helper function for creating bool pointers in tests
func boolPtr(b bool) *bool {
	return &b
}
