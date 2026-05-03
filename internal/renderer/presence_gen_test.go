package renderer

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
)

// TestPresenceExtractor_Agent_MatchesManual creates a fully-populated AgentConfig,
// runs both the manual extractAgentPresentFields() and the generated
// ExtractAgentPresentFields(), and verifies the generated output is a superset
// of the manual output (all manual keys present with identical values).
func TestPresenceExtractor_Agent_MatchesManual(t *testing.T) {
	tr := true
	agent := ast.AgentConfig{
		Name:                   "test-agent",
		Description:            "A test agent",
		Model:                  "sonnet",
		Effort:                 "high",
		MaxTurns:               10,
		Tools:                  []string{"Bash", "Read"},
		DisallowedTools:        []string{"Write"},
		Readonly:               &tr,
		PermissionMode:         "strict",
		DisableModelInvocation: &tr,
		UserInvocable:          &tr,
		Background:             &tr,
		Isolation:              "worktree",
		Memory:                 ast.FlexStringSlice{"mem-a"},
		InitialPrompt:          "You are a test agent.",
		Skills:                 []string{"tdd"},
		MCPServers:             map[string]ast.MCPConfig{"server1": {Name: "server1"}},
		Hooks:                  ast.HookConfig{"PreToolUse": {}},
		Body:                   "Agent instructions body.",
	}

	manual := extractAgentPresentFields(agent)
	generated := ExtractAgentPresentFields(agent)

	// Every key in the manual output must exist in the generated output
	// with the same value.
	for key, mVal := range manual {
		gVal, ok := generated[key]
		if !ok {
			t.Errorf("generated output missing key %q (manual has value %q)", key, mVal)
			continue
		}
		if gVal != mVal {
			t.Errorf("key %q: manual=%q generated=%q", key, mVal, gVal)
		}
	}
}

// TestPresenceExtractor_Skill_MatchesManual verifies the generated skill extractor
// matches the manual implementation for all fields the manual one checks.
func TestPresenceExtractor_Skill_MatchesManual(t *testing.T) {
	tr := true
	skill := ast.SkillConfig{
		Name:                   "test-skill",
		Description:            "A test skill",
		WhenToUse:              "When testing",
		License:                "MIT",
		AllowedTools:           []string{"Read", "Write"},
		DisableModelInvocation: &tr,
		UserInvocable:          &tr,
		ArgumentHint:           "file path",
		Body:                   "Skill body.",
	}

	manual := extractSkillPresentFields(skill)
	generated := ExtractSkillPresentFields(skill)

	for key, mVal := range manual {
		gVal, ok := generated[key]
		if !ok {
			t.Errorf("generated output missing key %q (manual has value %q)", key, mVal)
			continue
		}
		if gVal != mVal {
			t.Errorf("key %q: manual=%q generated=%q", key, mVal, gVal)
		}
	}
}

// TestPresenceExtractor_Rule_MatchesManual verifies the generated rule extractor
// matches the manual implementation for all fields the manual one checks.
func TestPresenceExtractor_Rule_MatchesManual(t *testing.T) {
	tr := true
	rule := ast.RuleConfig{
		Name:          "test-rule",
		Description:   "A test rule",
		AlwaysApply:   &tr,
		Activation:    "always",
		Paths:         []string{"*.go"},
		ExcludeAgents: []string{"code-review"},
		Body:          "Rule body.",
	}

	manual := extractRulePresentFields(rule)
	generated := ExtractRulePresentFields(rule)

	for key, mVal := range manual {
		gVal, ok := generated[key]
		if !ok {
			t.Errorf("generated output missing key %q (manual has value %q)", key, mVal)
			continue
		}
		if gVal != mVal {
			t.Errorf("key %q: manual=%q generated=%q", key, mVal, gVal)
		}
	}
}
