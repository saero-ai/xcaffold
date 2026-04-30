package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/renderer/claude"
	"github.com/stretchr/testify/require"
)

var realDataPath = os.Getenv("XCAFFOLD_TEST_FIXTURES")

func init() {
	if realDataPath == "" {
		realDataPath = filepath.Join(os.Getenv("HOME"), ".xcaffold", "test-fixtures")
	}
}

// TestRealData_Fixture_Exists guards later tests with a clear failure if the
// fixture repo is not present. Skipping on missing fixtures is intentional —
// this keeps CI green when xcaffold-test is not checked out, while still
// catching regressions for developers who have it locally.
func TestRealData_Fixture_Exists(t *testing.T) {
	claudeAgents := filepath.Join(realDataPath, ".claude", "agents")
	if _, err := os.Stat(claudeAgents); os.IsNotExist(err) {
		t.Skipf("fixture %s not present; skipping real-data tests", claudeAgents)
	}

	entries, err := os.ReadDir(claudeAgents)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(entries), 4, "expected at least 4 real agent fixtures in %s", claudeAgents)
}

// TestRealData_ImportedAgent_CompilesWithCanonicalFieldOrder performs the
// full round trip for ONE real agent file: read the markdown frontmatter,
// build an AgentConfig, recompile via the Claude renderer, and verify the
// regenerated frontmatter uses the canonical field ordering.
func TestRealData_ImportedAgent_CompilesWithCanonicalFieldOrder(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	agentPath := filepath.Join(realDataPath, ".claude", "agents", "backend-engineer.md")
	if _, err := os.Stat(agentPath); os.IsNotExist(err) {
		t.Skipf("fixture %s not present; skipping", agentPath)
	}

	// Copy the real agent file into the temp dir so instructions-file can be
	// a relative path (the parser rejects absolute paths in non-global scope).
	tmp := t.TempDir()
	fixtureData, err := os.ReadFile(agentPath)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "backend-engineer.md"), fixtureData, 0o600))

	xcfContent := `---
kind: agent
version: "1.0"
name: backend-engineer
description: "Backend engineer agent."
model: sonnet
effort: high
tools: [Bash, Read, Write, Edit, Glob, Grep]
---

` + string(fixtureData)
	xcfPath := filepath.Join(tmp, "agent.xcf")
	require.NoError(t, os.WriteFile(xcfPath, []byte(xcfContent), 0o600))

	config, err := parser.ParseFile(xcfPath)
	require.NoError(t, err)
	require.Contains(t, config.Agents, "backend-engineer")

	r := claude.New()
	out, _, err := renderer.Orchestrate(r, config, tmp)
	require.NoError(t, err)

	compiledPath := "agents/backend-engineer.md"
	content, ok := out.Files[compiledPath]
	require.True(t, ok, "renderer did not produce %s", compiledPath)

	orderedKeys := []string{
		"name:",
		"description:",
		"model:",
		"effort:",
		"tools:",
	}
	lastIdx := -1
	for _, key := range orderedKeys {
		idx := strings.Index(content, key)
		require.NotEqual(t, -1, idx, "compiled output missing %s", key)
		require.Greater(t, idx, lastIdx, "%s out of canonical order", key)
		lastIdx = idx
	}

	frontmatterEnd := strings.Index(content[4:], "---")
	require.Greater(t, frontmatterEnd, 0, "no closing frontmatter delimiter found")
}

// TestRealData_AllClaudeAgents_Parse verifies that every real agent md file
// can be referenced via instructions-file, parsed, and recompiled without
// error.
func TestRealData_AllClaudeAgents_Parse(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	claudeAgents := filepath.Join(realDataPath, ".claude", "agents")
	if _, err := os.Stat(claudeAgents); os.IsNotExist(err) {
		t.Skipf("fixture %s not present; skipping", claudeAgents)
	}

	entries, err := os.ReadDir(claudeAgents)
	require.NoError(t, err)

	r := claude.New()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		agentFile := filepath.Join(claudeAgents, entry.Name())
		id := strings.TrimSuffix(entry.Name(), ".md")

		t.Run(id, func(t *testing.T) {
			// Copy the real agent file into a temp dir so instructions-file
			// can use a relative path (absolute paths are rejected by the parser).
			tmp := t.TempDir()
			fixtureData, err := os.ReadFile(agentFile)
			require.NoError(t, err, "read fixture for %s", id)
			localName := entry.Name()
			require.NoError(t, os.WriteFile(filepath.Join(tmp, localName), fixtureData, 0o600))

			xcfContent := `---
kind: agent
version: "1.0"
name: ` + id + `
description: "Real-data validation agent."
model: sonnet
---

` + string(fixtureData)
			xcfPath := filepath.Join(tmp, "agent.xcf")
			require.NoError(t, os.WriteFile(xcfPath, []byte(xcfContent), 0o600))

			config, err := parser.ParseFile(xcfPath)
			require.NoError(t, err, "parse failed for %s", id)

			out, _, err := renderer.Orchestrate(r, config, tmp)
			require.NoError(t, err, "compile failed for %s", id)
			require.NotEmpty(t, out.Files["agents/"+id+".md"], "no output for %s", id)
		})
	}
}

// TestRealData_NewFields_RoundTripThroughTargets verifies that the new
// AgentConfig fields (DisableModelInvocation, UserInvocable) and the new
// TargetOverride.Provider pass-through survive a write + parse round trip,
// and that provider-specific fields do NOT leak into other targets.
func TestRealData_NewFields_RoundTripThroughTargets(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	xcfContent := `---
kind: agent
version: "1.0"
name: round-trip
description: "Round-trip validation for new fields."
model: sonnet
disable-model-invocation: true
user-invocable: false
targets:
  gemini:
    provider:
      temperature: 0.7
      timeout_mins: 15
      kind: local
  copilot:
    provider:
      target: github-copilot
      metadata:
        category: review
---

Round-trip test body.
`
	tmp := t.TempDir()
	xcfPath := filepath.Join(tmp, "agent.xcf")
	require.NoError(t, os.WriteFile(xcfPath, []byte(xcfContent), 0o600))

	config, err := parser.ParseFile(xcfPath)
	require.NoError(t, err)

	agent := config.Agents["round-trip"]
	require.NotNil(t, agent.DisableModelInvocation)
	require.True(t, *agent.DisableModelInvocation)
	require.NotNil(t, agent.UserInvocable)
	require.False(t, *agent.UserInvocable)

	gemini := agent.Targets["gemini"]
	require.Equal(t, 0.7, gemini.Provider["temperature"])
	require.Equal(t, 15, gemini.Provider["timeout_mins"])
	require.Equal(t, "local", gemini.Provider["kind"])

	copilot := agent.Targets["copilot"]
	require.Equal(t, "github-copilot", copilot.Provider["target"])

	r := claude.New()
	out, _, err := renderer.Orchestrate(r, config, tmp)
	require.NoError(t, err)

	claudeMd := out.Files["agents/round-trip.md"]
	require.NotContains(t, claudeMd, "disable-model-invocation", "Copilot-only agent field leaked into Claude output")
	require.NotContains(t, claudeMd, "user-invocable", "Copilot-only agent field leaked into Claude output")
	require.NotContains(t, claudeMd, "temperature", "provider-specific gemini field leaked into Claude output")
	require.NotContains(t, claudeMd, "timeout_mins", "provider-specific gemini field leaked into Claude output")
	require.NotContains(t, claudeMd, "github-copilot", "provider-specific copilot field leaked into Claude output")
}
