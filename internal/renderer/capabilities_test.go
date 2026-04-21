package renderer

import "testing"

func TestCapabilitySet_Fields(t *testing.T) {
	caps := CapabilitySet{
		Agents:              true,
		Skills:              true,
		Rules:               true,
		Workflows:           false,
		Hooks:               false,
		Settings:            true,
		MCP:                 false,
		Memory:              true,
		ProjectInstructions: true,
		SkillSubdirs:        []string{"references", "scripts"},
		ModelField:          true,
		RuleActivations:     []string{"always", "path-glob"},
	}
	if !caps.Agents {
		t.Error("expected Agents supported")
	}
	if caps.Hooks {
		t.Error("expected Hooks unsupported")
	}
	if len(caps.SkillSubdirs) != 2 {
		t.Errorf("expected 2 skill subdirs, got %d", len(caps.SkillSubdirs))
	}
}
