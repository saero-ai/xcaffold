# Drift Remediation

This tutorial walks through detecting and resolving drift in an xcaffold-managed project. Drift occurs when files inside a compiled output directory (`.claude/`, `.cursor/`, `.agents/`) are modified directly rather than through `scaffold.xcf`. xcaffold uses SHA-256 hashes to detect these changes precisely and gives you explicit control over remediation.

**Target reader:** Engineer maintaining an xcaffold-managed project.

**Prerequisites:**
- `xcaffold` installed and on `$PATH`
- A project directory with `scaffold.xcf` present
- First apply already run (`xcaffold apply --target claude`)

---

## Why Drift Matters

xcaffold owns the files it compiles. Everything under `.claude/` is generated output — deterministically derived from `scaffold.xcf`. The lock file `scaffold.claude.lock` records the SHA-256 hash of every source file and every artifact at the moment of last apply.

Direct edits to `.claude/`, `.cursor/`, or `.agents/` are not tracked by xcaffold. The next `xcaffold apply` will overwrite them without warning (unless a prior apply established a lock — in which case the drift guard blocks it, as described below). There is no merge path between manual edits and the compiled output.

`scaffold.xcf` is the only truth. Changes that must persist go there, then through `xcaffold apply`.

---

## Step-by-Step: Inducing and Remediating Drift

### Step 1 — Establish a clean baseline

```bash
xcaffold apply --target claude
```

This compiles `scaffold.xcf` into `.claude/` and writes `scaffold.claude.lock` with SHA-256 hashes of every output file. The lock is the reference state for all subsequent drift checks.

Expected output:
```
  [project] ✓ wrote /path/to/project/.claude/agents/developer.md  (sha256:<hex>)

[project] ✓ Apply complete. scaffold.claude.lock updated.
```

Commit `scaffold.xcf` and `scaffold.claude.lock` together.

### Step 2 — Simulate a shadow edit

```bash
echo "# MANUAL EDIT" >> .claude/agents/developer.md
```

This simulates a direct edit to a managed file — the kind that happens when someone tweaks an agent definition in place rather than updating `scaffold.xcf`.

### Step 3 — Detect drift

```bash
xcaffold diff --target claude
```

xcaffold recomputes SHA-256 hashes for all artifacts listed in `scaffold.claude.lock` and compares them against disk. Any mismatch is reported.

Expected output:
```
  [project] DRIFTED  /path/to/project/.claude/agents/developer.md
    expected: sha256:abc123...
    actual:   sha256:def456...

drift detected in 1 file(s) — run 'xcaffold apply --target claude' to restore managed state
```

`xcaffold diff` exits with code `1` when drift is found. This is intentional — it makes the command usable as a CI gate.

### Step 4 — Attempt apply (intentional failure)

```bash
xcaffold apply --target claude
```

When a lock file exists and drift is detected, `apply` refuses to proceed:

```
[project] drift detected! Target directory contains unrecorded changes. Use --force to overwrite
```

This is the drift guard. It prevents silent data loss — you cannot accidentally overwrite manual changes with a plain `apply`. The block is deliberate. To proceed, you must acknowledge the intent explicitly.

### Step 5 — Restore managed state

```bash
xcaffold apply --target claude --force
```

`--force` bypasses the drift guard and overwrites the output directory with freshly compiled content. The manual edit in `developer.md` is discarded. `scaffold.claude.lock` is updated with the new hashes.

Expected output:
```
  [project] ✓ wrote /path/to/project/.claude/agents/developer.md  (sha256:<hex>)

[project] ✓ Apply complete. scaffold.claude.lock updated.
```

### Step 6 — Verify clean state

```bash
xcaffold diff --target claude
```

Expected output:
```
  [project] clean    /path/to/project/.claude/agents/developer.md

No drift detected. All managed files are in sync.
```

Exit code `0`. The output directory matches the lock. The project is back under xcaffold's control.

---

## Reading Diff Output

`xcaffold diff` reports each artifact with a status code:

| Status | Meaning |
|--------|---------|
| `clean` | File on disk matches the hash recorded in the lock. |
| `DRIFTED` | File exists but its hash has changed since last apply. |
| `MISSING` | File recorded in the lock does not exist on disk. |
| `SRC DRIFTED` | A `.xcf` source file has been modified since last apply. |
| `SRC ADDED` | A new `.xcf` source file exists that was not present at last apply. |
| `SRC DELETED` | A `.xcf` source file recorded at last apply no longer exists. |

**Artifact drift** (`DRIFTED`, `MISSING`) means the compiled output no longer matches the source. Run `xcaffold apply --force` to restore.

**Source drift** (`SRC DRIFTED`, `SRC ADDED`, `SRC DELETED`) means `scaffold.xcf` or a referenced config file has changed since last apply. Run `xcaffold apply` (no `--force` needed) to recompile.

Example with all status types:
```
  [project] clean    /path/to/project/.claude/CLAUDE.md
  [project] DRIFTED  /path/to/project/.claude/agents/developer.md
    expected: sha256:abc123...
    actual:   sha256:def456...
  [project] MISSING  /path/to/project/.claude/agents/reviewer.md
  [project] SRC DRIFTED scaffold.xcf
    expected: sha256:abc123...
    actual:   sha256:def456...
  [project] SRC ADDED   config/rules.xcf
  [project] SRC DELETED config/old.xcf
```

---

## The Apply Safety Guard

`xcaffold apply` checks for artifact drift before writing any files. If `scaffold.claude.lock` exists and any artifact hash has diverged from disk, the command exits with:

```
[project] drift detected! Target directory contains unrecorded changes. Use --force to overwrite
```

This guard fires in two scenarios:

1. Someone edited a file in `.claude/` directly.
2. A tool or process outside xcaffold modified a managed file.

The guard does not fire on a first run (no lock file exists) or when sources have changed and no artifact drift is present.

To bypass: pass `--force`. To preview what would be written without committing: pass `--dry-run`.

---

## When to Use `--force`

Pass `--force` when you have deliberately drifted the output and want xcaffold to take back ownership:

- You modified files in `.claude/` during a refactoring session and are now ready to encode those changes in `scaffold.xcf` and recompile.
- `scaffold.xcf` itself changed and the compiled output is expected to diverge from the previous lock.
- An automated tool modified files in `.claude/` and you want to restore the xcaffold-managed state.

Do not use `--force` as a reflex. The drift guard exists to surface unrecorded changes before they are lost. If you are unsure why drift was detected, run `xcaffold diff --target claude` first to inspect exactly which files diverged.

Manual changes to compiled output are intentionally transient. Any change that must survive a future `xcaffold apply` belongs in `scaffold.xcf`.

---

## Backup Before Force

If you want to preserve the drifted state before overwriting, pass `--backup` alongside `--force`:

```bash
xcaffold apply --target claude --force --backup
```

xcaffold copies the current `.claude/` directory to a timestamped backup before writing:

```
[project] Backing up .claude/ -> .claude_bak_20260411_100000
```

Backup naming convention: `.<target>_bak_<YYYYMMDD_HHMMSS>`. For `--target claude`, the backup is `.claude_bak_<timestamp>`.

The backup is placed in the same parent directory as the output directory by default. If `project.backup-dir` is set in `scaffold.xcf`, backups go there instead:

```yaml
project:
  name: my-project
  backup-dir: .xcf-backups
```

Backups are not managed by xcaffold. They are not recorded in the lock file and will not be cleaned up automatically.

---

## GitOps Workflow: CI Drift Gate

Commit `scaffold.xcf` and `scaffold.claude.lock` together. Use `xcaffold diff` in CI to reject pull requests that contain manual edits to compiled output.

Add `.github/workflows/xcaffold-diff.yml` to your repository:

```yaml
name: xcaffold Drift Check

on:
  pull_request:
    paths:
      - "scaffold.xcf"
      - "scaffold.claude.lock"
      - ".claude/**"

jobs:
  drift-check:
    name: Verify .claude/ matches scaffold.claude.lock
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.24"

      - name: Install xcaffold
        run: go install github.com/saero-ai/xcaffold/cmd/xcaffold@latest

      - name: Run drift check
        run: xcaffold diff --target claude
        # Exits 1 if any .claude/ file has been manually edited
        # or if scaffold.claude.lock is out of date with scaffold.xcf.
```

The workflow triggers only on pull requests that touch `scaffold.xcf`, `scaffold.claude.lock`, or any file under `.claude/`. If `xcaffold diff` exits with a non-zero code, the check fails and the PR is blocked.

**Enforced invariant:** the only way to change `.claude/` in a merged commit is by updating `scaffold.xcf`, running `xcaffold apply --target claude`, and committing both files together.

---

## The Lock File

`scaffold.claude.lock` is a YAML file written by `xcaffold apply`. It is the authoritative record of what xcaffold last compiled. Do not edit it manually.

```yaml
last_applied: "2026-04-11T10:00:00Z"
xcaffold_version: "0.x.x"
claude_schema_version: "alpha"
target: "claude"
scope: "project"
config_directory: "."
source_files:
  - path: scaffold.xcf
    hash: "sha256:abc123..."
artifacts:
  - path: agents/developer.md
    hash: "sha256:def456..."
version: 2
```

Key fields:

| Field | Description |
|-------|-------------|
| `source_files` | SHA-256 hashes of all `.xcf` input files at last apply. Used to detect source drift. |
| `artifacts` | SHA-256 hashes of all compiled output files. Used by `xcaffold diff` to detect artifact drift. |
| `version` | Lock file schema version. Current: `2`. |
| `target` | Compilation target (`claude`, `cursor`, `antigravity`). |

The lock file name encodes the target: `scaffold.<target>.lock`. For the default target, this is `scaffold.claude.lock`.
