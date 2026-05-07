---
title: "kind: registry"
description: "Machine-wide project index. Source: ~/.xcaffold/registry.xcf. Managed by xcaffold init and apply."
---

# `kind: registry`

The `registry` kind defines the machine-wide index of all `xcaffold` projects initialized on the system. It is stored at `~/.xcaffold/registry.xcf` and is managed automatically by the CLI.

> [!IMPORTANT]
> This is a system-level resource. While it can be edited manually, it is primarily managed via `xcaffold init`, `xcaffold apply`, and `xcaffold registry` commands.

## Example Usage

```yaml
kind: registry
projects:
  - name: xcaffold-cli
    path: /Users/user/projects/xcaffold
    config_directory: .
    registered: 2026-05-01T10:00:00Z
    last_applied: 2026-05-05T18:00:00Z
    targets: [claude, cursor, gemini]
```

## Argument Reference

### Required Arguments

| Argument | Type | Description |
| :--- | :--- | :--- |
| `projects` | `[]Project` | List of all projects registered on the current machine. |

### Project Fields

| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | `string` | Unique name for the project. |
| `path` | `string` | Absolute path to the project root. |
| `config_directory` | `string` | Relative path from `path` to the directory containing `project.xcf`. |
| `targets` | `[]string` | List of compilation targets configured for this project. |
| `registered` | `timestamp` | UTC timestamp of when the project was first initialized. |
| `last_applied` | `timestamp` | UTC timestamp of the last successful `xcaffold apply`. |

## Behavior

1.  **Automatic Registration**: `xcaffold init` and `xcaffold apply` automatically add the current project to the registry.
2.  **Project Resolution**: When using `xcaffold apply --project <name>`, the CLI uses the registry to locate the project's configuration directory.
3.  **Global Scope Discovery**: The registry is used during `xcaffold apply --global` to discover project-specific agents or skills that should be indexed globally.

## Storage

The registry is stored as a plain YAML file at `~/.xcaffold/registry.xcf`. It does not use the standard `.xcf` frontmatter+body format; it is a single YAML document.
