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

### Supporting File Tracking

Skill subdirectory files (`references/`, `scripts/`, `assets/`, `examples/`) are tracked as individual artifacts in the state file alongside the main `SKILL.md` entry. Each supporting file's content is hashed with SHA-256 independently — for example, a skill with two reference files produces three artifact entries: `skills/<id>/SKILL.md`, `skills/<id>/references/api-spec.md`, and `skills/<id>/references/conventions.md`.

When a supporting file is added, removed, or modified, `xcaffold diff` reports it as artifact drift for the affected target. The same orphan cleanup logic applies: if a supporting file is removed from the source `xcf/skills/<id>/` directory, the corresponding output file is deleted on the next `xcaffold apply` and its entry is removed from the state file.

## Source Drift vs. Artifact Drift

xcaffold distinguishes two types of divergence:

**Source drift** — one or more `.xcf` files changed since the last apply. The compiled output is stale. Fix: run `xcaffold apply`.

**Artifact drift** — a compiled output file was manually edited on disk. The file no longer matches what xcaffold generated. Fix: run `xcaffold apply --force` to overwrite, or revert the manual edit.

`xcaffold diff` reports both types with per-file detail.

## xcaffold diff

Detailed per-file hash comparison between the state file and current disk contents. Reports added, removed, and modified artifacts for each target.

See [CLI Reference](../reference/cli.md) for flags and output format.

## Design Decisions

### No Remote State

xcaffold manages developer tooling configurations, not production infrastructure. Design consequences:

- `.xcf` source files live in git — that is the shared truth between developers
- State files are machine-local; each developer has their own
- No state locking, no conflict resolution, no remote backend
- Rebuilding state is cheap: `xcaffold apply --force` regenerates everything from source

### SHA-256 for Content-Addressable Verification

State files record SHA-256 hashes rather than file timestamps. Timestamps are unreliable across git checkouts, filesystem copies, and CI clones. SHA-256 guarantees that drift detection reflects actual content changes, not metadata differences. The same source files always produce the same hash, making state verification deterministic across machines.

## When This Matters

**CI pipelines checking for drift** — run `xcaffold diff` in CI to assert that compiled output is in sync with source. A non-zero exit code means a developer committed a manual edit to `.claude/` or `.cursor/` without re-running `xcaffold apply`. Treat it as a failing check.

**Teams where developers manually edit compiled output** — xcaffold detects these edits as artifact drift on the next `diff` or `apply`. Without state tracking, silent divergence between source and output accumulates undetected.

**Multi-target projects where different providers drift independently** — each target (`claude`, `cursor`, `gemini`, etc.) has its own artifact list in the state file. A manual edit to `.claude/agents/developer.md` does not affect the `cursor` drift status. `xcaffold diff` reports per-target so you know exactly which provider is out of sync.

## Related

- [Blueprints](blueprints.md)
- [CLI Reference](../reference/cli.md)
