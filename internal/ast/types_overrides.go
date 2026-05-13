package ast

import (
	"sort"
)

// ResourceOverrides stores parsed .<provider>.xcaf partial configs for all 10 kinds.
// Keyed as [kind][name][provider] → config struct. Populated by the import pipeline
// during filesystem scanning of <config-dir>.<provider>.xcaf files.
// Never serialized; used by the compiler for provider-specific config merging.
type ResourceOverrides struct {
	Agent    map[string]map[string]AgentConfig     `json:"-"`
	Skill    map[string]map[string]SkillConfig     `json:"-"`
	Rule     map[string]map[string]RuleConfig      `json:"-"`
	Workflow map[string]map[string]WorkflowConfig  `json:"-"`
	MCP      map[string]map[string]MCPConfig       `json:"-"`
	Hooks    map[string]map[string]NamedHookConfig `json:"-"`
	Settings map[string]map[string]SettingsConfig  `json:"-"`
	Policy   map[string]map[string]PolicyConfig    `json:"-"`
	Template map[string]map[string]TemplateConfig  `json:"-"`
	Context  map[string]map[string]ContextConfig   `json:"-"`
}

// AddAgent stores an AgentConfig override keyed by [name][provider].
func (r *ResourceOverrides) AddAgent(name, provider string, cfg AgentConfig) {
	if r.Agent == nil {
		r.Agent = make(map[string]map[string]AgentConfig)
	}
	if r.Agent[name] == nil {
		r.Agent[name] = make(map[string]AgentConfig)
	}
	r.Agent[name][provider] = cfg
}

// GetAgent retrieves an AgentConfig override by [name][provider].
func (r *ResourceOverrides) GetAgent(name, provider string) (AgentConfig, bool) {
	if r == nil || r.Agent == nil {
		return AgentConfig{}, false
	}
	if r.Agent[name] == nil {
		return AgentConfig{}, false
	}
	cfg, ok := r.Agent[name][provider]
	return cfg, ok
}

// AgentProviders returns a sorted list of provider names for a given agent.
func (r *ResourceOverrides) AgentProviders(name string) []string {
	if r == nil || r.Agent == nil || r.Agent[name] == nil {
		return nil
	}
	var providers []string
	for p := range r.Agent[name] {
		providers = append(providers, p)
	}
	sort.Strings(providers)
	return providers
}

// AddSkill stores a SkillConfig override keyed by [name][provider].
func (r *ResourceOverrides) AddSkill(name, provider string, cfg SkillConfig) {
	if r.Skill == nil {
		r.Skill = make(map[string]map[string]SkillConfig)
	}
	if r.Skill[name] == nil {
		r.Skill[name] = make(map[string]SkillConfig)
	}
	r.Skill[name][provider] = cfg
}

// GetSkill retrieves a SkillConfig override by [name][provider].
func (r *ResourceOverrides) GetSkill(name, provider string) (SkillConfig, bool) {
	if r == nil || r.Skill == nil {
		return SkillConfig{}, false
	}
	if r.Skill[name] == nil {
		return SkillConfig{}, false
	}
	cfg, ok := r.Skill[name][provider]
	return cfg, ok
}

// SkillProviders returns a sorted list of provider names for a given skill.
func (r *ResourceOverrides) SkillProviders(name string) []string {
	if r == nil || r.Skill == nil || r.Skill[name] == nil {
		return nil
	}
	var providers []string
	for p := range r.Skill[name] {
		providers = append(providers, p)
	}
	sort.Strings(providers)
	return providers
}

// AddRule stores a RuleConfig override keyed by [name][provider].
func (r *ResourceOverrides) AddRule(name, provider string, cfg RuleConfig) {
	if r.Rule == nil {
		r.Rule = make(map[string]map[string]RuleConfig)
	}
	if r.Rule[name] == nil {
		r.Rule[name] = make(map[string]RuleConfig)
	}
	r.Rule[name][provider] = cfg
}

// GetRule retrieves a RuleConfig override by [name][provider].
func (r *ResourceOverrides) GetRule(name, provider string) (RuleConfig, bool) {
	if r == nil || r.Rule == nil {
		return RuleConfig{}, false
	}
	if r.Rule[name] == nil {
		return RuleConfig{}, false
	}
	cfg, ok := r.Rule[name][provider]
	return cfg, ok
}

// RuleProviders returns a sorted list of provider names for a given rule.
func (r *ResourceOverrides) RuleProviders(name string) []string {
	if r == nil || r.Rule == nil || r.Rule[name] == nil {
		return nil
	}
	var providers []string
	for p := range r.Rule[name] {
		providers = append(providers, p)
	}
	sort.Strings(providers)
	return providers
}

// AddWorkflow stores a WorkflowConfig override keyed by [name][provider].
func (r *ResourceOverrides) AddWorkflow(name, provider string, cfg WorkflowConfig) {
	if r.Workflow == nil {
		r.Workflow = make(map[string]map[string]WorkflowConfig)
	}
	if r.Workflow[name] == nil {
		r.Workflow[name] = make(map[string]WorkflowConfig)
	}
	r.Workflow[name][provider] = cfg
}

// GetWorkflow retrieves a WorkflowConfig override by [name][provider].
func (r *ResourceOverrides) GetWorkflow(name, provider string) (WorkflowConfig, bool) {
	if r == nil || r.Workflow == nil {
		return WorkflowConfig{}, false
	}
	if r.Workflow[name] == nil {
		return WorkflowConfig{}, false
	}
	cfg, ok := r.Workflow[name][provider]
	return cfg, ok
}

// WorkflowProviders returns a sorted list of provider names for a given workflow.
func (r *ResourceOverrides) WorkflowProviders(name string) []string {
	if r == nil || r.Workflow == nil || r.Workflow[name] == nil {
		return nil
	}
	var providers []string
	for p := range r.Workflow[name] {
		providers = append(providers, p)
	}
	sort.Strings(providers)
	return providers
}

// AddMCP stores an MCPConfig override keyed by [name][provider].
func (r *ResourceOverrides) AddMCP(name, provider string, cfg MCPConfig) {
	if r.MCP == nil {
		r.MCP = make(map[string]map[string]MCPConfig)
	}
	if r.MCP[name] == nil {
		r.MCP[name] = make(map[string]MCPConfig)
	}
	r.MCP[name][provider] = cfg
}

// GetMCP retrieves an MCPConfig override by [name][provider].
func (r *ResourceOverrides) GetMCP(name, provider string) (MCPConfig, bool) {
	if r == nil || r.MCP == nil {
		return MCPConfig{}, false
	}
	if r.MCP[name] == nil {
		return MCPConfig{}, false
	}
	cfg, ok := r.MCP[name][provider]
	return cfg, ok
}

// MCPProviders returns a sorted list of provider names for a given MCP server.
func (r *ResourceOverrides) MCPProviders(name string) []string {
	if r == nil || r.MCP == nil || r.MCP[name] == nil {
		return nil
	}
	var providers []string
	for p := range r.MCP[name] {
		providers = append(providers, p)
	}
	sort.Strings(providers)
	return providers
}

// AddHooks stores a NamedHookConfig override keyed by [name][provider].
func (r *ResourceOverrides) AddHooks(name, provider string, cfg NamedHookConfig) {
	if r.Hooks == nil {
		r.Hooks = make(map[string]map[string]NamedHookConfig)
	}
	if r.Hooks[name] == nil {
		r.Hooks[name] = make(map[string]NamedHookConfig)
	}
	r.Hooks[name][provider] = cfg
}

// GetHooks retrieves a NamedHookConfig override by [name][provider].
func (r *ResourceOverrides) GetHooks(name, provider string) (NamedHookConfig, bool) {
	if r == nil || r.Hooks == nil {
		return NamedHookConfig{}, false
	}
	if r.Hooks[name] == nil {
		return NamedHookConfig{}, false
	}
	cfg, ok := r.Hooks[name][provider]
	return cfg, ok
}

// HooksProviders returns a sorted list of provider names for a given hooks block.
func (r *ResourceOverrides) HooksProviders(name string) []string {
	if r == nil || r.Hooks == nil || r.Hooks[name] == nil {
		return nil
	}
	var providers []string
	for p := range r.Hooks[name] {
		providers = append(providers, p)
	}
	sort.Strings(providers)
	return providers
}

// AddSettings stores a SettingsConfig override keyed by [name][provider].
func (r *ResourceOverrides) AddSettings(name, provider string, cfg SettingsConfig) {
	if r.Settings == nil {
		r.Settings = make(map[string]map[string]SettingsConfig)
	}
	if r.Settings[name] == nil {
		r.Settings[name] = make(map[string]SettingsConfig)
	}
	r.Settings[name][provider] = cfg
}

// GetSettings retrieves a SettingsConfig override by [name][provider].
func (r *ResourceOverrides) GetSettings(name, provider string) (SettingsConfig, bool) {
	if r == nil || r.Settings == nil {
		return SettingsConfig{}, false
	}
	if r.Settings[name] == nil {
		return SettingsConfig{}, false
	}
	cfg, ok := r.Settings[name][provider]
	return cfg, ok
}

// SettingsProviders returns a sorted list of provider names for a given settings block.
func (r *ResourceOverrides) SettingsProviders(name string) []string {
	if r == nil || r.Settings == nil || r.Settings[name] == nil {
		return nil
	}
	var providers []string
	for p := range r.Settings[name] {
		providers = append(providers, p)
	}
	sort.Strings(providers)
	return providers
}

// AddPolicy stores a PolicyConfig override keyed by [name][provider].
func (r *ResourceOverrides) AddPolicy(name, provider string, cfg PolicyConfig) {
	if r.Policy == nil {
		r.Policy = make(map[string]map[string]PolicyConfig)
	}
	if r.Policy[name] == nil {
		r.Policy[name] = make(map[string]PolicyConfig)
	}
	r.Policy[name][provider] = cfg
}

// GetPolicy retrieves a PolicyConfig override by [name][provider].
func (r *ResourceOverrides) GetPolicy(name, provider string) (PolicyConfig, bool) {
	if r == nil || r.Policy == nil {
		return PolicyConfig{}, false
	}
	if r.Policy[name] == nil {
		return PolicyConfig{}, false
	}
	cfg, ok := r.Policy[name][provider]
	return cfg, ok
}

// PolicyProviders returns a sorted list of provider names for a given policy.
func (r *ResourceOverrides) PolicyProviders(name string) []string {
	if r == nil || r.Policy == nil || r.Policy[name] == nil {
		return nil
	}
	var providers []string
	for p := range r.Policy[name] {
		providers = append(providers, p)
	}
	sort.Strings(providers)
	return providers
}

// AddTemplate stores a TemplateConfig override keyed by [name][provider].
func (r *ResourceOverrides) AddTemplate(name, provider string, cfg TemplateConfig) {
	if r.Template == nil {
		r.Template = make(map[string]map[string]TemplateConfig)
	}
	if r.Template[name] == nil {
		r.Template[name] = make(map[string]TemplateConfig)
	}
	r.Template[name][provider] = cfg
}

// GetTemplate retrieves a TemplateConfig override by [name][provider].
func (r *ResourceOverrides) GetTemplate(name, provider string) (TemplateConfig, bool) {
	if r == nil || r.Template == nil {
		return TemplateConfig{}, false
	}
	if r.Template[name] == nil {
		return TemplateConfig{}, false
	}
	cfg, ok := r.Template[name][provider]
	return cfg, ok
}

// TemplateProviders returns a sorted list of provider names for a given template.
func (r *ResourceOverrides) TemplateProviders(name string) []string {
	if r == nil || r.Template == nil || r.Template[name] == nil {
		return nil
	}
	var providers []string
	for p := range r.Template[name] {
		providers = append(providers, p)
	}
	sort.Strings(providers)
	return providers
}

// AddContext stores a ContextConfig override keyed by [name][provider].
func (r *ResourceOverrides) AddContext(name, provider string, cfg ContextConfig) {
	if r.Context == nil {
		r.Context = make(map[string]map[string]ContextConfig)
	}
	if r.Context[name] == nil {
		r.Context[name] = make(map[string]ContextConfig)
	}
	r.Context[name][provider] = cfg
}

// GetContext retrieves a ContextConfig override by [name][provider].
func (r *ResourceOverrides) GetContext(name, provider string) (ContextConfig, bool) {
	if r == nil || r.Context == nil {
		return ContextConfig{}, false
	}
	if r.Context[name] == nil {
		return ContextConfig{}, false
	}
	cfg, ok := r.Context[name][provider]
	return cfg, ok
}

// ContextProviders returns a sorted list of provider names for a given context.
func (r *ResourceOverrides) ContextProviders(name string) []string {
	if r == nil || r.Context == nil || r.Context[name] == nil {
		return nil
	}
	var providers []string
	for p := range r.Context[name] {
		providers = append(providers, p)
	}
	sort.Strings(providers)
	return providers
}
