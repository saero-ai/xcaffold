---
title: "Splitting a Project Into Multiple .xcf Files"
description: "Break a monolithic project.xcf into domain-scoped files with automatic merge and duplicate detection"
---

# Splitting a Project Into Multiple .xcf Files

xcaffold uses one file per resource: a `project.xcf` manifest plus individual `.xcf` files for agents, skills, rules, and other resources under `xcf/`. As projects grow, spreading resources across domain subdirectories keeps each file focused and easy to review and diff. The parser scans all `.xcf` files recursively, merges them, and validates the result as a single configuration.

The recommended split layout uses `project.xcf` (`kind: project`) at the project root and individual resource `.xcf` files under an `xcf/` subdirectory. All xcaffold commands should be run from the directory containing `project.xcf`.

**When to use this:** When a monolithic `project.xcf` exceeds a few hundred lines, or when different team members own distinct resource domains and need isolated files for review and diffing.

**Prerequisites:**
- Completed [Getting Started](../tutorials/getting-started.md) tutorial
- An existing `project.xcf` in your project

This how-to covers when and how to split, what merge rules apply per resource type, how duplicate IDs are caught, and what the compiled output looks like for each target.

---


## How the scanner works

`FindXCFFiles` performs a recursive `filepath.WalkDir` over the project directory. Every file ending in `.xcf` is included, sorted alphabetically. Hidden directories (names beginning with `.`) and `node_modules` are skipped entirely. There is no depth limit. For more detail on the compilation pipeline, see [Architecture Overview](../concepts/architecture.md).

**Naming conventions:**

- `project.xcf` is the recommended filename for the project manifest (`kind: project`). The parser identifies it by `kind: project`, not by filename — any name works, but `project.xcf` is the convention.
- Resource files under `xcf/` can use any name. Convention: `xcf/agents/developer.xcf`, `xcf/rules/code-style.xcf`.
- `xcaffold init` natively bootstraps a full `xcf/` multi-file layout by default. You do not need to manually migrate if you start with `init`.
- All xcaffold commands (`apply`, `diff`, `validate`, `graph`) run from the directory containing `project.xcf`.

---

## When to use each approach

| Project size | Recommended layout |
|---|---|
| 1-3 agents, few rules | `project.xcf` + flat `xcf/agents/` directory |
| 4+ agents, shared libraries | `project.xcf` + `xcf/agents/`, `xcf/skills/`, `xcf/rules/` subdirectories |
| Team-owned resources | Domain subdirectories under `xcf/` — each owner edits their own `.xcf` files |

Each layout produces identical compiled output. The choice is purely organizational.

---

## Minimal layout

A minimal project uses two files — the project manifest and one file per resource:

`project.xcf`:

```yaml
kind: project
version: "1.0"
name: my-project
targets:
  - claude
```

`xcf/agents/developer.xcf`:

```
---
kind: agent
version: "1.0"
name: developer
description: "General development agent"
model: "claude-sonnet-4-6"
tools: [Bash, Read, Write, Edit, Glob, Grep]
---
You write clean, maintainable code.
```

`xcf/rules/code-style.xcf`:

```yaml
kind: rule
version: "1.0"
name: code-style
always-apply: true
instructions: "Use 2-space indentation. No semicolons in TypeScript."
```

---

## Standard layout

The standard layout separates resources by type under `xcf/`:

```
my-project/
  project.xcf              # kind: project
  xcf/
    agents/
      developer.xcf          # kind: agent
    rules/
      code-style.xcf         # kind: rule
```

**`project.xcf`** — the project manifest:

```yaml
kind: project
version: "1.0"
name: my-project
targets:
  - claude
```

Resources are discovered automatically. xcaffold scans `xcf/` recursively for all `*.xcf` files and merges them into a single configuration. You do not list resources in `project.xcf`.

**`xcf/agents/developer.xcf`:**

```
---
kind: agent
version: "1.0"
name: developer
description: "General development agent"
model: "claude-sonnet-4-6"
tools: [Bash, Read, Write, Edit, Glob, Grep]
---
You write clean, maintainable code.
```

**`xcf/rules/code-style.xcf`:**

```yaml
kind: rule
version: "1.0"
name: code-style
always-apply: true
instructions: "Use 2-space indentation. No semicolons in TypeScript."
```

---

## Directory conventions

| Path | Contents |
|---|---|
| `project.xcf` | `kind: project` manifest — always at project root |
| `xcf/agents/*.xcf` | `kind: agent` documents |
| `xcf/skills/*.xcf` | `kind: skill` documents |
| `xcf/rules/*.xcf` | `kind: rule` documents |
| `xcf/workflows/*.xcf` | `kind: workflow` documents |
| `xcf/mcp/*.xcf` | `kind: mcp` documents |
| `xcf/hooks.xcf` | `kind: hooks` (singleton — one file) |
| `xcf/settings.xcf` | `kind: settings` (singleton — one file) |

The `xcf/` prefix is a convention, not a requirement. The parser scans recursively from the project root. You could use `configs/agents/dev.xcf` and it would work identically. The `xcf/` convention exists because `xcaffold import` generates it by default.

---
## Cross-file references

References between resources in different files resolve correctly. An agent defined in `agents.xcf` can reference a skill defined in `skills.xcf`. The merge step combines all resources into a single unified AST before the compiler runs; by compilation time there is no distinction between resources that came from different files.

---

## Merge rules by resource type

| Resource | Merge behavior |
|----------|---------------|
| `agents` | Additive. Duplicate ID = hard error. |
| `skills` | Additive. Duplicate ID = hard error. |
| `rules` | Additive. Duplicate ID = hard error. |
| `mcp` | Additive. Duplicate ID = hard error. |
| `workflows` | Additive. Duplicate ID = hard error. |
| `hooks` | Additive per event. Handlers from all files are merged, not replaced. |
| `version` | First file that declares it wins. Conflicting values = hard error. |
| `project` | First file that declares `project.name` wins. Conflicting values = hard error. |
| `extends` | First file that declares it wins. Conflicting values = hard error. |
| `settings` | Last file wins (full struct overwrite — not field-by-field merge). |
| `test` | Last file that declares a non-empty block wins. |

The alphabetical sort order of file names determines "first" and "last" for `version`, `project`, and `settings`. Keep `settings` in a single file to avoid unexpected overwrite behavior.

---

## Detecting duplicates

Declaring the same ID in two files is a hard parse error. The error names both files:

```
duplicate agent ID "frontend-dev" found in agents.xcf and developer.xcf
```

The same error format applies to skills, rules, MCP servers, and workflows, with the corresponding kind name in place of `agent`.

Resolve the conflict by removing the duplicate or renaming one of the IDs before running any xcaffold command.

---

## Path resolution and path safety

`instructions-file` and `references` paths resolve **relative to the root scan directory** — the directory passed to `ParseDirectory()`, which is determined once by `apply.go` as `filepath.Dir(configPath)`. All `.xcf` files in the scan share this same base directory, regardless of which file declares a given resource.

> **Path safety:** `..` traversal is rejected. A path like `../../shared/instructions.md` will fail at compile time with a path traversal error. All paths must resolve within or below the root scan directory.

Given this layout:

```
myproject/
  skills.xcf
  skills/
    component-builder.md
```

The declaration in `skills.xcf`:

```yaml
skills:
  component-builder:
    instructions-file: skills/component-builder.md
```

resolves to `myproject/skills/component-builder.md` — the root scan directory (`myproject/`), which is the same base used by every `.xcf` file in the project.

Because `FindXCFFiles` scans recursively, `.xcf` files in subdirectories are included in the merge. However, their `instructions-file` paths still resolve relative to the root scan directory, not relative to the subdirectory. Keep this in mind when reorganizing files.

---

## Tracking changes with `xcaffold diff`

The state file (`.xcaffold/project.xcf.state`) stores a `source_files` manifest: a list of every `.xcf` file path and its SHA-256 hash at the time of the last `apply`.

When you add, remove, or modify `.xcf` files between runs, `xcaffold diff` reports the change alongside artifact drift:

```
  [default] SRC ADDED   mcp.xcf
  [default] SRC DELETED rules-legacy.xcf
  [default] SRC DRIFTED agents.xcf
    expected: sha256:3a7f...
    actual:   sha256:9c2b...
```

`SRC ADDED` — a `.xcf` file exists on disk that was not in the previous state manifest.  
`SRC DELETED` — a `.xcf` file recorded in the state manifest no longer exists on disk.  
`SRC DRIFTED` — a `.xcf` file exists in both but its content hash has changed.

Any of these conditions causes `xcaffold apply` to recompile. Pass `--force` to recompile unconditionally regardless of source hash state.

---

## Dual-target output

The same multi-file project compiles to different output directories depending on `--target`.

**`xcaffold apply --target claude`**

Output directory: `.claude/`

```
.claude/
  agents/
    frontend-dev.md
    backend-dev.md
  rules/
    code-style.md
  skills/
    component-builder.md
    api-design.md
  settings.json
```

Agent files are Markdown with YAML frontmatter. Rule files are Markdown. `settings.json` contains the merged settings block and MCP server definitions.

**`xcaffold apply --target cursor`**

Output directory: `.cursor/`

```
.cursor/
  agents/
    frontend-dev.md
    backend-dev.md
  rules/
    code-style.mdc
  mcp.json
```

Rule files use the `.mdc` extension under the Cursor target. MCP servers are written to `mcp.json` rather than embedded in `settings.json`.

**`xcaffold apply --target antigravity`**

Output directory: `.agents/`

```
.agents/
  rules/
    code-style.md
    project-instructions.md
  skills/
    component-builder/
      SKILL.md
    api-design/
      SKILL.md
  workflows/
    deploy.md
  mcp_config.json
```

Antigravity supports rules, skills, workflows, and MCP servers. Individual rules are written as plain Markdown files (no YAML frontmatter) under `rules/`; project-level instructions are also written to `rules/project-instructions.md` with scope provenance markers. Skills are written with minimal YAML frontmatter (name and description only) in `skills/{id}/SKILL.md`. Workflows output to `workflows/{id}.md`. MCP servers are consolidated into `mcp_config.json`. Agents and hooks are not supported by Antigravity and are silently omitted from the output.

Two additional targets are also supported: `copilot` (output to `.github/`) and `gemini` (output to `.gemini/`). The merge and duplicate-detection logic is identical across all targets; only the output directory structure and file extensions differ.

---

## Applying to a specific directory

When the project directory is not the current working directory, pass `--config`:

```bash
xcaffold apply --target claude --config /path/to/myproject
xcaffold apply --target cursor --config /path/to/myproject
```

`--config` accepts either a directory path (scans all `.xcf` files in it) or a path to a single `.xcf` file (uses that file's parent directory as the scan root).

---

## Verification

After splitting, verify the merged configuration compiles without errors:

```bash
xcaffold validate
```

Expected output when all files merge cleanly:

```
syntax and cross-references: ok
policies: ok

validation passed
```

Then apply and inspect the output to confirm all resources from all files are present:

```bash
xcaffold apply --target claude
ls .claude/agents/
```

All agent files declared across your split `.xcf` files should appear in the output directory.

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `duplicate agent ID "X" found in A.xcf and B.xcf` | Same `name:` declared in two files | Remove one definition or rename the ID in one of the files |
| `instructions-file` not found after moving a resource file | Path resolves relative to root scan dir, not the `.xcf` file's location | Update the `instructions-file:` path to be relative to the root scan directory |
| Settings block from one file overwriting another | Two `kind: settings` documents exist; last alphabetically wins | Consolidate all settings into a single `kind: settings` file |
| New `.xcf` file not picked up by the scanner | File is inside a hidden directory (name starts with `.`) | Move the file to a non-hidden directory or rename the parent |

---

## Related

- [Architecture Overview](../concepts/architecture.md) — how `ParseDirectory` and `FindXCFFiles` work in the compilation pipeline
- [Schema Reference](../reference/schema.md) — full field tables for each `kind:` document type
- [CLI Reference: xcaffold diff](../reference/cli.md#xcaffold-diff)
- [CLI Reference: xcaffold validate](../reference/cli.md#xcaffold-validate)
- [Import Existing Config](import-existing-config.md) — generate split files from an existing provider directory
