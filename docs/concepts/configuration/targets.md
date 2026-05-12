---
title: "Targets"
description: "How targets control which providers compile a resource and how they enable per-provider customization."
---

# Targets

Targets are the mechanism that connects xcaffold's compilation pipeline to specific AI providers. They appear in two contexts with distinct roles: on the project manifest (declaring which providers to compile for) and on individual resources (filtering which of those providers receive the resource).

## Project-Level Targets

The `targets:` field on `kind: project` lists the providers to compile output for. When absent, the `--target` CLI flag must be provided at compile time. This is a declaration — "compile for these providers."

```yaml
kind: project
version: "1.0"
name: my-project
targets:
  - claude
  - cursor
  - gemini
```

Valid provider names are: `claude`, `cursor`, `gemini`, `copilot`, `antigravity`.

## Resource-Level Targets

The `targets:` field on individual resources (`kind: agent`, `kind: context`, `kind: skill`, and others) is a render filter. It controls which of the project's compilation targets receive this resource.

| Value | Behavior |
|---|---|
| Absent / empty | Resource compiles for **all** targets configured in the project or `--target` flag |
| `[claude]` | Resource compiles **only** for `claude`; skipped for all other targets |
| `[claude, cursor]` | Resource compiles for `claude` and `cursor` only |

`targets:` on a resource is a filter, not a compilation directive. It does not add targets — it restricts which of the already-configured project targets receive this resource. If a project only targets `claude`, a resource with `targets: [gemini]` produces no output.

## List vs. Map Syntax

Resource kinds use one of two target syntaxes depending on whether per-provider overrides are needed:

| Syntax | Used By | Purpose |
|---|---|---|
| `[]string` (list) | `kind: context`, `kind: blueprint` | Pure filtering — include/exclude from compilation |
| `map[string]TargetOverride` (map) | `kind: agent`, `kind: skill`, `kind: rule`, `kind: hooks`, `kind: mcp`, `kind: workflow`, `kind: settings` | Filtering AND per-provider field overrides |

The map syntax serves a dual purpose: when the `targets` key is present, the resource compiles only for listed providers (filtering). The map values hold provider-specific overrides via `TargetOverride`:

```yaml
targets:
  claude:
    suppress-fidelity-warnings: false
  gemini: {}
```

An empty map value (`{}`) means "include this target with no overrides." The `TargetOverride` fields are:

| Field | Type | Purpose |
|---|---|---|
| `suppress-fidelity-warnings` | `bool` | Silence expected fidelity warnings for this provider |
| `hooks` | `map[string]string` | Provider-specific hook path overrides |
| `skip-synthesis` | `bool` | Skip synthesis for this provider without excluding the resource |
| `provider` | `map[string]any` | Opaque pass-through for provider-native keys |

## Default Resolution for Contexts

When multiple `kind: context` resources match the same target provider and no blueprint is active, xcaffold requires exactly one to be marked `default: true`. This context is placed first in the composed output; remaining contexts follow in alphabetical order.

| Scenario (no blueprint) | Result |
|---|---|
| 1 context matches target | Renders regardless of `default` — no ambiguity |
| 2+ match, exactly 1 has `default: true` | All bodies compose; default goes first |
| 2+ match, none has `default: true` | Compiler error — mark one as default or use `--blueprint` |
| 2+ match, multiple have `default: true` | Compiler error — only one default allowed per target |

When a blueprint is active (`--blueprint name`), the blueprint's `contexts:` list explicitly selects which contexts compile. The `default` flag is ignored.

## Related

- [Layer Precedence](layer-precedence.md) — How targets resolve through the 4-tier hierarchy (`--target` flag → blueprint → project → error)
- [Multi-Target Rendering](../architecture/multi-target-rendering.md) — How the TargetRenderer interface translates the AST into provider-native output
- [Multi-Target Compilation](../../best-practices/multi-target-compilation.md) — Practical patterns for declaring targets, scoping resources, and managing overrides
