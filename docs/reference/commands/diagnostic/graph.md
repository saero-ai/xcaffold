---
title: "xcaffold graph"
description: "Visualize dependency topologies and execution chains across your agents and resources."
---

# xcaffold graph

Parses `.xcf` manifests and renders a visual dependency topology.

The `graph` command builds a directed acyclic graph (DAG) of the current configuration scope, analyzing how agents link to core skills, global rules, active workflows, and contextual memory fragments.

It natively outputs to standard visual formats (`mermaid`, `dot`), `json` for programmable inspection, and a stylized expanded terminal tree view.

## Usage

```bash
xcaffold graph [file] [flags]
```

## Options

| Flag | Default | Description |
|---|---|---|
| `-a, --agent <string>` | `""` | Isolate the graph topology to a single agent (and its downstream dependencies). |
| `--all` | `false` | Show the combined global topology along with all registered projects. |
| `-f, --format <string>` | `"terminal"` | Output format. Available options: `terminal`, `mermaid`, `dot`, `json`. |
| `--full` | `false` | Expand all nested relations completely in the terminal tree. Always true if using `--agent`. |
| `-p, --project <string>` | `""` | Target a specific managed project stored in the global registry (by name or path). |
| `--scan-output` | `false` | Scan the active `.xcaffold/` output directory for undeclared, provider-native artifacts that are bypassing AST definitions. |

## Behavior

### Scoped Topologies

By default, `xcaffold graph` displays the active project scope tree. 
It analyzes the local configuration and renders all agents, mapping their respective `tools` list to defined skills, inherited memory units, required policies, and execution hooks.

To see the global scope (resources available to all projects within the user environment), append the `-g, --global` flag.

### Output Formats

- **Terminal Tree (`terminal`)**: Default. Produces an ASCII, expanded tree view mapping the core structural entity relationships. 
- **Mermaid (`mermaid`)**: Outputs Mermaid.js compatible markdown blocks natively. Excellent for piping directly into architecture or workflow documentation.
- **Graphviz (`dot`)**: Outputs standard Graphviz DOT files. Best combined with `dot -Tsvg` to yield complex network visualizations.
- **JSON (`json`)**: Outputs a machine-readable array of nodes and edges connecting resources, which can be ingested by CI/CD compliance or internal QA pipelines.

## Examples

**Display the topology of the current working project in terminal:**
```bash
xcaffold graph
```

**Generate an architectural mermaid graphic into an artifact:**
```bash
xcaffold graph --format mermaid > architecture.md
```

**Trace the behavior of a single agent (e.g., frontend engineer):**
```bash
xcaffold graph --agent "frontend-engineer" --full
```
