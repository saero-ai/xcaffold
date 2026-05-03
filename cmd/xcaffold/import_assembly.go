package main

import (
	"reflect"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
)

func assembleMultiProviderResources(providerConfigs map[string]*ast.XcaffoldConfig, result *ast.XcaffoldConfig) {
	assembleAgents(providerConfigs, result)
	assembleSkills(providerConfigs, result)
	assembleRules(providerConfigs, result)
	assembleWorkflows(providerConfigs, result)

	// MCP, memory, hooks, settings: union (first-seen wins)
	for provider, cfg := range providerConfigs {
		for id, mc := range cfg.MCP {
			if _, exists := result.MCP[id]; !exists {
				result.MCP[id] = mc
				_ = provider
			}
		}
		if cfg.Memory != nil {
			if result.Memory == nil {
				result.Memory = make(map[string]ast.MemoryConfig)
			}
			for k, mc := range cfg.Memory {
				if _, exists := result.Memory[k]; !exists {
					result.Memory[k] = mc
				}
			}
		}
		for hookName, namedHook := range cfg.Hooks {
			if result.Hooks == nil {
				result.Hooks = make(map[string]ast.NamedHookConfig)
			}
			if _, exists := result.Hooks[hookName]; !exists {
				result.Hooks[hookName] = namedHook
			}
		}
		for name, sc := range cfg.Settings {
			if result.Settings == nil {
				result.Settings = make(map[string]ast.SettingsConfig)
			}
			if _, exists := result.Settings[name]; !exists {
				result.Settings[name] = sc
			}
		}
	}
}

func assembleAgents(providerConfigs map[string]*ast.XcaffoldConfig, result *ast.XcaffoldConfig) {
	byName := make(map[string]map[string]ast.AgentConfig)
	for provider, cfg := range providerConfigs {
		for name, agent := range cfg.Agents {
			if byName[name] == nil {
				byName[name] = make(map[string]ast.AgentConfig)
			}
			byName[name][provider] = agent
		}
	}
	for name, providerAgents := range byName {
		if len(providerAgents) == 1 {
			for provider, agent := range providerAgents {
				if agent.Targets == nil {
					agent.Targets = make(map[string]ast.TargetOverride)
				}
				agent.Targets[provider] = ast.TargetOverride{}
				result.Agents[name] = agent
			}
			continue
		}
		if agentConfigsIdentical(providerAgents) {
			for _, agent := range providerAgents {
				agent.Targets = buildTargetsMap(providerAgents)
				result.Agents[name] = agent
				break
			}
			continue
		}
		// Different: first provider becomes base, others become overrides
		base, overrides := splitAgentOverrides(providerAgents)
		base.Targets = buildTargetsMap(providerAgents)
		result.Agents[name] = base
		if result.Overrides == nil {
			result.Overrides = &ast.ResourceOverrides{}
		}
		for provider, override := range overrides {
			result.Overrides.AddAgent(name, provider, override)
		}
	}
}

func assembleSkills(providerConfigs map[string]*ast.XcaffoldConfig, result *ast.XcaffoldConfig) {
	byName := make(map[string]map[string]ast.SkillConfig)
	for provider, cfg := range providerConfigs {
		for name, skill := range cfg.Skills {
			if byName[name] == nil {
				byName[name] = make(map[string]ast.SkillConfig)
			}
			byName[name][provider] = skill
		}
	}
	for name, providerSkills := range byName {
		if len(providerSkills) == 1 {
			for provider, skill := range providerSkills {
				if skill.Targets == nil {
					skill.Targets = make(map[string]ast.TargetOverride)
				}
				skill.Targets[provider] = ast.TargetOverride{}
				result.Skills[name] = skill
			}
			continue
		}
		if skillConfigsIdentical(providerSkills) {
			for _, skill := range providerSkills {
				skill.Targets = buildTargetsMap(providerSkills)
				result.Skills[name] = skill
				break
			}
			continue
		}
		base, overrides := splitSkillOverrides(providerSkills)
		base.Targets = buildTargetsMap(providerSkills)
		result.Skills[name] = base
		if result.Overrides == nil {
			result.Overrides = &ast.ResourceOverrides{}
		}
		for provider, override := range overrides {
			result.Overrides.AddSkill(name, provider, override)
		}
	}
}

func assembleRules(providerConfigs map[string]*ast.XcaffoldConfig, result *ast.XcaffoldConfig) {
	byName := make(map[string]map[string]ast.RuleConfig)
	for provider, cfg := range providerConfigs {
		for name, rule := range cfg.Rules {
			if byName[name] == nil {
				byName[name] = make(map[string]ast.RuleConfig)
			}
			byName[name][provider] = rule
		}
	}
	for name, providerRules := range byName {
		if len(providerRules) == 1 {
			for provider, rule := range providerRules {
				if rule.Targets == nil {
					rule.Targets = make(map[string]ast.TargetOverride)
				}
				rule.Targets[provider] = ast.TargetOverride{}
				result.Rules[name] = rule
			}
			continue
		}
		if ruleConfigsIdentical(providerRules) {
			for _, rule := range providerRules {
				rule.Targets = buildTargetsMap(providerRules)
				result.Rules[name] = rule
				break
			}
			continue
		}
		base, overrides := splitRuleOverrides(providerRules)
		base.Targets = buildTargetsMap(providerRules)
		result.Rules[name] = base
		if result.Overrides == nil {
			result.Overrides = &ast.ResourceOverrides{}
		}
		for provider, override := range overrides {
			result.Overrides.AddRule(name, provider, override)
		}
	}
}

func assembleWorkflows(providerConfigs map[string]*ast.XcaffoldConfig, result *ast.XcaffoldConfig) {
	byName := make(map[string]map[string]ast.WorkflowConfig)
	for provider, cfg := range providerConfigs {
		for name, wf := range cfg.Workflows {
			if byName[name] == nil {
				byName[name] = make(map[string]ast.WorkflowConfig)
			}
			byName[name][provider] = wf
		}
	}
	for name, providerWFs := range byName {
		if len(providerWFs) == 1 {
			for provider, wf := range providerWFs {
				if wf.Targets == nil {
					wf.Targets = make(map[string]ast.TargetOverride)
				}
				wf.Targets[provider] = ast.TargetOverride{}
				result.Workflows[name] = wf
			}
			continue
		}
		if workflowConfigsIdentical(providerWFs) {
			for _, wf := range providerWFs {
				wf.Targets = buildTargetsMap(providerWFs)
				result.Workflows[name] = wf
				break
			}
			continue
		}
		base, overrides := splitWorkflowOverrides(providerWFs)
		base.Targets = buildTargetsMap(providerWFs)
		result.Workflows[name] = base
		if result.Overrides == nil {
			result.Overrides = &ast.ResourceOverrides{}
		}
		for provider, override := range overrides {
			result.Overrides.AddWorkflow(name, provider, override)
		}
	}
}

func buildTargetsMap[T any](providers map[string]T) map[string]ast.TargetOverride {
	targets := make(map[string]ast.TargetOverride, len(providers))
	for provider := range providers {
		targets[provider] = ast.TargetOverride{}
	}
	return targets
}

func agentConfigsIdentical(configs map[string]ast.AgentConfig) bool {
	var ref ast.AgentConfig
	first := true
	for _, cfg := range configs {
		if first {
			ref = cfg
			first = false
			continue
		}
		// Zero out Name since it's expected to differ
		cfg.Name = ""
		ref.Name = ""
		if !reflect.DeepEqual(cfg, ref) {
			return false
		}
	}
	return true
}

func skillConfigsIdentical(configs map[string]ast.SkillConfig) bool {
	var ref ast.SkillConfig
	first := true
	for _, cfg := range configs {
		if first {
			ref = cfg
			first = false
			continue
		}
		// Zero out Name since it's expected to differ
		cfg.Name = ""
		ref.Name = ""
		if !reflect.DeepEqual(cfg, ref) {
			return false
		}
	}
	return true
}

func ruleConfigsIdentical(configs map[string]ast.RuleConfig) bool {
	var ref ast.RuleConfig
	first := true
	for _, cfg := range configs {
		if first {
			ref = cfg
			first = false
			continue
		}
		// Zero out Name since it's expected to differ
		cfg.Name = ""
		ref.Name = ""
		if !reflect.DeepEqual(cfg, ref) {
			return false
		}
	}
	return true
}

func workflowConfigsIdentical(configs map[string]ast.WorkflowConfig) bool {
	var ref ast.WorkflowConfig
	first := true
	for _, cfg := range configs {
		if first {
			ref = cfg
			first = false
			continue
		}
		// Zero out Name since it's expected to differ
		cfg.Name = ""
		ref.Name = ""
		if !reflect.DeepEqual(cfg, ref) {
			return false
		}
	}
	return true
}

func hookConfigsIdentical(configs map[string]ast.NamedHookConfig) bool {
	var ref ast.NamedHookConfig
	first := true
	for _, cfg := range configs {
		if first {
			ref = cfg
			first = false
			continue
		}
		cfg.Name = ""
		ref.Name = ""
		if !reflect.DeepEqual(cfg, ref) {
			return false
		}
	}
	return true
}

// splitHookOverrides scores hooks: Events non-nil+non-empty (+5), each Artifact (+1).
func splitHookOverrides(configs map[string]ast.NamedHookConfig) (ast.NamedHookConfig, map[string]ast.NamedHookConfig) {
	scores := make(map[string]int, len(configs))
	for provider, cfg := range configs {
		s := 0
		if cfg.Events != nil && len(cfg.Events) > 0 {
			s += 5
		}
		s += len(cfg.Artifacts)
		scores[provider] = s
	}
	baseProv := selectBaseProvider(scores)
	base := configs[baseProv]
	overrides := make(map[string]ast.NamedHookConfig, len(configs)-1)
	for provider, cfg := range configs {
		if provider == baseProv {
			continue
		}
		overrides[provider] = cfg
	}
	return base, overrides
}

// assembleHooks performs 2-pass assembly of hooks like assembleAgents but no Targets field.
func assembleHooks(providerConfigs map[string]*ast.XcaffoldConfig, result *ast.XcaffoldConfig) {
	byName := make(map[string]map[string]ast.NamedHookConfig)
	for provider, cfg := range providerConfigs {
		for name, hook := range cfg.Hooks {
			if byName[name] == nil {
				byName[name] = make(map[string]ast.NamedHookConfig)
			}
			byName[name][provider] = hook
		}
	}
	for name, providerHooks := range byName {
		if len(providerHooks) == 1 {
			for _, hook := range providerHooks {
				if result.Hooks == nil {
					result.Hooks = make(map[string]ast.NamedHookConfig)
				}
				result.Hooks[name] = hook
			}
			continue
		}
		if hookConfigsIdentical(providerHooks) {
			for _, hook := range providerHooks {
				if result.Hooks == nil {
					result.Hooks = make(map[string]ast.NamedHookConfig)
				}
				result.Hooks[name] = hook
				break
			}
			continue
		}
		base, overrides := splitHookOverrides(providerHooks)
		if result.Hooks == nil {
			result.Hooks = make(map[string]ast.NamedHookConfig)
		}
		result.Hooks[name] = base
		if result.Overrides == nil {
			result.Overrides = &ast.ResourceOverrides{}
		}
		for provider, override := range overrides {
			result.Overrides.AddHooks(name, provider, override)
		}
	}
}

// selectBaseProvider selects the provider with the lowest score from the score map.
// Ties are broken alphabetically (provider name sort order).
func selectBaseProvider(scores map[string]int) string {
	var selected string
	var minScore int = -1

	for provider, score := range scores {
		if minScore == -1 || score < minScore || (score == minScore && provider < selected) {
			selected = provider
			minScore = score
		}
	}
	return selected
}

func splitAgentOverrides(configs map[string]ast.AgentConfig) (ast.AgentConfig, map[string]ast.AgentConfig) {
	// Score each provider by provider-specificity: Hooks (+10), Model (+1), Tools (+1)
	scores := make(map[string]int, len(configs))
	for provider, cfg := range configs {
		s := 0
		if len(cfg.Hooks) > 0 {
			s += 10
		}
		if cfg.Model != "" {
			s++
		}
		if len(cfg.Tools) > 0 {
			s++
		}
		scores[provider] = s
	}

	// Select base provider (lowest score, alphabetical tie-break)
	baseProv := selectBaseProvider(scores)

	base := configs[baseProv]
	overrides := make(map[string]ast.AgentConfig, len(configs)-1)
	for provider, cfg := range configs {
		if provider == baseProv {
			continue
		}
		// Strip body if identical to base
		if strings.TrimSpace(cfg.Body) == strings.TrimSpace(base.Body) {
			cfg.Body = ""
		}
		overrides[provider] = cfg
	}
	return base, overrides
}

func splitSkillOverrides(configs map[string]ast.SkillConfig) (ast.SkillConfig, map[string]ast.SkillConfig) {
	// Score each provider by provider-specificity: AllowedTools (+1)
	scores := make(map[string]int, len(configs))
	for provider, cfg := range configs {
		s := 0
		if len(cfg.AllowedTools) > 0 {
			s++
		}
		scores[provider] = s
	}

	// Select base provider (lowest score, alphabetical tie-break)
	baseProv := selectBaseProvider(scores)

	base := configs[baseProv]
	overrides := make(map[string]ast.SkillConfig, len(configs)-1)
	for provider, cfg := range configs {
		if provider == baseProv {
			continue
		}
		// Strip body if identical to base
		if strings.TrimSpace(cfg.Body) == strings.TrimSpace(base.Body) {
			cfg.Body = ""
		}
		overrides[provider] = cfg
	}
	return base, overrides
}

func splitRuleOverrides(configs map[string]ast.RuleConfig) (ast.RuleConfig, map[string]ast.RuleConfig) {
	// Rules have no provider-specific fields; all scores are 0
	// Use alphabetical tie-break for deterministic selection
	scores := make(map[string]int, len(configs))
	for provider := range configs {
		scores[provider] = 0
	}

	// Select base provider (alphabetical)
	baseProv := selectBaseProvider(scores)

	base := configs[baseProv]
	overrides := make(map[string]ast.RuleConfig, len(configs)-1)
	for provider, cfg := range configs {
		if provider == baseProv {
			continue
		}
		// Strip body if identical to base
		if strings.TrimSpace(cfg.Body) == strings.TrimSpace(base.Body) {
			cfg.Body = ""
		}
		overrides[provider] = cfg
	}
	return base, overrides
}

func splitWorkflowOverrides(configs map[string]ast.WorkflowConfig) (ast.WorkflowConfig, map[string]ast.WorkflowConfig) {
	// Workflows have no provider-specific fields; all scores are 0
	// Use alphabetical tie-break for deterministic selection
	scores := make(map[string]int, len(configs))
	for provider := range configs {
		scores[provider] = 0
	}

	// Select base provider (alphabetical)
	baseProv := selectBaseProvider(scores)

	base := configs[baseProv]
	overrides := make(map[string]ast.WorkflowConfig, len(configs)-1)
	for provider, cfg := range configs {
		if provider == baseProv {
			continue
		}
		// Strip body if identical to base
		if strings.TrimSpace(cfg.Body) == strings.TrimSpace(base.Body) {
			cfg.Body = ""
		}
		overrides[provider] = cfg
	}
	return base, overrides
}
