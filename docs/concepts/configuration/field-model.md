---
title: "Field Model"
description: "Two-layer field classification: xcaffold core roles and provider field support"
---

# Field Model

xcaffold uses a two-layer field classification system that separates
concerns between the compilation pipeline and provider rendering.

## Layer 1: xcaffold Core Fields

Every field on `AgentConfig`, `SkillConfig`, and `RuleConfig` carries a
role annotation (`+xcaf:role=`) that declares its function in the
compilation pipeline:

| Role | Meaning | Example Fields |
|------|---------|----------------|
| `identity` | Names the resource | `name` |
| `rendering` | Passed to the provider renderer | `model`, `tools`, `description` |
| `composition` | Resolved during compilation | `skills`, `rules`, `mcp` |
| `metadata` | Informational, not rendered | `color`, `when-to-use`, `license` |
| `filtering` | Controls compilation scope | `targets` |

Fields with role annotations are **xcaffold core fields**. They exist to
serve the compiler regardless of whether the target provider supports
them natively.

## Layer 2: Provider Field Support

Each provider declares field support in `providers/<name>/fields.yaml`:

| Classification | Meaning |
|----------------|---------|
| `optional` | Provider renders this field when present |
| `required` | Provider requires this field; absent field is a compilation error |
| `unsupported` | Provider does not render this field |

The `fields.yaml` schema is keyed by kind (`agent`, `skill`, `rule`) and
then by the xcaffold field name. For example:

```yaml
provider: claude
version: "1.0"
kinds:
  agent:
    description: { support: required }
    model: { support: optional }
    color: { support: unsupported }
```

## Fidelity Behavior

The fidelity checker combines both layers when determining whether to
render, skip, or reject a field:

| Layer 1 (Role) | Layer 2 (Provider Support) | Result |
|----------------|---------------------------|--------|
| has role | `optional` | Field is rendered when present |
| has role | `required` | Field must be present; absent field emits `FIELD_REQUIRED_FOR_TARGET` |
| has role | `unsupported` | Field is silently skipped (core fields serve the compiler regardless of provider) |
| (no role annotation) | `unsupported` | Error-level `FIELD_UNSUPPORTED` fidelity note if field is present |

Composition fields (`skills`, `rules`, `mcp`) are resolved during
compilation before the renderer runs. The provider never sees the
reference list — it sees the resolved output. This means a `rules: [...]`
declaration on an agent does not produce an error or a fidelity note when
targeting a provider that marks `rules` as `unsupported`; the compiler
resolves the rule content into the agent instructions before handing off
to the renderer.

## Override Semantics: ClearableList

List fields (`tools`, `skills`, `rules`, `allowed-tools`, `paths`, etc.)
use the `ClearableList` type for override merging. `ClearableList`
distinguishes three states that a plain `[]string` cannot:

| YAML Value | Internal State | Meaning |
|-----------|---------------|---------|
| *(absent)* | `Cleared=false`, `Values=nil` | Inherit from base |
| `[]` or `~` | `Cleared=true`, `Values=nil` | Clear — remove base values |
| `[a, b]` | `Cleared=false`, `Values=[a,b]` | Replace with these values |

This enables override files to explicitly remove inherited values without
ambiguity. A zero-value `ClearableList` always means "inherit"; an
explicit empty sequence always means "clear".

### Example

```yaml
# base agent.xcaf
tools: [Bash, Read, Write]

# agent.cursor.xcaf override
tools: []  # cleared — cursor agent gets no tools list
```

The cursor renderer sees a cleared `tools` field and omits it from the
compiled output, rather than inheriting the base `[Bash, Read, Write]`.
