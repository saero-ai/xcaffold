---
title: "Variables and Overrides"
description: "How to inject shared values with variable files and customize resources per provider with override manifests — without duplicating .xcaf files."
---

# Variables and Overrides

xcaffold provides two mechanisms for customizing compiled output without duplicating `.xcaf` files. **Variables** inject shared values into any manifest at compile time. **Overrides** let you adjust a single resource's fields for a specific provider while keeping the base manifest provider-neutral.

Used together, they keep your `xcaf/` tree small, readable, and non-repetitive regardless of how many providers you target.

---

## When to Use Variables

Variables are the right tool when the same value appears in multiple manifests or when the value differs between environments but the manifest structure does not:

- A model identifier used by several agents that you want to change in one place
- A shared API base URL that differs between local and CI environments
- A team name or product name that appears in multiple descriptions
- Any value that a developer should be able to override locally without touching committed files

If the difference between two environments is a handful of scalar values, use variables. If the difference is structural — different fields, different tools, different behavior — use overrides instead.

---

## Variable File Stack

xcaffold reads variable files in a fixed precedence order. Files loaded later win over files loaded earlier:

| Load order | File | Purpose |
|---|---|---|
| 1 (lowest) | `xcaf/project.vars` | Base variables, committed to git |
| 2 | `xcaf/project.<target>.vars` | Target-specific overrides (e.g., `project.claude.vars`) |
| 3 (highest) | `xcaf/project.vars.local` | Local developer overrides, gitignored |
| — | `--var-file ./path` flag | Custom file passed to `xcaffold apply` or `xcaffold validate` |

A practical three-layer setup looks like this:

`xcaf/project.vars` — committed, shared across all targets:
```
# Shared defaults
model = haiku-3.5
api-base = https://api.example.com
team = platform
```

`xcaf/project.claude.vars` — committed, applies only when `--target claude` is active:
```
# Claude-specific defaults
model = sonnet-4
```

`xcaf/project.vars.local` — gitignored, developer-local:
```
# Local overrides — never committed
api-base = http://localhost:8080
```

When compiling for the Claude target, `model` resolves to `sonnet-4` (from the target file). When compiling for any other target, `model` resolves to `haiku-3.5` (from the base file). A developer running locally sees `http://localhost:8080` for `api-base` regardless of target.

Add `xcaf/project.vars.local` to your `.gitignore` to keep local overrides out of version control.

---

## Variable Syntax and Types

Each line in a variable file follows `key = value` format:

```
# Comment lines begin with #
# Empty lines are ignored

model = haiku-3.5
max-tokens = 4096
enable-streaming = true
allowed-providers = [claude, cursor, gemini]
```

Variable names must start with a letter and may contain letters, digits, underscores, and hyphens: `^[a-zA-Z][_a-zA-Z0-9-]*$`. Names like `2nd-pass` or `-flag` are rejected at parse time.

Values are parsed as YAML scalars and sequences:

| Value written | Type resolved |
|---|---|
| `haiku-3.5` | string |
| `4096` | integer |
| `true` / `false` | boolean |
| `[claude, cursor]` | list of strings |

Strings do not require quotes unless they contain characters that YAML would otherwise interpret (colons, brackets, etc.).

---

## Using Variables in Manifests

Reference a variable in any `.xcaf` file using `${var.name}`:

```yaml
---
kind: agent
version: "1.0"
name: reviewer
description: "Code reviewer for the ${var.team} team."
model: ${var.model}
tools: [Read, Glob, Grep]
---
Review all changed files. Use ${var.model} to reason about trade-offs.
```

When `xcaffold apply` runs, every `${var.name}` is replaced with the resolved value before compilation proceeds.

### Environment Variable References

To reference a shell environment variable, use `${env.NAME}`. Environment variable access is opt-in: the variable name must appear in the project's `allowed-env-vars` field, declared in your `kind: project` manifest:

```yaml
---
kind: project
version: "1.0"
name: my-project
allowed-env-vars:
  - CI_ENVIRONMENT
  - DEPLOY_TARGET
---
```

Once declared, you can reference the variable in any manifest:

```yaml
---
kind: rule
version: "1.0"
name: env-context
description: "Active environment context."
always-apply: true
---
Current environment: ${env.CI_ENVIRONMENT}
```

If a manifest references `${env.NAME}` and `NAME` is not in `allowed-env-vars`, xcaffold reports an error and halts compilation. This prevents accidental leakage of sensitive environment variables into compiled output.

---

## Variable Composition

Variables can reference other variables. This lets you build derived values from shared primitives:

```
# xcaf/project.vars
provider = claude
base-url = https://api.example.com
api-url = ${var.base-url}/v1/${var.provider}
```

Resolution runs up to 10 passes, resolving references iteratively until all values are fully expanded. If two variables reference each other in a cycle (`a = ${var.b}`, `b = ${var.a}`), xcaffold detects the cycle and reports an error before compilation begins.

Keep composition shallow — one or two levels is usually sufficient. Deeply nested chains make it hard to trace the final resolved value.

---

## Pattern: Shared Values Between Frontmatter and Body

Variables are most effective when they bridge your frontmatter configuration and the agent's system prompt. Define shared facts once in `project.vars` and reference them in both places:

`xcaf/project.vars`:
```
framework = React 19
test-runner = vitest
orm = Drizzle
```

`xcaf/agents/frontend-dev/agent.xcaf`:
```yaml
---
kind: agent
version: "1.0"
name: frontend-dev
description: "Builds ${var.framework} components and writes ${var.test-runner} tests."
model: sonnet
tools: [Bash, Read, Write, Edit, Glob, Grep]
---
You are a frontend developer. This project uses ${var.framework} with ${var.orm} for data access.

Run tests with ${var.test-runner}. Do not use jest or mocha.
When writing queries, use ${var.orm} exclusively — no raw SQL.
```

Changing `test-runner = vitest` to `test-runner = jest` in one file updates both the agent's description and its behavioral instructions.

### Cross-Resource References

Beyond project variables, you can reference fields from other resources using `${resource_type.resource_name.field}`. This is useful when an agent needs to mention a skill or rule by its description:

```yaml
---
kind: agent
version: "1.0"
name: developer
description: "Developer following ${skill.tdd.description}."
model: sonnet
skills: [tdd, code-review]
---
You are a developer. Your primary methodology is: ${skill.tdd.description}.
When reviewing your own work, follow the "${skill.code-review.description}" process.
```

The compiler resolves `${skill.tdd.description}` to the actual `description:` value from the `tdd` skill at compile time. This keeps agent instructions synchronized with skill definitions — rename a skill's description in one place and every referencing agent updates automatically.

Cross-resource references support cycle detection: if skill A references agent B's description and agent B references skill A's description, xcaffold detects the cycle and reports an error.

---

## When to Use Overrides

Overrides are the right tool when a resource needs structurally different configuration for a specific provider:

- An agent that uses a more capable model on one provider and a faster model on another
- A skill that should expose additional tools only on a provider that enforces tool allowlists
- A rule body that must use provider-native phrasing to be interpreted correctly
- A workflow that requires different environment settings per provider

If the difference is just a value (same field, different content), a target-specific variable file may be cleaner. If the difference is which fields are present, use an override.

---

## Override File Convention

An override is a `.xcaf` file placed alongside the base resource and named `<kind>.<provider>.xcaf`:

```
xcaf/agents/
└── developer/
    ├── agent.xcaf            ← base resource (all providers)
    └── agent.claude.xcaf     ← override applied only when --target claude
```

The resource name is inferred from the parent directory — the override above applies to the `developer` agent. You do not declare the name again in the override file.

The same pattern applies to every supported kind:

```
xcaf/skills/
└── code-review/
    ├── skill.xcaf
    └── skill.cursor.xcaf     ← cursor-only override

xcaf/rules/
└── secure-coding/
    ├── rule.xcaf
    └── rule.gemini.xcaf      ← gemini-only override
```

Every override must have a corresponding base resource in the same directory. An override without a base file causes a parse error.

Supported kinds for overrides: `agent`, `skill`, `rule`, `workflow`, `mcp`, `hooks`, `settings`, `policy`, `template`. `memory` resources do not participate in the override system.

---

## Override Merge Behavior

When xcaffold compiles for a given target, it merges the override into the base using these rules:

| Field type | Merge behavior |
|---|---|
| Scalar (string, int, bool pointer) | Override value replaces base when non-zero / non-nil |
| List | Override list replaces entire base list when non-empty; explicit clear wipes the list |
| Map | Deep merge — override keys win, base keys without a matching override key are preserved |
| Body (markdown below `---`) | Override body replaces base body when non-empty; base body inherited when override body is absent |

A concrete example: a `developer` agent that uses a faster model on Cursor than on Claude:

`xcaf/agents/developer/agent.xcaf` (base):
```yaml
---
kind: agent
version: "1.0"
name: developer
description: "General-purpose software developer."
model: sonnet-4
tools: [Bash, Read, Write, Edit, Glob, Grep]
---
You are a software developer. Follow the project conventions.
```

`xcaf/agents/developer/agent.claude.xcaf` (Claude override):
```yaml
---
kind: agent
version: "1.0"
model: opus-4
tools: [Bash, Read, Write, Edit, Glob, Grep, WebFetch]
---
```

When compiling for `--target claude`, the compiled agent uses `model: opus-4` and the extended tool list. For all other targets, the base `model: sonnet-4` and the original tool list apply. The body is not present in the override, so the base body is inherited by all targets.

### Silencing Fidelity Warnings

xcaffold emits a fidelity note to stderr for each field that is dropped because a target provider has no native equivalent. If you intentionally authored a field knowing it will not appear in a specific provider's output, you can silence the warning using `suppress-fidelity-warnings`:

```yaml
---
kind: agent
version: "1.0"
name: developer
targets:
  gemini:
    suppress-fidelity-warnings: true
---
```

This tells xcaffold that the field drops for Gemini are deliberate. The compiled output is unchanged; only the warnings are suppressed.

---

## Target Filtering

A resource can be excluded from specific providers entirely using the `targets:` map on the base manifest. This is distinct from overrides — it removes the resource from compilation rather than adjusting its fields:

```yaml
---
kind: skill
version: "1.0"
name: copilot-helper
description: "Copilot-specific workflow helper."
targets:
  copilot: {}
---
```

A resource with a `targets:` map is compiled only for the providers listed. A resource without a `targets:` map is compiled for all providers. Use `targets:` when a resource is provider-specific by nature, not just by configuration.

---

## Decision Guide

| Scenario | Tool |
|---|---|
| Same field value repeated across multiple manifests | Variable |
| Value differs between local and CI environments | Variable + `project.vars.local` |
| Value differs between providers (same field) | Target-specific variable file |
| Field set, tool list, or body differs per provider | Override |
| Resource should only exist for one or two providers | `targets:` map on the base resource |
| Silencing a known fidelity warning | `targets.<provider>.suppress-fidelity-warnings` on the resource |
