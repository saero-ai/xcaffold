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
