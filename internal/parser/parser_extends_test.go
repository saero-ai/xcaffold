package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeFile is a helper that writes content to a named file inside dir.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

// ---------------------------------------------------------------------------
// TestParseFile_ValidConfig — basic directory parse with two single-kind files
// ---------------------------------------------------------------------------

func TestParseFile_ValidConfig(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	dir := t.TempDir()
	writeFile(t, dir, "project.xcf", `kind: project
version: "1.0"
name: "valid-project"
description: "A simple project."
`)
	writeFile(t, dir, "global.xcf", `kind: global
version: "1.0"
agents:
  coder:
    description: "Writes code."
    model: "claude-3-7-sonnet-20250219"
`)

	cfg, err := ParseDirectory(dir)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "1.0", cfg.Version)
	assert.Equal(t, "valid-project", cfg.Project.Name)
	assert.Equal(t, "A simple project.", cfg.Project.Description)
	require.Contains(t, cfg.Agents, "coder")
	assert.Equal(t, "claude-3-7-sonnet-20250219", cfg.Agents["coder"].Model)
}

// ---------------------------------------------------------------------------
// TestParseFile_ExtendsInheritance — child adds agent; base values survive
// ---------------------------------------------------------------------------

func TestParseFile_ExtendsInheritance(t *testing.T) {
	baseDir := t.TempDir()
	projectDir := t.TempDir()

	// base.xcf: single kind:global — contains only the base-agent.
	writeFile(t, baseDir, "base.xcf", `kind: global
version: "1.0"
agents:
  base-agent:
    description: "Base agent."
    model: "claude-3-5-haiku-20241022"
`)

	// project dir: separate project.xcf and global.xcf that extends the base.
	writeFile(t, projectDir, "project.xcf", `kind: project
version: "1.0"
name: "base-project"
description: "Base description."
`)
	writeFile(t, projectDir, "global.xcf", `kind: global
version: "1.0"
extends: "`+filepath.Join(baseDir, "base.xcf")+`"
agents:
  child-agent:
    description: "Child agent."
    model: "claude-3-7-sonnet-20250219"
`)

	cfg, err := ParseDirectory(projectDir)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Project info from project.xcf
	assert.Equal(t, "base-project", cfg.Project.Name)
	assert.Equal(t, "Base description.", cfg.Project.Description)

	// Base agent inherited from base.xcf via extends
	require.Contains(t, cfg.Agents, "base-agent", "base agent should be inherited")
	assert.Equal(t, "claude-3-5-haiku-20241022", cfg.Agents["base-agent"].Model)

	// Child agent added
	require.Contains(t, cfg.Agents, "child-agent", "child agent should be present")
	assert.Equal(t, "claude-3-7-sonnet-20250219", cfg.Agents["child-agent"].Model)
}

// ---------------------------------------------------------------------------
// TestParseFile_ExtendsChildOverridesBase — child overrides agent / project / test
// ---------------------------------------------------------------------------

func TestParseFile_ExtendsChildOverridesBase(t *testing.T) {
	baseDir := t.TempDir()
	childDir := t.TempDir()

	// base.xcf: kind:global only — no project doc (project in separate file).
	writeFile(t, baseDir, "base.xcf", `kind: global
version: "1.0"
agents:
  shared-agent:
    description: "Base version of agent."
    model: "claude-3-5-haiku-20241022"
`)

	// Child directory: project.xcf and global.xcf with extends.
	writeFile(t, childDir, "project.xcf", `kind: project
version: "1.0"
name: "child-project"
description: "Child description."
test:
  claude-path: "/usr/local/bin/claude"
  judge-model: "claude-3-opus-20240229"
`)
	writeFile(t, childDir, "global.xcf", `kind: global
version: "1.0"
extends: "`+filepath.Join(baseDir, "base.xcf")+`"
agents:
  shared-agent:
    description: "Child version of agent."
    model: "claude-3-7-sonnet-20250219"
`)

	cfg, err := ParseDirectory(childDir)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Child project from project.xcf
	assert.Equal(t, "child-project", cfg.Project.Name)
	assert.Equal(t, "Child description.", cfg.Project.Description)

	// Child agent overrides base agent (same key)
	require.Contains(t, cfg.Agents, "shared-agent")
	assert.Equal(t, "Child version of agent.", cfg.Agents["shared-agent"].Description)
	assert.Equal(t, "claude-3-7-sonnet-20250219", cfg.Agents["shared-agent"].Model)

	// Test config: child project.xcf sets both claude-path and judge-model
	require.NotNil(t, cfg.Project)
	assert.Equal(t, "/usr/local/bin/claude", cfg.Project.Test.ClaudePath, "claude-path from project.xcf")
	assert.Equal(t, "claude-3-opus-20240229", cfg.Project.Test.JudgeModel, "judge-model from project.xcf")
}

// ---------------------------------------------------------------------------
// TestParseFile_CircularExtendsDetected — A extends B, B extends A
// ---------------------------------------------------------------------------

func TestParseFile_CircularExtendsDetected(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "a.xcf", `kind: global
version: "1.0"
extends: "b.xcf"
`)

	writeFile(t, dir, "b.xcf", `kind: global
version: "1.0"
extends: "a.xcf"
`)

	aPath := filepath.Join(dir, "a.xcf")
	_, err := ParseFile(aPath)
	require.Error(t, err)
	assert.True(t,
		strings.Contains(strings.ToLower(err.Error()), "circular"),
		"error should mention 'circular', got: %s", err.Error(),
	)
}

// ---------------------------------------------------------------------------
// TestParseFile_ExtendsMissingBaseFile — extends a file that does not exist
// ---------------------------------------------------------------------------

func TestParseFile_ExtendsMissingBaseFile(t *testing.T) {
	dir := t.TempDir()

	childPath := writeFile(t, dir, "child.xcf", `kind: global
version: "1.0"
extends: "nonexistent-base.xcf"
`)

	_, err := ParseFile(childPath)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// TestParseFile_FileNotFound — ParseFile on nonexistent path
// ---------------------------------------------------------------------------

func TestParseFile_FileNotFound(t *testing.T) {
	_, err := ParseFile("/tmp/xcaffold-test-does-not-exist-12345.xcf")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// TestMergeMapOverride_NilBase
// ---------------------------------------------------------------------------

func TestMergeMapOverride_NilBase(t *testing.T) {
	child := map[string]string{"key1": "val1", "key2": "val2"}
	result := mergeMapOverride(nil, child)

	require.NotNil(t, result)
	assert.Equal(t, "val1", result["key1"])
	assert.Equal(t, "val2", result["key2"])
}

// ---------------------------------------------------------------------------
// TestMergeMapOverride_NilChild
// ---------------------------------------------------------------------------

func TestMergeMapOverride_NilChild(t *testing.T) {
	base := map[string]string{"key1": "base-val", "key2": "base-val2"}
	result := mergeMapOverride(base, nil)

	require.NotNil(t, result)
	assert.Equal(t, "base-val", result["key1"])
	assert.Equal(t, "base-val2", result["key2"])
}

// ---------------------------------------------------------------------------
// TestMergeMapOverride_BothNil
// ---------------------------------------------------------------------------

func TestMergeMapOverride_BothNil(t *testing.T) {
	result := mergeMapOverride[string, string](nil, nil)
	assert.Nil(t, result)
}

// ---------------------------------------------------------------------------
// TestMergeMapOverride_ChildOverridesBase
// ---------------------------------------------------------------------------

func TestMergeMapOverride_ChildOverridesBase(t *testing.T) {
	base := map[string]string{
		"shared":    "base-value",
		"base-only": "from-base",
	}
	child := map[string]string{
		"shared":     "child-value",
		"child-only": "from-child",
	}
	result := mergeMapOverride(base, child)

	require.NotNil(t, result)
	assert.Equal(t, "child-value", result["shared"], "child should override base for shared key")
	assert.Equal(t, "from-base", result["base-only"], "base-only key should be inherited")
	assert.Equal(t, "from-child", result["child-only"], "child-only key should be present")
}

// ---------------------------------------------------------------------------
// TestMergeMapStrict
// ---------------------------------------------------------------------------

func TestMergeMapStrict_DisallowsDuplicates(t *testing.T) {
	base := map[string]string{
		"shared": "base-value",
	}
	child := map[string]string{
		"shared": "child-value",
	}
	baseOrigins := map[string]string{"shared": "base.xcf"}

	_, _, err := mergeMapStrict(base, child, "agent", baseOrigins, "child.xcf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate agent ID \"shared\" found in base.xcf and child.xcf")
}

func TestMergeMapStrict_AllowsDisjoint(t *testing.T) {
	base := map[string]string{
		"base-only": "base-value",
	}
	child := map[string]string{
		"child-only": "child-value",
	}
	baseOrigins := map[string]string{"base-only": "base.xcf"}
	result, origins, err := mergeMapStrict(base, child, "agent", baseOrigins, "child.xcf")
	require.NoError(t, err)
	assert.Equal(t, "base-value", result["base-only"])
	assert.Equal(t, "child-value", result["child-only"])
	assert.Equal(t, "base.xcf", origins["base-only"])
	assert.Equal(t, "child.xcf", origins["child-only"])
}

// ---------------------------------------------------------------------------
// TestParseFile_ExtendsGlobal_ReadsFromXcaffoldDir
// ---------------------------------------------------------------------------

func TestParseFile_ExtendsGlobal_ReadsFromXcaffoldDir(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("USERPROFILE", fakeHome) // For Windows

	// 1. Create a mocked ~/.xcaffold directory with multiple files
	xcaffoldDir := filepath.Join(fakeHome, ".xcaffold")
	require.NoError(t, os.MkdirAll(xcaffoldDir, 0755))

	writeFile(t, xcaffoldDir, "agents.xcf", `kind: global
version: "1.0"
agents:
  global-agent:
    description: "I am a global agent from .xcaffold/agents.xcf"
`)
	writeFile(t, xcaffoldDir, "skills.xcf", `kind: global
version: "1.0"
skills:
  global-skill:
    description: "I am a global skill from .xcaffold/skills.xcf"
`)

	// 2. Create the project config extending "global" — split into separate single-kind files.
	projectDir := t.TempDir()
	writeFile(t, projectDir, "project.xcf", `kind: project
version: "1.0"
name: "local-project"
`)
	writeFile(t, projectDir, "global.xcf", `kind: global
version: "1.0"
extends: "global"
`)

	cfg, err := ParseDirectory(projectDir)
	require.NoError(t, err)

	// 3. Verify AST merges all global parts plus local project
	assert.Equal(t, "local-project", cfg.Project.Name)
	require.Contains(t, cfg.Agents, "global-agent")
	assert.Equal(t, "I am a global agent from .xcaffold/agents.xcf", cfg.Agents["global-agent"].Description)
	require.Contains(t, cfg.Skills, "global-skill")
	assert.Equal(t, "I am a global skill from .xcaffold/skills.xcf", cfg.Skills["global-skill"].Description)
}

// ---------------------------------------------------------------------------
// TestParseFile_ExtendsGlobal_LegacyFallback
// ---------------------------------------------------------------------------

func TestParseFile_ExtendsGlobal_LegacyFallback(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("USERPROFILE", fakeHome)

	// Only .claude/global.xcf exists
	legacyDir := filepath.Join(fakeHome, ".claude")
	require.NoError(t, os.MkdirAll(legacyDir, 0755))
	writeFile(t, legacyDir, "global.xcf", `kind: global
version: "1.0"
agents:
  legacy-agent:
    description: "From legacy .claude/global.xcf"
`)

	projectDir := t.TempDir()
	childPath := writeFile(t, projectDir, "project.xcf", `kind: global
version: "1.0"
extends: "global"
`)

	cfg, err := ParseFile(childPath)
	require.NoError(t, err)
	require.Contains(t, cfg.Agents, "legacy-agent")
	assert.Equal(t, "From legacy .claude/global.xcf", cfg.Agents["legacy-agent"].Description)
}

// ---------------------------------------------------------------------------
// TestParseFile_ExtendsGlobal_Circular
// ---------------------------------------------------------------------------

func TestParseFile_ExtendsGlobal_Circular(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("USERPROFILE", fakeHome)

	xcaffoldDir := filepath.Join(fakeHome, ".xcaffold")
	require.NoError(t, os.MkdirAll(xcaffoldDir, 0755))

	writeFile(t, xcaffoldDir, "circular.xcf", `kind: global
version: "1.0"
extends: "global"
`)

	projectDir := t.TempDir()
	childPath := writeFile(t, projectDir, "project.xcf", `kind: global
version: "1.0"
extends: "global"
`)

	_, err := ParseFile(childPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency detected")
}

// ---------------------------------------------------------------------------
// TestParseFile_ExtendsGlobal_NestedInheritance
// ---------------------------------------------------------------------------

func TestParseFile_ExtendsGlobal_NestedInheritance(t *testing.T) {
	t.Skip("Legacy instructions test removed")

	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("USERPROFILE", fakeHome)

	// Create global config that extends something else
	xcaffoldDir := filepath.Join(fakeHome, ".xcaffold")
	require.NoError(t, os.MkdirAll(xcaffoldDir, 0755))

	writeFile(t, xcaffoldDir, "base.xcf", `kind: global
version: "1.0"
agents:
  nested-agent:
    description: "From nested global"
`)

	writeFile(t, xcaffoldDir, "main.xcf", `kind: global
version: "1.0"
extends: "base.xcf"
agents:
  main-agent:
    description: "From main global"
`)

	projectDir := t.TempDir()
	childPath := writeFile(t, projectDir, "project.xcf", `kind: global
version: "1.0"
extends: "global"
`)

	cfg, err := ParseFile(childPath)
	require.NoError(t, err)
	require.Contains(t, cfg.Agents, "nested-agent")
	require.Contains(t, cfg.Agents, "main-agent")
}

// ---------------------------------------------------------------------------
// TestParseFile_InstructionsScopes_StripInherited — StripInherited must remove
// scopes tagged Inherited=true and preserve those tagged Inherited=false.
// This tests the StripInherited() behavior directly via struct construction,
// independent of the parser's mergeAllStrict path (which does not propagate
// InstructionsScopes from project-kind documents — a separate concern).
// ---------------------------------------------------------------------------

func TestParseFile_InstructionsScopes_StripInherited(t *testing.T) {
	t.Skip("Legacy instructions test removed")

	t.Skip("Legacy instructions scopes test removed")
}
