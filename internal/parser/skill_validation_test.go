package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSkillDirectory_NonExistentDir(t *testing.T) {
	result := ValidateSkillDirectory("/tmp/does-not-exist-xcaffold-test", "my-skill")
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Error(), "cannot read skill directory")
	require.Empty(t, result.Warnings)
}

func TestValidateSkillDirectory_UnknownSubdir(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "templates"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, "my-skill.xcf"), []byte("---\nkind: skill\n---\n"), 0o644)

	result := ValidateSkillDirectory(tmpDir, "my-skill")
	require.NotEmpty(t, result.Errors, "expected error for unknown subdir 'templates/'")
	assert.Contains(t, result.Errors[0].Error(), "templates")
}

func TestValidateSkillDirectory_StrayFile(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "my-skill.xcf"), []byte("---\nkind: skill\n---\n"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "notes.txt"), []byte("stray"), 0o644)

	result := ValidateSkillDirectory(tmpDir, "my-skill")
	require.NotEmpty(t, result.Errors, "expected error for stray file 'notes.txt'")
	assert.Contains(t, result.Errors[0].Error(), "notes.txt")
}

func TestValidateSkillDirectory_MaxDepth(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "references", "nested"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, "my-skill.xcf"), []byte("---\nkind: skill\n---\n"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "references", "nested", "deep.md"), []byte("deep"), 0o644)

	result := ValidateSkillDirectory(tmpDir, "my-skill")
	require.NotEmpty(t, result.Errors, "expected error for nested subdir")
	assert.Contains(t, result.Errors[0].Error(), "nested")
}

func TestValidateSkillDirectory_ValidStructure(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "references"), 0o755)
	os.MkdirAll(filepath.Join(tmpDir, "scripts"), 0o755)
	os.MkdirAll(filepath.Join(tmpDir, "assets"), 0o755)
	os.MkdirAll(filepath.Join(tmpDir, "examples"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, "my-skill.xcf"), []byte("---\nkind: skill\n---\n"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "references", "guide.md"), []byte("# Guide"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "scripts", "run.sh"), []byte("#!/bin/bash"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "assets", "template.json"), []byte("{}"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "examples", "sample.md"), []byte("# Sample"), 0o644)

	result := ValidateSkillDirectory(tmpDir, "my-skill")
	assert.Empty(t, result.Errors, "expected no errors for valid structure")
	assert.Empty(t, result.Warnings, "expected no warnings for valid structure")
}

func TestValidateSkillDirectory_FileTypeWarning(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "references"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, "my-skill.xcf"), []byte("---\nkind: skill\n---\n"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "references", "run.sh"), []byte("#!/bin/bash"), 0o644)

	result := ValidateSkillDirectory(tmpDir, "my-skill")
	assert.Empty(t, result.Errors, "expected no hard errors for misplaced file type")
	require.NotEmpty(t, result.Warnings, "expected warning for .sh in references/")
	assert.Contains(t, result.Warnings[0].Error(), ".sh")
}

func TestValidateSkillDirectory_SkillDirWithNoSubdirs(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "my-skill.xcf"), []byte("---\nkind: skill\n---\n"), 0o644)

	result := ValidateSkillDirectory(tmpDir, "my-skill")
	assert.Empty(t, result.Errors, "skill with only .xcf and no subdirs should be valid")
	assert.Empty(t, result.Warnings, "skill with only .xcf and no subdirs should have no warnings")
}

func TestValidateSkillDirectory_SkillMDAtRoot(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "my-skill.xcf"), []byte("---\nkind: skill\n---\n"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "SKILL.md"), []byte("# My Skill"), 0o644)

	result := ValidateSkillDirectory(tmpDir, "my-skill")
	require.NotEmpty(t, result.Errors, "SKILL.md at root should be rejected — only .xcf allowed")
	assert.Contains(t, result.Errors[0].Error(), "SKILL.md")
}
