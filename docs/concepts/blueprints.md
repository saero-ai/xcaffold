---
title: "Blueprints"
description: "Named resource subsets for targeted compilation — define blueprint files, select agents and skills, and compile project subsets"
---

# Blueprints

Blueprints let you define named subsets of your project resources for targeted compilation. Instead of compiling everything, you compile exactly what a specific role or workflow needs.

## Three Compilation Scopes

| Scope | Flag | What Compiles | State File Path |
| :--- | :--- | :--- | :--- |
| Global | `--global` | User-wide personal config (`~/.xcaffold/global.xcf`) | `~/.xcaffold/<state>.xcf.state` |
| Project | (default) | All resources in `xcf/` | `.xcaffold/project.xcf.state` |
| Blueprint | `--blueprint <name>` | Named resource subset selected by a `kind: blueprint` file | `.xcaffold/<blueprint-name>.xcf.state` |

Note: Scopes are mutually exclusive. One per invocation.

## Blueprint Files

- `kind: blueprint` frontmatter fields: `name` (required), `description`, `active`, `agents`, `skills`, `rules`, `settings`, `hooks`
- Where blueprint files live: `xcf/blueprints/` (recommended), or anywhere under `xcf/`
- They are version-controlled source files, not generated

## Defining a Blueprint

```yaml
# xcf/blueprints/backend.xcf
kind: blueprint
name: backend
description: Backend development configuration
active: true
agents:
  - developer
  - reviewer
skills:
  - tdd
rules:
  - testing
  - api-conventions
settings: backend-settings
hooks: ci-hooks
```

```yaml
# xcf/blueprints/frontend.xcf
kind: blueprint
name: frontend
description: Frontend development configuration
agents:
  - ui-developer
skills:
  - component-testing
rules:
  - accessibility
```

## The active Field

- At most one blueprint may have `active: true`
- Meaning: declared team default, visible in `xcaffold status` output
- Does NOT affect `xcaffold apply` behavior — you still pass `--blueprint` explicitly
- If no blueprint is active, `xcaffold status` shows "none"

## Transitive Dependencies

- Selecting an agent auto-includes its declared `skills:`, `rules:`, and `mcp:` from the agent definition
- Explicit blueprint fields override: if blueprint lists `skills: [tdd]`, only `tdd` is included regardless of agent transitive deps
- `xcaffold list --blueprint <name> --resolved` shows the fully resolved resource set
- Circular reference detection prevents infinite loops

## Compiling a Blueprint

Example:
```
$ xcaffold apply --blueprint backend
[backend] Compiling 2 agents, 1 skill, 2 rules for target: claude
[backend] ✓ Apply complete. .xcaffold/backend.xcf.state updated.
```

Note: `xcaffold apply` (no flag) always compiles ALL resources. `--blueprint` is opt-in narrowing.

## Per-Blueprint State Files

- Each blueprint gets its own state file: `.xcaffold/<blueprint-name>.xcf.state`
- Default (no blueprint): `.xcaffold/project.xcf.state`
- State files are gitignored, machine-local
- Each state file tracks source hashes and per-target artifacts independently

## Named Settings and Hooks

- `kind: settings` with a `name:` field defines a named settings configuration
- `kind: hooks` with a `name:` field defines a named hooks configuration
- Blueprint selects which to use via `settings: <name>` and `hooks: <name>`
- If omitted, the default (unnamed) settings/hooks apply

## Directory Layout

```
my-project/
  project.xcf
  xcf/
    agents/
      developer.xcf
      reviewer.xcf
      ui-developer.xcf
    skills/
      tdd/tdd.xcf
      component-testing/component-testing.xcf
    rules/
      testing.xcf
      api-conventions.xcf
      accessibility.xcf
    blueprints/
      backend.xcf
      frontend.xcf
    settings/
      backend-settings.xcf
    hooks/
      ci-hooks.xcf
```

## Related

- [State Files and Drift Detection](state-and-drift.md)
- [CLI Reference](../reference/cli.md)
- [Multi-Kind Documents](../reference/multi-kind.md)
