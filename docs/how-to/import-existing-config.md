# Import Existing Config

`xcaffold import` reads an existing platform configuration directory and generates xcaffold source files from it. This is the fastest way to adopt xcaffold on an existing project without rewriting your agent definitions from scratch.

---

## What it reads

Import auto-detects platform directories (`.claude/`, `.cursor/`, `.agents/`) and scans each one for resources using the same patterns:

| Source | Pattern (relative to platform dir) |
|---|---|
| Agents | `agents/*.md` |
| Skills | `skills/*/SKILL.md` |
| Rules | `rules/*.md`, `rules/*.mdc` |
| Workflows | `workflows/*.md` |
| Settings | `settings.json` (MCP servers, hooks, plugins, effort level) |
| Hooks | `hooks.json` |

For example, `.claude/agents/*.md` and `.claude/skills/*/SKILL.md` are scanned when a `.claude/` directory exists. The same patterns apply to `.cursor/` and `.agents/` if present.

If multiple platform directories are detected, xcaffold merges them into a single configuration. Duplicate resource IDs across directories produce a warning; the version with more content is kept.

---

## What it generates

Import produces a split-file layout:

```
my-project/
  scaffold.xcf              # kind: project — metadata, targets, ref lists
  xcf/
    agents/
      developer.xcf          # kind: agent (one per imported agent)
      reviewer.xcf
    skills/
      tdd.xcf                # kind: skill
    rules/
      code-style.xcf         # kind: rule
      testing.xcf
    hooks.xcf                # kind: hooks (if hooks.json existed)
    settings.xcf             # kind: settings (if settings had non-MCP fields)
    mcp/
      filesystem.xcf         # kind: mcp (one per MCP server)
```

---

## How instructions are inlined

Import parses each `.md` source file for YAML frontmatter. The frontmatter fields (`description`, `model`, `tools`, etc.) become top-level fields in the `.xcf` document. The markdown body after the closing `---` is stripped of leading/trailing whitespace and embedded as the `instructions:` field.

Source agent file (`.claude/agents/developer.md`):

```markdown
---
description: Full-stack developer
model: claude-sonnet-4-6
tools: [Read, Write, Edit, Bash, Glob, Grep]
---

You are a full-stack developer.
Write clean, tested code. Run tests before committing.
```

Generated xcf file (`xcf/agents/developer.xcf`):

```yaml
kind: agent
version: "1.0"
name: developer
description: Full-stack developer
model: claude-sonnet-4-6
tools: [Read, Write, Edit, Bash, Glob, Grep]
instructions: |
  You are a full-stack developer.
  Write clean, tested code. Run tests before committing.
```

The `instructions_file:` field is cleared during import — the body is always inlined. This makes the `.xcf` file self-contained.

---

## Running the import

### Basic import

From a project directory with an existing `.claude/` directory:

```bash
xcaffold import
```

Output:

```
[project] ✓ Import complete. Created scaffold.xcf with 8 resources.
  Split xcf/ files written to xcf/ directory.
  Run 'xcaffold apply' when ready to assume management.
```

### Import via init

`xcaffold init` detects existing platform directories and offers to import them:

```bash
xcaffold init
```

```
⚡ Detected existing agent configuration:

     .claude  — 5 agent(s), 3 skill(s), 7 rule(s)

  xcaffold will import this into a single scaffold.xcf.
Import .claude into scaffold.xcf? [Y/n]
```

### Multi-directory merge

When both `.claude/` and `.cursor/` exist:

```bash
xcaffold init
```

```
⚡ Detected existing agent configurations:

     .claude  — 5 agent(s), 3 skill(s), 7 rule(s)
     .cursor  — 2 agent(s), 1 rule(s)

  xcaffold consolidates multiple configs into one scaffold.xcf.
  This lets you compile to any target and switch providers seamlessly.

Select directories to import:
  [x] .claude — 5 agent(s), 3 skill(s), 7 rule(s)
  [x] .cursor — 2 agent(s), 1 rule(s)
```

Resources from all selected directories are merged. Duplicate agent IDs across directories produce an error.

---

## Auto-detection of targets

Import derives compilation targets from the platform directory names:

| Directory | Target |
|---|---|
| `.claude/` | `claude` |
| `.cursor/` | `cursor` |
| `.agents/` | `antigravity` |

The detected targets are written to the `targets:` field in `scaffold.xcf`:

```yaml
kind: project
version: "1.0"
name: my-project
targets:
  - claude
  - cursor
```

---

## Skill reference files

Skills may include non-markdown reference files (data files, templates) under `.claude/skills/<id>/references/`. Import copies these to `xcf/skills/<id>/references/` and updates the `references:` field in the generated `.xcf` file.

---

## After import

1. Review the generated files. Import uses `"Imported agent"`, `"Imported skill"`, `"Imported rule"` as default descriptions when the source file has no frontmatter `description:` field.
2. Run `xcaffold validate` to check for structural issues.
3. Run `xcaffold apply --target claude` to compile. The first apply after import will regenerate all output files.
4. Commit `scaffold.xcf`, `xcf/`, and the generated lock file.

The original platform directory (`.claude/`, etc.) is not modified or deleted by import. You can keep it until you verify the compiled output matches, then remove it.
