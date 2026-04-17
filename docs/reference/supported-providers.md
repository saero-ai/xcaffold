---
title: "Supported Providers & Platforms"
description: "Available target AI runtimes and their compilation scopes"
---

# Supported Providers & Platforms

`xcaffold` compiles agent architectures into whichever native format a target AI platform expects. This decouples your instructions from any single vendor's proprietary schema.

Below is the definitive capability matrix for the AI runtimes currently supported natively by the `xcaffold apply` and `xcaffold translate` pipelines. A single `.xcf` file can emit deterministic configurations for any combination of these platforms simultaneously.

## Capability Matrix

| Feature / Primitive | Claude Code <br>`(.claude/)` | Cursor <br>`(.cursor/)` | Antigravity <br>`(.agents/)` | GitHub Copilot <br>`(.github/)` | Gemini CLI <br>`(~/.gemini/)` |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Agents** | `agents/*.md` | `agents/*.md` | *N/A* | `agents/*.md` or `agents/*.agent.md` | `agents/*.md` |
| **Skills** | `skills/*/SKILL.md` | `skills/*/SKILL.md` | `skills/*/SKILL.md` | `skills/*/SKILL.md` | `skills/*/SKILL.md` |
| **Rules** | `rules/*.md` | `rules/*.md` or `rules/*.mdc` | `GEMINI.md` | `.github/copilot-instructions.md` | `GEMINI.md` |
| **Workflows** | *via Rules & Skills* | *via Rules* | `workflows/*.md` | *N/A* | *N/A* |
| **Shell Hooks** | `settings.json` ¹ | `hooks.json` | *N/A* | *N/A* | *N/A* |
| **MCP Servers** | `.mcp.json` ² | `.cursor/mcp.json` | `~/.gemini/antigravity/mcp_config.json` | *N/A* | `settings.json` |
| **Settings & Sandbox** | `settings.json` | Cursor Settings UI | `settings.json` | IDE settings | `settings.json` |
| **Project Instructions** | `CLAUDE.md` (nested) | `AGENTS.md` (nested) | `.agents/rules/*.md` | `.github/copilot-instructions.md` | `GEMINI.md` |
| **Memory Context** | Auto Memory (persistent) | Not supported | Knowledge Items | Not supported | `save_memory` tool |

---

### Notes

¹ **Claude Code hooks** are compiled into the `hooks` key inside `settings.json` (`.claude/settings.json`), not as standalone shell scripts.

² **Claude Code MCP** — The native project-scoped convention is `.mcp.json` at the repository root. `xcaffold apply` currently emits `mcp.json` inside the `.claude/` output directory (i.e., `.claude/mcp.json`), which Claude Code also recognises as a valid MCP configuration location.

> [!NOTE]
> Target capabilities are continuously expanding. For a granular block-by-block breakdown of per-field fidelity mappings per target, consult the [Schema Reference](../reference/schema.md).
