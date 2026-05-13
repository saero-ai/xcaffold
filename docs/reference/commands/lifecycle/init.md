---
title: "xcaffold init"
description: "Bootstrap the agentic environment and set up the xcaffold compiler."
---

# xcaffold init

Scaffolds the environment to prepare it for Harness-as-Code compilation.

The `init` command bootstraps a repository with a `project.xcaf` file and the `xcaf/` source directory. If it detects existing provider directories (e.g., `.claude/` or `.cursor/`), it asks whether you want to import them. If you confirm, `init` performs the import directly — it does not redirect you to run a separate command.

## Usage

```bash
xcaffold init [flags]
```

## Options

| Flag | Default | Description |
|---|---|---|
| `-y, --yes` | `false` | Accept all defaults non-interactively. Useful for CI/CD. |
| `--target <strings>` | `[]` | One or more compilation target providers (e.g., `claude`, `cursor`). Repeat or comma-separate for multiple targets. |
| `--upgrade` | `false` | Force-refresh toolkit files (xaff agent, xcaffold skill, xcaf-conventions rule) to the latest embedded versions. |
| `--json` | `false` | Output machine-readable JSON manifest instead of interactive logs. |

## Behavior

### The Bootstrap Phase
1. **Detection:** Checks for existing provider directories (`.claude/`, `.cursor/`, `.agents/`) and an existing `project.xcaf`. When provider directories are found, asks if you want to import them and performs the import directly if you confirm.
2. **Target Selection:** Presents an interactive multi-select prompt to choose one or more target providers. Skipped when `--target` is set.
3. **Synthesis:** Writes `project.xcaf` and the `xcaf/` source directory, including the Xaff authoring agent, the `xcaffold` skill, the `xcaf-conventions` rule, and provider override files for each selected target.
4. **Registration:** Registers the project in the local xcaffold registry.

## Output

When existing provider directories are detected, `init` displays a compiled output table summarizing the resources found:

```
  Kind               .claude  .agents  .gemini
  ─────────────────────────────────────────────
  Agents                  17       17        -
  Skills                  21       21        3
  Rules                   13       13        5
  Workflows                -        8        -
  MCP                      1        -        1
```

Kinds are listed as rows; detected providers as columns. Only kinds with at least one resource across any provider are shown. Unsupported kinds for a given provider are shown as `-`. After displaying the table, `init` asks if you want to import the detected configurations and performs the import directly if you confirm.

## Examples

**Start the interactive initialization wizard:**
```bash
xcaffold init
```

**Initialize non-interactively with a specific target (CI/CD mode):**
```bash
xcaffold init --yes --target claude
```

**Initialize with multiple targets:**
```bash
xcaffold init --target claude --target gemini
```

**Output a machine-readable JSON manifest (non-interactive):**
```bash
xcaffold init --json --target claude
```
