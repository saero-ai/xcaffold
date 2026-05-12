---
title: "kind: workflow"
description: "Defines a named, multi-step procedure compiled to provider-native workflow output for all five providers."
---

# `kind: workflow`

Defines a named, reusable multi-step procedure. All five providers compile workflow resources; the output format depends on each provider's native support and the active lowering strategy.

> **Required:** `kind`, `version`, `name`

## Source Directory

```
xcaf/workflows/<name>/workflow.xcaf
```

Each workflow is a directory-per-resource under `xcaf/workflows/`. The `.xcaf` file contains the frontmatter and optional body for that workflow.

## Example Usage

### Single-step workflow (body form)

```yaml
---
kind: workflow
version: "1.0"
name: run-component-audit
description: Audit all React components in the project against current conventions.
---
Run `xcaffold list --kind agent` to confirm the react-developer agent is active.

Then scan every file under `src/components/` and check:
1. All components use functional component syntax.
2. All components have a co-located `.test.tsx` file.
3. All props use `type`, not `interface`.

Report: "Audited <N> components. Found <M> violations."
```

### Multi-step workflow with ordered steps

```yaml
---
kind: workflow
version: "1.0"
api-version: workflow/v1
name: ship-feature
description: End-to-end workflow for shipping a new React component from branch to PR.
steps:
  - name: implement
    description: Build the component using the component-patterns skill.

  - name: review
    description: Self-review for convention compliance.

  - name: pr
    description: Open a pull request.
---
```

Step content is authored as named body sections using `## <step-name>` headings in the workflow body:

```markdown
## implement

1. Create a git worktree: `git worktree add .worktrees/feat-<name> feat/<name>`
2. Invoke the component-patterns skill to scaffold the component.
3. Run `pnpm test --filter frontend-app` — all tests must pass.

## review

1. Run `pnpm lint` — zero ESLint errors.
2. Check every new component has a co-located `.test.tsx`.
3. Confirm no `any` types appear in props.

## pr

1. Commit with: `feat(ui): add <ComponentName> component`
2. Push the branch and open a PR targeting `main`.
3. Include a Storybook screenshot in the PR description.
```

## Field Reference

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | `string` | Unique workflow identifier. Must match `[a-z0-9-]+`. |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `api-version` | `string` | Schema shape discriminator for workflow versioning. Default: `"workflow/v1"`. |
| `description` | `string` | Human-readable description of what this workflow accomplishes. |
| `steps` | `[]WorkflowStep` | Ordered procedural steps. Use when the workflow has multiple phases. |
| `targets` | `map[string]TargetOverride` | Per-provider overrides for compilation behavior. Does not restrict which providers compile this resource; to exclude a provider, set `skip-synthesis: true` within that provider's entry. |

### `steps` entry

Each entry in `steps` supports:

| Field | Type | Description |
|-------|------|-------------|
| `name` | `string` | (Required) Step identifier. Used as a heading in the compiled output and as the key for the corresponding `## <name>` body section. |
| `description` | `string` | (Optional) One-line step summary placed beneath the step heading in the compiled output. |

Step instructions are provided as `## <step-name>` body sections in the workflow source file, not as an inline `instructions:` field.

## Lowering Strategies

When a provider has no native workflow runtime, xcaffold lowers the workflow to a compatible pair of output files: a rule that describes the overall procedure and a skill per step that contains the step's instructions. A fidelity note is appended to the compiled output: `> workflow lowered to rule+skill`.

| Strategy | Value | Applies To |
|---|---|---|
| Rule + skill (default) | `rule-plus-skill` | Claude, Cursor, Copilot, Gemini |
| Prompt file | `prompt-file` | Copilot |
| Custom command | `custom-command` | Gemini |

To override the default strategy for a provider, set `lowering-strategy` inside `targets.<provider>.provider`:

```yaml
targets:
  copilot:
    provider:
      lowering-strategy: prompt-file
  gemini:
    provider:
      lowering-strategy: custom-command
```

When `lowering-strategy` is absent, `rule-plus-skill` is used.

## Compiled Output

### Claude

**Default strategy**: `rule-plus-skill`

**Output paths** (example: `ship-feature`):
- `rules/ship-feature-workflow.md` — procedure overview rule
- `skills/ship-feature-01-implement/SKILL.md`
- `skills/ship-feature-02-review/SKILL.md`
- `skills/ship-feature-03-pr/SKILL.md`

> workflow lowered to rule+skill

### Cursor

**Default strategy**: `rule-plus-skill`

**Output paths** (example: `ship-feature`):
- `rules/ship-feature-workflow.mdc` — procedure overview rule
- `skills/ship-feature-01-implement/SKILL.md`
- `skills/ship-feature-02-review/SKILL.md`
- `skills/ship-feature-03-pr/SKILL.md`

> workflow lowered to rule+skill

### Copilot

**Default strategy**: `rule-plus-skill`

**Output paths** (example: `ship-feature`):
- `.github/instructions/ship-feature-workflow.instructions.md` — procedure overview rule
- `.github/skills/ship-feature-01-implement/SKILL.md`
- `.github/skills/ship-feature-02-review/SKILL.md`
- `.github/skills/ship-feature-03-pr/SKILL.md`

> workflow lowered to rule+skill

**With `lowering-strategy: prompt-file`:**

**Output path**: `.github/prompts/ship-feature.prompt.md`

```markdown
---
mode: agent
x-xcaffold:
  compiled-from: workflow
  workflow-name: ship-feature
  api-version: workflow/v1
---

...step bodies concatenated...
```

### Gemini

**Default strategy**: `rule-plus-skill`

**Output paths** (example: `ship-feature`):
- `.gemini/rules/ship-feature-workflow.md` — procedure overview rule
- `GEMINI.md` — receives an `@`-import line referencing the rule
- `.gemini/skills/ship-feature-01-implement/SKILL.md`
- `.gemini/skills/ship-feature-02-review/SKILL.md`
- `.gemini/skills/ship-feature-03-pr/SKILL.md`

> workflow lowered to rule+skill

**With `lowering-strategy: custom-command`:**

**Output path**: `.gemini/commands/ship-feature.md`

```markdown
...step bodies concatenated with blank lines between them...
```

### Antigravity

**Default output path**: `.agents/workflows/ship-feature.md`

The default path writes a frontmatter block with `description:` followed by the workflow body verbatim. The `steps:` array entries are not rendered as section headings.

```markdown
---
description: "End-to-end workflow for shipping a new React component from branch to PR."
---

...workflow body written as-is...
```

**With `promote-rules-to-workflows: true`:**

Set this flag in the `antigravity` target override to activate native step-body rendering. Steps are concatenated under `## <step-name>` headers (level-2, no numbering).

```yaml
targets:
  antigravity:
    provider:
      promote-rules-to-workflows: true
```

Output:

```markdown
## implement

...step body...

## review

...step body...

## pr

...step body...
```
