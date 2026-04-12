---
title: "Configuration Scopes"
description: "Understanding Implicit Global Inheritance and runtime scopes."
---

# Configuration Scopes

When you operate an AI coding assistant, it typically aggregates configurations from two independently maintained directories: your **global user profile** (e.g., `~/.cursor/`) and your **local project repository** (e.g., `./.cursor/`).

Because `xcaffold` acts as a compiler for these configurations, it models this same duality:

1. **Global Scope** (`~/.xcaffold/`): Your personal, system-wide configurations representing *how you work* across all projects.
2. **Project Scope** (Directories containing `.xcf` files): The specific, contextual agent behaviors tailored *to the current codebase*.

---

## Multi-Kind Project Scope

A project scope is defined by exactly one `kind: project` document. This document declares the project name, targets, and resource references (bare name lists of agents, skills, rules, etc.). The actual resource definitions live in separate `.xcf` files under `xcf/` subdirectories.

`ParseDirectory` discovers all `.xcf` files recursively from the project root, parses each document, and routes it by `kind:`:

| Kind | Role |
|---|---|
| `project` | Project manifest — name, targets, child resource refs. Exactly 1 required. |
| `hooks` | Standalone hooks with `events:` wrapper |
| `settings` | Standalone settings |
| `config` | Legacy monolithic format (backward compatible) |

All discovered resource documents are merged into a single `ResourceScope` with strict deduplication. If the same resource ID appears in two files, parsing fails immediately with a duplicate ID error.

### Global Scope

Resources in `~/.xcaffold/` define the global scope. These are loaded via `loadGlobalBase` and provide system-wide defaults. Project-scope resources override global resources by ID during compilation.

---

## Implicit Global Inheritance

xcaffold implicitly evaluates constraints from both contexts at parse-time. Whenever xcaffold evaluates a project configuration, it automatically scans and parses your global directory (`~/.xcaffold/`) as a baseline template behind the scenes.

This provides two massive architectural benefits for developers:

1. **Cross-Reference Safety**: An agent explicitly defined in a local enterprise project can harmlessly `instructions_file:` reference a skill that exists purely in the user's global repository. The xcaffold compiler parses both spaces transparently, resolving relationships without emitting broken links or parser failures.
2. **Holistic Topology**: When running `xcaffold graph`, developers instantly receive a combined, unified view of the entire agentic topology—precisely what the underlying AI (Cursor, Claude) will actually understand when the terminal starts.

### The Compiler Hard Boundary

Implicit global inheritance deliberately extends *only* to the parser boundary limitation.

When you execute an `xcaffold apply` inside your project, the compilation runtime securely **strips** global resources from the aggregated AST before physically generating files.

Global resources are intentionally **NOT** written to the local project output directories (like `.claude/` or `.cursor/`). Because every known agentic provider autonomously combines the user's global environment at inference time, duplicating physical files into the local workspace would cause duplicate instruction injection and pollute git history unnecessarily.

To synchronize global changes to disc, execute a global-scoped apply independent of your project:

```bash
xcaffold apply --global
```

## Override Mechanics

When a configuration hierarchy is resolved (whether through implicit global resolution or explicit `extends:` directives), xcaffold's internal parser applies the overriding file on top of the base file.

### Per-Resource Merge Behavior

| Resource type | Merge rule |
|---|---|
| `agents:` | Child entry replaces base entry per ID. IDs present only in the base are kept. |
| `skills:` | Child entry replaces base entry per ID. IDs present only in the base are kept. |
| `rules:` | Child entry replaces base entry per ID. IDs present only in the base are kept. |
| `mcp:` | Child entry replaces base entry per ID. IDs present only in the base are kept. |
| `workflows:` | Child entry replaces base entry per ID. IDs present only in the base are kept. |
| `hooks:` | Additive. Both base and child handlers are kept. Child handlers are appended to base handlers for each event. |
| `project.test:` | Field-by-field overlay within `ProjectConfig`. `cli_path`, `claude_path`, and `judge_model` are replaced individually only when the child sets a non-empty value. |
| `settings:` | Last file in the directory wins (single-settings-file convention). Inherited and merged via `extends:`. |
| `project.local:` | Machine-local settings override within `ProjectConfig`. Not inherited via `extends:`. Compiles to `settings.local.json`. |

### Compiler Scope Merge

When a project config defines resources at both root level (global scope, from `extends:` or implicit global loading) and inside the `project:` block (workspace scope), the compiler merges them before rendering. Workspace resources override global resources by ID:

| Resource level | Source | Priority |
|---|---|---|
| Root-level `agents:`, `skills:`, etc. | Global or inherited | Lower (base) |
| `project.agents:`, `project.skills:`, etc. (legacy `kind: config`) | Workspace-specific | Higher (override) |

> **Format note:** In `kind: project` format, workspace-scoped resources are defined as bare name lists at the top level (e.g. `agents:` lists agent names, with definitions in separate `kind: agent` files). In `kind: config` format, they are nested under the `project:` block as `project.agents:`, `project.skills:`, etc. Both formats produce the same merge behavior.

After merging, inherited resources (those originating from `extends:` chains) are stripped from the compilation output to prevent duplication.

### Additive Hook Pipelines

Unlike static models (agents, rules), runtime execution lifecycle hooks accumulate across the inheritance chain securely. If the global base defines a `PreToolUse` handler and the local project also defines a `PreToolUse` handler, both run chronologically. The child's project-specific handler fires immediately after the overarching global policy baseline.

```yaml
# ~/.xcaffold/base.xcf
hooks:
  PreToolUse:
    - hooks:
        - type: command
          command: "echo pre-tool-use from global baseline"

# ./hooks.xcf (kind: hooks — standalone format, recommended)
kind: hooks
version: "1.0"
events:
  PreToolUse:
    - hooks:
        - type: command
          command: "echo pre-tool-use from project override"
```

> **Legacy format:** In `kind: config`, hooks are nested under `project: hooks:`. The standalone `kind: hooks` format with `events:` is the primary format for new configurations.

The compiled evaluation contains both handlers in order: global base, then project override.

## Circular Dependency Prevention

xcaffold prevents endless configuration loops by tracking file dependencies during parsing.

An explicit error terminates compilation instantly if:
- Implicit `global` resolution references itself
- Graph trace detects topological loops during `extends: /path/to/base.xcf` resolution

Example compiler termination:
```
circular extends detected: "/abs/path/to/base.xcf"
```
