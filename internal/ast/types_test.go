package ast

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestProjectConfig_InstructionsFields_RoundTrip(t *testing.T) {
	config := ProjectConfig{
		Name:         "payments-monorepo",
		Instructions: "Use pnpm. PostgreSQL 16.",
		InstructionsImports: []string{
			"xcf/instructions/style-guide.md",
		},
		InstructionsScopes: []InstructionsScope{
			{
				Path:             "packages/worker",
				InstructionsFile: "xcf/instructions/scopes/packages-worker.md",
				MergeStrategy:    "concat",
				SourceProvider:   "claude",
				SourceFilename:   "CLAUDE.md",
			},
		},
	}

	data, err := yaml.Marshal(config)
	require.NoError(t, err)
	content := string(data)
	require.Contains(t, content, "instructions:")
	require.Contains(t, content, "instructions-imports:")
	require.Contains(t, content, "instructions-scopes:")
	require.Contains(t, content, "merge-strategy: concat")
	require.Contains(t, content, "source-provider: claude")

	var out ProjectConfig
	require.NoError(t, yaml.Unmarshal(data, &out))
	require.Equal(t, "Use pnpm. PostgreSQL 16.", out.Instructions)
	require.Len(t, out.InstructionsImports, 1)
	require.Len(t, out.InstructionsScopes, 1)
	require.Equal(t, "packages/worker", out.InstructionsScopes[0].Path)
	require.Equal(t, "concat", out.InstructionsScopes[0].MergeStrategy)
}

func TestInstructionsScope_WithVariants_RoundTrip(t *testing.T) {
	scope := InstructionsScope{
		Path:          "packages/shared",
		MergeStrategy: "concat",
		Variants: map[string]InstructionsVariant{
			"claude": {
				InstructionsFile: "xcf/instructions/scopes/packages-shared.claude.md",
				SourceFilename:   "CLAUDE.md",
			},
			"cursor": {
				InstructionsFile: "xcf/instructions/scopes/packages-shared.cursor.md",
				SourceFilename:   "AGENTS.md",
			},
		},
		Reconciliation: &ReconciliationConfig{
			Strategy:       "per-target",
			LastReconciled: "2026-04-15T15:30:00Z",
			Notes:          "claude variant has security section absent from cursor variant",
		},
	}

	data, err := yaml.Marshal(scope)
	require.NoError(t, err)
	content := string(data)
	require.Contains(t, content, "variants:")
	require.Contains(t, content, "reconciliation:")
	require.Contains(t, content, "strategy: per-target")

	var out InstructionsScope
	require.NoError(t, yaml.Unmarshal(data, &out))
	require.Len(t, out.Variants, 2)
	require.NotNil(t, out.Reconciliation)
	require.Equal(t, "per-target", out.Reconciliation.Strategy)
}

func TestProjectConfig_MutualExclusivity_BothSet_IsNotParseable(t *testing.T) {
	// Mutual exclusivity is enforced at parser level, not yaml.Unmarshal level.
	// This test verifies the struct can hold both (parser rejects it separately).
	config := ProjectConfig{
		Name:             "test",
		Instructions:     "inline",
		InstructionsFile: "xcf/instructions/root.md",
	}
	_, err := yaml.Marshal(config)
	require.NoError(t, err) // struct allows it; parser rejects it
}
