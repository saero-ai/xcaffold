package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// writePolicyTestFile writes content to path, creating parent directories.
func writePolicyTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

// buildXcaffoldBinary compiles the xcaffold binary to a temp path and returns it.
func buildXcaffoldBinary(t *testing.T) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "xcaffold")
	cmd := exec.Command("go", "build", "-o", out, "github.com/saero-ai/xcaffold/cmd/xcaffold")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build xcaffold binary: %v\n%s", err, output)
	}
	return out
}

// TestApply_WithErrorPolicy_BlocksWrite verifies a policy violation with severity:error
// prevents compilation output from being written.
func TestApply_WithErrorPolicy_BlocksWrite(t *testing.T) {
	dir := t.TempDir()

	// scaffold.xcf: agent with no description (will trigger policy)
	writePolicyTestFile(t, filepath.Join(dir, "scaffold.xcf"), `kind: config
version: "1.0"
project:
  name: test-apply-policy
agents:
  no-description-agent:
    instructions: "You are a test agent."
`)

	// policies/require-description.xcf: error-level policy requiring descriptions
	writePolicyTestFile(t, filepath.Join(dir, "policies", "require-description.xcf"), `kind: policy
name: require-description
description: Agents must have descriptions
severity: error
target: agent
require:
  - field: description
    is_present: true
`)

	xcaffold := buildXcaffoldBinary(t)
	cmd := exec.Command(xcaffold, "apply")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()

	if err == nil {
		t.Errorf("expected apply to fail due to policy violation, but it succeeded.\nOutput: %s", out)
	}

	// .claude/ directory must not be created (apply was blocked)
	if _, statErr := os.Stat(filepath.Join(dir, ".claude")); statErr == nil {
		t.Error(".claude/ was created despite policy violation blocking apply")
	}
}

// TestApply_WithWarningPolicy_AllowsWrite verifies severity:warning does not block apply.
func TestApply_WithWarningPolicy_AllowsWrite(t *testing.T) {
	dir := t.TempDir()

	writePolicyTestFile(t, filepath.Join(dir, "scaffold.xcf"), `kind: config
version: "1.0"
project:
  name: test-apply-warning
agents:
  no-description-agent:
    instructions: "You are a test agent."
`)

	writePolicyTestFile(t, filepath.Join(dir, "policies", "warn-description.xcf"), `kind: policy
name: warn-description
description: Agents should have descriptions
severity: warning
target: agent
require:
  - field: description
    is_present: true
`)

	xcaffold := buildXcaffoldBinary(t)
	cmd := exec.Command(xcaffold, "apply", "--force")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()

	if err != nil {
		t.Errorf("expected apply to succeed with warning policy: %v\nOutput: %s", err, out)
	}
}

// TestValidate_ReportsViolationsExitCode1 verifies validate exits 1 on policy errors.
func TestValidate_ReportsViolationsExitCode1(t *testing.T) {
	dir := t.TempDir()

	writePolicyTestFile(t, filepath.Join(dir, "scaffold.xcf"), `kind: config
version: "1.0"
project:
  name: test-validate-fail
agents:
  no-description-agent:
    instructions: "You are a test agent."
`)

	writePolicyTestFile(t, filepath.Join(dir, "policies", "require-description.xcf"), `kind: policy
name: require-description
description: Agents must have descriptions
severity: error
target: agent
require:
  - field: description
    is_present: true
`)

	xcaffold := buildXcaffoldBinary(t)
	cmd := exec.Command(xcaffold, "validate")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "XCAFFOLD_SKIP_GLOBAL=true")
	_, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("expected validate to exit 1 due to policy violation, got exit 0")
	}
}

// TestValidate_NoPolicies_Passes verifies validate exits 0 when no policies and syntax is valid.
func TestValidate_NoPolicies_Passes(t *testing.T) {
	dir := t.TempDir()
	writePolicyTestFile(t, filepath.Join(dir, "scaffold.xcf"), `kind: config
version: "1.0"
project:
  name: test-validate-pass
agents:
  dev:
    instructions: "You are a developer."
    description: "Development agent with full context."
`)

	xcaffold := buildXcaffoldBinary(t)
	cmd := exec.Command(xcaffold, "validate")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "XCAFFOLD_SKIP_GLOBAL=true")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("expected validate to pass, got error: %v\nOutput: %s", err, out)
	}
}
