---
title: "kind: context"
description: "Workspace-level ambient context compiled into each provider's root instruction file."
---

# `kind: context`

Workspace-level ambient context compiled into each provider's root instruction file — `CLAUDE.md`, `GEMINI.md`, `AGENTS.md`, or `.github/copilot-instructions.md`.

A `kind: context` resource holds the prose that xcaffold renders into the root instruction file for one or more AI providers. Unlike agents, skills, and rules — which target specific AI behaviors — context tells the AI tool about the project itself: its structure, conventions, and how to work within it.

The markdown body is rendered verbatim as the provider's instruction file content.

> **Required:** `kind`, `version`, `name`

## Source Directory

```
xcaf/context/<name>/context.xcaf
```

## Example Usage

```yaml
---
kind: context
version: "1.0"
name: main
targets: [claude]           # optional — see Targets below
description: "..."          # optional, human-readable summary
---
You are working on xcaffold, a deterministic Harness-as-Code compiler...

## Project Layout
...
```

## Field Reference

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `kind` | `string` | Resource type. Must be `context`. |
| `version` | `string` | File format version. Must be `"1.0"`. |
| `name` | `string` | Unique identifier for this context block. Must match `[a-z0-9-]+`. |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `description` | `string` | Human-readable purpose of this context block. |
| `default` | `bool` | Controls inclusion in bare apply. When `true`, marks this context as the tie-breaker when multiple contexts match the same target. When `false`, excludes this context from bare `xcaffold apply` (it is only used when a blueprint explicitly selects it). Omitting this field includes the context normally. |
| `path` | `string` | Subdirectory scope for this context. When set, the context is rendered into a provider instruction file at `<path>/<instruction-file>` rather than the project root. See [Subdirectory Scoping](#subdirectory-scoping) below. |
| `targets` | `[]string` | Restricts this context to specific provider targets. When absent, context renders for all targets. |

The **markdown body** (content after the closing `---`) contains the workspace context prose that is rendered verbatim to the provider's instruction file.

## Targets and Default Resolution

The `targets:` field filters which compilation targets receive this context. When multiple contexts match a target during bare apply, the `default` field determines which one is used:

- `default: true` — this context wins the tie-breaker; it is placed first in the composed output
- `default: false` — this context is excluded from bare apply entirely; it only participates when a blueprint explicitly selects it
- field omitted — context participates normally; if exactly one context matches after filtering, no tie-breaker is needed

See [Targets](../../../concepts/configuration/targets.md) for the full concept, including filter semantics and default resolution rules.

## Filesystem-as-Schema

When a context `.xcaf` file lives at `xcaf/context/<name>/context.xcaf`, the `kind:` and `name:` fields can be omitted from the YAML. The parser infers:
- `kind: context` from the parent directory name (`context/`)
- `name:` from the grandparent directory name (e.g., `main` from `context/main/context.xcaf`)

When `kind:` or `name:` are present in the YAML, they must match the inferred values.

## Compiled Output

When `path` is absent or empty, context renders to the provider's root instruction file:

| Provider target | Output file |
|---|---|
| `claude` | `CLAUDE.md` (project root) |
| `gemini` | `GEMINI.md` (project root) |
| `cursor` | `AGENTS.md` (project root) |
| `copilot` | `.github/copilot-instructions.md` |
| `antigravity` (deprecated) | `GEMINI.md` (project root) |
| `antigravity2` | `GEMINI.md` (project root) |
| `codex` | `AGENTS.md` (project root) |

When `path` is set, the instruction file is placed inside that subdirectory instead:

| Provider target | Example with `path: "frontend"` |
|---|---|
| `claude` | `frontend/CLAUDE.md` |
| `gemini` | `frontend/GEMINI.md` |
| `cursor` | `frontend/AGENTS.md` |

See [Subdirectory Scoping](#subdirectory-scoping) for full details.

## Subdirectory Scoping

Set `path` to scope a context to a specific subdirectory of the project:

```yaml
---
kind: context
version: "1.0"
name: frontend-ctx
path: frontend
targets: [claude, gemini, cursor]
---
You are working in the frontend subdirectory. Use TypeScript and React conventions.
```

xcaffold renders this context to `frontend/CLAUDE.md`, `frontend/GEMINI.md`, and `frontend/AGENTS.md` respectively. The project root instruction file is not created unless a separate root context (with `path` absent or empty) also exists.

Multiple path-bearing contexts can coexist, each scoping to a different directory:

```yaml
---
name: backend-ctx
path: backend
---
Backend service conventions...
```

```yaml
---
name: frontend-ctx
path: frontend
---
Frontend conventions...
```

This produces `backend/CLAUDE.md` and `frontend/CLAUDE.md` as separate files.

## Default Import Naming

When running `xcaffold import`, xcaffold creates context files using these defaults:

| Source file | Generated context | Frontmatter |
|---|---|---|
| `CLAUDE.md` (Claude)| `xcaf/context/claude/context.xcaf` | `name: claude` / `targets: [claude]` |
| `AGENTS.md` (Cursor) | `xcaf/context/cursor/context.xcaf` | `name: cursor` / `targets: [cursor]` |
| `AGENTS.md` (Codex) | `xcaf/context/codex/context.xcaf` | `name: codex` / `targets: [codex]` |
| `GEMINI.md` (Gemini CLI) | `xcaf/context/gemini/context.xcaf` | `name: gemini` / `targets: [gemini]` |
| `GEMINI.md` (Antigravity) | `xcaf/context/antigravity/context.xcaf` | `name: antigravity` / `targets: [antigravity]` |
| `GEMINI.md` (Antigravity 2) | `xcaf/context/antigravity2/context.xcaf` | `name: antigravity2` / `targets: [antigravity2]` |
| `.github/copilot-instructions.md` (Github Copilot) | `xcaf/context/copilot/context.xcaf` | `name: copilot` / `targets: [copilot]` |

You can rename the file and change the `name:` field freely — the `name` is a human identifier only.

## Related

- Context kind design rationale is documented in the project architecture decisions
- [kind: agent](agent.md)
- [kind: rule](rule.md)
- [Project Structure](../../../best-practices/project-structure.md)
