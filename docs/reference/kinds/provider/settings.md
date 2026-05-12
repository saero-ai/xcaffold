---
title: "kind: settings"
description: "Declares global or workspace-scoped provider settings merged into each provider's configuration at compile time. Produces no output files."
---

# `kind: settings`

Declares provider settings applied globally or scoped to a specific workspace. Settings are merged into provider-native configuration files at compile time. Produces **no additional output files** — the values are integrated into the provider's existing config.

Uses **pure YAML format** (no frontmatter `---` delimiters).

> **Required:** `kind`, `version`

## Source Directory

```
xcaf/settings/<name>/settings.xcaf
```

`kind: settings` is a singleton per name. The `name` field is optional and defaults to `"default"`. Place each named settings file in its own subdirectory: `xcaf/settings/<name>/settings.xcaf`.

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

## Field Reference

### Required Fields

All `kind: settings` fields are optional. The only required top-level keys are `kind` and `version`.

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | `string` | Unique settings identifier. Defaults to `"default"`. Must match `[a-z0-9-]+`. |
| `description` | `string` | Human-readable purpose of this settings block. |
| `model` | `string` | Default model for all sessions. Overridden per-agent by `AgentConfig.model`. |
| `effort-level` | `string` | Default reasoning effort: `"low"`, `"medium"`, `"high"`, `"max"`. |
| `include-git-instructions` | `bool` | Inject git context into the system prompt. |
| `respect-gitignore` | `bool` | Exclude `.gitignore`-matched files from context. |
| `auto-memory-enabled` | `bool` | Enable automatic memory writes after sessions. |
| `auto-memory-directory` | `string` | Directory for auto-written memory files. |
| `skip-dangerous-mode-permission-prompt` | `bool` | Skip the bypass confirmation dialog. |
| `cleanup-period-days` | `int` | How many days to retain session artifacts. |
| `default-shell` | `string` | Shell used for `Bash` tool invocations. |
| `language` | `string` | Response language override. |
| `always-thinking-enabled` | `bool` | Force extended thinking in every session. |
| `available-models` | `[]string` | Models available for selection in the provider UI. |
| `md-excludes` | `[]string` | Glob patterns for paths excluded from root context file scanning. |
| `output-style` | `string` | Output verbosity style. |
| `plans-directory` | `string` | Directory for storing plan files. |
| `otel-headers-helper` | `string` | Helper command for generating OpenTelemetry request headers. |
| `disable-all-hooks` | `bool` | Disable all lifecycle hooks globally for this settings scope. |
| `attribution` | `bool` | Enable commit attribution metadata on agent-authored commits. |
| `disable-skill-shell-execution` | `bool` | Prevent skills from executing shell commands. |
| `sandbox` | `SandboxConfig` | OS-level process isolation for Bash commands. See [sandbox block](#sandbox-block). |
| `status-line` | `StatusLineConfig` | Status line display configuration. See [status-line block](#status-line-block). |
| `enabled-plugins` | `map[string]bool` | Plugin enable/disable flags keyed by plugin name. |
| `env` | `map[string]string` | Environment variables injected into every agent session. |
| `agent` | `any` | Agent configuration object passed through to the provider unchanged. |
| `worktree` | `any` | Worktree configuration object passed through to the provider unchanged. |
| `auto-mode` | `any` | Auto-mode behavior configuration passed through to the provider unchanged. |
| `mcp-servers` | `map[string]MCPConfig` | Inline MCP server declarations. Merged with `kind: mcp` declarations; settings wins on conflict. |
| `hooks` | `HookConfig` | Lifecycle hook definitions for this settings scope. |
| `permissions` | `PermissionsConfig` | Tool permission rules. See [permissions block](#permissions-block). |
| `targets` | `map[string]TargetOverride` | Per-provider overrides. Settings rarely use `targets:` since most settings fields are provider-universal; the field is available for cases where a setting applies only to specific providers. |

### `permissions` block

- `default-mode` — `string`. Execution mode: `"default"`, `"acceptEdits"`, `"auto"`, `"bypassPermissions"`, `"plan"`.
- `allow` — `[]string`. Permitted tool invocations (e.g., `"Bash(pnpm test *)"`, `"Edit(**/src/**)"`)
- `deny` — `[]string`. Forbidden tool invocations.
- `ask` — `[]string`. Tool invocations that require confirmation.
- `additional-directories` — `[]string`. Extra directories the agent can access.
- `disable-bypass-permissions-mode` — `string`. Controls whether bypass-permissions mode can be used. Accepted values are provider-specific (e.g., `"always"` to prevent bypassing).

### `sandbox` block

- `enabled` — `bool`. Enable OS-level process isolation for Bash commands.
- `auto-allow-bash-if-sandboxed` — `bool`. Auto-approve Bash commands without prompting when sandboxed.
- `fail-if-unavailable` — `bool`. Fail if sandboxing is requested but unavailable on the host OS.
- `allow-unsandboxed-commands` — `bool`. Allow commands to run unsandboxed when sandbox is active.
- `excluded-commands` — `[]string`. Commands exempt from sandboxing (e.g., `sudo`, `su`).
- `filesystem.allow-write` — `[]string`. Paths the sandboxed process may write to.
- `filesystem.deny-write` — `[]string`. Paths the sandboxed process may not write to.
- `filesystem.allow-read` — `[]string`. Paths the sandboxed process may read from.
- `filesystem.deny-read` — `[]string`. Paths the sandboxed process may not read from.
- `network.http-proxy-port` — `int`. HTTP proxy port for outbound requests.
- `network.socks-proxy-port` — `int`. SOCKS proxy port for outbound requests.
- `network.allow-managed-domains-only` — `bool`. Restrict outbound traffic to `allowed-domains`.
- `network.allow-unix-sockets` — `[]string`. Unix domain socket paths permitted for outbound connections.
- `network.allow-local-binding` — `bool`. Permit the sandboxed process to bind to localhost ports.
- `network.allowed-domains` — `[]string`. Domains permitted for outbound connections when `allow-managed-domains-only` is true.

### `status-line` block

- `type` — `string`. Status line type (e.g., `"command"`).
- `command` — `string`. Shell command whose output is displayed in the provider status line.

## Compiled Output

Settings are merged at compile time in this order (later wins):

1. Global settings (from `extends:` base config)
2. Project settings (`kind: settings` in `project.xcaf`)
3. `kind: mcp` declarations merged into `mcp-servers` map. For Claude, the merged MCP servers are written to `.claude/mcp.json` — not embedded in `settings.json`.

Each provider serializes the merged result into its native format:

### Claude

```json
// .claude/settings.json
{
  "model": "claude-sonnet-4-5-20250514",
  "effortLevel": "medium",
  "includeGitInstructions": true,
  "respectGitignore": true,
  "autoMemoryEnabled": true,
  "permissions": {
    "defaultMode": "acceptEdits",
    "allow": [
      "Bash(pnpm test *)",
      "Bash(pnpm lint)",
      "Bash(pnpm build)"
    ]
  }
}

// .claude/mcp.json (MCP servers are written here, not in settings.json)
{
  "mcpServers": {
    "linear": {
      "command": "node",
      "args": ["..."]
    }
  }
}
```

### Cursor

```json
{
  "model": "claude-sonnet-4-5-20250514",
  "effortLevel": "medium",
  "mcpServers": {
    "linear": {
      "command": "node",
      "args": ["..."]
    }
  }
}
```

### Copilot

```json
{
  "model": "claude-sonnet-4-5-20250514",
  "permissions": {
    "defaultMode": "acceptEdits",
    "allow": [
      "Bash(pnpm test *)"
    ]
  }
}
```

### Gemini

```json
{
  "model": "claude-sonnet-4-5-20250514",
  "includeGitInstructions": true,
  "respectGitignore": true,
  "mcpServers": {
    "linear": {
      "command": "node",
      "args": ["..."]
    }
  }
}
```

### Antigravity

```json
{
  "model": "claude-sonnet-4-5-20250514",
  "effortLevel": "medium",
  "autoMemoryEnabled": true,
  "mcpServers": {
    "linear": {
      "command": "node",
      "args": ["..."]
    }
  }
}
```
