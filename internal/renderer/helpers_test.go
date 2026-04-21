package renderer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/output"
)

func TestCompileSkillSubdir_CopiesFiles(t *testing.T) {
	tmpDir := t.TempDir()
	refDir := filepath.Join(tmpDir, "refs")
	os.MkdirAll(refDir, 0o755)
	os.WriteFile(filepath.Join(refDir, "doc.md"), []byte("# Reference"), 0o644)

	out := &output.Output{Files: make(map[string]string)}
	err := CompileSkillSubdir("my-skill", "references", []string{"refs/doc.md"}, tmpDir, out)
	if err != nil {
		t.Fatal(err)
	}

	expected := "skills/my-skill/references/doc.md"
	if _, ok := out.Files[expected]; !ok {
		t.Errorf("expected file %q in output", expected)
	}
}

func TestCompileSkillSubdir_RejectsPathTraversal(t *testing.T) {
	out := &output.Output{Files: make(map[string]string)}
	err := CompileSkillSubdir("my-skill", "references", []string{"../../etc/passwd"}, t.TempDir(), out)
	if err == nil {
		t.Error("expected error for path traversal, got nil")
	}
}

func TestCompileSkillSubdir_EmptyPaths(t *testing.T) {
	out := &output.Output{Files: make(map[string]string)}
	err := CompileSkillSubdir("my-skill", "references", nil, t.TempDir(), out)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Files) != 0 {
		t.Errorf("expected no files for nil paths, got %d", len(out.Files))
	}
}

func TestCompileSkillSubdir_GlobExpansion(t *testing.T) {
	tmpDir := t.TempDir()
	scriptDir := filepath.Join(tmpDir, "scripts")
	os.MkdirAll(scriptDir, 0o755)
	os.WriteFile(filepath.Join(scriptDir, "a.sh"), []byte("#!/bin/sh\necho a"), 0o644)
	os.WriteFile(filepath.Join(scriptDir, "b.sh"), []byte("#!/bin/sh\necho b"), 0o644)

	out := &output.Output{Files: make(map[string]string)}
	err := CompileSkillSubdir("my-skill", "scripts", []string{"scripts/*.sh"}, tmpDir, out)
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"a.sh", "b.sh"} {
		key := "skills/my-skill/scripts/" + name
		if _, ok := out.Files[key]; !ok {
			t.Errorf("expected glob-expanded file %q in output", key)
		}
	}
}

func TestCompileSkillSubdir_MissingLiteralFile(t *testing.T) {
	out := &output.Output{Files: make(map[string]string)}
	err := CompileSkillSubdir("my-skill", "assets", []string{"nonexistent.png"}, t.TempDir(), out)
	if err == nil {
		t.Error("expected error for missing literal file, got nil")
	}
}
