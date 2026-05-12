---
title: "kind: memory"
description: "Defines persistent context for an agent persona. Source: `xcaf/agents/<agent-id>/memory/<name>.md`."
---

# `kind: memory`

Defines persistent context or "long-term memory" for a specific agent persona. Memory resources are discovered via directory convention and synthesized into the provider-native agent configuration during compilation.

Memory is authored as plain Markdown files — not `.xcaf` files. Each file lives directly inside the agent's `memory/` directory. `MEMORY.md` files are skipped during discovery.

## Example Usage

```markdown
---
name: project-context
description: Architectural overview and core constraints.
---
# Project Context

This project uses a monorepo structure with a Go CLI and a Next.js platform.
All code must adhere to the current architectural standards.
```

Save this file at `xcaf/agents/<agent-id>/memory/project-context.md`.

The frontmatter is optional. When omitted, `name` is inferred from the filename stem (e.g. `project-context.md` → `name: project-context`) and `description` falls back to the first 120 characters of the body.

## Argument Reference

### Required Arguments

There are no required arguments. A memory file with only a Markdown body is valid.

### Optional Arguments

| Argument | Type | Description | Provider Support |
| :--- | :--- | :--- | :--- |
| `name` | `string` | Identifier for the memory entry. When present, must match `^[a-z0-9-]+$`. Inferred from filename stem when absent. | All providers |
| `description` | `string` | One-sentence summary of the memory content. Falls back to the first 120 chars of the body when absent. | **Claude**: Used as metadata. **Others**: Dropped. |

> **Note:** Memory files do not have a `kind:` field in frontmatter. The `memory/` directory path is the discriminator.

## Filesystem-as-Schema

When a memory file is placed at `xcaf/agents/<agent-id>/memory/<name>.md`, xcaffold automatically infers:

- **kind**: `memory` — derived from the `memory/` directory.
- **name**: `<name>` — derived from the filename stem.
- **agent**: `<agent-id>` — derived from the parent agent directory.

`MEMORY.md` files (exact match, case-sensitive) are excluded from discovery.

## Variable Expansion

Memory file content supports variable expansion. You may reference project and environment variables in the Markdown body:

```markdown
This project targets the ${var.target_provider} provider.
Secrets are loaded from ${env.SECRET_ENV_VAR}.
```

Both `${var.*}` and `${env.*}` expansion patterns are resolved at compile time.

## Compiled Output

Memory content is synthesized based on the target provider's capabilities.

| Provider | Output | Notes |
| :--- | :--- | :--- |
| **Claude Code** | `memory: user` scalar in agent frontmatter; content written to `agent-memory/<agentRef>/<name>.md`; entries indexed in `MEMORY.md` | Full support |
| **Gemini CLI** | Content appended unconditionally to `GEMINI.md` under `## Gemini Added Memories` | Memory capability is deferred; this output is a best-effort approximation |
| **Antigravity** | Written as Knowledge Item files | Memory capability is deferred; not yet available |
| **Cursor** | No output | Emits a fidelity note; memory is not supported by the Cursor agent format |
| **Copilot** | No output | Emits a fidelity note; memory is not supported by the Copilot agent format |

Memory is excluded from the override inheritance system. Memory files are not merged or inherited across agent scopes.
