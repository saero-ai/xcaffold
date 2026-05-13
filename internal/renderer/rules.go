package renderer

import (
	"fmt"

	"github.com/saero-ai/xcaffold/internal/ast"
)

// BuildRuleDescriptionFrontmatter returns the YAML line (e.g. "description: ...\n")
// if the target encodes description in frontmatter, or empty string otherwise.
func BuildRuleDescriptionFrontmatter(rule ast.RuleConfig, caps CapabilitySet) string {
	if rule.Description == "" || caps.RuleEncoding.Description != "frontmatter" {
		return ""
	}
	return fmt.Sprintf("description: %s\n", YAMLScalar(rule.Description))
}

// BuildRuleProsePrefix returns the description as plain prose, followed by two newlines,
// if the target encodes description as prose. Otherwise returns an empty string.
func BuildRuleProsePrefix(rule ast.RuleConfig, caps CapabilitySet) string {
	if rule.Description == "" || caps.RuleEncoding.Description != "prose" {
		return ""
	}
	return fmt.Sprintf("%s\n\n", rule.Description)
}

// ValidateRuleActivation checks if the rule's activation is natively supported by the target.
func ValidateRuleActivation(rule ast.RuleConfig, caps CapabilitySet) bool {
	activation := ResolvedActivation(rule)
	for _, supported := range caps.RuleActivations {
		if activation == supported {
			return true
		}
	}
	return false
}
