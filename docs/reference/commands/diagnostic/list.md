---
title: "xcaffold list"
description: "List all discovered resources in the current scope."
---

# xcaffold list

Scans the target configuration scope and displays all parsed resources grouped by kind.

The `list` command reads the project manifest and all `.xcaf` files under `xcaf/`, then prints a categorized inventory of every discovered resource: agents, skills, rules, workflows, MCP servers, contexts, hooks, settings, memory entries, and blueprints. Resources are sorted alphabetically within each section. Rules are grouped by directory prefix (`cli/`, `platform/`, `(root)`).

**Usage:**

```
xcaffold list [flags]
```

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--agent [name]` | — | `string` | `""` | List agents. Without a value, lists all agents. With a value, filters by substring match. |
| `--skill [name]` | — | `string` | `""` | List skills. Same filter behavior as `--agent`. |
| `--rule [name]` | — | `string` | `""` | List rules. Same filter behavior as `--agent`. |
| `--workflow [name]` | — | `string` | `""` | List workflows. Same filter behavior as `--agent`. |
| `--mcp [name]` | — | `string` | `""` | List MCP servers. Same filter behavior as `--agent`. |
| `--context [name]` | — | `string` | `""` | List contexts. Same filter behavior as `--agent`. |
| `--hook` | — | `bool` | `false` | List hooks. |
| `--setting` | — | `bool` | `false` | List settings. |
| `--verbose` | `-v` | `bool` | `false` | Show individual memory entry names per agent instead of aggregate counts. |
| `--global` | `-g` | `bool` | `false` | Operate on user-wide global config (`~/.xcaffold/global.xcaf`). *(Not yet available — hidden flag)* |
| `--no-color` | — | `bool` | `false` | Disable ANSI color and UTF-8 glyphs. Also honoured via the `NO_COLOR` environment variable. |
| `--blueprint [name]` | — | `string` | `""` | Show resources belonging to a named blueprint. *(Hidden flag)* |
| `--resolved` | — | `bool` | `false` | Include transitive dependencies when listing a blueprint. Use with `--blueprint`. *(Hidden flag)* |

## Behavior

### Default mode

Running `xcaffold list` without kind-filter flags prints all sections: a header breadcrumb, then each resource kind as a titled block with one name per line. Kinds with zero resources are omitted from the header and do not produce a section.

### Kind-filter mode

When one or more kind-filter flags are set (`--agent`, `--skill`, `--rule`, etc.), the output is restricted to only those sections. The header still shows the full project summary.

String-valued kind flags accept an optional name argument for filtering:
- `--agent` — lists all agents
- `--agent dev` — lists agents whose name contains `dev`

Positional arguments are not accepted. To filter by name, use `--<kind>=<name>` syntax.

### Scope

By default, `xcaffold list` operates on the project-level manifest. The `--global` flag is registered but not yet functional for this command. Using it prints `Global scope is not yet available.` and exits with code 1.

## Sample output

### Default — all resources

```
sandbox  ·  12 agents  ·  14 skills  ·  23 rules

AGENTS  (12)
  auth-specialist
  core-services-developer
  data-architect
  database-engineer
  docs-specialist
  go-cli-developer
  macos-developer
  nestjs-api-developer
  platform-frontend-dev
  project-devops
  quality-engineer
  worker-developer

SKILLS  (14)
  adr-management
  commit-changes
  document-feature
  glass-morphic-ui
  provider-ground-truth
  ...

RULES  (23)

  cli/  (4)
    build-go-cli
    open-source-standards
    testing-framework
    worktree-index-safety

  platform/  (13)
    api-conventions
    auth-patterns
    ...

  (root)  (6)
    adr-governance
    git-naming-conventions
    ...

MEMORY  (23 entries across 6 agents)
  auth-specialist (3)
  database-engineer (1)
  ...

BLUEPRINTS
  (none)
```

### Kind filter — agents only

```
sandbox  ·  12 agents  ·  14 skills  ·  23 rules

AGENTS  (12)
  auth-specialist
  core-services-developer
  ...
```

### No-color mode

```
sandbox  .  12 agents  .  14 skills  .  23 rules
```

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success. |
| `1` | Parse error or no project manifest found. |

## Examples

**List all project resources:**
```bash
xcaffold list
```

**List only agents:**
```bash
xcaffold list --agent
```

**List a specific agent:**
```bash
xcaffold list --agent go-cli-developer
```

**Show detailed memory entries:**
```bash
xcaffold list -v
```
