---
title: "Supported Providers & Platforms"
description: "Available target AI runtimes and their compilation scopes"
---

# Supported Providers & Platforms

`xcaffold` compiles agent architectures into whichever native format a target AI platform expects. This decouples your instructions from any single vendor's proprietary schema.

All currently supported targets are **harness providers** — AI coding tools that read static configuration files at runtime. xcaffold compiles continuously to harness providers; your `.xcaf` manifests remain the source of truth.

Below is the definitive capability matrix for the AI runtimes currently supported natively by the `xcaffold apply` pipeline. A single `.xcaf` file can emit deterministic configurations for any combination of these platforms simultaneously.## Capability Matrix

| Feature / Primitive | Claude Code <br>`(.claude/)` | Cursor <br>`(.cursor/)` | Gemini CLI <br>`(.gemini/)` | GitHub Copilot <br>`(.github/)` | Antigravity <br>`(.agents/)` | Codex (Preview) <br>`(.codex/)` |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Agents** | `agents/*.md` | `agents/*.md` | `agents/*.md` | `agents/*.md` or `agents/*.agent.md` | `agents/*.md` | `agents/*.toml` |
| **Skills** | `skills/*/SKILL.md` | `skills/*/SKILL.md` | `skills/*/SKILL.md` | `skills/*/SKILL.md` | `skills/*/SKILL.md` | `.agents/skills/*/SKILL.md` |
| **Rules** | `rules/*.md` | `rules/*.md` or `rules/*.mdc` | `rules/*.md` | `instructions/*.instructions.md` | `rules/*.md` | *Not supported* ⁴ |
| **Workflows** | *via Rules & Skills* | *via Rules* | *via Rules & Skills* | *via Rules & Skills* | `workflows/*.md` | *Not supported* |
| **Shell Hooks** | `settings.json` ¹ | `hooks.json` | `settings.json` | `hooks/xcaffold-hooks.json` | `hooks.json` | `hooks.json` |
| **MCP Servers** | `.mcp.json` ² | `.cursor/mcp.json` | `settings.json` | `.vscode/mcp.json` | `.agents/mcp_config.json` | `config.toml` |
| **Settings & Sandbox** | `settings.json` | Cursor Settings UI | `settings.json` | IDE settings | *N/A* | *Not supported* |
| **Project Instructions** | `CLAUDE.md` (nested) | `AGENTS.md` (nested) | `GEMINI.md` | `.github/copilot-instructions.md` | `GEMINI.md` | `AGENTS.md` (nested) |
| **Nested Context Support** | `✓` Yes `{path}/CLAUDE.md` | `✓` Yes `{path}/AGENTS.md` | `✓` Yes `{path}/GEMINI.md` | `✓` Yes (glob-based)⁵ | `✓` Yes `{path}/GEMINI.md` | ✗ No (consolidates to root) |
| **Memory Context** | Auto Memory (persistent) | Not supported | Not supported | Not supported | Not supported | *Not supported* |

---

### Notes

¹ **Claude Code hooks** are compiled into the `hooks` key inside `settings.json` (`.claude/settings.json`), not as standalone shell scripts.

² **Claude Code MCP** — The native project-scoped convention is `.mcp.json` at the repository root. `xcaffold apply` currently emits `mcp.json` inside the `.claude/` output directory (i.e., `.claude/mcp.json`), which Claude Code also recognises as a valid MCP configuration location.

⁴ **Codex rules** — Codex uses Starlark `.rules` files, a fundamentally different paradigm that cannot be compiled from `.xcaf` rule declarations. A `RENDERER_KIND_UNSUPPORTED` fidelity note is emitted.

⁵ **GitHub Copilot nested contexts** — When a context has `path: "services/api/"`, Copilot's renderer creates `.github/instructions/{context-name}.instructions.md` with an `applyTo:` field set to the glob pattern `services/api/**`. This allows Copilot to load context-specific instructions only when the user edits files matching that pattern. Codex consolidates all contexts at the project root regardless of the `path:` field.

> [!NOTE]
> **Unified Antigravity Provider:** Antigravity CLI (`agy`), Antigravity IDE, and Antigravity 2.0 runtime are unified into the canonical `antigravity` provider (`.agents/`). Legacy `antigravity2` configurations have been consolidated into `antigravity`.

---

## Provider Import Support

`xcaffold import` reads an existing provider directory and synthesises a `project.xcaf` from the artifacts found on disk. Each provider importer handles the kinds it natively understands; unknown files are captured as `provider-extras` and emitted as fidelity notes during `xcaffold apply`.

### Kind Matrix

| Kind | Claude | Cursor | Gemini | Copilot | Antigravity | Codex (Preview) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **agent** | `.claude/agents/*.md` | `.cursor/agents/*.md` | `.gemini/agents/*.md` | `.github/agents/*.{md,agent.md}` | `.agents/agents/*.md` | `.codex/agents/*.toml` |
| **skill** | `.claude/skills/*/SKILL.md` | `.cursor/skills/*/SKILL.md` | `.gemini/skills/*/SKILL.md` | `.github/skills/*.md` | `.agents/skills/*/SKILL.md` | — (imported from `.agents/skills/`) |
| **rule** | `.claude/rules/*.md` | `.cursor/rules/*.mdc` | `.gemini/rules/*.md` | `.github/instructions/*.instructions.md` | `.agents/rules/*.md` | — (Starlark, not supported) |
| **hook** | `.claude/settings.json` (`hooks` key) | `.cursor/hooks.json` | `.gemini/settings.json` | — | `.agents/hooks.json` | `.codex/hooks.json` |
| **mcp** | `.claude/mcp.json` | `.cursor/mcp.json` | `.gemini/settings.json` | `.github/copilot/mcp-config.json` | `.agents/mcp_config.json` | — (config.toml not yet supported for import) |
| **settings** | `.claude/settings.json` | Cursor Settings UI | `.gemini/settings.json` | — | — | — |
| **memory** | `.claude/agent-memory/**` | — | — | — | — | — |
| **workflow** | `.claude/workflows/*.md` | — | — | `.github/workflows/copilot-setup-steps.yml` | `.agents/workflows/*.md` | — |
| **provider-extras** | unrecognised `.claude/` files | unrecognised `.cursor/` files | unrecognised `.gemini/` files | unrecognised `.github/` files | unrecognised `.agents/` files | unrecognised `.codex/` files |

`SourceProvider` is recorded on every imported resource so `xcaffold apply` can emit targeted fidelity notes when a resource cannot be faithfully translated to a different provider.

---

### Skill Subdirectory Support

Skills may declare `references/`, `scripts/`, `assets/`, and `examples/` subdirectories. Each provider handles these differently during compilation:

| Subdirectory | Claude Code | Cursor | Gemini CLI | GitHub Copilot | Antigravity |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `references/` | `references/` | `references/` | `references/` | `references/` | &rarr; `examples/` |
| `scripts/` | `scripts/` | `scripts/` | `scripts/` | `scripts/` | `scripts/` |
| `assets/` | `assets/` | `assets/` | `assets/` | `assets/` | &rarr; `resources/` |
| `examples/` | flat | &rarr; `references/` | &rarr; `references/` | `examples/` | `examples/` |

- **&rarr;** — Directory name is translated to the provider-native equivalent.
- **FidelityNote** — Provider does not support this concept; a `FIELD_UNSUPPORTED` fidelity note is emitted.
- **flat** — Files are placed alongside `SKILL.md` without a subdirectory wrapper.

---

## Internal Registry

For developers contributing to `xcaffold`, the internal routing logic explicitly connects target scopes to their respective provider packages:

| Provider | InputDir | OutputDir | Importer | Renderer |
|----------|----------|-----------|----------|----------|
| claude | `.claude` | `.claude` | `providers/claude/` | `providers/claude/` |
| cursor | `.cursor` | `.cursor` | `providers/cursor/` | `providers/cursor/` |
| gemini | `.gemini` | `.gemini` | `providers/gemini/` | `providers/gemini/` |
| copilot | `.github` | `.github` | `providers/copilot/` | `providers/copilot/` |
| antigravity | `.agents` | `.agents` | `providers/antigravity/` | `providers/antigravity/` |
| codex | `.codex` | `.codex` | `providers/codex/` | `providers/codex/` |

