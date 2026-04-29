---
title: "kind: mcp"
description: "Declares a Model Context Protocol server compiled to provider-native JSON configuration files."
---

# `kind: mcp`

Declares a Model Context Protocol (MCP) server. Compiled to provider-native JSON configuration files that register the server when the agent session starts.

> **Required:** `kind`, `version`, `name`, `type`, `command`

## Source Directory

```
xcf/mcp/<name>/mcp.xcf
```

Each MCP server declaration is a directory-per-resource under `xcf/mcp/`. The `.xcf` file contains the frontmatter for that server.

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
- `targets` — (Optional) `map[string]TargetOverride`. Per-provider overrides. Resources with a `targets:` field are compiled only for the listed providers. When absent, the resource is compiled for all providers that support MCP configuration.

## Compiled Output

<ProviderTabs>
  <ProviderTab id="claude">
    **Output path**: `.claude/.mcp.json`

    ```json
    {
      "mcpServers": {
        "browser-tools": {
          "name": "browser-tools",
          "command": "npx",
          "args": ["@agentdeskai/browser-tools-mcp@latest"]
        }
      }
    }
    ```

    Multiple MCP declarations are merged into a single `.mcp.json`. The `name` field is explicitly included in the Claude output.
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
    > **Target Dropped**: Copilot has no native MCP server configuration format. `MCP_NO_NATIVE_TARGET` fidelity note emitted to stderr. No files are written for this target.
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
    **Output path**: `.agents/settings.json` (merged into `mcpServers` key)

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
  </ProviderTab>
</ProviderTabs>

> [!WARNING]
> **Copilot** does not support MCP server configuration. Declaring an `mcp:` kind with Copilot in `targets` will silently produce no output for that provider and emit a `MCP_NO_NATIVE_TARGET` fidelity note to stderr.
>
> **Settings merge**: If your `kind: settings` block also declares `mcp-servers`, those are merged with `kind: mcp` declarations at compile time. The `kind: settings` values win on conflict.
