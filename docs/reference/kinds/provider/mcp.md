---
title: "kind: mcp"
description: "Declares a Model Context Protocol server compiled to provider-native JSON configuration files."
---

# `kind: mcp`

Declares a Model Context Protocol (MCP) server. Compiled to provider-native JSON configuration files that register the server when the agent session starts.

> **Required:** `kind`, `version`, `name`, `type`, `command`

## Source Directory

```
xcaf/mcp/<name>/mcp.xcaf
```

Each MCP server declaration is a directory-per-resource under `xcaf/mcp/`. The `.xcaf` file contains the frontmatter for that server.

## Example Usage

### stdio MCP server

```yaml
---
kind: mcp
version: "1.0"
name: browser-tools
type: stdio
command: npx
args:
  - "@agentdeskai/browser-tools-mcp@latest"
---
```

### SSE (remote) MCP server

```yaml
---
kind: mcp
version: "1.0"
name: linear
type: sse
url: "https://mcp.linear.app/sse"
---
```

### stdio server with environment variables

```yaml
---
kind: mcp
version: "1.0"
name: github
type: stdio
command: npx
args:
  - "@modelcontextprotocol/server-github"
env:
  GITHUB_PERSONAL_ACCESS_TOKEN: "${GITHUB_PERSONAL_ACCESS_TOKEN}"
---
```

## Argument Reference

The following arguments are supported:

- `name` — (Required) Unique MCP server identifier. Must match `[a-z0-9-]+`.
- `type` — (Required) Transport protocol: `"stdio"` or `"sse"`.
- `command` — (Required for `type: stdio`) The executable to run (e.g., `npx`, `node`, `python`).
- `args` — (Optional) `[]string`. Arguments passed to `command`.
- `url` — (Required for `type: sse`) The SSE server endpoint URL.
- `env` — (Optional) `map[string]string`. Environment variables injected into the server process. Supports `${VAR_NAME}` references to host environment variables.
- `cwd` — (Optional) `string`. Working directory for the server process. Defaults to the project root when absent.
- `headers` — (Optional) `map[string]string`. HTTP headers sent with each SSE request. Only applicable when `type: sse`.
- `disabled` — (Optional) `*bool`. When `true`, the MCP server is excluded from compiled output for all targets.
- `disabled-tools` — (Optional) `[]string`. Names of tools provided by this server that are excluded from the compiled output.
- `oauth` — (Optional) `map[string]string`. OAuth configuration key-value pairs passed through to the provider-native format.
- `auth-provider-type` — (Optional) `string`. Authentication provider type used for OAuth flows (e.g., `"github"`, `"google"`).
- `targets` — (Optional) `map[string]TargetOverride`. Per-provider overrides. Resources with a `targets:` field are compiled only for the listed providers. When absent, the resource is compiled for all providers that support MCP configuration.

## Compiled Output

<ProviderTabs>
  <ProviderTab id="claude">
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
  </ProviderTab>

  <ProviderTab id="cursor">
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

    > The `name` field is omitted from the Cursor output (it uses the map key as the identifier). Multiple MCP declarations are merged into `.cursor/mcp.json`.
  </ProviderTab>

  <ProviderTab id="copilot">
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

    > Copilot uses VS Code's MCP format with a `"servers"` top-level key (not `"mcpServers"`). Multiple MCP declarations are merged into `.vscode/mcp.json`. A `MCP_GLOBAL_CONFIG_ONLY` fidelity note is emitted because Copilot does not support per-project MCP server activation — the file is written, but servers must be enabled globally in VS Code settings.
  </ProviderTab>

  <ProviderTab id="gemini">
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

    > MCP declarations are merged alongside any other Gemini settings in `.gemini/settings.json`.
  </ProviderTab>

  <ProviderTab id="antigravity">
    No project-local file is written. MCP servers must be configured globally at `~/.gemini/antigravity/mcp_config.json`. A `MCP_GLOBAL_CONFIG_ONLY` fidelity note is emitted at compile time.
  </ProviderTab>
</ProviderTabs>

> [!WARNING]
> **Antigravity** does not support project-local MCP configuration. MCP servers must be registered globally at `~/.gemini/antigravity/mcp_config.json`. A `MCP_GLOBAL_CONFIG_ONLY` fidelity note is emitted at compile time; no project file is written.
>
> **Settings merge**: If your `kind: settings` block also declares `mcp-servers`, those are merged with `kind: mcp` declarations at compile time. The `kind: settings` values win on conflict.
