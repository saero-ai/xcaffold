package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse_Memory_MinimalValid(t *testing.T) {
	xcf := `
kind: global
version: "1.0"
memory:
  user-role:
    name: user-role
    type: user
    instructions: "Robert is the founder."
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "scaffold.xcf")
	require.NoError(t, os.WriteFile(path, []byte(xcf), 0o600))

	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	config, err := ParseFile(path)
	require.NoError(t, err)

	m, ok := config.Memory["user-role"]
	require.True(t, ok, "user-role memory entry must be present")
	require.Equal(t, "user", m.Type)
	require.Equal(t, "Robert is the founder.", m.Instructions)
}

func TestParse_Memory_UnknownField_Fails(t *testing.T) {
	xcf := `
kind: global
version: "1.0"
memory:
  user-role:
    name: user-role
    type: user
    unknown_field: should-fail
    instructions: "test"
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "scaffold.xcf")
	require.NoError(t, os.WriteFile(path, []byte(xcf), 0o600))

	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	_, err := ParseFile(path)
	require.Error(t, err, "unknown field in memory block must produce a parse error")
	require.Contains(t, err.Error(), "unknown_field")
}

func TestParse_Memory_InstructionsFileReservedPrefix_Fails(t *testing.T) {
	cases := []struct {
		name string
		path string
	}{
		{"claude home", "~/.claude/projects/test/memory/arch.md"},
		{"gemini home", "~/.gemini/memory/arch.md"},
		{"agents dir", ".agents/memory/arch.md"},
		{"cursorrules file", ".cursorrules"},
		{"copilot instructions", ".github/copilot-instructions.md"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			xcf := fmt.Sprintf(`
kind: global
version: "1.0"
memory:
  arch:
    name: arch
    type: reference
    instructions-file: %s
`, tc.path)
			tmp := t.TempDir()
			path := filepath.Join(tmp, "scaffold.xcf")
			require.NoError(t, os.WriteFile(path, []byte(xcf), 0o600))

			t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
			_, err := ParseFile(path)
			require.Error(t, err, "path %q must be rejected", tc.path)
		})
	}
}

func TestParse_Memory_InstructionsMutuallyExclusive(t *testing.T) {
	xcf := `
kind: global
version: "1.0"
memory:
  user-role:
    name: user-role
    type: user
    instructions: "inline"
    instructions-file: xcf/memory/user-role.md
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "scaffold.xcf")
	require.NoError(t, os.WriteFile(path, []byte(xcf), 0o600))

	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	_, err := ParseFile(path)
	require.Error(t, err, "setting both instructions and instructions-file must be a parse error")
}

func TestParse_Fixture_MemoryEntries(t *testing.T) {
	path := "../../testing/fixtures/full.xcf"
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	config, err := ParseFile(path)
	require.NoError(t, err)

	require.NotEmpty(t, config.Memory, "full fixture must have at least one memory entry")

	var hasSeedOnce, hasTracked bool
	for _, m := range config.Memory {
		if m.Lifecycle == "tracked" {
			hasTracked = true
		}
		if m.Lifecycle == "" || m.Lifecycle == "seed-once" {
			hasSeedOnce = true
		}
	}
	require.True(t, hasSeedOnce, "fixture must have a seed-once (or default) memory entry")
	require.True(t, hasTracked, "fixture must have a tracked memory entry")
}
