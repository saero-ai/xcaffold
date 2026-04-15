package ast

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestMemoryConfig_Parse_MinimalValid(t *testing.T) {
	m := MemoryConfig{
		Name:         "user-role",
		Type:         "user",
		Instructions: "Robert is the founder.",
	}

	data, err := yaml.Marshal(m)
	require.NoError(t, err)
	content := string(data)

	require.Contains(t, content, "name: user-role")
	require.Contains(t, content, "type: user")
	require.Contains(t, content, "instructions:")
}

func TestMemoryConfig_Parse_FullFields(t *testing.T) {
	m := MemoryConfig{
		Name:             "arch-decisions",
		Type:             "reference",
		Description:      "Key architectural decisions.",
		Lifecycle:        "tracked",
		InstructionsFile: "xcf/memory/arch-decisions.md",
	}

	data, err := yaml.Marshal(m)
	require.NoError(t, err)
	content := string(data)

	require.Contains(t, content, "lifecycle: tracked")
	require.Contains(t, content, "instructions-file: xcf/memory/arch-decisions.md")
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
