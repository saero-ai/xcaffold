---
title: "Supported Providers & Platforms"
description: "Available target AI runtimes and their compilation scopes"
---

# Supported Providers & Platforms

`xcaffold` compiles agent architectures into whichever native format a target AI platform expects. This decouples your instructions from any single vendor's proprietary schema.

Below is the definitive capability matrix for the AI runtimes currently supported natively by the `xcaffold apply` and `xcaffold translate` pipelines. A single `.xcaf` file can emit deterministic configurations for any combination of these platforms simultaneously.

## Capability Matrix

| Feature / Primitive | Claude Code <br>`(.claude/)` | Cursor <br>`(.cursor/)` | Gemini CLI <br>`(.gemini/)` | GitHub Copilot <br>`(.github/)` | Antigravity <br>`(.agents/)` |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Agents** | `agents/*.md` | `agents/*.md` | `agents/*.md` | `agents/*.md` or `agents/*.agent.md` | *N/A* |
| **Skills** | `skills/*/SKILL.md` | `skills/*/SKILL.md` | `skills/*/SKILL.md` | `skills/*/SKILL.md` | `skills/*/SKILL.md` |
| **Rules** | `rules/*.md` | `rules/*.md` or `rules/*.mdc` | `GEMINI.md` | `.github/copilot-instructions.md` | `GEMINI.md` |
| **Workflows** | *via Rules & Skills* | *via Rules* | *via Rules & Skills* | *via Rules & Skills* | `workflows/*.md` |
| **Shell Hooks** | `settings.json` ¹ | `hooks.json` | `settings.json` | `hooks/xcaffold-hooks.json` | *N/A* |
| **MCP Servers** | `.mcp.json` ² | `.cursor/mcp.json` | `settings.json` | `.vscode/mcp.json` | `~/.gemini/antigravity/mcp_config.json` ³ |
| **Settings & Sandbox** | `settings.json` | Cursor Settings UI | `settings.json` | IDE settings | `settings.json` |
| **Project Instructions** | `CLAUDE.md` (nested) | `AGENTS.md` (nested) | `GEMINI.md` | `.github/copilot-instructions.md` | `.agents/rules/*.md` |
| **Memory Context** | Auto Memory (persistent) | Not supported | Not supported | Not supported | Not supported |

---

### Notes

¹ **Claude Code hooks** are compiled into the `hooks` key inside `settings.json` (`.claude/settings.json`), not as standalone shell scripts.

² **Claude Code MCP** — The native project-scoped convention is `.mcp.json` at the repository root. `xcaffold apply` currently emits `mcp.json` inside the `.claude/` output directory (i.e., `.claude/mcp.json`), which Claude Code also recognises as a valid MCP configuration location.

³ **Antigravity MCP** — Antigravity reads MCP configuration exclusively from the global file `~/.gemini/antigravity/mcp_config.json`. No project-local MCP file is written. A `MCP_GLOBAL_CONFIG_ONLY` fidelity note is emitted. Configure MCP servers via the Antigravity MCP Store UI or edit the global config directly.

> [!NOTE]
> Target capabilities are continuously expanding. For a granular block-by-block breakdown of per-field fidelity mappings per target, consult the [Schema Reference](../reference/schema.md).

---

## Provider Import Support

`xcaffold import` reads an existing provider directory and synthesises a `project.xcaf` from the artifacts found on disk. Each provider importer handles the kinds it natively understands; unknown files are captured as `provider-extras` and emitted as fidelity notes during `xcaffold apply`.

### Kind Matrix

| Kind | Claude | Cursor | Gemini | Copilot | Antigravity |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **agent** | `.claude/agents/*.md` | `.cursor/agents/*.md` | `.gemini/agents/*.md` | `.github/agents/*.{md,agent.md}` | — |
| **skill** | `.claude/skills/*/SKILL.md` | `.cursor/skills/*/SKILL.md` | `.gemini/skills/*/SKILL.md` | `.github/skills/*/SKILL.md` | `.agents/skills/*/SKILL.md` |
| **rule** | `.claude/rules/*.md` | `.cursor/rules/*.{md,mdc}` | `.gemini/rules/*.md` | `.github/copilot-instructions.md` | `.agents/rules/*.md` |
| **hook** | `.claude/settings.json` (`hooks` key) | `.cursor/hooks.json` | `.gemini/settings.json` | `.github/hooks/xcaffold-hooks.json` | — |
| **mcp** | `.mcp.json` | `.cursor/mcp.json` | `.gemini/settings.json` | `.vscode/mcp.json` | — |
| **settings** | `.claude/settings.json` | Cursor Settings UI | `.gemini/settings.json` | — | `.agents/settings.json` |
| **provider-extras** | unrecognised `.claude/` files | unrecognised `.cursor/` files | unrecognised `.gemini/` files | unrecognised `.github/` files | unrecognised `.agents/` files |

`SourceProvider` is recorded on every imported resource so `xcaffold apply` can emit targeted fidelity notes when a resource cannot be faithfully translated to a different provider.

---

### Skill Subdirectory Support

Skills may declare `references/`, `scripts/`, `assets/`, and `examples/` subdirectories. Each provider handles these differently during compilation:

| Subdirectory | Claude Code | Cursor | Gemini CLI | GitHub Copilot | Antigravity |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `references/` | `references/` | `references/` | `references/` | co-located | &rarr; `examples/` |
| `scripts/` | `scripts/` | `scripts/` | `scripts/` | co-located | `scripts/` |
| `assets/` | `assets/` | `assets/` | `assets/` | co-located | &rarr; `resources/` |
| `examples/` | flat | &rarr; `references/` | &rarr; `references/` | co-located | `examples/` |

- **co-located** — Copilot flattens all subdirectory contents alongside `SKILL.md`.
- **&rarr;** — Directory name is translated to the provider-native equivalent.
- **FidelityNote** — Provider does not support this concept; a `FIELD_UNSUPPORTED` fidelity note is emitted.
- **flat** — Files are placed alongside `SKILL.md` without a subdirectory wrapper.

---

## Internal Registry

For developers contributing to `xcaffold`, the internal routing logic explicitly connects target scopes to their respective compiler packages:

| Provider | InputDir | OutputDir | Importer | Renderer |
|----------|----------|-----------|----------|----------|
| claude | `.claude` | `.claude` | `internal/importer/claude/` | `internal/renderer/claude/` |
| cursor | `.cursor` | `.cursor` | `internal/importer/cursor/` | `internal/renderer/cursor/` |
| gemini | `.gemini` | `.gemini` | `internal/importer/gemini/` | `internal/renderer/gemini/` |
| copilot | `.github` | `.github` | `internal/importer/copilot/` | `internal/renderer/copilot/` |
| antigravity | `.agents` | `.agents` | `internal/importer/antigravity/` | `internal/renderer/antigravity/` |
