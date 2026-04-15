# Splitting a Project Into Multiple .xcf Files

A single `scaffold.xcf` works well for small projects. As projects grow, a monolithic file becomes difficult to maintain — a large agent roster mixed with MCP server definitions and global rules is hard to review and harder to diff. xcaffold supports splitting a project into multiple `.xcf` files; the parser scans, merges, and validates them as a single configuration.

The recommended split layout uses `scaffold.xcf` (`kind: project`) at the project root and individual resource `.xcf` files under an `xcf/` subdirectory. All xcaffold commands should be run from the directory containing `scaffold.xcf`.

This how-to covers when and how to split, what merge rules apply per resource type, how duplicate IDs are caught, and what the compiled output looks like for each target.

---

## When to split

Split when:

- The `scaffold.xcf` file exceeds a few hundred lines and logical domains are clearly distinct.
- Different team members own different resources (agents vs. MCP servers vs. rules) and want isolated files to review.
- You want to conditionally include or exclude a domain file during development without commenting out large YAML blocks.

A single file remains correct when the project is small or the overhead of coordinating multiple files outweighs the benefit.

---

## How the scanner works

`FindXCFFiles` performs a **recursive** `filepath.WalkDir` over the project directory. Every file ending in `.xcf` is included, sorted alphabetically. Hidden directories (names beginning with `.`) and `node_modules` are skipped entirely. There is no depth limit.

`ParseDirectory` calls `FindXCFFiles` internally. It parses each file independently and then merges the results. When the CLI detects a `scaffold.xcf` at the project root, it treats that file's parent directory as the config directory and scans the whole tree — including the `xcf/` subdirectory.

**Naming conventions:**

- `scaffold.xcf` is the recommended filename for the project manifest (`kind: project`). Users can name it anything, but `scaffold.xcf` is what `xcaffold init` generates and what the CLI looks for by default.
- Resource files under `xcf/` can use any name. Convention: `xcf/agents/developer.xcf`, `xcf/rules/code-style.xcf`.
- All xcaffold commands (`apply`, `diff`, `validate`, `graph`) run from the directory containing `scaffold.xcf`.

---

## Splitting by domain

The recommended layout uses `kind: project` at the root and individual `kind:` documents under `xcf/`:

```
myproject/
  scaffold.xcf              # kind: project — metadata, targets, ref lists
  xcf/
    agents/
      frontend-dev.xcf      # kind: agent
      backend-dev.xcf       # kind: agent
    skills/
      component-builder.xcf # kind: skill
      api-design.xcf        # kind: skill
    rules/
      code-style.xcf        # kind: rule
    mcp/
      filesystem.xcf        # kind: mcp
    settings.xcf            # kind: settings
```

`scaffold.xcf` holds the project manifest with ref lists pointing to child resources:

```yaml
kind: project
version: "1.0"
name: myproject
description: "Multi-agent development assistant"
targets:
  - claude
agents:
  - frontend-dev
  - backend-dev
skills:
  - component-builder
  - api-design
rules:
  - code-style
mcp:
  - filesystem
```

Each resource file under `xcf/` is a standalone document with `kind:`, `version:`, and `name:`:

**`xcf/agents/frontend-dev.xcf`:**

```yaml
kind: agent
version: "1.0"
name: frontend-dev
description: "Frontend engineering agent"
model: claude-sonnet-4-5
skills:
  - component-builder
rules:
  - code-style
```

**`xcf/skills/component-builder.xcf`:**

```yaml
kind: skill
version: "1.0"
name: component-builder
description: "Builds React components"
instructions-file: skills/component-builder.md
```

**`xcf/rules/code-style.xcf`:**

```yaml
kind: rule
version: "1.0"
name: code-style
description: "House coding standards"
instructions-file: rules/code-style.md
```

**`xcf/mcp/filesystem.xcf`:**

```yaml
kind: mcp
version: "1.0"
name: filesystem
command: npx
args: ["-y", "@modelcontextprotocol/server-filesystem", "/workspace"]
```

**`xcf/settings.xcf`:**

```yaml
kind: settings
version: "1.0"
model: claude-opus-4-5
```

> **Legacy flat layout:** You can also use a flat layout with all `.xcf` files in the project root (e.g., `agents.xcf`, `rules.xcf`). This still works but the `xcf/` subdirectory layout is preferred for clarity.

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

The lock file (`scaffold.<target>.lock`) stores a `source_files` manifest: a list of every `.xcf` file path and its SHA-256 hash at the time of the last `apply`.

When you add, remove, or modify `.xcf` files between runs, `xcaffold diff` reports the change alongside artifact drift:

```
  [default] SRC ADDED   mcp.xcf
  [default] SRC DELETED rules-legacy.xcf
  [default] SRC DRIFTED agents.xcf
    expected: sha256:3a7f...
    actual:   sha256:9c2b...
```

`SRC ADDED` — a `.xcf` file exists on disk that was not in the previous lock manifest.  
`SRC DELETED` — a `.xcf` file recorded in the lock manifest no longer exists on disk.  
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

Rules use the `.mdc` extension under the cursor target. MCP servers are written to `mcp.json` rather than embedded in `settings.json`.

Two additional targets exist: `antigravity` (output to `.agents/`) and `agentsmd` (output to the project root). The merge and duplicate-detection logic is identical across all targets; only the output directory structure and file extensions differ.

---

## Applying to a specific directory

When the project directory is not the current working directory, pass `--config`:

```bash
xcaffold apply --target claude --config /path/to/myproject
xcaffold apply --target cursor --config /path/to/myproject
```

`--config` accepts either a directory path (scans all `.xcf` files in it) or a path to a single `.xcf` file (uses that file's parent directory as the scan root).
