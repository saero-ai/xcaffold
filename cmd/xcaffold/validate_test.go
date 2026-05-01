package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCmd_ValidConfig(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	dir := t.TempDir()
	xcf := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(`kind: project
version: "1.0"
name: "test"
`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "global.xcf"), []byte(`kind: global
version: "1.0"
agents:
  developer:
    description: "Dev agent"
    skills: [deploy]
skills:
  deploy:
    description: "Deploy skill"
`), 0600))

	// Set the package-level xcfPath directly (PersistentPreRunE would normally do this)
	oldPath := xcfPath
	xcfPath = xcf
	defer func() { xcfPath = oldPath }()

	err := runValidate(validateCmd, []string{})
	assert.NoError(t, err)
}

func TestValidateCmd_InvalidCrossRef(t *testing.T) {
	dir := t.TempDir()
	xcf := filepath.Join(dir, "project.xcf")
	content := `---
kind: project
version: "1.0"
name: "test"
---
kind: global
version: "1.0"
agents:
  developer:
    description: "Dev agent"
    skills: [nonexistent]
`
	require.NoError(t, os.WriteFile(xcf, []byte(content), 0600))

	oldPath := xcfPath
	xcfPath = xcf
	defer func() { xcfPath = oldPath }()

	err := runValidate(validateCmd, []string{})
	assert.Error(t, err)
}

func TestValidate_GlobalFlag_FileNotFound(t *testing.T) {
	home := t.TempDir() // no .xcaffold/global.xcf inside
	t.Setenv("HOME", home)

	globalFlag = true
	globalXcfPath = filepath.Join(home, ".xcaffold", "global.xcf")
	defer func() { globalFlag = false }()

	err := runValidate(validateCmd, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "global scope is not yet available")
}

func TestValidate_GlobalFlag_InvalidYAML(t *testing.T) {
	home := t.TempDir()
	xcaffoldDir := filepath.Join(home, ".xcaffold")
	require.NoError(t, os.MkdirAll(xcaffoldDir, 0700))

	globalXCF := filepath.Join(xcaffoldDir, "global.xcf")
	require.NoError(t, os.WriteFile(globalXCF, []byte(":\tinvalid: yaml: :::\n"), 0600))

	t.Setenv("HOME", home)
	globalFlag = true
	globalXcfPath = globalXCF
	defer func() { globalFlag = false }()

	err := runValidate(validateCmd, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "global scope is not yet available")
}

func TestValidate_GlobalFlag_ValidFile(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	home := t.TempDir()
	xcaffoldDir := filepath.Join(home, ".xcaffold")
	require.NoError(t, os.MkdirAll(xcaffoldDir, 0700))

	globalXCF := filepath.Join(xcaffoldDir, "global.xcf")
	content := `kind: global
version: "1.0"
agents:
  reviewer:
    name: Reviewer
`
	require.NoError(t, os.WriteFile(globalXCF, []byte(content), 0600))

	t.Setenv("HOME", home)
	globalFlag = true
	globalXcfPath = globalXCF
	defer func() { globalFlag = false }()

	err := runValidate(validateCmd, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "global scope is not yet available")
}

func TestValidate_BlueprintFlag_MutualExclusion_WithGlobal(t *testing.T) {
	validateBlueprintFlag = "my-blueprint"
	globalFlag = true
	defer func() {
		validateBlueprintFlag = ""
		globalFlag = false
	}()

	err := runValidate(validateCmd, []string{})
	require.Error(t, err)
	// Global scope guard fires before mutual exclusion check
	assert.Contains(t, err.Error(), "global scope is not yet available")
}

// TestCheckBashWithoutHook_ProjectHook_NoWarn verifies that a project-level
// PreToolUse hook declared in the "default" block suppresses the Bash warning.
// Before the fix this failed because the code indexed cfg.Hooks["PreToolUse"]
// (wrong level) instead of cfg.Hooks["default"].Events["PreToolUse"].
func TestCheckBashWithoutHook_ProjectHook_NoWarn(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {
					Name:  "Dev",
					Body:  "instructions",
					Tools: []string{"Bash"},
					Hooks: ast.HookConfig{},
				},
			},
		},
		Hooks: map[string]ast.NamedHookConfig{
			"default": {
				Name: "default",
				Events: ast.HookConfig{
					"PreToolUse": []ast.HookMatcherGroup{
						{Hooks: []ast.HookHandler{{Command: "validate.sh"}}},
					},
				},
			},
		},
	}

	warnings := checkBashWithoutHook(cfg)
	assert.Empty(t, warnings, "project-level PreToolUse hook must suppress Bash warning")
}

// TestCheckBashWithoutHook_NoHook_Warns verifies that the warning fires when
// neither the project nor the agent has a PreToolUse hook.
func TestCheckBashWithoutHook_NoHook_Warns(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {
					Name:  "Dev",
					Body:  "instructions",
					Tools: []string{"Bash"},
					Hooks: ast.HookConfig{},
				},
			},
		},
		Hooks: map[string]ast.NamedHookConfig{},
	}

	warnings := checkBashWithoutHook(cfg)
	assert.Len(t, warnings, 1, "missing PreToolUse hook must produce a warning")
	assert.Contains(t, warnings[0], "PreToolUse")
}

func TestValidateCmd_StructuralChecks(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	dir := t.TempDir()
	xcf := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(`kind: project
version: "1.0"
name: "test"
`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "global.xcf"), []byte(`kind: global
version: "1.0"
agents:
  developer:
    description: "Dev agent"
skills:
  orphan:
    description: "No agent references this skill"
`), 0600))

	oldPath := xcfPath
	xcfPath = xcf
	defer func() { xcfPath = oldPath }()

	validateStructural = true
	defer func() { validateStructural = false }()

	err := runValidate(validateCmd, []string{})
	// Structural checks warn but don't fail
	assert.NoError(t, err)
}

func TestValidate_ManifestInXcaffoldDir_ParsesFullProjectRoot(t *testing.T) {
	// TC-17: When validate.go uses filepath.Dir(".xcaffold/project.xcf") = ".xcaffold/"
	// as the ParseDirectory root, files at xcf/agents/ are NOT found.
	// This causes invalid xcf/agents/ files to silently pass validation.
	// Fix: derive parseRoot by walking up past .xcaffold/ to the true project root.
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	dir := t.TempDir()

	xcaffoldDir := filepath.Join(dir, ".xcaffold")
	require.NoError(t, os.MkdirAll(xcaffoldDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(xcaffoldDir, "project.xcf"), []byte(`kind: project
name: test
version: "1.0"
`), 0644))

	// Agent at the TRUE project root with an invalid cross-reference.
	// skill "nonexistent" does NOT exist. If ParseDirectory scans the correct root,
	// this cross-reference is caught and validation FAILS.
	// If ParseDirectory scans .xcaffold/ (the bug), this file is never parsed,
	// cross-reference is never checked, and validation falsely PASSES.
	agentsDir := filepath.Join(dir, "xcf", "agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "bad-agent.xcf"), []byte(`kind: agent
version: "1.0"
name: Bad Agent
description: "Agent with a broken skill ref"
skills: [nonexistent]
---
do stuff
`), 0644))

	oldPath := xcfPath
	xcfPath = filepath.Join(xcaffoldDir, "project.xcf")
	t.Cleanup(func() { xcfPath = oldPath })

	err := runValidate(validateCmd, []string{})

	// Before fix: validation PASSES (bad-agent.xcf not found → cross-ref unchecked)
	// After fix:  validation FAILS  (bad-agent.xcf found → cross-ref caught → error)
	require.Error(t, err, "validate must detect cross-ref error in xcf/agents/ when manifest is in .xcaffold/")
}

func TestValidateCmd_OutputContainsHeader(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	dir := t.TempDir()
	xcf := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(`kind: project
version: "1.0"
name: "test"
`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "global.xcf"), []byte(`kind: global
version: "1.0"
agents:
  developer:
    description: "Dev agent"
    skills: [deploy]
skills:
  deploy:
    description: "Deploy skill"
`), 0600))

	// Create xcf directory with at least one .xcf file
	xcfDir := filepath.Join(dir, "xcf")
	require.NoError(t, os.MkdirAll(xcfDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(xcfDir, "project.xcf"), []byte(`kind: project
version: "1.0"
name: "test"
`), 0600))

	oldPath := xcfPath
	xcfPath = xcf
	defer func() { xcfPath = oldPath }()

	// Capture stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	oldStdout := os.Stdout
	os.Stdout = w

	defer func() {
		os.Stdout = oldStdout
		_ = w.Close()
	}()

	err = runValidate(validateCmd, []string{})
	assert.NoError(t, err)

	_ = w.Close()
	os.Stdout = oldStdout

	output, err := io.ReadAll(r)
	require.NoError(t, err)
	outputStr := string(output)

	// Output should contain header line (first non-empty line after header)
	// The header format is: <project>  ·  <context>
	assert.Regexp(t, `\S+\s+.*never applied|last applied`, outputStr, "output must contain header with last applied timestamp")
	// Should contain glyph or "ok" for at least one check
	assert.Regexp(t, `(✓|ok).*syntax`, outputStr, "output must indicate syntax check passed")
	// Should contain "files checked" in footer
	assert.Contains(t, outputStr, ".xcf files checked", "output must contain file count")
	// Should contain "Validation passed"
	assert.Contains(t, outputStr, "Validation passed", "output must contain validation result")
}
