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
		SkillArtifactDirs: map[string]string{
			"references": "references",
			"scripts":    "scripts",
		},
		ModelField:      true,
		RuleActivations: []string{"always", "path-glob"},
	}
	if !caps.Agents {
		t.Error("expected Agents supported")
	}
	if caps.Hooks {
		t.Error("expected Hooks unsupported")
	}
	if len(caps.SkillArtifactDirs) != 2 {
		t.Errorf("expected 2 skill artifact dirs, got %d", len(caps.SkillArtifactDirs))
	}
}

func TestCapabilitySet_AgentTools(t *testing.T) {
	caps := CapabilitySet{
		AgentToolsField:      true,
		AgentNativeToolsOnly: false,
	}
	if !caps.AgentToolsField {
		t.Error("expected AgentToolsField supported")
	}
	if caps.AgentNativeToolsOnly {
		t.Error("expected AgentNativeToolsOnly unsupported")
	}
}

func TestCapabilitySet_SkillArtifactDirs(t *testing.T) {
	caps := CapabilitySet{
		SkillArtifactDirs: map[string]string{
			"references": "references",
			"scripts":    "scripts",
			"assets":     "assets",
			"examples":   "",
		},
	}
	if caps.SkillArtifactDirs["references"] != "references" {
		t.Error("expected references mapping")
	}
	if caps.SkillArtifactDirs["examples"] != "" {
		t.Error("expected examples to flatten (empty string)")
	}
}
