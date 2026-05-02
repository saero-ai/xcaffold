package importer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/importer"
	_ "github.com/saero-ai/xcaffold/internal/importer/claude"
)

func TestScanDir_Claude_CountsAllKinds(t *testing.T) {
	dir := t.TempDir()

	paths := map[string]string{
		"agents/dev.md":         "---\nname: dev\n---\n",
		"agents/reviewer.md":    "---\nname: reviewer\n---\n",
		"skills/tdd/SKILL.md":   "---\nname: tdd\n---\n",
		"rules/security.md":     "# Security\n",
		"workflows/release.md":  "---\nname: release\n---\n",
		"hooks/pre-commit.sh":   "#!/bin/sh\n",
		"settings.json":         `{"mcpServers":{}}`,
		"agent-memory/dev/m.md": "# Memory\n",
	}
	for rel, content := range paths {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	importers := importer.DefaultImporters()
	var claudeImp importer.ProviderImporter
	for _, imp := range importers {
		if imp.Provider() == "claude" {
			claudeImp = imp
			break
		}
	}
	if claudeImp == nil {
		t.Fatal("claude importer not registered")
	}

	counts := importer.ScanDir(claudeImp, dir)

	if counts[importer.KindAgent] != 2 {
		t.Errorf("agents: got %d, want 2", counts[importer.KindAgent])
	}
	if counts[importer.KindSkill] != 1 {
		t.Errorf("skills: got %d, want 1", counts[importer.KindSkill])
	}
	if counts[importer.KindRule] != 1 {
		t.Errorf("rules: got %d, want 1", counts[importer.KindRule])
	}
	if counts[importer.KindWorkflow] != 1 {
		t.Errorf("workflows: got %d, want 1", counts[importer.KindWorkflow])
	}
	if counts[importer.KindHookScript] < 1 {
		t.Errorf("hooks: got %d, want >= 1", counts[importer.KindHookScript])
	}
	if counts[importer.KindSettings] != 1 {
		t.Errorf("settings: got %d, want 1", counts[importer.KindSettings])
	}
	if counts[importer.KindMemory] != 1 {
		t.Errorf("memory: got %d, want 1", counts[importer.KindMemory])
	}
}

func TestScanDir_EmptyDir_ReturnsEmptyMap(t *testing.T) {
	dir := t.TempDir()

	importers := importer.DefaultImporters()
	var claudeImp importer.ProviderImporter
	for _, imp := range importers {
		if imp.Provider() == "claude" {
			claudeImp = imp
			break
		}
	}
	if claudeImp == nil {
		t.Fatal("claude importer not registered")
	}

	counts := importer.ScanDir(claudeImp, dir)
	if len(counts) != 0 {
		t.Errorf("expected empty map, got %v", counts)
	}
}
