package analyzer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
)

func TestAnalyzeTokens_AgentsInline(t *testing.T) {
	a := New()
	config := &ast.XcaffoldConfig{
		Agents: map[string]ast.AgentConfig{
			"agent-one": {
				Instructions: "This is a test instruction.",
				Description:  "This is a test description.",
			},
		},
	}

	report := a.AnalyzeTokens(config, "")

	if len(report.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(report.Entries))
	}

	entry := report.Entries[0]
	if entry.Kind != "agent" || entry.ID != "agent-one" {
		t.Errorf("unexpected entry kind/id: %s/%s", entry.Kind, entry.ID)
	}
	if entry.Tokens == 0 {
		t.Errorf("expected >0 tokens, got %d", entry.Tokens)
	}
	if len(report.Warnings) > 0 {
		t.Errorf("unexpected warnings: %v", report.Warnings)
	}
}

func TestAnalyzeTokens_AgentsFromFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "instructions.md")
	_ = os.WriteFile(filePath, []byte("File content here. More words to count."), 0600)

	a := New()
	config := &ast.XcaffoldConfig{
		Agents: map[string]ast.AgentConfig{
			"agent-file": {
				InstructionsFile: "instructions.md",
			},
		},
	}

	report := a.AnalyzeTokens(config, dir)

	if len(report.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(report.Entries))
	}
	if report.Entries[0].Tokens == 0 {
		t.Errorf("expected >0 tokens for file instruction")
	}
	if len(report.Warnings) > 0 {
		t.Errorf("unexpected warnings: %v", report.Warnings)
	}
}

func TestAnalyzeTokens_MissingFile_Warns(t *testing.T) {
	a := New()
	config := &ast.XcaffoldConfig{
		Agents: map[string]ast.AgentConfig{
			"missing-agent": {
				InstructionsFile: "does-not-exist.md",
			},
		},
	}

	report := a.AnalyzeTokens(config, "/tmp/somewhere")

	if len(report.Entries) != 1 {
		t.Fatalf("expected 1 entry (degraded to empty string)")
	}
	if len(report.Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(report.Warnings))
	}
}

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
