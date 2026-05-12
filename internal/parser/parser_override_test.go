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
		{"agent.claude.xcaf", "agent", "claude", true},
		{"skill.gemini.xcaf", "skill", "gemini", true},
		{"rule.cursor.xcaf", "rule", "cursor", true},
		{"workflow.copilot.xcaf", "workflow", "copilot", true},
		{"mcp.antigravity.xcaf", "mcp", "antigravity", true},
		{"agent.xcaf", "", "", false},
		{"custom-name.xcaf", "", "", false},
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
	xcafDir := filepath.Join(dir, "xcaf", "agents", "developer")
	os.MkdirAll(xcafDir, 0755)

	// Write valid base file
	os.WriteFile(filepath.Join(xcafDir, "agent.xcaf"), []byte("kind: agent\nversion: \"1.0\"\nname: developer\n"), 0644)
	// Write override with unknown provider
	os.WriteFile(filepath.Join(xcafDir, "agent.foobar.xcaf"), []byte("model: opus\n"), 0644)

	_, err := ParseDirectory(dir)
	if err == nil {
		t.Fatal("expected error for unknown provider token 'foobar'")
	}
}

func TestParse_OverrideFile_StoresInOverrides(t *testing.T) {
	dir := t.TempDir()
	xcafDir := filepath.Join(dir, "xcaf", "agents", "developer")
	os.MkdirAll(xcafDir, 0755)

	// Base agent
	base := "---\nkind: agent\nversion: \"1.0\"\nname: developer\nmodel: sonnet\n---\nUniversal instructions.\n"
	os.WriteFile(filepath.Join(xcafDir, "agent.xcaf"), []byte(base), 0644)

	// Claude override (partial — no kind/version/name)
	override := "---\nmodel: opus\ntools:\n  - Bash\n  - Read\n---\nClaude-specific instructions.\n"
	os.WriteFile(filepath.Join(xcafDir, "agent.claude.xcaf"), []byte(override), 0644)

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
	if len(overrideCfg.Tools.Values) != 2 {
		t.Fatalf("expected 2 override tools, got %d", len(overrideCfg.Tools.Values))
	}
	if overrideCfg.Body != "Claude-specific instructions." {
		t.Fatalf("expected override body, got %q", overrideCfg.Body)
	}
}

func TestParse_OverrideFile_RequiresBaseFile(t *testing.T) {
	dir := t.TempDir()
	xcafDir := filepath.Join(dir, "xcaf", "agents", "developer")
	os.MkdirAll(xcafDir, 0755)

	// Override WITHOUT base
	override := "---\nmodel: opus\n---\n"
	os.WriteFile(filepath.Join(xcafDir, "agent.claude.xcaf"), []byte(override), 0644)

	_, err := ParseDirectory(dir)
	if err == nil {
		t.Fatal("expected error when override file has no base")
	}
}

func TestParse_OverrideFile_StoresContextOverride(t *testing.T) {
	dir := t.TempDir()
	xcafDir := filepath.Join(dir, "xcaf", "contexts", "project-rules")
	os.MkdirAll(xcafDir, 0755)

	base := "---\nkind: context\nversion: \"1.0\"\nname: project-rules\ndescription: Project coding rules\n---\nUniversal context body.\n"
	os.WriteFile(filepath.Join(xcafDir, "context.xcaf"), []byte(base), 0644)

	override := "---\ndescription: Claude-specific rules\n---\nClaude-specific context body.\n"
	os.WriteFile(filepath.Join(xcafDir, "context.claude.xcaf"), []byte(override), 0644)

	config, err := ParseDirectory(dir)
	if err != nil {
		t.Fatalf("ParseDirectory failed: %v", err)
	}

	ctx, ok := config.Overrides.GetContext("project-rules", "claude")
	if !ok {
		t.Fatal("expected context override for 'project-rules' / 'claude' to be stored")
	}
	if ctx.Description != "Claude-specific rules" {
		t.Errorf("Description: want %q, got %q", "Claude-specific rules", ctx.Description)
	}
	if ctx.Body != "Claude-specific context body." {
		t.Errorf("Body: want %q, got %q", "Claude-specific context body.", ctx.Body)
	}
}
