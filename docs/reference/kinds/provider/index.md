---
title: "Provider Kinds"
description: "xcaf kinds that compile into provider-native output files."
---

# Provider Kinds

Provider kinds are `.xcaf` resources that xcaffold compiles into files inside each target provider's output directory. Run `xcaffold apply` to render them.

| Kind | What it compiles to | Supported providers |
|---|---|---|
| [`agent`](./agent) | `agents/<id>.md` with YAML frontmatter (Copilot: `.agent.md`) | All providers |
| [`skill`](./skill) | `skills/<id>/SKILL.md` | All providers |
| [`rule`](./rule) | `rules/<id>.md` or appended inline | All providers except Codex |
| [`mcp`](./mcp) | Provider-specific JSON config file | Claude, Cursor, Copilot, Gemini, Antigravity, Codex |
| [`workflow`](./workflow) | `workflows/<id>.md` | All providers except Codex |
| [`memory`](./memory) | Agent-scoped memory files | Claude (`agent-memory/<agent>/<name>.md`) |
| [`context`](./context) | `CLAUDE.md`, `GEMINI.md`, `AGENTS.md`, etc. | All providers |
| [`settings`](./settings) | Provider-native settings file | Claude, Cursor, Copilot, Gemini |
| [`hooks`](./hooks) | Provider-native hook configuration | Claude, Cursor, Copilot, Gemini, Antigravity, Codex |

Each provider may drop or transform fields that it does not support. Unsupported features emit a fidelity note to stderr rather than failing the build.
