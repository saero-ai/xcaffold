---
title: "Supported Providers & Platforms"
description: "Available target AI runtimes and their compilation scopes"
---

# Supported Providers & Platforms

`xcaffold` compiles agent architectures into whichever native format a target AI platform expects. This decouples your instructions from any single vendor's proprietary schema.

All currently supported targets are **harness providers** — AI coding tools that read static configuration files at runtime. xcaffold compiles continuously to harness providers; your `.xcaf` manifests remain the source of truth.

Below is the definitive capability matrix for the AI runtimes currently supported natively by the `xcaffold apply` pipeline. A single `.xcaf` file can emit deterministic configurations for any combination of these platforms simultaneously.

## Capability Matrix

| Feature / Primitive | Claude Code <br>`(.claude/)` | Cursor <br>`(.cursor/)` | Gemini CLI <br>`(.gemini/)` | GitHub Copilot <br>`(.github/)` | Antigravity (deprecated) <br>`(.agents/)` | Antigravity 2 <br>`(.agents/)` | Codex (Preview) <br>`(.codex/)` |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Agents** | `agents/*.md` | `agents/*.md` | `agents/*.md` | `agents/*.md` or `agents/*.agent.md` | `agents/*.md` ⁴ | `agents/agent.json` ⁶ | `agents/*.toml` |
| **Skills** | `skills/*/SKILL.md` | `skills/*/SKILL.md` | `skills/*/SKILL.md` | `skills/*/SKILL.md` | `skills/*/SKILL.md` | `skills/*/SKILL.md` | `.agents/skills/*/SKILL.md` |
| **Rules** | `rules/*.md` | `rules/*.md` or `rules/*.mdc` | `rules/*.md` | `instructions/*.instructions.md` | `rules/*.md` | `rules/*.md` | *Not supported* ⁵ |
| **Workflows** | *via Rules & Skills* | *via Rules* | *via Rules & Skills* | *via Rules & Skills* | `workflows/*.md` | *via Rules & Skills* | *Not supported* |
| **Shell Hooks** | `settings.json` ¹ | `hooks.json` | `settings.json` | `hooks/xcaffold-hooks.json` | *N/A* | *N/A* | `hooks.json` |
| **MCP Servers** | `.mcp.json` ² | `.cursor/mcp.json` | `settings.json` | `.vscode/mcp.json` | `~/.gemini/antigravity/mcp_config.json` ³ | `.agents/mcp.json` ⁷ | `config.toml` |
| **Settings & Sandbox** | `settings.json` | Cursor Settings UI | `settings.json` | IDE settings | *N/A* | `settings.json` | *Not supported* |
| **Project Instructions** | `CLAUDE.md` (nested) | `AGENTS.md` (nested) | `GEMINI.md` | `.github/copilot-instructions.md` | `GEMINI.md` | `GEMINI.md` | `AGENTS.md` (nested) |
| **Memory Context** | Auto Memory (persistent) | Not supported | Not supported | Not supported | Not supported | Not supported | *Not supported* |

---

### Notes

¹ **Claude Code hooks** are compiled into the `hooks` key inside `settings.json` (`.claude/settings.json`), not as standalone shell scripts.

² **Claude Code MCP** — The native project-scoped convention is `.mcp.json` at the repository root. `xcaffold apply` currently emits `mcp.json` inside the `.claude/` output directory (i.e., `.claude/mcp.json`), which Claude Code also recognises as a valid MCP configuration location.

³ **Antigravity MCP (v1)** — Antigravity v1 reads MCP configuration exclusively from the global file `~/.gemini/antigravity/mcp_config.json`. No project-local MCP file is written. A `MCP_GLOBAL_CONFIG_ONLY` fidelity note is emitted. Configure MCP servers via the Antigravity MCP Store UI or edit the global config directly.

⁴ **Antigravity agents (v1)** are compiled as specialist profiles with a `RENDERER_KIND_DOWNGRADED` fidelity note. The resulting Markdown is a condensed specialist description rather than a full agent instruction set.

⁵ **Codex rules** — Codex uses Starlark `.rules` files, a fundamentally different paradigm that cannot be compiled from `.xcaf` rule declarations. A `RENDERER_KIND_UNSUPPORTED` fidelity note is emitted.

⁶ **Antigravity 2 agents** are compiled to `agents/agent.json` as structured JSON agent definitions. See [Antigravity 2.0 Runtime Behavior](#antigravity-20-runtime-behavior) for known discovery limitations.

⁷ **Antigravity 2 MCP** — Unlike v1 (global-only), Antigravity 2.0 supports workspace-local MCP configuration at `.agents/mcp_config.json`.

> [!NOTE]
> Target capabilities are continuously expanding. For a granular block-by-block breakdown of per-field fidelity mappings per target, consult the [Kind Reference](./kinds/index.md).

---

## Provider Import Support

`xcaffold import` reads an existing provider directory and synthesises a `project.xcaf` from the artifacts found on disk. Each provider importer handles the kinds it natively understands; unknown files are captured as `provider-extras` and emitted as fidelity notes during `xcaffold apply`.

### Kind Matrix

| Kind | Claude | Cursor | Gemini | Copilot | Antigravity (deprecated) | Antigravity 2 | Codex (Preview) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **agent** | `.claude/agents/*.md` | `.cursor/agents/*.md` | `.gemini/agents/*.md` | `.github/agents/*.{md,agent.md}` | `.agents/prompts/*.md` | `.agents/agents/agent.json` | `.codex/agents/*.toml` |
| **skill** | `.claude/skills/*/SKILL.md` | `.cursor/skills/*/SKILL.md` | `.gemini/skills/*/SKILL.md` | `.github/skills/*.md` | `.agents/skills/*/SKILL.md` | `.agents/skills/*/SKILL.md` | — (imported from `.agents/skills/`) |
| **rule** | `.claude/rules/*.md` | `.cursor/rules/*.mdc` | `.gemini/rules/*.md` | `.github/instructions/*.instructions.md` | `.agents/rules/*.md` | `.agents/rules/*.md` | — (Starlark, not supported) |
| **hook** | `.claude/settings.json` (`hooks` key) | `.cursor/hooks.json` | `.gemini/settings.json` | — | — | — | `.codex/hooks.json` |
| **mcp** | `.claude/mcp.json` | `.cursor/mcp.json` | `.gemini/settings.json` | `.github/copilot/mcp-config.json` | `.agents/mcp_config.json` | `.agents/mcp.json` | — (config.toml not yet supported for import) |
| **settings** | `.claude/settings.json` | Cursor Settings UI | `.gemini/settings.json` | — | — | `.agents/settings.json` | — |
| **provider-extras** | unrecognised `.claude/` files | unrecognised `.cursor/` files | unrecognised `.gemini/` files | unrecognised `.github/` files | unrecognised `.agents/` files | unrecognised `.agents/` files | unrecognised `.codex/` files |

`SourceProvider` is recorded on every imported resource so `xcaffold apply` can emit targeted fidelity notes when a resource cannot be faithfully translated to a different provider.

---

### Skill Subdirectory Support

Skills may declare `references/`, `scripts/`, `assets/`, and `examples/` subdirectories. Each provider handles these differently during compilation:

| Subdirectory | Claude Code | Cursor | Gemini CLI | GitHub Copilot | Antigravity (deprecated) | Antigravity 2 |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| `references/` | `references/` | `references/` | `references/` | `references/` | &rarr; `examples/` | `references/` |
| `scripts/` | `scripts/` | `scripts/` | `scripts/` | `scripts/` | `scripts/` | `scripts/` |
| `assets/` | `assets/` | `assets/` | `assets/` | `assets/` | &rarr; `resources/` | `assets/` |
| `examples/` | flat | &rarr; `references/` | &rarr; `references/` | `examples/` | `examples/` | `examples/` |

- **&rarr;** — Directory name is translated to the provider-native equivalent.
- **FidelityNote** — Provider does not support this concept; a `FIELD_UNSUPPORTED` fidelity note is emitted.
- **flat** — Files are placed alongside `SKILL.md` without a subdirectory wrapper.

---

## Antigravity 2.0 Runtime Behavior

> **Tested against:** `agy` CLI v1.0.6, Antigravity 2.0 desktop (June 2026). Behavior may change in future releases.

xcaffold compiles structurally correct output for Antigravity 2.0 across all supported kinds. However, the Antigravity 2.0 runtime does not consume every compiled artifact identically. The table below documents observed runtime behavior per resource type.

| Resource | Compiled Output | Runtime Discovery | Notes |
|----------|----------------|-------------------|-------|
| **Rules** | `.agents/rules/*.md` | Auto-loaded and applied | Works as expected. |
| **Skills** | `.agents/skills/*/SKILL.md` | Auto-discovered by name | Works as expected. Invocable via skill name. |
| **Workflows** | `.agents/workflows/*.md` | Registered as `/<name>` slash commands | Works as expected. |
| **MCP Servers** | `.agents/mcp_config.json` | Loaded and tools available | Works as expected. Servers start on demand. |
| **Knowledge Items** | `.agents/knowledge/*.md` | Visible and readable | Works as expected. Available as contextual memory. |
| **Hooks** | `.agents/hooks.json` | **Not loaded from this path** | Hooks are configured via Gemini settings (`settings.json`), not `.agents/hooks.json`. The compiled file is structurally correct but the runtime reads hooks from the settings config instead. ⁸ |
| **Agents** | `.agents/agents/*/agent.json` | **Not auto-discovered** | Agent files exist on disk and are readable, but the runtime does not auto-register them as invocable subagents. The LLM can read agent.json and use `define_subagent` to create agents manually at runtime. ⁹ |
| **Shared Output Dir** | `.agents/` | — | Antigravity v1, Antigravity 2.0, and Gemini CLI all write to `.agents/` and `GEMINI.md`. When compiling for multiple targets, the last-compiled target wins. |

### ⁸ Hooks Runtime Path

The Antigravity 2.0 documentation describes `.agents/hooks.json` as the workspace-level hooks configuration. In practice (agy v1.0.6), the runtime loads hooks from the Gemini-format settings config rather than the standalone hooks.json file. xcaffold compiles hooks.json with the correct schema (PreToolUse, PostToolUse, etc.), and the file will be consumed if/when the runtime adds direct hooks.json loading. Until then, hooks should also be configured via the antigravity2 settings path.

### ⁹ Agent Auto-Discovery

The Antigravity 2.0 documentation states that file-based agents "are loaded at conversation start." In practice (agy v1.0.6), the `/agents` command shows only built-in subagents (`research`, `self`). Custom agents defined in `.agents/agents/*/agent.json` are not auto-registered as invocable subagents. The runtime can read the files and offers to create agents via `define_subagent` using the JSON definition as input. This may be a planned feature gated behind the `/teamwork-preview` flag (Ultra plan, $200/month) or not yet shipped in the current CLI version.

xcaffold continues to compile agent.json files because the format matches official documentation and the files serve as a useful reference even without auto-discovery.

---

## Internal Registry

For developers contributing to `xcaffold`, the internal routing logic explicitly connects target scopes to their respective provider packages:

| Provider | InputDir | OutputDir | Importer | Renderer |
|----------|----------|-----------|----------|----------|
| claude | `.claude` | `.claude` | `providers/claude/` | `providers/claude/` |
| cursor | `.cursor` | `.cursor` | `providers/cursor/` | `providers/cursor/` |
| gemini | `.gemini` | `.gemini` | `providers/gemini/` | `providers/gemini/` |
| copilot | `.github` | `.github` | `providers/copilot/` | `providers/copilot/` |
| antigravity (deprecated) | `.agents` | `.agents` | `providers/antigravity/` | `providers/antigravity/` |
| antigravity2 | `.agents` | `.agents` | `providers/antigravity2/` | `providers/antigravity2/` |
| codex | `.codex` | `.codex` | `providers/codex/` | `providers/codex/` |
