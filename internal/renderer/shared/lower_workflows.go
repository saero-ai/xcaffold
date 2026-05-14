// Package shared provides renderer helpers shared across provider subpackages
// (gemini, copilot, etc.) that cannot live in the parent renderer package due to
// import cycles introduced by the translator → renderer dependency chain.
package shared

import (
	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/translator"
)

// LowerWorkflows translates each workflow in config into provider-native primitives
// via translator.TranslateWorkflow. Rule and skill primitives are merged back into
// a shallow copy of config (Rules and Skills maps). Primitives that carry their own
// output path ("custom-command", "prompt-file") are returned in the directFiles map
// keyed by p.Path. Native "workflow" primitives are merged back into the config's
// Workflows map. The original config is never mutated.
//
// target is the renderer's canonical name and is forwarded to
// translator.TranslateWorkflow for provider-specific lowering.
func LowerWorkflows(config *ast.XcaffoldConfig, target string) (*ast.XcaffoldConfig, map[string]string, []renderer.FidelityNote) {
	if len(config.Workflows) == 0 {
		return config, nil, nil
	}

	merged, rs := shallowCopyScope(config)
	var directFiles map[string]string
	var notes []renderer.FidelityNote

	for _, id := range renderer.SortedKeys(config.ResourceScope.Workflows) {
		wf := config.ResourceScope.Workflows[id]
		if wf.Name == "" {
			wf.Name = id
		}
		primitives, wfNotes := translator.TranslateWorkflow(&wf, target)
		notes = append(notes, wfNotes...)

		for _, p := range primitives {
			directFiles = applyPrimitive(p, wf, &rs, directFiles)
		}
	}

	merged.ResourceScope = rs
	return &merged, directFiles, notes
}

// shallowCopyScope returns a shallow copy of config and its ResourceScope with
// independent copies of the Rules, Skills, and Workflows maps so that mutations
// do not affect the original.
func shallowCopyScope(config *ast.XcaffoldConfig) (ast.XcaffoldConfig, ast.ResourceScope) {
	merged := *config
	rs := config.ResourceScope

	rules := make(map[string]ast.RuleConfig, len(rs.Rules))
	for k, v := range rs.Rules {
		rules[k] = v
	}
	skills := make(map[string]ast.SkillConfig, len(rs.Skills))
	for k, v := range rs.Skills {
		skills[k] = v
	}
	workflows := make(map[string]ast.WorkflowConfig, len(rs.Workflows))
	for k, v := range rs.Workflows {
		workflows[k] = v
	}

	rs.Rules = rules
	rs.Skills = skills
	rs.Workflows = workflows
	return merged, rs
}

// applyPrimitive dispatches a single TargetPrimitive produced by TranslateWorkflow
// into the appropriate output: rules/skills maps, directFiles map, or workflows map.
// It returns the updated directFiles map (nil if no path-bearing primitives).
func applyPrimitive(p translator.TargetPrimitive, wf ast.WorkflowConfig, rs *ast.ResourceScope, directFiles map[string]string) map[string]string {
	body := p.Content
	if body == "" {
		body = p.Body
	}
	switch p.Kind {
	case "rule":
		rs.Rules[p.ID] = ast.RuleConfig{Description: wf.Description, Body: body}
	case "skill":
		rs.Skills[p.ID] = ast.SkillConfig{Name: p.ID, Body: body}
	case "custom-command", "prompt-file":
		if directFiles == nil {
			directFiles = make(map[string]string)
		}
		directFiles[p.Path] = body
	case "workflow":
		wfCopy := wf
		rs.Workflows[p.ID] = wfCopy
	}
	return directFiles
}
