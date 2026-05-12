---
title: "Policy Organization"
description: "Use-case patterns for governing your xcaffold project with kind: policy compile-time rules."
---

# Policy Organization

Policies give you compile-time governance over your xcaffold project. Rather than relying on code review or runtime observation to catch misconfigurations, you express constraints as `kind: policy` files that run automatically on every `xcaffold apply` and `xcaffold validate`. A violation at `error` severity blocks compilation entirely.

This guide covers practical use cases — how to use policies effectively, not what every field does. For field-level reference, see the [Schema Reference](../reference/schema.md#policyconfig).

---

## Use Case 1 — Enforcing Quality Standards Across All Agents

The most common use of policies is ensuring every agent has the metadata needed for it to be useful to other developers and AI tools:

```
---
kind: policy
version: "1.0"
name: require-agent-description
description: All agents must have a description so delegation and auto-invocation work correctly.
severity: warning
target: agent
require:
  - field: description
    is-present: true
    min-length: 20
---
```

This emits a `warning` (visible in stderr but not blocking) whenever an agent is missing a `description` or has a description shorter than 20 characters. Pair it with a stricter model policy:

```
---
kind: policy
version: "1.0"
name: require-approved-models
description: Agents may only use pre-approved model identifiers to control cost and capability.
severity: error
target: agent
require:
  - field: model
    one-of:
      - sonnet-4
      - haiku-3.5
      - opus-4
---
```

Because this is `error` severity, compiling an agent with `model: gpt-4o` will fail immediately with a clear diagnostic. No agent using an unapproved model will ever make it into the compiled output.

---

## Use Case 2 — Locking Down Production Agents

For production deployments, you often want stricter rules on agents that have broad permissions. Use `match:` to apply a policy only to agents with specific naming patterns:

```
---
kind: policy
version: "1.0"
name: prod-agents-require-instructions
description: Production agents must have a system prompt body describing their role and constraints.
severity: error
target: agent
match:
  name-matches: "*-prod"
require:
  - field: instructions
    is-present: true
    min-length: 50
---
```

```
---
kind: policy
version: "1.0"
name: prod-agents-limit-tools
description: Production agents must not have more than 5 tools to reduce blast radius.
severity: error
target: agent
match:
  name-matches: "*-prod"
require:
  - field: tools
    max-count: 5
---
```

Both policies target only agents whose names end in `-prod` (e.g., `api-prod`, `db-prod`). Development agents are unaffected.

> [!NOTE]
> The `instructions` field maps to the agent's markdown body — the system prompt below the frontmatter delimiters.

---

## Use Case 3 — Preventing Secrets From Leaking Into Output

One of the highest-value uses of policies is blocking hardcoded credentials and tokens from ever appearing in compiled output. Use `target: output` with `deny.content-matches` to run a regex over every compiled file:

```
---
kind: policy
version: "1.0"
name: no-hardcoded-api-keys
description: Compiled output must never contain hardcoded API keys or tokens.
severity: error
target: output
deny:
  - content-matches: "sk-[a-zA-Z0-9]{20,}"
  - content-matches: "ghp_[a-zA-Z0-9]{36}"
  - content-contains:
      - "AKIA"
      - "Bearer eyJ"
---
```

This runs after compilation — before any files are written to disk — and fails immediately if a compiled artifact contains a pattern matching a known secret format. The patterns above catch OpenAI keys, GitHub tokens, AWS access key IDs, and common JWT patterns.

---

## Use Case 4 — Governing Skill Quality

Skills without proper documentation make it hard for agents to know when to invoke them. Enforce baseline quality on all skills:

```
---
kind: policy
version: "1.0"
name: require-skill-description
description: All skills must have a description. Agents use descriptions for automatic invocation decisions.
severity: warning
target: skill
require:
  - field: description
    is-present: true
---
```

For skills that carry procedural instructions, you may want to require a body (the markdown content below the frontmatter):

```
---
kind: policy
version: "1.0"
name: skills-must-have-instructions
description: Skills must include a body with procedural instructions, not just metadata.
severity: warning
target: skill
require:
  - field: instructions
    is-present: true
    min-length: 20
---
```

> [!NOTE]
> The `instructions` field maps to the skill's markdown body, just as it does for agents and rules.

---

## Use Case 5 — Disabling a Built-In Check

xcaffold ships built-in compiler policies. If a built-in check doesn't apply to your project (for example, a project that intentionally ships minimal skills without descriptions), you can silence it without deleting your source files:

```
---
kind: policy
version: "1.0"
name: allow-empty-skills
severity: off
target: skill
---
```

Setting `severity: off` disables the policy entirely. This is version-controlled and self-documenting — better than a command-line flag that disappears from institutional memory.

---

## Organizing Policy Files

Store policy files in `xcaf/policies/`. The scanner discovers them by `kind: policy` frontmatter — placement within `xcaf/` is convention, not enforced:

```
xcaf/
└── policies/
    ├── quality/
    │   ├── require-agent-description.xcaf
    │   └── require-skill-description.xcaf
    ├── security/
    │   ├── no-hardcoded-api-keys.xcaf
    │   └── skills-must-have-instructions.xcaf
    └── production/
        ├── prod-agents-require-instructions.xcaf
        └── prod-agents-limit-tools.xcaf
```

Grouping by concern (quality, security, production) makes it easy to apply CODEOWNERS rules and understand what governance your project has at a glance.

---

## Severity Decision Guide

| When to use | Severity |
|---|---|
| Missing metadata that degrades UX or auto-invocation | `warning` |
| Unapproved models, tools, or patterns that could cause cost overruns | `error` |
| Security-critical constraints (secrets, dangerous commands) | `error` |
| Rules you want to exist but not block CI right now | `warning` |
| Built-in checks that don't apply to your project | `off` |

---

## Supported Policy Fields

The `require:` clause evaluates fields by name. Not every resource field is available for policy evaluation — only the fields listed below are handled by the policy engine. Using a field not in this table will always evaluate as empty, producing a false violation.

### `target: agent` — require fields

| Field name | What it checks | Supports |
|---|---|---|
| `name` | Agent's `name:` value | `is-present`, `min-length`, `one-of` |
| `description` | Agent's `description:` value | `is-present`, `min-length`, `one-of` |
| `model` | Agent's `model:` value | `is-present`, `min-length`, `one-of` |
| `instructions` | Agent's markdown body (system prompt) | `is-present`, `min-length` |
| `tools` | Agent's `tools:` list | `max-count` |
| `skills` | Agent's `skills:` list | `max-count` |
| `rules` | Agent's `rules:` list | `max-count` |

### `target: skill` — require fields

| Field name | What it checks | Supports |
|---|---|---|
| `name` | Skill's `name:` value | `is-present`, `min-length`, `one-of` |
| `description` | Skill's `description:` value | `is-present`, `min-length`, `one-of` |
| `instructions` | Skill's markdown body | `is-present`, `min-length` |

### `target: rule` — require fields

| Field name | What it checks | Supports |
|---|---|---|
| `name` | Rule's `name:` value | `is-present`, `min-length`, `one-of` |
| `description` | Rule's `description:` value | `is-present`, `min-length`, `one-of` |
| `instructions` | Rule's markdown body | `is-present`, `min-length` |

### `target: output` — deny fields

| Field name | What it checks |
|---|---|
| `content-matches` | Regex match against compiled file content |
| `content-contains` | Substring match against compiled file content (case-insensitive) |
| `path-contains` | Substring match against compiled file paths |

### `match:` conditions

| Condition | What it does |
|---|---|
| `name-matches` | Glob pattern against resource name (e.g., `*-prod`) |
| `has-tool` | True when the resource's tool list includes the named tool |
| `has-field` | True when the named field is present and non-empty |
| `target-includes` | True when the resource's `targets:` map includes the named provider |
