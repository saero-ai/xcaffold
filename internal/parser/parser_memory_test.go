package parser

import (
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
	xcf := `
kind: global
version: "1.0"
memory:
  arch:
    name: arch
    type: reference
    instructions-file: .claude/memory/arch.md
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "scaffold.xcf")
	require.NoError(t, os.WriteFile(path, []byte(xcf), 0o600))

	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	_, err := ParseFile(path)
	require.Error(t, err, "instructions-file pointing at provider output directory must be rejected")
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
