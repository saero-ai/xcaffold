package ast

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// --- BlueprintConfig tests ---

func TestBlueprintConfig_Fields_Exist(t *testing.T) {
	p := BlueprintConfig{
		Name:        "backend",
		Description: "Backend engineering",
		Extends:     "base",
		Agents:      ClearableList{Values: []string{"developer"}},
		Skills:      ClearableList{Values: []string{"tdd"}},
		Rules:       ClearableList{Values: []string{"testing"}},
		Workflows:   ClearableList{Values: []string{"commit-changes"}},
		MCP:         ClearableList{Values: []string{"database-tools"}},
		Policies:    ClearableList{Values: []string{"security"}},
		Memory:      ClearableList{Values: []string{"arch-decisions"}},
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

func TestSkillConfig_FieldCount_OverrideScoringGuard(t *testing.T) {
	const expected = 14
	rt := reflect.TypeOf(SkillConfig{})
	if rt.NumField() != expected {
		t.Fatalf("SkillConfig has %d fields, expected %d — update override scoring if fields changed", rt.NumField(), expected)
	}
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
				"zeta":  {Name: "tdd", AllowedTools: ClearableList{Values: []string{"Read"}}},
				"alpha": {Name: "tdd", AllowedTools: ClearableList{Values: []string{"Write"}}},
				"gamma": {Name: "tdd", AllowedTools: ClearableList{Values: []string{"Edit"}}},
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

	skill1 := SkillConfig{Name: "test", AllowedTools: ClearableList{Values: []string{"Bash"}}}
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

	wf1 := WorkflowConfig{Name: "ci"}
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

func TestResourceOverrides_AddTemplate_StoresAndRetrievesTemplateConfigs(t *testing.T) {
	ro := &ResourceOverrides{}

	tmpl1 := TemplateConfig{Name: "scaffold", Description: "Project scaffold template"}
	ro.AddTemplate("scaffold", "claude", tmpl1)

	retrieved, ok := ro.GetTemplate("scaffold", "claude")
	require.True(t, ok)
	require.Equal(t, tmpl1, retrieved)

	_, ok = ro.GetTemplate("scaffold", "cursor")
	require.False(t, ok)
}

func TestResourceOverrides_AddHooks_StoresAndRetrievesHookConfigs(t *testing.T) {
	ro := &ResourceOverrides{}

	hooks1 := NamedHookConfig{Name: "pre-commit", Artifacts: []string{"lint.sh"}}
	ro.AddHooks("pre-commit", "claude", hooks1)

	retrieved, ok := ro.GetHooks("pre-commit", "claude")
	require.True(t, ok)
	require.Equal(t, hooks1, retrieved)

	_, ok = ro.GetHooks("pre-commit", "cursor")
	require.False(t, ok)
}

func TestResourceOverrides_HooksProviders_ListsProviders(t *testing.T) {
	ro := &ResourceOverrides{}
	ro.AddHooks("pre-commit", "gemini", NamedHookConfig{Name: "pre-commit"})
	ro.AddHooks("pre-commit", "claude", NamedHookConfig{Name: "pre-commit"})
	ro.AddHooks("pre-commit", "cursor", NamedHookConfig{Name: "pre-commit"})

	providers := ro.HooksProviders("pre-commit")
	require.Equal(t, []string{"claude", "cursor", "gemini"}, providers)
}

func TestResourceOverrides_AddSettings_StoresAndRetrievesSettingsConfigs(t *testing.T) {
	ro := &ResourceOverrides{}

	s1 := SettingsConfig{Name: "default"}
	ro.AddSettings("default", "claude", s1)

	retrieved, ok := ro.GetSettings("default", "claude")
	require.True(t, ok)
	require.Equal(t, s1, retrieved)

	_, ok = ro.GetSettings("default", "gemini")
	require.False(t, ok)
}

func TestResourceOverrides_SettingsProviders_ListsProviders(t *testing.T) {
	ro := &ResourceOverrides{}
	ro.AddSettings("default", "cursor", SettingsConfig{Name: "default"})
	ro.AddSettings("default", "claude", SettingsConfig{Name: "default"})

	providers := ro.SettingsProviders("default")
	require.Equal(t, []string{"claude", "cursor"}, providers)
}

func TestResourceOverrides_AddPolicy_StoresAndRetrievesPolicyConfigs(t *testing.T) {
	ro := &ResourceOverrides{}

	p1 := PolicyConfig{Name: "require-desc", Severity: "error", Target: "agent"}
	ro.AddPolicy("require-desc", "claude", p1)

	retrieved, ok := ro.GetPolicy("require-desc", "claude")
	require.True(t, ok)
	require.Equal(t, p1, retrieved)

	_, ok = ro.GetPolicy("require-desc", "copilot")
	require.False(t, ok)
}

func TestResourceOverrides_PolicyProviders_ListsProviders(t *testing.T) {
	ro := &ResourceOverrides{}
	ro.AddPolicy("require-desc", "copilot", PolicyConfig{Name: "require-desc"})
	ro.AddPolicy("require-desc", "claude", PolicyConfig{Name: "require-desc"})

	providers := ro.PolicyProviders("require-desc")
	require.Equal(t, []string{"claude", "copilot"}, providers)
}

func TestResourceOverrides_TemplateProviders_ListsProviders(t *testing.T) {
	ro := &ResourceOverrides{}
	ro.AddTemplate("scaffold", "gemini", TemplateConfig{Name: "scaffold"})
	ro.AddTemplate("scaffold", "claude", TemplateConfig{Name: "scaffold"})

	providers := ro.TemplateProviders("scaffold")
	require.Equal(t, []string{"claude", "gemini"}, providers)
}

// --- ClearableList tests ---

func TestClearableList_UnmarshalYAML_AbsentField(t *testing.T) {
	type wrapper struct {
		Items *ClearableList `yaml:"items"`
	}
	var w wrapper
	err := yaml.Unmarshal([]byte("name: test\n"), &w)
	require.NoError(t, err)
	require.Nil(t, w.Items)
}

func TestClearableList_UnmarshalYAML_ExplicitNull(t *testing.T) {
	type wrapper struct {
		Items *ClearableList `yaml:"items"`
	}
	var w wrapper
	err := yaml.Unmarshal([]byte("items: ~\n"), &w)
	require.NoError(t, err)
	require.Nil(t, w.Items)
}

func TestClearableList_UnmarshalYAML_EmptySequence(t *testing.T) {
	type wrapper struct {
		Items *ClearableList `yaml:"items"`
	}
	var w wrapper
	err := yaml.Unmarshal([]byte("items: []\n"), &w)
	require.NoError(t, err)
	require.NotNil(t, w.Items)
	require.True(t, w.Items.Cleared)
	require.Nil(t, w.Items.Values)
	require.Equal(t, 0, w.Items.Len())
	require.False(t, w.Items.IsEmpty())
}

func TestClearableList_UnmarshalYAML_PopulatedSequence(t *testing.T) {
	type wrapper struct {
		Items *ClearableList `yaml:"items"`
	}
	var w wrapper
	err := yaml.Unmarshal([]byte("items: [a, b]\n"), &w)
	require.NoError(t, err)
	require.NotNil(t, w.Items)
	require.False(t, w.Items.Cleared)
	require.Equal(t, []string{"a", "b"}, w.Items.Values)
	require.Equal(t, 2, w.Items.Len())
	require.False(t, w.Items.IsEmpty())
}

func TestClearableList_MarshalYAML_ZeroValue(t *testing.T) {
	c := ClearableList{}
	iface, err := c.MarshalYAML()
	require.NoError(t, err)
	require.Nil(t, iface)
}

func TestClearableList_MarshalYAML_Cleared(t *testing.T) {
	c := ClearableList{Cleared: true}
	out, err := yaml.Marshal(struct {
		Items *ClearableList `yaml:"items"`
	}{Items: &c})
	require.NoError(t, err)
	require.Contains(t, string(out), "items:")
}

func TestClearableList_MarshalYAML_WithValues(t *testing.T) {
	c := ClearableList{Values: []string{"x", "y"}}
	out, err := yaml.Marshal(struct {
		Items *ClearableList `yaml:"items"`
	}{Items: &c})
	require.NoError(t, err)
	require.Contains(t, string(out), "x")
	require.Contains(t, string(out), "y")
}

// --- StripInherited tests ---

func TestStripInherited_Policies(t *testing.T) {
	cfg := &XcaffoldConfig{
		ResourceScope: ResourceScope{
			Policies: map[string]PolicyConfig{
				"keep-this": {
					Name:      "keep-this",
					Inherited: false,
				},
				"remove-this": {
					Name:      "remove-this",
					Inherited: true,
				},
			},
		},
	}

	cfg.StripInherited()

	// Non-inherited policy should remain
	require.Contains(t, cfg.Policies, "keep-this")
	require.Equal(t, false, cfg.Policies["keep-this"].Inherited)

	// Inherited policy should be removed
	require.NotContains(t, cfg.Policies, "remove-this")
	require.Len(t, cfg.Policies, 1)
}

func TestStripInherited_Memory(t *testing.T) {
	cfg := &XcaffoldConfig{
		ResourceScope: ResourceScope{
			Memory: map[string]MemoryConfig{
				"keep-this": {
					Name:      "keep-this",
					Inherited: false,
				},
				"remove-this": {
					Name:      "remove-this",
					Inherited: true,
				},
			},
		},
	}

	cfg.StripInherited()

	// Non-inherited memory should remain
	require.Contains(t, cfg.Memory, "keep-this")
	require.Equal(t, false, cfg.Memory["keep-this"].Inherited)

	// Inherited memory should be removed
	require.NotContains(t, cfg.Memory, "remove-this")
	require.Len(t, cfg.Memory, 1)
}

func TestStripInherited_Contexts(t *testing.T) {
	cfg := &XcaffoldConfig{
		ResourceScope: ResourceScope{
			Contexts: map[string]ContextConfig{
				"keep-this": {
					Name:      "keep-this",
					Inherited: false,
				},
				"remove-this": {
					Name:      "remove-this",
					Inherited: true,
				},
			},
		},
	}

	cfg.StripInherited()

	// Non-inherited context should remain
	require.Contains(t, cfg.Contexts, "keep-this")
	require.Equal(t, false, cfg.Contexts["keep-this"].Inherited)

	// Inherited context should be removed
	require.NotContains(t, cfg.Contexts, "remove-this")
	require.Len(t, cfg.Contexts, 1)
}

func TestStripInherited_Templates(t *testing.T) {
	cfg := &XcaffoldConfig{
		ResourceScope: ResourceScope{
			Templates: map[string]TemplateConfig{
				"keep-this": {
					Name:      "keep-this",
					Inherited: false,
				},
				"remove-this": {
					Name:      "remove-this",
					Inherited: true,
				},
			},
		},
	}

	cfg.StripInherited()

	// Non-inherited template should remain
	require.Contains(t, cfg.Templates, "keep-this")
	require.Equal(t, false, cfg.Templates["keep-this"].Inherited)

	// Inherited template should be removed
	require.NotContains(t, cfg.Templates, "remove-this")
	require.Len(t, cfg.Templates, 1)
}

func TestStripInherited_All11Types(t *testing.T) {
	cfg := &XcaffoldConfig{
		ResourceScope: ResourceScope{
			Agents: map[string]AgentConfig{
				"keep-agent": {Name: "keep-agent", Inherited: false},
				"rm-agent":   {Name: "rm-agent", Inherited: true},
			},
			Skills: map[string]SkillConfig{
				"keep-skill": {Name: "keep-skill", Inherited: false},
				"rm-skill":   {Name: "rm-skill", Inherited: true},
			},
			Rules: map[string]RuleConfig{
				"keep-rule": {Name: "keep-rule", Inherited: false},
				"rm-rule":   {Name: "rm-rule", Inherited: true},
			},
			MCP: map[string]MCPConfig{
				"keep-mcp": {Name: "keep-mcp", Inherited: false},
				"rm-mcp":   {Name: "rm-mcp", Inherited: true},
			},
			Workflows: map[string]WorkflowConfig{
				"keep-workflow": {Name: "keep-workflow", Inherited: false},
				"rm-workflow":   {Name: "rm-workflow", Inherited: true},
			},
			Policies: map[string]PolicyConfig{
				"keep-policy": {Name: "keep-policy", Inherited: false},
				"rm-policy":   {Name: "rm-policy", Inherited: true},
			},
			Memory: map[string]MemoryConfig{
				"keep-memory": {Name: "keep-memory", Inherited: false},
				"rm-memory":   {Name: "rm-memory", Inherited: true},
			},
			Contexts: map[string]ContextConfig{
				"keep-context": {Name: "keep-context", Inherited: false},
				"rm-context":   {Name: "rm-context", Inherited: true},
			},
			Templates: map[string]TemplateConfig{
				"keep-template": {Name: "keep-template", Inherited: false},
				"rm-template":   {Name: "rm-template", Inherited: true},
			},
		},
		Hooks: map[string]NamedHookConfig{
			"keep-hooks": {Name: "keep-hooks", Inherited: false},
			"rm-hooks":   {Name: "rm-hooks", Inherited: true},
		},
		Settings: map[string]SettingsConfig{
			"keep-settings": {Name: "keep-settings", Inherited: false},
			"rm-settings":   {Name: "rm-settings", Inherited: true},
		},
	}

	cfg.StripInherited()

	// Verify all 11 types: agents, skills, rules, mcp, workflows, hooks,
	// settings, policies, memory, contexts, templates
	require.Len(t, cfg.Agents, 1)
	require.Contains(t, cfg.Agents, "keep-agent")
	require.NotContains(t, cfg.Agents, "rm-agent")

	require.Len(t, cfg.Skills, 1)
	require.Contains(t, cfg.Skills, "keep-skill")
	require.NotContains(t, cfg.Skills, "rm-skill")

	require.Len(t, cfg.Rules, 1)
	require.Contains(t, cfg.Rules, "keep-rule")
	require.NotContains(t, cfg.Rules, "rm-rule")

	require.Len(t, cfg.MCP, 1)
	require.Contains(t, cfg.MCP, "keep-mcp")
	require.NotContains(t, cfg.MCP, "rm-mcp")

	require.Len(t, cfg.Workflows, 1)
	require.Contains(t, cfg.Workflows, "keep-workflow")
	require.NotContains(t, cfg.Workflows, "rm-workflow")

	require.Len(t, cfg.Hooks, 1)
	require.Contains(t, cfg.Hooks, "keep-hooks")
	require.NotContains(t, cfg.Hooks, "rm-hooks")

	require.Len(t, cfg.Settings, 1)
	require.Contains(t, cfg.Settings, "keep-settings")
	require.NotContains(t, cfg.Settings, "rm-settings")

	require.Len(t, cfg.Policies, 1)
	require.Contains(t, cfg.Policies, "keep-policy")
	require.NotContains(t, cfg.Policies, "rm-policy")

	require.Len(t, cfg.Memory, 1)
	require.Contains(t, cfg.Memory, "keep-memory")
	require.NotContains(t, cfg.Memory, "rm-memory")

	require.Len(t, cfg.Contexts, 1)
	require.Contains(t, cfg.Contexts, "keep-context")
	require.NotContains(t, cfg.Contexts, "rm-context")

	require.Len(t, cfg.Templates, 1)
	require.Contains(t, cfg.Templates, "keep-template")
	require.NotContains(t, cfg.Templates, "rm-template")
}

// Helper function for creating bool pointers in tests
func boolPtr(b bool) *bool {
	return &b
}
