---
title: "xcaffold list"
description: "List all discovered resources, blueprints, and active agents."
---

# xcaffold list

Scans the target configuration scope (Global or Project) and displays all parsed compilation resources.

The `list` command allows you to verify that the AST parser is correctly indexing all of your `.xcf` definitions before you compile them into a target provider format. It groups rules by folder, presents agent memory allocations, and provides a clean, responsive three-column grid layout depending on your terminal width.

## Usage

```bash
xcaffold list [flags]
```

## Options

| Flag | Default | Description |
|---|---|---|
| `-v, --verbose` | `false` | Enable verbose output to show explicit memory node names allocated per agent, instead of aggregate counts. |
| `-g, --global` | `false` | Operate on the user-wide global configuration registry (`~/.xcaffold/global.xcf`). |
| `--config <string>` | `""` | Path to a specific `project.xcf` file for parsing resources from a standalone monorepo sub-directory. |

## Behavior

### Grouping and Sorting
Resources (such as `rules`) are intelligently grouped by their declared source path or taxonomy (e.g., `cli/`, `platform/`, `(root)`) making it easy to see where specific governance rules originate. Items within categories are sorted alphabetically.

### Memory Inclusion
For defined `agents`, the command tallies the number of associated memory entries directly next to the agent ID. When `--verbose` is provided, the output expands to explicitly list the filenames making up the contextual memory boundary for that agent.

### Scope Inspection
By default, `xcaffold list` analyzes the project-level manifest (`./project.xcf`). Using the `--global` flag bypasses all local configurations, emitting a pure list of cross-project primitives available system-wide.

## Examples

**Display a concise list of all project-level resources:**
```bash
xcaffold list
```

**List all resources with expanded agent memory references:**
```bash
xcaffold list -v
```

**List all elements defined in the global configuration store:**
```bash
xcaffold list --global 
```
