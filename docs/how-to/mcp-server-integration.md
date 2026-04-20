---
title: "Binding MCP Tool Servers to Agents"
description: "Define stdio, SSE, and HTTP MCP servers and reference them from agents in .xcf files"
---

# Binding MCP Tool Servers to Agents

You want to give your agents access to external tools — databases, file systems, APIs — without hard-coding tool implementations into agent instructions. MCP (Model Context Protocol) servers extend agents with external tools and are declared in your `.xcf` file, compiled deterministically into the target platform's configuration on every `xcaffold apply` run.

Five compilation targets are supported: `claude`, `cursor`, `antigravity`, `copilot`, and `gemini`. This guide covers `claude` and `cursor` in full, with normalization differences called out explicitly.

**When to use this:** When you need to give agents structured access to external tools, and want that wiring tracked as source under xcaffold management.

**Prerequisites:**
- Completed [Getting Started](../tutorials/getting-started.md) tutorial

---

## Defining a Local stdio Server

A stdio server launches a local process over standard input/output. Set `command` and `args`; xcaffold infers the transport type.

```yaml
version: "1.0"
project:
  name: my-project

mcp:
  filesystem:
    type: stdio
    command: npx
    args:
      - -y
      - "@modelcontextprotocol/server-filesystem"
      - /workspace
    cwd: /workspace
    env:
      NODE_ENV: production
```

For the full field reference, see [Schema Reference: MCPConfig](../reference/schema.md#mcpconfig).

---

## Defining a Remote HTTP/SSE Server

A remote server communicates over HTTP or SSE. Set `url` and, if required, `headers` and `authProviderType`.

```yaml
mcp:
  web-search:
    type: sse
    url: https://mcp.example.com/search
    headers:
      Authorization: "Bearer ${MCP_SEARCH_TOKEN}"
    authProviderType: oauth
    oauth:
      client_id: "${OAUTH_CLIENT_ID}"
      client_secret: "${OAUTH_CLIENT_SECRET}"
```

---

## Standalone MCP Documents (`kind: mcp`)

In a multi-kind project, each MCP server can be defined in its own `.xcf` file using `kind: mcp`. The `MCPConfig` fields are inlined at the top level (not nested under an `mcp:` key):

```yaml
kind: mcp
version: "1.0"
name: filesystem
type: stdio
command: npx
args:
  - -y
  - "@modelcontextprotocol/server-filesystem"
  - /workspace
```

Place these files under `xcf/mcp/` (e.g., `xcf/mcp/filesystem.xcf`). `ParseDirectory` discovers them automatically and merges them into the project configuration. Each `kind: mcp` document defines one server; the `name:` field becomes the server ID. Duplicate IDs across files produce a parse error.

---

## The Three Definition Locations

xcaffold provides three places to define MCP servers. Each has a distinct scope.

### 1. Top-Level `mcp:` (Global Shorthand)

```yaml
mcp:
  filesystem:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem", /workspace]
```

Servers defined here are merged into `settings.mcpServers` during compilation. This is the primary declaration point for project-wide servers.

### 2. `settings.mcpServers` (Settings Block)

```yaml
settings:
  mcpServers:
    filesystem:
      command: npx
      args: ["-y", "@modelcontextprotocol/server-filesystem", /workspace]
```

Servers defined in `settings.mcpServers` take precedence over top-level `mcp:` on key conflicts. When both declare the same server ID, the `settings.mcpServers` definition wins and overwrites the top-level entry.

### 3. `agents.<id>.mcpServers` (Agent-Scoped Inline)

```yaml
agents:
  data-analyst:
    model: claude-opus-4-5
    mcpServers:
      local-db:
        command: /usr/local/bin/db-mcp
        args: ["--dsn", "${DATABASE_URL}"]
```

xcaffold accepts MCP servers scoped to individual agents in the `.xcf` AST. Compilation output depends on the target provider — verify that the target provider supports agent-level MCP configuration before using this feature.

---

## Conflict Resolution

When the same server ID appears in both `mcp:` and `settings.mcpServers`, `settings.mcpServers` wins. The compiler merges them in this order:

```
mcpShorthand (mcp:) → then overwritten by settings.MCPServers
```

This mirrors the behavior in `compileSettingsJSON` in `internal/renderer/claude/claude.go`:

```go
for k, v := range mcpShorthand {
    mcpServers[k] = v
}
for k, v := range settings.MCPServers {
    mcpServers[k] = v  // settings wins on conflict
}
```

---

## Referencing MCP Servers from Agents

Use `agents.<id>.mcp` to attach globally-declared MCP servers to a specific agent. The value is a list of top-level `mcp:` server IDs.

```yaml
mcp:
  filesystem:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem", /workspace]
  web-search:
    type: sse
    url: https://mcp.example.com/search

agents:
  researcher:
    model: claude-opus-4-5
    mcp:
      - filesystem
      - web-search
    instructions: You are a research agent.
```

**Cross-reference validation is enforced at parse time.** Referencing an undefined server ID is a hard parse error:

```
agent "researcher" references undefined mcp server "web-search"
```

The parser (`internal/parser/parser.go`, `validateCrossReferences`) checks every entry in `agent.mcp` against the top-level `mcp:` map before compilation proceeds.

---

## Disabling a Server

Set `disabled: true` to deactivate a server without removing its definition. This preserves the configuration for environments where the server may be conditionally enabled.

```yaml
mcp:
  local-db:
    command: /usr/local/bin/db-mcp
    disabled: true
```

The `disabled` field is a `*bool`. Omitting it leaves the server active.

---

## Suppressing Specific Tools

Use `disabledTools` to suppress individual tools exposed by a server, without disabling the server entirely.

```yaml
mcp:
  filesystem:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem", /workspace]
    disabledTools:
      - write_file
      - delete_file
```

`disabledTools` is a `[]string`. Tools named here are excluded from the server's advertised tool list at runtime.

---

## Visualizing with `xcaffold graph`

`xcaffold graph` renders MCP server nodes and their edges to agents in the terminal topology view.

```
xcaffold graph --full
```

Example output (terminal format):

```
┌──────────────────────────────────────────────────────────┐
│  my-project  •  2 agents  •  2 mcp servers               │
└──────────────────────────────────────────────────────────┘

  [ AGENTS ]
  ● researcher [claude-opus-4-5]
      │
      ├─▶ [Skills]
      │    └─▶ research-skill
      │
      └─▶ [Servers]
               └─(mcp)─▶ filesystem
               └─(mcp)─▶ web-search
```

MCP server nodes appear in the graph as `(mcp)-> server-name` edges from the agents that reference them. Unreferenced servers are reported in the orphan section:

```
  Unreferenced mcp:     local-db
```

Use `--format mermaid` or `--format dot` to produce embeddable or rendered graph output:

```
xcaffold graph --format mermaid > topology.md
xcaffold graph --format dot | dot -Tsvg > topology.svg
```

---

## Dual-Target Output Comparison

The same `mcp:` declaration compiles to different output files and shapes depending on the target.

**Input (`project.xcf`):**

```yaml
version: "1.0"
project:
  name: my-project

mcp:
  filesystem:
    type: stdio
    command: npx
    args:
      - -y
      - "@modelcontextprotocol/server-filesystem"
      - /workspace
    env:
      NODE_ENV: production
```

### Claude Target (`--target claude`)

Output: `.claude/mcp.json`

The top-level `mcp:` block is written to a standalone `mcp.json` file wrapped in an `{"mcpServers": {...}}` envelope. The `type` field is included verbatim. No `$schema` key is emitted in `mcp.json` — the schema key lives in `settings.json`.

```json
{
  "mcpServers": {
    "filesystem": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/workspace"],
      "env": {
        "NODE_ENV": "production"
      }
    }
  }
}
```

### Cursor Target (`--target cursor`)

Output: `.cursor/mcp.json`

The MCP servers are emitted to a separate `mcp.json` file under the `mcpServers` envelope key. Cursor ground truth shows that remote servers use the `url` field name in mcp.json configuration. xcaffold's output may normalize field names — verify the actual compiled output matches your Cursor configuration expectations.

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/workspace"],
      "env": {
        "NODE_ENV": "production"
      }
    }
  }
}
```

**For a remote server**, compare the xcaffold input to the compiled Claude and Cursor outputs:

`.xcf` input:
```yaml
mcp:
  web-search:
    type: sse
    url: https://mcp.example.com/search
    headers:
      Authorization: "Bearer ${TOKEN}"
```

Claude output (`.claude/mcp.json`):
```json
{
  "mcpServers": {
    "web-search": {
      "type": "sse",
      "url": "https://mcp.example.com/search",
      "headers": {
        "Authorization": "Bearer ${TOKEN}"
      }
    }
  }
}
```

Cursor output (`.cursor/mcp.json`):
```json
{
  "mcpServers": {
    "web-search": {
      "url": "https://mcp.example.com/search",
      "headers": {
        "Authorization": "Bearer ${TOKEN}"
      }
    }
  }
}
```


Fields `cwd`, `authProviderType`, `oauth`, `disabled`, and `disabledTools` are xcaffold-specific and are dropped when compiling to the Cursor target, as Cursor's mcp.json schema does not include them.

---

## Verification

After declaring your servers and running `xcaffold apply`, verify the connections with:

```bash
xcaffold graph --full
```

Expected output includes each declared server as an edge from the agent that references it:

```
  └─▶ [Servers]
           └─(mcp)─▶ filesystem
```

Any server not referenced by an agent appears in the `Unreferenced mcp:` section. To verify the compiled JSON shape, inspect the output file directly:

```bash
cat .claude/mcp.json
```

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `agent "X" references undefined mcp server "Y"` | Server declared in `mcp:` but agent's `mcp:` list uses a different ID | Ensure the ID in `agents.<id>.mcp` matches the key in the top-level `mcp:` map exactly |
| Cursor output missing `serverUrl` | Remote server declared with `url:` but targeting `claude` — no normalization needed | For Cursor, xcaffold renames `url` to `serverUrl` automatically; verify `--target cursor` is set |
| Server appears in graph but not in compiled output | `disabled: true` is set on the server | Remove `disabled` or set `disabled: false` |
| Duplicate server ID parse error | Same server key defined in both `mcp:` and a `kind: mcp` standalone document | Remove one definition or rename the duplicate |

---

## Related

- [Architecture Overview](../concepts/architecture.md) — compilation pipeline from `.xcf` to target output
- [Schema Reference: MCPConfig](../reference/schema.md#mcpconfig) — full field table for all MCP server fields
- [CLI Reference: xcaffold graph](../reference/cli.md#xcaffold-graph)
- [Splitting a Project Into Multiple .xcf Files](multi-file-projects.md) — organizing MCP servers in standalone `kind: mcp` documents
