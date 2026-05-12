---
title: "Blueprint Design"
description: "Best practices for designing blueprints that segment agent configurations by team role, environment, or workflow."
---

# Blueprint Design

A blueprint is a named resource subset that selects which agents, skills, rules, workflows, MCP servers, policies, memory entries, contexts, settings, and hooks are compiled. Without a `--blueprint` flag, `xcaffold apply` compiles everything. Blueprints narrow the scope.

## When to Introduce Blueprints

A project with a handful of agents rarely needs blueprints. Introduce them when:

- **Multiple developers work on different subsystems.** A backend engineer does not need frontend agents consuming context tokens, and vice versa.
- **CI pipelines require a stripped-down configuration.** Interactive hooks, MCP servers, and development-only settings should not run in automated environments.
- **Code review and feature development need different tooling.** A reviewer benefits from linting rules and audit agents; a feature developer needs TDD skills and database tools.

## Rely on Transitive Dependencies

When a blueprint selects an agent, the compiler automatically includes that agent's declared skills, rules, and MCP servers. You do not need to enumerate every dependency.

```yaml
# xcaf/agents/api-developer.xcaf
kind: agent
version: "1.0"
name: api-developer
skills: [tdd, schema-design]
rules: [secure-code, api-conventions]
mcp: [database-tools]
```

```yaml
# xcaf/blueprints/backend.xcaf
kind: blueprint
version: "1.0"
name: backend
description: "Backend services — API, database, and testing."
agents:
  - api-developer
  - database-engineer
```

Running `xcaffold apply --blueprint backend` compiles both agents plus their transitive dependencies (`tdd`, `schema-design`, `secure-code`, `api-conventions`, `database-tools`). No need to list them again in the blueprint.

Use `xcaffold list --blueprint backend --resolved` to see the fully resolved resource set.

## Adding Extra Resources

Blueprint resource lists are **additive** — they merge with auto-resolved agent dependencies. If you need a skill that no agent references, add it to the blueprint directly:

```yaml
kind: blueprint
version: "1.0"
name: backend
agents:
  - api-developer
skills:
  - security-audit
```

The resolved skill set is `tdd` + `schema-design` (from `api-developer`) + `security-audit` (from the blueprint). If a skill appears in both the agent's dependencies and the blueprint's explicit list, the compiler returns an error — remove the duplicate from the blueprint since it is already included via the agent.

## Extending Blueprints

Use `extends` to create a child blueprint that inherits from a parent. The child merges its resource lists with the parent's using set-union, and can override specific fields.

### Example: Base blueprint with CI variant

```yaml
# xcaf/blueprints/backend.xcaf
kind: blueprint
version: "1.0"
name: backend
description: "Backend development — API services, database, and testing."
agents:
  - api-developer
  - database-engineer
skills:
  - tdd
  - schema-design
rules:
  - secure-code
  - api-conventions
mcp:
  - database-tools
memory:
  - shared-context
contexts:
  - backend-context
settings: development
hooks: pre-commit
```

```yaml
# xcaf/blueprints/backend-ci.xcaf
kind: blueprint
version: "1.0"
name: backend-ci
description: "CI pipeline — backend agents without interactive hooks or MCP."
extends: backend
mcp: []
hooks: ""
settings: restricted
```

`backend-ci` inherits all agents, skills, rules, and contexts from `backend`, but overrides three things:

- `mcp: []` — no MCP servers in CI (database-tools requires a running connection)
- `hooks: ""` — no interactive pre-commit hooks in automation
- `settings: restricted` — uses a read-only settings profile instead of `development`

### Extends resolution

- **Set union:** child lists are merged with parent lists, duplicates removed.
- **Override:** setting a field to an empty value (`[]` for lists, `""` for strings) in the child overrides the parent's value entirely.
- **Chain depth:** extends chains can go up to 10 levels deep. Cycles are detected and rejected at parse time.
- **Non-existent parent:** referencing a parent blueprint that does not exist produces a parse error.

## Context Selection

Blueprints control which `kind: context` files render to provider root files (CLAUDE.md, GEMINI.md). At compile time, at most **one context per target provider** is allowed.

```yaml
# xcaf/context/main.xcaf — shared project context
---
kind: context
name: main
default: true
---
This project uses Go 1.24 with Cobra for CLI commands.
Run tests with `go test ./...`.

# xcaf/context/backend-context.xcaf — backend-specific
---
kind: context
name: backend-context
targets: [claude]
---
Focus on API design and database schema integrity.
Use the database-tools MCP server for schema exploration.

# xcaf/context/frontend-context.xcaf — frontend-specific
---
kind: context
name: frontend-context
targets: [claude]
---
Focus on React component composition and accessibility.
Use the browser-tools MCP server for visual testing.
```

### How contexts resolve

- **One context per target:** if multiple context files match the same target without a blueprint, the one marked `default: true` renders first, followed by the rest in alphabetical order. If none is marked as default, the compiler returns an error. See the [context reference — Default Resolution](../reference/kinds/provider/context.md#default-resolution) for the full resolution table.
- **Blueprint selection overrides:** when `--blueprint` is used, only contexts listed in the blueprint's `contexts:` field compile. The `default` flag is ignored entirely.
- **Omitted `contexts:` in blueprint = no contexts:** unlike `policies:` (where omission means "evaluate all"), omitting `contexts:` from a blueprint compiles zero context resources. This is intentional — context prose is specific to the workflow the blueprint represents, so there is no safe default.
- **No targets = all targets:** a context with no `targets:` field renders for every configured provider.

### Blueprint + context example

```yaml
# Backend blueprint selects backend-context for Claude
kind: blueprint
version: "1.0"
name: backend
contexts:
  - backend-context
agents:
  - api-developer
  - database-engineer

# Frontend blueprint selects frontend-context
kind: blueprint
version: "1.0"
name: frontend
contexts:
  - frontend-context
agents:
  - react-developer
```

Running `xcaffold apply --blueprint backend` renders only `backend-context` to CLAUDE.md. Running without `--blueprint` renders `main` (the default context).

## Policies and Blueprints

Blueprints can select which `kind: policy` resources are evaluated. Policies enforce quality constraints on agents, skills, and rules — for example, requiring every agent to have a description, or denying specific tool patterns in compiled output.

When a blueprint selects `policies: [security-baseline]`, only that policy evaluates against the blueprint's compiled resources. When `policies:` is omitted, all policies evaluate. Setting `policies: []` disables all policy evaluation for that blueprint.

## Named Settings and Hooks

Blueprints can select which `kind: settings` and `kind: hooks` configuration to use:

```yaml
# xcaf/settings/development.xcaf
kind: settings
version: "1.0"
name: development
model: sonnet
permissions:
  allow: [Read, Write, Edit, Bash]

# xcaf/settings/restricted.xcaf
kind: settings
version: "1.0"
name: restricted
model: haiku
permissions:
  allow: [Read, Grep, Glob]
  deny: [Bash, Write]
```

When a blueprint omits `settings` or `hooks`, all named configurations are included. When specified, only the named entry is compiled.

## Checking Blueprint State

State is tracked independently per blueprint. Use `xcaffold status` to inspect drift:

```bash
xcaffold status --blueprint backend
```

```
my-project  ·  last applied 2 minutes ago

  claude        90 artifacts    1 modified
  cursor        72 artifacts    synced

Sources  54 .xcaf files  ·  no changes since last apply

Modified files:

  claude
    not on disk       agents/developer.md

Run 'xcaffold apply' to restore.
```

Drift in one blueprint does not affect another. Before switching between blueprints, run status to verify your compiled output is in sync with your sources.
