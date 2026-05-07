package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateHooksDirectory_MissingScript_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, "xcaf", "hooks", "enforce")
	os.MkdirAll(hooksDir, 0755)

	cfg := &ast.XcaffoldConfig{
		Hooks: map[string]ast.NamedHookConfig{
			"enforce": {
				Name: "enforce",
				Events: ast.HookConfig{
					"PreToolUse": []ast.HookMatcherGroup{
						{
							Matcher: "Bash",
							Hooks: []ast.HookHandler{
								{
									Type:    "command",
									Command: "bash xcaf/hooks/enforce/scripts/enforce-standards.sh",
								},
							},
						},
					},
				},
			},
		},
	}

	errors := validateHooksDirectory(cfg, dir)
	require.NotEmpty(t, errors, "missing script should produce an error")
	assert.Contains(t, errors[0], "enforce-standards.sh")
}

func TestValidateHooksDirectory_MissingArtifactDir_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, "xcaf", "hooks", "enforce")
	os.MkdirAll(hooksDir, 0755)

	cfg := &ast.XcaffoldConfig{
		Hooks: map[string]ast.NamedHookConfig{
			"enforce": {
				Name:      "enforce",
				Artifacts: []string{"scripts", "data"},
			},
		},
	}

	errors := validateHooksDirectory(cfg, dir)
	require.NotEmpty(t, errors, "missing artifact dirs should produce errors")
	assert.GreaterOrEqual(t, len(errors), 2)
}

func TestValidateHooksDirectory_ValidHookDir_NoErrors(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, "xcaf", "hooks", "enforce")
	scriptsDir := filepath.Join(hooksDir, "scripts")
	os.MkdirAll(scriptsDir, 0755)
	os.WriteFile(filepath.Join(scriptsDir, "enforce-standards.sh"), []byte("#!/bin/bash\nexit 0\n"), 0755)

	cfg := &ast.XcaffoldConfig{
		Hooks: map[string]ast.NamedHookConfig{
			"enforce": {
				Name:      "enforce",
				Artifacts: []string{"scripts"},
				Events: ast.HookConfig{
					"PreToolUse": []ast.HookMatcherGroup{
						{
							Matcher: "Bash",
							Hooks: []ast.HookHandler{
								{
									Type:    "command",
									Command: "bash xcaf/hooks/enforce/scripts/enforce-standards.sh",
								},
							},
						},
					},
				},
			},
		},
	}

	errors := validateHooksDirectory(cfg, dir)
	require.Empty(t, errors, "valid hook dir should produce no errors: %v", errors)
}

func TestValidateHooksDirectory_NoHooks_NoErrors(t *testing.T) {
	dir := t.TempDir()
	cfg := &ast.XcaffoldConfig{}

	errors := validateHooksDirectory(cfg, dir)
	require.Empty(t, errors)
}
