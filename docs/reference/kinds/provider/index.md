---
title: "Provider Kinds"
description: "xcaf kinds that compile into provider-native output files."
---

# Provider Kinds

Provider kinds are `.xcaf` resources that xcaffold compiles into files inside each target provider's output directory. Run `xcaffold apply` to render them.

| Kind | What it compiles to | Supported providers |
|---|---|---|
| [`agent`](./agent) | `agents/<id>.md` with YAML frontmatter | Claude, Cursor, Copilot, Gemini |
| [`skill`](./skill) | `skills/<id>/SKILL.md` | All 5 providers |
| [`rule`](./rule) | `rules/<id>.md` or appended inline | All 5 providers |
| [`mcp`](./mcp) | Provider-specific JSON config file | Claude, Cursor, Gemini, Antigravity |
| [`workflow`](./workflow) | `workflows/<id>/WORKFLOW.md` | All 5 providers |
| [`memory`](./memory) | Agent-scoped memory files | Claude (native), Gemini (partial) |
| [`context`](./context) | `CLAUDE.md`, `.cursorrules`, etc. | All 5 providers |
| [`settings`](./settings) | `claude.json`, `settings.json` | Claude, Gemini, Antigravity |
| [`hooks`](./hooks) | `claude.json` event handlers | Claude, Antigravity |

Each provider may drop or transform fields that it does not support. Unsupported features emit a fidelity note to stderr rather than failing the build.
