---
title: "kind: registry"
description: "Internal project registry file at ~/.xcaffold/registry.xcaf. Not a compilable resource kind."
---

# `kind: registry`

The registry is an internal metadata file that tracks all xcaffold-managed projects on the local machine. It is stored at `~/.xcaffold/registry.xcaf` and managed by the CLI.

> **Internal kind.** Registry files are not parsed by the compiler, not included in `xcaffold apply` output, and cannot appear in the `xcaf/` source tree. They are operational metadata only.

## File Location

```
~/.xcaffold/registry.xcaf
```

Created automatically by `xcaffold init` or the first `xcaffold apply` when the global home directory is bootstrapped.

## File Format

```yaml
kind: registry
projects:
  - name: my-app
    path: /Users/dev/projects/my-app
    registered: 2026-05-01T10:00:00Z
    last_applied: 2026-05-20T14:30:00Z
    config_directory: "."
    targets:
      - claude
      - gemini
  - name: backend-api
    path: /Users/dev/projects/backend
    registered: 2026-05-10T08:00:00Z
    targets:
      - cursor
```

## Field Reference

### Document Fields

| Field | Type | Description |
|:------|:-----|:------------|
| `kind` | `string` | Must be `"registry"`. |
| `projects` | `[]Project` | Array of registered project entries. |

### Project Entry Fields

| Field | Type | Description |
|:------|:-----|:------------|
| `name` | `string` | Project name. Inferred from `project.xcaf` or directory basename. |
| `path` | `string` | Absolute filesystem path to the project root. |
| `registered` | `time` | ISO 8601 timestamp of initial registration. |
| `last_applied` | `time` | ISO 8601 timestamp of last successful `xcaffold apply`. Zero if never applied. |
| `config_directory` | `string` | Relative path to the xcaf config directory within the project. |
| `targets` | `[]string` | Compilation targets detected or declared for this project. |

## Lifecycle

| Event | Effect on Registry |
|-------|-------------------|
| `xcaffold init` | Registers the project if not already present. |
| `xcaffold apply` | Registers (or updates) the project and sets `last_applied`. |
| `xcaffold registry add <path>` | Explicitly adds a project entry. |
| `xcaffold registry remove <name>` | Removes the entry. Does not delete files on disk. |
| `xcaffold registry prune` | Removes entries whose `path` no longer exists. |

## Relationship to Other Kinds

The registry is **not** part of the compilation pipeline. It exists alongside the compiler, not inside it.

| Kind | Relationship |
|------|-------------|
| `project` | Registry entries point to directories containing `kind: project` manifests. |
| `global` | The global scope (`~/.xcaffold/xcaf/global.xcaf`) is independent of the registry. The registry tracks projects, not the global config. |
| `blueprint` | Registry entries record targets but not active blueprints. |

## Differences from Compilable Kinds

| Aspect | Compilable Kinds | `kind: registry` |
|--------|-----------------|-------------------|
| Location | `xcaf/` source tree | `~/.xcaffold/registry.xcaf` |
| Parsed by compiler | Yes | No (explicitly skipped) |
| Produces output files | Yes | No |
| User-authored | Yes | Machine-managed |
| Appears in `xcaffold list` | Yes | No |
| Has `version` field | Yes | No |

## Notes

- The registry uses atomic writes (temp file + rename) to prevent corruption under concurrent CLI invocations.
- The `XCAFFOLD_HOME` environment variable overrides the default `~/.xcaffold/` location. Used primarily for test isolation.
- Registry files with invalid YAML produce a descriptive error with recovery instructions.
