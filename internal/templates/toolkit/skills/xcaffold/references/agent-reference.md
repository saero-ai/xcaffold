# ============================================================
# Agent Kind — Full Field Reference
# ============================================================
# This file is NOT parsed by xcaffold.
# Copy fields from here into your xcaf/agents/<name>/agent.xcaf
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
tools: [Read, Grep, Glob]   # Allowed tools. Omit to inherit all. Supported: claude, cursor, gemini, copilot, antigravity2.
disallowed-tools: []         # Denylist. Applied before tools allowlist (Claude). Supported: claude, cursor, gemini, copilot, antigravity2.
readonly: false             # Restrict to read-only tools (Cursor/Antigravity2).

# ── Permissions & Invocation Control ─────────────────────────
permission-mode: default     # default | acceptEdits | auto | dontAsk | bypassPermissions | plan (Claude). Not supported by Antigravity2.
disable-model-invocation: false  # Prevent automatic agent selection (Copilot/Cursor). Not supported by Antigravity2.
user-invocable: true         # false = programmatic-only access (Copilot). Supported: claude, cursor, copilot, antigravity2.

# ── Lifecycle ────────────────────────────────────────────────
background: false           # Run as background task (Claude/Cursor). Not supported by Antigravity2.
isolation: ""               # "worktree" for git worktree isolation (Claude). Not supported by Antigravity2.
when: ""                    # Compile-time conditional (xcaffold-specific).

# ── Memory & Context ─────────────────────────────────────────
memory: ""                  # Claude memory scope: user | project | local. Not supported by Antigravity2.
color: ""                   # Display color (Claude): red/blue/green/yellow/purple/orange/pink/cyan. Not supported by Antigravity2.
initial-prompt: ""           # Auto-submitted first turn. Supported: claude, cursor, gemini, antigravity2.

# ── Composition (references) ─────────────────────────────────
skills: []                  # Skills loaded into agent context. Supported: claude, cursor, gemini, copilot, antigravity2.
rules: []                   # Rules applied to this agent. Supported: claude, cursor, gemini, copilot, antigravity2.
mcp: []                     # MCP server references (by name). Not supported by Antigravity2.

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
# targets: keys are provider names. Valid override fields:
#   suppress-fidelity-warnings: bool  — suppress compile-time fidelity warnings
#   skip-synthesis: bool              — skip synthetic field generation
#   provider: map                     — opaque provider-native pass-through (any key/value)
#
# targets:
#   gemini:
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
---
Your agent instructions here. This becomes the body of the compiled
markdown file (e.g., .claude/agents/my-agent.md).

# OR reference an external file (mutually exclusive with body):
# instructions-file: "agents/my-agent.md"
