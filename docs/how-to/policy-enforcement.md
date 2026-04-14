# Enforcing Project Policies

xcaffold evaluates policies during `xcaffold apply` and `xcaffold validate`. Violations with `severity: error` block compilation and prevent `.claude/` from being written. You can write custom policies in `.xcf` files and override or disable any built-in policy by name.

---

## Writing a Custom Policy

Create policy `.xcf` files alongside your other resource files. Reference them from your `kind: project` manifest using the `policies:` list.

**Example `kind: project` manifest referencing custom policies:**

```yaml
kind: project
version: "1.0"
name: my-project
targets:
  - claude
policies:
  - require-approved-model
  - no-leaked-todos
```

Each name in `policies:` maps to a `kind: policy` document — either in the same file (separated by `---`) or in a separate `.xcf` file parsed alongside the project manifest.

**Example: require all agents to have a model from an approved list.**

Create `require-approved-model.xcf`:

```yaml
kind: policy
name: require-approved-model
description: Agents must use an approved model identifier
severity: error
target: agent
require:
  - field: model
    one_of:
      - claude-opus-4-5
      - claude-sonnet-4-5
      - claude-haiku-3-5
```

This policy evaluates every agent in the configuration. If an agent's `model` field does not match one of the listed values, the engine emits an error-severity violation.

**Policy schema fields:**

| Field | Type | Purpose |
|---|---|---|
| `kind` | `string` | Must be `policy` |
| `name` | `string` | Unique identifier. Matches against built-in names for overrides |
| `description` | `string` | Human-readable explanation of the policy's purpose |
| `severity` | `string` | `error` (blocks apply), `warning` (diagnostic only), or `off` (disabled) |
| `target` | `string` | Resource type to evaluate: `agent`, `skill`, `output`, or `settings` |
| `match` | `PolicyMatch` | Optional filter conditions. See [Using Match Conditions](#using-match-conditions-to-filter-resources) |
| `require` | `[]PolicyRequire` | Field presence and value constraints on matched resources |
| `deny` | `[]PolicyDeny` | Content and path patterns that must not appear in compiled output |

See `docs/reference/schema.md` for the full field reference including `PolicyRequire` and `PolicyDeny` sub-schemas.

---

## Running Policy Evaluation

Both `xcaffold apply` and `xcaffold validate` load built-in policies first, then merge with user-defined policies from the parsed configuration. Custom policies with the same `name` as a built-in override it.

### Passing run (no violations)

```
$ xcaffold validate
syntax and cross-references: ok
policies: ok

validation passed
```

### Run with warning violations

Warnings are printed to stderr but do not block compilation. `xcaffold apply` writes the output directory and exits with code 0.

```
$ xcaffold validate
syntax and cross-references: ok
POLICY VIOLATION [warning] agent-has-description
  agent: backend-dev
  field "description" must be present and non-empty

validation passed
```

### Run with error violations

Error-severity violations block the write. `xcaffold apply` exits with code 1 and the output directory is not modified.

```
$ xcaffold apply
POLICY VIOLATION [error] require-approved-model
  agent: backend-dev
  field "model" value "gpt-4o" is not in approved list [claude-opus-4-5 claude-sonnet-4-5 claude-haiku-3-5]

[my-project] apply blocked: 1 policy error(s) found
```

```
$ xcaffold validate
syntax and cross-references: ok
POLICY VIOLATION [error] path-safety
  file: .claude/agents/../../../etc/passwd
  output path contains forbidden string ".."

validation failed: policy violations found
```

The violation format depends on the target type. Agent and skill violations print the resource name. Output violations print the file path.

---

## Overriding Built-in Policies

xcaffold ships with four built-in policies embedded in the binary:

| Name | Severity | Target | Purpose |
|---|---|---|---|
| `path-safety` | `error` | `output` | Blocks compiled output paths containing `..` traversal sequences |
| `settings-schema` | `error` | `settings` | Rejects settings output containing `"permissions": null` |
| `no-empty-skills` | `warning` | `skill` | Warns when a skill has no `instructions` content |
| `agent-has-description` | `warning` | `agent` | Warns when an agent omits the `description` field |

To disable a built-in policy, create a `kind: policy` document with the same `name` and set `severity: off`, then reference it in your `kind: project` manifest's `policies:` list. The engine resolves overrides by name: user-defined policies replace any built-in with the same name.

**Example: disable path-safety during a migration.**

Create `allow-traversal.xcf`:

```yaml
kind: policy
name: path-safety
description: Temporarily disable path safety during repo migration
severity: off
target: output
```

When `severity` is `off`, the engine skips evaluation entirely. Only `kind`, `name`, `severity`, and `target` are required in the override file.

**Example: downgrade a built-in error to a warning.**

```yaml
kind: policy
name: settings-schema
description: Downgrade settings-schema to warning during initial setup
severity: warning
target: settings
deny:
  - content_contains: ["\"permissions\": null"]
```

This replaces the built-in `settings-schema` policy. Because the full policy definition is replaced (not merged), you must re-declare the `deny` rules if you want the same checks to run at the new severity level.

---

## Using Match Conditions to Filter Resources

The `match` block restricts which resources a policy applies to. All conditions within a single `match` block are AND-ed. An empty or omitted `match` block means the policy applies to all resources of the given `target` type.

**Example: only check agents that have the Bash tool.**

```yaml
kind: policy
name: bash-agents-need-hooks
description: Agents with Bash tool access must have a description of at least 50 characters
severity: warning
target: agent
match:
  has_tool: Bash
require:
  - field: description
    min_length: 50
```

**Example: match agents by name glob.**

```yaml
kind: policy
name: deployer-model-restriction
description: Deployer agents must use opus
severity: error
target: agent
match:
  name_matches: "deploy*"
require:
  - field: model
    one_of:
      - claude-opus-4-5
```

**Match condition fields:**

| Field | Type | Behavior |
|---|---|---|
| `has_tool` | `string` | Matches agents whose `tools` list contains this value |
| `has_field` | `string` | Matches resources where the named field is present and non-empty |
| `name_matches` | `string` | Glob pattern (`filepath.Match` syntax) tested against the resource ID |
| `target_includes` | `string` | Matches resources whose target configuration includes this key |

All conditions are optional. When multiple conditions are set, a resource must satisfy every condition to be evaluated by the policy.

---

## Denying Content Patterns in Compiled Output

The `deny` block checks compiled output files for forbidden content or path patterns. Deny rules are evaluated against every file in the compiled output map. Each rule can use one or more of three check types.

**Example: block leaked TODO markers.**

```yaml
kind: policy
name: no-leaked-todos
description: Compiled agent files must not contain TODO or FIXME markers
severity: error
target: output
deny:
  - content_contains:
      - "TODO"
      - "FIXME"
      - "HACK"
```

`content_contains` is case-insensitive. A match against any file in the compiled output triggers a violation.

**Example: block API key patterns.**

```yaml
kind: policy
name: no-api-keys
description: Compiled output must not contain API key patterns
severity: error
target: output
deny:
  - content_matches: "sk-[a-zA-Z0-9]{20,}"
  - content_matches: "AKIA[0-9A-Z]{16}"
```

`content_matches` accepts a Go regular expression. Each `deny` entry is evaluated independently.

**Example: block path traversal in output file paths.**

```yaml
kind: policy
name: path-traversal-guard
description: Output paths must not contain directory traversal
severity: error
target: output
deny:
  - path_contains: ".."
```

This is the same check performed by the built-in `path-safety` policy.

**Deny rule fields:**

| Field | Type | Behavior |
|---|---|---|
| `content_contains` | `[]string` | Case-insensitive substring match against file content |
| `content_matches` | `string` | Go regex pattern tested against file content |
| `path_contains` | `string` | Substring match against the compiled output file path |

A single `deny` entry can combine `content_contains`, `content_matches`, and `path_contains`. Each check runs independently — any match produces a separate violation.

---

## Combining Require and Deny Rules

A single policy can declare both `require` and `deny` blocks. The engine evaluates both independently against all matched resources.

```yaml
kind: policy
name: strict-agent-standards
description: Enforce description length and block shell references
severity: error
target: agent
match:
  has_tool: Bash
require:
  - field: description
    is_present: true
  - field: description
    min_length: 20
deny:
  - content_contains:
      - "rm -rf"
      - "sudo"
```

**Require rule fields:**

| Field | Type | Behavior |
|---|---|---|
| `field` | `string` | Name of the resource field to check |
| `is_present` | `*bool` | When `true`, the field must exist and be non-empty |
| `min_length` | `*int` | Minimum character count for the field value |
| `max_count` | `*int` | Maximum item count for list-type fields (e.g., `tools`) |
| `one_of` | `[]string` | The field value must be one of the listed strings |

Multiple `require` entries are evaluated independently. A violation is emitted for each failing check.
