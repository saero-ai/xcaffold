package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidate_WorkflowID_PathTraversal verifies that a workflow ID containing
// ".." is rejected at parse time.
func TestValidate_WorkflowID_PathTraversal(t *testing.T) {
	input := `
version: "1.0"
project:
  name: "test-project"
workflows:
  "../escape":
    instructions: "bad"
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err, "workflow ID with path traversal must be rejected")
	assert.Contains(t, err.Error(), "workflow id contains invalid characters")
}

// TestValidate_WorkflowID_ForwardSlash verifies that forward slashes in workflow
// IDs are rejected.
func TestValidate_WorkflowID_ForwardSlash(t *testing.T) {
	input := `
version: "1.0"
project:
  name: "test-project"
workflows:
  "evil/path":
    instructions: "bad"
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err, "workflow ID with forward slash must be rejected")
	assert.Contains(t, err.Error(), "workflow id contains invalid characters")
}

// TestValidate_WorkflowID_Backslash verifies that backslashes in workflow IDs
// are rejected.
func TestValidate_WorkflowID_Backslash(t *testing.T) {
	input := `
version: "1.0"
project:
  name: "test-project"
workflows:
  "evil\\path":
    instructions: "bad"
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err, "workflow ID with backslash must be rejected")
	assert.Contains(t, err.Error(), "workflow id contains invalid characters")
}

// TestValidate_Workflow_MutualExclusivity verifies that setting both instructions
// and instructions_file on the same workflow is a parse error.
func TestValidate_Workflow_MutualExclusivity(t *testing.T) {
	input := `
version: "1.0"
project:
  name: "test-project"
workflows:
  deploy:
    instructions: "Inline instructions."
    instructions_file: "workflows/deploy.md"
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err, "both instructions and instructions_file set must be rejected")
	assert.Contains(t, err.Error(), "mutually exclusive")
}

// TestValidate_Workflow_InstructionsFile_AbsolutePath_Rejected verifies that an
// absolute path in a workflow's instructions_file is rejected.
func TestValidate_Workflow_InstructionsFile_AbsolutePath_Rejected(t *testing.T) {
	input := `
version: "1.0"
project:
  name: "test-project"
workflows:
  release:
    instructions_file: "/etc/passwd"
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err, "absolute instructions_file path must be rejected")
	assert.Contains(t, err.Error(), "relative path")
}

// TestValidate_Workflow_InstructionsFile_PathTraversal_Rejected verifies that
// path traversal in a workflow's instructions_file is rejected.
func TestValidate_Workflow_InstructionsFile_PathTraversal_Rejected(t *testing.T) {
	input := `
version: "1.0"
project:
  name: "test-project"
workflows:
  release:
    instructions_file: "../outside/release.md"
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err, "instructions_file with path traversal must be rejected")
	assert.Contains(t, err.Error(), "instructions_file")
}

// TestValidate_Workflow_ValidRelativePath_Accepted verifies that a valid relative
// instructions_file path on a workflow parses successfully.
func TestValidate_Workflow_ValidRelativePath_Accepted(t *testing.T) {
	input := `
version: "1.0"
project:
  name: "test-project"
workflows:
  release:
    description: "Release workflow"
    instructions_file: "workflows/release.md"
`
	cfg, err := Parse(strings.NewReader(input))
	require.NoError(t, err, "valid relative instructions_file should be accepted")
	wf, ok := cfg.Workflows["release"]
	require.True(t, ok)
	assert.Equal(t, "workflows/release.md", wf.InstructionsFile)
}

// TestValidate_Workflow_ValidInlineInstructions_Accepted verifies that a workflow
// with only inline instructions parses successfully.
func TestValidate_Workflow_ValidInlineInstructions_Accepted(t *testing.T) {
	input := `
version: "1.0"
project:
  name: "test-project"
workflows:
  build:
    description: "Build workflow"
    instructions: "Run go build ./..."
`
	cfg, err := Parse(strings.NewReader(input))
	require.NoError(t, err, "workflow with inline instructions should be accepted")
	wf, ok := cfg.Workflows["build"]
	require.True(t, ok)
	assert.Equal(t, "Run go build ./...", wf.Instructions)
}

// TestMerge_Workflows_ChildAddsNew verifies that a child config's workflows are
// present in the merged result when the base has none.
func TestMerge_Workflows_ChildAddsNew(t *testing.T) {
	base := writeTemp(t, "base.xcf", `
version: "1.0"
project:
  name: "base-project"
`)
	child := writeTemp(t, "child.xcf", fmt.Sprintf(`
extends: %q
version: "1.0"
project:
  name: "child-project"
workflows:
  deploy:
    instructions: "Deploy to production."
`, base))

	cfg, err := ParseFile(child)
	require.NoError(t, err)
	require.Contains(t, cfg.Workflows, "deploy", "child workflow must be present in merged config")
}

// TestMerge_Workflows_BasePreserved verifies that base workflows survive when the
// child introduces a different workflow ID.
func TestMerge_Workflows_BasePreserved(t *testing.T) {
	base := writeTemp(t, "base.xcf", `
version: "1.0"
project:
  name: "base-project"
workflows:
  build:
    instructions: "Run go build."
`)
	child := writeTemp(t, "child.xcf", fmt.Sprintf(`
extends: %q
version: "1.0"
project:
  name: "child-project"
workflows:
  deploy:
    instructions: "Deploy to production."
`, base))

	cfg, err := ParseFile(child)
	require.NoError(t, err)
	assert.Contains(t, cfg.Workflows, "build", "base workflow must be inherited")
	assert.Contains(t, cfg.Workflows, "deploy", "child workflow must be present")
}

// TestMerge_Workflows_ChildOverridesBase verifies that when base and child define
// the same workflow ID, the child's definition wins.
func TestMerge_Workflows_ChildOverridesBase(t *testing.T) {
	base := writeTemp(t, "base.xcf", `
version: "1.0"
project:
  name: "base-project"
workflows:
  deploy:
    instructions: "Base deploy instructions."
`)
	child := writeTemp(t, "child.xcf", fmt.Sprintf(`
extends: %q
version: "1.0"
project:
  name: "child-project"
workflows:
  deploy:
    instructions: "Child deploy instructions."
`, base))

	cfg, err := ParseFile(child)
	require.NoError(t, err)
	wf, ok := cfg.Workflows["deploy"]
	require.True(t, ok)
	assert.Equal(t, "Child deploy instructions.", wf.Instructions, "child definition must override base")
}

// writeTemp creates a temporary file with the given name and content inside a
// t.TempDir() and returns the absolute path.
func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}
