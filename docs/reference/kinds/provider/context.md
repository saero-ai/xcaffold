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
You are working on xcaffold, a deterministic Agent-as-Code compiler...

## Project Layout
...
```

## Field Reference

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | `string` | Unique identifier for this context block. Must match `[a-z0-9-]+`. |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `description` | `string` | Human-readable purpose of this context block. |
| `default` | `bool` | Marks this context as tie-breaker when multiple match the same target. |
| `targets` | `[]string` | Restricts this context to specific provider targets. When absent, context renders for all targets. |

The **markdown body** (content after the closing `---`) contains the workspace context prose that is rendered verbatim to the provider's instruction file.

## Targets and Default Resolution

The `targets:` field filters which compilation targets receive this context. When multiple contexts match a target, the `default` field controls ordering. See [Targets](../../../concepts/configuration/targets.md) for the full concept, including filter semantics and default resolution rules.

## Filesystem-as-Schema

When a context `.xcaf` file lives at `xcaf/context/<name>/context.xcaf`, the `kind:` and `name:` fields can be omitted from the YAML. The parser infers:
- `kind: context` from the parent directory name (`context/`)
- `name:` from the grandparent directory name (e.g., `main` from `context/main/context.xcaf`)

When `kind:` or `name:` are present in the YAML, they must match the inferred values.

## Compiled Output

| Provider target | Output file |
|---|---|
| `claude` | `CLAUDE.md` (project root) |
| `gemini` | `GEMINI.md` (project root) |
| `cursor` | `AGENTS.md` (project root) |
| `copilot` | `.github/copilot-instructions.md` |
| `antigravity` | `GEMINI.md` (project root) |

## Default Import Naming

When running `xcaffold import`, xcaffold creates context files using these defaults:

| Source file | Generated context | Frontmatter |
|---|---|---|
| `CLAUDE.md` | `xcaf/context/claude/context.xcaf` | `name: claude` / `targets: [claude]` |
| `GEMINI.md` | `xcaf/context/gemini/context.xcaf` | `name: gemini` / `targets: [gemini]` |
| `AGENTS.md` (Cursor) | `xcaf/context/cursor/context.xcaf` | `name: cursor` / `targets: [cursor]` |
| `AGENTS.md` (Antigravity) | `xcaf/context/antigravity/context.xcaf` | `name: antigravity` / `targets: [antigravity]` |
| `.github/copilot-instructions.md` | `xcaf/context/copilot/context.xcaf` | `name: copilot` / `targets: [copilot]` |

You can rename the file and change the `name:` field freely — the `name` is a human identifier only.

## Related

- Context kind design rationale is documented in the project architecture decisions
- [kind: agent](agent.md)
- [kind: rule](rule.md)
- [Project Structure](../../../best-practices/project-structure.md)
