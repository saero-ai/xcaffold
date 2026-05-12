---
title: "Blueprint Design"
description: "Best practices for designing blueprints that segment agent configurations by team role, environment, or workflow."
---

# Blueprint Design

A blueprint is a named resource subset that selects which agents, skills, rules, workflows, MCP servers, policies, memory entries, contexts, settings, and hooks are compiled. Without a `--blueprint` flag, `xcaffold apply` compiles everything. Blueprints narrow the scope. In Harness-as-Code terms, a blueprint is a scoped projection of the full harness — only the resources relevant to a specific role, environment, or workflow are compiled.

## When to Introduce Blueprints

A project with a handful of agents rarely needs blueprints. Introduce them when:

- **Multiple developers work on different subsystems.** A backend engineer does not need frontend agents consuming context tokens, and vice versa.
- **CI pipelines require a stripped-down configuration.** Interactive hooks, MCP servers, and development-only settings should not run in automated environments.
- **Code review and feature development need different tooling.** A reviewer benefits from linting rules and audit agents; a feature developer needs TDD skills and database tools.

## Rely on Transitive Dependencies

When a blueprint selects an agent, the compiler automatically includes that agent's declared skills, rules, and MCP servers. You do not need to enumerate every dependency.

```yaml
# xcaf/agents/api-developer.xcaf
---
kind: agent
version: "1.0"
name: api-developer
skills: [tdd, schema-design]
rules: [secure-code, api-conventions]
mcp: [database-tools]
---
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
description: "CI pipeline — backend agents."
extends: backend
settings: default
```

`backend-ci` inherits all agents, skills, rules, and contexts from `backend`. 

### Extends resolution

- **Set union:** child resource lists (agents, skills, rules, etc.) are merged with parent lists. Elements from the parent appear first, followed by elements from the child not already present in the parent.
- **Merge behavior:** empty lists in a child blueprint do not remove parent resources; they result in a union that preserves all parent selections.
- **Chain depth:** extends chains can go up to 5 levels deep. Cycles are detected and rejected at parse time.
- **Non-existent parent:** referencing a parent blueprint that does not exist produces a parse error.

## Context Selection

Blueprints control which `kind: context` files render to provider root files (CLAUDE.md, GEMINI.md). Multiple contexts matching the same target are **composed** (joined) into a single instruction set.

```yaml
# xcaf/context/main.xcaf — shared project context
---
kind: context
version: "1.0"
name: main
default: true
---
This project uses Go 1.24 with Cobra for CLI commands.
Run tests with `go test ./...`.

# xcaf/context/backend-context.xcaf — backend-specific
---
kind: context
version: "1.0"
name: backend-context
targets: [claude]
---
Focus on API design and database schema integrity.
Use the database-tools MCP server for schema exploration.

# xcaf/context/frontend-context.xcaf — frontend-specific
---
kind: context
version: "1.0"
name: frontend-context
targets: [claude]
---
Focus on React component composition and accessibility.
Use the browser-tools MCP server for visual testing.
```

### How contexts resolve

- **Context composition:** if multiple context files match the same target, all bodies are composed (joined with `\n\n`). The one marked `default: true` is placed first; remaining contexts follow in sorted name order. If none is marked as default and multiple match, the compiler returns an error. See the [context reference — Default Resolution](../reference/kinds/provider/context.md#default-resolution) for the full resolution table.
- **Blueprint selection overrides:** when `--blueprint` is used, only contexts listed in the blueprint's `contexts:` field compile. The `default` flag is ignored entirely.
- **Omitted `contexts:` in blueprint = no contexts:** unlike `policies:` (where omission means "evaluate all"), omitting `contexts:` from a blueprint compiles zero context resources. This is intentional — context prose is specific to the workflow the blueprint represents, so there is no safe default.
- **No targets = all targets:** a context with no `targets:` field is eligible for every configured provider.

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

Blueprints can select which `kind: policy` resources are evaluated. Policies enforce quality constraints on agents, skills, and rules.

When a blueprint selects `policies: [security-baseline]`, only that policy evaluates against the blueprint's compiled resources. 

**Note on Omission:** Unlike other resource types, omitting the `policies:` field in a blueprint (or setting it to `[]`) disables all policy evaluation for that blueprint.

## Named Settings and Hooks

Blueprints can select which `kind: settings` and `kind: hooks` configuration to include in the compilation.

**Note on Selection:** The `xcaffold` compiler currently expects the selected settings or hooks entry to be named `default` for it to be automatically applied to the provider's configuration. If a blueprint selects a non-default name (e.g., `settings: restricted`), that configuration will be included in the resource set but may not be automatically active unless the target provider's renderer is specifically configured to handle it.

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

## Focused Context for Delegated Work

Blueprints scope both agents and workspace context simultaneously. When you apply with `--blueprint`, only the contexts listed in the blueprint's `contexts:` field are compiled to the provider's root instruction file (e.g., `CLAUDE.md`). Agents compiled under that blueprint receive only the relevant context — not every context defined in the project.

This is particularly valuable when orchestrating subagent work. A main agent dispatching a backend task can compile with the `backend` blueprint, ensuring the subagent's workspace context covers database conventions and API patterns — without injecting frontend CSS rules, design system tokens, or unrelated team standards.

```yaml
# xcaf/blueprints/backend.xcaf
kind: blueprint
version: "1.0"
name: backend
description: "Backend services — API and database."
agents:
  - api-developer
  - database-engineer
contexts:
  - backend-context
rules:
  - api-conventions
  - migration-safety
```

```yaml
# xcaf/blueprints/frontend.xcaf
kind: blueprint
version: "1.0"
name: frontend
description: "Frontend — React components and styling."
agents:
  - react-developer
contexts:
  - frontend-context
rules:
  - accessibility
  - css-conventions
```

Running `xcaffold apply --blueprint backend` compiles only `backend-context` to `CLAUDE.md`, only the two backend agents, and only the backend rules. The frontend CSS conventions, accessibility rules, and `frontend-context` are excluded entirely. The subagent receives a clean, focused workspace instruction set with no stale or irrelevant guidance.

Without blueprints, every agent in the project receives the full workspace context — which grows linearly with the number of teams and domains. Blueprints are the mechanism for keeping that context focused.

## Composing from the Catalog

xcaffold's resource model is designed for mix-and-match composition. Every resource kind (`agent`, `skill`, `rule`, `workflow`, `mcp`, `policy`, `context`) is an independent unit with a declared identity. Blueprints let you assemble these units into purpose-specific bundles.

When your project grows a library of reusable resources — skills for TDD, rules for security, policies for quality — blueprints become the interface for composing them into role-specific or environment-specific configurations. Rather than maintaining separate agent definitions for each team, maintain one shared pool of resources and let blueprints select the right combination.

The pattern scales: a CI blueprint selects a restrictive settings profile and no interactive hooks. A review blueprint selects read-only agents and audit rules. A full-development blueprint selects everything. Each is a single file that references existing resources by ID — no duplication.

## Decision Guide

| Situation | Approach |
|---|---|
| Multiple developers need different agent sets | Create role-based blueprints (`backend`, `frontend`, `reviewer`) |
| CI needs a minimal configuration | Create a `ci` blueprint with only automation-relevant agents |
| A blueprint needs everything another has, plus more | Use `extends:` to inherit and add resources |
| An agent's transitive dependencies are not enough | Add extra resources directly to the blueprint's resource lists |
| You need to select specific context documents per blueprint | Use the `contexts:` list to include only relevant workspace instructions |
| All agents should share a policy but only in production | Add the policy to a `production` blueprint, not to every agent |

## Related

- [Blueprint Reference](../reference/kinds/xcaffold/blueprint.md) — field-level documentation for blueprint resources
- [Project Structure](project-structure.md) — when to introduce blueprints as a project grows
- [Agent Design Patterns](agent-design-patterns.md) — how agents declare dependencies that blueprints resolve
