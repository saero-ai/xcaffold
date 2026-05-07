package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

// setupAndChdir changes into dir and restores CWD on test cleanup.
func setupAndChdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(orig) })
}

// writeFile creates a file at path with content, creating parent dirs.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

// ─── Gap 1: Claude Code — root .mcp.json ─────────────────────────────────────

// TestImportScope_Claude_ReadsMCPJson verifies that importScope reads the
// project-root .mcp.json (sibling to .claude/) and imports MCP server entries
// into the generated project.xcaf.
func TestImportScope_Claude_ReadsMCPJson(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()
	setupAndChdir(t, tmp)

	// Minimal .claude/agents/ so importScope has content
	writeFile(t, filepath.Join(tmp, ".claude", "agents", "dev.md"), "# Dev\n")

	// Root-level .mcp.json (sibling to .claude/). This is how Claude Code
	// stores project-scoped MCP servers per provider documentation.
	mcpJSON := `{
  "mcpServers": {
    "playwright": {
      "command": "npx",
      "args": ["@playwright/mcp@latest"]
    }
  }
}`
	writeFile(t, filepath.Join(tmp, ".mcp.json"), mcpJSON)

	require.NoError(t, importScope(".claude", "project.xcaf", "project", "claude"))

	// In split-file format, MCP servers go to xcaf/mcp/<name>/mcp.xcaf.
	// Ref lists are no longer maintained in project.xcaf; resources are discovered from xcaf/ directory.
	scaffoldData, err := os.ReadFile("project.xcaf")
	require.NoError(t, err)
	assert.NotContains(t, string(scaffoldData), "mcp:", "mcp ref lists are no longer in project.xcaf")

	// The full server config (command, args) goes in xcaf/mcp/playwright/mcp.xcaf.
	mcpXcaf, err := os.ReadFile(filepath.Join(tmp, "xcaf", "mcp", "playwright", "mcp.xcaf"))
	require.NoError(t, err, "xcaf/mcp/playwright/mcp.xcaf must be created")
	assert.Contains(t, string(mcpXcaf), "npx", "MCP server command must appear in xcaf/mcp/playwright/mcp.xcaf")
}

// ─── Gap 2: Claude Code — agent-memory auto-snapshot ────────────────────────

// TestImportScope_Claude_AgentMemoryAutoSnapshot verifies that importScope
// automatically snapshots .claude/agent-memory/<agent>/ into xcaf/agents/<id>/memory/
// without requiring the --with-memory flag.
func TestImportScope_Claude_AgentMemoryAutoSnapshot(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()
	setupAndChdir(t, tmp)

	// Minimal .claude/agents/
	writeFile(t, filepath.Join(tmp, ".claude", "agents", "dev.md"), "# Dev\n")

	// .claude/agent-memory/dev/ — project-local agent memory store
	memIndexContent := "# Dev Agent Memory\n\n- [note](note.md) — a helpful note\n"
	writeFile(t, filepath.Join(tmp, ".claude", "agent-memory", "dev", "MEMORY.md"), memIndexContent)
	writeFile(t, filepath.Join(tmp, ".claude", "agent-memory", "dev", "note.md"), "some useful note")

	require.NoError(t, importScope(".claude", "project.xcaf", "project", "claude"))

	// xcaf/agents/dev/memory/ must have been created with plain .md files.
	// MEMORY.md is an auto-generated index and must be skipped on import.
	assert.DirExists(t, filepath.Join(tmp, "xcaf", "agents", "dev", "memory"),
		"xcaf/agents/dev/memory/ must be created from .claude/agent-memory/dev/")
	assert.NoFileExists(t, filepath.Join(tmp, "xcaf", "agents", "dev", "memory", "MEMORY.md"),
		"MEMORY.md is auto-generated and must NOT be imported as a memory entry")
	assert.FileExists(t, filepath.Join(tmp, "xcaf", "agents", "dev", "memory", "note.md"),
		"note.md must be written into xcaf/agents/dev/memory/")
}

// ─── Gap 3: Claude Code — CLAUDE.md project instructions ────────────────────

// TestImportScope_Claude_ProjectContexts verifies that importScope captures
// the root CLAUDE.md and writes a sidecar at xcaf/instructions/root.md, and
// that project.xcaf references it via project.instructions-file.
func TestImportScope_Gemini_SettingsAndMCP(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()
	setupAndChdir(t, tmp)

	// Minimal .gemini/ structure
	writeFile(t, filepath.Join(tmp, ".gemini", "rules", "style.md"), "# Style\n\nUse 2-space indent.\n")

	// .gemini/settings.json with mcpServers — this is how Gemini CLI stores
	// project-scope MCP per provider documentation.
	geminiSettings := `{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"]
    }
  }
}`
	writeFile(t, filepath.Join(tmp, ".gemini", "settings.json"), geminiSettings)

	require.NoError(t, importScope(".gemini", "project.xcaf", "project", "gemini"))

	// In split-file format, MCP servers go to xcaf/mcp/<name>/mcp.xcaf.
	// Ref lists are no longer maintained in project.xcaf; resources are discovered from xcaf/ directory.
	scaffoldData, err := os.ReadFile("project.xcaf")
	require.NoError(t, err)
	assert.NotContains(t, string(scaffoldData), "mcp:", "mcp ref lists are no longer in project.xcaf")

	// Full config in xcaf/mcp/github/mcp.xcaf.
	mcpXcaf, err := os.ReadFile(filepath.Join(tmp, "xcaf", "mcp", "github", "mcp.xcaf"))
	require.NoError(t, err, "xcaf/mcp/github/mcp.xcaf must be created")
	assert.Contains(t, string(mcpXcaf), "-y", "MCP server args must appear in xcaf/mcp/github/mcp.xcaf")
}

// TestImportScope_Gemini_ProjectContexts verifies that importScope for a
// .gemini/ directory captures the root GEMINI.md into xcaf/instructions/root.md.
func TestImportScope_Cursor_MCPJson(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()
	setupAndChdir(t, tmp)

	writeFile(t, filepath.Join(tmp, ".cursor", "rules", "style.mdc"), "Use 2-space indent.\n")

	// .cursor/mcp.json — Cursor's project-scope MCP config per provider documentation.
	cursorMCP := `{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
    }
  }
}`
	writeFile(t, filepath.Join(tmp, ".cursor", "mcp.json"), cursorMCP)

	require.NoError(t, importScope(".cursor", "project.xcaf", "project", "cursor"))

	// In split-file format, MCP servers go to xcaf/mcp/<name>/mcp.xcaf.
	// Ref lists are no longer maintained in project.xcaf; resources are discovered from xcaf/ directory.
	scaffoldData, err := os.ReadFile("project.xcaf")
	require.NoError(t, err)
	assert.NotContains(t, string(scaffoldData), "mcp:", "mcp ref lists are no longer in project.xcaf")

	mcpXcaf, err := os.ReadFile(filepath.Join(tmp, "xcaf", "mcp", "filesystem", "mcp.xcaf"))
	require.NoError(t, err, "xcaf/mcp/filesystem/mcp.xcaf must be created")
	assert.Contains(t, string(mcpXcaf), "server-filesystem", "MCP server command must appear in xcaf/mcp/filesystem/mcp.xcaf")
}

// TestImportScope_Cursor_HooksJson verifies that importScope for a .cursor/
// directory reads .cursor/hooks.json and imports hook entries.
func TestImportScope_Cursor_HooksJson(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()
	setupAndChdir(t, tmp)

	writeFile(t, filepath.Join(tmp, ".cursor", "rules", "style.mdc"), "Use 2-space indent.\n")

	// .cursor/hooks.json — Cursor's hook config. Shape matches xcaffold HookConfig.
	cursorHooks := `{
  "PreToolUse": [
    {
      "matcher": "Bash",
      "hooks": [
        {"type": "command", "command": "echo pre-tool"}
      ]
    }
  ]
}`
	writeFile(t, filepath.Join(tmp, ".cursor", "hooks.json"), cursorHooks)

	require.NoError(t, importScope(".cursor", "project.xcaf", "project", "cursor"))

	// Hooks are written to xcaf/hooks/<name>/hooks.xcaf in split-file format.
	hooksXcaf, err := os.ReadFile(filepath.Join(tmp, "xcaf", "hooks", "default", "hooks.xcaf"))
	require.NoError(t, err, "xcaf/hooks/default/hooks.xcaf must be created from .cursor/hooks.json")
	assert.Contains(t, string(hooksXcaf), "PreToolUse", "hooks must be imported from .cursor/hooks.json")
}

// TestImportScope_Cursor_ProjectContexts verifies that importScope for a
// .cursor/ directory captures the root AGENTS.md into xcaf/instructions/root.md.
func TestImportScope_Antigravity_NoMemoryCrash(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()
	setupAndChdir(t, tmp)

	writeFile(t, filepath.Join(tmp, ".agents", "rules", "style.md"), "# Style\n")

	require.NoError(t, importScope(".agents", "project.xcaf", "project", "antigravity"))

	// xcaf/agents/ must NOT exist (Antigravity imports rules only, no agent
	// memory dirs should be created).
	_, err := os.Stat(filepath.Join(tmp, "xcaf", "agents"))
	assert.True(t, os.IsNotExist(err),
		"xcaf/agents/ must NOT be created for Antigravity (KIs are app-managed, not filesystem)")
}

// ─── Backward compatibility ───────────────────────────────────────────────────

// TestImportScope_BackwardCompat_ThreeArgSignature verifies that the existing
// 3-arg callers (before the provider parameter was added) still compile and work.
// This is a compile-time check — if it builds, it passes.
// NOTE: This test exercises the new 4-arg signature from the init path.
func TestImportScope_ClaudeWithProvider_DoesNotBreakExistingBehavior(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()
	setupAndChdir(t, tmp)

	writeFile(t, filepath.Join(tmp, ".claude", "agents", "dev.md"),
		"---\nname: dev\ndescription: Dev\n---\n\nDo the work.\n")
	writeFile(t, filepath.Join(tmp, ".claude", "settings.json"), `{"hooks": {}}`)

	// 4-arg call with "claude" must behave identically to old 3-arg behavior
	require.NoError(t, importScope(".claude", "project.xcaf", "project", "claude"))

	data, err := os.ReadFile("project.xcaf")
	require.NoError(t, err)
	assert.Contains(t, string(data), "kind: project")
	// Agent 'dev' will be created in xcaf/agents/dev/agent.xcaf, not listed in project.xcaf ref list
	// Ref lists are no longer maintained in project.xcaf
	assert.NotContains(t, string(data), "agents:")
}

// ─── Gitignore directory filtering ───────────────────────────────────────────

// TestImportScope_GitignoreFilter verifies that standard directories like
// .worktrees and directories matching top-level .gitignore exclusions are
// skipped during project instruction discovery, preventing them from being
// included in project.xcaf or generating absolute paths.
