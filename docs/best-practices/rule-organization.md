---
title: "Rule Organization"
description: "How to structure rule resources, choose activation modes, and get consistent behavior across providers."
---

# Rule Organization

Rules are behavioral guidelines compiled into provider-native instruction files. They control *when* an AI agent receives specific instructions — always, only when touching certain files, or only when explicitly invoked. Unlike agents or skills, rules don't orchestrate a task; they set boundaries and conventions that apply across all tasks.

This guide covers practical use cases — how to use rules effectively and how activation works across providers. For field-level reference, see the [Schema Reference](../reference/kinds/provider/rule.md).

## Choosing an Activation Mode

The `activation:` field is the most important decision when authoring a rule. It determines which conversations the rule enters.

| Mode | When to use |
|---|---|
| `always` | Universal standards that must apply in every conversation (coding style, security constraints, commit format) |
| `path-glob` | File-type or directory-specific guidance (frontend CSS conventions, test writing standards, database schema rules) |
| `manual-mention` | Specialized guidance that is too narrow or verbose to load in every conversation — referenced by name when needed |
| `model-decided` | Context-sensitive guidance where relevance depends on what the user is asking, not on which files are open |
| `explicit-invoke` | Strictly opt-in rules that agents or users must consciously invoke (onboarding checklists, migration playbooks) |

When no `activation:` is set and no `paths:` are provided, the compiler defaults to `always`.

### Explicit vs. shorthand

`activation: always` and `always-apply: true` produce identical compiled output. Use whichever reads more clearly in your file — `activation:` makes the intent explicit, while `always-apply:` is concise for simple rules. The same equivalence holds for `manual-mention`:

| Shorthand | Equivalent |
|---|---|
| `always-apply: true` | `activation: always` |
| `always-apply: false` | `activation: manual-mention` |

If `activation:` is set, it takes precedence over `always-apply:`.

## Global Standards

A rule with `activation: always` is loaded into every conversation and applies unconditionally. Use this for conventions that must hold regardless of context:

```yaml
---
kind: rule
version: "1.0"
name: commit-format
description: "Commit messages must follow Conventional Commits."
activation: always
---
All commit messages must use the format `type(scope): description`.

Allowed types: feat, fix, refactor, docs, test, chore.
Never use vague messages like "misc", "WIP", or "updates".
```

You can also write this as:

```yaml
---
kind: rule
version: "1.0"
name: commit-format
description: "Commit messages must follow Conventional Commits."
always-apply: true
---
All commit messages must use the format `type(scope): description`.
```

Both produce the same compiled output.

## Path-Scoped Rules

A rule with `activation: path-glob` is only loaded when the active file or conversation context matches one of the patterns in `paths:`. This is the right choice when a guideline is specific to part of your codebase.

```yaml
---
kind: rule
version: "1.0"
name: frontend-css
description: "CSS authoring conventions for the frontend package."
activation: path-glob
paths:
  - "packages/web/**/*.css"
  - "packages/web/**/*.tsx"
---
Use Tailwind utility classes exclusively. Do not write custom CSS unless there is no Tailwind equivalent.
Class order: layout → spacing → typography → color → interaction.
```

```yaml
---
kind: rule
version: "1.0"
name: test-conventions
description: "Test file authoring standards."
activation: path-glob
paths:
  - "**/*_test.go"
  - "**/test/**"
---
Every test function must follow the naming convention `Test<Unit>_<Feature>_<Scenario>`.
Always add a negative case (error path) for every positive case tested.
```

Rules do not require subdirectories — a flat file at `xcaf/rules/test-conventions.xcaf` is valid. Use the directory-per-resource layout (`xcaf/rules/test-conventions/rule.xcaf`) only when you expect to add supporting files alongside the rule in the future.

## On-Demand Guidance

A rule with `activation: manual-mention` is not loaded automatically. It enters the conversation only when the user or agent references it by name. Use this for guidance that is relevant in specific, predictable situations but would add noise if always present:

```yaml
---
kind: rule
version: "1.0"
name: database-migration-checklist
description: "Pre-flight checklist for writing database migrations."
activation: manual-mention
---
Before writing a migration:
1. Check that the migration is reversible (has a Down step).
2. Verify the migration does not rename a column used by more than two queries.
3. Add a comment explaining why the schema change is needed.
4. Run the migration against a test database before committing.
```

A user invokes this by mentioning it in conversation: "Apply the database-migration-checklist before we write this migration."

## AI-Decided Relevance

A rule with `activation: model-decided` lets the model itself determine whether the rule applies to the current conversation. The model reads the rule's `description:` field to make that determination:

```yaml
---
kind: rule
version: "1.0"
name: performance-review
description: "Apply when the user is investigating performance problems, profiling, or optimizing for latency or throughput."
activation: model-decided
---
When analyzing performance, always measure before optimizing.
Prefer benchmarks (`go test -bench`) over intuition.
Document the baseline measurement alongside any optimization.
```

The description is the signal the model uses to decide. Write it as a precise "apply when" statement rather than a general summary. Note that `model-decided` is only supported by a subset of providers — see the Provider Support table below.

## Strictly Opt-In Rules

A rule with `activation: explicit-invoke` requires a deliberate invocation to load. It is stricter than `manual-mention` — it does not activate from casual name references in conversation. Use it for high-stakes playbooks where accidental activation could cause harm:

```yaml
---
kind: rule
version: "1.0"
name: production-deploy-runbook
description: "Runbook for authorizing and executing a production deployment."
activation: explicit-invoke
---
Production deployments require explicit authorization.
1. Confirm the release SHA has passed all CI checks.
2. Notify the on-call engineer before any schema migration.
3. Deploy during the maintenance window (UTC 02:00–04:00) unless critical.
```

## Provider Activation Support

Provider support for activation modes varies. Some modes (like `model-decided` and `explicit-invoke`) are only available on providers that support agent-decided file selection. See the [Schema Reference](../reference/kinds/provider/rule.md) for the full provider activation support matrix.

## Organizing Rule Files

The scanner discovers rules by `kind: rule` frontmatter — placement within `xcaf/` is convention, not enforced. Two layouts are supported:

**Flat files** — simpler, preferred for projects with few rules:

```
xcaf/
└── rules/
    ├── commit-format.xcaf
    ├── frontend-css.xcaf
    └── test-conventions.xcaf
```

**Directory-per-resource** — preferred when rules may grow to include supporting files:

```
xcaf/
└── rules/
    ├── commit-format/
    │   └── rule.xcaf
    ├── frontend-css/
    │   └── rule.xcaf
    └── test-conventions/
        └── rule.xcaf
```

Unlike `kind: skill` or `kind: agent`, rules do not require the directory layout — a flat file is fully valid and will not trigger CLI warnings. Adopt directories for rules only when you anticipate adding reference documents or examples alongside the rule, or when grouping by concern:

```
xcaf/
└── rules/
    ├── code-quality/
    │   ├── commit-format.xcaf
    │   └── no-todos.xcaf
    ├── security/
    │   ├── no-secrets.xcaf
    │   └── no-exec-strings.xcaf
    └── file-type/
        ├── frontend-css.xcaf
        └── test-conventions.xcaf
```

Grouping by concern makes it easy to understand what categories of guidance your project enforces at a glance.

## Rule Composition with Agents

Rules are referenced from agent manifests using the `rules:` field. Unlike rules, agents should always use the directory-per-resource layout (`xcaf/agents/<name>/agent.xcaf`) to support memory discovery without CLI warnings:

```yaml
# xcaf/agents/developer/agent.xcaf
---
kind: agent
version: "1.0"
name: developer
description: "General software developer."
model: sonnet
tools: [Bash, Read, Write, Edit, Glob, Grep]
rules:
  - commit-format
  - test-conventions
---
You are a software developer.
```

Only the rules listed in `rules:` are compiled into the agent's context bundle. An agent that lists no rules receives none.

Rules that are not referenced by any agent are still compiled and written to disk. Their activation mode determines whether the provider loads them — an `always` rule with no agent reference is still loaded globally by providers that support global instruction files.

## Decision Guide

| I need... | Use |
|---|---|
| A rule that applies to every conversation, no exceptions | `activation: always` |
| A rule that applies only when certain files are open or edited | `activation: path-glob` with `paths:` globs |
| A rule that loads when the user mentions it by name | `activation: manual-mention` |
| A rule where the model decides if it's relevant | `activation: model-decided` |
| A rule that requires explicit, deliberate invocation | `activation: explicit-invoke` |
| A shorthand for always-on | `always-apply: true` |
| A shorthand for mention-to-activate | `always-apply: false` |
| Universal support across all providers | `always` or `path-glob` |
| On-demand loading with broadest provider support | `manual-mention` (Cursor only for now) |

## Related

- [Schema Reference — RuleConfig](../reference/kinds/provider/rule.md) — field-level documentation for all rule fields
- [Policy Organization](policy-organization.md) — compile-time governance rules that block invalid output
- [Agent Design Patterns](agent-design-patterns.md) — patterns for agent composition that reference rules
- [Workspace Context](workspace-context.md) — project-level instructions that apply across all agents
