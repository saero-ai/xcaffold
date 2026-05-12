---
title: "kind: registry"
description: "Machine-wide project index. Source: ~/.xcaffold/registry.xcaf. Managed by xcaffold init and apply."
---

# `kind: registry`

The `registry` kind defines the machine-wide index of all `xcaffold` projects initialized on the system. It is stored at `~/.xcaffold/registry.xcaf` and is managed automatically by the CLI.

> [!IMPORTANT]
> This is a system-level resource. While it can be edited manually, it is primarily managed via `xcaffold init`, `xcaffold apply`, and the `xcaffold registry` command (hidden — not shown in `--help` output).

> **Required:** `kind`, `projects`

## Source Directory

```
~/.xcaffold/registry.xcaf
```

## Example Usage

```yaml
kind: registry
projects:
  - name: xcaffold-cli
    path: ~/projects/xcaffold
    config_directory: .
    registered: 2026-05-01T10:00:00Z
    last_applied: 2026-05-05T18:00:00Z
    targets: [claude, cursor, gemini]
```

## Field Reference

### Required Fields

| Field | Type | Description |
| :--- | :--- | :--- |
| `projects` | `[]Project` | List of all projects registered on the current machine. |

### Project Entry

Each item in `projects` is a project record with the following fields.

#### Required Project Fields

| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | `string` | Unique name for the project. |
| `path` | `string` | Absolute path to the project root. |
| `registered` | `timestamp` | UTC timestamp of when the project was first initialized. |

#### Optional Project Fields

| Field | Type | Description |
| :--- | :--- | :--- |
| `config_directory` | `string` | Relative path from `path` to the directory containing `project.xcaf`. Omitted when empty. |
| `targets` | `[]string` | List of compilation targets configured for this project. Omitted when empty. |
| `last_applied` | `timestamp` | UTC timestamp of the last successful `xcaffold apply`. Omitted until first apply. |

## Behavior

1.  **Automatic Registration**: `xcaffold init` and `xcaffold apply` automatically add the current project to the registry.
2.  **Project Resolution**: When using `xcaffold apply --project <name>`, the CLI uses the registry to locate the project's configuration directory.
3.  **Global Compilation**: `xcaffold apply --global` compiles the user-wide configuration at `~/.xcaffold/global.xcaf`, aggregating resources discovered from provider directories (e.g., `~/.claude/`, `~/.gemini/`). The registry is not consulted during global compilation.

## Storage

The registry is stored as a plain YAML file at `~/.xcaffold/registry.xcaf`. It does not use the standard `.xcf` frontmatter+body format; it is a single YAML document.
