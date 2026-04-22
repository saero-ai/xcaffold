package renderer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/output"
)

func TestCompileSkillSubdir_CopiesFiles(t *testing.T) {
	tmpDir := t.TempDir()
	refDir := filepath.Join(tmpDir, "refs")
	os.MkdirAll(refDir, 0o755)
	os.WriteFile(filepath.Join(refDir, "doc.md"), []byte("# Reference"), 0o644)

	out := &output.Output{Files: make(map[string]string)}
	err := CompileSkillSubdir("my-skill", "references", "references", []string{"refs/doc.md"}, tmpDir, out)
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
	err := CompileSkillSubdir("my-skill", "references", "references", []string{"../../etc/passwd"}, t.TempDir(), out)
	if err == nil {
		t.Error("expected error for path traversal, got nil")
	}
}

func TestCompileSkillSubdir_EmptyPaths(t *testing.T) {
	out := &output.Output{Files: make(map[string]string)}
	err := CompileSkillSubdir("my-skill", "references", "references", nil, t.TempDir(), out)
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
	err := CompileSkillSubdir("my-skill", "scripts", "scripts", []string{"scripts/*.sh"}, tmpDir, out)
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
	err := CompileSkillSubdir("my-skill", "assets", "assets", []string{"nonexistent.png"}, t.TempDir(), out)
	if err == nil {
		t.Error("expected error for missing literal file, got nil")
	}
}

func TestCompileSkillSubdir_OutputNameTranslation(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "examples"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, "examples", "sample.md"), []byte("# Sample"), 0o644)

	out := &output.Output{Files: make(map[string]string)}
	err := CompileSkillSubdir("my-skill", "examples", "resources", []string{"examples/sample.md"}, tmpDir, out)
	if err != nil {
		t.Fatal(err)
	}

	expected := "skills/my-skill/resources/sample.md"
	if _, ok := out.Files[expected]; !ok {
		keys := make([]string, 0, len(out.Files))
		for k := range out.Files {
			keys = append(keys, k)
		}
		t.Errorf("expected output at %q, got keys: %v", expected, keys)
	}
}

func TestSortedKeys_ReturnsSorted(t *testing.T) {
	m := map[string]int{"charlie": 3, "alpha": 1, "bravo": 2}
	got := SortedKeys(m)
	want := []string{"alpha", "bravo", "charlie"}
	if len(got) != len(want) {
		t.Fatalf("expected %d keys, got %d", len(want), len(got))
	}
	for i, k := range got {
		if k != want[i] {
			t.Errorf("index %d: got %q want %q", i, k, want[i])
		}
	}
}

func TestSortedKeys_EmptyMap(t *testing.T) {
	got := SortedKeys(map[string]string{})
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}

func TestSortedKeys_CustomStringType(t *testing.T) {
	type EventName string
	m := map[EventName]int{"stop": 1, "start": 2, "pause": 3}
	got := SortedKeys(m)
	if len(got) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(got))
	}
	if got[0] != "pause" || got[1] != "start" || got[2] != "stop" {
		t.Errorf("unexpected order: %v", got)
	}
}

func TestYAMLScalar_PlainString(t *testing.T) {
	got := YAMLScalar("hello")
	if got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestYAMLScalar_QuotesSpecialChars(t *testing.T) {
	got := YAMLScalar("value: with colon")
	if got == "" {
		t.Error("expected non-empty YAML scalar output")
	}
	if !strings.HasPrefix(got, `"`) {
		t.Errorf("expected quoted output for colon-containing string, got %q", got)
	}
}

func TestYAMLScalar_QuotesHashChar(t *testing.T) {
	got := YAMLScalar("key#val")
	if !strings.HasPrefix(got, `"`) {
		t.Errorf("expected quoted output for hash char, got %q", got)
	}
}

func TestYAMLScalar_PlainURL(t *testing.T) {
	// URLs with slashes are fine unquoted; colons trigger quoting
	got := YAMLScalar("https://example.com")
	if !strings.HasPrefix(got, `"`) {
		t.Errorf("expected quoted output for URL with colon, got %q", got)
	}
}

func TestStripAllFrontmatter_SingleBlock(t *testing.T) {
	input := "---\nkey: val\n---\n# Body"
	got := StripAllFrontmatter(input)
	if strings.Contains(got, "key: val") {
		t.Errorf("expected frontmatter stripped, got: %q", got)
	}
	if !strings.Contains(got, "# Body") {
		t.Errorf("expected body preserved, got: %q", got)
	}
}

func TestStripAllFrontmatter_DoubleBlock(t *testing.T) {
	input := "---\n---\n\n---\n---\n\n# Body"
	got := StripAllFrontmatter(input)
	if strings.HasPrefix(strings.TrimSpace(got), "---") {
		t.Errorf("expected all frontmatter blocks stripped, got: %q", got)
	}
	if !strings.Contains(got, "# Body") {
		t.Errorf("expected body preserved, got: %q", got)
	}
}

func TestStripAllFrontmatter_NoFrontmatter(t *testing.T) {
	input := "# Just a heading\n\nSome content."
	got := StripAllFrontmatter(input)
	if got != input {
		t.Errorf("expected unchanged content, got: %q", got)
	}
}
