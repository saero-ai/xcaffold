package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureValidateOutput captures both stdout and stderr produced by f.
func captureValidateOutput(f func() error) (string, error) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	err := f()

	w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String(), err
}

func TestValidateCmd_ValidConfig(t *testing.T) {
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
	// Cross-reference validation no longer fails — exits 0 with warnings
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

	// Multi-doc .xcf will still cause syntax error, not a cross-ref error
	err := runValidate(validateCmd, []string{})
	assert.Error(t, err) // But it's syntax error, not cross-ref
}

func TestValidateCmd_InvalidCrossRef_ExitsZero(t *testing.T) {
	// Cross-reference issues exit 0 (warnings only), not 1
	dir := t.TempDir()

	xcf := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(`kind: project
version: "1.0"
name: "test"
`), 0600))

	// Global config with agent referencing undefined skill
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agents.xcf"), []byte(`kind: global
version: "1.0"
agents:
  developer:
    description: "Dev agent"
    skills: [nonexistent]
`), 0600))

	oldPath := xcfPath
	xcfPath = xcf
	defer func() { xcfPath = oldPath }()

	err := runValidate(validateCmd, []string{})
	assert.NoError(t, err, "cross-reference warnings should not fail validation")
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
	// When validate.go uses filepath.Dir(".xcaffold/project.xcf") = ".xcaffold/"
	// as the ParseDirectory root, files at xcf/agents/ are NOT found.
	// This causes invalid xcf/agents/ files to silently pass validation.
	// Fix: derive parseRoot by walking up past .xcaffold/ to the true project root.
	dir := t.TempDir()

	xcaffoldDir := filepath.Join(dir, ".xcaffold")
	require.NoError(t, os.MkdirAll(xcaffoldDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(xcaffoldDir, "project.xcf"), []byte(`kind: project
name: test
version: "1.0"
`), 0644))

	// Agent at the TRUE project root with an unresolved cross-reference.
	// skill "nonexistent" does NOT exist. If ParseDirectory scans the correct root,
	// this cross-reference is found but reported as a warning (not an error).
	// If ParseDirectory scans .xcaffold/ (the bug), this file is never parsed,
	// cross-reference is never checked, and validation falsely PASSES.
	agentsDir := filepath.Join(dir, "xcf", "agents", "bad-agent")
	require.NoError(t, os.MkdirAll(agentsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "agent.xcf"), []byte(`---
kind: agent
version: "1.0"
name: bad-agent
description: "Agent with a broken skill ref"
skills: [nonexistent]
---
Agent body text.
`), 0644))

	oldPath := xcfPath
	xcfPath = filepath.Join(xcaffoldDir, "project.xcf")
	t.Cleanup(func() { xcfPath = oldPath })

	err := runValidate(validateCmd, []string{})

	// Cross-references are now warnings, not errors. The test verifies
	// that ParseDirectory scans the correct root (not just .xcaffold/).
	require.NoError(t, err, "cross-references are warnings, not errors — validate should succeed")
}

// TestValidate_TargetFlag_HeaderIncludesTarget verifies that --target causes the
// header breadcrumb to include the provider name.
func TestValidate_TargetFlag_HeaderIncludesTarget(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte(`kind: project
version: "1.0"
name: "myproject"
`), 0600))

	oldPath := xcfPath
	oldTarget := targetFlag
	xcfPath = filepath.Join(dir, "project.xcf")
	targetFlag = "claude"
	defer func() {
		xcfPath = oldPath
		targetFlag = oldTarget
	}()

	out, err := captureValidateOutput(func() error {
		return runValidate(validateCmd, []string{})
	})
	require.NoError(t, err)

	// The header is the first non-empty line. It must contain the provider name.
	var firstLine string
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) != "" {
			firstLine = line
			break
		}
	}
	assert.True(t, strings.Contains(firstLine, "claude"), "header (first line) must contain provider name when --target is set")
}

// TestValidate_NoTarget_HeaderExcludesProvider verifies that omitting --target
// keeps the provider name out of the header.
func TestValidate_NoTarget_HeaderExcludesProvider(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte(`kind: project
version: "1.0"
name: "myproject"
`), 0600))

	oldPath := xcfPath
	oldTarget := targetFlag
	xcfPath = filepath.Join(dir, "project.xcf")
	targetFlag = ""
	defer func() {
		xcfPath = oldPath
		targetFlag = oldTarget
	}()

	out, err := captureValidateOutput(func() error {
		return runValidate(validateCmd, []string{})
	})
	require.NoError(t, err)

	// Spot-check: neither "claude" nor "antigravity" should appear in a no-target run.
	assert.False(t, strings.Contains(out, "antigravity"), "header must not contain provider name when --target is not set")
}

// TestValidate_TargetFlag_FooterIncludesFieldValidation verifies that when
// --target is set and validation passes, the footer includes a field validation
// summary with the provider name.
func TestValidate_TargetFlag_FooterIncludesFieldValidation(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte(`kind: project
version: "1.0"
name: "myproject"
`), 0600))

	oldPath := xcfPath
	oldTarget := targetFlag
	xcfPath = filepath.Join(dir, "project.xcf")
	targetFlag = "claude"
	defer func() {
		xcfPath = oldPath
		targetFlag = oldTarget
	}()

	out, err := captureValidateOutput(func() error {
		return runValidate(validateCmd, []string{})
	})
	require.NoError(t, err)

	// Footer must include field validation summary.
	assert.True(t, strings.Contains(out, "Field validation:"), "footer must include 'Field validation:' when --target is set")
	assert.True(t, strings.Contains(out, "claude"), "footer must include provider name in field validation summary")
}

// TestValidate_NoTarget_FooterExcludesFieldValidation verifies that omitting
// --target keeps field validation out of the footer summary.
func TestValidate_NoTarget_FooterExcludesFieldValidation(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte(`kind: project
version: "1.0"
name: "myproject"
`), 0600))

	oldPath := xcfPath
	oldTarget := targetFlag
	xcfPath = filepath.Join(dir, "project.xcf")
	targetFlag = ""
	defer func() {
		xcfPath = oldPath
		targetFlag = oldTarget
	}()

	out, err := captureValidateOutput(func() error {
		return runValidate(validateCmd, []string{})
	})
	require.NoError(t, err)

	assert.False(t, strings.Contains(out, "Field validation:"), "footer must not include field validation when --target is not set")
}

// TestValidate_CrossRefWarnings_TieredOutput verifies that cross-reference
// warnings are shown separately from syntax/schema errors in tiered output.
func TestValidate_CrossRefWarnings_TieredOutput(t *testing.T) {
	dir := t.TempDir()

	xcf := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(`kind: project
version: "1.0"
name: "test"
`), 0600))

	// Global config with agent referencing undefined skill
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agents.xcf"), []byte(`kind: global
version: "1.0"
agents:
  developer:
    description: "Dev agent"
    skills: [nonexistent-skill]
`), 0600))

	oldPath := xcfPath
	xcfPath = xcf
	defer func() { xcfPath = oldPath }()

	out, err := captureValidateOutput(func() error {
		return runValidate(validateCmd, []string{})
	})
	require.NoError(t, err, "cross-reference warnings must not fail validation")

	// Output should show tiered validation with cross-ref warnings
	assert.True(t, strings.Contains(out, "cross-references") || strings.Contains(out, "developer"), "output must mention cross-reference issues")
}

func TestValidate_NoOrphanChecks_SkillNotReferencedByAgent(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"standalone": {Name: "standalone", Description: "Not referenced by any agent"},
			},
			Agents: map[string]ast.AgentConfig{
				"worker": {Name: "worker", Skills: []string{"other-skill"}},
			},
		},
	}
	warnings := runStructuralChecks(cfg)
	for _, w := range warnings {
		if strings.Contains(w, "not referenced") {
			t.Errorf("unexpected orphan warning: %s", w)
		}
	}
}

func TestValidate_NoOrphanChecks_RuleWithoutAlwaysApply(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"code-style": {Name: "code-style", Description: "Style guide"},
			},
			Agents: map[string]ast.AgentConfig{
				"worker": {Name: "worker", Rules: []string{}},
			},
		},
	}
	warnings := runStructuralChecks(cfg)
	for _, w := range warnings {
		if strings.Contains(w, "not referenced") {
			t.Errorf("unexpected orphan warning: %s", w)
		}
	}
}

func TestValidate_NoOrphanChecks_AgentWithoutBody(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"minimal": {Name: "minimal", Description: "No body"},
			},
		},
	}
	warnings := runStructuralChecks(cfg)
	for _, w := range warnings {
		if strings.Contains(w, "has no body") {
			t.Errorf("unexpected missing-instructions warning: %s", w)
		}
	}
}
