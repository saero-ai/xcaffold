---
title: "xcaffold graph"
description: "Render a dependency graph of agents and their linked resources."
---

# xcaffold graph

Parses `.xcaf` manifests and renders a visual dependency graph.

The `graph` command builds a directed acyclic graph (DAG) of the current configuration scope, showing how agents relate to skills, rules, workflows, MCP servers, and policies. Output can be rendered as a terminal tree, Mermaid diagram, Graphviz DOT file, or JSON edge list.

**Usage:**

```
xcaffold graph [file] [flags]
```

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--agent <name>` | `-a` | `string` | `""` | Target a specific agent; shows only its topology. |
| `--project <name>` | `-p` | `string` | `""` | Target a specific managed project by registered name or path. |
| `--full` | `-f` | `bool` | `false` | Show the fully expanded topology tree. Always true when `--agent` targets a specific agent. |
| `--all` | — | `bool` | `false` | Show global topology and all registered projects. Mutually exclusive with `--project` and `--global`. |
| `--scan-output` | — | `bool` | `false` | Scan compiled output directories for undeclared artifacts. |
| `--format` | — | `string` | `"terminal"` | Output format: `terminal`, `mermaid`, `dot`, `json`. |
| `--global` | `-g` | `bool` | `false` | Operate on user-wide global config (`~/.xcaffold/global.xcaf`). Mutually exclusive with `--all`. |
| `--no-color` | — | `bool` | `false` | Disable ANSI color and UTF-8 glyphs. Also honoured via the `NO_COLOR` environment variable. |

## Behavior

### Terminal tree

The default `terminal` format renders each agent as a tree node with its tools, linked skills, rules, and MCP servers nested beneath it. Branch glyphs use `│`, `├──`, and `└──` aligned at column 2:

```
  ● agent-name [model-alias]
      │
      ├─▶ [Capabilities]
      │    ├─(tool)─▶ Read
      │    └─(tool)─▶ Bash
      │
      ├─▶ [Skills]
      │    ├─▶ skill-a
      │    └─▶ skill-b
      │
      └─▶ [Rules]
           └─▶ rule-a
```

The header breadcrumb uses `·` as a separator (falls back to `.` when `--no-color` is set or `NO_COLOR` is set). Kinds with zero resources are omitted from the header count.

### Output formats

- **`terminal`** — Default. Stylized tree view suitable for interactive inspection.
- **`mermaid`** — Mermaid.js compatible markdown block. Pipe to a file or include in docs.
- **`dot`** — Graphviz DOT format. Combine with `dot -Tsvg` to produce network diagrams.
- **`json`** — Machine-readable array of nodes and edges. Suitable for CI/CD integration.

### Scope

By default, `xcaffold graph` operates on the project-level manifest in the current directory. Pass a file path as a positional argument to target a specific `.xcaf` file directly. Use `--global` to switch to the user-wide global scope, `--project` to target a registered project by name, or `--all` to render global topology plus all registered projects.

### Scanning undeclared artifacts

When `--scan-output` is provided, the command scans the compiled output directory for files that are present on disk but not declared in the current manifest. Undeclared entries are listed under a `[ UNDECLARED FILES ]` section in terminal mode or included in the `disk_entries` array in JSON mode.

## Sample output

### Terminal — project graph

```
┌─────────────────────────────────────────────────────────────────┐
│   sandbox  ·  2 agents  ·  3 skills  ·  4 rules                │
└─────────────────────────────────────────────────────────────────┘

  [ AGENTS ]
  ● auth-specialist [...-sonnet]
      │
      ├─▶ [Capabilities]
      │    ├─(tool)─▶ Read
      │    └─(tool)─▶ Bash
      │
      ├─▶ [Skills]
      │    ├─▶ feature-lifecycle
      │    └─▶ commit-changes
      │
      └─▶ [Rules]
           └─▶ secure-coding
```

### Mermaid output

```bash
xcaffold graph --format mermaid
```

```
graph LR
  subgraph Agents
    agent_auth_specialist["auth-specialist / claude-sonnet-4-6"]
  end
  subgraph Skills
    skill_feature_lifecycle["feature-lifecycle"]
    skill_commit_changes["commit-changes"]
  end
  agent_auth_specialist -->|"skill"| skill_feature_lifecycle
  agent_auth_specialist -->|"skill"| skill_commit_changes
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

**Target a registered project by name:**
```bash
xcaffold graph --project my-api
```

**Show global topology and all registered projects:**
```bash
xcaffold graph --all
```

**Export a Mermaid diagram:**
```bash
xcaffold graph --format mermaid > architecture.md
```

**Export a Graphviz DOT file:**
```bash
xcaffold graph --format dot | dot -Tsvg > graph.svg
```

**Inspect the global scope graph:**
```bash
xcaffold graph --global
```

**Scan for undeclared output artifacts:**
```bash
xcaffold graph --scan-output
```

**Show expanded topology for a project:**
```bash
xcaffold graph --full
```

**Export machine-readable JSON for CI:**
```bash
xcaffold graph --format json | jq .
```
