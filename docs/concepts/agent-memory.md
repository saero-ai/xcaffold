---
title: "Agent Memory"
description: "Durable, agent-scoped context that persists across sessions in supported providers."
---

# Agent Memory

Memory resources (`kind: memory`) are durable, agent-scoped contexts that persist across sessions. Each memory entry carries a body of markdown text (inline or from an external file) that an agent can load at the start of a session to restore relevant project, user, or feedback context without re-deriving it from scratch.

When you run `xcaffold apply`, memory entries are rendered into each target provider's native persistence mechanism where one exists. For providers that have no first-class cross-session memory format, xcaffold emits a `MEMORY_NO_NATIVE_TARGET` fidelity note and writes no files for that provider. See [Provider Support](#provider-support) for the current status per target.

---

## Example Usage

Place each memory file under `xcf/memory/<agent-id>/`:

```
xcf/
  memory/
    security-reviewer/
      audit_log_owner.xcf
```

`xcf/memory/security-reviewer/audit_log_owner.xcf`:

```yaml
---
kind: memory
version: "1.0"
name: audit_log_owner
description: "Who owns the project audit log."
type: project
lifecycle: seed-once
instructions: |
  The security team owns the audit log. Route all audit-log
  questions to them rather than the feature team.
---
```

Directory placement encodes agent affiliation. A file at `xcf/memory/<agent-id>/<name>.xcf` binds the memory entry to `<agent-id>`. At compile time the parser sets `AgentRef` to the directory segment immediately containing the file. Memory files placed directly under `xcf/memory/` (no agent subdirectory) fall back to the `default` agent ref.

### Inline body vs. `instructions-file`

The markdown body can also be written after the closing `---` delimiter instead of inside the `instructions:` field:

```yaml
---
kind: memory
version: "1.0"
name: audit_log_owner
description: "Who owns the project audit log."
type: project
---
The security team owns the audit log. Route all audit-log
questions to them rather than the feature team.
```

Both forms are equivalent. When `instructions:` is set in the YAML frontmatter, the body after the closing `---` is silently ignored.

---

## Argument Reference

| Field | Required | Type | Description |
|---|---|---|---|
| `kind` | yes | string | Must be `memory`. |
| `version` | yes | string | Schema version. Use `"1.0"`. |
| `name` | yes | string | Unique identifier for this memory entry within the project. Used as the output filename (`<name>.md`). |
| `description` | no | string | Human-readable summary of what this entry records. |
| `type` | no | enum | Semantic category of the memory. One of `user`, `feedback`, `project`, `reference`. Has no effect on compilation today; reserved for agent-side filtering. |
| `lifecycle` | no | enum | Controls seeding behavior. One of `seed-once` (default) or `tracked`. See [Lifecycle](#lifecycle) below. |
| `instructions` | no | string | Inline markdown body. Mutually exclusive with `instructions-file`. |
| `instructions-file` | no | string | Relative path to an external markdown file whose contents become the memory body. Mutually exclusive with `instructions`. |
| `targets` | no | map | Per-target overrides. Keys are provider names (`claude`, `gemini`, etc.). Values follow the `TargetOverride` schema. |

### Lifecycle

- **`seed-once`** (default): xcaffold writes the file on the first `apply` and does not overwrite it on subsequent runs if the file already exists on disk. This preserves any edits an agent has written back to the file since the last seed.
- **`tracked`**: xcaffold overwrites the file on every `apply`, discarding any agent-written edits. Use this for memory entries that must remain exactly as declared in the `.xcf` source.

### `type` values

| Value | Intended use |
|---|---|
| `user` | Context about the user: role, preferences, skill level. |
| `feedback` | Guidance the agent has received: corrections, confirmed approaches. |
| `project` | Ongoing work, goals, decisions, and incidents. |
| `reference` | Pointers to external resources — dashboards, issue trackers, runbooks. |

---

## Attributes Reference

After `xcaffold apply`, each memory entry produces one markdown file:

| Invocation | Output path |
|---|---|
| `xcaffold apply` | `.claude/agent-memory/<agent-id>/project_<name>.md` |
| `xcaffold apply --global` | `~/.claude/agent-memory/<agent-id>/project_<name>.md` |

`<agent-id>` is derived from the `xcf/memory/<agent-id>/` directory that contains the `.xcf` file. When no agent subdirectory is present (file lives directly under `xcf/memory/`), `<agent-id>` falls back to `default`.

Example: `xcf/memory/security-reviewer/audit_log_owner.xcf` compiles to `.claude/agent-memory/security-reviewer/project_audit_log_owner.md`.

---

## Provider Support

| Provider | Status | Behavior |
|---|---|---|
| Claude Code | Supported | Writes `agent-memory/<agent-id>/project_<name>.md` under `.claude/` (project) or `~/.claude/` (global). Files are loaded by Claude Code as persistent agent context. |
| Gemini CLI | Deferred | Emits a `MEMORY_NO_NATIVE_TARGET` fidelity note; no files written. Gemini CLI appends agent-written memories to `GEMINI.md` under a `## Gemini Added Memories` section via its `save_memory` tool. A future xcaffold release will compile `kind: memory` entries to that section. |
| Antigravity | Deferred | Emits a `MEMORY_NO_NATIVE_TARGET` fidelity note; no files written. Antigravity persists knowledge items in a `brain/` directory. A future xcaffold release will compile `kind: memory` entries to that format. |
| Cursor | Not supported | Cursor has no native cross-session memory mechanism. Emits a `MEMORY_NO_NATIVE_TARGET` fidelity note; no files written. |
| GitHub Copilot | Not supported | GitHub Copilot has no native cross-session memory mechanism. Emits a `MEMORY_NO_NATIVE_TARGET` fidelity note; no files written. |

Fidelity notes are printed to stderr during `xcaffold apply` and included in the plan output of `xcaffold plan`.

---

## Migration

Earlier builds of xcaffold exposed `--include-memory` and `--reseed` flags on the `apply` command. Both flags have been removed. Memory compilation is now part of the default `xcaffold apply` pipeline alongside agents, skills, and rules — no flag is needed to include it.

Drift detection is unified with the rest of the compiled output. The `scaffold.lock` state file tracks the SHA-256 of every compiled memory file. If you have manually edited a memory file and want xcaffold to overwrite it with the current `.xcf` source, pass the existing `--force` flag to `xcaffold apply`.
