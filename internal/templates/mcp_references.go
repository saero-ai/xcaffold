package templates

// RenderMCPReference returns an annotated template showing every field
// of the mcp kind with descriptions, types, defaults, and provider support notes.
//
// The generated content is written to .xcaffold/schemas/mcp.xcf.reference
// and is NOT parsed by xcaffold. Users copy fields from this file into their
// xcf/mcp/<name>/<name>.xcf as needed.
//
// Note: MCP kind has no body field and no targets (per-provider overrides).
// Use agent-level mcp-servers: inline map for agent-scoped MCP.
func RenderMCPReference() string {
	return `# ============================================================
# MCP Kind — Full Field Reference
# ============================================================
# This file is NOT parsed by xcaffold.
# Copy fields from here into your xcf/mcp/<name>/<name>.xcf
# Provider support: YES = compiled, dropped = silently removed
# ============================================================
# Note: MCP has NO body and NO targets (per-provider overrides).
# For agent-scoped MCP, use agent-level mcp-servers: inline map.
# ============================================================

kind: mcp
version: "1.0"

# ── Identity ─────────────────────────────────────────────────
# name: yaml-serializable but excluded from JSON output (json:"-").
name: my-mcp-server         # REQUIRED. Lowercase + hyphens. Pattern: ^[a-z0-9-]+$

# ── Connection ───────────────────────────────────────────────
type: stdio                 # Server type. Common values: stdio, sse.

# command: shell command to start the MCP server. Used with stdio type.
command: npx                # Optional. e.g. "npx", "/usr/local/bin/my-server"

# args: arguments passed to the command.
args:                       # Optional.
  - "-y"
  - "@modelcontextprotocol/server-filesystem"
  - "."

# url: URL for remote MCP servers (SSE endpoint).
# url: "https://my-mcp-server.example.com/sse"

# cwd: working directory for the MCP server process.
# cwd: "/path/to/working/dir"

# ── Environment & Headers ────────────────────────────────────
# env: map of environment variable name to value.
# env:
#   MY_API_KEY: "${MY_API_KEY}"
#   DEBUG: "false"

# headers: HTTP headers for remote server connections.
# headers:
#   Authorization: "Bearer ${TOKEN}"
#   X-Custom-Header: "value"

# ── Authentication ───────────────────────────────────────────
# auth-provider-type: authentication provider type for remote servers.
# auth-provider-type: "oauth2"

# oauth: map of OAuth configuration key-value pairs.
# oauth:
#   client-id: "${CLIENT_ID}"
#   client-secret: "${CLIENT_SECRET}"

# ── Control ──────────────────────────────────────────────────
# disabled: pointer type — omitting differs from false.
# When true, the server is registered but not started.
# disabled: true

# disabled-tools: list of tool names to disable on this server.
# disabled-tools:
#   - "dangerous-tool"
#   - "another-tool"
`
}
