---
title: "xcaffold validate"
description: "Ensure your definitions are structurally sound and error-free."
---

# xcaffold validate

Executes deep linting checks against your configuration without triggering compilation or drift detection locks. 

The `validate` command acts as your primary quality-gate against YAML syntax issues, missing properties across resources, unresolved foreign references, and structural invariants. If validation fails, compilation halts identically.

## Usage

```bash
xcaffold validate [flags]
```

## Options

| Flag | Default | Description |
|---|---|---|
| `--structural` | `false` | Evaluate deep structural invariants beyond AST validity (e.g., checks for orphaned resources that are defined but never mapped to agents, or missing native instruction pathways). |
| `-g, --global` | `false` | Re-target execution and operate purely against the global configuration manifest instead of the local project scope. |

## Behavior

### Check Levels 
`xcaffold validate` analyzes your codebase through a 6-pass verification pipeline:
1. **YAML Validity & Known Fields:** Rejects typos or malformed structures in the manifest instantly using strict mapping.
2. **Cross-Reference Integrity:** If an `agent` requires a `skill` (or `rule`/`mcp`), the execution graph ensures the target entity was natively defined to eliminate "not found" dead-ends.
3. **File Path Viability:** External files defined through `instructions-file` or nested skill references must provably exist on disk.
4. **Plugin Adherence:** Third-party extensions defined by the `enabledPlugins` map must be recognized by the global registry constraints.
5. **Targets Field Validation:** Warns when resources declare a `targets` field, explaining that target-filtered resources will only be compiled for listed providers.
6. **Conflict File Handling:** Reads `.xcaffold/project.xcf.conflict` if present and displays unresolved import conflicts. Conflicts are created by `xcaffold import` when multi-provider assembly produces ambiguous field values.

### Structural Verification
Running with `--structural` elevates the check beyond referential integrity toward best practice enforcement. It emits non-fatal warnings (or errors depending on your local policy definition) when you have resources defined within `.xcf` files that are _never mathematically utilized or reachable_ by an active agent in your topology.

## Examples

**Standard schema validation for your project configurations:**
```bash
xcaffold validate
```

**Assess your local code for orphaned elements or broken dependencies:**
```bash
xcaffold validate --structural
```

**Audit the integrity of your Global `.xcf` workspace settings:**
```bash
xcaffold validate --global
```
