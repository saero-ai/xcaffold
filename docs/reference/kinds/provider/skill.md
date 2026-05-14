---
title: "kind: skill"
description: "Defines a reusable procedure. Source: xcaf/skills/<name>/skill.xcaf. Compiled to skills/<name>/SKILL.md per provider."
---

# `kind: skill`

Defines a reusable procedure that agents invoke on-demand. Compiled to `skills/<id>/SKILL.md` with YAML frontmatter for all five target providers.

> **Required:** `kind`, `version`, `name`

## Source Directory

```
xcaf/skills/<name>/skill.xcaf
```

Per-provider overrides use a sibling file with a provider suffix:

```
xcaf/skills/<name>/skill.claude.xcaf
xcaf/skills/<name>/skill.cursor.xcaf
```

## Example Usage

### Minimal skill

```yaml
---
kind: skill
version: "1.0"
name: conventional-commits
description: >-
  Structured Git commit workflow using Conventional Commits with pre-commit
  checks, logical grouping, and anti-jargon enforcement.
---
# Conventional Commits

Use `feat`, `fix`, `chore`, `docs`, `refactor`, `test` as commit type.
Format: `<type>(<scope>): <description>`

Never include `Co-Authored-By` trailers.
Run `git diff --staged` before composing the message.
```

### Simple skill — instructions and rules

```yaml
---
kind: skill
version: "1.0"
name: component-patterns
description: >-
  Step-by-step procedure for implementing a new React component following
  frontend-app conventions — file layout, prop types, Tailwind styling,
  accessibility attributes, and co-located test scaffolding.
---
# Component Patterns

## When to Use
Invoke this skill whenever you are asked to create a new React component
or refactor an existing one to meet project standards.

## Step 1 — Confirm the component location
All new components live under `src/components/<Category>/<ComponentName>/`.
If the category does not exist, create it.

## Step 2 — Create the file structure
Each component directory contains four files:
- `index.ts` — re-export barrel
- `ComponentName.tsx` — implementation
- `ComponentName.test.tsx` — co-located tests
- `ComponentName.stories.tsx` — Storybook stories (optional)

## Step 3 — Implement with forwardRef
Use `forwardRef` for all interactive components. Accept a `className`
prop and merge it with `cn()` from `@/lib/utils`.

## Step 4 — Write co-located tests
Every component ships with at least two tests: renders correctly
and responds to its primary interaction.

## Step 5 — Export from the barrel
Append the public export to `src/components/index.ts`.

## Rules
- One component per directory. No multi-component files.
- All props must have TypeScript types. No `any`.
- Accessibility: every interactive element needs an aria label or role.
```

### Skill with artifacts — single folder

The `artifacts` field declares subdirectories that are composed alongside the compiled `SKILL.md`. Files within each subdirectory are discovered automatically from the filesystem.

**Source directory layout:**

```
xcaf/skills/component-patterns/
  skill.xcaf
  references/
    design-tokens.md
    shadcn-primitives.md
```

**Manifest:**

```yaml
---
kind: skill
version: "1.0"
name: component-patterns
description: >-
  Step-by-step procedure for implementing a new React component following
  frontend-app conventions.
artifacts:
  - references
---
# Component Patterns

## When to Use
Invoke this skill when creating or refactoring React components.

## Step 1 — Confirm the component location
…

## Reference Material
Consult `references/design-tokens.md` for spacing and color values.
Consult `references/shadcn-primitives.md` for base component APIs.
```

The `references/` directory and all its files are placed alongside `SKILL.md` in the compiled output (e.g., `.claude/skills/component-patterns/references/design-tokens.md`).

### Skill with artifacts — multiple folders

A skill can declare more than one artifact subdirectory. Each name maps to a sibling directory on disk.

**Source directory layout:**

```
xcaf/skills/deploy-checklist/
  skill.xcaf
  references/
    pre-deploy-gates.md
    rollback-playbook.md
  scripts/
    smoke-test.sh
    health-check.sh
  examples/
    staging.env.example
    production.env.example
```

**Manifest:**

```yaml
---
kind: skill
version: "1.0"
name: deploy-checklist
description: >-
  Production deployment checklist with automated smoke tests
  and rollback procedures.
artifacts:
  - references
  - scripts
  - examples
---
# Deploy Checklist

## Step 1 — Run pre-deploy gates
Review the checklist in `references/pre-deploy-gates.md`.

## Step 2 — Execute smoke tests
Run `scripts/smoke-test.sh` against the staging environment.

## Step 3 — Verify health
Run `scripts/health-check.sh` and confirm all endpoints return 200.

## Step 4 — Rollback plan
If any check fails, follow `references/rollback-playbook.md`.
```

**Valid artifact names:**

| Name | Typical contents |
|------|-----------------|
| `references` | Lookup tables, checklists, design tokens, specification excerpts |
| `scripts` | Shell scripts, automation helpers |
| `assets` | Static files, images, templates |
| `examples` | Sample configurations, fixture files |
| *(custom)* | Any kebab-case directory name is accepted |

## Field Reference

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `kind` | `string` | Resource type. Must be `skill`. |
| `version` | `string` | File format version. Must be `"1.0"`. |
| `name` | `string` | Unique skill identifier. Must match `[a-z0-9-]+`. |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `description` | `string` | Human-readable purpose of this skill. What it does and when to invoke it. |
| `when-to-use` | `string` | Guidance for the model on when to invoke this skill. Emitted only for Claude (as `when_to_use` in SKILL.md frontmatter); ignored by other providers. |
| `license` | `string` | SPDX license identifier (e.g. `"MIT"`, `"Apache-2.0"`). Emitted for Claude and Copilot; ignored by other providers. |
| `allowed-tools` | `[]string` | Tools the skill is permitted to use. Claude emits as a space-separated string. Copilot emits as a YAML list. Gemini drops this field with a fidelity warning. Cursor and Antigravity do not emit this field. |
| `disable-model-invocation` | `bool` | When `true`, prevents the skill from spawning a sub-agent. Claude-only. |
| `user-invocable` | `bool` | When `true`, exposes the skill as a slash command the user can invoke directly. Claude-only. |
| `argument-hint` | `string` | Hint text shown during slash-command invocation. Has effect only when `user-invocable: true`. Claude-only. |
| `artifacts` | `[]string` | Named subdirectories to compose with the skill. Valid canonical names: `references`, `scripts`, `assets`, `examples`. Custom names also accepted. Files within each subdirectory are discovered automatically from the filesystem. |
| `targets` | `map[string]TargetOverride` | Per-provider overrides keyed by provider name. |

## Filesystem-as-Schema

When a skill `.xcaf` file lives at `xcaf/skills/<name>/skill.xcaf`, the `kind:` and `name:` fields can be omitted from the YAML. The parser infers:
- `kind: skill` from the parent directory name (`skills/`)
- `name:` from the grandparent directory name (e.g., `component-patterns` from `skills/component-patterns/skill.xcaf`)

When `kind:` or `name:` are present in the YAML, they must match the inferred values.

## Compiled Output

### Claude

**Output path**: `.claude/skills/component-patterns/SKILL.md`

```markdown
---
name: component-patterns
description: >-
  Step-by-step procedure for implementing a new React component following
  frontend-app conventions — file layout, prop types, Tailwind styling,
  accessibility attributes, and co-located test scaffolding.
---
# Component Patterns

## When to Use
Invoke this skill whenever you are asked to create a new React component
or refactor an existing one to meet project standards.
...
```

When `artifacts` are declared, files discovered in each artifact subdirectory are placed alongside `SKILL.md`. For example, a skill declaring `artifacts: [references]` with `references/design-tokens.md` compiles to `.claude/skills/component-patterns/references/design-tokens.md`.

### Cursor

**Output path**: `.cursor/skills/component-patterns/SKILL.md`

Frontmatter is limited to `name` and `description`. Provider-specific fields (`when-to-use`, `license`, `disable-model-invocation`, `user-invocable`, `argument-hint`) are omitted. Artifact subdirectories are seeded at `.cursor/skills/component-patterns/<artifact>/`. Example files are collapsed into `references/` rather than a separate `examples/` subdirectory.

### Copilot

**Output path**: `.github/skills/component-patterns/SKILL.md`

Emits `name`, `description`, and `license` in the frontmatter. Artifact subdirectories are seeded at `.github/skills/component-patterns/<artifact>/`. Example files are seeded in an `examples/` subdirectory.

### Gemini

**Output path**: `.gemini/skills/component-patterns/SKILL.md`

Frontmatter is limited to `name` and `description`. Provider-specific fields are omitted. Artifact subdirectories are seeded at `.gemini/skills/component-patterns/<artifact>/`. Example files are collapsed into `references/` rather than a separate `examples/` subdirectory.

### Antigravity

**Output path**: `.agents/skills/component-patterns/SKILL.md`

Antigravity emits only `name` and `description` in the frontmatter. All other frontmatter fields (`when-to-use`, `license`, `allowed-tools`, `disable-model-invocation`, `user-invocable`, `argument-hint`) are stripped. The markdown body is preserved.

### Codex (Preview)

**Output path**: `.agents/skills/component-patterns/SKILL.md`

Codex shares the Antigravity skill output path (`.agents/skills/`). Xcaffold emits `name` and `description` in the frontmatter; all other frontmatter fields are stripped. The markdown body is preserved. Artifact subdirectories are placed as-is alongside `SKILL.md`.

### Artifact path remapping

Some providers remap artifact directory names during compilation:

| Source directory | Claude | Cursor | Copilot | Gemini | Antigravity | Codex |
|------------------|--------|--------|---------|--------|-------------|-------|
| `references/` | `references/` | `references/` | `references/` | `references/` | `examples/` | `references/` |
| `scripts/` | `scripts/` | `scripts/` | `scripts/` | `scripts/` | `scripts/` | `scripts/` |
| `assets/` | `assets/` | `assets/` | `assets/` | `assets/` | `resources/` | `assets/` |
| `examples/` | `examples/` | `references/` | `examples/` | `references/` | `examples/` | `examples/` |

## Provider Fidelity

Skill output is not uniform across providers. The table below summarises what each provider preserves.

| Field | Claude | Cursor | Copilot | Gemini | Antigravity | Codex (Preview) |
|-------|--------|--------|---------|--------|-------------|-----------------|
| `name` | yes | yes | yes | yes | yes | yes |
| `description` | yes | yes | yes | yes | yes | yes |
| `when-to-use` | yes (`when_to_use`) | no | no | no | no | no |
| `license` | yes | no | yes | no | no | no |
| `allowed-tools` | yes (space-separated) | no | yes (YAML list) | no (warning) | no | no |
| `disable-model-invocation` | yes | no | no | no | no | no |
| `user-invocable` | yes | no | no | no | no | no |
| `argument-hint` | yes | no | no | no | no | no |
| Markdown body | yes | yes | yes | yes | yes | yes |
| `references/` subdirectory | yes | yes | yes | yes | compiled to `examples/` | yes |
| `examples/` placement | skill root | `references/` | `examples/` | `references/` | `examples/` | `examples/` |
| `scripts/` subdirectory | yes | yes | yes | yes | yes | yes |
| `assets/` subdirectory | yes | yes | yes | yes | compiled to `resources/` | yes |
