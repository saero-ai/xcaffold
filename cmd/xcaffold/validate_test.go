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

func TestValidate_GlobalFlag_FileNotFound(t *testing.T) {
	home := t.TempDir() // no .xcaffold/global.xcf inside
	t.Setenv("HOME", home)

	globalFlag = true
	globalXcfPath = filepath.Join(home, ".xcaffold", "global.xcf")
	defer func() { globalFlag = false }()

	err := runValidate(validateCmd, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to scan directory")
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
    instructions: "Review code."
`
	require.NoError(t, os.WriteFile(globalXCF, []byte(content), 0600))

	t.Setenv("HOME", home)
	globalFlag = true
	globalXcfPath = globalXCF
	defer func() { globalFlag = false }()

	err := runValidate(validateCmd, []string{})
	assert.NoError(t, err)
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
	assert.Contains(t, err.Error(), "--blueprint cannot be used with --global")
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
					Name:         "Dev",
					Instructions: "instructions",
					Tools:        []string{"Bash"},
					Hooks:        ast.HookConfig{},
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
					Name:         "Dev",
					Instructions: "instructions",
					Tools:        []string{"Bash"},
					Hooks:        ast.HookConfig{},
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
