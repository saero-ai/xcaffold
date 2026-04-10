---
title: "Drift Detection and State"
description: "How SHA-256 lock manifests provide deterministic state synchronization"
---

# Drift Detection and State

xcaffold's compilation model is deterministic: the same `.xcf` source always produces the same output. But determinism alone is not enough. After files are written to disk, they can change — through manual edits, tooling side-effects, or external agents operating on the output directory. The lock manifest system exists to detect and respond to these changes without relying on version control, file timestamps, or any other environmental assumption.

---

## Why a Lock File Instead of Git

Git is a natural instinct for tracking file state, but it is unsuitable as xcaffold's primary state mechanism for several reasons.

First, target output directories vary in their relationship to git. The `.claude/` directory is often committed to a project repository, while `.cursor/` or `.agents/` directories may be gitignored, and globally-scoped outputs live in `~/.xcaffold/` — entirely outside any repository. A mechanism that depends on git would produce inconsistent behavior depending on the user's configuration.

Second, git tracks *intent* (staged changes, commits) rather than *physical content*. xcaffold's drift detection needs to answer a narrower question: does the file on disk match what xcaffold last wrote? A SHA-256 hash of file content answers this exactly, without ceremony.

The lock manifest — serialized as `scaffold.<target>.lock` — is xcaffold's platform-agnostic record of what was generated and what source files drove that generation. It is written by `state.Write()` (`internal/state/state.go:127`) and read by `state.Read()` (`internal/state/state.go:143`). Its schema is versioned independently of xcaffold itself via the `version` field in `LockManifest` (`internal/state/state.go:42-52`), currently at `lockFileVersion = 2` (`internal/state/state.go:16`).

---

## Per-Target Lock Files

Each compilation target maintains its own independent lock manifest. The path for a given target is computed by `LockFilePath()` (`internal/state/state.go:22-29`):

- For the `claude` target (or an empty target string), the function returns `basePath` unchanged — `scaffold.lock` — preserving backward compatibility with existing projects.
- For all other targets, the target name is inserted before the file extension: `scaffold.cursor.lock`, `scaffold.antigravity.lock`, and so on.

This design means drift in one target is entirely isolated from others. A manual edit to a file in `.cursor/` has no effect on the `scaffold.claude.lock` manifest, and vice versa. Running `xcaffold diff --target cursor` checks only cursor artifacts; running it without a flag checks only claude artifacts. The two manifests are independent truth states.

The `LockManifest` struct records the `Target` and `Scope` fields (`internal/state/state.go:47-48`) so that each manifest is self-describing — a manifest read in isolation carries enough metadata to identify which target and scope it was generated for.

---

## Source File Tracking

The lock manifest records not only the *output* artifacts xcaffold generated, but also the *input* source files that drove compilation. Each entry in `SourceFiles` (`internal/state/state.go:36-39`, `49`) stores a relative path and a SHA-256 hash of the `.xcf` file's content at apply time.

`SourcesChanged()` (`internal/state/state.go:162-197`) compares these stored hashes against the current files on disk. It returns `true` — indicating that recompilation is necessary — in any of these conditions:

- The previous manifest has no source file entries (first run, or a manifest written before source tracking was introduced).
- The count of source files has changed (a file was added or removed).
- Any source file's current hash differs from the stored hash.

When sources are unchanged, `applyScope()` in `apply.go` exits early rather than recompiling. This is the smart skip mechanism: it avoids redundant compilation when the source state has not changed. The `--force` flag bypasses this check, always proceeding through compilation regardless of source hashes.

This dual tracking — inputs and outputs — means the system can distinguish between two kinds of divergence: source drift (the declaration changed) and output drift (the generated files changed). `xcaffold diff` surfaces both, as seen in `diffScope()` (`cmd/xcaffold/diff.go:77-153`), which independently checks `Artifacts` and `SourceFiles` against disk.

---

## Orphan Cleanup as Declarative State

The lock manifest enables a property that distinguishes declarative systems from imperative ones: the ability to *remove* what is no longer declared.

When xcaffold compiles a new version of `scaffold.xcf`, it produces a new set of output files. Some files that existed in the previous compilation may no longer appear — an agent was removed, a skill was renamed, a rule was deleted. Without explicit cleanup, these stale files would persist on disk indefinitely, creating a gap between what is declared and what exists.

`FindOrphans()` (`internal/state/state.go:224-237`) closes this gap. It takes the old `LockManifest` and the new compilation's file map as inputs, and returns the set difference: artifact paths recorded in the old manifest that do not appear in the new output. Results are sorted alphabetically for deterministic ordering.

`cleanOrphans()` in `apply.go` (line 474) acts on these results. During a real apply, each orphan path is resolved to an absolute path and removed from disk. After deletion, `cleanEmptyDirsUpToTarget()` removes any parent directories that became empty as a result, keeping the output directory clean without removing the output root itself.

The consequence is that the topology of the output directory at any point in time reflects exactly what is currently declared in the source `.xcf` files — nothing more, nothing less. Files are not accumulated; they are reconciled.

---

## Global vs. Project Scope

xcaffold supports two compilation scopes: project and global. These are not merely organizational labels — they correspond to distinct filesystem locations with independent lock manifests and independent drift states.

Project-scope lock files live alongside `scaffold.xcf` in the project directory. Global-scope lock files live in `~/.xcaffold/`, the global xcaffold home directory. The `Scope` field in `GenerateOpts` (`internal/state/state.go:62`) carries this distinction into the manifest itself.

Because the manifests are independent, a drift condition in the global scope does not affect project-scope checks, and a project applying cleanly does not reset the global manifest. Each scope is a self-contained truth state. `runApply()` in `apply.go` dispatches to `applyScope()` with the correct `lockFile` path depending on whether `--global` is set, and `runDiff()` in `diff.go` similarly computes the correct `targetLock` path per scope before calling `diffScope()`.

---

## Legacy Lock Migration

xcaffold originally wrote a single `scaffold.lock` regardless of target. When per-target lock files were introduced, projects using the old format required a migration path that preserved their drift state without requiring a full recompilation.

`MigrateLegacyLock()` (`internal/state/state.go:202-220`) handles this transparently. When `applyScope()` runs, it calls `MigrateLegacyLock()` before reading the lock. If a `scaffold.lock` exists at the legacy path and the target-specific path does not yet exist, the legacy file is renamed — not copied — to the target-specific path (e.g., `scaffold.cursor.lock`). The rename is atomic on supported filesystems. If the target-specific file already exists, migration is skipped entirely; the existing state is authoritative.

The original `scaffold.lock` file is not deleted by the migration. It persists until the user removes it. This means projects transitioning between xcaffold versions encounter no silent data loss: the legacy manifest remains available for inspection, and the renamed target-specific manifest takes over as the active truth state for future applies.
