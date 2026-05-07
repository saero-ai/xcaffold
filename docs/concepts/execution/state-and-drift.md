---
title: "State Files and Drift Detection"
description: "How xcaffold tracks compilation state, detects drift, and organizes state files per blueprint"
---

# State Files and Drift Detection

xcaffold uses state files to track what was compiled, when, and from which sources. This enables drift detection without relying on git or file timestamps.

## The .xcaffold/ Directory

The `.xcaffold/` directory is xcaffold's machine-local state store. It is created by `xcaffold init` when a new project is bootstrapped.

`xcaffold apply` writes the state file (`.xcaffold/project.xcaf.state`) after each successful compilation. It also automatically appends `.xcaffold/` to the project's `.gitignore` on first run if the entry is not already present.

The directory contains:

- `project.xcaf` — the compiled manifest (written by `init`)
- One state file per blueprint, plus the default (written by `apply`)
- `schemas/` — field reference companion files (written by `init`)

```
.xcaffold/
  project.xcaf              # project manifest (created by xcaffold init)
  project.xcaf.state        # default state (created by xcaffold apply)
  backend.xcaf.state        # xcaffold apply --blueprint backend
  frontend.xcaf.state       # xcaffold apply --blueprint frontend
  schemas/                 # field reference docs (created by xcaffold init)
```

The `.xcaffold/` gitignore entry is added automatically by `xcaffold apply`. You do not need to add it manually, but you can if you want it committed before the first apply.

## State File Schema

Each state file records the source inputs and per-target outputs of a compilation:

```yaml
version: 1
xcaffold-version: "1.0.0"
blueprint: ""
source-files:
  - path: project.xcaf
    hash: "sha256:abc123..."
  - path: xcaf/agents/developer.xcaf
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
| `source-files` | Shared | All `.xcaf` inputs regardless of target. Same sources compile to all targets. |
| `targets.<name>.artifacts` | Per-target | Output file hashes for a specific target. |
| `blueprint` | Top-level | Empty string for default compilation, or the blueprint name. |

### Supporting File Tracking

Skill subdirectory files (`references/`, `scripts/`, `assets/`, `examples/`) are tracked as individual artifacts in the state file alongside the main `SKILL.md` entry. Each supporting file's content is hashed with SHA-256 independently — for example, a skill with two reference files produces three artifact entries: `skills/<id>/SKILL.md`, `skills/<id>/references/api-spec.md`, and `skills/<id>/references/conventions.md`.

When a supporting file is added, removed, or modified, `xcaffold status` reports it as artifact drift for the affected provider. The same orphan cleanup logic applies: if a supporting file is removed from the source `xcaf/skills/<id>/` directory, the corresponding output file is deleted on the next `xcaffold apply` and its entry is removed from the state file.

## Source Drift vs. Artifact Drift

xcaffold distinguishes two types of divergence:

**Source drift** — one or more `.xcaf` files changed since the last apply. The compiled output is stale. Fix: run `xcaffold apply`.

**Artifact drift** — a compiled output file was manually edited on disk. The file no longer matches what xcaffold generated. Fix: run `xcaffold apply --force` to overwrite, or revert the manual edit.

`xcaffold status` reports both types with per-file detail, using these labels:

| Label | Type | Meaning |
|-------|------|---------|
| `synced` | Artifact | All tracked files match the recorded hashes. |
| `modified` | Artifact | File exists but its content differs from the recorded hash. |
| `missing` | Artifact | File is tracked in the state file but does not exist on disk. |
| `source changed` | Source | A `.xcaf` file changed since the last apply. Compiled output is stale. |
| `source removed` | Source | A `.xcaf` file tracked at last apply no longer exists on disk. |
| `new source` | Source | A `.xcaf` file exists that was not present at the last apply. |

## xcaffold status

Detailed per-file hash comparison between the state file and current disk contents. Reports drift labels per file, grouped by provider.

See [xcaffold status](../reference/commands/diagnostic/status.md) for flags, sample output, and exit codes.

## Design Decisions

### No Remote State

xcaffold manages developer tooling configurations, not production infrastructure. 

xcaffold output is deterministic: given the same `.xcaf` sources, `xcaffold apply` always produces the same files. There is no shared mutable reality to protect. The `.xcaf` source files in git are the shared truth. Each developer's state file is just a local cache of what was last compiled — it can be regenerated from source at any time with `xcaffold apply --force`.

Design consequences:

- `.xcaf` source files live in git — that is the shared truth between developers
- State files are machine-local; each developer has their own
- No state locking, no conflict resolution, no remote backend needed
- Rebuilding state is free: `xcaffold apply --force` regenerates everything from source
- In CI, just run `xcaffold apply` to regenerate — no state file synchronisation required

### SHA-256 for Content-Addressable Verification

State files record SHA-256 hashes rather than file timestamps. Timestamps are unreliable across git checkouts, filesystem copies, and CI clones. SHA-256 guarantees that drift detection reflects actual content changes, not metadata differences. The same source files always produce the same hash, making state verification deterministic across machines.

## When This Matters

**CI pipelines checking for drift** — run `xcaffold status --no-color` in CI to assert that compiled output is in sync with source. A non-zero exit code means a developer committed a manual edit to `.claude/` or `.cursor/` without re-running `xcaffold apply`. Treat it as a failing check.

**Teams where developers manually edit compiled output** — xcaffold detects these edits as artifact drift on the next `status` or `apply`. Without state tracking, silent divergence between source and output accumulates undetected.

**Multi-provider projects where different providers drift independently** — each provider (`claude`, `cursor`, `gemini`, etc.) has its own artifact list in the state file. A manual edit to `.claude/agents/developer.md` does not affect the `cursor` drift status. `xcaffold status` reports per-provider so you know exactly which provider is out of sync.

## Related

- [Blueprints](blueprints.md)
- [xcaffold status](../reference/commands/diagnostic/status.md)
