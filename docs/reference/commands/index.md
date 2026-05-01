---
title: Commands
description: "Reference guide for all Xcaffold CLI commands."
---

# Core CLI Concepts

The Xcaffold CLI translates your YAML configuration logic into strict native artifact dependencies via a standard command set.

## Command Scopes

By default, operations strictly interact within the specific project configuration contextual to your `$CWD` resolution path. When the `--global` flag explicitly enforces scope, commands automatically evaluate your `.xcaffold/global.xcf` directory map instead, executing identically. 

## Global Flags

All top-level commands accept the following persistent flags inherited across the CLI pipeline:

| Flag | Default | Description |
|---|---|---|
| `--config <path>` | `""` | Provide direct execution paths explicitly targeting a static `project.xcf` resource (or directories representing multi-file dependencies) preventing local directory walk fallback routines. |
| `-g, --global` | `false` | Instructs the CLI execution layer resolving operations directly against user-wide structural constraints inside `~/.xcaffold/global.xcf`. |
| `--version` | — | Returns binary metadata execution structures specifying `<version> (commit: <sha>, date: <date>)`. |

## Available Commands

### Lifecycle Commands
[**`init`**](./lifecycle/init.md) — Initialize a new repository with default schemas or templates.
[**`apply`**](./lifecycle/apply.md) — Compile .xcf resources into provider-native agent configuration files.
[**`import`**](./lifecycle/import.md) — Snapshot native legacy workspaces seamlessly translating configurations into Xcaffold abstractions.

### Inspection & State
[**`status`**](./diagnostic/status.md) — Calculate drift and compilation integrity directly validating target health.
[**`graph`**](./diagnostic/graph.md) — Formulate dependencies topologies rendering nodes and schemas dynamically.
[**`list`**](./diagnostic/list.md) — Audit and list strictly defined execution entities resolving structural paths directly.

### Utilities & Preview
[**`validate`**](./utility/validate.md) — Check .xcf syntax, cross-references, structural invariants, and policy compliance.
[**`test`**](./utility/test.md) _(Preview)_ — Execute explicit behavioral traces simulating execution boundaries programmatically via an LLM.
[**`export`**](./utility/export.md) _(Preview)_ — Wrap independently scaffolded resources into distributable artifact plugin bundles.
