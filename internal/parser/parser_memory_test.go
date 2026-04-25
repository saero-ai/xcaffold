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
	require.Equal(t, "Robert is the founder.", m.Instructions)
}

func TestParse_Memory_UnknownField_Fails(t *testing.T) {
	xcf := `
kind: global
version: "1.0"
memory:
  user-role:
    name: user-role
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
    instructions: "inline"
    instructions-file: xcf/agents/dev/memory/user-role.md
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "project.xcf")
	require.NoError(t, os.WriteFile(path, []byte(xcf), 0o600))

	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	_, err := ParseFile(path)
	require.Error(t, err, "setting both instructions and instructions-file must be a parse error")
}

// TestParse_Memory_LifecycleField_Rejected verifies that the removed lifecycle:
// field is now rejected as unknown by KnownFields(true).
func TestParse_Memory_LifecycleField_Rejected(t *testing.T) {
	xcf := `
kind: global
version: "1.0"
memory:
  user-role:
    name: user-role
    lifecycle: permanent
    instructions: "test"
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "project.xcf")
	require.NoError(t, os.WriteFile(path, []byte(xcf), 0o600))

	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	_, err := ParseFile(path)
	require.Error(t, err, "lifecycle field must be rejected as unknown after removal")
}

// TestParse_Memory_TypeField_Rejected verifies that the removed type: field is
// now rejected as unknown by KnownFields(true).
func TestParse_Memory_TypeField_Rejected(t *testing.T) {
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
	require.Error(t, err, "type field must be rejected as unknown after removal")
}

// TestParse_Memory_MinimalFields_Passes verifies that a memory entry with only
// name and instructions (no type or lifecycle) parses successfully.
func TestParse_Memory_MinimalFields_Passes(t *testing.T) {
	cases := []struct {
		name string
		xcf  string
	}{
		{"name-and-instructions", `
kind: global
version: "1.0"
memory:
  entry:
    name: entry
    instructions: "test"
`},
		{"name-description-instructions", `
kind: global
version: "1.0"
memory:
  entry:
    name: entry
    description: "A memory entry."
    instructions: "test"
`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			path := filepath.Join(tmp, "project.xcf")
			require.NoError(t, os.WriteFile(path, []byte(tc.xcf), 0o600))

			t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
			_, err := ParseFile(path)
			require.NoError(t, err, "minimal memory entry must parse without error")
		})
	}
}

// TestParse_Memory_StandaloneKindFile verifies that a kind: memory .xcf
// file located under xcf/agents/<agentID>/memory/ is (a) parsed into config.Memory
// and (b) annotated with AgentRef derived from the parent directory name.
func TestParse_Memory_StandaloneKindFile(t *testing.T) {
	dir := t.TempDir()
	memDir := filepath.Join(dir, "xcf", "agents", "auth-specialist", "memory")
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
	require.Equal(t, "auth-specialist", entry.AgentRef, "AgentRef must be derived from xcf/agents/<agentID>/memory/ directory")
	require.NotEmpty(t, entry.Instructions)
}

func TestParse_Fixture_MemoryEntries(t *testing.T) {
	path := "../../testing/fixtures/full.xcf"
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	config, err := ParseFile(path)
	require.NoError(t, err)

	require.NotEmpty(t, config.Memory, "full fixture must have at least one memory entry")

	// Verify at least one entry has instructions and at least one uses instructions-file.
	var hasInline, hasFile bool
	for _, m := range config.Memory {
		if m.Instructions != "" {
			hasInline = true
		}
		if m.InstructionsFile != "" {
			hasFile = true
		}
	}
	require.True(t, hasInline, "fixture must have at least one inline-instructions memory entry")
	require.True(t, hasFile, "fixture must have at least one instructions-file memory entry")
}

// TestParse_Memory_FrontmatterBody verifies that the markdown body of a
// frontmatter-format kind: memory file is assigned to Instructions when the
// YAML instructions: field is not set.
func TestParse_Memory_FrontmatterBody(t *testing.T) {
	xcf := `---
kind: memory
version: "1.0"
name: body-test
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

// TestParse_Memory_AgentScopedDirectory verifies that a kind:memory file
// located at xcf/agents/<agentID>/memory/<name>.xcf has AgentRef set to
// <agentID> (the segment BEFORE "memory" in the path, not after).
func TestParse_Memory_AgentScopedDirectory(t *testing.T) {
	dir := t.TempDir()
	agentMemDir := filepath.Join(dir, "xcf", "agents", "auth-specialist", "memory")
	require.NoError(t, os.MkdirAll(agentMemDir, 0o755))
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
name: audit_log_owner
description: "who owns the audit log"
instructions: "Audit log is owned by the security team."
---
`)
	require.NoError(t, os.WriteFile(filepath.Join(agentMemDir, "audit_log_owner.xcf"), body, 0o644))

	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	cfg, err := ParseDirectory(dir)
	require.NoError(t, err)

	entry, ok := cfg.Memory["audit_log_owner"]
	require.True(t, ok, "expected config.Memory[audit_log_owner] to be present")
	require.Equal(t, "auth-specialist", entry.AgentRef,
		"AgentRef must be derived from xcf/agents/<agentID>/memory/ — the segment BEFORE memory")
	require.NotEmpty(t, entry.Instructions)
}

func TestParse_Memory_OldFieldsRejected_AfterRemoval(t *testing.T) {
	cases := []struct {
		name  string
		field string
		value string
	}{
		{"type field", "type", "user"},
		{"lifecycle field", "lifecycle", "seed-once"},
		{"targets field", "targets", "claude: {}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			xcf := fmt.Sprintf(`
kind: global
version: "1.0"
memory:
  entry:
    name: entry
    %s: %s
    instructions: "test"
`, tc.field, tc.value)
			tmp := t.TempDir()
			path := filepath.Join(tmp, "project.xcf")
			require.NoError(t, os.WriteFile(path, []byte(xcf), 0o600))
			t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
			_, err := ParseFile(path)
			require.Error(t, err, "field %q must be rejected as unknown", tc.field)
		})
	}
}

func TestParse_Agent_FlatFileRejected(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, "xcf", "agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte(`---
kind: project
version: "1.0"
name: demo
targets:
  - claude
---
`), 0o644))
	// Flat agent file: xcf/agents/dev.xcf — must be rejected
	body := []byte(`---
kind: agent
version: "1.0"
name: dev
description: "Developer agent."
---
`)
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "dev.xcf"), body, 0o644))

	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	_, err := ParseDirectory(dir)
	require.Error(t, err, "flat kind:agent file under xcf/agents/ must be rejected")
	require.Contains(t, err.Error(), "subdirectory")
}

func TestParse_Agent_SubdirFileAccepted(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, "xcf", "agents", "dev")
	require.NoError(t, os.MkdirAll(agentDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte(`---
kind: project
version: "1.0"
name: demo
targets:
  - claude
---
`), 0o644))
	body := []byte(`---
kind: agent
version: "1.0"
name: dev
description: "Developer agent."
instructions: "You are a developer."
---
`)
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "dev.xcf"), body, 0o644))

	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	cfg, err := ParseDirectory(dir)
	require.NoError(t, err, "kind:agent in xcf/agents/<id>/<id>.xcf must be accepted")
	_, ok := cfg.Agents["dev"]
	require.True(t, ok, "agent must be present in config.Agents")
}
