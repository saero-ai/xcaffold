---
title: "kind: global"
description: "Declares shared resource definitions inherited by all projects that extend this base config. Produces no output files."
---

# `kind: global`

Declares shared resource definitions that are inherited by all projects via the `extends:` field in `project.xcf`. Global configs serve as a base layer — any resource (agent, skill, rule, MCP server, settings, hooks) declared in a global config is available to inheriting projects without re-declaring it.

Global configs produce **no output files** on their own. Output is produced only when a project config compiles with `extends:` pointing to this global.

Uses **pure YAML format** (no frontmatter `---` delimiters).

> **Required:** `kind`, `version`, `name`

## Example Usage

### Shared organization baseline

`xcf/global/org-baseline.xcf`:
```yaml
kind: global
version: "1.0"
name: org-baseline
description: >-
  Shared agent baseline for all Acme Corp projects. Provides the
  conventional-commits skill, the code-review rule, and
  org-wide security policies applicable to every repository.
skills:
  - id: conventional-commits
    path: xcf/global/skills/conventional-commits.xcf
  - id: systematic-debugging
    path: xcf/global/skills/systematic-debugging.xcf
rules:
  - id: no-secrets-in-code
    path: xcf/global/rules/no-secrets-in-code.xcf
  - id: code-review-standards
    path: xcf/global/rules/code-review-standards.xcf
policies:
  - id: require-approved-model
    path: xcf/global/policies/require-approved-model.xcf
settings:
  - id: org-defaults
    path: xcf/global/settings/org-defaults.xcf
```

Project `project.xcf` inheriting the global:
```yaml
---
kind: project
version: "1.0"
name: frontend-app
extends: xcf/global/org-baseline.xcf
targets:
  - claude
  - cursor
  - gemini
agents:
  - id: react-developer
    path: xcf/agents/react-developer/react-developer.xcf
skills:
  - component-patterns
rules:
  - react-conventions
---
This project builds on the org baseline with React-specific conventions.
```

The compiled output for `frontend-app` includes both the project-local resources (`react-developer`, `component-patterns`, `react-conventions`) and the inherited global resources (`conventional-commits`, `systematic-debugging`, `no-secrets-in-code`, `code-review-standards`).

## Argument Reference

The following arguments are supported:

- `name` — (Required) Unique global config identifier.
- `version` — (Required) Schema version. Use `"1.0"`.
- `description` — (Optional) `string`. What this baseline provides.
- `skills` — (Optional) `[]ResourceRef`. Skills to make available to inheriting projects.
- `rules` — (Optional) `[]ResourceRef`. Rules to make available to inheriting projects.
- `agents` — (Optional) `[]ResourceRef`. Agents to make available to inheriting projects.
- `mcp` — (Optional) `[]ResourceRef`. MCP servers to make available to inheriting projects.
- `policies` — (Optional) `[]ResourceRef`. Policies to enforce across all inheriting projects.
- `settings` — (Optional) `[]ResourceRef`. Settings to merge into inheriting projects.
- `hooks` — (Optional) `[]ResourceRef`. Hooks to merge into inheriting projects.

### `ResourceRef`

Each entry in a resource list supports:

- `id` — (Required) Identifier used to reference the resource from the inheriting project's `skills:`, `rules:`, etc. lists.
- `path` — (Required) Relative path from the global config root to the resource `.xcf` file.

## Behavior

When a project declares `extends:`, the compiler:

1. Parses the global config at the referenced path.
2. Marks all global resources with `Inherited = true`.
3. Merges global resources into the project's compiled resource set.
4. Strips inherited resources from local project file output (they are not re-emitted as new `.xcf` files).

Inherited resources can be overridden by re-declaring the same `id` in the project config. The local declaration wins.

> [!WARNING]
> Global configs are not validated in isolation — `xcaffold validate` always requires a project root. Run `xcaffold validate --from-root <project-dir>` to validate a project that extends a global.
