---
title: "kind: blueprint"
description: "Defines resource selection and inheritance for compilation. Source: `xcaf/blueprints/<id>/blueprint.xcaf`."
---

# `kind: blueprint`

Defines a named subset of project resources to be processed by the xcaffold compiler. Blueprints enable environment-specific configurations (e.g., `production`, `ci`, `onboarding`) by selectively including agents, skills, and settings.

> **Required:** `kind`, `version`, `name`

## Source Directory

```
xcaf/blueprints/<name>/blueprint.xcaf
```

## Example Usage

```yaml
kind: blueprint
version: "1.0"
name: onboarding
description: Minimal resource set for new contributors.
extends: base
agents: [onboarding-guide]
rules: [style-guide]
settings: standard
```

## Field Reference

### Required Fields

| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | `string` | Unique identifier for the blueprint. Must match `[a-z0-9-]+`. |

### Optional Fields

#### Identity & Inheritance

| Field | Type | Description |
| :--- | :--- | :--- |
| `description` | `string` | Human-readable purpose of this blueprint. |
| `extends` | `string` | Name of another blueprint to inherit selections from. |

#### Resource Selectors

| Field | Type | Description |
| :--- | :--- | :--- |
| `agents` | `[]string` | Agent resource IDs to include in this blueprint. |
| `skills` | `[]string` | Skill resource IDs to include. |
| `rules` | `[]string` | Rule resource IDs to include. |
| `workflows` | `[]string` | Workflow resource IDs to include. |
| `mcp` | `[]string` | MCP server resource IDs to include. |
| `policies` | `[]string` | Policy resource IDs to include. |
| `memory` | `[]string` | Memory resource IDs to include. |
| `contexts` | `[]string` | Context resource IDs to include. |

#### Singleton Selectors

| Field | Type | Description |
| :--- | :--- | :--- |
| `settings` | `string` | Name of the settings block to use. |
| `hooks` | `string` | Name of the hooks block to use. |

#### Multi-Target

| Field | Type | Description |
| :--- | :--- | :--- |
| `targets` | `[]string` | Restricts this blueprint to specific provider targets. When absent, falls through to project targets or `--target` flag. |

## Filesystem-as-Schema

When a blueprint is defined at `xcaf/blueprints/<id>/blueprint.xcaf`, Xcaffold automatically infers:
- **kind**: `blueprint` derived from the `blueprints/` directory.
- **name**: `<id>` derived from the directory segment between the kind and the filename.

## Behavior

1.  **Selection Logic**: Only resources explicitly listed in the active blueprint (or its parents) are processed.
2.  **Merging**: Lists are concatenated and deduplicated. Singletons (`settings`, `hooks`) use the value from the child-most blueprint.
3.  **Invocation**: Specify the active blueprint via `xcaffold apply --blueprint <name>`.
