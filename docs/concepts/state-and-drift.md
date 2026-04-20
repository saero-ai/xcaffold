---
title: "State Files and Drift Detection"
description: "How xcaffold tracks compilation state, detects drift, and organizes state files per blueprint"
---

# State Files and Drift Detection

xcaffold uses state files to track what was compiled, when, and from which sources. This enables drift detection without relying on git or file timestamps.

## The .xcaffold/ Directory

The `.xcaffold/` directory is xcaffold's machine-local state store. It is:

- Gitignored — each developer maintains their own state
- Auto-created by `xcaffold apply`
- Contains one state file per blueprint, plus the default

```
.xcaffold/
  project.xcf.state      # default (no --blueprint flag)
  backend.xcf.state      # xcaffold apply --blueprint backend
  frontend.xcf.state     # xcaffold apply --blueprint frontend
```

Add to `.gitignore`:

```
.xcaffold/
```

## State File Schema

Each state file records the source inputs and per-target outputs of a compilation:

```yaml
version: 1
xcaffold-version: "1.0.0"
blueprint: ""
source-files:
  - path: project.xcf
    hash: "sha256:abc123..."
  - path: xcf/agents/developer.xcf
    hash: "sha256:def456..."
targets:
  claude:
    last-applied: "2026-04-20T01:33:00Z"
    artifacts:
      - path: agents/developer.md
        hash: "sha256:789abc..."
  cursor:
    last-applied: "2026-04-20T01:30:00Z"
    artifacts:
      - path: agents/developer.md
        hash: "sha256:345ghi..."
```

| Field | Scope | Purpose |
| :--- | :--- | :--- |
| `source-files` | Shared | All `.xcf` inputs regardless of target. Same sources compile to all targets. |
| `targets.<name>.artifacts` | Per-target | Output file hashes for a specific target. |
| `blueprint` | Top-level | Empty string for default compilation, or the blueprint name. |

## Source Drift vs. Artifact Drift

xcaffold distinguishes two types of divergence:

**Source drift** — one or more `.xcf` files changed since the last apply. The compiled output is stale. Fix: run `xcaffold apply`.

**Artifact drift** — a compiled output file was manually edited on disk. The file no longer matches what xcaffold generated. Fix: run `xcaffold apply --force` to overwrite, or revert the manual edit.

`xcaffold diff` reports both types with per-file detail. `xcaffold status` reports a high-level summary.

## xcaffold status

Displays active blueprint, per-target compilation freshness, and drift indicators at a glance.

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

For a walkthrough, see [Checking Project Status](../how-to/checking-project-status.md).

## xcaffold diff

Detailed per-file hash comparison between the state file and current disk contents. Reports added, removed, and modified artifacts for each target.

See [CLI Reference](../reference/cli.md) for flags and output format.

## No Remote State

xcaffold manages developer tooling configurations, not production infrastructure. Design consequences:

- `.xcf` source files live in git — that is the shared truth between developers
- State files are machine-local; each developer has their own
- No state locking, no conflict resolution, no remote backend
- Rebuilding state is cheap: `xcaffold apply --force` regenerates everything from source

## Related

- [Blueprints](blueprints.md)
- [CLI Reference](../reference/cli.md)
- [Checking Project Status](../how-to/checking-project-status.md)
