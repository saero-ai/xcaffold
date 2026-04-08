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
// TestParseFile_ValidConfig — basic file parse, no extends
// ---------------------------------------------------------------------------

func TestParseFile_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "base.xcf", `
version: "1.0"
project:
  name: "valid-project"
  description: "A simple project."
agents:
  coder:
    description: "Writes code."
    model: "claude-3-7-sonnet-20250219"
`)

	cfg, err := ParseFile(path)
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
	dir := t.TempDir()

	writeFile(t, dir, "base.xcf", `
version: "1.0"
project:
  name: "base-project"
  description: "Base description."
agents:
  base-agent:
    description: "Base agent."
    model: "claude-3-5-haiku-20241022"
`)

	childPath := writeFile(t, dir, "child.xcf", `
extends: "base.xcf"
version: "1.0"
agents:
  child-agent:
    description: "Child agent."
    model: "claude-3-7-sonnet-20250219"
`)

	cfg, err := ParseFile(childPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Base project inherited
	assert.Equal(t, "base-project", cfg.Project.Name)
	assert.Equal(t, "Base description.", cfg.Project.Description)

	// Base agent inherited
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
	dir := t.TempDir()

	writeFile(t, dir, "base.xcf", `
version: "1.0"
project:
  name: "base-project"
  description: "Base description."
agents:
  shared-agent:
    description: "Base version of agent."
    model: "claude-3-5-haiku-20241022"
test:
  claude_path: "/usr/local/bin/claude"
  judge_model: "claude-3-5-haiku-20241022"
`)

	childPath := writeFile(t, dir, "child.xcf", `
extends: "base.xcf"
version: "1.0"
project:
  name: "child-project"
  description: "Child description."
agents:
  shared-agent:
    description: "Child version of agent."
    model: "claude-3-7-sonnet-20250219"
test:
  judge_model: "claude-3-opus-20240229"
`)

	cfg, err := ParseFile(childPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Child project overrides base
	assert.Equal(t, "child-project", cfg.Project.Name)
	assert.Equal(t, "Child description.", cfg.Project.Description)

	// Child agent overrides base agent (same key)
	require.Contains(t, cfg.Agents, "shared-agent")
	assert.Equal(t, "Child version of agent.", cfg.Agents["shared-agent"].Description)
	assert.Equal(t, "claude-3-7-sonnet-20250219", cfg.Agents["shared-agent"].Model)

	// Test config: child overrides judge_model, base claude_path is inherited
	assert.Equal(t, "/usr/local/bin/claude", cfg.Test.ClaudePath, "base claude_path should be inherited")
	assert.Equal(t, "claude-3-opus-20240229", cfg.Test.JudgeModel, "child judge_model should override base")
}

// ---------------------------------------------------------------------------
// TestParseFile_CircularExtendsDetected — A extends B, B extends A
// ---------------------------------------------------------------------------

func TestParseFile_CircularExtendsDetected(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "a.xcf", `
extends: "b.xcf"
version: "1.0"
project:
  name: "a-project"
`)

	writeFile(t, dir, "b.xcf", `
extends: "a.xcf"
version: "1.0"
project:
  name: "b-project"
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

	childPath := writeFile(t, dir, "child.xcf", `
extends: "nonexistent-base.xcf"
version: "1.0"
project:
  name: "child-project"
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
	tracker := make(map[string]map[string]string)
	tracker["agent"] = map[string]string{"shared": "base.xcf"}

	_, err := mergeMapStrict(base, child, "agent", "child.xcf", tracker)
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
	tracker := make(map[string]map[string]string)
	result, err := mergeMapStrict(base, child, "agent", "child.xcf", tracker)
	require.NoError(t, err)
	assert.Equal(t, "base-value", result["base-only"])
	assert.Equal(t, "child-value", result["child-only"])
}
