---
title: "Agent Design Patterns"
description: "Practical patterns for designing effective xcaffold agents — from tool scoping to composition to provider-targeted overrides."
---

# Agent Design Patterns

Agents are the primary resource kind in xcaffold, defining the personas, tool access, and behavioral constraints for AI interactions — compiled to provider-native format on every `xcaffold apply`. This guide covers practical patterns for writing agents that are focused, maintainable, and correctly scoped across providers. For field-level reference, see the [agent reference](../reference/kinds/provider/agent.md#argument-reference).

## Anatomy of an Agent

Every agent lives in its own subdirectory under `xcaf/agents/<name>/agent.xcaf`. See the [agent reference](../reference/kinds/provider/agent.md#filesystem-as-schema) for directory layout rules and filesystem-as-schema inference.

A well-formed agent covers the fields that matter for its role and nothing else:

```yaml
---
kind: agent
version: "1.0"
name: developer
description: "Implements and tests Go code. Runs build and test commands. Does not create documentation."
model: sonnet
tools: [Bash, Read, Write, Edit, Glob, Grep]
skills: [tdd, git-workflow]
rules: [secure-coding]
max-turns: 30
---
You are a Go software developer. Your task is to implement the feature described in the conversation, write tests for it, and confirm the tests pass before stopping.

When implementing, read existing code first. Do not guess at types, interfaces, or package paths — verify them.

Stop when tests pass and the implementation matches the stated requirements. Do not refactor adjacent code unless explicitly asked.
```

Agent fields are grouped by purpose: identity, model and execution, tool access, composition, permissions, lifecycle, memory, inline definitions, and multi-target overrides. See the [agent reference](../reference/kinds/provider/agent.md#argument-reference) for the complete field schema.

**The body IS the system prompt.** The markdown prose below the frontmatter delimiters is passed verbatim as the agent's system prompt instructions in every provider that supports agents. It is the universal instruction layer — what you write here shapes behavior across all targets.

## Specialized Agents

A common mistake is writing one large general-purpose agent and trying to cover every task through prompt engineering. Narrow agents are more reliable: a reviewer that can only read cannot accidentally modify files, and a database agent that only knows SQL tools cannot touch application code.

### Code reviewer

```yaml
---
kind: agent
version: "1.0"
name: code-reviewer
description: "Reviews pull request diffs for correctness, test coverage, and style. Read-only."
model: sonnet
tools: [Read, Glob, Grep]
rules: [review-standards]
readonly: true
---
You are a code reviewer. You read the diff provided in the conversation and produce a structured review.

Your review covers: correctness (logic errors, edge cases), test coverage (is new behavior tested?), and style (naming, documentation). You do not suggest refactoring unless it directly relates to a defect.

Output your review as a markdown list with severity labels: ERROR, WARNING, SUGGESTION.
```

### API developer

```yaml
---
kind: agent
version: "1.0"
name: api-developer
description: "Implements REST API handlers. Writes handler functions, request/response types, and unit tests."
model: sonnet
tools: [Bash, Read, Write, Edit, Glob, Grep]
skills: [tdd, rest-conventions]
rules: [secure-coding, no-direct-db-access]
max-turns: 40
---
You are an API developer. Implement the endpoint described in the conversation. Always write a handler, a request type, a response type, and at least one test.

Do not modify database schema files or ORM definitions. If you need a new database query, write a placeholder and note it in your response.
```

### Database administrator

```yaml
---
kind: agent
version: "1.0"
name: database-admin
description: "Writes SQL migrations and queries. Does not modify application code."
model: sonnet
tools: [Read, Write, Edit, Glob, Grep]
skills: [sql-style]
rules: [migration-safety]
---
You are a database administrator. Write SQL migration files and queries as directed.

Always write reversible migrations (up + down). Never use DROP TABLE or TRUNCATE in up migrations. Test queries with EXPLAIN ANALYZE before finalizing.
```

Narrow agents are easier to reason about, easier to test, and produce fewer out-of-scope side effects. When an agent's description says exactly what it does and does not do, providers and users make better delegation decisions.

## Composition

Skills and rules are compiled into every agent that references them. Rather than copying the same instructions into every agent body, attach shared resources by ID.

### Two agents sharing one skill

```yaml
---
kind: agent
version: "1.0"
name: frontend-developer
description: "Implements React components and pages."
model: sonnet
tools: [Bash, Read, Write, Edit, Glob, Grep]
skills: [tdd, git-workflow]
---
You implement React components using TypeScript and Tailwind CSS.
```

```yaml
---
kind: agent
version: "1.0"
name: backend-developer
description: "Implements Go service handlers."
model: sonnet
tools: [Bash, Read, Write, Edit, Glob, Grep]
skills: [tdd, git-workflow]
---
You implement Go handlers, middleware, and service logic.
```

Both agents reference `tdd` and `git-workflow`. The compiler includes each skill's compiled output in both agents' provider-native files. The skill body is maintained once; both agents benefit automatically when the skill is updated.

### Rules govern, skills instruct

Use `rules:` for constraints (what the agent must or must not do), and `skills:` for procedural workflows (step-by-step processes the agent follows):

```yaml
skills: [tdd, git-workflow]       # How to do the work
rules: [secure-coding, no-secrets]  # Constraints on the work
```

This separation makes it easier to audit what governs an agent versus what it is instructed to do.

## Tool Scoping

xcaffold provides three tool-control mechanisms. Each serves a different pattern:

**`tools:`** — explicit allow-list. Name exactly which tools the agent may invoke. Shorter lists reduce surface area. This is the primary control for most agents.

**`disallowed-tools:`** — deny-list applied on top of `tools:`. Useful when inheriting a broad tool set and carving out one specific tool. Claude-only; other providers emit a warning.

**`readonly: true`** — semantic declaration of read-only intent. The provider enforces this regardless of what `tools:` lists. Supported by Claude and Cursor.

See the [agent reference](../reference/kinds/provider/agent.md#argument-reference) for field semantics, types, and provider support details.

### Decision guide — tool controls

| Situation | Use |
|---|---|
| You know exactly which tools the agent needs | `tools:` with a precise list |
| You need to block one tool from an otherwise broad grant | `disallowed-tools:` |
| You want to declare read-only intent clearly | `readonly: true` |
| The agent runs in production with restricted trust | `permission-mode:` (Claude only) |

## Provider-Scoped Overrides

Different providers have different capabilities. The `targets:` map lets you override specific fields per provider without duplicating the entire agent definition.

### Override files for per-provider differences

To change a resource field (like `model`, `tools`, or the body) for a specific provider, place an override file alongside the base resource:

```
xcaf/agents/
└── code-reviewer/
    ├── agent.xcaf            ← base definition (all providers)
    ├── agent.cursor.xcaf     ← Cursor override
    └── agent.gemini.xcaf     ← Gemini override
```

**Base** (`agent.xcaf`):
```yaml
---
kind: agent
version: "1.0"
name: code-reviewer
description: "Reviews code for correctness and coverage."
model: sonnet
tools: [Read, Glob, Grep]
readonly: true
---
You are a code reviewer. Read the code described in the conversation and produce a structured review.
```

**Cursor override** (`agent.cursor.xcaf`):
```yaml
---
kind: agent
version: "1.0"
model: gpt-4o
---
```

**Gemini override** (`agent.gemini.xcaf`):
```yaml
---
kind: agent
version: "1.0"
model: gemini-2.5-pro
---
```

The base `model: sonnet` applies to Claude and other providers without an override. Cursor compiles with `gpt-4o`, Gemini with `gemini-2.5-pro`. All other fields (`tools:`, `readonly:`, the body) come from the base for every target.

Override merge rules are covered in [Variables and Overrides](variables-and-overrides.md).

### Scoping an agent to specific providers

The `targets:` map controls which providers compile an agent. When `targets:` is present, only the listed providers receive it:

```yaml
---
kind: agent
version: "1.0"
name: mcp-explorer
description: "Explores MCP server schemas and data."
model: sonnet
tools: [Read, Glob, Grep]
targets:
  claude: {}
  copilot: {}
---
Use MCP tools to explore server schemas. Report findings as structured summaries.
```

This agent compiles only for Claude and Copilot. Gemini, Cursor, and Antigravity skip it with an info note.

The `targets:` map also supports provider-specific metadata — `suppress-fidelity-warnings: true` to silence expected field drops, and `provider:` for opaque pass-through keys.

### Provider support

Not all providers support every agent field. Fields without native support are dropped silently during compilation. See the [agent reference](../reference/kinds/provider/agent.md#compiled-output) for the full provider support matrix and compiled output examples per target.

## The System Prompt (Body)

The markdown body of an agent `.xcaf` file is its system prompt. Write it with the same discipline you apply to the rest of the manifest.

### Keep it focused

The body should describe what the agent is and how it operates — not duplicate what the tools, skills, and rules already express. If you have a `tdd` skill attached, do not re-explain TDD in the body. If you have a `secure-coding` rule, do not re-enumerate the security constraints. Let the compiled resources do that work.

A focused body for a 40-field manifest may be 5–8 lines. That is correct. Longer is not better.

### Use imperative voice

```
✗ You should try to read the code before making changes.
✅ Read the existing implementation before writing any code.
```

```
✗ It would be helpful if you could include tests.
✅ Write tests before implementing. Do not mark a task complete without a passing test.
```

Imperative voice removes ambiguity. LLMs interpret "you should try" as optional.

### State what the agent does NOT do

The most useful sentence in many agent bodies is a negative constraint:

```
Do not modify schema files. If a schema change is required, stop and describe what is needed.
```

Explicit negative constraints prevent the most common out-of-scope behaviors. They are more reliable than trusting `tools:` alone to prevent unwanted actions.

### Length guidance

| Body length | Appropriate when |
|---|---|
| 3–8 lines | Role is narrow and well-defined; skills and rules cover the rest |
| 8–20 lines | Role needs nuanced behavioral guidance not expressible in skills/rules |
| 20+ lines | Break into a skill with its own references/ subdirectory |

If your agent body is growing past 20 lines, extract the repeating content into a `kind: skill` and attach it via `skills:`. The skill can reference detailed documents in its `references/` directory without bloating the agent manifest.

## Decision Guide

| Need | Pattern | Field |
|---|---|---|
| Limit what the agent can do | Precise allow-list | `tools:` |
| Block one tool from a broad grant | Deny specific tool | `disallowed-tools:` (Claude only) |
| Declare read-only intent explicitly | Meta-constraint | `readonly:` |
| Reuse instructions across agents | Attach a skill | `skills:` |
| Apply governance constraints | Attach a rule | `rules:` |
| Use different model per provider | Per-target override | `targets:` |
| Agent has large system prompt | Extract to skill | `kind: skill` + `skills:` |
| Agent needs MCP server access | Reference or inline | `mcp:` or `mcp-servers:` |
| Prevent spawning sub-agents | Block sub-invocation | `disable-model-invocation:` (Claude only) |
| Agent runs non-interactively | Headless mode | `background:` (Claude, Cursor) |
| Expose agent as slash command | Slash invocation | `user-invocable:` (Claude only) |
| Enforce compile-time constraints on agents | Governance | `kind: policy` targeting `agent` |

## Related

- [Agent Reference](../reference/kinds/provider/agent.md) — complete field schema, provider support matrix, compiled output per target
- [Variables and Overrides](variables-and-overrides.md) — override file conventions, merge rules, variable resolution
- [Supported Providers](../reference/supported-providers.md) — cross-provider feature comparison
- [Rule Organization](rule-organization.md) — patterns for the `rules:` resources agents reference
