---
title: "Project Manifest Format"
description: "Reference for the project.xcf manifest file — metadata fields, targets, instructions, and advisory resource refs"
---

# Project Manifest Format

## Overview

`kind: project` is the required project manifest. It declares:

- Project identity (name, description, author, repository)
- Compilation targets
- Instructions (inline or file-referenced)
- Test configuration
- Local settings overrides

It does **not** contain resource definitions. Agents, skills, rules, and other resources are defined in separate `.xcf` files under `xcf/` and discovered automatically by `ParseDirectory`.

## Recommended Filename

`project.xcf`

The parser identifies the project manifest by `kind: project`, not by filename. Any filename works, but `project.xcf` is the convention and what `xcaffold init` generates.

## Field Reference

| Field | Type | Required | Description |
|---|---|---|---|
| `kind` | `string` | **Required** | Must be `"project"`. |
| `version` | `string` | **Required** | Schema version. Current: `"1.0"`. |
| `name` | `string` | **Required** | Canonical project name. Used in registry, graph output, and state files. |
| `description` | `string` | Optional | Human-readable project description. |
| `author` | `string` | Optional | Project author or maintainer. |
| `homepage` | `string` | Optional | Project homepage URL. |
| `repository` | `string` | Optional | Source repository URL. |
| `license` | `string` | Optional | SPDX license identifier. |
| `backup-dir` | `string` | Optional | Custom directory for `--backup` output. Default: `.<target>_bak_<timestamp>`. |
| `targets` | `[]string` | Optional | Compilation targets (e.g. `["claude", "cursor"]`). |
| `instructions` | `string` | Optional | Inline instructions injected into all compiled outputs. Mutually exclusive with `instructions-file`. |
| `instructions-file` | `string` | Optional | Path to a file containing instructions. Mutually exclusive with `instructions`. |
| `extends` | `string` | Optional | Path to a parent `.xcf` config for inheritance. |
| `test` | `TestConfig` | Optional | Configuration for `xcaffold test`. |
| `local` | `SettingsConfig` | Optional | Local settings overrides compiled to `settings.local.json`. |
| `blueprints` | `[]string` | Optional | Advisory list of blueprint names. Not used for discovery. |

## Advisory Resource Ref Lists

The following fields are accepted by the parser but **do not affect compilation**:

| Field | Type | Purpose |
|---|---|---|
| `agents` | `[]string` | Advisory. `xcaffold validate` warns if out of sync with `xcf/`. |
| `skills` | `[]string` | Advisory. Same semantics. |
| `rules` | `[]string` | Advisory. Same semantics. |
| `workflows` | `[]string` | Advisory. Same semantics. |
| `mcp` | `[]string` | Advisory. Same semantics. |
| `policies` | `[]string` | Advisory. Same semantics. |

`ParseDirectory` discovers resources by scanning `xcf/` recursively. These lists exist for documentation purposes and IDE tooling that may read the manifest. They are always safe to omit.

## Example — Minimal

```yaml
kind: project
version: "1.0"
name: my-app
targets:
  - claude
```

## Example — With Instructions

```yaml
kind: project
version: "1.0"
name: my-api
description: REST API backend
targets:
  - claude
  - cursor
instructions: |
  You are a backend developer working on my-api.
  Follow the project conventions in xcf/rules/.
```

Or with a file reference:

```yaml
kind: project
version: "1.0"
name: my-api
targets:
  - claude
instructions-file: INSTRUCTIONS.md
```

## Example — With Test Config

```yaml
kind: project
version: "1.0"
name: my-app
targets:
  - claude
test:
  cli-path: claude
  judge-model: claude-sonnet-4-6
```

## Example — With Local Overrides

```yaml
kind: project
version: "1.0"
name: my-app
targets:
  - claude
local:
  model: haiku
```

Local settings compile to `settings.local.json` (gitignored) and override the main `settings.json` for developer-specific preferences.

## Related

- [Configuration Scopes](../concepts/configuration-scopes.md)
- [Schema Reference](schema.md)
- [Multi-Kind Documents](multi-kind.md)
