package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSkillDirectory_NonExistentDir(t *testing.T) {
	result := ValidateSkillDirectory("/tmp/does-not-exist-xcaffold-test", "my-skill", nil)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Error(), "cannot read skill directory")
	require.Empty(t, result.Warnings)
}

func TestValidateSkillDirectory_SubdirNotInArtifacts_Warning(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "templates"), 0o755)
	os.MkdirAll(filepath.Join(tmpDir, "references"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, "my-skill.xcf"), []byte("---\nkind: skill\n---\n"), 0o644)

	// Create a skill with only references artifact, but templates exists on disk
	artifacts := []string{"references"}
	result := ValidateSkillDirectory(tmpDir, "my-skill", artifacts)

	// Should produce a WARNING, not an error
	require.Empty(t, result.Errors, "subdir not in artifacts should warn, not error")
	require.NotEmpty(t, result.Warnings, "expected warning for subdirectory not in artifacts")
	assert.Contains(t, result.Warnings[0].Error(), "templates")
	assert.Contains(t, result.Warnings[0].Error(), "not declared in artifacts")
}

func TestValidateSkillDirectory_StrayFile(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "my-skill.xcf"), []byte("---\nkind: skill\n---\n"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "notes.txt"), []byte("stray"), 0o644)

	result := ValidateSkillDirectory(tmpDir, "my-skill", nil)
	require.NotEmpty(t, result.Errors, "expected error for stray file 'notes.txt'")
	assert.Contains(t, result.Errors[0].Error(), "notes.txt")
}

func TestValidateSkillDirectory_MaxDepth(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "references", "nested"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, "my-skill.xcf"), []byte("---\nkind: skill\n---\n"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "references", "nested", "deep.md"), []byte("deep"), 0o644)

	artifacts := []string{"references"}
	result := ValidateSkillDirectory(tmpDir, "my-skill", artifacts)
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

	artifacts := []string{"references", "scripts", "assets", "examples"}
	result := ValidateSkillDirectory(tmpDir, "my-skill", artifacts)
	assert.Empty(t, result.Errors, "expected no errors for valid structure")
	assert.Empty(t, result.Warnings, "expected no warnings for valid structure")
}

func TestValidateSkillDirectory_FileTypeWarning(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "references"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, "my-skill.xcf"), []byte("---\nkind: skill\n---\n"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "references", "run.sh"), []byte("#!/bin/bash"), 0o644)

	artifacts := []string{"references"}
	result := ValidateSkillDirectory(tmpDir, "my-skill", artifacts)
	assert.Empty(t, result.Errors, "expected no hard errors for misplaced file type")
	require.NotEmpty(t, result.Warnings, "expected warning for .sh in references/")
	assert.Contains(t, result.Warnings[0].Error(), ".sh")
}

func TestValidateSkillDirectory_SkillDirWithNoSubdirs(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "my-skill.xcf"), []byte("---\nkind: skill\n---\n"), 0o644)

	result := ValidateSkillDirectory(tmpDir, "my-skill", nil)
	assert.Empty(t, result.Errors, "skill with only .xcf and no subdirs should be valid")
	assert.Empty(t, result.Warnings, "skill with only .xcf and no subdirs should have no warnings")
}

func TestValidateSkillDirectory_HiddenFilesIgnored(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "my-skill.xcf"), []byte("---\nkind: skill\n---\n"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, ".DS_Store"), []byte("binary"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, ".gitkeep"), []byte(""), 0o644)

	result := ValidateSkillDirectory(tmpDir, "my-skill", nil)
	if len(result.Errors) != 0 {
		t.Errorf("hidden files should be silently ignored, got errors: %v", result.Errors)
	}
}

func TestValidateSkillDirectory_SkillMDAtRoot(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "my-skill.xcf"), []byte("---\nkind: skill\n---\n"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "SKILL.md"), []byte("# My Skill"), 0o644)

	result := ValidateSkillDirectory(tmpDir, "my-skill", nil)
	require.NotEmpty(t, result.Errors, "SKILL.md at root should be rejected — only .xcf allowed")
	assert.Contains(t, result.Errors[0].Error(), "SKILL.md")
}

func TestValidateSkillDirectory_OverrideFiles_AcceptsKindProviderXcf(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "skill.xcf"), []byte("---\nkind: skill\n---\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "skill.claude.xcf"), []byte("---\nkind: skill\n---\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "skill.gemini.xcf"), []byte("---\nkind: skill\n---\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "skill.cursor.xcf"), []byte("---\nkind: skill\n---\n"), 0o644)

	result := ValidateSkillDirectory(dir, "tdd", nil)
	require.Empty(t, result.Errors, "override files should not produce errors: %v", result.Errors)
	require.Empty(t, result.Warnings, "override files should not produce warnings: %v", result.Warnings)
}

func TestValidateSkillDirectory_OverrideFiles_RejectsInvalidPattern(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "skill.xcf"), []byte("---\nkind: skill\n---\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "random.file.xcf"), []byte("stuff"), 0o644)

	result := ValidateSkillDirectory(dir, "tdd", nil)
	require.NotEmpty(t, result.Errors, "invalid pattern should produce an error")
}

func TestValidateSkillDirectory_XcfInExamples_Allowed(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "skill.xcf"), []byte("---\nkind: skill\n---\n"), 0o644)

	exDir := filepath.Join(dir, "examples")
	os.MkdirAll(exDir, 0o755)
	os.WriteFile(filepath.Join(exDir, "sample-agent.xcf"), []byte("---\nkind: agent\n---\n"), 0o644)
	os.WriteFile(filepath.Join(exDir, "readme.md"), []byte("# Examples\n"), 0o644)

	artifacts := []string{"examples"}
	result := ValidateSkillDirectory(dir, "tdd", artifacts)
	require.Empty(t, result.Errors, "no errors expected: %v", result.Errors)
	require.Empty(t, result.Warnings, ".xcf in examples should not warn: %v", result.Warnings)
}

func TestValidateSkillDirectory_SubdirInArtifacts_Allowed(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "db"), 0o755)
	os.MkdirAll(filepath.Join(tmpDir, "tests"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, "skill.xcf"), []byte("---\nkind: skill\n---\n"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "db", "schema.sql"), []byte("CREATE TABLE"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "tests", "fixtures.json"), []byte("{}"), 0o644)

	artifacts := []string{"db", "tests"}
	result := ValidateSkillDirectory(tmpDir, "my-skill", artifacts)

	// Should produce zero errors and zero warnings
	assert.Empty(t, result.Errors, "custom artifact dirs should be allowed")
	assert.Empty(t, result.Warnings, "custom artifact dirs should not warn")
}

func TestValidateSkillDirectory_ArtifactDeclaredButMissing_Error(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "db"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, "skill.xcf"), []byte("---\nkind: skill\n---\n"), 0o644)

	// Declare both db and nonexistent, but only create db
	artifacts := []string{"db", "nonexistent"}
	result := ValidateSkillDirectory(tmpDir, "my-skill", artifacts)

	// Should produce a hard error for the missing declared artifact
	require.NotEmpty(t, result.Errors, "missing declared artifact should be an error")
	assert.Contains(t, result.Errors[0].Error(), "nonexistent")
	assert.Contains(t, result.Errors[0].Error(), "does not exist")
}

func TestValidateSkillDirectory_EmptyArtifacts_AllSubdirsWarn(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "references"), 0o755)
	os.MkdirAll(filepath.Join(tmpDir, "scripts"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, "skill.xcf"), []byte("---\nkind: skill\n---\n"), 0o644)

	// Pass empty artifacts — no dirs are declared
	result := ValidateSkillDirectory(tmpDir, "my-skill", []string{})

	// Should produce warnings for both canonical dirs since they're not declared
	require.NotEmpty(t, result.Warnings, "undeclared dirs should warn")
	warningText := fmt.Sprintf("%v", result.Warnings)
	assert.Contains(t, warningText, "references")
	assert.Contains(t, warningText, "scripts")
	assert.Contains(t, warningText, "not declared in artifacts")
}
