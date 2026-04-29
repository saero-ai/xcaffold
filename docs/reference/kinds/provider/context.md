---
title: "kind: context"
description: "Workspace-level ambient context compiled into each provider's root instruction file."
---

# kind: context
Workspace-level ambient context compiled into each provider's root instruction file — `CLAUDE.md`, `GEMINI.md`, `AGENTS.md`, or `.github/copilot-instructions.md`.

## Overview

A `kind: context` resource holds the prose that xcaffold renders into the root instruction file for one or more AI providers. Unlike agents, skills, and rules — which target specific AI behaviors — context tells the AI tool about the project itself: its structure, conventions, and how to work within it.

Context files live in `xcf/context/` and use the standard frontmatter + body format. The markdown body is rendered verbatim as the provider's instruction file content.

## Format

```yaml
---
kind: context
version: "1.0"
name: main
targets: [claude]           # optional — see Targets below
description: "..."          # optional, human-readable summary
---
You are working on xcaffold, a deterministic Agent-as-Code compiler...

## Project Layout
...
```

## Fields

| Field | Required | Type | Description |
|---|---|---|---|
| `kind` | Yes | string | Must be `context` |
| `version` | Yes | string | Schema version, e.g. `"1.0"` |
| `name` | Yes | string | Unique identifier within the project. Used as the xcf filename stem. |
| `targets` | No | list of strings | Provider filter — see [Targets](#targets) below |
| `description` | No | string | Short human-readable description of this context file |

The **markdown body** (content after the closing `---`) contains the workspace context prose that is rendered verbatim to the provider's instruction file.

## Targets

`targets:` is a **render filter**. It controls which compilation targets receive this context file.

| `targets:` value | Behavior |
|---|---|
| Absent / empty (`[]`) | Context renders for **all** targets configured in the active `xcaffold apply` invocation |
| `[claude]` | Context renders **only** when compiling for the `claude` target; **skipped** for all other targets |
| `[claude, cursor]` | Context renders for `claude` and `cursor` targets only |

This allows multi-provider projects to author provider-specific workspace context where the content differs between tools:

```yaml
# xcf/context/main.xcf — Claude Code only
---
kind: context
name: main
targets: [claude]
---
You are working on xcaffold. Use CLAUDE.md conventions...

# xcf/context/antigravity.xcf — Antigravity only
---
kind: context
name: antigravity
targets: [antigravity]
---
You are working on xcaffold. As an Antigravity agent...
```

For projects targeting a single provider, omit `targets:` entirely:

```yaml
---
kind: context
name: main
---
You are working on xcaffold...
```

> [!NOTE]
> `targets:` is a filter, not a compilation directive. It does not add targets — it restricts which of the already-configured project targets receive this context. If your project only targets `claude`, a context with `targets: [gemini]` produces no output.

## File Location

Context files are placed in `xcf/context/`:

```
xcf/
└── context/
    ├── main.xcf           # root context for claude (or all targets)
    ├── gemini.xcf         # root context for gemini only
    └── antigravity.xcf    # root context for antigravity only
```

Files are auto-discovered by `xcaffold apply` — no explicit reference in `project.xcf` is required.

## Provider Output

| Provider target | Output file |
|---|---|
| `claude` | `CLAUDE.md` (project root) |
| `gemini` | `GEMINI.md` (project root) |
| `cursor` | `AGENTS.md` (project root) |
| `copilot` | `.github/copilot-instructions.md` |
| `antigravity` | `AGENTS.md` (project root) |

## Default Import Naming

When running `xcaffold import`, xcaffold creates context files using these defaults:

| Source file | Generated context | Frontmatter |
|---|---|---|
| `CLAUDE.md` | `xcf/context/main.xcf` | `name: main` / `targets: [claude]` |
| `GEMINI.md` | `xcf/context/gemini.xcf` | `name: gemini` / `targets: [gemini]` |
| `AGENTS.md` (Cursor) | `xcf/context/cursor.xcf` | `name: cursor` / `targets: [cursor]` |
| `AGENTS.md` (Antigravity) | `xcf/context/antigravity.xcf` | `name: antigravity` / `targets: [antigravity]` |
| `.github/copilot-instructions.md` | `xcf/context/copilot.xcf` | `name: copilot` / `targets: [copilot]` |

You can rename the file and change the `name:` field freely — the `name` is a human identifier only.

## Related

- Context kind design rationale is documented in the project architecture decisions
- [kind: agent](agent.md)
- [kind: rule](rule.md)
- [Project Layouts](../../best-practices/project-layouts.md)
