package antigravity_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer/antigravity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAntigravityRenderer_FullConfig exercises rules, skills, and silently-skipped
// resource types (agents, hooks, MCP) in a single compilation pass and verifies
// all AG-specific normalizations are applied correctly.
func TestAntigravityRenderer_FullConfig(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Project: ast.ProjectConfig{Name: "integration-test"},
		Agents: map[string]ast.AgentConfig{
			"worker": {
				Name:         "Worker Agent",
				Instructions: "Process jobs efficiently.",
			},
		},
		Skills: map[string]ast.SkillConfig{
			"deploy": {
				Name:         "Deploy Skill",
				Description:  "Handles deployment steps.",
				Instructions: "Run the deploy pipeline.",
				// CC-only fields that must be dropped
				Model:  "claude-opus-4-5",
				Effort: "high",
				Tools:  []string{"Bash"},
			},
		},
		Rules: map[string]ast.RuleConfig{
			"formatting": {
				Description:  "Code style rules",
				Instructions: "Always use gofmt.",
				Paths:        []string{"**/*.go"}, // must be dropped for AG
			},
			"global-safety": {
				Instructions: "Never delete production data.",
			},
		},
		MCP: map[string]ast.MCPConfig{
			"remote-api": {
				Type: "http",
				URL:  "https://mcp.example.com/v1",
			},
		},
		Hooks: ast.HookConfig{
			"PreToolUse": {
				{
					Matcher: "Bash",
					Hooks: []ast.HookHandler{
						{Type: "command", Command: "echo pre"},
					},
				},
			},
		},
	}

	r := antigravity.New()
	out, err := r.Compile(config, "")
	require.NoError(t, err)
	require.NotNil(t, out)

	// ── Rules ─────────────────────────────────────────────────────────────────

	fmtContent, ok := out.Files["rules/formatting.md"]
	require.True(t, ok, "expected rules/formatting.md in output")

	// No frontmatter delimiters
	assert.False(t, strings.HasPrefix(fmtContent, "---"), "AG rule must not start with --- delimiter")

	// Description as heading
	assert.Contains(t, fmtContent, "# Code style rules")

	// No globs/paths/alwaysApply
	assert.NotContains(t, fmtContent, "globs:")
	assert.NotContains(t, fmtContent, "paths:")
	assert.NotContains(t, fmtContent, "alwaysApply:")

	// Body preserved
	assert.Contains(t, fmtContent, "Always use gofmt.")

	safetyContent, ok := out.Files["rules/global-safety.md"]
	require.True(t, ok, "expected rules/global-safety.md in output")
	assert.Contains(t, safetyContent, "Never delete production data.")

	// ── Skill ─────────────────────────────────────────────────────────────────

	skillContent, ok := out.Files["skills/deploy/SKILL.md"]
	require.True(t, ok, "expected skills/deploy/SKILL.md in output")
	assert.Contains(t, skillContent, "name: Deploy Skill")
	assert.Contains(t, skillContent, "description: Handles deployment steps.")
	assert.Contains(t, skillContent, "Run the deploy pipeline.")

	// CC-only fields must be absent
	for _, dropped := range []string{"model:", "effort:", "tools:"} {
		assert.NotContains(t, skillContent, dropped, "CC-only field %q must be absent", dropped)
	}

	// ── Silent skips ──────────────────────────────────────────────────────────

	// Agents must not be emitted
	for path := range out.Files {
		assert.False(t, strings.HasPrefix(path, "agents/"), "agents must not be emitted for AG target, got %q", path)
	}

	// MCP and hooks must not be emitted
	_, hasMCP := out.Files["mcp.json"]
	assert.False(t, hasMCP, "mcp.json must not be emitted for AG target")

	// ── File count ────────────────────────────────────────────────────────────
	// 2 rules + 1 skill = 3 files (agents + MCP + hooks silently skipped)
	assert.Len(t, out.Files, 3, "expected exactly 3 output files (rules + skill only)")
}

// TestAntigravityRenderer_Rule_InstructionsFile_ReadsFromDisk verifies that
// instructions_file: paths are resolved correctly from baseDir.
func TestAntigravityRenderer_Rule_InstructionsFile_ReadsFromDisk(t *testing.T) {
	dir := t.TempDir()
	body := "# Style\n\nAlways write clean code."
	require.NoError(t, os.WriteFile(filepath.Join(dir, "rule-body.md"), []byte(body), 0600))

	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		Rules: map[string]ast.RuleConfig{
			"from-file": {
				Description:      "File-sourced rule",
				InstructionsFile: "rule-body.md",
			},
		},
	}

	out, err := r.Compile(config, dir)
	require.NoError(t, err)

	content := out.Files["rules/from-file.md"]
	// Frontmatter in the source file must be stripped; only body remains
	assert.NotContains(t, content, "---")
	assert.Contains(t, content, "Always write clean code.")
}

// TestAntigravityRenderer_Skill_InstructionsFile_ReadsFromDisk verifies that
// skill bodies sourced from instructions_file: are resolved from baseDir.
func TestAntigravityRenderer_Skill_InstructionsFile_ReadsFromDisk(t *testing.T) {
	dir := t.TempDir()
	body := "---\nname: Old Name\n---\n\nStep 1: read the code.\nStep 2: ship it."
	require.NoError(t, os.WriteFile(filepath.Join(dir, "skill-body.md"), []byte(body), 0600))

	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		Skills: map[string]ast.SkillConfig{
			"from-file": {
				Name:             "File Skill",
				Description:      "Sourced from disk",
				InstructionsFile: "skill-body.md",
			},
		},
	}

	out, err := r.Compile(config, dir)
	require.NoError(t, err)

	content := out.Files["skills/from-file/SKILL.md"]
	// Frontmatter from the source file must be stripped
	assert.Contains(t, content, "Step 1: read the code.")
	assert.Contains(t, content, "Step 2: ship it.")
	// The skill's own frontmatter must use the AST values, not the file's frontmatter
	assert.Contains(t, content, "name: File Skill")
	assert.NotContains(t, content, "Old Name")
}

// TestAntigravityRenderer_Rule_InstructionsFile_TraversalRejected verifies that
// instructions_file: values that traverse above the project root are rejected.
func TestAntigravityRenderer_Rule_InstructionsFile_TraversalRejected(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		Rules: map[string]ast.RuleConfig{
			"bad-rule": {
				InstructionsFile: "../../../etc/passwd",
			},
		},
	}

	_, err := r.Compile(config, "/tmp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "traverses above the project root")
}
