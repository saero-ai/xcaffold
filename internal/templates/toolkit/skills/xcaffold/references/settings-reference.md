# ============================================================
# Settings Kind — Full Field Reference
# ============================================================
# This file is NOT parsed by xcaffold.
# Copy fields from here into your xcaf/settings.xcaf
# Settings is a SINGLETON kind — place at xcaf/ root, not in a subdirectory.
# Format: pure YAML (no --- frontmatter delimiters, no body).
# Most fields are Claude-specific; non-Claude providers drop unsupported fields.
# ============================================================

kind: settings
version: "1.0"

# ── Identity ─────────────────────────────────────────────────
name: default               # Optional. Defaults to "default" when omitted.
description: "..."          # Optional.

# ── Model & Execution ────────────────────────────────────────
# model: claude-sonnet-4-6  # Model alias or full ID.
# effort-level: high        # low | medium | high | max
# language: en              # Response language.
# output-style: concise     # Output style preference.
# default-shell: /bin/zsh   # Shell for Bash tool.
# always-thinking-enabled: false

# ── Memory ───────────────────────────────────────────────────
# auto-memory-enabled: true
# auto-memory-directory: .xcaf/memory

# ── Maintenance ──────────────────────────────────────────────
# cleanup-period-days: 30
# include-git-instructions: true
# respect-gitignore: true
# skip-dangerous-mode-permission-prompt: false

# ── Hooks & Execution Control ────────────────────────────────
# disable-all-hooks: false
# disable-skill-shell-execution: false
# attribution: true

# ── Directories ──────────────────────────────────────────────
# plans-directory: docs/plans

# ── Platform Behavior (any value accepted) ───────────────────
# agent: claude
# worktree: true
# auto-mode: supervised

# ── Environment ──────────────────────────────────────────────
# env:
#   MY_VAR: "value"

# ── Inline MCP Servers (same structure as kind: mcp) ─────────
# mcp-servers:
#   my-server:
#     type: stdio
#     command: npx
#     args: ["-y", "@my/mcp-server"]

# ── Inline Hooks (same structure as kind: hooks events) ──────
# hooks:
#   PreToolUse:
#     - matcher: "Bash"
#       hooks:
#         - type: command
#           command: "echo check"

# ── Permissions ──────────────────────────────────────────────
# permissions:
#   allow:
#     - "Bash(npm test *)"
#   deny:
#     - "Bash(rm -rf /)"
#   ask:
#     - "Bash(git push *)"
#   default-mode: default
#   additional-directories: [/tmp/workspace]
#   disable-bypass-permissions-mode: "always"

# ── Sandbox ──────────────────────────────────────────────────
# sandbox:
#   enabled: false
#   auto-allow-bash-if-sandboxed: true
#   fail-if-unavailable: false
#   allow-unsandboxed-commands: false
#   excluded-commands: [sudo, su]
#   filesystem:
#     allow-write: [/tmp, ./build]
#     deny-write: [/etc]
#     allow-read: [/usr/local/bin]
#     deny-read: [/root]
#   network:
#     allowed-domains: [api.anthropic.com, github.com]
#     allow-local-binding: true
#     allow-unix-sockets: [/var/run/docker.sock]

# ── Status Line ──────────────────────────────────────────────
# status-line:
#   type: command
#   command: "echo status"

# ── Plugins ──────────────────────────────────────────────────
# enabled-plugins:
#   superpowers: true

# ── Available Models ─────────────────────────────────────────
# available-models:
#   - claude-opus-4-5
#   - claude-sonnet-4-5

# ── Exclude Patterns ─────────────────────────────────────────
# md-excludes:
#   - node_modules/**
#   - .git/**

# ── Multi-Target (per-provider overrides) ────────────────────
# targets:
#   claude:
#     suppress-fidelity-warnings: false
