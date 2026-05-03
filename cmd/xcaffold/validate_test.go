package main

import (
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

func TestValidate_GlobalFlag_ReturnsGuardError(t *testing.T) {
	globalFlag = true
	defer func() { globalFlag = false }()

	err := runValidate(validateCmd, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "global scope is not yet available")
}

func TestValidate_GlobalFlag_WithBlueprint_ReturnsGuardError(t *testing.T) {
	validateBlueprintFlag = "my-blueprint"
	globalFlag = true
	defer func() {
		validateBlueprintFlag = ""
		globalFlag = false
	}()

	err := runValidate(validateCmd, []string{})
	require.Error(t, err)
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

	err := runValidate(validateCmd, []string{})
	// Structural checks warn but don't fail
	assert.NoError(t, err)
}

// TestValidate_TargetFlag_EmitsFidelityErrors verifies that --target causes
// validate to fail when the project contains a resource with fields that are
// unsupported by the specified target provider.
func TestValidate_TargetFlag_EmitsFidelityErrors(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte(`kind: project
version: "1.0"
name: "field-test"
`), 0600))

	// effort: low is unsupported by antigravity — produces a LevelError fidelity note.
	// Agents must be in xcf/agents/<id>/<file>.xcf (directory-per-resource layout).
	devDir := filepath.Join(dir, "xcf", "agents", "dev")
	require.NoError(t, os.MkdirAll(devDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(devDir, "agent.xcf"), []byte(`---
kind: agent
version: "1.0"
name: dev
description: "Dev agent"
effort: low
---
You are a developer.
`), 0600))

	oldPath := xcfPath
	oldTarget := targetFlag
	xcfPath = filepath.Join(dir, "project.xcf")
	targetFlag = "antigravity"
	defer func() {
		xcfPath = oldPath
		targetFlag = oldTarget
	}()

	err := runValidate(validateCmd, []string{})
	require.Error(t, err, "validate must fail when target has unsupported fields")
	assert.Contains(t, err.Error(), "validation failed")
}

// TestValidate_NoTarget_NoFieldCheck verifies that omitting --target skips
// the compile-time field validation check entirely.
func TestValidate_NoTarget_NoFieldCheck(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte(`kind: project
version: "1.0"
name: "field-test"
`), 0600))

	// effort: low is unsupported by antigravity, but without --target no check runs.
	devDir := filepath.Join(dir, "xcf", "agents", "dev")
	require.NoError(t, os.MkdirAll(devDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(devDir, "agent.xcf"), []byte(`---
kind: agent
version: "1.0"
name: dev
description: "Dev agent"
effort: low
---
You are a developer.
`), 0600))

	oldPath := xcfPath
	oldTarget := targetFlag
	xcfPath = filepath.Join(dir, "project.xcf")
	targetFlag = ""
	defer func() {
		xcfPath = oldPath
		targetFlag = oldTarget
	}()

	err := runValidate(validateCmd, []string{})
	assert.NoError(t, err, "validate without --target must not fail on unsupported fields")
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
