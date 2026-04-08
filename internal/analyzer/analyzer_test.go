package analyzer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanOutputDir(t *testing.T) {
	dir := t.TempDir()

	// Create some fake artifacts
	_ = os.MkdirAll(filepath.Join(dir, "agents"), 0755)
	_ = os.MkdirAll(filepath.Join(dir, "skills", "fake-skill"), 0755)

	_ = os.WriteFile(filepath.Join(dir, "agents", "undeclared-agent.md"), []byte("agent content"), 0600)
	_ = os.WriteFile(filepath.Join(dir, "skills", "fake-skill", "SKILL.md"), []byte("skill content"), 0600)
	_ = os.WriteFile(filepath.Join(dir, "agents", "declared-agent.md"), []byte("agent content"), 0600)

	declared := map[string]bool{
		"agent:declared-agent": true,
	}

	a := New()
	entries, err := a.ScanOutputDir(dir, declared)
	if err != nil {
		t.Fatalf("ScanOutputDir failed: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 undeclared entries, got %d", len(entries))
	}

	// Should be sorted
	if entries[0].Kind != "agent" || entries[0].ID != "undeclared-agent" {
		t.Errorf("expected undeclared-agent, got %v", entries[0])
	}
	if entries[1].Kind != "skill" || entries[1].ID != "fake-skill" {
		t.Errorf("expected fake-skill, got %v", entries[1])
	}
}
