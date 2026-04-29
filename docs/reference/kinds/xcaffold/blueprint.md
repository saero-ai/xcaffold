---
title: "kind: blueprint"
description: "Names a resource subset for conditional or partial compilation. Produces no output files."
---

# `kind: blueprint`

Defines a named resource subset that selects which agents, skills, rules, workflows, MCP servers, policies, memory entries, contexts, settings, and hooks are included during compilation. Pass a blueprint with `--blueprint <name>` to limit the compiled output to that subset.

Blueprints produce **no output files** — they are evaluated purely during compilation target selection.

Uses **pure YAML format** (no frontmatter `---` delimiters).

> **Required:** `kind`, `version`, `name`

## Example Usage

### Minimal: frontend-only blueprint

```yaml
kind: blueprint
version: "1.0"
name: frontend-only
description: Compile only the react-developer agent and its dependencies.
agents:
  - react-developer
skills:
  - component-patterns
rules:
  - react-conventions
  - no-server-imports-in-ui
mcp:
  - browser-tools
```

Run:
```bash
xcaffold apply --blueprint frontend-only
```

### Extending a base blueprint

```yaml
kind: blueprint
version: "1.0"
name: frontend-ci
description: CI-safe blueprint — no hooks, no MCP servers.
extends: frontend-only
mcp: []
hooks: ""
```

## Argument Reference

The following arguments are supported:

- `name` — (Required) Unique blueprint identifier. Must match `[a-z0-9-]+`.
- `description` — (Optional) `string`. What scenario this blueprint is intended for.
- `extends` — (Optional) `string`. ID of another blueprint to inherit from. Inherited fields are merged and can be overridden.
- `agents` — (Optional) `[]string`. Agent IDs to include. Must match `name` in declared agent resources.
- `skills` — (Optional) `[]string`. Skill IDs to include.
- `rules` — (Optional) `[]string`. Rule IDs to include.
- `workflows` — (Optional) `[]string`. Workflow IDs to include.
- `mcp` — (Optional) `[]string`. MCP server IDs to include.
- `policies` — (Optional) `[]string`. Policy IDs to include.
- `memory` — (Optional) `[]string`. Memory entry IDs to include.
- `contexts` — (Optional) `[]string`. Context IDs to include. Controls which workspace prose renders to provider root files (CLAUDE.md, GEMINI.md, etc.).
- `settings` — (Optional) `string`. Settings ID to apply. Only one settings block is active per compilation.
- `hooks` — (Optional) `string`. Hooks ID to apply.

## Usage

```bash
# Apply only the resources declared in the frontend-only blueprint
xcaffold apply --blueprint frontend-only

# Validate using a CI-specific blueprint
xcaffold validate --blueprint frontend-ci

# Check drift status for a specific blueprint
xcaffold status --blueprint frontend-only
```

When no `--blueprint` flag is provided, xcaffold compiles all discovered resources.
