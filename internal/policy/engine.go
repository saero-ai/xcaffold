package policy

import (
	"embed"

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

	var violations []Violation
	for _, p := range merged {
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
	var violations []Violation
	for name, resource := range resources {
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
	var violations []Violation
	for filePath, content := range compiled.Files {
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
	for filePath, content := range compiled.Files {
		if filePath == "settings.json" || filePath == ".claude/settings.json" {
			for _, deny := range p.Deny {
				violations = append(violations, evaluateDenyOnFile(p.Name, filePath, content, deny)...)
			}
		}
	}
	return violations
}
