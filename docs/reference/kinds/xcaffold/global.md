---
title: "kind: global"
description: "Declares shared resource definitions inherited by all projects that extend this base config. Produces no output files."
---

# `kind: global`

Declares shared resource definitions that are inherited by all projects via the `extends:` field in `project.xcaf`. Global configs serve as a base layer — any resource (agent, skill, rule, MCP server, settings, hooks) declared in a global config is available to inheriting projects without re-declaring it.

Global configs produce **no output files** on their own. Output is produced only when a project config compiles with `extends:` pointing to this global.

Uses **pure YAML format** (no frontmatter `---` delimiters).

> **Required:** `kind`, `version`, `name`

## Example Usage

### Shared organization baseline

`xcaf/global/org-baseline.xcaf`:
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
    path: xcaf/global/skills/conventional-commits.xcaf
  - id: systematic-debugging
    path: xcaf/global/skills/systematic-debugging.xcaf
rules:
  - id: no-secrets-in-code
    path: xcaf/global/rules/no-secrets-in-code.xcaf
  - id: code-review-standards
    path: xcaf/global/rules/code-review-standards.xcaf
policies:
  - id: require-approved-model
    path: xcaf/global/policies/require-approved-model.xcaf
settings:
  - id: org-defaults
    path: xcaf/global/settings/org-defaults.xcaf
```

Project `project.xcaf` inheriting the global:
```yaml
---
kind: project
version: "1.0"
name: frontend-app
extends: xcaf/global/org-baseline.xcaf
targets:
  - claude
  - cursor
  - gemini
agents:
  - id: react-developer
    path: xcaf/agents/react-developer/react-developer.xcaf
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
- `path` — (Required) Relative path from the global config root to the resource `.xcaf` file.

## Behavior

When a project declares `extends:`, the compiler:

1. Parses the global config at the referenced path.
2. Marks all global resources with `Inherited = true`.
3. Merges global resources into the project's compiled resource set.
4. Strips inherited resources from local project file output (they are not re-emitted as new `.xcaf` files).

Inherited resources can be overridden by re-declaring the same `id` in the project config. The local declaration wins.

> [!WARNING]
> Global configs are not validated in isolation — `xcaffold validate` always requires a project root. Run `xcaffold validate --from-root <project-dir>` to validate a project that extends a global.
