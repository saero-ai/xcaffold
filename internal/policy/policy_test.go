package policy_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/policy"
)

// TestScanDir_FindsPolicyFiles verifies ScanDir returns only kind: policy files.
func TestScanDir_FindsPolicyFiles(t *testing.T) {
	dir := t.TempDir()

	// Write a valid policy file
	writeFile(t, filepath.Join(dir, "a.xcf"), "kind: policy\nname: a\ndescription: d\nseverity: warning\ntarget: agent\n")
	// Write a config file (should be excluded)
	writeFile(t, filepath.Join(dir, "b.xcf"), "kind: config\nversion: \"1.0\"\n")
	// Write a registry file (should be excluded)
	writeFile(t, filepath.Join(dir, "c.xcf"), "kind: registry\n")
	// Write a file with no kind (backward-compat config, should be excluded)
	writeFile(t, filepath.Join(dir, "d.xcf"), "version: \"1.0\"\nagents: {}\n")

	files, err := policy.ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 policy file, got %d: %v", len(files), files)
	}
	if filepath.Base(files[0]) != "a.xcf" {
		t.Errorf("expected a.xcf, got %s", filepath.Base(files[0]))
	}
}

// TestScanDir_EmptyDir returns no files and no error for an empty directory.
func TestScanDir_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	files, err := policy.ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

// TestScanDir_NonExistentDir returns an error for a missing directory.
func TestScanDir_NonExistentDir(t *testing.T) {
	_, err := policy.ScanDir("/tmp/xcaffold-nonexistent-scanner-test-dir")
	if err == nil {
		t.Error("expected error for nonexistent dir, got nil")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

// TestParseFile_ValidPolicy parses a well-formed policy file.
func TestParseFile_ValidPolicy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "valid.xcf")
	writeFile(t, path, `kind: policy
name: test-agent-has-description
description: Test policy that requires agent descriptions
severity: warning
target: agent
require:
  - field: description
    is_present: true
`)
	cfg, err := policy.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if cfg.Kind != "policy" || cfg.Name != "test-agent-has-description" {
		t.Errorf("Unexpected header values: %+v", cfg)
	}
	if len(cfg.Require) != 1 || cfg.Require[0].Field != "description" {
		t.Errorf("Require block not parsed correctly")
	}
}

// TestParseFile_UnknownField fails when the YAML contains undeclared fields.
func TestParseFile_UnknownField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "unknown.xcf")
	writeFile(t, path, `kind: policy
name: ""
description: ""
severity: bad-value
target: agent
unknown_field: will_fail_strict_parse
`)
	_, err := policy.ParseFile(path)
	if err == nil {
		t.Error("expected parser error for unknown_field, got nil")
	}
}

// TestParseFile_InvalidYAML fails on bad syntax.
func TestParseFile_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.xcf")
	writeFile(t, path, `kind: policy\ntarget: [missing bracket`)
	_, err := policy.ParseFile(path)
	if err == nil {
		t.Error("expected parser error for bad YAML, got nil")
	}
}

// TestMatch_HasTool_Matches returns true when agent has the specified tool.
func TestMatch_HasTool_Matches(t *testing.T) {
	m := policy.PolicyMatch{HasTool: "Bash"}
	agent := map[string]any{"instructions": "x", "tools": []string{"Bash", "Read"}}
	if !policy.MatchAgent(m, agent) {
		t.Error("expected match for agent with Bash tool")
	}
}

// TestMatch_HasTool_NoMatch returns false when agent lacks the specified tool.
func TestMatch_HasTool_NoMatch(t *testing.T) {
	m := policy.PolicyMatch{HasTool: "Bash"}
	agent := map[string]any{"instructions": "x", "tools": []string{"Read"}}
	if policy.MatchAgent(m, agent) {
		t.Error("expected no match for agent without Bash")
	}
}

// TestMatch_EmptyMatch matches all agents when no conditions are set.
func TestMatch_EmptyMatch(t *testing.T) {
	m := policy.PolicyMatch{}
	agent := map[string]any{"instructions": "x"}
	if !policy.MatchAgent(m, agent) {
		t.Error("empty match should match all agents")
	}
}

// TestMatch_NameMatches_Glob tests glob matching on resource names.
func TestMatch_NameMatches_Glob(t *testing.T) {
	m := policy.PolicyMatch{NameMatches: "backend-*"}
	if !policy.MatchName(m, "backend-dev") {
		t.Error("expected match for backend-dev against backend-*")
	}
	if policy.MatchName(m, "frontend-dev") {
		t.Error("expected no match for frontend-dev against backend-*")
	}
}

// TestRequire_IsPresent_Empty_Violation reports a violation when field is empty.
func TestRequire_IsPresent_Empty_Violation(t *testing.T) {
	present := true
	req := policy.PolicyRequire{Field: "description", IsPresent: &present}
	viols := policy.EvalRequire("agent", "my-agent", req, map[string]string{"description": ""})
	if len(viols) == 0 {
		t.Error("expected violation for empty description with is_present: true")
	}
}

// TestRequire_MinLength_Short_Violation reports violation when string is too short.
func TestRequire_MinLength_Short_Violation(t *testing.T) {
	minLen := 50
	req := policy.PolicyRequire{Field: "description", MinLength: &minLen}
	viols := policy.EvalRequire("agent", "my-agent", req, map[string]string{"description": "too short"})
	if len(viols) == 0 {
		t.Error("expected violation for description shorter than 50 chars")
	}
}

// TestRequire_OneOf_InvalidValue_Violation reports violation when value not in list.
func TestRequire_OneOf_InvalidValue_Violation(t *testing.T) {
	req := policy.PolicyRequire{Field: "model", OneOf: []string{"claude-sonnet-4-5-20250514", "claude-haiku-4-5-20251001"}}
	viols := policy.EvalRequire("agent", "my-agent", req, map[string]string{"model": "gpt-4"})
	if len(viols) == 0 {
		t.Error("expected violation for model not in approved list")
	}
}

// TestDeny_ContentContains_Found_Violation reports violation for denied string in content.
func TestDeny_ContentContains_Found_Violation(t *testing.T) {
	deny := policy.PolicyDeny{ContentContains: []string{"TODO", "FIXME"}}
	files := map[string]string{"agents/dev.md": "# Dev Agent\nTODO: add instructions"}
	viols := policy.EvalDeny("test-policy", policy.SeverityError, deny, files)
	if len(viols) == 0 {
		t.Error("expected violation for TODO in content")
	}
}

// TestDeny_PathContains_Traversal_Violation reports violation for .. in output path.
func TestDeny_PathContains_Traversal_Violation(t *testing.T) {
	deny := policy.PolicyDeny{PathContains: ".."}
	files := map[string]string{"../escape.md": "content"}
	viols := policy.EvalDeny("path-safety", policy.SeverityError, deny, files)
	if len(viols) == 0 {
		t.Error("expected violation for .. in path")
	}
}
