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
	input := `kind: global
version: "1.0"
workflows:
  "../escape":

`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err, "workflow ID with path traversal must be rejected")
	assert.Contains(t, err.Error(), "workflow id contains invalid characters")
}

// TestValidate_WorkflowID_ForwardSlash verifies that forward slashes in workflow
// IDs are rejected.
func TestValidate_WorkflowID_ForwardSlash(t *testing.T) {
	input := `kind: global
version: "1.0"
workflows:
  "evil/path":

`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err, "workflow ID with forward slash must be rejected")
	assert.Contains(t, err.Error(), "workflow id contains invalid characters")
}

// TestValidate_WorkflowID_Backslash verifies that backslashes in workflow IDs
// are rejected.
func TestValidate_WorkflowID_Backslash(t *testing.T) {
	input := `kind: global
version: "1.0"
workflows:
  "evil\\path":

`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err, "workflow ID with backslash must be rejected")
	assert.Contains(t, err.Error(), "workflow id contains invalid characters")
}

// TestValidate_Workflow_MutualExclusivity verifies that setting both instructions
// and instructions-file on the same workflow is a parse error.
func TestValidate_Workflow_MutualExclusivity(t *testing.T) {
	t.Skip("Legacy instructions test removed")

	t.Skip("Legacy instructions test removed")

	input := `kind: global
version: "1.0"
workflows:
  deploy:


`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err, "both instructions and instructions-file set must be rejected")
	assert.Contains(t, err.Error(), "mutually exclusive")
}

// TestValidate_Workflow_InstructionsFile_AbsolutePath_Rejected verifies that an
// absolute path in a workflow's instructions-file is rejected.
func TestValidate_Workflow_InstructionsFile_AbsolutePath_Rejected(t *testing.T) {
	t.Skip("Legacy instructions test removed")

	t.Skip("Legacy instructions test removed")

	input := `kind: global
version: "1.0"
workflows:
  release:

`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err, "absolute instructions-file path must be rejected")
	assert.Contains(t, err.Error(), "relative path")
}

// TestValidate_Workflow_InstructionsFile_PathTraversal_Rejected verifies that
// path traversal in a workflow's instructions-file is rejected.
func TestValidate_Workflow_InstructionsFile_PathTraversal_Rejected(t *testing.T) {
	t.Skip("Legacy instructions test removed")

	t.Skip("Legacy instructions test removed")

	input := `kind: global
version: "1.0"
workflows:
  release:

`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err, "instructions-file with path traversal must be rejected")
	assert.Contains(t, err.Error(), "instructions-file")
}

// TestValidate_Workflow_ValidRelativePath_Accepted verifies that a valid relative
// instructions-file path on a workflow parses successfully.
func TestValidate_Workflow_ValidRelativePath_Accepted(t *testing.T) {
	t.Skip("Legacy instructions test removed")

	t.Skip("Legacy instructions test removed")

	input := `kind: global
version: "1.0"
workflows:
  release:
    description: "Release workflow"

`
	cfg, err := Parse(strings.NewReader(input))
	require.NoError(t, err, "valid relative instructions-file should be accepted")
	wf, ok := cfg.Workflows["release"]
	require.True(t, ok)
	assert.Equal(t, "workflows/release.md", wf.Description)
}

// TestValidate_Workflow_ValidInlineInstructions_Accepted verifies that a workflow
// with only inline instructions parses successfully.
func TestValidate_Workflow_ValidInlineInstructions_Accepted(t *testing.T) {
	input := `---
kind: workflow
version: "1.0"
name: build
---
Run go build ./...
`
	cfg, err := Parse(strings.NewReader(input))
	require.NoError(t, err, "workflow with inline instructions should be accepted")
	wf, ok := cfg.Workflows["build"]
	require.True(t, ok)
	assert.Equal(t, "Run go build ./...", wf.Body)
}

// TestMerge_Workflows_ChildAddsNew verifies that a child config's workflows are
// present in the merged result when the base has none.
func TestMerge_Workflows_ChildAddsNew(t *testing.T) {
	base := writeTemp(t, "base.xcf", `kind: project
version: "1.0"
name: "base-project"
`)
	child := writeTemp(t, "child.xcf", fmt.Sprintf(`kind: global
version: "1.0"
extends: %q
workflows:
  deploy:

`, base))

	cfg, err := ParseFile(child)
	require.NoError(t, err)
	require.Contains(t, cfg.Workflows, "deploy", "child workflow must be present in merged config")
}

// TestMerge_Workflows_BasePreserved verifies that base workflows survive when the
// child introduces a different workflow ID.
func TestMerge_Workflows_BasePreserved(t *testing.T) {
	base := writeTemp(t, "base.xcf", `kind: global
version: "1.0"
workflows:
  build:

`)
	child := writeTemp(t, "child.xcf", fmt.Sprintf(`kind: global
version: "1.0"
extends: %q
workflows:
  deploy:

`, base))

	cfg, err := ParseFile(child)
	require.NoError(t, err)
	assert.Contains(t, cfg.Workflows, "build", "base workflow must be inherited")
	assert.Contains(t, cfg.Workflows, "deploy", "child workflow must be present")
}

// TestMerge_Workflows_ChildOverridesBase verifies that when base and child define
// the same workflow ID, the child's definition wins.
func TestMerge_Workflows_ChildOverridesBase(t *testing.T) {
	tmp := t.TempDir()
	baseDir := filepath.Join(tmp, "base")
	childDir := filepath.Join(tmp, "child")
	require.NoError(t, os.MkdirAll(filepath.Join(baseDir, "xcf", "workflows"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(childDir, "xcf", "workflows"), 0755))

	baseWf := `---
kind: workflow
version: "1.0"
name: deploy
---
Base deploy instructions.
`
	childWf := `---
kind: workflow
version: "1.0"
name: deploy
---
Child deploy instructions.
`
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "xcf", "workflows", "deploy.xcf"), []byte(baseWf), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(childDir, "xcf", "workflows", "deploy.xcf"), []byte(childWf), 0600))

	// Child extends base
	childProject := fmt.Sprintf(`kind: project
version: "1.0"
name: child
extends: %s
`, filepath.Join(baseDir, ".xcaffold", "project.xcf"))
	require.NoError(t, os.MkdirAll(filepath.Join(baseDir, ".xcaffold"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(childDir, ".xcaffold"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, ".xcaffold", "project.xcf"), []byte("kind: project\nversion: \"1.0\"\nname: base"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(childDir, ".xcaffold", "project.xcf"), []byte(childProject), 0600))

	cfg, err := ParseDirectory(childDir)
	require.NoError(t, err)
	wf, ok := cfg.Workflows["deploy"]
	require.True(t, ok)
	assert.Equal(t, "Child deploy instructions.", wf.Body, "child definition must override base")
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

// TestParse_Workflow_FullSchema verifies that a workflow with all supported
// fields (api-version, steps, targets with lowering-strategy) round-trips
// through the parser without error.
func TestParse_Workflow_FullSchema(t *testing.T) {
	input := `---
kind: workflow
version: "1.0"
name: code-review
api-version: workflow/v1
description: Multi-step PR review procedure.
steps:
  - name: analyze
    description: Read the diff and summarize changed modules.

  - name: lint
    description: Check style and flag violations.

  - name: summarize

targets:
  claude:
    provider:
      lowering-strategy: rule-plus-skill
  copilot:
    provider:
      lowering-strategy: prompt-file
---
## analyze
Read the diff.

## lint
Run golangci-lint.

## summarize
Write the summary.
`
	path := writeTemp(t, "code-review.xcf", input)

	config, err := ParseFile(path)
	require.NoError(t, err)

	wf, ok := config.Workflows["code-review"]
	require.True(t, ok)
	require.Equal(t, "workflow/v1", wf.ApiVersion)
	require.Len(t, wf.Steps, 3)
	require.Equal(t, "analyze", wf.Steps[0].Name)
	assert.Equal(t, "Read the diff.", wf.Steps[0].Body)
	require.Equal(t, "lint", wf.Steps[1].Name)
	assert.Equal(t, "Run golangci-lint.", wf.Steps[1].Body)
	require.Equal(t, "rule-plus-skill", wf.Targets["claude"].Provider["lowering-strategy"])
	require.Equal(t, "prompt-file", wf.Targets["copilot"].Provider["lowering-strategy"])
}

// TestParse_Workflow_StepsAndInstructions_Mutex verifies that setting both
// top-level instructions and steps on a workflow is a parse error.
func TestParse_Workflow_StepsAndInstructions_Mutex(t *testing.T) {
	t.Skip("Legacy instructions test removed")

	t.Skip("Legacy instructions test removed")

	input := `
kind: global
version: "1.0"
workflows:
  bad:
    name: bad

    steps:
      - name: step-one

`
	path := writeTemp(t, "project.xcf", input)

	_, err := ParseFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "steps")
	require.Contains(t, err.Error(), "instructions")
}

// TestParse_Workflow_StepMissingName verifies that a workflow step without a
// name field is rejected.
func TestParse_Workflow_StepMissingName(t *testing.T) {
	t.Skip("Legacy instructions test removed")

	t.Skip("Legacy instructions test removed")

	input := `
kind: global
version: "1.0"
workflows:
  nameless:
    name: nameless
    steps:
      - description: No name field here.

`
	path := writeTemp(t, "project.xcf", input)

	_, err := ParseFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "name")
}

// TestParse_Workflow_InvalidLoweringStrategy verifies that an unrecognized
// lowering-strategy value in targets.<provider>.provider is rejected.
func TestParse_Workflow_InvalidLoweringStrategy(t *testing.T) {
	input := `---
kind: workflow
version: "1.0"
name: bad-strategy
steps:
  - name: step-one

targets:
  claude:
    provider:
      lowering-strategy: invalid-value
---
## step-one
Step one body.
`
	path := writeTemp(t, "bad-strategy.xcf", input)

	_, err := ParseFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "lowering-strategy")
}

// TestParse_Workflow_InstructionsFileUnderReservedPrefix_IsRejected verifies
// that a workflow step instructions-file pointing at .agents/ is rejected.
func TestParse_Workflow_InstructionsFileUnderReservedPrefix_IsRejected(t *testing.T) {
	t.Skip("Legacy instructions test removed")

	t.Skip("Legacy instructions test removed")

	input := `
kind: global
version: "1.0"
workflows:
  smuggled:
    name: smuggled
    steps:
      - name: step-one

`
	path := writeTemp(t, "project.xcf", input)

	_, err := ParseFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "reserved")
}

// TestParse_Workflow_Step_InstructionsAndFile_Mutex verifies that a step with
// both instructions and instructions-file set is rejected.
func TestParse_Workflow_Step_InstructionsAndFile_Mutex(t *testing.T) {
	t.Skip("Legacy instructions test removed")

	t.Skip("Legacy instructions test removed")

	input := `
kind: global
version: "1.0"
workflows:
  bad-step:
    name: bad-step
    steps:
      - name: step-one


`
	path := writeTemp(t, "project.xcf", input)

	_, err := ParseFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "mutually exclusive")
}

// TestParse_Workflow_Step_MissingBody verifies that a step with neither
// instructions nor instructions-file is rejected.
func TestParse_Workflow_Step_MissingBody(t *testing.T) {
	input := `
kind: global
version: "1.0"
workflows:
  empty-step:
    name: empty-step
    steps:
      - name: step-one
`
	path := writeTemp(t, "project.xcf", input)

	_, err := ParseFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "instructions")
}

// TestParse_Workflow_Step_ReservedPrefix_GithubPrompts verifies that a step
// instructions-file pointing at .github/prompts/ is rejected.
func TestParse_Workflow_Step_ReservedPrefix_GithubPrompts(t *testing.T) {
	t.Skip("Legacy instructions test removed")

	t.Skip("Legacy instructions test removed")

	input := `
kind: global
version: "1.0"
workflows:
  smuggled:
    name: smuggled
    steps:
      - name: step-one

`
	path := writeTemp(t, "project.xcf", input)

	_, err := ParseFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "reserved")
}

// TestParse_Workflow_UnknownApiVersion_IsRejected verifies that an api-version
// other than "workflow/v1" is rejected.
func TestParse_Workflow_UnknownApiVersion_IsRejected(t *testing.T) {
	input := `
kind: global
version: "1.0"
workflows:
  future:
    api-version: workflow/v2
    name: future
    steps:
      - name: step-one

`
	path := writeTemp(t, "project.xcf", input)

	_, err := ParseFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "api-version")
}
