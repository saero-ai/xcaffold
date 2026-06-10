---
title: "xcaffold apply"
description: "Compile .xcaf resources into provider-native agent configuration files."
---

# xcaffold apply

Compile .xcaf resources into provider-native agent configuration files.

The `apply` command compiles every `.xcaf` file in the project into provider-native output files (`.claude/`, `.cursor/`, `.gemini/`, etc.). It is a strict one-way generation — manual edits in the output directory are overwritten on the next apply. Use `xcaffold import` to sync manual edits back to `.xcaf` sources before applying.

**Usage:**

```
xcaffold apply [flags]
```

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--backup` | — | `bool` | `false` | Back up the output directory to a timestamped archive before writing. |
| `--blueprint <name>` | — | `string` | `""` | Compile only the named blueprint's resources. |
| `--dry-run` | — | `bool` | `false` | Preview changes without writing to disk. Shows a diff of what would change. |
| `--force` | — | `bool` | `false` | Overwrite output files even when drift is detected. |
| `--global` | `-g` | `bool` | `false` | Compile the global config (`~/.xcaffold/xcaf/global.xcaf`). |
| `--yes` | `-y` | `bool` | `false` | Skip the confirmation prompt. Useful for CI/CD pipelines. |
| `--no-color` | — | `bool` | `false` | Disable ANSI color and UTF-8 glyphs. Also honoured via `NO_COLOR`. |
| `--target <name>` | — | `string` | `""` | Compilation target platform (`antigravity` (deprecated), `antigravity2`, `claude`, `copilot`, `cursor`, `gemini`). When omitted, reads targets from `project.xcaf`. |
| `--output-dir <path>` | — | `string` | `""` | Redirect compiled output to a directory (relative to CWD or absolute). Provider files write to `<path>/.claude/`, root files to `<path>/CLAUDE.md`. State remains at project root. |
| `--var-file <path>` | — | `string` | `""` | Load variables from a custom file instead of the default `xcaf/project.vars`. |

## Behavior

### Compilation sequence

1. **Parsing** — reads all `.xcaf` files, validates syntax, and checks cross-references. Unknown fields cause an immediate error.
2. **Smart skip** — compares source file hashes against the last recorded state. If sources are unchanged, apply exits early with no writes. Use `--force` to skip this check and recompile.
3. **Compilation** — transforms resources into the provider-native format selected by `--target`.
4. **Policy evaluation** — checks compiled output against built-in and any project-defined `kind: policy` rules. Policy errors block the write phase; warnings are printed to stderr and do not block.
5. **Drift detection** — compares the output directory against the recorded state. If manual edits are found, apply lists the affected files and exits without writing. Use `--force` to overwrite, or run `xcaffold import` first to preserve edits.
6. **Write** — writes compiled files to the output directory, purges files from previous compilations that are no longer in scope, and records a new state snapshot.

### Drift detection

When drift is detected, apply lists each affected file with its status (`missing` or `modified`) and exits `1`. Two options are available:

- `xcaffold import` — reads the drifted files and syncs them back to `.xcaf` sources. Run apply again after importing.
- `xcaffold apply --force` — overwrites the output directory, discarding any manual edits.

### Multi-target projects

When `--target` is not provided and the `project.xcaf` declares a `targets:` list, apply compiles for each declared target in sequence. Passing `--target` explicitly limits compilation to that single platform.

### Output directory redirection

By default, `apply` writes to the project root. The `--output-dir` flag redirects all output:

```
xcaffold apply --output-dir=.worktrees/backend/ --blueprint=backend
```

Provider files write to `<output-dir>/.claude/`, root files to `<output-dir>/CLAUDE.md`. The state manifest remains at `<project-root>/.xcaffold/` with the output directory recorded per target. Subsequent `xcaffold status` reads the stored path automatically.

Relative paths resolve from the current working directory. Absolute paths are used as-is. The directory is created if it doesn't exist.

`--output-dir` cannot be used with `--global`.

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Apply succeeded, or sources unchanged (skip). |
| `1` | Error: parse failure, compilation error, policy violation, drift detected, or unknown target. |

## Sample output

### Successful apply

```
sandbox  ·  claude  ·  applied just now

✓  Apply complete. 90 files written to .claude/
  Run 'xcaffold import' to sync manual edits back to .xcaf sources.
```

### Sources unchanged

```
sandbox  ·  claude  ·  applied 2 hours ago

  ✓  Sources unchanged. Nothing to compile.

→ Run 'xcaffold apply --force' to recompile.
```

### Drift detected

```
sandbox  ·  claude  ·  applied 2 hours ago

  ✗  Drift detected in 2 files:

    ✗  missing     CLAUDE.md  (root)
    ✗  modified    agents/reviewer.md

  To preserve manual edits, run 'xcaffold import' first.

→ Run 'xcaffold apply --force' to overwrite.
```

## Examples

**Compile the project in the current directory:**
```bash
xcaffold apply
```

**Compile for a specific target platform:**
```bash
xcaffold apply --target cursor
```

**Preview what would change without writing:**
```bash
xcaffold apply --dry-run
```

**Overwrite output even when drift is detected:**
```bash
xcaffold apply --force
```

**Back up the output directory before writing:**
```bash
xcaffold apply --backup
```

## Notes

- `--global` compiles the user-wide global config at `~/.xcaffold/xcaf/global.xcaf`. State is stored at `~/.xcaffold/state/`.
- When `targets:` is declared in `global.xcaf`, `--global` compiles all listed targets automatically. Pass `--target` to override.
- `--blueprint` and `--global` cannot be combined. Blueprints are project-scoped.
- The state file is written to `.xcaffold/project.xcaf.state` and is machine-local. It should be gitignored (apply adds the entry automatically). When `--output-dir` is specified, the output directory is encoded in the state filename (e.g., `.xcaffold/project@custom-out.xcaf.state`), preventing state collisions between different output locations. See [State Files and Drift Detection](../../../concepts/execution/state-and-drift.md) for schema details.
- Blueprint switching (cross-scope cleanup) only affects blueprints targeting the same output directory. Applying a different blueprint with a different `--output-dir` does not remove the previous blueprint's artifacts.
- Policy rules are evaluated after successful compilation. If compilation fails, the policy phase is skipped.
- For guidance on authoring policy resources, see [Policy Best Practices](../../../best-practices/policy-organization.md).
