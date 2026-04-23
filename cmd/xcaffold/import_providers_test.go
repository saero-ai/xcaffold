package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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
// automatically snapshots .claude/agent-memory/<agent>/ into xcf/memory/
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

	// xcf/memory/dev/ must have been created
	assert.DirExists(t, filepath.Join(tmp, "xcf", "memory", "dev"),
		"xcf/memory/dev/ must be created from .claude/agent-memory/dev/")
	assert.FileExists(t, filepath.Join(tmp, "xcf", "memory", "dev", "MEMORY.xcf"),
		"MEMORY.xcf must be written into xcf/memory/dev/")
	assert.FileExists(t, filepath.Join(tmp, "xcf", "memory", "dev", "note.xcf"),
		"note.xcf must be written into xcf/memory/dev/")
}

// ─── Gap 3: Claude Code — CLAUDE.md project instructions ────────────────────

// TestImportScope_Claude_ProjectInstructions verifies that importScope captures
// the root CLAUDE.md and writes a sidecar at xcf/instructions/root.md, and
// that project.xcf references it via project.instructions-file.
func TestImportScope_Claude_ProjectInstructions(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()
	setupAndChdir(t, tmp)

	writeFile(t, filepath.Join(tmp, ".claude", "agents", "dev.md"), "# Dev\n")
	writeFile(t, filepath.Join(tmp, "CLAUDE.md"), "# Project Rules\n\nDo not expose secrets.\n")

	require.NoError(t, importScope(".claude", filepath.Join(".xcaffold", "project.xcf"), "project", "claude"))

	// xcf/instructions/root.xcf must exist with CLAUDE.md content
	sidecar, err := os.ReadFile(filepath.Join(tmp, "xcf", "instructions", "root.xcf"))
	require.NoError(t, err, "xcf/instructions/root.xcf must be created")
	assert.Contains(t, string(sidecar), "Do not expose secrets")

	// project.xcf is rewritten by runProjectInstructionsDiscovery with WriteProjectFile,
	// which includes instructions-file in the project block.
	xcf, err := os.ReadFile(filepath.Join(".xcaffold", "project.xcf"))
	require.NoError(t, err)
	assert.Contains(t, string(xcf), "instructions-file", "project.xcf must contain instructions-file reference")
}

// ─── Gap 4: Gemini CLI — .gemini/settings.json MCP + project instructions ───

// TestImportScope_Gemini_SettingsAndMCP verifies that importScope for a .gemini/
// directory reads .gemini/settings.json and imports mcpServers entries.
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

// TestImportScope_Gemini_ProjectInstructions verifies that importScope for a
// .gemini/ directory captures the root GEMINI.md into xcf/instructions/root.md.
func TestImportScope_Gemini_ProjectInstructions(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()
	setupAndChdir(t, tmp)

	writeFile(t, filepath.Join(tmp, ".gemini", "rules", "style.md"), "# Style\n")
	writeFile(t, filepath.Join(tmp, "GEMINI.md"), "# Gemini Project Instructions\n\nAlways use structured output.\n")

	require.NoError(t, importScope(".gemini", filepath.Join(".xcaffold", "project.xcf"), "project", "gemini"))

	// xcf/instructions/root.xcf from GEMINI.md
	sidecar, err := os.ReadFile(filepath.Join(tmp, "xcf", "instructions", "root.xcf"))
	require.NoError(t, err, "xcf/instructions/root.xcf must be created from GEMINI.md")
	assert.Contains(t, string(sidecar), "structured output")

	// project.xcf rewritten with instructions-file
	xcf, err := os.ReadFile(filepath.Join(".xcaffold", "project.xcf"))
	require.NoError(t, err)
	assert.Contains(t, string(xcf), "instructions-file")
}

// ─── Gap 5: Cursor — .cursor/mcp.json ───────────────────────────────────────

// TestImportScope_Cursor_MCPJson verifies that importScope for a .cursor/
// directory reads .cursor/mcp.json and imports mcpServers entries.
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

// TestImportScope_Cursor_ProjectInstructions verifies that importScope for a
// .cursor/ directory captures the root AGENTS.md into xcf/instructions/root.md.
func TestImportScope_Cursor_ProjectInstructions(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()
	setupAndChdir(t, tmp)

	writeFile(t, filepath.Join(tmp, ".cursor", "rules", "style.mdc"), "Use 2-space indent.\n")
	writeFile(t, filepath.Join(tmp, "AGENTS.md"), "# Cursor Project Instructions\n\nFollow the style guide.\n")

	require.NoError(t, importScope(".cursor", filepath.Join(".xcaffold", "project.xcf"), "project", "cursor"))

	// xcf/instructions/root.xcf from AGENTS.md
	sidecar, err := os.ReadFile(filepath.Join(tmp, "xcf", "instructions", "root.xcf"))
	require.NoError(t, err, "xcf/instructions/root.xcf must be created from AGENTS.md")
	assert.Contains(t, string(sidecar), "style guide")

	// project.xcf rewritten with instructions-file
	xcf, err := os.ReadFile(filepath.Join(".xcaffold", "project.xcf"))
	require.NoError(t, err)
	assert.Contains(t, string(xcf), "instructions-file")
}

// ─── Gap 6: GitHub Copilot — project instructions ───────────────────────────

// TestImportScope_Copilot_ProjectInstructions verifies that importScope for a
// .github/ directory captures .github/copilot-instructions.md into a sidecar.
func TestImportScope_Copilot_ProjectInstructions(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()
	setupAndChdir(t, tmp)

	// Minimal .github/ with at least one rule so importScope has something
	writeFile(t, filepath.Join(tmp, ".github", "instructions", "style.instructions.md"),
		"---\napplyTo: '**/*.go'\n---\nFollow Go idioms.\n")
	writeFile(t, filepath.Join(tmp, ".github", "copilot-instructions.md"),
		"# Copilot Project Instructions\n\nWrite idiomatic code.\n")

	require.NoError(t, importScope(".github", filepath.Join(".xcaffold", "project.xcf"), "project", "copilot"))

	// xcf/instructions/root.xcf from copilot-instructions.md
	sidecar, err := os.ReadFile(filepath.Join(tmp, "xcf", "instructions", "root.xcf"))
	require.NoError(t, err, "xcf/instructions/root.xcf must be created from copilot-instructions.md")
	assert.Contains(t, string(sidecar), "idiomatic code")

	// project.xcf rewritten with instructions-file
	xcf, err := os.ReadFile(filepath.Join(".xcaffold", "project.xcf"))
	require.NoError(t, err)
	assert.Contains(t, string(xcf), "instructions-file")
}

// ─── Gap 7: Antigravity — no memory crash ────────────────────────────────────

// TestImportScope_Antigravity_NoMemoryCrash verifies that importScope for a
// .agents/ directory completes without error and does NOT create xcf/memory/
// (Antigravity KIs are stored in the AI's app data, not the filesystem).
func TestImportScope_Antigravity_NoMemoryCrash(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()
	setupAndChdir(t, tmp)

	writeFile(t, filepath.Join(tmp, ".agents", "rules", "style.md"), "# Style\n")

	require.NoError(t, importScope(".agents", filepath.Join(".xcaffold", "project.xcf"), "project", "antigravity"))

	// xcf/memory/ must NOT exist (no file-system access to Antigravity KIs)
	_, err := os.Stat(filepath.Join(tmp, "xcf", "memory"))
	assert.True(t, os.IsNotExist(err),
		"xcf/memory/ must NOT be created for Antigravity (KIs are app-managed, not filesystem)")
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
func TestImportScope_GitignoreFilter(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()
	setupAndChdir(t, tmp)

	// Create root CLAUDE.md
	writeFile(t, filepath.Join(tmp, "CLAUDE.md"), "# Root Instructions\n")
	// Create minimal .claude to trigger import
	writeFile(t, filepath.Join(tmp, ".claude", "agents", "dev.md"), "# Dev\n")

	// Create a dummy .worktrees directory with its own CLAUDE.md.
	// This must be ignored.
	writeFile(t, filepath.Join(tmp, ".worktrees", "some-branch", "CLAUDE.md"), "# Branch Instructions\n")

	// Create a custom ignored directory via .gitignore
	writeFile(t, filepath.Join(tmp, ".gitignore"), "node_modules/\ncustom_ignore\n")
	writeFile(t, filepath.Join(tmp, "custom_ignore", "CLAUDE.md"), "# Ignored Custom\n")

	// Create a valid nested CLAUDE.md that SHOULD be included
	writeFile(t, filepath.Join(tmp, "src", "CLAUDE.md"), "# Valid Nested\n")

	require.NoError(t, importScope(".claude", filepath.Join(".xcaffold", "project.xcf"), "project", "claude"))

	// Check the sidecars created
	instructionsDir := filepath.Join(tmp, "xcf", "instructions")

	// root.xcf should exist
	assert.FileExists(t, filepath.Join(instructionsDir, "root.xcf"), "Root CLAUDE.md should be imported")

	// src.md should exist (it's flattened as scopes/src.md or similar, checking for presence in tree)
	foundSrc := false
	foundIgnored := false
	_ = filepath.WalkDir(instructionsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		content, _ := os.ReadFile(path)
		strContent := string(content)
		if strings.Contains(strContent, "Valid Nested") {
			foundSrc = true
		}
		if strings.Contains(strContent, "Branch Instructions") || strings.Contains(strContent, "Ignored Custom") {
			foundIgnored = true
		}
		return nil
	})

	assert.True(t, foundSrc, "Nested CLAUDE.md in src/ should be imported")
	assert.False(t, foundIgnored, "CLAUDE.md in .worktrees or custom_ignore should be skipped")
}
