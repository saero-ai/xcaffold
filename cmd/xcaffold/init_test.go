package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInitWizard_GeneratesMultiKindFormat verifies that buildXCFContent emits
// a multi-kind scaffold with a kind: project document and a kind: agent document.
func TestInitWizard_GeneratesMultiKindFormat(t *testing.T) {
	ans := wizardAnswers{
		name:      "test-project",
		desc:      "",
		target:    "claude",
		wantAgent: true,
	}
	content := buildXCFContent(ans)

	// Must contain kind: project (not kind: config)
	assert.Contains(t, content, "kind: project")
	assert.NotContains(t, content, "kind: config")

	// name must be at top level, not nested under project:
	assert.Contains(t, content, `name: "test-project"`)
	assert.NotContains(t, content, "project:")

	// Must declare targets
	assert.Contains(t, content, "targets:")
	assert.Contains(t, content, "- claude")

	// Must contain kind: agent document
	assert.Contains(t, content, "kind: agent")
	assert.Contains(t, content, "name: developer")

	// Must have --- separator between documents
	assert.Contains(t, content, "---")

	// Must be parseable as a valid XcaffoldConfig
	config, err := parser.Parse(strings.NewReader(content))
	require.NoError(t, err)
	require.NotNil(t, config.Project)
	assert.Equal(t, "test-project", config.Project.Name)
	assert.Equal(t, []string{"claude"}, config.Project.Targets)
	assert.Contains(t, config.Agents, "developer")
}

// TestInitWizard_GeneratesMultiKindFormat_NoAgent verifies that when wantAgent
// is false, only the kind: project document is emitted (no separator, no agent).
func TestInitWizard_GeneratesMultiKindFormat_NoAgent(t *testing.T) {
	ans := wizardAnswers{
		name:      "test-project",
		desc:      "",
		target:    "claude",
		wantAgent: false,
	}
	content := buildXCFContent(ans)
	assert.Contains(t, content, "kind: project")
	assert.NotContains(t, content, "kind: config")
	assert.NotContains(t, content, "kind: agent")
	assert.Contains(t, content, "targets:")
}

// TestRunInit_GlobalFlag_NotBlockedByExistingScaffoldXCF verifies that
// --global bypasses the local scaffold.xcf idempotency check.
// Regression: globalFlag was checked AFTER the scaffold.xcf stat, causing
// `xcaffold init --global` to silently no-op when a local scaffold.xcf existed.
func TestRunInit_GlobalFlag_NotBlockedByExistingScaffoldXCF(t *testing.T) {
	// Create a temp dir with a scaffold.xcf already present.
	dir := t.TempDir()
	xcfPath := filepath.Join(dir, "scaffold.xcf")
	require.NoError(t, os.WriteFile(xcfPath, []byte("kind: project\nname: existing\n"), 0600))

	// Change to the temp dir so the idempotency check finds scaffold.xcf.
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(orig) }()

	// Set globalFlag = true to simulate --global, and restore afterwards.
	globalFlag = true
	defer func() { globalFlag = false }()

	// Build a minimal cobra.Command so runInit can write output.
	cmd := &cobra.Command{}
	var buf strings.Builder
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// runInit must NOT return nil with "Nothing to do" — it must reach initGlobal.
	// initGlobal will fail because ~/.xcaffold/global.xcf may or may not exist,
	// but the key assertion is that the early-return idempotency message is NOT printed.
	_ = runInit(cmd, nil)

	output := buf.String()
	assert.NotContains(t, output, "Nothing to do",
		"--global should bypass the local scaffold.xcf idempotency check")
}

// TestInitWizard_GeneratesMultiKindFormat_TargetCursor verifies that the
// targets field reflects the chosen target when it is not "claude".
func TestInitWizard_GeneratesMultiKindFormat_TargetCursor(t *testing.T) {
	ans := wizardAnswers{
		name:      "cursor-project",
		desc:      "",
		target:    "cursor",
		wantAgent: false,
	}
	content := buildXCFContent(ans)
	assert.Contains(t, content, "kind: project")
	assert.Contains(t, content, "- cursor")

	// Must round-trip through the parser
	config, err := parser.Parse(strings.NewReader(content))
	require.NoError(t, err)
	require.NotNil(t, config.Project)
	assert.Equal(t, []string{"cursor"}, config.Project.Targets)
}

// TestBuildXCFContent_CanonicalFieldOrdering verifies that the agent document
// emits fields in canonical order: name, description, model, effort, tools, instructions.
func TestBuildXCFContent_CanonicalFieldOrdering(t *testing.T) {
	ans := wizardAnswers{
		name:      "test-project",
		desc:      "",
		target:    "claude",
		wantAgent: true,
	}

	content := buildXCFContent(ans)

	// Extract only the agent document (everything after the --- separator).
	parts := strings.SplitN(content, "---\n", 2)
	require.Len(t, parts, 2, "expected --- document separator in generated content")
	agentDoc := parts[1]

	orderedKeys := []string{
		"name: developer",
		"description:",
		"\nmodel:",
		"effort:",
		"tools:",
		"instructions:",
	}

	lastIdx := -1
	for _, key := range orderedKeys {
		idx := strings.Index(agentDoc, key)
		require.NotEqual(t, -1, idx, "key %q not found in agent document", key)
		require.Greater(t, idx, lastIdx, "key %q appeared before a prior key in agent document", key)
		lastIdx = idx
	}
}

func TestInit_WritesAgentReferenceByDefault(t *testing.T) {
	tmp := t.TempDir()

	// Ensure flag is false (default state).
	noReferencesFlag = false
	require.NoError(t, writeReferenceTemplates(tmp))

	refPath := filepath.Join(tmp, "xcf", "references", "agent.xcf.reference")
	_, err := os.Stat(refPath)
	require.NoError(t, err, "agent.xcf.reference must exist at %s", refPath)

	data, err := os.ReadFile(refPath)
	require.NoError(t, err)
	require.Contains(t, string(data), "Agent Kind — Full Field Reference")
}

func TestInit_SkipsReferencesWithFlag(t *testing.T) {
	tmp := t.TempDir()

	noReferencesFlag = true
	defer func() { noReferencesFlag = false }()

	require.NoError(t, writeReferenceTemplates(tmp))

	refPath := filepath.Join(tmp, "xcf", "references", "agent.xcf.reference")
	_, err := os.Stat(refPath)
	require.True(t, os.IsNotExist(err), "reference file must NOT be created when --no-references is set")
}

func TestBuildXCFContent_IncludesReferencePointer(t *testing.T) {
	ans := wizardAnswers{
		name:      "test",
		target:    "claude",
		wantAgent: true,
	}
	content := buildXCFContent(ans)

	require.Contains(t, content, "xcf/references/agent.xcf.reference")
}

func TestWriteReferenceTemplates_GeneratesSkillReference(t *testing.T) {
	tmp := t.TempDir()
	noReferencesFlag = false
	require.NoError(t, writeReferenceTemplates(tmp))

	path := filepath.Join(tmp, "xcf", "references", "skill.xcf.reference")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	body := string(data)
	require.Contains(t, body, "Skill Kind — Full Field Reference")
	require.Contains(t, body, "allowed-tools:")
	require.Contains(t, body, "disable-model-invocation:")
	require.Contains(t, body, "targets:")
}

func TestWriteReferenceTemplates_NoReferencesFlag_SkipsSkill(t *testing.T) {
	tmp := t.TempDir()
	noReferencesFlag = true
	t.Cleanup(func() { noReferencesFlag = false })

	require.NoError(t, writeReferenceTemplates(tmp))

	path := filepath.Join(tmp, "xcf", "references", "skill.xcf.reference")
	_, err := os.Stat(path)
	require.True(t, os.IsNotExist(err), "skill.xcf.reference should not exist when --no-references is set")
}

func TestInit_EndToEnd_GeneratesFieldOrderedAgent(t *testing.T) {
	tmp := t.TempDir()

	ans := wizardAnswers{
		name:      "e2e-test",
		desc:      "End-to-end test project",
		target:    "claude",
		wantAgent: true,
	}
	content := buildXCFContent(ans)
	xcfPath := filepath.Join(tmp, "scaffold.xcf")
	require.NoError(t, os.WriteFile(xcfPath, []byte(content), 0o600))

	noReferencesFlag = false
	require.NoError(t, writeReferenceTemplates(tmp))

	config, err := parser.ParseFile(xcfPath)
	require.NoError(t, err, "generated scaffold.xcf must be parseable")
	require.NotNil(t, config)

	agent, ok := config.Agents["developer"]
	require.True(t, ok, "developer agent must be present")
	require.Equal(t, "General software developer agent.", agent.Description)
	require.NotEmpty(t, agent.Model)

	refPath := filepath.Join(tmp, "xcf", "references", "agent.xcf.reference")
	_, err = os.Stat(refPath)
	require.NoError(t, err, "agent.xcf.reference must exist")
}

func TestInit_E2E_SkillReferenceArtifact(t *testing.T) {
	tmp := t.TempDir()
	noReferencesFlag = false

	// Run init's reference generation step
	require.NoError(t, writeReferenceTemplates(tmp))

	// Verify both agent and skill references exist
	for _, name := range []string{"agent.xcf.reference", "skill.xcf.reference"} {
		path := filepath.Join(tmp, "xcf", "references", name)
		_, err := os.Stat(path)
		require.NoError(t, err, "expected %s to exist", name)
	}

	// Verify skill reference contains canonical-only field names
	skillData, err := os.ReadFile(filepath.Join(tmp, "xcf", "references", "skill.xcf.reference"))
	require.NoError(t, err)
	skillBody := string(skillData)
	require.Contains(t, skillBody, "allowed-tools:")
	require.NotContains(t, skillBody, "\ntools:") // legacy name must not appear
}
