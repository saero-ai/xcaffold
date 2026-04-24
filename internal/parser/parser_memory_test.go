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
	path := filepath.Join(tmp, "project.xcf")
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
	path := filepath.Join(tmp, "project.xcf")
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
			path := filepath.Join(tmp, "project.xcf")
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
	path := filepath.Join(tmp, "project.xcf")
	require.NoError(t, os.WriteFile(path, []byte(xcf), 0o600))

	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	_, err := ParseFile(path)
	require.Error(t, err, "setting both instructions and instructions-file must be a parse error")
}

func TestParse_Memory_InvalidLifecycle_Fails(t *testing.T) {
	xcf := `
kind: global
version: "1.0"
memory:
  user-role:
    name: user-role
    type: user
    lifecycle: permanent
    instructions: "test"
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "project.xcf")
	require.NoError(t, os.WriteFile(path, []byte(xcf), 0o600))

	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	_, err := ParseFile(path)
	require.Error(t, err, "unknown lifecycle value must be rejected at parse time")
	require.Contains(t, err.Error(), "lifecycle")
	require.Contains(t, err.Error(), "permanent")
}

func TestParse_Memory_InvalidType_Fails(t *testing.T) {
	xcf := `
kind: global
version: "1.0"
memory:
  user-role:
    name: user-role
    type: secret
    instructions: "test"
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "project.xcf")
	require.NoError(t, os.WriteFile(path, []byte(xcf), 0o600))

	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	_, err := ParseFile(path)
	require.Error(t, err, "unknown type value must be rejected at parse time")
	require.Contains(t, err.Error(), "type")
	require.Contains(t, err.Error(), "secret")
}

func TestParse_Memory_ValidLifecycleAndType_Passes(t *testing.T) {
	cases := []struct {
		lifecycle string
		memType   string
	}{
		{"seed-once", "user"},
		{"tracked", "feedback"},
		{"", "project"},
		{"seed-once", "reference"},
	}
	for _, tc := range cases {
		t.Run(tc.lifecycle+"/"+tc.memType, func(t *testing.T) {
			lifecycleLine := ""
			if tc.lifecycle != "" {
				lifecycleLine = "    lifecycle: " + tc.lifecycle + "\n"
			}
			xcf := fmt.Sprintf(`
kind: global
version: "1.0"
memory:
  entry:
    name: entry
    type: %s
%s    instructions: "test"
`, tc.memType, lifecycleLine)
			tmp := t.TempDir()
			path := filepath.Join(tmp, "project.xcf")
			require.NoError(t, os.WriteFile(path, []byte(xcf), 0o600))

			t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
			_, err := ParseFile(path)
			require.NoError(t, err, "valid lifecycle=%q type=%q must not fail", tc.lifecycle, tc.memType)
		})
	}
}

// TestParse_Memory_StandaloneKindFile verifies that a kind: memory .xcf
// file located under xcf/memory/<agentID>/ is (a) parsed into config.Memory
// and (b) annotated with AgentRef derived from the parent directory name.
func TestParse_Memory_StandaloneKindFile(t *testing.T) {
	dir := t.TempDir()
	memDir := filepath.Join(dir, "xcf", "memory", "auth-specialist")
	require.NoError(t, os.MkdirAll(memDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte(`---
kind: project
version: "1.0"
name: demo
targets:
  - claude
---
`), 0o644))
	body := []byte(`---
kind: memory
version: "1.0"
name: project_audit_log_owner
description: "who owns the audit log"
instructions: "Audit log is owned by the security team."
---
`)
	require.NoError(t, os.WriteFile(filepath.Join(memDir, "project_audit_log_owner.xcf"), body, 0o644))

	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	cfg, err := ParseDirectory(dir)
	require.NoError(t, err)

	entry, ok := cfg.Memory["project_audit_log_owner"]
	require.True(t, ok, "expected config.Memory[project_audit_log_owner] to be present")
	require.Equal(t, "auth-specialist", entry.AgentRef, "AgentRef must be derived from xcf/memory/<agentID>/ directory")
	require.NotEmpty(t, entry.Instructions)
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

// TestParse_Memory_FrontmatterBody verifies that the markdown body of a
// frontmatter-format kind: memory file is assigned to Instructions when the
// YAML instructions: field is not set.
func TestParse_Memory_FrontmatterBody(t *testing.T) {
	xcf := `---
kind: memory
version: "1.0"
name: body-test
type: project
---
This is the memory body content.
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "body-test.xcf")
	require.NoError(t, os.WriteFile(path, []byte(xcf), 0o600))

	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	config, err := ParseFile(path)
	require.NoError(t, err)

	m, ok := config.Memory["body-test"]
	require.True(t, ok, "body-test memory entry must be present")
	require.Equal(t, "This is the memory body content.", m.Instructions,
		"markdown body must be assigned to Instructions when yaml instructions: is empty")
}

// TestParse_Memory_FrontmatterBody_YAMLWins verifies that when both the YAML
// instructions: field and a markdown body are present, the YAML field wins and
// the body is discarded.
func TestParse_Memory_FrontmatterBody_YAMLWins(t *testing.T) {
	xcf := `---
kind: memory
version: "1.0"
name: yaml-wins
type: project
instructions: "YAML wins"
---
This body should be discarded.
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "yaml-wins.xcf")
	require.NoError(t, os.WriteFile(path, []byte(xcf), 0o600))

	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	config, err := ParseFile(path)
	require.NoError(t, err)

	m, ok := config.Memory["yaml-wins"]
	require.True(t, ok, "yaml-wins memory entry must be present")
	require.Equal(t, "YAML wins", m.Instructions,
		"yaml instructions: field must win over markdown body")
}
