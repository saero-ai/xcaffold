---
title: "kind: project"
description: "Root manifest that declares compilation targets and references all named xcaf resources. One per project."
---

# `kind: project`

The root manifest for an xcaffold project. Declares which providers to target, references all named resources (agents, skills, rules, MCP servers, policies), and configures project-wide settings.

There is **exactly one** project manifest per project, located at `project.xcaf` (at the repository root). Produces no provider output files — the manifest drives the compilation pipeline.

> [!IMPORTANT]
> The `kind: project` manifest does **not** support a markdown body. Workspace-level instructions must be declared using [`kind: context`](../provider/context). Adding a body after the closing `---` in a project manifest will cause a parse error.

> **Required:** `kind`, `version`, `name`

## Source Directory

```
project.xcaf
```

Located at the repository root. There is exactly one project manifest per project.

## Example Usage

### Minimal project

```yaml
kind: project
version: "1.0"
name: frontend-app
targets:
  - claude
  - cursor
```

### Full project manifest — React TypeScript SaaS

```yaml
kind: project
version: "1.0"
name: frontend-app
description: >-
  React TypeScript SaaS frontend built with Next.js 15, Tailwind CSS v4, and
  Shadcn/UI. Deployed on Vercel. All components use the design system defined
  in src/components/ui/.
author: Acme Corp
repository: https://github.com/acme/frontend-app
targets:
  - claude
  - cursor
  - gemini
  - copilot
  - antigravity
target-options:
  copilot:
    hooks:
      copilot-instructions: ".copilot/instructions.md"
  cursor:
    suppress-fidelity-warnings: false
```

## Field Reference

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | `string` | Unique project identifier. Must match `[a-z0-9-]+`. |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `description` | `string` | Human-readable project description. |
| `version` | `string` | Project version (distinct from the schema version in `kind`/`version`). |
| `author` | `string` | Project author or organization name. |
| `homepage` | `string` | Project URL. |
| `repository` | `string` | Source repository URL. |
| `license` | `string` | SPDX license identifier. |
| `extends` | `string` | Path to a global config to inherit resources from. |
| `backup-dir` | `string` | Directory where provider output is backed up before each overwrite. |
| `allowed-env-vars` | `[]string` | Environment variable names that may be injected via `${env.NAME}` inside `.xcaf` files. Variables not listed here are rejected at compile time. |
| `targets` | `[]string` | Provider targets to compile: `claude`, `cursor`, `gemini`, `copilot`, `antigravity`. When empty, the `--target` flag must be provided at compile time. |
| `target-options` | `map[string]TargetOverride` | Per-provider compile-time overrides. See [target-options block](#target-options-block). |

> **Note:** Resources (agents, skills, rules, MCP servers, policies, workflows, memory, and contexts) are not declared inline in `project.xcaf`. They are discovered automatically from `xcaf/` subdirectories (e.g., `xcaf/agents/<name>/agent.xcaf`). See [Resource File Format](#resource-file-format-one-kind-per-file) in the kinds index for directory layout details.

### Resource Discovery

Resources are not declared inline in the project manifest. Instead, xcaffold discovers them from the `xcaf/` directory tree:

- `xcaf/agents/<name>/agent.xcaf` — Agent definitions
- `xcaf/skills/<name>/skill.xcaf` — Skill definitions
- `xcaf/rules/<name>/rule.xcaf` — Rule definitions
- `xcaf/policies/<name>/policy.xcaf` — Policy constraints

All discovered resources are merged into the project's compilation scope automatically. No explicit entry in `project.xcaf` is required.

### `target-options` block

Per-provider compile-time overrides keyed by provider name. Each value is a `TargetOverride` with the following fields:

| Field | Type | Description |
|-------|------|-------------|
| `hooks` | `map[string]string` | Provider-specific hook path overrides. |
| `suppress-fidelity-warnings` | `bool` | Suppress fidelity notes for this provider during `apply`. |
| `skip-synthesis` | `bool` | Skip synthesis passes for this provider. |
| `provider` | `map[string]any` | Opaque pass-through map emitted verbatim into the provider's native config. |

```yaml
target-options:
  copilot:
    hooks:
      copilot-instructions: ".copilot/instructions.md"
    provider:
      groups:
        - copilot-chat
  cursor:
    suppress-fidelity-warnings: false
    skip-synthesis: false
```

### `targets` on resources vs. project

The `targets:` field has different roles depending on where it appears (project manifest vs. individual resources). See [Targets](../../../concepts/configuration/targets.md) for the full explanation.

## Import

```bash
xcaffold import --target claude
```

`xcaffold import` reverse-engineers existing provider directories into `.xcaf` source files and reconstructs a `project.xcaf` with discovered resources. Instructions found in provider root files (e.g. `CLAUDE.md`) are imported as `kind: context` resources.
