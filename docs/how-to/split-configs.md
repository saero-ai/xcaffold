# Split Configurations

xcaffold supports two configuration layouts: a single `scaffold.xcf` file containing all resources as multi-kind YAML documents, and a split layout with `scaffold.xcf` (`kind: project`) at the root and individual `.xcf` files under `xcf/`.

This guide covers the progression from single-file to split, when to use each, and the conventions that keep split projects manageable.

---

## When to use each approach

| Project size | Recommended layout |
|---|---|
| 1-3 agents, few rules | Single `scaffold.xcf` with multi-kind documents |
| 4+ agents, shared libraries | Split: `scaffold.xcf` + `xcf/` subdirectories |
| Team-owned resources | Split: each owner edits their own `.xcf` files |

There is no functional difference — both produce identical compiled output. The choice is organizational.

---

## Single-file layout

All resources live in one file as `---`-separated YAML documents:

```yaml
kind: project
version: "1.0"
name: my-project
targets:
  - claude

---
kind: agent
version: "1.0"
name: developer
description: "General development agent"
instructions: |
  You write clean, maintainable code.
model: "claude-sonnet-4-6"
tools: [Bash, Read, Write, Edit, Glob, Grep]

---
kind: rule
version: "1.0"
name: code-style
always-apply: true
instructions: "Use 2-space indentation. No semicolons in TypeScript."
```

---

## Split-file layout

The same project split into files:

```
my-project/
  scaffold.xcf              # kind: project
  xcf/
    agents/
      developer.xcf          # kind: agent
    rules/
      code-style.xcf         # kind: rule
```

**`scaffold.xcf`** — the project manifest:

```yaml
kind: project
version: "1.0"
name: my-project
targets:
  - claude
agents:
  - developer
rules:
  - code-style
```

The `agents:` and `rules:` fields are bare name lists — they reference the `name:` field in each child `.xcf` file. These ref lists are informational; the parser discovers files by scanning the directory tree, not by reading the list. However, listing them explicitly documents the project structure.

**`xcf/agents/developer.xcf`:**

```yaml
kind: agent
version: "1.0"
name: developer
description: "General development agent"
instructions: |
  You write clean, maintainable code.
model: "claude-sonnet-4-6"
tools: [Bash, Read, Write, Edit, Glob, Grep]
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
| `scaffold.xcf` | `kind: project` manifest — always at project root |
| `xcf/agents/*.xcf` | `kind: agent` documents |
| `xcf/skills/*.xcf` | `kind: skill` documents |
| `xcf/rules/*.xcf` | `kind: rule` documents |
| `xcf/workflows/*.xcf` | `kind: workflow` documents |
| `xcf/mcp/*.xcf` | `kind: mcp` documents |
| `xcf/hooks.xcf` | `kind: hooks` (singleton — one file) |
| `xcf/settings.xcf` | `kind: settings` (singleton — one file) |

The `xcf/` prefix is a convention, not a requirement. The parser scans recursively from the project root. You could use `configs/agents/dev.xcf` and it would work identically. The `xcf/` convention exists because `xcaffold import` generates it by default.

---

## How import generates split files

`xcaffold import` reads an existing platform directory (`.claude/`, `.cursor/`, `.agents/`) and writes the split layout automatically:

```bash
xcaffold import
```

Output:

```
[project] ✓ Import complete. Created scaffold.xcf with 12 resources.
  Split xcf/ files written to xcf/ directory.
  Run 'xcaffold apply' when ready to assume management.
```

The import command:

1. Reads agent `.md` files, strips YAML frontmatter, and embeds the body as `instructions:`.
2. Reads skill `SKILL.md` files the same way.
3. Reads rule `.md` files the same way.
4. Parses `settings.json` for MCP servers and settings.
5. Parses `hooks.json` for hook definitions.
6. Writes `scaffold.xcf` (`kind: project`) with ref lists.
7. Writes individual `.xcf` files under `xcf/`.

See [Import Existing Config](import-existing-config.md) for the full import workflow.

---

## Where to run xcaffold

All xcaffold commands run from the directory containing `scaffold.xcf`:

```bash
cd my-project/
xcaffold apply --target claude
xcaffold validate
xcaffold graph --full
```

If `scaffold.xcf` is elsewhere, use `--config`:

```bash
xcaffold apply --target claude --config /path/to/my-project
```

The `--config` flag accepts either a directory path (scans all `.xcf` files in it) or a path to a single `.xcf` file (uses that file's parent directory as the scan root).

---

## scaffold.xcf naming

`scaffold.xcf` is the recommended filename for the project manifest. It is what `xcaffold init` generates and what the CLI looks for by default. Users can name it anything — `dev.xcf`, `production.xcf`, etc. — but the idempotency check in `xcaffold init` only looks for `scaffold.xcf`.

Resource files under `xcf/` have no naming requirements. Convention: use the resource `name:` as the filename (`developer.xcf` for `name: developer`).
