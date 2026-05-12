---
title: "xcaffold validate"
description: "Check .xcaf syntax and schema, cross-references, and structural invariants."
---

# `xcaffold validate`

Performs a dry-run validation of your `.xcaf` configuration. It checks for syntax errors, broken references, policy violations, and structural invariants without writing any files to the provider directories.

## Usage

```bash
xcaffold validate [flags]
```

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--target` | | `string` | `""` | Validate field support for a specific provider target. When set, runs field-level fidelity checks and returns a non-zero exit code on any field error. |
| `--var-file` | | `string` | `""` | Load variables from a custom file instead of the default `xcaf/project.vars`. |
| `--global` | `-g` | `bool` | `false` | Operate on the user-wide global configuration. Inherited from the root command. |

> **Note:** The `--blueprint` flag is available but hidden. It restricts validation to the named blueprint only.

> **Note:** Global scope (`--global`) is not yet fully implemented. Running `xcaffold validate --global` will return an error.

## Behavior

### Validation Tiers

1. **Syntax and Schema**: Checks that all `.xcaf` files are valid YAML and adhere to the latest schema version (e.g., `version: "1.0"`).
2. **Cross-References**: Ensures that all resource links (e.g., an agent referencing a skill) resolve to valid resource IDs. Unresolved cross-references are reported as non-fatal warnings and do not cause a non-zero exit code.
3. **Directory Integrity**: Validates that all referenced supporting files (scripts, references, artifacts) exist on the filesystem. Checked separately for skill directories and hook directories.
4. **Structural Checks**: Runs invariant checks. The only structural check is: warning if an agent has the `Bash` tool enabled without a `PreToolUse` hook for command validation.
5. **Policy Evaluation**: Evaluates all project and global policies against the resources. This includes a simulated compilation to check for output-level policy violations. Policies are aggregated and reported as a single count.
6. **Field Validation**: If `--target` is specified, checks for missing required fields or unsupported field types for that provider.

## Examples

**Run a full validation of the project:**

```bash
xcaffold validate
```

**Validate field support for Claude:**

```bash
xcaffold validate --target claude
```

**Load variables from a custom file:**

```bash
xcaffold validate --var-file ./custom.vars
```

## Sample Output

```text
xcaffold-project  ·  validating 14 resources

  ✓  syntax and schema (22 files)
  ✓  skill directories
  ✓  hook directories
  ✓  policies (2 checked)

✓  Validation passed.  14 .xcaf files checked.
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success (all checks passed) |
| `1` | Failure (one or more errors found) |
