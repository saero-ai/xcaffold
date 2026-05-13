package compiler

import (
	"fmt"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
)

// CodeResourceTargetSkipped is the machine-readable fidelity code emitted when a
// resource is removed because its Targets map does not include the current target.
const CodeResourceTargetSkipped = "RESOURCE_TARGET_SKIPPED"

// resolveTargetOverrides applies provider-specific overrides and target filtering
// to every resource in config, mutating config in place. It returns a slice of
// FidelityNote warnings for each resource that was removed because its Targets
// map did not include target.
//
// Processing order per resource type:
//  1. Apply override from config.Overrides (if present) by merging into base.
//  2. If the resource has a non-empty Targets map and target is not in it, remove
//     the resource and emit a warning note.
//  3. If the resource has no Targets map, keep it unchanged.
//
// MCPConfig has a Targets field; both overrides and target filtering are applied.
// config.Overrides may be nil — all nil-checks are handled internally.
func resolveTargetOverrides(config *ast.XcaffoldConfig, target string) []renderer.FidelityNote {
	var notes []renderer.FidelityNote
	notes = append(notes, resolveAgentOverrides(config, target)...)
	notes = append(notes, resolveSkillOverrides(config, target)...)
	notes = append(notes, resolveRuleOverrides(config, target)...)
	notes = append(notes, resolveWorkflowOverrides(config, target)...)
	notes = append(notes, resolveMCPOverrides(config, target)...)
	notes = append(notes, resolveHookOverrides(config, target)...)
	notes = append(notes, resolveSettingsOverrides(config, target)...)
	resolvePolicyOverrides(config, target)
	resolveTemplateOverrides(config, target)
	resolveContextOverrides(config, target)
	return notes
}

// resolveAgentOverrides applies overrides and target filtering for agents.
func resolveAgentOverrides(config *ast.XcaffoldConfig, target string) []renderer.FidelityNote {
	var notes []renderer.FidelityNote
	for name, agent := range config.Agents {
		if override, ok := config.Overrides.GetAgent(name, target); ok {
			agent = mergeAgentConfig(agent, override)
			config.Agents[name] = agent
		}
		if len(agent.Targets) > 0 {
			if _, listed := agent.Targets[target]; !listed {
				delete(config.Agents, name)
				notes = append(notes, newSkippedNote(target, "agent", name))
			}
		}
	}
	return notes
}

// resolveSkillOverrides applies overrides and target filtering for skills.
func resolveSkillOverrides(config *ast.XcaffoldConfig, target string) []renderer.FidelityNote {
	var notes []renderer.FidelityNote
	for name, skill := range config.Skills {
		if override, ok := config.Overrides.GetSkill(name, target); ok {
			skill = mergeSkillConfig(skill, override)
			config.Skills[name] = skill
		}
		if len(skill.Targets) > 0 {
			if _, listed := skill.Targets[target]; !listed {
				delete(config.Skills, name)
				notes = append(notes, newSkippedNote(target, "skill", name))
			}
		}
	}
	return notes
}

// resolveRuleOverrides applies overrides and target filtering for rules.
func resolveRuleOverrides(config *ast.XcaffoldConfig, target string) []renderer.FidelityNote {
	var notes []renderer.FidelityNote
	for name, rule := range config.Rules {
		if override, ok := config.Overrides.GetRule(name, target); ok {
			rule = mergeRuleConfig(rule, override)
			config.Rules[name] = rule
		}
		if len(rule.Targets) > 0 {
			if _, listed := rule.Targets[target]; !listed {
				delete(config.Rules, name)
				notes = append(notes, newSkippedNote(target, "rule", name))
			}
		}
	}
	return notes
}

// resolveWorkflowOverrides applies overrides and target filtering for workflows.
func resolveWorkflowOverrides(config *ast.XcaffoldConfig, target string) []renderer.FidelityNote {
	var notes []renderer.FidelityNote
	for name, workflow := range config.Workflows {
		if override, ok := config.Overrides.GetWorkflow(name, target); ok {
			workflow = mergeWorkflowConfig(workflow, override)
			config.Workflows[name] = workflow
		}
		if len(workflow.Targets) > 0 {
			if _, listed := workflow.Targets[target]; !listed {
				delete(config.Workflows, name)
				notes = append(notes, newSkippedNote(target, "workflow", name))
			}
		}
	}
	return notes
}

// resolveMCPOverrides applies overrides and target filtering for MCP configs.
func resolveMCPOverrides(config *ast.XcaffoldConfig, target string) []renderer.FidelityNote {
	var notes []renderer.FidelityNote
	for name, mcp := range config.MCP {
		if override, ok := config.Overrides.GetMCP(name, target); ok {
			mcp = mergeMCPConfig(mcp, override)
			config.MCP[name] = mcp
		}
		if len(mcp.Targets) > 0 {
			if _, listed := mcp.Targets[target]; !listed {
				delete(config.MCP, name)
				notes = append(notes, newSkippedNote(target, "mcp", name))
			}
		}
	}
	return notes
}

// resolveHookOverrides applies overrides and target filtering for hooks.
func resolveHookOverrides(config *ast.XcaffoldConfig, target string) []renderer.FidelityNote {
	var notes []renderer.FidelityNote
	for name, hook := range config.Hooks {
		if override, ok := config.Overrides.GetHooks(name, target); ok {
			hook = mergeNamedHookConfig(hook, override)
			config.Hooks[name] = hook
		}
		if len(hook.Targets) > 0 {
			if _, listed := hook.Targets[target]; !listed {
				delete(config.Hooks, name)
				notes = append(notes, newSkippedNote(target, "hooks", name))
			}
		}
	}
	return notes
}

// resolveSettingsOverrides applies overrides and target filtering for settings.
func resolveSettingsOverrides(config *ast.XcaffoldConfig, target string) []renderer.FidelityNote {
	var notes []renderer.FidelityNote
	for name, settings := range config.Settings {
		if override, ok := config.Overrides.GetSettings(name, target); ok {
			settings = mergeSettingsConfig(settings, override)
			config.Settings[name] = settings
		}
		if len(settings.Targets) > 0 {
			if _, listed := settings.Targets[target]; !listed {
				delete(config.Settings, name)
				notes = append(notes, newSkippedNote(target, "settings", name))
			}
		}
	}
	return notes
}

// resolvePolicyOverrides applies overrides for policies (no Targets map).
func resolvePolicyOverrides(config *ast.XcaffoldConfig, target string) {
	for name, policy := range config.Policies {
		if override, ok := config.Overrides.GetPolicy(name, target); ok {
			policy = mergePolicyConfig(policy, override)
			config.Policies[name] = policy
		}
	}
}

// resolveTemplateOverrides applies overrides for templates (no Targets field).
func resolveTemplateOverrides(config *ast.XcaffoldConfig, target string) {
	for name, tmpl := range config.Templates {
		if override, ok := config.Overrides.GetTemplate(name, target); ok {
			tmpl = mergeTemplateConfig(tmpl, override)
			config.Templates[name] = tmpl
		}
	}
}

// resolveContextOverrides applies overrides for contexts (Targets is []string, not map).
func resolveContextOverrides(config *ast.XcaffoldConfig, target string) {
	for name, ctx := range config.Contexts {
		if override, ok := config.Overrides.GetContext(name, target); ok {
			ctx = mergeContextConfig(ctx, override)
			config.Contexts[name] = ctx
		}
	}
}

// newSkippedNote constructs a warning FidelityNote for a resource that was
// excluded because the current target was not in its declared Targets map.
func newSkippedNote(target, kind, resource string) renderer.FidelityNote {
	return renderer.FidelityNote{
		Level:      renderer.LevelWarning,
		Target:     target,
		Kind:       kind,
		Resource:   resource,
		Code:       CodeResourceTargetSkipped,
		Reason:     fmt.Sprintf("%s %q is not declared for target %q and will be skipped", kind, resource, target),
		Mitigation: fmt.Sprintf("Add %q to the resource's targets map to include it for this target", target),
	}
}
