# Binding MCP Tool Servers to Agents

MCP (Model Context Protocol) servers extend agents with external tools — file systems, databases, APIs, and custom executables. In xcaffold, MCP servers are declared in your `.xcf` file and compiled deterministically into the target platform's configuration on every `xcaffold apply` run.

Four compilation targets are supported: `claude`, `cursor`, `antigravity`, and `agentsmd`. This guide covers `claude` and `cursor` in full, with normalization differences called out explicitly.

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

**Fields:**

| Field | Type | Purpose |
|---|---|---|
| `command` | `string` | Executable to launch |
| `args` | `[]string` | Arguments passed to the executable |
| `cwd` | `string` | Working directory for the process |
| `env` | `map[string]string` | Environment variables injected at launch |
| `type` | `string` | Transport hint (`stdio`, `sse`, `http`). Claude target writes this field; Cursor infers it from the presence of `command` vs `url` and omits the field entirely. |

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

**Fields:**

| Field | Type | Purpose |
|---|---|---|
| `url` | `string` | Server endpoint URL |
| `headers` | `map[string]string` | HTTP headers sent with every request |
| `authProviderType` | `string` | Auth provider hint (e.g. `oauth`) |
| `oauth` | `map[string]string` | OAuth configuration key/value pairs |

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

Agent-scoped servers are compiled directly into the agent's output file (the agent's `.md` frontmatter for the `claude` target). They do not appear in `settings.json`. Use this when a server is meaningful only within one agent's context.

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

**Input (`scaffold.xcf`):**

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

The MCP servers are emitted to a separate `mcp.json` file under the `mcpServers` envelope key. Two normalizations apply:

- `url` is renamed to `serverUrl`
- `type` is omitted entirely — Cursor infers the transport from the presence of `command` (stdio) or `serverUrl` (http/sse)

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

**For a remote server**, the `url` → `serverUrl` normalization is visible:

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
      "serverUrl": "https://mcp.example.com/search",
      "headers": {
        "Authorization": "Bearer ${TOKEN}"
      }
    }
  }
}
```

`type` is absent from the Cursor output. `url` becomes `serverUrl`. Fields `cwd`, `authProviderType`, `oauth`, `disabled`, and `disabledTools` are not included in the Cursor `cursorMCPEntry` shape and are dropped on compilation to that target.

---

## Complete `MCPConfig` Field Reference

| Field | Type | Description |
|---|---|---|
| `command` | `string` | Executable path or name for stdio servers |
| `args` | `[]string` | Arguments passed to `command` |
| `cwd` | `string` | Working directory for the process |
| `env` | `map[string]string` | Environment variables |
| `type` | `string` | Transport hint: `stdio`, `sse`, or `http` |
| `url` | `string` | Endpoint URL for remote servers |
| `headers` | `map[string]string` | HTTP request headers |
| `authProviderType` | `string` | Auth provider type (e.g. `oauth`) |
| `oauth` | `map[string]string` | OAuth key/value configuration |
| `disabled` | `*bool` | When `true`, server is inactive but retained in config |
| `disabledTools` | `[]string` | Individual tools to suppress from this server |
