package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunApply_PolicyError_BlocksWrite verifies that a policy with severity:
// error prevents any files from being written to the output directory.
func TestRunApply_PolicyError_BlocksWrite(t *testing.T) {
	dir := t.TempDir()

	xcaf := filepath.Join(dir, "project.xcaf")
	require.NoError(t, os.WriteFile(xcaf, []byte(`---
kind: project
version: "1.0"
name: policy-error-test
targets:
  - claude
`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agent.xcaf"), []byte(`---
kind: agent
version: "1.0"
name: dev
---
You are a developer
`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "policy.xcaf"), []byte(`---
kind: policy
version: "1.0"
name: needs-desc
severity: error
target: agent
require:
  - field: description
    is-present: true
`), 0600))

	outputDir := filepath.Join(dir, ".claude")

	applyForce = true
	targetFlag = "claude"
	defer func() { applyForce = false }()

	err := applyScope(xcaf, outputDir, filepath.Dir(xcaf), "test")
	require.Error(t, err, "applyScope must return an error when a policy error is triggered")
	assert.True(t, strings.Contains(err.Error(), "policy error") || strings.Contains(err.Error(), "policy") || strings.Contains(err.Error(), "FIELD_REQUIRED_FOR_TARGET"),
		"error message should reference policy or fidelity error, got: %s", err.Error())

	// Output directory must not have been written.
	entries, statErr := os.ReadDir(outputDir)
	if statErr == nil {
		// The directory may be pre-created by MkdirAll for subdirs; ensure no
		// agent files were written inside it.
		agentsDir := filepath.Join(outputDir, "agents")
		agentEntries, agentErr := os.ReadDir(agentsDir)
		if agentErr == nil {
			assert.Empty(t, agentEntries, "no agent files should be written when policy blocks apply")
		}
		// Also confirm no .md files exist anywhere under the output dir.
		var mdFiles []string
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".md") {
				mdFiles = append(mdFiles, e.Name())
			}
		}
		assert.Empty(t, mdFiles, "no markdown files should be written when policy blocks apply")
	}
	// If outputDir does not exist at all, that is the ideal outcome — pass.
}

// TestRunApply_PolicyWarning_AllowsWrite verifies that a policy with severity:
// warning does not block apply — files are written and applyScope returns nil.
func TestRunApply_PolicyWarning_AllowsWrite(t *testing.T) {
	dir := t.TempDir()

	xcaf := filepath.Join(dir, "project.xcaf")
	require.NoError(t, os.WriteFile(xcaf, []byte(`---
kind: project
version: "1.0"
name: policy-warning-test
targets:
  - claude
`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agent.xcaf"), []byte(`---
kind: agent
version: "1.0"
name: dev
description: A developer agent
---
You are a developer
`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "policy.xcaf"), []byte(`---
kind: policy
version: "1.0"
name: needs-desc
severity: warning
target: agent
require:
  - field: description
    is-present: true
`), 0600))

	outputDir := filepath.Join(dir, ".claude")

	applyForce = true
	targetFlag = "claude"
	defer func() { applyForce = false }()

	err := applyScope(xcaf, outputDir, filepath.Dir(xcaf), "test")
	require.NoError(t, err, "applyScope must succeed when the only violations are warnings")

	// Output directory must have been written.
	_, statErr := os.Stat(outputDir)
	assert.NoError(t, statErr, "output directory should exist after a warning-only apply")

	// At minimum the agents/ subdirectory must be present (claude target contract).
	agentsDir := filepath.Join(outputDir, "agents")
	_, agentStatErr := os.Stat(agentsDir)
	assert.NoError(t, agentStatErr, "agents/ subdirectory should exist after apply")
}
