package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSkillDirectory_UnknownSubdir(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "templates"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, "my-skill.xcf"), []byte("---\nkind: skill\n---\n"), 0o644)

	errs := ValidateSkillDirectory(tmpDir, "my-skill")
	if len(errs) == 0 {
		t.Fatal("expected error for unknown subdir 'templates/', got none")
	}
	if !strings.Contains(errs[0].Error(), "templates") {
		t.Errorf("error should mention 'templates', got: %v", errs[0])
	}
}

func TestValidateSkillDirectory_StrayFile(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "my-skill.xcf"), []byte("---\nkind: skill\n---\n"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "notes.txt"), []byte("stray"), 0o644)

	errs := ValidateSkillDirectory(tmpDir, "my-skill")
	if len(errs) == 0 {
		t.Fatal("expected error for stray file 'notes.txt', got none")
	}
	if !strings.Contains(errs[0].Error(), "notes.txt") {
		t.Errorf("error should mention 'notes.txt', got: %v", errs[0])
	}
}

func TestValidateSkillDirectory_MaxDepth(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "references", "nested"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, "my-skill.xcf"), []byte("---\nkind: skill\n---\n"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "references", "nested", "deep.md"), []byte("deep"), 0o644)

	errs := ValidateSkillDirectory(tmpDir, "my-skill")
	if len(errs) == 0 {
		t.Fatal("expected error for nested subdir, got none")
	}
	if !strings.Contains(errs[0].Error(), "nested") {
		t.Errorf("error should mention 'nested', got: %v", errs[0])
	}
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

	errs := ValidateSkillDirectory(tmpDir, "my-skill")
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid structure, got: %v", errs)
	}
}

func TestValidateSkillDirectory_FileTypeWarning(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "references"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, "my-skill.xcf"), []byte("---\nkind: skill\n---\n"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "references", "run.sh"), []byte("#!/bin/bash"), 0o644)

	errs := ValidateSkillDirectory(tmpDir, "my-skill")
	if len(errs) == 0 {
		t.Fatal("expected warning for .sh in references/, got none")
	}
	if !strings.Contains(errs[0].Error(), ".sh") {
		t.Errorf("error should mention '.sh', got: %v", errs[0])
	}
}

func TestValidateSkillDirectory_SkillDirWithNoSubdirs(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "my-skill.xcf"), []byte("---\nkind: skill\n---\n"), 0o644)

	errs := ValidateSkillDirectory(tmpDir, "my-skill")
	if len(errs) != 0 {
		t.Errorf("skill with only .xcf and no subdirs should be valid, got: %v", errs)
	}
}
