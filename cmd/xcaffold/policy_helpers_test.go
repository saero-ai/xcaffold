package main

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
)

func TestDeepCopyConfig_PreservesBody(t *testing.T) {
	// Create a config with various resources that have Body fields
	config := &ast.XcaffoldConfig{
		Kind:    "project",
		Version: "1.0",
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"reviewer": {
					Name:        "reviewer",
					Description: "Code review agent",
					Model:       "claude-sonnet",
					Tools:       []string{"Read", "Write"},
					Body:        "You are a code reviewer.\n\nProvide detailed feedback.",
				},
			},
			Skills: map[string]ast.SkillConfig{
				"tdd": {
					Name:         "tdd",
					Description:  "Test-driven development",
					AllowedTools: []string{"Bash", "Read"},
					Body:         "Follow Red-Green-Refactor.\n\nFirst write the test.",
				},
			},
			Rules: map[string]ast.RuleConfig{
				"security": {
					Name:        "security",
					Description: "Security best practices",
					Body:        "Never commit secrets.\n\nUse environment variables.",
				},
			},
		},
		Project: &ast.ProjectConfig{
			Name:        "test-project",
			Description: "A test project",
			Body:        "This is the project documentation.",
		},
	}

	// Deep copy the config
	copied := deepCopyConfig(config)

	// Verify all Body fields are preserved
	if copied.Project.Body != config.Project.Body {
		t.Errorf("ProjectConfig.Body not preserved: got %q, want %q", copied.Project.Body, config.Project.Body)
	}

	reviewer, exists := copied.Agents["reviewer"]
	if !exists {
		t.Fatal("reviewer agent not found in copied config")
	}
	if reviewer.Body != config.Agents["reviewer"].Body {
		t.Errorf("AgentConfig.Body not preserved: got %q, want %q", reviewer.Body, config.Agents["reviewer"].Body)
	}

	tddSkill, exists := copied.Skills["tdd"]
	if !exists {
		t.Fatal("tdd skill not found in copied config")
	}
	if tddSkill.Body != config.Skills["tdd"].Body {
		t.Errorf("SkillConfig.Body not preserved: got %q, want %q", tddSkill.Body, config.Skills["tdd"].Body)
	}

	securityRule, exists := copied.Rules["security"]
	if !exists {
		t.Fatal("security rule not found in copied config")
	}
	if securityRule.Body != config.Rules["security"].Body {
		t.Errorf("RuleConfig.Body not preserved: got %q, want %q", securityRule.Body, config.Rules["security"].Body)
	}
}

func TestDeepCopyConfig_PreservesAllFields(t *testing.T) {
	// Create a config with multiple field types to ensure nothing else is lost
	config := &ast.XcaffoldConfig{
		Kind:    "global",
		Version: "1.0",
		Extends: "base.xcf",
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"developer": {
					Name:        "developer",
					Description: "Software developer",
					Model:       "claude-opus",
					Tools:       []string{"Bash", "Read", "Write", "Edit"},
					Body:        "You are a developer.",
				},
			},
		},
		Settings: map[string]ast.SettingsConfig{
			"general": {
				Name: "general",
			},
		},
	}

	// Deep copy
	copied := deepCopyConfig(config)

	// Verify top-level fields
	if copied.Kind != config.Kind {
		t.Errorf("Kind not preserved: got %q, want %q", copied.Kind, config.Kind)
	}
	if copied.Version != config.Version {
		t.Errorf("Version not preserved: got %q, want %q", copied.Version, config.Version)
	}
	if copied.Extends != config.Extends {
		t.Errorf("Extends not preserved: got %q, want %q", copied.Extends, config.Extends)
	}

	// Verify agent fields
	dev, exists := copied.Agents["developer"]
	if !exists {
		t.Fatal("developer agent not found")
	}
	if dev.Name != "developer" {
		t.Errorf("Agent Name not preserved: got %q, want %q", dev.Name, "developer")
	}
	if dev.Description != "Software developer" {
		t.Errorf("Agent Description not preserved: got %q, want %q", dev.Description, "Software developer")
	}
	if dev.Model != "claude-opus" {
		t.Errorf("Agent Model not preserved: got %q, want %q", dev.Model, "claude-opus")
	}
	if len(dev.Tools) != 4 {
		t.Errorf("Agent Tools not preserved: got %d items, want 4", len(dev.Tools))
	}
	if dev.Body != "You are a developer." {
		t.Errorf("Agent Body not preserved: got %q, want %q", dev.Body, "You are a developer.")
	}

	// Verify settings
	_, settingsExist := copied.Settings["general"]
	if !settingsExist {
		t.Fatal("settings not preserved")
	}
}

func TestDeepCopyConfig_EmptyConfig(t *testing.T) {
	// Test that empty configs are handled correctly
	config := &ast.XcaffoldConfig{
		Kind:    "global",
		Version: "1.0",
	}

	copied := deepCopyConfig(config)

	if copied.Kind != "global" {
		t.Errorf("Kind not preserved in empty config: got %q, want %q", copied.Kind, "global")
	}
	if copied.Version != "1.0" {
		t.Errorf("Version not preserved in empty config: got %q, want %q", copied.Version, "1.0")
	}
	if len(copied.Agents) != 0 {
		t.Errorf("Empty agents map corrupted: got %d items, want 0", len(copied.Agents))
	}
}

func TestDeepCopyConfig_ComplexBody(t *testing.T) {
	// Test multi-line Body fields with special characters
	complexBody := `# Introduction

This is a complex body with:
- Markdown formatting
- Multiple lines
- Special characters: !@#$%^&*()
- Code blocks

` + "```go\nfunc example() {\n  return 42\n}\n```"

	config := &ast.XcaffoldConfig{
		Kind:    "project",
		Version: "1.0",
		Project: &ast.ProjectConfig{
			Name: "test",
			Body: complexBody,
		},
	}

	copied := deepCopyConfig(config)

	if copied.Project.Body != complexBody {
		t.Errorf("Complex Body not preserved.\nGot:\n%s\n\nWant:\n%s", copied.Project.Body, complexBody)
	}
}
