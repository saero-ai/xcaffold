---
title: "Translating Configurations Between Providers"
description: "Convert agent configurations directly between providers with fidelity modes, optimizer passes, and workflow lowering"
---

# Translating Configurations Between Providers

You have an existing configuration for one AI coding platform and need the same agent definitions in a different platform's native format. `xcaffold translate` converts agent configurations from one provider format to another in a single command. Unlike `xcaffold import`, which produces an xcaffold project suitable for ongoing management, `translate` is a one-shot cross-provider conversion that writes directly to the target provider's directory format.

**When to use this:** When you want a one-shot conversion between provider formats without adopting xcaffold for ongoing project management. Use `xcaffold import` instead if you want drift detection and re-compilation.

**Prerequisites:** Completed [Getting Started](../tutorials/getting-started.md) tutorial.

---

## When to use `translate` vs `import`

| Goal | Command |
|------|---------|
| Adopt xcaffold to manage an existing `.claude/` or `.cursor/` setup | `xcaffold import` |
| Convert your Antigravity rules to Claude format without creating an xcaffold project | `xcaffold translate` |
| Preview what a cross-provider migration would produce without writing files | `xcaffold translate --dry-run` |
| Inspect the intermediate xcaffold IR before compiling | `xcaffold translate --save-xcf ir.xcf` |

---

## Basic usage

```bash
xcaffold translate --from claude --to cursor
```

xcaffold reads the Claude agent configuration from `.claude/` in the current directory, compiles it to Cursor's format, and writes the output to `.cursor/`.

### With explicit paths

```bash
xcaffold translate \
  --from antigravity \
  --to claude \
  --source-dir ./my-project \
  --output-dir ./claude-output
```

When `--source-dir` points to a project root, `resolveScopeDir` automatically descends into the provider's conventional subdirectory (e.g., `.claude/`, `.cursor/`, `.agents/`, `.github/`).

---

## Supported providers

| Provider | Value |
|----------|-------|
| Claude Code | `claude` |
| Cursor | `cursor` |
| Antigravity | `antigravity` |
| GitHub Copilot | `copilot` |
| Gemini CLI | `gemini` |

---

## Fidelity modes

Fidelity controls how translate handles information that cannot be represented in the target format.

| Mode | Behavior |
|------|----------|
| `warn` (default) | Emit fidelity notes to stderr; continue |
| `strict` | Exit non-zero on any information loss |
| `lossy` | Allow information loss silently |

```bash
xcaffold translate --from claude --to cursor --fidelity strict
```

---

## Previewing the translation

### Dry-run (no files written)

```bash
xcaffold translate --from claude --to cursor --dry-run
```

Lists every file that would be written without writing any of them.

### Show diff

```bash
xcaffold translate --from claude --to cursor --diff
```

Shows a diff between the current on-disk state and the translation result. Three output formats are supported:

```bash
xcaffold translate --from claude --to cursor --diff --diff-format unified    # default
xcaffold translate --from claude --to cursor --diff --diff-format json
xcaffold translate --from claude --to cursor --diff --diff-format markdown
```

Use `--diff-only` to show the diff and skip writing files (equivalent to `--diff --dry-run`).

### Idempotency check

```bash
xcaffold translate --from claude --to cursor --idempotent-check
```

Exits non-zero if any output file would change from its current on-disk state. Useful in CI to verify that committed translated configs are up to date.

---

## Working with the intermediate IR

### Save the IR for inspection

```bash
xcaffold translate --from antigravity --to claude --save-xcf ir.xcf
```

Writes the xcaffold intermediate representation (a `scaffold.xcf` file) to `ir.xcf` before compiling. Useful for debugging translation fidelity or using the IR as input to `xcaffold apply` later.

### Load IR directly (skip import phase)

```bash
xcaffold translate --from antigravity --to claude --xcf ir.xcf
```

Bypasses the import phase and loads the xcaffold IR from the specified file. The `--from` flag is still required to disambiguate the source schema.

---

## Workflow lowering

Workflows are a first-class schema concept in xcaffold but have no native primitive in most providers. `xcaffold translate` lowers workflows using the strategy specified by `targets.<target>.provider.lowering-strategy` in the workflow definition.

| Strategy | Target | Output |
|----------|--------|--------|
| `rule-plus-skill` | `claude`, `cursor` (default) | One rule with provenance marker + one skill per step |
| `prompt-file` | `copilot` | `.github/prompts/<name>.prompt.md` |
| `custom-command` | `gemini` | Lowered to Custom Command (NOT DOCUMENTED: exact storage path) |
| native (via `promote-rules-to-workflows: true`) | `antigravity` | Single `workflows/<name>.md` |

**Example workflow with explicit lowering strategy:**

```yaml
kind: workflow
version: "1.0"
name: code-review

steps:
  - name: analyze
    instructions: |
      Read the changed files and identify code smells.
  - name: suggest
    instructions: |
      Propose concrete fixes for the identified issues.

targets:
  copilot:
    provider:
      lowering-strategy: prompt-file
  gemini:
    provider:
      lowering-strategy: custom-command
```

When translating this workflow to Claude or Cursor without an explicit `lowering-strategy`, xcaffold defaults to `rule-plus-skill`. Provenance markers (`x-xcaffold:` YAML blocks) are embedded in the rule body so that `xcaffold import` can reconstruct the original workflow on round-trip.

---

## Optimization passes

After compilation, translate runs an ordered list of named transformation passes against the compiled file map. Required passes are automatically prepended based on the target; user passes (via `--optimize`) are appended.

### Required passes by target

| Target | Required passes |
|--------|----------------|
| `antigravity`, `copilot` | `flatten-scopes`, `inline-imports` |
| `cursor`, `gemini` | `inline-imports` |
| `claude` | (none) |

### Available passes

| Pass | Description |
|------|-------------|
| `flatten-scopes` | Flattens nested scope blocks into a single flat structure |
| `inline-imports` | Resolves `@`-import directives by inlining referenced content |
| `dedupe` | Removes files whose content is identical to another file (earliest key wins) |
| `extract-common` | Hoists repeated content blocks into a shared file |
| `prune-unused` | Removes files not referenced by any agent or rule |
| `normalize-paths` | Rewrites output paths to conform to target conventions |
| `split-large-rules` | Splits rules files exceeding the budget into two parts |

### Dependency ordering

`extract-common` must precede `inline-imports` — extracted blocks need to exist before imports are inlined. If you specify them in the wrong order, xcaffold automatically reorders them and emits an info note.

```bash
xcaffold translate --from claude --to cursor --optimize dedupe --optimize split-large-rules
```

### Size budgets for `split-large-rules`

`split-large-rules` uses a per-target default budget unless overridden:

| Target | Default budget |
|--------|---------------|
| `claude` | `lines:200` |
| `cursor` | `lines:500` |
| `antigravity` | `bytes:12000` |
| `copilot` | `bytes:4000` |
| `gemini` | none |

---

## Generating an audit report

```bash
xcaffold translate --from claude --to cursor --audit-out audit.json
```

Writes a machine-readable JSON report to `audit.json` containing:
- `from` / `to` provider names
- Fidelity notes (level, code, message, suggestion) per translated resource
- File count written
- Timestamp

---

## Including memory

```bash
xcaffold translate --from antigravity --to claude --include-memory
```

Translates memory entries from the source provider's memory store and seeds them into the target provider's memory directory. Use `--reseed` to force re-seeding of entries marked `lifecycle: seed-once` even if they were already seeded.

---

## Multi-scope translation

```bash
xcaffold translate --from claude --to cursor --scope both
```

Translates both project-scope (`.claude/`) and global-scope (`~/.claude/`) configurations. Use `--scope project` (default) or `--scope global` to restrict to one scope.

---

## After translation

The translated files are written directly to the target provider's directory. No `scaffold.xcf` is created unless you pass `--save-xcf`. xcaffold does not track drift or manage state for translated output — the translated directory is not under xcaffold's lock-based management.

If you want ongoing management (drift detection, re-compilation, provider switching), use `xcaffold import` instead to produce a managed xcaffold project.

---

## Verification

Run a dry-run to confirm xcaffold can read your source configuration and enumerate the files it would write:

```bash
xcaffold translate --from <source> --to <target> --dry-run
```

Expected output lists each file path that would be created or overwritten. Exit code `0` with no errors confirms the source configuration parsed successfully.

For a strict fidelity check before committing translated output to version control:

```bash
xcaffold translate --from <source> --to <target> --fidelity strict
```

Exit code `0` means no information was dropped during translation.

---

## Troubleshooting

| Error | Cause | Fix |
|---|---|---|
| `source directory not found` | `--source-dir` points to a path that does not contain the expected provider subdirectory | Confirm the directory exists and contains `.claude/`, `.cursor/`, or `.agents/` as appropriate |
| Fidelity notes appear with `warn` mode | The target does not support all fields in the source (e.g., security fields dropped for cursor) | Use `--audit-out` to capture all notes, or switch to `--fidelity strict` to treat loss as an error |
| Workflow not present in translated output | Workflow was lowered to a rule+skill pair by default | Use `--save-xcf` and inspect the IR to confirm the workflow was parsed; check the lowering strategy |

---

## Related

- [CLI Reference: xcaffold translate](../reference/cli.md#xcaffold-translate)
- [Schema Reference: WorkflowConfig](../reference/schema.md#workflowconfig)
- [Schema Reference: MemoryConfig](../reference/schema.md#memoryconfig)
- [Import Existing Config](import-existing-config.md)
