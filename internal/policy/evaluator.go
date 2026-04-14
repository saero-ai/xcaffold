package policy

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
)

// fieldAccessor abstracts field lookups over agent, skill, and rule configs.
type fieldAccessor interface {
	fieldValue(name string) string
	fieldCount(name string) int
	tools() []string
	targets() []string
}

// --- agentAccessor ---

type agentAccessor struct {
	cfg ast.AgentConfig
}

func newAgentAccessor(cfg ast.AgentConfig) fieldAccessor {
	return agentAccessor{cfg: cfg}
}

func (a agentAccessor) fieldValue(name string) string {
	switch name {
	case "name":
		return a.cfg.Name
	case "description":
		return a.cfg.Description
	case "model":
		return a.cfg.Model
	case "instructions":
		return a.cfg.Instructions
	default:
		return ""
	}
}

func (a agentAccessor) fieldCount(name string) int {
	switch name {
	case "tools":
		return len(a.cfg.Tools)
	case "skills":
		return len(a.cfg.Skills)
	case "rules":
		return len(a.cfg.Rules)
	default:
		return 0
	}
}

func (a agentAccessor) tools() []string   { return a.cfg.Tools }
func (a agentAccessor) targets() []string { return nil }

// --- skillAccessor ---

type skillAccessor struct {
	cfg ast.SkillConfig
}

func newSkillAccessor(cfg ast.SkillConfig) fieldAccessor {
	return skillAccessor{cfg: cfg}
}

func (s skillAccessor) fieldValue(name string) string {
	switch name {
	case "name":
		return s.cfg.Name
	case "description":
		return s.cfg.Description
	case "instructions":
		return s.cfg.Instructions
	default:
		return ""
	}
}

func (s skillAccessor) fieldCount(name string) int {
	switch name {
	case "tools":
		return len(s.cfg.Tools)
	default:
		return 0
	}
}

func (s skillAccessor) tools() []string   { return s.cfg.Tools }
func (s skillAccessor) targets() []string { return nil }

// --- ruleAccessor ---

type ruleAccessor struct {
	cfg ast.RuleConfig
}

func newRuleAccessor(cfg ast.RuleConfig) fieldAccessor {
	return ruleAccessor{cfg: cfg}
}

func (r ruleAccessor) fieldValue(name string) string {
	switch name {
	case "name":
		return r.cfg.Name
	case "description":
		return r.cfg.Description
	case "instructions":
		return r.cfg.Instructions
	default:
		return ""
	}
}

func (r ruleAccessor) fieldCount(_ string) int { return 0 }
func (r ruleAccessor) tools() []string         { return nil }
func (r ruleAccessor) targets() []string       { return nil }

// --- evaluateRequire ---

// evaluateRequire checks a single PolicyRequire constraint against a resource.
// Returns a *Violation if the constraint is violated, or nil if it passes.
func evaluateRequire(policyName, resourceName string, req ast.PolicyRequire, acc fieldAccessor) *Violation {
	if req.IsPresent != nil && *req.IsPresent {
		val := acc.fieldValue(req.Field)
		if val == "" {
			return &Violation{
				PolicyName:   policyName,
				ResourceName: resourceName,
				Message:      fmt.Sprintf("field %q must be present and non-empty", req.Field),
			}
		}
	}

	if req.MinLength != nil {
		val := acc.fieldValue(req.Field)
		if len(val) < *req.MinLength {
			return &Violation{
				PolicyName:   policyName,
				ResourceName: resourceName,
				Message: fmt.Sprintf("field %q length %d is below minimum %d",
					req.Field, len(val), *req.MinLength),
			}
		}
	}

	if req.MaxCount != nil {
		count := acc.fieldCount(req.Field)
		if count > *req.MaxCount {
			return &Violation{
				PolicyName:   policyName,
				ResourceName: resourceName,
				Message: fmt.Sprintf("field %q has %d items, maximum is %d",
					req.Field, count, *req.MaxCount),
			}
		}
	}

	if len(req.OneOf) > 0 {
		val := acc.fieldValue(req.Field)
		if !containsString(req.OneOf, val) {
			return &Violation{
				PolicyName:   policyName,
				ResourceName: resourceName,
				Message: fmt.Sprintf("field %q value %q is not in approved list %v",
					req.Field, val, req.OneOf),
			}
		}
	}

	return nil
}

// --- evaluateDenyOnFile ---

// evaluateDenyOnFile checks a PolicyDeny against compiled file content and path.
// Returns all violations found (one per triggered check).
func evaluateDenyOnFile(policyName, filePath, content string, deny ast.PolicyDeny) []Violation {
	var violations []Violation

	for _, forbidden := range deny.ContentContains {
		if strings.Contains(strings.ToLower(content), strings.ToLower(forbidden)) {
			violations = append(violations, Violation{
				PolicyName: policyName,
				FilePath:   filePath,
				Message:    fmt.Sprintf("forbidden string %q found in file content", forbidden),
			})
		}
	}

	if deny.ContentMatches != "" {
		re, err := regexp.Compile(deny.ContentMatches)
		if err != nil {
			violations = append(violations, Violation{
				PolicyName: policyName,
				FilePath:   filePath,
				Message:    fmt.Sprintf("invalid regex in content_matches: %s", err),
			})
		} else if re.MatchString(content) {
			violations = append(violations, Violation{
				PolicyName: policyName,
				FilePath:   filePath,
				Message:    fmt.Sprintf("forbidden pattern %q matched in file content", deny.ContentMatches),
			})
		}
	}

	if deny.PathContains != "" {
		if strings.Contains(filePath, deny.PathContains) {
			violations = append(violations, Violation{
				PolicyName: policyName,
				FilePath:   filePath,
				Message:    fmt.Sprintf("forbidden string %q found in file path", deny.PathContains),
			})
		}
	}

	return violations
}

// containsString reports whether slice contains s.
func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
