package renderer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRewriteHookCommandPath(t *testing.T) {
	tests := []struct {
		name    string
		command string
		srcBase string
		dstBase string
		want    string
	}{
		{
			name:    "script path rewritten",
			command: "bash xcf/hooks/enforce/scripts/enforce-standards.sh",
			srcBase: "xcf/hooks/enforce/scripts/",
			dstBase: ".claude/hooks/",
			want:    "bash .claude/hooks/enforce-standards.sh",
		},
		{
			name:    "non-matching command unchanged",
			command: "echo hello",
			srcBase: "xcf/hooks/enforce/scripts/",
			dstBase: ".claude/hooks/",
			want:    "echo hello",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RewriteHookCommandPath(tt.command, tt.srcBase, tt.dstBase)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCompileHookArtifacts(t *testing.T) {
	tmp := t.TempDir()
	scriptsDir := filepath.Join(tmp, "scripts")
	os.MkdirAll(scriptsDir, 0755)
	os.WriteFile(filepath.Join(scriptsDir, "run.sh"), []byte("#!/bin/bash\necho hi"), 0644)

	files, err := CompileHookArtifacts("test-hook", []string{"scripts"}, tmp, ".claude/hooks")
	if err != nil {
		t.Fatalf("CompileHookArtifacts error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	content, ok := files[".claude/hooks/run.sh"]
	if !ok {
		t.Fatal("expected .claude/hooks/run.sh in output")
	}
	if content != "#!/bin/bash\necho hi" {
		t.Errorf("content = %q, want script content", content)
	}
}

func TestCompileHookArtifacts_MissingDir(t *testing.T) {
	tmp := t.TempDir()
	files, err := CompileHookArtifacts("test", []string{"nonexistent"}, tmp, ".out")
	if err != nil {
		t.Fatalf("expected no error for missing dir, got: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files for missing dir, got %d", len(files))
	}
}
