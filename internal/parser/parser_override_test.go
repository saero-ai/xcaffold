package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse_OverrideSuffix_DetectsProvider(t *testing.T) {
	tests := []struct {
		filename string
		wantKind string
		wantProv string
		wantOK   bool
	}{
		{"agent.claude.xcf", "agent", "claude", true},
		{"skill.gemini.xcf", "skill", "gemini", true},
		{"rule.cursor.xcf", "rule", "cursor", true},
		{"workflow.copilot.xcf", "workflow", "copilot", true},
		{"mcp.antigravity.xcf", "mcp", "antigravity", true},
		{"agent.xcf", "", "", false},
		{"custom-name.xcf", "", "", false},
	}
	for _, tt := range tests {
		kind, prov, ok := classifyOverrideFile(tt.filename)
		if ok != tt.wantOK {
			t.Errorf("classifyOverrideFile(%q): ok=%v, want %v", tt.filename, ok, tt.wantOK)
		}
		if kind != tt.wantKind || prov != tt.wantProv {
			t.Errorf("classifyOverrideFile(%q): got (%q,%q), want (%q,%q)",
				tt.filename, kind, prov, tt.wantKind, tt.wantProv)
		}
	}
}

func TestParse_OverrideSuffix_RejectsUnknownProvider(t *testing.T) {
	dir := t.TempDir()
	xcfDir := filepath.Join(dir, "xcf", "agents", "developer")
	os.MkdirAll(xcfDir, 0755)

	// Write valid base file
	os.WriteFile(filepath.Join(xcfDir, "agent.xcf"), []byte("kind: agent\nversion: \"1.0\"\nname: developer\n"), 0644)
	// Write override with unknown provider
	os.WriteFile(filepath.Join(xcfDir, "agent.foobar.xcf"), []byte("model: opus\n"), 0644)

	_, err := ParseDirectory(dir)
	if err == nil {
		t.Fatal("expected error for unknown provider token 'foobar'")
	}
}
