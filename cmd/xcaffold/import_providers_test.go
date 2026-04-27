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
// into the generated project.xcf.
func TestImportScope_Claude_ReadsMCPJson(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()
	setupAndChdir(t, tmp)

	// Minimal .claude/agents/ so importScope has content
	writeFile(t, filepath.Join(tmp, ".claude", "agents", "dev.md"), "# Dev\n")

	// Root-level .mcp.json (sibling to .claude/). This is how Claude Code
	// stores project-scoped MCP servers per ground truth.
	mcpJSON := `{
  "mcpServers": {
    "playwright": {
      "command": "npx",
      "args": ["@playwright/mcp@latest"]
    }
  }
}`
	writeFile(t, filepath.Join(tmp, ".mcp.json"), mcpJSON)

	require.NoError(t, importScope(".claude", filepath.Join(".xcaffold", "project.xcf"), "project", "claude"))

	// In split-file format, MCP servers go to xcf/mcp/<name>.xcf.
	// The root project.xcf only lists the MCP server ID in its 'mcp:' field.
	scaffoldData, err := os.ReadFile(filepath.Join(".xcaffold", "project.xcf"))
	require.NoError(t, err)
	assert.Contains(t, string(scaffoldData), "playwright", "project.xcf must list the MCP server ID")

	// The full server config (command, args) goes in xcf/mcp/playwright.xcf.
	mcpXcf, err := os.ReadFile(filepath.Join(tmp, "xcf", "mcp", "playwright.xcf"))
	require.NoError(t, err, "xcf/mcp/playwright.xcf must be created")
	assert.Contains(t, string(mcpXcf), "npx", "MCP server command must appear in xcf/mcp/playwright.xcf")
}

// ─── Gap 2: Claude Code — agent-memory auto-snapshot ────────────────────────

// TestImportScope_Claude_AgentMemoryAutoSnapshot verifies that importScope
// automatically snapshots .claude/agent-memory/<agent>/ into xcf/agents/<id>/memory/
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

	require.NoError(t, importScope(".claude", filepath.Join(".xcaffold", "project.xcf"), "project", "claude"))

	// xcf/agents/dev/memory/ must have been created with plain .md files.
	// MEMORY.md is an auto-generated index and must be skipped on import.
	assert.DirExists(t, filepath.Join(tmp, "xcf", "agents", "dev", "memory"),
		"xcf/agents/dev/memory/ must be created from .claude/agent-memory/dev/")
	assert.NoFileExists(t, filepath.Join(tmp, "xcf", "agents", "dev", "memory", "MEMORY.md"),
		"MEMORY.md is auto-generated and must NOT be imported as a memory entry")
	assert.FileExists(t, filepath.Join(tmp, "xcf", "agents", "dev", "memory", "note.md"),
		"note.md must be written into xcf/agents/dev/memory/")
}

// ─── Gap 3: Claude Code — CLAUDE.md project instructions ────────────────────

// TestImportScope_Claude_ProjectContexts verifies that importScope captures
// the root CLAUDE.md and writes a sidecar at xcf/instructions/root.md, and
// that project.xcf references it via project.instructions-file.
func TestImportScope_Gemini_SettingsAndMCP(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()
	setupAndChdir(t, tmp)

	// Minimal .gemini/ structure
	writeFile(t, filepath.Join(tmp, ".gemini", "rules", "style.md"), "# Style\n\nUse 2-space indent.\n")

	// .gemini/settings.json with mcpServers — this is how Gemini CLI stores
	// project-scope MCP per ground truth.
	geminiSettings := `{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"]
    }
  }
}`
	writeFile(t, filepath.Join(tmp, ".gemini", "settings.json"), geminiSettings)

	require.NoError(t, importScope(".gemini", filepath.Join(".xcaffold", "project.xcf"), "project", "gemini"))

	// In split-file format, MCP servers go to xcf/mcp/<name>.xcf.
	// project.xcf lists only the server ID.
	scaffoldData, err := os.ReadFile(filepath.Join(".xcaffold", "project.xcf"))
	require.NoError(t, err)
	assert.Contains(t, string(scaffoldData), "github", "project.xcf must list the MCP server ID")

	// Full config in xcf/mcp/github.xcf.
	mcpXcf, err := os.ReadFile(filepath.Join(tmp, "xcf", "mcp", "github.xcf"))
	require.NoError(t, err, "xcf/mcp/github.xcf must be created")
	assert.Contains(t, string(mcpXcf), "-y", "MCP server args must appear in xcf/mcp/github.xcf")
}

// TestImportScope_Gemini_ProjectContexts verifies that importScope for a
// .gemini/ directory captures the root GEMINI.md into xcf/instructions/root.md.
func TestImportScope_Cursor_MCPJson(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()
	setupAndChdir(t, tmp)

	writeFile(t, filepath.Join(tmp, ".cursor", "rules", "style.mdc"), "Use 2-space indent.\n")

	// .cursor/mcp.json — Cursor's project-scope MCP config per ground truth.
	cursorMCP := `{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
    }
  }
}`
	writeFile(t, filepath.Join(tmp, ".cursor", "mcp.json"), cursorMCP)

	require.NoError(t, importScope(".cursor", filepath.Join(".xcaffold", "project.xcf"), "project", "cursor"))

	// In split-file format, MCP servers go to xcf/mcp/<name>.xcf.
	scaffoldData, err := os.ReadFile(filepath.Join(".xcaffold", "project.xcf"))
	require.NoError(t, err)
	assert.Contains(t, string(scaffoldData), "filesystem", "project.xcf must list the MCP server ID")

	mcpXcf, err := os.ReadFile(filepath.Join(tmp, "xcf", "mcp", "filesystem.xcf"))
	require.NoError(t, err, "xcf/mcp/filesystem.xcf must be created")
	assert.Contains(t, string(mcpXcf), "server-filesystem", "MCP server command must appear in xcf/mcp/filesystem.xcf")
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

	require.NoError(t, importScope(".cursor", filepath.Join(".xcaffold", "project.xcf"), "project", "cursor"))

	// Hooks are written to xcf/hooks.xcf in split-file format.
	hooksXcf, err := os.ReadFile(filepath.Join(tmp, "xcf", "hooks.xcf"))
	require.NoError(t, err, "xcf/hooks.xcf must be created from .cursor/hooks.json")
	assert.Contains(t, string(hooksXcf), "PreToolUse", "hooks must be imported from .cursor/hooks.json")
}

// TestImportScope_Cursor_ProjectContexts verifies that importScope for a
// .cursor/ directory captures the root AGENTS.md into xcf/instructions/root.md.
func TestImportScope_Antigravity_NoMemoryCrash(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()
	setupAndChdir(t, tmp)

	writeFile(t, filepath.Join(tmp, ".agents", "rules", "style.md"), "# Style\n")

	require.NoError(t, importScope(".agents", filepath.Join(".xcaffold", "project.xcf"), "project", "antigravity"))

	// xcf/agents/ must NOT exist (Antigravity imports rules only, no agent
	// memory dirs should be created).
	_, err := os.Stat(filepath.Join(tmp, "xcf", "agents"))
	assert.True(t, os.IsNotExist(err),
		"xcf/agents/ must NOT be created for Antigravity (KIs are app-managed, not filesystem)")
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
	require.NoError(t, importScope(".claude", filepath.Join(".xcaffold", "project.xcf"), "project", "claude"))

	data, err := os.ReadFile(filepath.Join(".xcaffold", "project.xcf"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "kind: project")
	assert.Contains(t, string(data), "dev")
}

// ─── Gitignore directory filtering ───────────────────────────────────────────

// TestImportScope_GitignoreFilter verifies that standard directories like
// .worktrees and directories matching top-level .gitignore exclusions are
// skipped during project instruction discovery, preventing them from being
// included in project.xcf or generating absolute paths.
