---
title: "xcaffold registry"
description: "Manage the global project registry — list, add, remove, prune, and inspect registered projects."
---

# xcaffold registry

Manage the global project registry at `~/.xcaffold/registry.xcaf`. The registry tracks all xcaffold-managed projects by name, path, targets, and timestamps.

Projects are registered automatically by `xcaffold init` and `xcaffold apply`. The `registry` command provides explicit management: listing, adding, removing, pruning stale entries, and inspecting project metadata.

**Usage:**

```
xcaffold registry [command]
```

Running `xcaffold registry` with no subcommand defaults to `registry list`.

## Subcommands

| Subcommand | Description |
|---|---|
| [`list`](#registry-list) | List all registered projects |
| [`add`](#registry-add) | Register a new project by path |
| [`remove`](#registry-remove) | Unregister a project by name or path |
| [`prune`](#registry-prune) | Remove entries for deleted directories |
| [`info`](#registry-info) | Show detailed metadata for a project |

---

## registry list

Display all registered projects in tabular format.

```
xcaffold registry list [flags]
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--json` | — | `bool` | `false` | Output as a JSON array of project objects. |
| `--verbose` | `-v` | `bool` | `false` | Include registration timestamp and config directory. |

### Output Columns

| Column | Description |
|--------|-------------|
| NAME | Project name (from `project.xcaf` or directory basename) |
| PATH | Absolute path to the project |
| TARGETS | Compilation targets or count (expanded in verbose mode) |
| LAST APPLIED | Timestamp of last `xcaffold apply`, or "never" |
| STATUS | `ok`, `stale` (path missing), or `orphan` (no xcaf files) |

Projects are sorted alphabetically by name.

### Examples

```bash
# List all projects
xcaffold registry list

# JSON output for scripting
xcaffold registry list --json

# Verbose with registration timestamps
xcaffold registry list -v
```

---

## registry add

Explicitly register a project at the given filesystem path.

```
xcaffold registry add PATH
```

### Behavior

1. Resolves `PATH` to an absolute path.
2. Verifies the path exists on disk. Returns an error if not.
3. Reads `project.xcaf` at the path to detect the project name and targets. Falls back to the directory basename if no `project.xcaf` is found.
4. If the path is already registered, updates the existing entry.
5. If a different project with the same name exists, appends a parent-directory suffix to avoid collision.

### Examples

```bash
# Register a project
xcaffold registry add ~/projects/my-app

# Register current directory
xcaffold registry add .
```

---

## registry remove

Remove a project from the registry by name or path. Does not delete any files on disk.

```
xcaffold registry remove NAME_OR_PATH
```

### Behavior

1. Matches by name first, then by absolute path.
2. Returns an error if no match is found.
3. Prints a confirmation with the removed project's name and path.

### Examples

```bash
# Remove by name
xcaffold registry remove my-app

# Remove by path
xcaffold registry remove ~/projects/my-app
```

---

## registry prune

Remove all registry entries whose paths no longer exist on disk.

```
xcaffold registry prune [flags]
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--dry-run` | — | `bool` | `false` | Show what would be removed without modifying the registry. |

### Output Format

```
Pruned: "old-project" (/path/to/deleted) — path does not exist
Pruned 1 of 5 projects.
```

When nothing needs pruning:

```
Registry is clean. 0 stale entries found.
```

### Examples

```bash
# Preview stale entries
xcaffold registry prune --dry-run

# Remove stale entries
xcaffold registry prune
```

---

## registry info

Display detailed metadata and filesystem status for a single registered project.

```
xcaffold registry info NAME_OR_PATH [flags]
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--json` | — | `bool` | `false` | Output as a JSON object. |

### Output Fields

| Field | Description |
|-------|-------------|
| Name | Registered project name |
| Path | Absolute path |
| Registered | ISO timestamp of initial registration |
| Last Applied | ISO timestamp of last apply, or "never" |
| Targets | Compilation targets (comma-separated) or "none" |
| Config Dir | Relative path to xcaf config within the project |
| Exists | Whether the path exists on disk |
| Has xcaf/ | Whether an `xcaf/` directory is present |
| Has project.xcaf | Whether a `project.xcaf` file is present |

### Examples

```bash
# Inspect by name
xcaffold registry info my-app

# JSON output for scripting
xcaffold registry info my-app --json

# Inspect by path
xcaffold registry info ~/projects/my-app
```

---

## Registry File

The registry is stored at `~/.xcaffold/registry.xcaf` as a `kind: registry` YAML document. This file is managed by xcaffold — do not edit it manually unless recovering from corruption.

```yaml
kind: registry
projects:
  - name: my-app
    path: /Users/dev/projects/my-app
    registered: 2026-05-01T10:00:00Z
    last_applied: 2026-05-20T14:30:00Z
    targets:
      - claude
      - gemini
```

If the registry file becomes corrupt, `xcaffold registry prune` or manual editing of `~/.xcaffold/registry.xcaf` can restore it.

## Global Flags

All subcommands inherit the root-level persistent flags:

| Flag | Default | Description |
|---|---|---|
| `--config <path>` | `""` | Path to `project.xcaf` (not typically used with registry commands). |
| `--no-color` | `false` | Disable ANSI color and UTF-8 glyphs. |
| `--verbose` | `false` | Show fidelity notes and policy warnings. |

## Notes

- The registry is project-agnostic. All subcommands work from any directory, not just inside an xcaffold project.
- Auto-registration on `init` and `apply` is preserved. Use `registry add` only for manual registration.
- `prune` is explicit, not automatic. Entries for temporarily unmounted volumes are not silently removed.
