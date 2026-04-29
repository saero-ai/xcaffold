---
title: "kind: policy"
description: "Declarative compile-time constraint evaluated during xcaffold apply and validate. Produces no output files."
---

# `kind: policy`

Defines a compile-time constraint evaluated against the parsed AST during `xcaffold apply` and `xcaffold validate`. Policies produce **no output files** — they run in-process and emit diagnostics to stderr.

Uses **pure YAML format** (no frontmatter `---` delimiters).

> **Required:** `kind`, `version`, `name`, `severity`, `target`

## Example Usage

### Require an approved model

```yaml
kind: policy
version: "1.0"
name: require-approved-model
description: >-
  All agents in frontend-app must declare a model from the approved list.
  Unapproved models lack the tool-use capabilities required by
  component-patterns and the browser-tools MCP.
severity: error
target: agent
require:
  - field: model
    one-of:
      - claude-opus-4-5-20250514
      - claude-sonnet-4-5-20250514
      - claude-haiku-4-5-20251001
```

### Deny secrets in compiled output

```yaml
kind: policy
version: "1.0"
name: no-secrets-in-output
description: Prevent API keys and tokens from appearing in compiled provider files.
severity: error
target: output
deny:
  - content-contains:
      - "sk-ant-"
      - "ANTHROPIC_API_KEY="
      - "ghp_"
  - content-matches: "(AKIA[0-9A-Z]{16})"
```

### Require component descriptions on agents

```yaml
kind: policy
version: "1.0"
name: require-agent-description
description: All agents must declare a description for delegation routing.
severity: warning
target: agent
match:
  name-matches: "*-developer"
require:
  - field: description
    is-present: true
    min-length: 20
```

## Argument Reference

The following arguments are supported:

- `name` — (Required) Unique policy identifier. Must match `[a-z0-9-]+`.
- `version` — (Required) Schema version. Use `"1.0"`.
- `severity` — (Required) Diagnostic severity:
  - `"error"` — blocks `xcaffold apply` with a non-zero exit code
  - `"warning"` — emits to stderr but does not block apply
  - `"off"` — policy is loaded but not evaluated
- `target` — (Required) Resource type the policy applies to: `"agent"`, `"skill"`, `"rule"`, `"hook"`, `"settings"`, `"output"`.
- `description` — (Optional) `string`. Human-readable policy intent.
- `match` — (Optional) Filter conditions (see [match block](#match-block)).
- `require` — (Optional) `[]PolicyRequire`. Field value constraints (see [require block](#require-block)).
- `deny` — (Optional) `[]PolicyDeny`. Forbidden content patterns (see [deny block](#deny-block)).

### `match` block

Narrows which resources this policy applies to. All conditions are AND-ed.

- `has-tool` — `string`. Tool name that must be present in `tools` for the policy to apply.
- `has-field` — `string`. Field name that must be non-zero on the resource.
- `name-matches` — `string`. Glob pattern matched against the resource `name` (e.g., `"*-developer"`).
- `target-includes` — `string`. Target name that must appear in the resource's `targets` map.

### `require` block

Asserts a field meets a value constraint. All entries must pass.

- `field` — (Required) Dot-path to the checked field: `"model"`, `"permissions.defaultMode"`.
- `is-present` — `bool`. `true` = field must be non-zero; `false` = field must be zero/empty.
- `min-length` — `int`. Minimum string or list length.
- `max-count` — `int`. Maximum list item count.
- `one-of` — `[]string`. Allowed values. Policy passes if the field value matches any entry.

### `deny` block

Asserts forbidden content does not appear. A single match causes the policy to fail.

- `content-contains` — `[]string`. Substrings that must not appear in compiled output. Requires `target: output`.
- `content-matches` — `string`. Regex forbidden in compiled output content. Requires `target: output`.
- `path-contains` — `string`. Substring that must not appear in any compiled output file path.

## Diagnostics

When a policy violation is detected, xcaffold emits a structured diagnostic:

```
[policy error] require-approved-model: agent "react-developer" — field "model" must be
  one of [claude-opus-4-5-20250514, claude-sonnet-4-5-20250514, claude-haiku-4-5-20251001],
  got "gpt-4o"
Error: 1 policy violation(s) blocked apply. Run `xcaffold validate` to see all violations.
```

`severity: warning` diagnostics use `[policy warning]` prefix and do not block apply.
