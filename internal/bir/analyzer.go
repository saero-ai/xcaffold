package bir

import "github.com/saero-ai/xcaffold/internal/ast"

// Analyze converts an XcaffoldConfig into a ProjectIR.
// Phase 0: pass-through that catalogs each primitive without semantic analysis.
func Analyze(config *ast.XcaffoldConfig, baseDir string) (*ProjectIR, error) {
	ir := &ProjectIR{}

	for id, agent := range config.Agents {
		ir.Units = append(ir.Units, SemanticUnit{
			ID: id, SourceKind: SourceAgent, ResolvedBody: agent.Instructions,
		})
	}
	for id, skill := range config.Skills {
		ir.Units = append(ir.Units, SemanticUnit{
			ID: id, SourceKind: SourceSkill, ResolvedBody: skill.Instructions,
		})
	}
	for id, rule := range config.Rules {
		ir.Units = append(ir.Units, SemanticUnit{
			ID: id, SourceKind: SourceRule, ResolvedBody: rule.Instructions,
		})
	}
	for id, wf := range config.Workflows {
		ir.Units = append(ir.Units, SemanticUnit{
			ID: id, SourceKind: SourceWorkflow, ResolvedBody: wf.Instructions,
		})
	}

	return ir, nil
}
