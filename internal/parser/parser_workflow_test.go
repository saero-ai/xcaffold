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
    instructions: "bad"
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
    instructions: "bad"
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
    instructions: "bad"
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err, "workflow ID with backslash must be rejected")
	assert.Contains(t, err.Error(), "workflow id contains invalid characters")
}

// TestValidate_Workflow_MutualExclusivity verifies that setting both instructions
// and instructions-file on the same workflow is a parse error.
func TestValidate_Workflow_MutualExclusivity(t *testing.T) {
	input := `kind: global
version: "1.0"
workflows:
  deploy:
    instructions: "Inline instructions."
    instructions-file: "workflows/deploy.md"
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err, "both instructions and instructions-file set must be rejected")
	assert.Contains(t, err.Error(), "mutually exclusive")
}

// TestValidate_Workflow_InstructionsFile_AbsolutePath_Rejected verifies that an
// absolute path in a workflow's instructions-file is rejected.
func TestValidate_Workflow_InstructionsFile_AbsolutePath_Rejected(t *testing.T) {
	input := `kind: global
version: "1.0"
workflows:
  release:
    instructions-file: "/etc/passwd"
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err, "absolute instructions-file path must be rejected")
	assert.Contains(t, err.Error(), "relative path")
}

// TestValidate_Workflow_InstructionsFile_PathTraversal_Rejected verifies that
// path traversal in a workflow's instructions-file is rejected.
func TestValidate_Workflow_InstructionsFile_PathTraversal_Rejected(t *testing.T) {
	input := `kind: global
version: "1.0"
workflows:
  release:
    instructions-file: "../outside/release.md"
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err, "instructions-file with path traversal must be rejected")
	assert.Contains(t, err.Error(), "instructions-file")
}

// TestValidate_Workflow_ValidRelativePath_Accepted verifies that a valid relative
// instructions-file path on a workflow parses successfully.
func TestValidate_Workflow_ValidRelativePath_Accepted(t *testing.T) {
	input := `kind: global
version: "1.0"
workflows:
  release:
    description: "Release workflow"
    instructions-file: "workflows/release.md"
`
	cfg, err := Parse(strings.NewReader(input))
	require.NoError(t, err, "valid relative instructions-file should be accepted")
	wf, ok := cfg.Workflows["release"]
	require.True(t, ok)
	assert.Equal(t, "workflows/release.md", wf.InstructionsFile)
}

// TestValidate_Workflow_ValidInlineInstructions_Accepted verifies that a workflow
// with only inline instructions parses successfully.
func TestValidate_Workflow_ValidInlineInstructions_Accepted(t *testing.T) {
	input := `kind: global
version: "1.0"
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
	base := writeTemp(t, "base.xcf", `kind: project
version: "1.0"
name: "base-project"
`)
	child := writeTemp(t, "child.xcf", fmt.Sprintf(`kind: global
version: "1.0"
extends: %q
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
	base := writeTemp(t, "base.xcf", `kind: global
version: "1.0"
workflows:
  build:
    instructions: "Run go build."
`)
	child := writeTemp(t, "child.xcf", fmt.Sprintf(`kind: global
version: "1.0"
extends: %q
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
	base := writeTemp(t, "base.xcf", `kind: global
version: "1.0"
workflows:
  deploy:
    instructions: "Base deploy instructions."
`)
	child := writeTemp(t, "child.xcf", fmt.Sprintf(`kind: global
version: "1.0"
extends: %q
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

// TestParse_Workflow_FullSchema verifies that a workflow with all supported
// fields (api-version, steps, targets with lowering-strategy) round-trips
// through the parser without error.
func TestParse_Workflow_FullSchema(t *testing.T) {
	input := `
kind: global
version: "1.0"
workflows:
  code-review:
    api-version: workflow/v1
    name: code-review
    description: Multi-step PR review procedure.
    steps:
      - name: analyze
        description: Read the diff and summarize changed modules.
        instructions-file: xcf/workflows/code-review/01-analyze.md
      - name: lint
        description: Check style and flag violations.
        instructions: Lint the changed files.
      - name: summarize
        instructions: Write the review comment.
    targets:
      claude:
        provider:
          lowering-strategy: rule-plus-skill
      copilot:
        provider:
          lowering-strategy: prompt-file
`
	path := writeTemp(t, "project.xcf", input)

	config, err := ParseFile(path)
	require.NoError(t, err)

	wf, ok := config.Workflows["code-review"]
	require.True(t, ok)
	require.Equal(t, "workflow/v1", wf.ApiVersion)
	require.Len(t, wf.Steps, 3)
	require.Equal(t, "analyze", wf.Steps[0].Name)
	require.Equal(t, "xcf/workflows/code-review/01-analyze.md", wf.Steps[0].InstructionsFile)
	require.Equal(t, "lint", wf.Steps[1].Name)
	require.Equal(t, "rule-plus-skill", wf.Targets["claude"].Provider["lowering-strategy"])
	require.Equal(t, "prompt-file", wf.Targets["copilot"].Provider["lowering-strategy"])
}

// TestParse_Workflow_StepsAndInstructions_Mutex verifies that setting both
// top-level instructions and steps on a workflow is a parse error.
func TestParse_Workflow_StepsAndInstructions_Mutex(t *testing.T) {
	input := `
kind: global
version: "1.0"
workflows:
  bad:
    name: bad
    instructions: Top-level body.
    steps:
      - name: step-one
        instructions: Step body.
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
	input := `
kind: global
version: "1.0"
workflows:
  nameless:
    name: nameless
    steps:
      - description: No name field here.
        instructions: Body.
`
	path := writeTemp(t, "project.xcf", input)

	_, err := ParseFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "name")
}

// TestParse_Workflow_InvalidLoweringStrategy verifies that an unrecognized
// lowering-strategy value in targets.<provider>.provider is rejected.
func TestParse_Workflow_InvalidLoweringStrategy(t *testing.T) {
	input := `
kind: global
version: "1.0"
workflows:
  bad-strategy:
    name: bad-strategy
    steps:
      - name: step-one
        instructions: Body.
    targets:
      claude:
        provider:
          lowering-strategy: invalid-value
`
	path := writeTemp(t, "project.xcf", input)

	_, err := ParseFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "lowering-strategy")
}

// TestParse_Workflow_InstructionsFileUnderReservedPrefix_IsRejected verifies
// that a workflow step instructions-file pointing at .agents/ is rejected.
func TestParse_Workflow_InstructionsFileUnderReservedPrefix_IsRejected(t *testing.T) {
	input := `
kind: global
version: "1.0"
workflows:
  smuggled:
    name: smuggled
    steps:
      - name: step-one
        instructions-file: .agents/workflows/smuggled.md
`
	path := writeTemp(t, "project.xcf", input)

	_, err := ParseFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "reserved")
}

// TestParse_Workflow_Step_InstructionsAndFile_Mutex verifies that a step with
// both instructions and instructions-file set is rejected.
func TestParse_Workflow_Step_InstructionsAndFile_Mutex(t *testing.T) {
	input := `
kind: global
version: "1.0"
workflows:
  bad-step:
    name: bad-step
    steps:
      - name: step-one
        instructions: Inline body.
        instructions-file: xcf/workflows/bad/step.md
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
	input := `
kind: global
version: "1.0"
workflows:
  smuggled:
    name: smuggled
    steps:
      - name: step-one
        instructions-file: .github/prompts/smuggled.md
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
        instructions: Body.
`
	path := writeTemp(t, "project.xcf", input)

	_, err := ParseFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "api-version")
}
