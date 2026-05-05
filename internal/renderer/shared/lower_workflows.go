// Package shared provides renderer helpers shared across provider subpackages
// (gemini, copilot, etc.) that cannot live in the parent renderer package due to
// import cycles introduced by the translator → renderer dependency chain.
package shared

import (
	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/translator"
)

// LowerWorkflows translates each workflow in config into rule and skill primitives
// via translator.TranslateWorkflow, then returns a shallow copy of config with the
// lowered primitives merged into Rules and Skills. The original config is never
// mutated. Fidelity notes from the lowering are also returned.
//
// target is the renderer's canonical name and is
// forwarded to translator.TranslateWorkflow for provider-specific lowering.
func LowerWorkflows(config *ast.XcaffoldConfig, target string) (*ast.XcaffoldConfig, []renderer.FidelityNote) {
	if len(config.Workflows) == 0 {
		return config, nil
	}

	// Shallow-copy ResourceScope so we can merge without mutating the input.
	merged := *config
	rs := config.ResourceScope

	mergedRules := make(map[string]ast.RuleConfig, len(rs.Rules))
	for k, v := range rs.Rules {
		mergedRules[k] = v
	}

	mergedSkills := make(map[string]ast.SkillConfig, len(rs.Skills))
	for k, v := range rs.Skills {
		mergedSkills[k] = v
	}

	var notes []renderer.FidelityNote

	for _, id := range renderer.SortedKeys(rs.Workflows) {
		wf := rs.Workflows[id]
		if wf.Name == "" {
			wf.Name = id
		}
		primitives, wfNotes := translator.TranslateWorkflow(&wf, target)
		notes = append(notes, wfNotes...)

		for _, p := range primitives {
			switch p.Kind {
			case "rule":
				body := p.Content
				if body == "" {
					body = p.Body
				}
				mergedRules[p.ID] = ast.RuleConfig{
					Description: wf.Description,
					Body:        body,
				}
			case "skill":
				body := p.Content
				if body == "" {
					body = p.Body
				}
				mergedSkills[p.ID] = ast.SkillConfig{
					Name: p.ID,
					Body: body,
				}
			}
		}
	}

	rs.Rules = mergedRules
	rs.Skills = mergedSkills
	merged.ResourceScope = rs
	return &merged, notes
}
