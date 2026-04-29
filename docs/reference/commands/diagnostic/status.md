---
title: "xcaffold status"
description: "Show compilation state and check for drift across all targets."
---

# xcaffold status

Provides a high-level summary of sync metrics across all compiled targets with inline file status reporting.

The `status` command evaluates the SHA-256 state manifest (`.xcaffold/<blueprint>.xcf.state`) generated during compilation and compares it against both your source `.xcf` files and the generated provider-native files.

This command completely replaces the legacy `xcaffold diff` command conceptually (which will act as a deprecated alias to `status`).

## Usage

```bash
xcaffold status [flags]
```

## Options

| Flag | Default | Description |
|---|---|---|
| `--all` | `false` | Show all files when scoping to a target (default: drifted only). |
| `--blueprint <name>` | `""` | Show status filtered to the named blueprint's state file. |
| `--target <target>` | `""` | Focus drift inspection on a single compilation target (e.g., `claude`, `cursor`, `gemini`). |
| `--global` | `false` | Show status for the global config state (`~/.xcaffold/global.xcf.state`). |

## Behavior

### Target Overview Mode (Default)
Running `xcaffold status` without `--target` displays an overview of all applied targets. 
It lists the last apply time, active blueprint, total artifact count, and high-level sync status for each target. Any drifted (modified, deleted, or unsynced) files are listed directly below the summary table as an inline diff queue.

### Single Target Mode
When you provide a `--target` (e.g., `xcaffold status --target claude`), the command zooms in on the state file for that specific platform. It exclusively outputs the drift file list for that target. Use `--all` alongside `--target` to list every tracked artifact regardless of its drift status.

### Status Codes

| Status | Meaning |
|---|---|
| `clean` | File hash matches lock. |
| `DRIFTED` | File content has changed since the last `apply`. |
| `MISSING` | File is tracked in lock but does not exist on disk. |
| `SRC DELETED` | Source `.xcf` file tracked in lock no longer exists. |
| `SRC DRIFTED` | Source `.xcf` file content changed. |
| `SRC ADDED` | A new `.xcf` file exists that was not compiled during the last `apply`. |

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | No drift detected across queried targets. |
| `1` | Drift detected (source changed or output modified). |
| `2` | No state file found (you must run `xcaffold apply` first). |

## Examples

**Show overview across all targets:**
```bash
xcaffold status
```

**Check drift for a specific workspace constraint (e.g., Cursor):**
```bash
xcaffold status --target cursor
```

**View all tracked files for a target, even ones that are clean:**
```bash
xcaffold status --target claude --all
```
