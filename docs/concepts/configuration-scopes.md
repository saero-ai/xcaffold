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

When a configuration arborescence is collapsed (whether through implicit global resolution or explicit `extends:` directives), xcaffold's internal parser applies the overriding file on top of the base file via the `mergeConfigOverride` directive.

### Per-Resource Merge Behavior

| Resource type | Merge rule |
|---|---|
| `agents:` | Child entry replaces base entry per ID. IDs present only in the base are kept. |
| `skills:` | Child entry replaces base entry per ID. IDs present only in the base are kept. |
| `rules:` | Child entry replaces base entry per ID. IDs present only in the base are kept. |
| `mcp:` | Child entry replaces base entry per ID. IDs present only in the base are kept. |
| `workflows:` | Child entry replaces base entry per ID. IDs present only in the base are kept. |
| `hooks:` | Additive. Both base and child handlers are kept. Child handlers are appended to base handlers for each event. |
| `test:` | Field-by-field overlay. `cli_path`, `claude_path`, and `judge_model` are replaced individually only when the child sets a non-empty value. |
| `settings:` / `local:` | Last file in the directory wins (single-settings-file convention). Not inherited via `extends:`. |

### Additive Hook Pipelines

Unlike static models (agents, rules), runtime execution lifecycle hooks accumulate across the inheritance chain securely. If the global base defines a `PreToolUse` handler and the local project also defines a `PreToolUse` handler, both run chronologically. The child's project-specific handler fires immediately after the overarching global policy baseline.

```yaml
# ~/.xcaffold/base.xcf
hooks:
  PreToolUse:
    - hooks:
        - type: command
          command: "echo pre-tool-use from global baseline"

# ./scaffold.xcf
hooks:
  PreToolUse:
    - hooks:
        - type: command
          command: "echo pre-tool-use from project override"
```

The compiled evaluation contains both handlers in order: global base, then project override.

## Circular Dependency Prevention

xcaffold strictly guarantees that a deterministic directed acyclic graph (DAG) avoids endless topological loop resolution via the parser's state machine.

An absolute path trace mapping is utilized recursively during load times. An explicit error terminates compilation instantly if:
- Implicit `global` resolution references itself
- Graph trace detects topological loops during `extends: /path/to/base.xcf` resolution

Example compiler termination:
```
circular extends detected: "/abs/path/to/base.xcf"
```
