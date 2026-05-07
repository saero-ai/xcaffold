---
title: "xcaffold validate"
description: "Check .xcaf syntax, cross-references, structural invariants, and policy compliance."
---

# xcaffold validate

Check .xcaf syntax, cross-references, structural invariants, and policy compliance.

The `validate` command parses every `.xcaf` file in the project, verifies cross-reference integrity, checks structural invariants, and evaluates policy rules — all without modifying any output on disk. Use it as a pre-commit gate or in CI to confirm the project is compilable before running `xcaffold apply`.

**Usage:**

```
xcaffold validate [flags]
```

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--target <provider>` | — | `string` | `""` | Validate field support for a specific provider target (`claude`, `cursor`, `antigravity`, `copilot`, `gemini`). |
| `--blueprint <name>` | — | `string` | `""` | Validate only the named blueprint's resources. Internal use. |
| `--global` | `-g` | `bool` | `false` | Operate on the global config (`~/.xcaffold/global.xcaf`). Not yet available — prints an error and exits. |
| `--no-color` | — | `bool` | `false` | Disable ANSI color and UTF-8 glyphs. Also honoured via the `NO_COLOR` environment variable. |
| `--var-file <path>` | — | `string` | `""` | Load variables from a custom file instead of the default `xcf/project.vars`. |

> `--blueprint` is a hidden flag intended for internal use. It does not appear in `--help` output.

## Behavior

### Default mode

Running `xcaffold validate` without flags prints a breadcrumb header followed by the result of each check phase:

1. **Syntax and cross-references** — parses every `.xcaf` file using strict known-fields mode. Reports YAML errors and unknown keys. Verifies that agent `skills`, `rules`, and `mcp` references resolve to defined resources.
2. **Skill directories** — walks `xcaf/skills/` and validates that each skill subdirectory has the expected structure (presence of a `SKILL.md` body file, no unexpected files).
3. **Structural checks** — runs invariant checks on the parsed config. See [Structural checks](#structural-checks) for the full list.
4. **Policy evaluation** — compiles the project in-memory and evaluates all active policy rules against the compiled output. See [Policy evaluation](#policy-evaluation) for details.

A footer line reports the total warning count and the number of `.xcaf` files checked.

### Field validation (--target)

When `--target <provider>` is specified, `validate` performs an additional compile-time field validation pass after policy evaluation:

- **Unsupported fields** — any resource field not supported by the target provider produces an error and fails validation.
- **Required fields** — any field required by the target provider but absent from a resource produces an error and fails validation.
- **Suppressed resources** — resources with a `target-options.<provider>.suppress-fidelity-warnings: true` override are excluded from the field check.

On success, the output includes a `field validation (<provider>)` line in the check list, and the footer appends a field validation summary: `Field validation: <provider> (0 errors)`. On failure, the specific unsupported or missing fields are reported before the `Validation failed` footer.

This flag is useful as a pre-commit gate to catch provider incompatibilities before running `xcaffold apply`.

### Policy evaluation

Policy rules are evaluated after a successful in-memory compilation pass. Built-in policies cover:

- **path-safety** — flags file paths containing `..` or other unsafe sequences.
- Additional built-in policies shipped with the binary.

You can define project-level policy overrides in `.xcaf` files with `kind: policy`. Policy violations are classified as `error` or `warning` by the rule definition. Errors cause a non-zero exit; warnings are reported but do not fail validation.

### Structural checks

Structural checks emit warnings for the following conditions:

| Check | Condition |
|-------|-----------|
| Orphan skill | A skill is defined but not referenced by any agent. |
| Orphan rule | A rule is defined but not referenced by any agent and has neither `paths` nor `always-apply: true`. |
| Missing instructions | An agent has no body content. |
| Bash without hook | An agent has the `Bash` tool but no `PreToolUse` hook (neither project-level nor agent-level) to validate commands before execution. |

Structural warnings do not cause a non-zero exit on their own. They are counted toward the footer warning total.

## Output labels

| Glyph | Meaning |
|-------|---------|
| `✓` | Phase passed with no issues. |
| `△` | Warning — phase completed but issues were found. |
| `✗` | Error — phase failed. |

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | All checks passed. Warnings may be present. |
| `1` | One or more errors found (parse error, cross-reference error, policy error, or `--global` used). |
| `2` | Internal error — compilation failed during policy evaluation. |

## Sample output

### Clean validation

```
sandbox  ·  last applied 3 days ago

  ✓  syntax and cross-references
  ✓  skill directories
  ✓  structural checks
  ✓  policies (4 checked)

✓  Validation passed.  52 .xcaf files checked.
```

### Validation with structural warnings

```
sandbox  ·  last applied 3 days ago

  ✓  syntax and cross-references
  ✓  skill directories

  structural warnings:
    △  skill "orphan-skill" is defined but not referenced by any agent
    △  rule "unused-rule" is defined but not referenced by any agent and has no paths or always-apply

  ✓  policies (4 checked)

✓  Validation passed with 2 warnings.  52 .xcaf files checked.
```

### Validation with policy warnings

```
sandbox  ·  last applied 3 days ago

  ✓  syntax and cross-references
  ✓  skill directories
  ✓  structural checks

  policy warnings:
    △  [model-pinning] developer: no model field set; will use provider default

  ✓  policies (4 checked, 1 warning)

✓  Validation passed with 1 warning.  52 .xcaf files checked.
```

### Validation with policy errors

```
sandbox  ·  last applied 3 days ago

  ✓  syntax and cross-references
  ✓  skill directories
  ✓  structural checks

  policy errors:
    ✗  [path-safety] ../etc/passwd: forbidden string ".." found in file path

✗  Validation failed: 1 policy error found.
```

### Clean validation with --target

```
sandbox  ·  antigravity  ·  last applied 3 days ago

  ✓  syntax and cross-references
  ✓  skill directories
  ✓  structural checks
  ✓  policies (4 checked)
  ✓  field validation (antigravity)

✓  Validation passed.  52 .xcaf files checked.  Field validation: antigravity (0 errors).
```

### Validation with --target field errors

```
sandbox  ·  antigravity  ·  last applied 3 days ago

  ✓  syntax and cross-references
  ✓  skill directories
  ✓  structural checks
ERROR (antigravity): field "effort" is unsupported by antigravity; use a agent.antigravity.xcaf override or remove from base manifest

✗  Validation failed: compilation failed with 1 error(s):
  FIELD_UNSUPPORTED agent/dev: field "effort" is unsupported by antigravity; ...
```

### Parse error

```
sandbox  ·  last applied 3 days ago

  ✗  syntax and cross-references

✗  Validation failed: xcaf/agents/developer.xcaf:12: unknown field "allowedTools"
```

## Examples

**Validate the project in the current directory:**
```bash
xcaffold validate
```

**Validate field support for a specific provider:**
```bash
xcaffold validate --target antigravity
xcaffold validate --target cursor
```

**Validate without color output (useful in CI):**
```bash
xcaffold validate --no-color
# or
NO_COLOR=1 xcaffold validate
```

**Validate a project in a specific directory:**
```bash
xcaffold validate --config /path/to/project
```

## Notes

- `--global` is accepted as a flag but prints `Global scope is not yet available` and exits `1`. Global validation will be supported in a future release.
- Policy rules are evaluated against an in-memory compilation result. If compilation itself fails, the policy phase is skipped and reported as `policies (skipped: compilation error)`.
- For guidance on organizing and authoring policy resources, see [Policy Best Practices](../../best-practices/policy-organization.md).
