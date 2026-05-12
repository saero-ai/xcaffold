---
title: "kind: policy"
description: "Defines declarative constraints and governance rules. Source: `xcaf/policies/<id>/policy.xcaf`."
---

# `kind: policy`

Defines declarative constraints and governance rules evaluated during `xcaffold validate` and `xcaffold apply`. Policies ensure that project resources and compiled output adhere to organizational standards and security best practices.

> **Required:** `kind`, `version`, `name`, `severity`, `target`

## Source Directory

```
xcaf/policies/<name>/policy.xcaf
```

## Example Usage

```yaml
kind: policy
version: "1.0"
name: require-description
severity: error
target: agent
require:
  - field: description
    is-present: true
```

## Field Reference

### Required Fields

| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | `string` | Unique identifier for the policy. Must match `[a-z0-9-]+`. |
| `severity` | `string` | Violation level when the policy fails: `error`, `warning`, `off`. |
| `target` | `string` | Resource kind this policy evaluates: `agent`, `skill`, `rule`, `hook`, `settings`, `output`. |

### Optional Fields

#### Identity & Matching

| Field | Type | Description |
| :--- | :--- | :--- |
| `description` | `string` | Human-readable purpose of this policy. |
| `match` | `PolicyMatch` | Filter conditions selecting which resources to evaluate. |

#### `PolicyMatch` Fields

| Field | Type | Description |
| :--- | :--- | :--- |
| `has-tool` | `string` | Matches resources with a specific tool granted. |
| `has-field` | `string` | Matches resources with a specific field defined. |
| `name-matches` | `string` | Regex pattern matching the resource name. |
| `target-includes` | `string` | Matches resources targeting a specific provider. |

#### Constraints

| Field | Type | Description |
| :--- | :--- | :--- |
| `require` | `[]PolicyRequire` | List of field constraints applied to matched resources. |
| `deny` | `[]PolicyDeny` | Forbidden patterns in compiled output content or paths. |

#### `PolicyRequire` Fields

| Field | Type | Description |
| :--- | :--- | :--- |
| `field` | `string` | The field name to evaluate. |
| `is-present` | `bool` | Whether the field must be defined. |
| `min-length` | `int` | Minimum character length for string fields. |
| `max-count` | `int` | Maximum element count for list fields. |
| `one-of` | `[]string` | List of permitted values for the field. |

#### `PolicyDeny` Fields

| Field | Type | Description |
| :--- | :--- | :--- |
| `content-contains` | `[]string` | Forbidden literal strings in the content. |
| `content-matches` | `string` | Forbidden regex pattern in the content. |
| `path-contains` | `string` | Forbidden literal string in the output path. |

## Filesystem-as-Schema

When a policy is defined at `xcaf/policies/<id>/policy.xcaf`, Xcaffold automatically infers:
- **kind**: `policy` derived from the `policies/` directory.
- **name**: `<id>` derived from the directory segment between the kind and the filename.

## Behavior

1.  **Pipeline Integration**: Policies are evaluated during `validate` and `apply` phases.
2.  **Gatekeeping**: An `error` level violation prevents `xcaffold apply` from writing any files to disk.
3.  **Global Scope**: Policy `.xcaf` files placed under `~/.xcaffold/` are evaluated globally for all projects. The scanner recursively discovers all `.xcaf` files in the global home directory.
