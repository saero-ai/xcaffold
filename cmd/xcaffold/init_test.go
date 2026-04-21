package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInitWizard_GeneratesMultiKindFormat verifies that buildXCFContent emits
// a multi-kind scaffold with a kind: project document and a kind: agent document.
func TestInitWizard_Targets_IsSlice(t *testing.T) {
	ans := wizardAnswers{
		name:    "test-project",
		targets: []string{"claude", "cursor"},
	}
	// targets must be a []string field, not a string
	require.Len(t, ans.targets, 2)
	assert.Equal(t, "claude", ans.targets[0])
	assert.Equal(t, "cursor", ans.targets[1])
}

// TestRunInit_GlobalFlag_NotBlockedByExistingScaffoldXCF verifies that
// --global bypasses the local project.xcf idempotency check.
// Regression: globalFlag was checked AFTER the project.xcf stat, causing
// `xcaffold init --global` to silently no-op when a local project.xcf existed.
func TestRunInit_GlobalFlag_NotBlockedByExistingScaffoldXCF(t *testing.T) {
	// Create a temp dir with a project.xcf already present.
	dir := t.TempDir()
	xcfPath := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(xcfPath, []byte("kind: project\nname: existing\n"), 0600))

	// Change to the temp dir so the idempotency check finds project.xcf.
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
		"--global should bypass the local project.xcf idempotency check")
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

// --- Multi-File Generation Tests ---

func TestWriteXCFDirectory_CreatesLayout(t *testing.T) {
	ans := wizardAnswers{
		name:      "multi-file-test",
		desc:      "Test project",
		targets:   []string{"claude", "cursor"},
		wantAgent: true,
	}

	tmpDir := t.TempDir()
	err := writeXCFDirectory(tmpDir, ans)
	require.NoError(t, err)

	// Verify top-level project.xcf
	scaffoldBytes, err := os.ReadFile(filepath.Join(tmpDir, "project.xcf"))
	require.NoError(t, err)
	content := string(scaffoldBytes)
	assert.Contains(t, content, "kind: project")
	assert.Contains(t, content, "- claude")
	assert.Contains(t, content, "rules:")
	assert.Contains(t, content, "policies:")

	// Verify xcf/ agents directory and file
	agentFile := filepath.Join(tmpDir, "xcf", "agents", "developer.xcf")
	assert.FileExists(t, agentFile)
	agentBytes, _ := os.ReadFile(agentFile)
	assert.Contains(t, string(agentBytes), "kind: agent")

	// Verify xcf/ rules directory and file
	ruleFile := filepath.Join(tmpDir, "xcf", "rules", "conventions.xcf")
	assert.FileExists(t, ruleFile)
	ruleBytes, _ := os.ReadFile(ruleFile)
	assert.Contains(t, string(ruleBytes), "kind: rule")

	// Verify xcf/ policies directory and files (split into one file per policy)
	descPolicyFile := filepath.Join(tmpDir, "xcf", "policies", "require-agent-description.xcf")
	assert.FileExists(t, descPolicyFile)
	descPolicyBytes, _ := os.ReadFile(descPolicyFile)
	assert.Contains(t, string(descPolicyBytes), "kind: policy")
	assert.Contains(t, string(descPolicyBytes), "require-agent-description")

	instrPolicyFile := filepath.Join(tmpDir, "xcf", "policies", "require-agent-instructions.xcf")
	assert.FileExists(t, instrPolicyFile)
	instrPolicyBytes, _ := os.ReadFile(instrPolicyFile)
	assert.Contains(t, string(instrPolicyBytes), "kind: policy")
	assert.Contains(t, string(instrPolicyBytes), "require-agent-instructions")

	// Verify xcf/ settings file
	settingsFile := filepath.Join(tmpDir, "xcf", "settings.xcf")
	assert.FileExists(t, settingsFile)
	settingsBytes, _ := os.ReadFile(settingsFile)
	assert.Contains(t, string(settingsBytes), "kind: settings")
}

func TestWriteXCFDirectory_NoAgent_StillCreatesScaffold(t *testing.T) {
	ans := wizardAnswers{
		name:      "no-agent-test",
		targets:   []string{"claude"},
		wantAgent: false, // User chose NO starter agent
	}

	tmpDir := t.TempDir()
	err := writeXCFDirectory(tmpDir, ans)
	require.NoError(t, err)

	// project.xcf should exist
	assert.FileExists(t, filepath.Join(tmpDir, "project.xcf"))

	// but xcf/ should NOT contain an agents/developer.xcf
	assert.NoFileExists(t, filepath.Join(tmpDir, "xcf", "agents", "developer.xcf"))
}

// --- Target Flag & Flow Tests ---

// TestInit_GeneratesProjectXcf verifies that writeXCFDirectory writes project.xcf
// (not project.xcf) as the top-level entry point.
func TestInit_GeneratesProjectXcf(t *testing.T) {
	ans := wizardAnswers{
		name:      "my-project",
		targets:   []string{"claude"},
		wantAgent: false,
	}

	tmpDir := t.TempDir()
	err := writeXCFDirectory(tmpDir, ans)
	require.NoError(t, err)

	_, projectXcfErr := os.Stat(filepath.Join(tmpDir, "project.xcf"))

	assert.NoError(t, projectXcfErr, "project.xcf should be generated")
}
