package renderer

import "github.com/saero-ai/xcaffold/internal/ast"

// ResolvedActivation returns the effective activation mode for a rule,
// applying the always-apply legacy alias and path-presence heuristic.
// It does NOT mutate the RuleConfig; renderers call this at compile time.
//
// Precedence (highest to lowest):
//  1. Activation field (explicit, takes precedence over all legacy fields)
//  2. AlwaysApply legacy alias (true → always, false → manual-mention)
//  3. Paths non-empty → path-glob
//  4. Default → always
func ResolvedActivation(rule ast.RuleConfig) string {
	if rule.Activation != "" {
		return rule.Activation
	}
	if rule.AlwaysApply != nil {
		if *rule.AlwaysApply {
			return ast.RuleActivationAlways
		}
		return ast.RuleActivationManualMention
	}
	if len(rule.Paths.Values) > 0 {
		return ast.RuleActivationPathGlob
	}
	return ast.RuleActivationAlways
}
