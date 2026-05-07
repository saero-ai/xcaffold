---
title: "xcaffold init"
description: "Bootstrap the agentic environment and set up the xcaffold compiler."
---

# xcaffold init

Scaffolds the environment to prepare it for Agent-as-Code compilation.

The `init` command bootstraps a repository with a `project.xcaf` file and the `xcaf/` source directory. If it detects existing provider directories (e.g., `.claude/` or `.cursor/`), it prompts you to run `xcaffold import` instead, enabling a smooth migration path into the xcaffold ecosystem.

## Usage

```bash
xcaffold init [flags]
```

## Options

| Flag | Default | Description |
|---|---|---|
| `-y, --yes` | `false` | Accept all defaults non-interactively. Useful for CI/CD. |
| `--target <strings>` | `""` | One or more compilation target providers (e.g., `claude`, `cursor`). Repeat or comma-separate for multiple targets. |
| `--json` | `false` | Output machine-readable JSON manifest instead of interactive logs. |

## Behavior

### The Bootstrap Phase
1. **Detection:** Checks for existing provider directories (`.claude/`, `.cursor/`, `.agents/`) and an existing `project.xcaf`. Offers to run `xcaffold import` when provider directories are found.
2. **Target Selection:** Presents an interactive multi-select prompt to choose one or more target providers. Skipped when `--target` is set.
3. **Synthesis:** Writes `project.xcaf` and the `xcaf/` source directory, including the Xaff authoring agent, the `xcaffold` skill, the `xcaf-conventions` rule, and provider override files for each selected target.
4. **Registration:** Registers the project in the local xcaffold registry.

## Output

When existing provider directories are detected, `init` displays a compiled output table summarizing the resources found:

```
  ┌─── COMPILED OUTPUT ─────────────┐
  Kind               .claude/  .agents/   .gemini/
  ──────────────────────────────────────────────────
  Agents                   17        17          0
  Skills                   21        21          3
  Rules                    13        13          5
  Workflows                 0         8          0
  MCP                       1         0          1
```

Kinds are listed as rows; detected providers as columns. Only kinds with at least one resource across any provider are shown. After displaying the table, `init` prompts to run `xcaffold import` to adopt the detected configurations.

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
