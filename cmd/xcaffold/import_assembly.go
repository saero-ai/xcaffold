package main

import (
	"reflect"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
)

func assembleMultiProviderResources(providerConfigs map[string]*ast.XcaffoldConfig, result *ast.XcaffoldConfig) {
	assembleAgents(providerConfigs, result)
	assembleSkills(providerConfigs, result)
	assembleRules(providerConfigs, result)
	assembleWorkflows(providerConfigs, result)
	assembleHooks(providerConfigs, result)
	assembleSettings(providerConfigs, result)

	for _, cfg := range providerConfigs {
		for id, mc := range cfg.MCP {
			if _, exists := result.MCP[id]; !exists {
				result.MCP[id] = mc
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

func assembleSkillsCase(name string, providerSkills map[string]ast.SkillConfig, result *ast.XcaffoldConfig) {
	if len(providerSkills) == 1 {
		for provider, skill := range providerSkills {
			if skill.Targets == nil {
				skill.Targets = make(map[string]ast.TargetOverride)
			}
			skill.Targets[provider] = ast.TargetOverride{}
			result.Skills[name] = skill
		}
		return
	}
	if skillConfigsIdentical(providerSkills) {
		for _, skill := range providerSkills {
			skill.Targets = buildTargetsMap(providerSkills)
			result.Skills[name] = skill
			break
		}
		return
	}
	base, overrides := splitSkillOverrides(providerSkills)
	base.Targets = buildTargetsMap(providerSkills)
	unionSkillArtifacts(&base, providerSkills)

	result.Skills[name] = base
	if result.Overrides == nil {
		result.Overrides = &ast.ResourceOverrides{}
	}
	for provider, override := range overrides {
		if !isEmptySkillOverride(override) {
			result.Overrides.AddSkill(name, provider, override)
		}
	}
}

func unionSkillArtifacts(base *ast.SkillConfig, providerSkills map[string]ast.SkillConfig) {
	allArtifacts := make(map[string]bool)
	for _, s := range providerSkills {
		for _, a := range s.Artifacts {
			allArtifacts[a] = true
		}
	}
	if len(allArtifacts) > 0 {
		base.Artifacts = make([]string, 0, len(allArtifacts))
		for a := range allArtifacts {
			base.Artifacts = append(base.Artifacts, a)
		}
		sort.Strings(base.Artifacts)
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
		assembleSkillsCase(name, providerSkills, result)
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
			if !isEmptyRuleOverride(override) {
				result.Overrides.AddRule(name, provider, override)
			}
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
			if !isEmptyWorkflowOverride(override) {
				result.Overrides.AddWorkflow(name, provider, override)
			}
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
		// Zero out Name since it's expected to differ
		cfg.Name = ""
		ref.Name = ""
		if !reflect.DeepEqual(cfg, ref) {
			return false
		}
	}
	return true
}

// splitHookOverrides scores hooks: Events non-nil+non-empty (+5); +1 per Artifact entry.
func splitHookOverrides(configs map[string]ast.NamedHookConfig) (ast.NamedHookConfig, map[string]ast.NamedHookConfig) {
	scores := make(map[string]int, len(configs))
	for provider, cfg := range configs {
		s := 0
		if len(cfg.Events) > 0 {
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

func assembleHooksCase(name string, providerHooks map[string]ast.NamedHookConfig, result *ast.XcaffoldConfig) {
	if len(providerHooks) == 1 {
		for _, hook := range providerHooks {
			result.Hooks[name] = hook
		}
		return
	}
	if hookConfigsIdentical(providerHooks) {
		for _, hook := range providerHooks {
			result.Hooks[name] = hook
			break
		}
		return
	}
	base, overrides := splitHookOverrides(providerHooks)
	unionHookArtifacts(&base, providerHooks)

	result.Hooks[name] = base
	if result.Overrides == nil {
		result.Overrides = &ast.ResourceOverrides{}
	}
	for provider, override := range overrides {
		result.Overrides.AddHooks(name, provider, override)
	}
}

func unionHookArtifacts(base *ast.NamedHookConfig, providerHooks map[string]ast.NamedHookConfig) {
	allArtifacts := make(map[string]bool)
	for _, h := range providerHooks {
		for _, a := range h.Artifacts {
			allArtifacts[a] = true
		}
	}
	if len(allArtifacts) > 0 {
		base.Artifacts = make([]string, 0, len(allArtifacts))
		for a := range allArtifacts {
			base.Artifacts = append(base.Artifacts, a)
		}
		sort.Strings(base.Artifacts)
	}
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
	if result.Hooks == nil {
		result.Hooks = make(map[string]ast.NamedHookConfig)
	}
	for name, providerHooks := range byName {
		assembleHooksCase(name, providerHooks, result)
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
		if len(cfg.Tools.Values) > 0 {
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

// scoreSkillSpecificity calculates the specificity score for a skill across all provider-specific fields.
func scoreSkillSpecificity(cfg ast.SkillConfig) int {
	s := 0
	if len(cfg.AllowedTools.Values) > 0 {
		s++
	}
	if cfg.DisableModelInvocation != nil {
		s++
	}
	if cfg.WhenToUse != "" {
		s++
	}
	if cfg.ArgumentHint != "" {
		s++
	}
	return s
}

func splitSkillOverrides(configs map[string]ast.SkillConfig) (ast.SkillConfig, map[string]ast.SkillConfig) {
	// Score each provider by provider-specificity across 8 fields
	scores := make(map[string]int, len(configs))
	for provider, cfg := range configs {
		scores[provider] = scoreSkillSpecificity(cfg)
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

func settingsConfigsIdentical(configs map[string]ast.SettingsConfig) bool {
	var ref ast.SettingsConfig
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

func scoreSettingsConfig(cfg ast.SettingsConfig) int {
	return scoreSettingsPointerFields(cfg) +
		scoreSettingsCollectionFields(cfg) +
		scoreSettingsStringFields(cfg)
}

func countNilPointers(ptrs ...interface{}) int {
	count := 0
	for _, p := range ptrs {
		if p != nil {
			count++
		}
	}
	return count
}

// scoreSettingsPointerFields counts non-nil boolean/struct pointer fields.
func scoreSettingsPointerFields(cfg ast.SettingsConfig) int {
	return countNilPointers(
		cfg.Agent,
		cfg.Worktree,
		cfg.AutoMode,
		cfg.CleanupPeriodDays,
		cfg.IncludeGitInstructions,
		cfg.SkipDangerousModePermissionPrompt,
		cfg.Permissions,
		cfg.Sandbox,
		cfg.AutoMemoryEnabled,
		cfg.DisableAllHooks,
		cfg.Attribution,
		cfg.StatusLine,
		cfg.RespectGitignore,
		cfg.DisableSkillShellExecution,
		cfg.AlwaysThinkingEnabled,
	)
}

// scoreSettingsCollectionFields counts non-empty map and slice fields.
func scoreSettingsCollectionFields(cfg ast.SettingsConfig) int {
	s := 0
	if len(cfg.MCPServers) > 0 {
		s++
	}
	if len(cfg.Hooks) > 0 {
		s++
	}
	if len(cfg.Env) > 0 {
		s++
	}
	if len(cfg.EnabledPlugins) > 0 {
		s++
	}
	if len(cfg.AvailableModels) > 0 {
		s++
	}
	if len(cfg.MdExcludes) > 0 {
		s++
	}
	return s
}

// scoreSettingsStringFields counts non-empty string fields.
func scoreSettingsStringFields(cfg ast.SettingsConfig) int {
	s := 0
	if cfg.EffortLevel != "" {
		s++
	}
	if cfg.DefaultShell != "" {
		s++
	}
	if cfg.Language != "" {
		s++
	}
	if cfg.OutputStyle != "" {
		s++
	}
	if cfg.PlansDirectory != "" {
		s++
	}
	if cfg.Model != "" {
		s++
	}
	if cfg.OtelHeadersHelper != "" {
		s++
	}
	if cfg.AutoMemoryDirectory != "" {
		s++
	}
	return s
}

func splitSettingsOverrides(configs map[string]ast.SettingsConfig) (ast.SettingsConfig, map[string]ast.SettingsConfig) {
	scores := make(map[string]int, len(configs))
	for provider, cfg := range configs {
		scores[provider] = scoreSettingsConfig(cfg)
	}
	baseProv := selectBaseProvider(scores)
	base := configs[baseProv]
	overrides := make(map[string]ast.SettingsConfig, len(configs)-1)
	for provider, cfg := range configs {
		if provider == baseProv {
			continue
		}
		overrides[provider] = cfg
	}
	return base, overrides
}

func assembleSettings(providerConfigs map[string]*ast.XcaffoldConfig, result *ast.XcaffoldConfig) {
	byName := make(map[string]map[string]ast.SettingsConfig)
	for provider, cfg := range providerConfigs {
		for name, sc := range cfg.Settings {
			if byName[name] == nil {
				byName[name] = make(map[string]ast.SettingsConfig)
			}
			byName[name][provider] = sc
		}
	}
	if result.Settings == nil {
		result.Settings = make(map[string]ast.SettingsConfig)
	}
	for name, providerSettings := range byName {
		if len(providerSettings) == 1 {
			for _, sc := range providerSettings {
				result.Settings[name] = sc
			}
			continue
		}
		if settingsConfigsIdentical(providerSettings) {
			for _, sc := range providerSettings {
				result.Settings[name] = sc
				break
			}
			continue
		}
		base, overrides := splitSettingsOverrides(providerSettings)
		result.Settings[name] = base
		if result.Overrides == nil {
			result.Overrides = &ast.ResourceOverrides{}
		}
		for provider, override := range overrides {
			result.Overrides.AddSettings(name, provider, override)
		}
	}
}

// isEmptySkillOverride returns true if the skill override contains no meaningful content.
// An override is empty if it has no allowed tools, and all optional fields are false/nil/empty, and the body is empty.
func isEmptySkillOverride(override ast.SkillConfig) bool {
	if len(override.AllowedTools.Values) > 0 {
		return false
	}
	if override.DisableModelInvocation != nil {
		return false
	}
	if override.WhenToUse != "" {
		return false
	}
	if override.ArgumentHint != "" {
		return false
	}
	if strings.TrimSpace(override.Body) != "" {
		return false
	}
	return true
}

// isEmptyRuleOverride returns true if the rule override contains no meaningful content.
// Rules have no provider-specific fields, so an override is empty if its body is empty.
func isEmptyRuleOverride(override ast.RuleConfig) bool {
	return strings.TrimSpace(override.Body) == ""
}

// isEmptyWorkflowOverride returns true if the workflow override contains no meaningful content.
// Workflows have no provider-specific fields, so an override is empty if its body is empty.
func isEmptyWorkflowOverride(override ast.WorkflowConfig) bool {
	return strings.TrimSpace(override.Body) == ""
}
