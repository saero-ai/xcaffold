---
title: "Importing Existing Configurations"
description: "Adopt xcaffold on an existing project by importing native agent configuration directories into .xcf files"
---

# Importing Existing Configurations

You have an existing `.claude/`, `.cursor/`, or `.agents/` directory and want to bring it under xcaffold management without rewriting all your definitions. `xcaffold import` reads an existing platform configuration directory and generates xcaffold source files from it. This is the fastest way to adopt xcaffold on an existing project without rewriting your agent definitions from scratch.

**When to use this:** When you have an existing native agent configuration directory and want to manage it through xcaffold going forward.

**Prerequisites:**
- Completed [Getting Started](../tutorials/getting-started.md) tutorial
- An existing `.claude/`, `.cursor/`, or `.agents/` directory in your project

---

## What it reads


Import auto-detects platform directories (`.claude/`, `.cursor/`, `.agents/`) and scans each one for resources using the same patterns:

| Source | Pattern (relative to platform dir) |
|---|---|
| Agents | `agents/*.md` |
| Skills | `skills/*/SKILL.md` |
| Rules | `rules/**/*.md`, `rules/**/*.mdc` (recursive) |
| Workflows | `workflows/*.md` |
| Settings | `settings.json` (MCP servers, hooks, plugins, effort level) |
| Hooks | `hooks.json` |

Rules are walked recursively, so files inside subdirectories like `rules/cli/` or `rules/platform/` are included. The subdirectory path becomes part of the rule ID using forward-slash notation. For example, `.claude/rules/cli/build-go-cli.md` is imported with the ID `cli/build-go-cli`.

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

The `instructions-file:` field is cleared during import — the body is always inlined. This makes the `.xcf` file self-contained.

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

**Flags:**

| Flag | Default | Description |
|---|---|---|
| `--with-memory` | `false` | Include any agent-written memory files found in the platform directory in the extracted IR. Memory entries are stored in the `memory:` block of the generated `scaffold.xcf`. |
| `--auto-merge` | `false` | When multiple provider directories are detected, automatically merge without interactive prompts. |

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

---

## Verification

After import, verify the generated configuration is structurally sound:

```bash
xcaffold validate
```

Expected output when the imported config is valid:

```
syntax and cross-references: ok
policies: ok

validation passed
```

Then compile and confirm the output matches your original directory:

```bash
xcaffold apply --target claude
```

Inspect the compiled `.claude/` directory and compare it to your original to confirm no definitions were dropped or mangled during import.

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| Duplicate agent ID error during import | Same agent name exists in `.claude/` and `.cursor/` | Rename one agent before importing, or use `--auto-merge` to keep the version with more content |
| Agent `description` shows "Imported agent" | Source `.md` file had no `description:` in its frontmatter | Edit the generated `.xcf` file and add a `description:` field |
| `xcaffold validate` fails after import | Frontmatter fields in the source file used provider-native keys not recognized by xcaffold | Check the error message for the unknown field name and remove or map it in the generated `.xcf` |
| MCP servers missing from `scaffold.xcf` | Source `settings.json` had no `mcpServers` key | Inspect the source `settings.json` directly and add any missing servers as `kind: mcp` documents manually |
| Rules in `rules/cli/` or `rules/platform/` not imported | Occurs only on xcaffold versions before this fix | Upgrade xcaffold; nested rule IDs use slash notation, e.g. `cli/build-go-cli` |

---

## Related

- [Translating Configurations Between Providers](xcaffold-translate.md) — for one-shot cross-provider conversion without creating an xcaffold project
- [CLI Reference: xcaffold import](../reference/cli.md#xcaffold-import)
- [Splitting a Project Into Multiple .xcf Files](multi-file-projects.md) — the split-file layout that import generates
