---
title: "Project Registry"
description: "How xcaffold tracks projects across your machine for cross-project awareness and workspace management."
---

# Project Registry

xcaffold maintains a registry of all managed projects at `~/.xcaffold/registry.xcaf`. The registry provides cross-project awareness — the ability to reference, inspect, and manage projects by name from any directory.

## Why a Registry

Without a registry, xcaffold operates in isolation per project. Each `xcaffold apply` or `xcaffold status` requires navigating to the project directory. The registry enables:

- **Name-based resolution.** Reference a project by its registered name instead of its filesystem path.
- **Cross-project visibility.** `xcaffold registry list` shows all managed projects, their targets, and their health status (`ok`, `stale`, `orphan`) from any working directory.
- **Stale entry detection.** Projects that have been moved or deleted are surfaced as `stale` rather than silently failing.

## How Projects Get Registered

Projects are registered automatically during normal workflow:

| Action | Registration Effect |
|--------|-------------------|
| `xcaffold init` | Registers the project with its name and detected targets. |
| `xcaffold apply` | Registers (or updates) the project and records the `last_applied` timestamp. |
| `xcaffold registry add <path>` | Explicitly registers a project from any directory. |

Registration is idempotent — running `init` or `apply` on an already-registered project updates its metadata without creating duplicates.

## Registry Location

The registry file lives at `~/.xcaffold/registry.xcaf`. This path is determined by `GlobalHome()`, which defaults to `~/.xcaffold/` and can be overridden via the `XCAFFOLD_HOME` environment variable.

The `~/.xcaffold/` directory is created automatically on the first run of any xcaffold command.

## Project Status

Each registered project has one of three statuses:

| Status | Meaning |
|--------|---------|
| `ok` | Path exists and contains xcaf configuration files. |
| `stale` | Path no longer exists on disk. The project was likely deleted or moved. |
| `orphan` | Path exists but contains no `xcaf/` directory or `project.xcaf`. |

Use `xcaffold registry prune` to remove stale entries. Orphan entries may indicate a project that was partially cleaned up or not yet initialized.

## Registry and Global Scope

The project registry and global scope (`~/.xcaffold/xcaf/global.xcaf`) are independent systems that share the `~/.xcaffold/` home directory:

- The **registry** tracks project locations and metadata.
- The **global scope** defines shared resources inherited implicitly by all projects.

They do not depend on each other. A project can be registered without using global scope, and global scope can exist without any registered projects.

## Housekeeping

The registry grows as projects are initialized. Over time, entries accumulate for projects that no longer exist. Use the registry management commands to keep it clean:

```bash
# See all projects and their status
xcaffold registry list

# Remove entries for deleted directories
xcaffold registry prune

# Preview what would be removed
xcaffold registry prune --dry-run

# Remove a specific project
xcaffold registry remove my-old-project

# Inspect a project's metadata
xcaffold registry info my-app
```

Pruning is always explicit. xcaffold never removes registry entries automatically, because a missing path might be a temporarily unmounted volume rather than a deleted project.
