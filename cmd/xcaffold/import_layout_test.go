package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectLayout_Flat(t *testing.T) {
	dir := t.TempDir()
	xcafDir := filepath.Join(dir, "xcaf")
	rulesDir := filepath.Join(xcafDir, "rules")
	os.MkdirAll(rulesDir, 0755)
	os.WriteFile(filepath.Join(rulesDir, "security.xcaf"), []byte(""), 0644)
	os.WriteFile(filepath.Join(rulesDir, "logging.xcaf"), []byte(""), 0644)

	got := detectLayout(xcafDir, "rule")
	if got != layoutFlat {
		t.Errorf("detectLayout = %v, want layoutFlat", got)
	}
}

func TestDetectLayout_Nested(t *testing.T) {
	dir := t.TempDir()
	xcafDir := filepath.Join(dir, "xcaf")
	rulesDir := filepath.Join(xcafDir, "rules")
	securityDir := filepath.Join(rulesDir, "security")
	os.MkdirAll(securityDir, 0755)
	os.WriteFile(filepath.Join(securityDir, "rule.xcaf"), []byte(""), 0644)

	got := detectLayout(xcafDir, "rule")
	if got != layoutNested {
		t.Errorf("detectLayout = %v, want layoutNested", got)
	}
}

func TestDetectLayout_Empty(t *testing.T) {
	dir := t.TempDir()
	xcafDir := filepath.Join(dir, "xcaf")
	// Don't create the directory at all

	got := detectLayout(xcafDir, "rule")
	if got != layoutNested {
		t.Errorf("detectLayout = %v, want layoutNested (default)", got)
	}
}

func TestDetectLayout_MajorityVote(t *testing.T) {
	dir := t.TempDir()
	xcafDir := filepath.Join(dir, "xcaf")
	rulesDir := filepath.Join(xcafDir, "rules")
	os.MkdirAll(rulesDir, 0755)

	// Create 3 flat files
	os.WriteFile(filepath.Join(rulesDir, "rule1.xcaf"), []byte(""), 0644)
	os.WriteFile(filepath.Join(rulesDir, "rule2.xcaf"), []byte(""), 0644)
	os.WriteFile(filepath.Join(rulesDir, "rule3.xcaf"), []byte(""), 0644)

	// Create 1 nested subdir
	os.MkdirAll(filepath.Join(rulesDir, "nested"), 0755)

	got := detectLayout(xcafDir, "rule")
	if got != layoutFlat {
		t.Errorf("detectLayout = %v, want layoutFlat (majority 3 flat vs 1 nested)", got)
	}
}

func TestKindToPlural(t *testing.T) {
	tests := []struct {
		kind string
		want string
	}{
		{"rule", "rules"},
		{"agent", "agents"},
		{"skill", "skills"},
		{"workflow", "workflows"},
		{"mcp", "mcp"},
		{"context", "contexts"},
		{"unknown", "unknowns"},
	}
	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			got := kindToPlural(tt.kind)
			if got != tt.want {
				t.Errorf("kindToPlural(%q) = %q, want %q", tt.kind, got, tt.want)
			}
		})
	}
}
