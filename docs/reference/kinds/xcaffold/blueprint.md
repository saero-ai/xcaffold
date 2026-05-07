---
title: "kind: blueprint"
description: "Defines resource selection and inheritance for compilation. Source: `xcf/blueprints/<id>/blueprint.xcaf`."
---

# `kind: blueprint`

Defines a named subset of project resources to be processed by the xcaffold compiler. Blueprints enable environment-specific configurations (e.g., `production`, `ci`, `onboarding`) by selectively including agents, skills, and settings.

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

## Argument Reference

### Required Arguments

| Argument | Type | Description |
| :--- | :--- | :--- |
| `kind` | `string` | Must be `blueprint`. |
| `version` | `string` | Resource schema version (e.g., `"1.0"`). |
| `name` | `string` | Unique identifier for the blueprint. |

### Optional Arguments

#### Identity & Inheritance

| Argument | Type | Description |
| :--- | :--- | :--- |
| `description` | `string` | Human-readable purpose of this blueprint. |
| `extends` | `string` | Name of a parent blueprint to inherit selections from. |

#### Resource Selectors

These fields define which resources are included in the compilation. Lists are merged when using `extends`.

| Argument | Type | Description |
| :--- | :--- | :--- |
| `agents` | `[]string` | List of Agent IDs to include. |
| `skills` | `[]string` | List of Skill IDs to include. |
| `rules` | `[]string` | List of Rule IDs to include. |
| `workflows` | `[]string` | List of Workflow IDs to include. |
| `mcp` | `[]string` | List of MCP Server IDs to include. |
| `policies` | `[]string` | List of Policy IDs to include. |
| `memory` | `[]string` | List of Memory IDs to include. |
| `contexts` | `[]string` | List of Context IDs to include. |

#### Singleton Selectors

These fields accept a single ID and override any value inherited from a parent blueprint.

| Argument | Type | Description |
| :--- | :--- | :--- |
| `settings` | `string` | The ID of the Settings block to apply. |
| `hooks` | `string` | The ID of the Hooks block to apply. |

#### Multi-Target

| Argument | Type | Description |
| :--- | :--- | :--- |
| `targets` | `[]string` | Restricts this blueprint to specific provider targets (e.g., `claude`, `cursor`). |

## Filesystem-as-Schema

When a blueprint is defined at `xcf/blueprints/<id>/blueprint.xcaf`, Xcaffold automatically infers:
- **kind**: `blueprint` derived from the `blueprints/` directory.
- **name**: `<id>` derived from the directory segment between the kind and the filename.

## Behavior

1.  **Selection Logic**: Only resources explicitly listed in the active blueprint (or its parents) are processed.
2.  **Merging**: Lists are concatenated and deduplicated. Singletons (`settings`, `hooks`) use the value from the child-most blueprint.
3.  **Invocation**: Specify the active blueprint via `xcaffold apply --blueprint <name>`.
