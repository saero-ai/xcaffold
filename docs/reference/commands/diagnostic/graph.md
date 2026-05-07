---
title: "xcaffold graph"
description: "Render a dependency graph of agents and their linked resources."
---

# xcaffold graph

Parses `.xcaf` manifests and renders a visual dependency graph.

The `graph` command builds a directed acyclic graph (DAG) of the current configuration scope, showing how agents relate to skills, rules, workflows, MCP servers, and memory. Output can be rendered as a terminal tree, Mermaid diagram, Graphviz DOT file, or JSON edge list.

**Usage:**

```
xcaffold graph [flags]
```

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--agent [name]` | — | `string` | `""` | Filter graph to agents matching name (or all agents if no value). |
| `--skill [name]` | — | `string` | `""` | Filter to skills. |
| `--rule [name]` | — | `string` | `""` | Filter to rules. |
| `--workflow [name]` | — | `string` | `""` | Filter to workflows. |
| `--mcp [name]` | — | `string` | `""` | Filter to MCP servers. |
| `--context [name]` | — | `string` | `""` | Filter to contexts. |
| `--hooks` | — | `bool` | `false` | Include hooks in output. |
| `--settings` | — | `bool` | `false` | Include settings in output. |
| `--format` | `-f` | `string` | `"terminal"` | Output format: `terminal`, `mermaid`, `dot`, `json`. |
| `--full` | — | `bool` | `false` | Expand all nested relations in the terminal tree. Always true when `--agent` names a specific agent. |
| `--global` | `-g` | `bool` | `false` | Operate on user-wide global config (`~/.xcaffold/global.xcaf`). |
| `--no-color` | — | `bool` | `false` | Disable ANSI color and UTF-8 glyphs. Also honoured via the `NO_COLOR` environment variable. |

## Behavior

### Terminal tree

The default `terminal` format renders each agent as a tree node with its tools, linked skills, rules, and memory entries nested beneath it. Branch glyphs use `│`, `├──`, and `└──` aligned at column 2:

```
  ● agent-name
  │   tools    Read  Edit  Write  Bash  Glob  Grep
  │
  ├── skills
  │     ├── skill-a
  │     └── skill-b
  │
  └── memory  (2 entries)
        ├── entry-a
        └── entry-b
```

The header breadcrumb uses `·` as a separator (falls back to `.` when `--no-color` is set or `NO_COLOR` is set). Kinds with zero resources are omitted from the header count.

### Kind-filter mode

When one or more kind-filter flags are provided, only nodes of those kinds appear in the graph. Without kind filters, all resource types are included.

### Output formats

- **`terminal`** — Default. Stylized tree view suitable for interactive inspection.
- **`mermaid`** — Mermaid.js compatible markdown block. Pipe to a file or include in docs.
- **`dot`** — Graphviz DOT format. Combine with `dot -Tsvg` to produce network diagrams.
- **`json`** — Machine-readable array of nodes and edges. Suitable for CI/CD integration.

### Scope

By default, `xcaffold graph` operates on the project-level manifest. Using `--global` switches to the user-wide global scope.

## Sample output

### Terminal — project graph

```
sandbox  ·  12 agents  ·  14 skills  ·  23 rules

  ● auth-specialist
  │   tools    Read  Bash  Glob  Grep
  │
  ├── skills
  │     ├── feature-lifecycle
  │     └── commit-changes
  │
  └── memory  (3 entries)
        ├── corrections.md
        ├── feedback-notes.md
        └── user-profile.md

  ● database-engineer
  │   tools    Read  Write  Edit  Bash  Glob  Grep
  │
  └── skills
        └── tdd
```

### No-color mode

```
sandbox  .  12 agents  .  14 skills  .  23 rules
```

### Mermaid output

```bash
xcaffold graph --format mermaid
```

```
graph LR
  auth-specialist --> feature-lifecycle
  auth-specialist --> commit-changes
  database-engineer --> tdd
```

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success. |
| `1` | Parse error or no project manifest found. |

## Examples

**Display the full project dependency graph:**
```bash
xcaffold graph
```

**Inspect a single agent's dependency tree:**
```bash
xcaffold graph --agent auth-specialist
```

**Export a Mermaid diagram:**
```bash
xcaffold graph --format mermaid > architecture.md
```

**Export a Graphviz DOT file:**
```bash
xcaffold graph --format dot | dot -Tsvg > graph.svg
```

**Show only agents and their linked skills:**
```bash
xcaffold graph --agent --skill
```

**Inspect the global scope graph:**
```bash
xcaffold graph --global
```
