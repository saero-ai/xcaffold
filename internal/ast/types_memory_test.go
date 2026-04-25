package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestMemoryConfig_Parse_MinimalValid(t *testing.T) {
	input := `
name: user-role
instructions: "Robert is the founder."
`
	var m MemoryConfig
	require.NoError(t, yaml.Unmarshal([]byte(input), &m))
	assert.Equal(t, "user-role", m.Name)
	assert.Equal(t, "Robert is the founder.", m.Instructions)
}

func TestMemoryConfig_Parse_FullFields(t *testing.T) {
	input := `
name: arch-decisions
description: "Key architectural decisions."
instructions-file: xcf/agents/reviewer/memory/arch-decisions.md
`
	var m MemoryConfig
	require.NoError(t, yaml.Unmarshal([]byte(input), &m))
	assert.Equal(t, "arch-decisions", m.Name)
	assert.Equal(t, "Key architectural decisions.", m.Description)
	assert.Equal(t, "xcf/agents/reviewer/memory/arch-decisions.md", m.InstructionsFile)
}

func TestMemoryConfig_InheritedNotSerialized(t *testing.T) {
	m := MemoryConfig{
		Name:      "test",
		Inherited: true,
	}
	data, err := yaml.Marshal(m)
	require.NoError(t, err)
	require.NotContains(t, string(data), "inherited")
}

func TestMemoryConfig_AgentRef_NotSerialized(t *testing.T) {
	m := MemoryConfig{Name: "project_audit_log_owner", AgentRef: "auth-specialist"}
	if m.AgentRef != "auth-specialist" {
		t.Fatalf("expected AgentRef=auth-specialist, got %q", m.AgentRef)
	}
	data, err := yaml.Marshal(m)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "agent-ref")
	assert.NotContains(t, string(data), "agentref")
}

func TestStripInherited_Memory(t *testing.T) {
	cfg := &XcaffoldConfig{}
	cfg.Memory = map[string]MemoryConfig{
		"inherited-mem": {Name: "inherited-mem", Inherited: true},
		"local-mem":     {Name: "local-mem", Inherited: false},
	}
	cfg.StripInherited()
	assert.NotContains(t, cfg.Memory, "inherited-mem")
	assert.Contains(t, cfg.Memory, "local-mem")
	assert.Equal(t, "local-mem", cfg.Memory["local-mem"].Name)
}
