---
title: "kind: settings"
description: "Declares global or workspace-scoped provider settings merged into each provider's configuration at compile time. Produces no output files."
---

# `kind: settings`

Declares provider settings applied globally or scoped to a specific workspace. Settings are merged into provider-native configuration files at compile time. Produces **no additional output files** — the values are integrated into the provider's existing config.

Uses **pure YAML format** (no frontmatter `---` delimiters).

> **Required:** `kind`, `version`, `name`

## Example Usage

### Minimal: set a default model

```yaml
kind: settings
version: "1.0"
name: project-defaults
model: claude-sonnet-4-5-20250514
```

### Full: permissions, hooks, and MCP for frontend-app

```yaml
kind: settings
version: "1.0"
name: frontend-app-settings
model: claude-sonnet-4-5-20250514
effort-level: medium
include-git-instructions: true
respect-gitignore: true
auto-memory-enabled: true
permissions:
  default-mode: acceptEdits
  allow:
    - "Bash(pnpm test *)"
    - "Bash(pnpm lint)"
    - "Bash(pnpm build)"
    - "Bash(git status)"
    - "Bash(git diff *)"
  deny:
    - "Bash(rm -rf *)"
    - "Bash(git push --force *)"
mcp-servers:
  linear:
    type: sse
    url: "https://mcp.linear.app/sse"
```

## Argument Reference

The following arguments are supported:

- `name` — (Required) Unique settings identifier. Must match `[a-z0-9-]+`.
- `model` — (Optional) `string`. Default model for all sessions. Overridden per-agent by `AgentConfig.model`.
- `effort-level` — (Optional) `string`. Default effort: `"low"`, `"medium"`, `"high"`, `"max"`.
- `include-git-instructions` — (Optional) `bool`. Inject git context into the system prompt.
- `respect-gitignore` — (Optional) `bool`. Exclude `.gitignore`-matched files from context.
- `auto-memory-enabled` — (Optional) `bool`. Enable automatic memory writes after sessions.
- `auto-memory-directory` — (Optional) `string`. Directory for auto-written memory files.
- `skip-dangerous-mode-permission-prompt` — (Optional) `bool`. Skip the bypass confirmation dialog.
- `cleanup-period-days` — (Optional) `int`. How many days to retain session artifacts.
- `default-shell` — (Optional) `string`. Shell used for `Bash` tool invocations (e.g., `"bash"`, `"zsh"`).
- `language` — (Optional) `string`. Response language override.
- `always-thinking-enabled` — (Optional) `bool`. Force extended thinking in every session.
- `available-models` — (Optional) `[]string`. Models available for selection in the provider UI.
- `claude-md-excludes` — (Optional) `[]string`. Glob patterns for files excluded from `CLAUDE.md` auto-inclusion.
- `mcp-servers` — (Optional) `map[string]MCPConfig`. Inline MCP server declarations. Merged with `kind: mcp` declarations; settings wins on conflict.
- `permissions` — (Optional) `PermissionsConfig` (see [permissions block](#permissions-block)).

### `permissions` block

- `default-mode` — `string`. Execution mode: `"default"`, `"acceptEdits"`, `"auto"`, `"bypassPermissions"`, `"plan"`.
- `allow` — `[]string`. Permitted tool invocations (e.g., `"Bash(pnpm test *)"`, `"Edit(**/src/**)"`)
- `deny` — `[]string`. Forbidden tool invocations.
- `ask` — `[]string`. Tool invocations that require confirmation.
- `additional-directories` — `[]string`. Extra directories the agent can access.

## Compiled Output

Settings are merged at compile time in this order (later wins):

1. Global settings (from `extends:` base config)
2. Project settings (`kind: settings` in `project.xcf`)
3. `kind: mcp` declarations merged into `mcp-servers` map

Each provider serializes the merged result into its native format:

<ProviderTabs
  claude={`{
  "model": "claude-sonnet-4-5-20250514",
  "effort_level": "medium",
  "include_git_instructions": true,
  "respect_gitignore": true,
  "auto_memory_enabled": true,
  "permissions": {
    "default_mode": "acceptEdits",
    "allow": [
      "Bash(pnpm test *)",
      "Bash(pnpm lint)",
      "Bash(pnpm build)"
    ]
  },
  "mcpServers": {
    "linear": {
      "command": "node",
      "args": ["..."]
    }
  }
}`}
  cursor={`{
  "model": "claude-sonnet-4-5-20250514",
  "effortLevel": "medium",
  "mcpServers": {
    "linear": {
      "command": "node",
      "args": ["..."]
    }
  }
}`}
  github={`{
  "model": "claude-sonnet-4-5-20250514",
  "permissions": {
    "defaultMode": "acceptEdits",
    "allow": [
      "Bash(pnpm test *)"
    ]
  }
}`}
  gemini={`{
  "model": "claude-sonnet-4-5-20250514",
  "includeGitInstructions": true,
  "respectGitignore": true,
  "mcpServers": {
    "linear": {
      "command": "node",
      "args": ["..."]
    }
  }
}`}
  antigravity={`{
  "model": "claude-sonnet-4-5-20250514",
  "effortLevel": "medium",
  "autoMemoryEnabled": true,
  "mcpServers": {
    "linear": {
      "command": "node",
      "args": ["..."]
    }
  }
}`}
/>
