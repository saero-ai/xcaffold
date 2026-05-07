---
title: "Drift Remediation"
description: "Detect, diagnose, and restore managed files when compiled output has been modified directly"
---

# Drift Remediation

This tutorial walks through detecting and resolving drift in an xcaffold-managed project. Drift occurs when files inside a compiled output directory (`.claude/`, `.cursor/`, `.agents/`) are modified directly rather than through `project.xcaf`. xcaffold uses SHA-256 hashes to detect these changes precisely and gives you explicit control over remediation. For background on why drift matters, see [Concepts: Drift Detection and State](../concepts/architecture.md#drift-detection-and-state).

**Time to complete:** ~10 minutes

**Prerequisites:**
- `xcaffold` installed and on `$PATH`
- A project directory with `project.xcaf` present
- First apply already run (`xcaffold apply --target claude`)

---

## Step 1 — Establish a clean baseline

```bash
xcaffold apply --target claude
```

This compiles `project.xcaf` into `.claude/` and writes `.xcaffold/project.xcaf.state` with SHA-256 hashes of every output file. The state file is the reference state for all subsequent drift checks.

**Expected output:**
```
  [project] ✓ wrote /path/to/project/.claude/agents/developer.md  (sha256:<hex>)

[project] ✓ Apply complete. .xcaffold/project.xcaf.state updated.
```

Commit `project.xcaf` and `.xcaffold/project.xcaf.state` together.

---

## Step 2 — Simulate a shadow edit

```bash
echo "# MANUAL EDIT" >> .claude/agents/developer.md
```

This simulates a direct edit to a managed file — the kind that happens when someone tweaks an agent definition in place rather than updating `project.xcaf`.

---

## Step 3 — Detect drift

```bash
xcaffold diff --target claude
```

xcaffold recomputes SHA-256 hashes for all artifacts listed in `.xcaffold/project.xcaf.state` and compares them against disk. Any mismatch is reported.

**Expected output:**
```
  [project] DRIFTED  /path/to/project/.claude/agents/developer.md
    expected: sha256:abc123...
    actual:   sha256:def456...

drift detected in 1 file(s) — run 'xcaffold apply --target claude' to restore managed state
```

`xcaffold diff` exits with code `1` when drift is found. This is intentional — it makes the command usable as a CI gate. For the full diff status reference, see [CLI Reference](../../reference/commands/index.md).

---

## Step 4 — Attempt apply (intentional failure)

```bash
xcaffold apply --target claude
```

When a state file exists and drift is detected, `apply` refuses to proceed:

**Expected output:**
```
[project] drift detected! Target directory contains unrecorded changes. Use --force to overwrite
```

This drift guard prevents silent data loss. To proceed you must acknowledge the intent explicitly with `--force`. To preview what would be written without committing, use `--dry-run` instead. For guidance on when `--force` is appropriate, see [Drift Detection and State](../concepts/architecture.md#drift-detection-and-state).

---

## Step 5 — Restore managed state

```bash
xcaffold apply --target claude --force
```

`--force` bypasses the drift guard and overwrites the output directory with freshly compiled content. The manual edit in `developer.md` is discarded. `.xcaffold/project.xcaf.state` is updated with the new hashes.

If you want to preserve the drifted state before overwriting, add `--backup`:

```bash
xcaffold apply --target claude --force --backup
```

**Expected output:**
```
  [project] ✓ wrote /path/to/project/.claude/agents/developer.md  (sha256:<hex>)

[project] ✓ Apply complete. .xcaffold/project.xcaf.state updated.
```

---

## Step 6 — Verify clean state

```bash
xcaffold diff --target claude
```

**Expected output:**
```
  [project] clean    /path/to/project/.claude/agents/developer.md

No drift detected. All managed files are in sync.
```

Exit code `0`. The output directory matches the state file. The project is back under xcaffold's control.

---

## What You Built

You induced drift on a managed file, used `xcaffold diff` to detect it precisely by SHA-256 hash comparison, observed the drift guard block a plain `apply`, and restored the project to a clean state with `--force`. You now have the complete remediation workflow: detect with `diff`, guard on `apply`, restore with `--force`.

---

## Next Steps

- **Getting Started** — if you haven't compiled your first agent yet: [Getting Started](../basics/getting-started.md)
- **Multi-Agent Workspace** — define multiple agents, rules, and skills: [Multi-Agent Workspace](multi-agent-workspace.md)
- **Drift detection concepts** — how the state file, SHA-256 hashes, and source vs. artifact drift work: [Concepts: Drift Detection and State](../../concepts/execution/state-and-drift.md)
- **CLI reference** — full flag reference for `diff`, `apply`, and `--force`: [CLI Reference](../../reference/commands/index.md)
