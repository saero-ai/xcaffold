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
// MCPConfig has no Targets field; only overrides are applied for MCP resources.
// config.Overrides may be nil — all nil-checks are handled internally.
func resolveTargetOverrides(config *ast.XcaffoldConfig, target string) []renderer.FidelityNote {
	var notes []renderer.FidelityNote

	// --- Agents ---
	for name, agent := range config.Agents {
		// Apply override if present.
		if override, ok := config.Overrides.GetAgent(name, target); ok {
			agent = mergeAgentConfig(agent, override)
			config.Agents[name] = agent
		}
		// Filter: non-empty Targets map that does not include target → remove.
		if len(agent.Targets) > 0 {
			if _, listed := agent.Targets[target]; !listed {
				delete(config.Agents, name)
				notes = append(notes, newSkippedNote(target, "agent", name))
				continue
			}
		}
	}

	// --- Skills ---
	for name, skill := range config.Skills {
		if override, ok := config.Overrides.GetSkill(name, target); ok {
			skill = mergeSkillConfig(skill, override)
			config.Skills[name] = skill
		}
		if len(skill.Targets) > 0 {
			if _, listed := skill.Targets[target]; !listed {
				delete(config.Skills, name)
				notes = append(notes, newSkippedNote(target, "skill", name))
				continue
			}
		}
	}

	// --- Rules ---
	for name, rule := range config.Rules {
		if override, ok := config.Overrides.GetRule(name, target); ok {
			rule = mergeRuleConfig(rule, override)
			config.Rules[name] = rule
		}
		if len(rule.Targets) > 0 {
			if _, listed := rule.Targets[target]; !listed {
				delete(config.Rules, name)
				notes = append(notes, newSkippedNote(target, "rule", name))
				continue
			}
		}
	}

	// --- Workflows ---
	for name, workflow := range config.Workflows {
		if override, ok := config.Overrides.GetWorkflow(name, target); ok {
			workflow = mergeWorkflowConfig(workflow, override)
			config.Workflows[name] = workflow
		}
		if len(workflow.Targets) > 0 {
			if _, listed := workflow.Targets[target]; !listed {
				delete(config.Workflows, name)
				notes = append(notes, newSkippedNote(target, "workflow", name))
				continue
			}
		}
	}

	// --- MCP ---
	// MCPConfig has no Targets field, so only override application is needed.
	for name, mcp := range config.MCP {
		if override, ok := config.Overrides.GetMCP(name, target); ok {
			mcp = mergeMCPConfig(mcp, override)
			config.MCP[name] = mcp
		}
	}

	return notes
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
