package policy

import (
	"embed"
	"sort"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
	"gopkg.in/yaml.v3"
)

//go:embed builtin/*.xcf
var builtinFS embed.FS

// Evaluate runs all policies against the config snapshot and compiled output.
// User policies override built-in policies with the same name.
func Evaluate(
	userPolicies map[string]ast.PolicyConfig,
	configSnapshot *ast.XcaffoldConfig,
	compiled *output.Output,
) []Violation {
	merged := mergeBuiltins(userPolicies)

	// Sort policy names for deterministic evaluation order.
	policyNames := make([]string, 0, len(merged))
	for name := range merged {
		policyNames = append(policyNames, name)
	}
	sort.Strings(policyNames)

	var violations []Violation
	for _, name := range policyNames {
		p := merged[name]
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

func mergeBuiltins(userPolicies map[string]ast.PolicyConfig) map[string]ast.PolicyConfig {
	merged := make(map[string]ast.PolicyConfig)

	entries, _ := builtinFS.ReadDir("builtin")
	for _, entry := range entries {
		data, err := builtinFS.ReadFile("builtin/" + entry.Name())
		if err != nil {
			continue
		}
		var p ast.PolicyConfig
		if err := yaml.Unmarshal(data, &p); err != nil {
			continue
		}
		merged[p.Name] = p
	}

	for name, p := range userPolicies {
		merged[name] = p
	}

	return merged
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
	var violations []Violation
	// Settings target only checks known settings file paths.
	for _, filePath := range []string{"settings.json", ".claude/settings.json"} {
		content, ok := compiled.Files[filePath]
		if !ok {
			continue
		}
		for _, deny := range p.Deny {
			violations = append(violations, evaluateDenyOnFile(p.Name, filePath, content, deny)...)
		}
	}
	return violations
}
