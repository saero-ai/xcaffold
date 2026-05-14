---
title: Commands
description: "Reference guide for all Xcaffold CLI commands."
---

# Core CLI Concepts

The Xcaffold CLI translates your YAML configuration logic into strict native artifact dependencies via a standard command set.

## Command Scopes

By default, operations strictly interact within the specific project configuration contextual to your `$CWD` resolution path. When the `--global` flag explicitly enforces scope, commands automatically evaluate your `.xcaffold/global.xcaf` directory map instead, executing identically. 

## Global Flags

All top-level commands accept the following persistent flags:

| Flag | Default | Description |
|---|---|---|
| `--config <path>` | `""` | Path to `project.xcaf` or a directory containing one. Overrides directory-walk resolution. |
| `-g, --global` | `false` | Operate on user-wide global config (`~/.xcaffold/global.xcaf`). |
| `--no-color` | `false` | Disable ANSI color and UTF-8 glyphs. Also honored via the `NO_COLOR` environment variable. |
| `--verbose` | `false` | Show fidelity notes and policy warnings during compilation. |
| `--xcaf <kind>` | `""` | Display the field schema for a resource kind (e.g., `agent`, `skill`, `rule`). |
| `--out <path>` | `"."` | Generate a template `.xcaf` file for the kind specified by `--xcaf`. |
| `--version` | — | Print version, commit SHA, and build date. |

## Available Commands

### Lifecycle

| Command | Description |
|---|---|
| [`init`](./lifecycle/init.md) | Bootstrap a new `project.xcaf` and `xcaf/` source directory. |
| [`apply`](./lifecycle/apply.md) | Compile `.xcaf` resources into provider-native files. |
| [`import`](./lifecycle/import.md) | Import existing provider config into `project.xcaf`. |
| [`validate`](./lifecycle/validate.md) | Check `.xcaf` syntax, cross-references, and structural invariants. |

### Inspection & State

| Command | Description |
|---|---|
| [`status`](./diagnostic/status.md) | Show compilation state and drift across providers. |
| [`graph`](./diagnostic/graph.md) | Visualize the resource dependency graph. |
| [`list`](./diagnostic/list.md) | List discovered resources and blueprints. |

### Utilities

| Command | Description |
|---|---|
| [`help`](./utility/help.md) | Display help for any command or resource kind schema. |

## Quick Reference

### Provider Targets

```
antigravity  claude  copilot  cursor  gemini
```

### Exit Codes

| Code | Meaning |
|---|---|
| `0` | Success, or no changes detected. |
| `1` | Error: validation failure, drift detected, policy violation, or unknown target. |

### Common Invocations

```bash
# Bootstrap
xcaffold init
xcaffold init --yes --target claude

# Compile
xcaffold apply
xcaffold apply --target cursor
xcaffold apply --dry-run
xcaffold apply --force --yes
xcaffold apply --blueprint staging

# Import provider config
xcaffold import
xcaffold import --target claude
xcaffold import --dry-run
xcaffold import --agent                    # all agents
xcaffold import --agent=reviewer           # single agent by name
xcaffold import --skill --rule             # multiple kinds

# Validate
xcaffold validate
xcaffold validate --target claude
xcaffold validate --var-file ./custom.vars

# Check drift
xcaffold status
xcaffold status --target claude
xcaffold status --target claude --all

# List resources
xcaffold list
xcaffold list --agent
xcaffold list --skill=my-skill
xcaffold list --verbose

# Dependency graph
xcaffold graph
xcaffold graph --format mermaid
xcaffold graph --format dot | dot -Tsvg > topology.svg
xcaffold graph --format json | jq .
xcaffold graph --agent reviewer
xcaffold graph --full

# Schema introspection
xcaffold --xcaf agent
xcaffold --xcaf skill --out .
xcaffold --xcaf rule --out ./templates/
```

### Selection & Filter Flags

#### Scope Flags

Flags that narrow the scope of a command to a specific target, blueprint, project, or agent.

| Flag | `init` | `apply` | `import` | `validate` | `status` | `list` | `graph` |
|---|---|---|---|---|---|---|---|
| `--target` | Yes | Yes | Yes | Yes | Yes | — | — |
| `--blueprint` | — | Yes | — | Yes | Yes | Yes | Yes |
| `--agent` | — | — | — | — | — | — | Yes |

`init --target` accepts multiple values (`--target claude,cursor`). All other `--target` flags accept a single provider.  
`graph --agent` is optional.

#### Per-Kind Resource Filters

The `import` and `list` commands accept per-kind filter flags. When present without a value, they match all resources of that kind. When given a value, they filter by name.

| Flag | `import` | `list` | Value |
|---|---|---|---|
| `--agent [name]` | Yes | Yes | Optional string (default `*`) |
| `--skill [name]` | Yes | Yes | Optional string (default `*`) |
| `--rule [name]` | Yes | Yes | Optional string (default `*`) |
| `--workflow [name]` | Yes | Yes | Optional string (default `*`) |
| `--mcp [name]` | Yes | Yes | Optional string (default `*`) |
| `--context [name]` | — | Yes | Optional string (default `*`) |
| `--hook` | Yes | Yes | Boolean |
| `--setting` | Yes | Yes | Boolean |
| `--memory` | Yes | — | Boolean |
