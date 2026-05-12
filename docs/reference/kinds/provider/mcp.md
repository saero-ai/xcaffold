---
title: "kind: mcp"
description: "Declares a Model Context Protocol server compiled to provider-native JSON configuration files."
---

# `kind: mcp`

Declares a Model Context Protocol (MCP) server. Compiled to provider-native JSON configuration files that register the server when the agent session starts.

> **Required:** `kind`, `version`, `name`

## Source Directory

```
xcaf/mcp/<name>/mcp.xcaf
```

Each MCP server declaration is a directory-per-resource under `xcaf/mcp/`. The `.xcaf` file contains the YAML configuration for that server.

## Example Usage

### stdio MCP server

```yaml
kind: mcp
version: "1.0"
name: browser-tools
type: stdio
command: npx
args:
  - "@agentdeskai/browser-tools-mcp@latest"
```

### SSE (remote) MCP server

```yaml
kind: mcp
version: "1.0"
name: linear
type: sse
url: "https://mcp.linear.app/sse"
```

### stdio server with environment variables

```yaml
kind: mcp
version: "1.0"
name: github
type: stdio
command: npx
args:
  - "@modelcontextprotocol/server-github"
env:
  GITHUB_PERSONAL_ACCESS_TOKEN: "${GITHUB_PERSONAL_ACCESS_TOKEN}"
```

## Field Reference

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | `string` | Unique MCP server identifier. Must match `[a-z0-9-]+`. |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `description` | `string` | Human-readable purpose of this MCP server. |
| `type` | `string` | Server transport protocol: `"stdio"` or `"sse"`. Typically required — determines the transport protocol. |
| `command` | `string` | Shell command to start the MCP server process (required when `type: stdio`). |
| `args` | `[]string` | Arguments passed to the server command. |
| `url` | `string` | The SSE server endpoint URL (required when `type: sse`). |
| `env` | `map[string]string` | Environment variables injected into the server process. Supports `${VAR_NAME}` references to host environment variables. |
| `cwd` | `string` | Working directory for the server process. Defaults to the project root when absent. |
| `headers` | `map[string]string` | HTTP headers sent with each SSE request. Only applicable when `type: sse`. |
| `disabled` | `bool` | When `true`, the MCP server is excluded from compiled output for all targets. |
| `disabled-tools` | `[]string` | Names of tools provided by this server that are excluded from the compiled output. |
| `oauth` | `map[string]string` | OAuth configuration key-value pairs passed through to the provider-native format. |
| `auth-provider-type` | `string` | Authentication provider type used for OAuth flows. Passed through to provider-native format. |
| `targets` | `map[string]TargetOverride` | Per-provider overrides. When present, the resource is compiled only for listed providers. |

## Compiled Output

### Claude

**Output path**: `.claude/mcp.json`

```json
{
  "mcpServers": {
    "browser-tools": {
      "command": "npx",
      "args": ["@agentdeskai/browser-tools-mcp@latest"]
    }
  }
}
```

Multiple MCP declarations are merged into a single `.claude/mcp.json`.

### Cursor

**Output path**: `.cursor/mcp.json`

```json
{
  "mcpServers": {
    "browser-tools": {
      "command": "npx",
      "args": ["@agentdeskai/browser-tools-mcp@latest"]
    }
  }
}
```

The `name` field is omitted from the Cursor output (it uses the map key as the identifier). Multiple MCP declarations are merged into `.cursor/mcp.json`.

### Copilot

**Output path**: `.vscode/mcp.json`

```json
{
  "servers": {
    "browser-tools": {
      "command": "npx",
      "args": ["@agentdeskai/browser-tools-mcp@latest"]
    }
  }
}
```

Copilot uses VS Code's MCP format with a `"servers"` top-level key (not `"mcpServers"`). Multiple MCP declarations are merged into `.vscode/mcp.json`. A `MCP_GLOBAL_CONFIG_ONLY` fidelity note is emitted. The file is written to `.vscode/mcp.json`, but Copilot requires servers to also be registered via `github.copilot.chat.mcp` in VS Code's global `settings.json` to activate them.

### Gemini

**Output path**: `.gemini/settings.json` (merged into `mcpServers` key)

```json
{
  "mcpServers": {
    "browser-tools": {
      "command": "npx",
      "args": ["@agentdeskai/browser-tools-mcp@latest"]
    }
  }
}
```

MCP declarations are merged alongside any other Gemini settings in `.gemini/settings.json`.

### Antigravity

No project-local file is written. MCP servers must be configured globally at `~/.gemini/antigravity/mcp_config.json`. A `MCP_GLOBAL_CONFIG_ONLY` fidelity note is emitted at compile time.

> [!WARNING]
> **Antigravity** does not support project-local MCP configuration. MCP servers must be registered globally at `~/.gemini/antigravity/mcp_config.json`. A `MCP_GLOBAL_CONFIG_ONLY` fidelity note is emitted at compile time; no project file is written.
>
> **Settings merge**: If your `kind: settings` block also declares `mcp-servers`, those are merged with `kind: mcp` declarations at compile time. The `kind: settings` values win on conflict.
