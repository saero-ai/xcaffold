# Enforcing Project Policies

Xcaffold ships with a zero-dependency, deterministic **Policy Enforcement Engine**. Unlike linters that run after generation or in CI, the Xcaffold policy engine evaluates your configuration *during* `xcaffold apply` and `xcaffold validate`, executing against the fully-resolved, inherited graph.

This engine enforces architectural constraints (e.g., path boundaries, schema validation, agent metadata) and hard-blocks the creation of `.claude/` artifacts if `SeverityError` violations are found.

This guide covers how the engine operates, how to override built-in policies, and best practices.

---

## 1. Engine Behavior & Built-ins

When you run `xcaffold apply`, the compiler takes an exact snapshot of your configuration graph (after processing `extends:` and `references:`) and runs it against active policies. 

### Current Built-in Policies

| ID | Severity | Purpose |
|----|----------|---------|
| `path-safety` | `error` | Ensures `instructions_file` and `references:` use normalized, non-traversing file paths (blocks `../` and absolute paths outside the workspace). |
| `settings-schema` | `error` | Verifies `custom_instructions` does not exceed 100 lines and valid permissions are defined in agent scopes. |
| `no-empty-skills` | `warning` | Warns if a `skill` defines no tools and no workflows, meaning it provides zero agentic capabilities. |
| `agent-has-description` | `warning` | Warns if an agent omits the `description` field, which lowers precision when the router dispatches tasks. |

> [!NOTE]
> Policies flagged `error` will fail `xcaffold apply` with exit code 1 and block the `.claude/` write. Policies flagged `warning` will print diagnostics but allow the generation to succeed.

---

## 2. Overriding Policies

Xcaffold supports selective, deterministic overrides. You can disable or change the severity of a built-in policy simply by creating a policy with the *exact same ID* in your local project directory.

Policies are ordinary `.xcf` files containing a `kind: policy` block. By default, Xcaffold scans for files ending in `.xcf` to gather your project graph. 

### Disabling a Built-in Policy

If you have a legitimate architectural reason to disable the `path-safety` rule temporarily, create a file named `policy-overrides.xcf` in your project root:

```yaml
kind: policy
name: path-safety
description: Disable path safety explicitly for the legacy migration cleanup
severity: off
target: output
```

Because your custom YAML targets the existing `path-safety` name with `severity: off`, the built-in policy is dynamically suppressed.

> [!NOTE]
> `severity: off` short-circuits evaluation, so only `kind`, `name`, `severity`, and `target` are required for overrides.

---

## 3. Best Practices

### A. Reserve `error` for Security and Pathing

Only mark policies as `error` (fail-closed) if violating them causes downstream compiler crashes, breaks sandbox integrity, or leaks internal resources outside the target repository boundary. 
- **Good Error**: Path traversal blocks.
- **Good Error**: Schema validation blocks.

### B. Use `warning` for Linting and Code Polish

If a configuration is sub-optimal but still mathematically valid to the `claude` compiler, label it `warning`.
- **Warning**: Missing agent descriptions.
- **Warning**: Redundant or unattached skills.

### C. Name-Based Toggling 

Always prefer scoped overrides over disabling global inheritance. If one file needs an exception, override the policy name locally via `kind: policy`, complete the migration, and then remove the override file to automatically snap back to strict constraints.

### D. Centralize Overrides

If you must invoke an override, it's highly recommended to isolate them in `.xcaffold/policies.xcf` or `overrides.xcf` rather than scattering `kind: policy` markers randomly throughout your domain files. 
