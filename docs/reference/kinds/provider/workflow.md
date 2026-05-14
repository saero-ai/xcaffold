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

Each workflow is a directory-per-resource under `xcaf/workflows/`. The `.xcaf` file contains pure YAML fields for that workflow — there is no body section.

## Example Usage

### Basic workflow (inline instructions)

```yaml
kind: workflow
version: "1.0"
name: run-component-audit
description: Audit all React components against conventions.
steps:
  - name: audit
    instructions: |
      Run `xcaffold list --kind agent` to confirm the react-developer agent is active.
      Then scan every file under `src/components/` and check:
      1. All components use functional component syntax.
      2. All components have a co-located `.test.tsx` file.
      3. All props use `type`, not `interface`.
      Report: "Audited <N> components. Found <M> violations."
```

### Multi-step workflow with skill references

```yaml
kind: workflow
version: "1.0"
name: ship-feature
description: End-to-end workflow for shipping a new React component.
activation: always
steps:
  - name: implement
    skill: component-patterns
    description: Build the component using the patterns skill.
  - name: review
    instructions: |
      1. Run `pnpm lint` — zero ESLint errors.
      2. Check every new component has a `.test.tsx`.
      3. Confirm no `any` types appear in props.
  - name: pr
    instructions: |
      1. Commit with: `feat(ui): add <ComponentName> component`
      2. Push the branch and open a PR targeting `main`.
```

## Field Reference

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `kind` | `string` | Resource type. Must be `workflow`. |
| `version` | `string` | File format version. Must be `"1.0"`. |
| `name` | `string` | Unique workflow identifier. Must match `[a-z0-9-]+`. |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `activation` | `string` | Controls rule generation. Set to `"always"` to emit an always-active rule, or provide a list of path globs. Omit for basic lowering (no rule). |
| `description` | `string` | Human-readable description of what this workflow accomplishes. |
| `steps` | `[]WorkflowStep` | Ordered procedural steps. |
| `targets` | `map[string]TargetOverride` | Per-provider overrides for compilation behavior. Does not restrict which providers compile this resource; to exclude a provider, set `skip-synthesis: true` within that provider's entry. |

### `steps` entry

Each entry in `steps` supports:

| Field | Type | Description |
|-------|------|-------------|
| `name` | `string` | (Required) Step identifier. Used as a heading in the compiled output. |
| `description` | `string` | (Optional) One-line step summary placed beneath the step heading in the compiled output. |
| `instructions` | `string` | (Optional) Inline step content. Provide either `instructions` or `skill`, not both. |
| `skill` | `string` | (Optional) Reference to an existing skill by name. The skill's compiled content is used as the step body. Provide either `skill` or `instructions`, not both. |

### `targets` entry

Each entry in `targets` supports:

| Field | Type | Description |
|-------|------|-------------|
| `skip-synthesis` | `bool` | When `true`, excludes this provider from compilation entirely. |
| `suppress-fidelity-warnings` | `bool` | When `true`, silences fidelity notes for this provider. |
| `provider` | `map[string]any` | Opaque pass-through map for provider-specific directives. |

#### `provider` directives

The `provider` map accepts provider-specific keys. xcaffold recognizes:

| Key | Valid Values | Description |
|-----|-------------|-------------|
| `lowering-strategy` | `prompt-file`, `custom-command`, `rule-plus-skill` | Overrides the structure-inferred lowering strategy for this provider. |
| `promote-rules-to-workflows` | `true` / `false` | Emits native workflow output instead of lowering to skill primitives. Used by Antigravity. |

## Lowering Strategies

When a provider has no native workflow runtime, xcaffold lowers the workflow to compatible output files. The default lowering strategy is inferred from the workflow's structure:

- If any step references a `skill:` → **orchestrator mode**: a main skill plus one sub-skill per step.
- If all steps use inline `instructions:` → **basic mode**: a single skill file with each step as a `##`-headed section.

When `activation:` is set, an additional rule primitive is emitted alongside the skill output.

| Strategy | Value | Applies To |
|---|---|---|
| Basic (default, all instructions) | *(structure-inferred)* | Claude, Cursor, Copilot, Gemini |
| Orchestrator (any skill ref) | *(structure-inferred)* | Claude, Cursor, Copilot, Gemini |
| Rule + skill (explicit) | `rule-plus-skill` | Claude, Cursor, Copilot, Gemini |
| Prompt file | `prompt-file` | Copilot |
| Custom command | `custom-command` | Gemini |

To override the inferred strategy for a provider, set `lowering-strategy` inside `targets.<provider>.provider`:

```yaml
targets:
  copilot:
    provider:
      lowering-strategy: prompt-file
  gemini:
    provider:
      lowering-strategy: custom-command
```

## Compiled Output

### Claude

**Default strategy**: basic mode (single skill with `##`-headed step sections)

**Output path** (example: `run-component-audit`):
- `skills/run-component-audit/SKILL.md`

When `activation:` is set, a rule is also emitted:
- `rules/run-component-audit-workflow.md` — activation rule

> workflow lowered to skill

### Cursor

**Default strategy**: basic mode (single skill with `##`-headed step sections)

**Output path** (example: `run-component-audit`):
- `skills/run-component-audit/SKILL.md`

When `activation:` is set, a rule is also emitted:
- `rules/run-component-audit-workflow.mdc` — activation rule

> workflow lowered to skill

### Copilot

**Default strategy**: basic mode (single skill with `##`-headed step sections)

**Output path** (example: `run-component-audit`):
- `.github/skills/run-component-audit/SKILL.md`

When `activation:` is set, a rule is also emitted:
- `.github/instructions/run-component-audit-workflow.instructions.md` — activation rule

> workflow lowered to skill

**With `lowering-strategy: prompt-file`:**

**Output path**: `.github/prompts/run-component-audit.prompt.md`

```markdown
---
mode: agent
x-xcaffold:
  compiled-from: workflow
  workflow-name: run-component-audit
  api-version: workflow/v1
---

...step instructions concatenated...
```

### Gemini

**Default strategy**: basic mode (single skill with `##`-headed step sections)

**Output paths** (example: `run-component-audit`):
- `.gemini/skills/run-component-audit/SKILL.md`
- `GEMINI.md` — receives an `@`-import line referencing the skill

When `activation:` is set, a rule is also emitted:
- `.gemini/rules/run-component-audit-workflow.md` — activation rule

> workflow lowered to skill

**With `lowering-strategy: custom-command`:**

**Output path**: `.gemini/commands/run-component-audit.md`

```markdown
...step instructions concatenated with blank lines between them...
```

### Antigravity

**Default output path**: `.agents/workflows/run-component-audit.md`

Antigravity always renders native workflow output. Steps are concatenated under `## <step-name>` headers (level-2, no numbering).

```markdown
## implement

...step instructions...

## review

...step instructions...

## pr

...step instructions...
```
