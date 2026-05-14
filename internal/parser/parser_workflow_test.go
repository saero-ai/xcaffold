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
// with inline YAML instructions parses successfully. (Workflows are pure YAML now.)
func TestValidate_Workflow_ValidInlineInstructions_Accepted(t *testing.T) {
	input := `---
kind: workflow
version: "1.0"
name: build
steps:
  - name: compile
    instructions: "Run go build ./..."
---
This markdown body area is ignored (workflows are pure YAML).
`
	cfg, err := Parse(strings.NewReader(input))
	require.NoError(t, err, "workflow with inline YAML instructions should be accepted")
	wf, ok := cfg.Workflows["build"]
	require.True(t, ok)
	require.Len(t, wf.Steps, 1)
	assert.Equal(t, "Run go build ./...", wf.Steps[0].Instructions)
	// Body should not be assigned for pure YAML workflows
	assert.Empty(t, wf.Body)
}

// TestMerge_Workflows_ChildAddsNew verifies that a child config's workflows are
// present in the merged result when the base has none.
func TestMerge_Workflows_ChildAddsNew(t *testing.T) {
	base := writeTemp(t, "base.xcaf", `kind: project
version: "1.0"
name: "base-project"
`)
	child := writeTemp(t, "child.xcaf", fmt.Sprintf(`kind: global
version: "1.0"
extends: %q
workflows:
  deploy:
    steps:
      - name: verify
        instructions: "Verify the deployment"
`, base))

	cfg, err := ParseFile(child)
	require.NoError(t, err)
	require.Contains(t, cfg.Workflows, "deploy", "child workflow must be present in merged config")
}

// TestMerge_Workflows_BasePreserved verifies that base workflows survive when the
// child introduces a different workflow ID.
func TestMerge_Workflows_BasePreserved(t *testing.T) {
	base := writeTemp(t, "base.xcaf", `kind: global
version: "1.0"
workflows:
  build:
    steps:
      - name: compile
        instructions: "Compile the code"
`)
	child := writeTemp(t, "child.xcaf", fmt.Sprintf(`kind: global
version: "1.0"
extends: %q
workflows:
  deploy:
    steps:
      - name: verify
        instructions: "Verify the deployment"
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
	require.NoError(t, os.MkdirAll(filepath.Join(baseDir, "xcaf", "workflows"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(childDir, "xcaf", "workflows"), 0755))

	baseWf := `---
kind: workflow
version: "1.0"
name: deploy
steps:
  - name: deploy-base
    instructions: "Base deploy instructions"
---
`
	childWf := `---
kind: workflow
version: "1.0"
name: deploy
steps:
  - name: deploy-child
    instructions: "Child deploy instructions"
---
`
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "xcaf", "workflows", "deploy.xcaf"), []byte(baseWf), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(childDir, "xcaf", "workflows", "deploy.xcaf"), []byte(childWf), 0600))

	// Child extends base
	childProject := fmt.Sprintf(`kind: project
version: "1.0"
name: child
extends: %s
`, filepath.Join(baseDir, ".xcaffold", "project.xcaf"))
	require.NoError(t, os.MkdirAll(filepath.Join(baseDir, ".xcaffold"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(childDir, ".xcaffold"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, ".xcaffold", "project.xcaf"), []byte("kind: project\nversion: \"1.0\"\nname: base"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(childDir, ".xcaffold", "project.xcaf"), []byte(childProject), 0600))

	cfg, err := ParseDirectory(childDir)
	require.NoError(t, err)
	wf, ok := cfg.Workflows["deploy"]
	require.True(t, ok)
	require.Len(t, wf.Steps, 1)
	assert.Equal(t, "deploy-child", wf.Steps[0].Name, "child definition must override base")
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
// through the parser without error. Pure YAML format.
func TestParse_Workflow_FullSchema(t *testing.T) {
	input := `---
kind: workflow
version: "1.0"
name: code-review
api-version: workflow/v1
description: Multi-step PR review procedure.
steps:
  - name: analyze
    instructions: "Read the diff and summarize changed modules"

  - name: lint
    instructions: "Run golangci-lint to check style and flag violations"

  - name: summarize
    instructions: "Write a summary of the changes"

targets:
  claude:
    provider:
      lowering-strategy: rule-plus-skill
  copilot:
    provider:
      lowering-strategy: prompt-file
---
This markdown body area is ignored (workflows are pure YAML).
`
	path := writeTemp(t, "code-review.xcaf", input)

	config, err := ParseFile(path)
	require.NoError(t, err)

	wf, ok := config.Workflows["code-review"]
	require.True(t, ok)
	require.Equal(t, "workflow/v1", wf.ApiVersion)
	require.Len(t, wf.Steps, 3)
	require.Equal(t, "analyze", wf.Steps[0].Name)
	assert.Equal(t, "Read the diff and summarize changed modules", wf.Steps[0].Instructions)
	require.Equal(t, "lint", wf.Steps[1].Name)
	assert.Equal(t, "Run golangci-lint to check style and flag violations", wf.Steps[1].Instructions)
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
	path := writeTemp(t, "project.xcaf", input)

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
	path := writeTemp(t, "project.xcaf", input)

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
    instructions: "Do something"

targets:
  claude:
    provider:
      lowering-strategy: invalid-value
---
`
	path := writeTemp(t, "bad-strategy.xcaf", input)

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
	path := writeTemp(t, "project.xcaf", input)

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
	path := writeTemp(t, "project.xcaf", input)

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
	path := writeTemp(t, "project.xcaf", input)

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
	path := writeTemp(t, "project.xcaf", input)

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
	path := writeTemp(t, "project.xcaf", input)

	_, err := ParseFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "api-version")
}

// T-11: TestValidation_Workflow_NoSteps — Workflow with zero steps should error
func TestValidation_Workflow_NoSteps(t *testing.T) {
	input := `kind: global
version: "1.0"
workflows:
  empty-wf:
    name: empty-wf
    description: "has no steps"
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must define at least one step")
}

// T-12: TestValidation_Step_NoSkillNoInstructions — Step with neither skill nor instructions
func TestValidation_Step_NoSkillNoInstructions(t *testing.T) {
	input := `kind: global
version: "1.0"
workflows:
  bad-wf:
    name: bad-wf
    steps:
      - name: empty-step
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must define skill reference or instructions")
}

// T-13: TestValidation_Step_SkillOnly — Step with skill but no instructions is valid
func TestValidation_Step_SkillOnly(t *testing.T) {
	input := `kind: global
version: "1.0"
workflows:
  ok-wf:
    name: ok-wf
    steps:
      - name: step-one
        skill: tdd
`
	cfg, err := Parse(strings.NewReader(input))
	require.NoError(t, err)
	require.Equal(t, "tdd", cfg.Workflows["ok-wf"].Steps[0].Skill)
}

// T-14: TestValidation_Step_InstructionsOnly — Step with instructions but no skill is valid
func TestValidation_Step_InstructionsOnly(t *testing.T) {
	input := `kind: global
version: "1.0"
workflows:
  ok-wf:
    name: ok-wf
    steps:
      - name: step-one
        instructions: "Do the thing"
`
	cfg, err := Parse(strings.NewReader(input))
	require.NoError(t, err)
	require.Equal(t, "Do the thing", cfg.Workflows["ok-wf"].Steps[0].Instructions)
}

// T-15: TestValidation_Step_BothSkillAndInstructions — Step with both fields is valid
func TestValidation_Step_BothSkillAndInstructions(t *testing.T) {
	input := `kind: global
version: "1.0"
workflows:
  ok-wf:
    name: ok-wf
    steps:
      - name: step-one
        skill: tdd
        instructions: "Extra context for the skill"
`
	cfg, err := Parse(strings.NewReader(input))
	require.NoError(t, err)
	step := cfg.Workflows["ok-wf"].Steps[0]
	require.Equal(t, "tdd", step.Skill)
	require.Equal(t, "Extra context for the skill", step.Instructions)
}

// T-16: TestValidation_WorkflowBodySteps_NoLongerMutex — Verify the old body+steps mutex error is gone.
// A workflow with steps does not trigger the old "mutually exclusive" error.
func TestValidation_WorkflowBodySteps_NoLongerMutex(t *testing.T) {
	input := `kind: global
version: "1.0"
workflows:
  my-wf:
    name: my-wf
    steps:
      - name: step-one
        instructions: "First step"
`
	cfg, err := Parse(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, cfg.Workflows["my-wf"].Steps, 1)
}

// T-17: TestParser_Workflow_PureYAML_NoBodyAssignment — Workflow with YAML instructions parses correctly
func TestParser_Workflow_PureYAML_NoBodyAssignment(t *testing.T) {
	input := `kind: global
version: "1.0"
workflows:
  my-wf:
    name: my-wf
    steps:
      - name: step-one
        instructions: "First step instructions"
      - name: step-two
        skill: review
`
	cfg, err := Parse(strings.NewReader(input))
	require.NoError(t, err)
	wf := cfg.Workflows["my-wf"]
	require.Len(t, wf.Steps, 2)
	require.Equal(t, "First step instructions", wf.Steps[0].Instructions)
	require.Equal(t, "review", wf.Steps[1].Skill)
}

// T-18: TestParser_Workflow_MarkdownBodyIgnored — Markdown body area is not parsed into step bodies for workflows.
// Step instructions come from YAML only.
func TestParser_Workflow_MarkdownBodyIgnored(t *testing.T) {
	input := `---
kind: workflow
version: "1.0"
name: my-wf
steps:
  - name: step-one
    instructions: "YAML instruction"
---
## step-one
This markdown heading should be ignored (not assigned to step body).

## unused-heading
This section does nothing.
`
	path := writeTemp(t, "workflow.xcaf", input)
	cfg, err := ParseFile(path)
	require.NoError(t, err)
	wf, ok := cfg.Workflows["my-wf"]
	require.True(t, ok)
	require.Len(t, wf.Steps, 1)
	require.Equal(t, "YAML instruction", wf.Steps[0].Instructions)
	// The markdown body area should not be assigned to wf.Body or step bodies
	require.Empty(t, wf.Body, "workflow body should not be assigned from markdown for pure YAML workflows")
}
