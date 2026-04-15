package templates

// RenderAgentReference returns an annotated template showing every field
// of the agent kind with descriptions, types, defaults, and provider support notes.
//
// The generated content is written to xcf/references/agent.xcf.reference and
// is NOT parsed by xcaffold. Users copy fields from this file into their
// scaffold.xcf as needed.
func RenderAgentReference() string {
	return `# ============================================================
# Agent Kind — Full Field Reference
# ============================================================
# This file is NOT parsed by xcaffold.
# Copy fields from here into your scaffold.xcf as needed.
# Field definitions follow the xcaffold schema specification.
# ============================================================

---
kind: agent
version: "1.0"

# ── Identity (xcaffold envelope) ─────────────────────────────
name: my-agent              # REQUIRED. Unique ID, lowercase + hyphens.
description: "..."          # Recommended. When to delegate to this agent.

# ── Model & Execution ────────────────────────────────────────
model: sonnet               # Alias (sonnet/opus/haiku) or full model ID.
effort: high                # low | medium | high | max (max = Opus only).
max-turns: 10                # Max agentic turns. Gemini default: 30.
mode: ""                    # Execution mode (xcaffold-specific).

# ── Tool Access ──────────────────────────────────────────────
tools: [Read, Grep, Glob]   # Allowed tools. Omit to inherit all.
disallowed-tools: []         # Denylist. Applied before tools allowlist (Claude).
readonly: false             # Restrict to read-only tools (Cursor/Antigravity).

# ── Permissions & Invocation Control ─────────────────────────
permission-mode: default     # default | acceptEdits | auto | dontAsk | bypassPermissions | plan (Claude).
disable-model-invocation: false  # Prevent automatic agent selection (Copilot/Cursor).
user-invocable: true         # false = programmatic-only access (Copilot).

# ── Lifecycle ────────────────────────────────────────────────
background: false           # Run as background task (Claude/Cursor/Antigravity).
isolation: ""               # "worktree" for git worktree isolation (Claude).
when: ""                    # Compile-time conditional (xcaffold-specific).

# ── Memory & Context ─────────────────────────────────────────
memory: ""                  # Claude memory scope: user | project | local.
color: ""                   # Display color (Claude): red/blue/green/yellow/purple/orange/pink/cyan.
initial-prompt: ""           # Auto-submitted first turn (Claude).

# ── Composition (references) ─────────────────────────────────
skills: []                  # Skills loaded into agent context.
rules: []                   # Rules applied to this agent.
mcp: []                     # MCP server references (by name).
assertions: []              # Test assertions (evaluated by xcaffold test --judge).

# ── Inline Composition ───────────────────────────────────────
# mcp-servers:                 # Inline MCP server definitions.
#   my-server:
#     command: "/path/to/server"
#     args: ["--flag"]

# hooks:                      # Lifecycle hooks scoped to this agent.
#   PreToolUse:
#     - matcher: "Bash"
#       hooks:
#         - type: command
#           command: "validate.sh"

# ── Multi-Target (per-provider overrides) ────────────────────
# targets:
#   gemini:
#     instructions-override: |
#       Alternative body for Gemini only.
#     suppress-fidelity-warnings: false
#     skip-synthesis: false
#     provider:                  # Provider-native pass-through fields.
#       temperature: 0.7         # Gemini: 0.0-2.0, default 1.0
#       timeout_mins: 15         # Gemini: max execution time, default 10
#       kind: local              # Gemini: local | remote (A2A protocol)
#   copilot:
#     provider:
#       target: github-copilot   # Copilot: vscode | github-copilot
#       metadata:                # Copilot: custom name/value annotations
#         category: review

# ── Instructions (always last) ───────────────────────────────
instructions: |
  Your agent instructions here. This becomes the body of the compiled
  markdown file (e.g., .claude/agents/my-agent.md).

# OR reference an external file (mutually exclusive with instructions):
# instructions-file: "agents/my-agent.md"
`
}
