package renderer

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
)

// TestPresenceExtractor_Agent_PopulatedFields verifies that the generated
// ExtractAgentPresentFields returns the expected keys for a fully-populated
// AgentConfig.
func TestPresenceExtractor_Agent_PopulatedFields(t *testing.T) {
	tr := true
	agent := ast.AgentConfig{
		Name:                   "test-agent",
		Description:            "A test agent",
		Model:                  "sonnet",
		Effort:                 "high",
		MaxTurns:               10,
		Tools:                  ast.ClearableList{Values: []string{"Bash", "Read"}},
		DisallowedTools:        ast.ClearableList{Values: []string{"Write"}},
		Readonly:               &tr,
		PermissionMode:         "strict",
		DisableModelInvocation: &tr,
		UserInvocable:          &tr,
		Background:             &tr,
		Isolation:              "worktree",
		Memory:                 ast.FlexStringSlice{"mem-a"},
		InitialPrompt:          "You are a test agent.",
		Skills:                 ast.ClearableList{Values: []string{"tdd"}},
		MCPServers:             map[string]ast.MCPConfig{"server1": {Name: "server1"}},
		Hooks:                  ast.HookConfig{"PreToolUse": {}},
		Body:                   "Agent instructions body.",
	}

	got := ExtractAgentPresentFields(agent)

	expected := []string{
		"name", "description", "model", "effort", "max-turns",
		"tools", "disallowed-tools", "readonly", "permission-mode",
		"disable-model-invocation", "user-invocable", "background",
		"isolation", "memory", "initial-prompt", "skills",
		"mcp-servers", "hooks", "body",
	}
	for _, key := range expected {
		if _, ok := got[key]; !ok {
			t.Errorf("missing expected key %q in generated output", key)
		}
	}
}

// TestPresenceExtractor_Skill_PopulatedFields verifies the generated skill
// extractor returns expected keys for a fully-populated SkillConfig.
func TestPresenceExtractor_Skill_PopulatedFields(t *testing.T) {
	tr := true
	skill := ast.SkillConfig{
		Name:                   "test-skill",
		Description:            "A test skill",
		WhenToUse:              "When testing",
		License:                "MIT",
		AllowedTools:           ast.ClearableList{Values: []string{"Read", "Write"}},
		DisableModelInvocation: &tr,
		UserInvocable:          &tr,
		ArgumentHint:           "file path",
		Body:                   "Skill body.",
	}

	got := ExtractSkillPresentFields(skill)

	expected := []string{
		"name", "description", "when-to-use", "license",
		"allowed-tools", "disable-model-invocation",
		"user-invocable", "argument-hint", "body",
	}
	for _, key := range expected {
		if _, ok := got[key]; !ok {
			t.Errorf("missing expected key %q in generated output", key)
		}
	}
}

// TestPresenceExtractor_Rule_PopulatedFields verifies the generated rule
// extractor returns expected keys for a fully-populated RuleConfig.
func TestPresenceExtractor_Rule_PopulatedFields(t *testing.T) {
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

	got := ExtractRulePresentFields(rule)

	expected := []string{
		"name", "description", "always-apply", "activation",
		"paths", "exclude-agents", "body",
	}
	for _, key := range expected {
		if _, ok := got[key]; !ok {
			t.Errorf("missing expected key %q in generated output", key)
		}
	}
}
