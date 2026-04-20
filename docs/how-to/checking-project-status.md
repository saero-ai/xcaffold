---
title: "Checking Project Status"
description: "Use xcaffold status to inspect compilation freshness, detect drift, and view active blueprint context"
---

# Checking Project Status

## Quick Status Check

```
$ xcaffold status
Blueprint: backend (active)
Targets:   claude, cursor

  claude:  applied 2 hours ago, 9 artifacts (all clean)
  cursor:  applied 2 hours ago, 9 artifacts (1 drifted)

Sources:   7 files
  changed  xcf/skills/tdd/tdd.xcf    <- re-apply needed

Run 'xcaffold apply --blueprint backend' to sync.
```

Exit codes:

| Code | Meaning |
|------|---------|
| `0` | No drift detected — all targets clean, all sources unchanged. |
| `1` | Drift detected — source changed or artifact modified since last apply. |
| `2` | No state file found — project has never been applied. |

## Blueprint-Scoped Status

Check a specific blueprint:

```
$ xcaffold status --blueprint frontend
Blueprint: frontend
Targets:   claude

  claude:  applied 5 minutes ago, 4 artifacts (all clean)

Sources:   3 files (all clean)
```

Without `--blueprint`, status reads the default state file (`.xcaffold/project.xcf.state`). With `--blueprint <name>`, it reads `.xcaffold/<name>.xcf.state`.

## Understanding the Output

| Line | Meaning |
|------|---------|
| **Blueprint** | Which blueprint has `active: true` in its definition. Shows `none` if no blueprint is active. |
| **Targets** | Compilation targets recorded in the state file. |
| **Per-target line** | Last applied timestamp, artifact count, drift indicator (`all clean` or `N drifted`). |
| **Sources** | Count of tracked `.xcf` source files. Lists individually any that changed since last apply. |
| **Suggested command** | Shown only when drift is detected. |

## Drift: Status vs. Diff

| | `xcaffold status` | `xcaffold diff` |
|---|---|---|
| **Purpose** | Quick health check | Detailed file-level comparison |
| **Output** | Summary counts | Per-file hash differences |
| **Exit code** | Non-zero on any drift | Non-zero on any drift |
| **Use when** | Before committing, in CI gates | Investigating which files drifted |

## Reconciliation Warnings

Status may emit warnings for:

- **Orphaned state entries** — artifacts in the state file that no longer have a corresponding `.xcf` source. Indicates a resource was removed but not recompiled.
- **Unregistered resources** — `.xcf` files in `xcf/` with no matching entry in the state file. Indicates new resources that haven't been compiled yet.

Both resolve after running `xcaffold apply`.

## Related

- [CLI Reference — xcaffold status](../reference/cli.md)
- [State Files and Drift Detection](../concepts/state-and-drift.md)
