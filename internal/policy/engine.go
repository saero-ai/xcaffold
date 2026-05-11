package policy

import (
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
)

// Evaluate runs all user-defined policies against the config snapshot and compiled output.
func Evaluate(
	userPolicies map[string]ast.PolicyConfig,
	configSnapshot *ast.XcaffoldConfig,
	compiled *output.Output,
) []Violation {
	if len(userPolicies) == 0 {
		return nil
	}

	policyNames := make([]string, 0, len(userPolicies))
	for name := range userPolicies {
		policyNames = append(policyNames, name)
	}
	sort.Strings(policyNames)

	var violations []Violation
	for _, name := range policyNames {
		p := userPolicies[name]
		if p.Severity == SeverityOff {
			continue
		}
		vs := evaluatePolicy(p, configSnapshot, compiled)
		for i := range vs {
			vs[i].Severity = p.Severity
		}
		violations = append(violations, vs...)
	}
	return violations
}

func evaluatePolicy(p ast.PolicyConfig, config *ast.XcaffoldConfig, compiled *output.Output) []Violation {
	switch p.Target {
	case "agent":
		return evaluateMapResources(p, config.Agents, func(a ast.AgentConfig) fieldAccessor {
			return newAgentAccessor(a)
		})
	case "skill":
		return evaluateMapResources(p, config.Skills, func(s ast.SkillConfig) fieldAccessor {
			return newSkillAccessor(s)
		})
	case "rule":
		return evaluateMapResources(p, config.Rules, func(r ast.RuleConfig) fieldAccessor {
			return newRuleAccessor(r)
		})
	case "output":
		return evaluateOutput(p, compiled)
	case "settings":
		return evaluateSettings(p, compiled)
	default:
		return nil
	}
}

func evaluateMapResources[T any](p ast.PolicyConfig, resources map[string]T, newAccessor func(T) fieldAccessor) []Violation {
	// Sort resource names for deterministic evaluation order.
	names := make([]string, 0, len(resources))
	for name := range resources {
		names = append(names, name)
	}
	sort.Strings(names)

	var violations []Violation
	for _, name := range names {
		resource := resources[name]
		acc := newAccessor(resource)
		if !matchResource(p.Match, name, acc) {
			continue
		}
		for _, req := range p.Require {
			if v := evaluateRequire(p.Name, name, req, acc); v != nil {
				violations = append(violations, *v)
			}
		}
	}
	return violations
}

func evaluateOutput(p ast.PolicyConfig, compiled *output.Output) []Violation {
	if compiled == nil {
		return nil
	}
	// Sort file paths for deterministic evaluation order.
	paths := make([]string, 0, len(compiled.Files))
	for fp := range compiled.Files {
		paths = append(paths, fp)
	}
	sort.Strings(paths)

	var violations []Violation
	for _, filePath := range paths {
		content := compiled.Files[filePath]
		for _, deny := range p.Deny {
			violations = append(violations, evaluateDenyOnFile(p.Name, filePath, content, deny)...)
		}
	}
	return violations
}

func evaluateSettings(p ast.PolicyConfig, compiled *output.Output) []Violation {
	if compiled == nil {
		return nil
	}
	paths := make([]string, 0, len(compiled.Files))
	for fp := range compiled.Files {
		if strings.HasSuffix(fp, "settings.json") || strings.HasSuffix(fp, "settings.local.json") {
			paths = append(paths, fp)
		}
	}
	sort.Strings(paths)

	var violations []Violation
	for _, filePath := range paths {
		content := compiled.Files[filePath]
		for _, deny := range p.Deny {
			violations = append(violations, evaluateDenyOnFile(p.Name, filePath, content, deny)...)
		}
	}
	return violations
}
