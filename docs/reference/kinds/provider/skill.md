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

### Full skill — component patterns with references

```yaml
---
kind: skill
version: "1.0"
name: component-patterns
description: >-
  Step-by-step procedure for implementing a new React component following
  frontend-app conventions — file layout, prop types, Tailwind styling,
  accessibility attributes, and co-located test scaffolding.
artifacts: [references]
---
# Component Patterns

## When to Use
Invoke this skill whenever you are asked to create a new React component
or refactor an existing one to meet project standards.

## Step 1 — Confirm the component's location
All new components live under `src/components/<Category>/<ComponentName>/`.
If the category does not exist, create it.

## Step 2 — Create the file structure
```
src/components/<Category>/<ComponentName>/
  index.ts                  ← re-export barrel
  <ComponentName>.tsx       ← implementation
  <ComponentName>.test.tsx  ← co-located tests
  <ComponentName>.stories.tsx
```

## Step 3 — Implement the component
```tsx
import { type ComponentProps, forwardRef } from 'react';
import { cn } from '@/lib/utils';
import { buttonVariants } from './variants';

type ButtonProps = ComponentProps<'button'> & {
  variant?: 'primary' | 'secondary' | 'ghost';
  size?: 'sm' | 'md' | 'lg';
};

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ variant = 'primary', size = 'md', className, children, ...props }, ref) => (
    <button
      ref={ref}
      className={cn(buttonVariants({ variant, size }), className)}
      {...props}
    >
      {children}
    </button>
  )
);
Button.displayName = 'Button';
```

## Step 4 — Write the test
```tsx
import { render, screen } from '@testing-library/react';
import { Button } from '.';

describe('Button', () => {
  it('renders children', () => {
    render(<Button>Click me</Button>);
    expect(screen.getByRole('button', { name: /click me/i })).toBeInTheDocument();
  });

  it('applies variant classes', () => {
    render(<Button variant="ghost">Ghost</Button>);
    expect(screen.getByRole('button')).toHaveClass('bg-transparent');
  });
});
```

## Step 5 — Add to barrel
Append to `src/components/index.ts`:
```ts
export * from './<Category>/<ComponentName>';
```

## Deliverables
Report: "Created <ComponentName> with <N> props, <M> tests passing."
```

## Field Reference

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
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
…
```

Files discovered in the `references/` artifact subdirectory are placed at `.claude/skills/component-patterns/references/` (e.g., `design-tokens.md`, `shadcn-primitives.md`).

### Cursor

**Output path**: `.cursor/skills/component-patterns/SKILL.md`

Frontmatter is limited to `name` and `description`. Provider-specific fields (`when-to-use`, `license`, `disable-model-invocation`, `user-invocable`, `argument-hint`) are omitted. Reference files are seeded at `.cursor/skills/component-patterns/references/`. Example files are collapsed into `references/` rather than a separate `examples/` subdirectory.

### Copilot

**Output path**: `.github/skills/component-patterns/SKILL.md`

Emits `name`, `description`, and `license` in the frontmatter. Reference files are seeded at `.github/skills/component-patterns/references/`. Example files are seeded in an `examples/` subdirectory.

### Gemini

**Output path**: `.gemini/skills/component-patterns/SKILL.md`

Frontmatter is limited to `name` and `description`. Provider-specific fields are omitted. Reference files are seeded at `.gemini/skills/component-patterns/references/`. Example files are collapsed into `references/` rather than a separate `examples/` subdirectory.

### Antigravity

**Output path**: `.agents/skills/component-patterns/SKILL.md`

Antigravity emits only `name` and `description` in the frontmatter. All other frontmatter fields (`when-to-use`, `license`, `allowed-tools`, `disable-model-invocation`, `user-invocable`, `argument-hint`) are stripped. The markdown body is preserved.

Artifact subdirectories are compiled with path remapping:

| Source directory | Antigravity output directory |
|------------------|------------------------------|
| `references/` | `examples/` |
| `scripts/` | `scripts/` |
| `assets/` | `resources/` |
| `examples/` | `examples/` |

## Provider Fidelity

Skill output is not uniform across providers. The table below summarises what each provider preserves.

| Field | Claude | Cursor | Copilot | Gemini | Antigravity |
|-------|--------|--------|---------|--------|-------------|
| `name` | yes | yes | yes | yes | yes |
| `description` | yes | yes | yes | yes | yes |
| `when-to-use` | yes (`when_to_use`) | no | no | no | no |
| `license` | yes | no | yes | no | no |
| `allowed-tools` | yes (space-separated) | no | yes (YAML list) | no (warning) | no |
| `disable-model-invocation` | yes | no | no | no | no |
| `user-invocable` | yes | no | no | no | no |
| `argument-hint` | yes | no | no | no | no |
| Markdown body | yes | yes | yes | yes | yes |
| `references/` subdirectory | yes | yes | yes | yes | compiled to `examples/` |
| `examples` placement | skill root | `references/` | `examples/` | `references/` | `examples/` |
| `scripts/` subdirectory | yes | yes | yes | yes | yes |
| `assets/` subdirectory | yes | yes | yes | yes | compiled to `resources/` |
