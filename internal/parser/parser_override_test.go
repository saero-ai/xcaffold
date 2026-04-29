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

func TestParse_OverrideFile_StoresInOverrides(t *testing.T) {
	dir := t.TempDir()
	xcfDir := filepath.Join(dir, "xcf", "agents", "developer")
	os.MkdirAll(xcfDir, 0755)

	// Base agent
	base := "---\nkind: agent\nversion: \"1.0\"\nname: developer\nmodel: sonnet\n---\nUniversal instructions.\n"
	os.WriteFile(filepath.Join(xcfDir, "agent.xcf"), []byte(base), 0644)

	// Claude override (partial — no kind/version/name)
	override := "---\nmodel: opus\ntools:\n  - Bash\n  - Read\n---\nClaude-specific instructions.\n"
	os.WriteFile(filepath.Join(xcfDir, "agent.claude.xcf"), []byte(override), 0644)

	cfg, err := ParseDirectory(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Base agent should be in Agents map
	agent, ok := cfg.Agents["developer"]
	if !ok {
		t.Fatal("expected base agent 'developer'")
	}
	if agent.Model != "sonnet" {
		t.Fatalf("expected base model sonnet, got %s", agent.Model)
	}

	// Override should be in Overrides map
	if cfg.Overrides == nil {
		t.Fatal("expected Overrides to be initialized")
	}
	overrideCfg, ok := cfg.Overrides.GetAgent("developer", "claude")
	if !ok {
		t.Fatal("expected claude override for 'developer'")
	}
	if overrideCfg.Model != "opus" {
		t.Fatalf("expected override model opus, got %s", overrideCfg.Model)
	}
	if len(overrideCfg.Tools) != 2 {
		t.Fatalf("expected 2 override tools, got %d", len(overrideCfg.Tools))
	}
	if overrideCfg.Body != "Claude-specific instructions." {
		t.Fatalf("expected override body, got %q", overrideCfg.Body)
	}
}

func TestParse_OverrideFile_RequiresBaseFile(t *testing.T) {
	dir := t.TempDir()
	xcfDir := filepath.Join(dir, "xcf", "agents", "developer")
	os.MkdirAll(xcfDir, 0755)

	// Override WITHOUT base
	override := "---\nmodel: opus\n---\n"
	os.WriteFile(filepath.Join(xcfDir, "agent.claude.xcf"), []byte(override), 0644)

	_, err := ParseDirectory(dir)
	if err == nil {
		t.Fatal("expected error when override file has no base")
	}
}
