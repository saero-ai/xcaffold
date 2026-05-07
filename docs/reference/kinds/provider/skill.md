---
title: "kind: skill"
description: "Defines a reusable procedure. Source: xcaf/skills/<name>/skill.xcaf. Compiled to skills/<name>/SKILL.md per provider."
---

# `kind: skill`

Defines a reusable procedure that agents invoke on-demand. Compiled to `skills/<id>/SKILL.md` with YAML frontmatter for all five target providers.

> **Required:** `kind`, `version`, `name`

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
references:
  - xcaf/skills/component-patterns/references/design-tokens.md
  - xcaf/skills/component-patterns/references/shadcn-primitives.md
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

## Argument Reference

The following arguments are supported:

- `name` — (Required) Unique skill identifier. Must match `[a-z0-9-]+`.
- `description` — (Optional) `string`. What this skill does and when to invoke it.
- `references` — (Optional) `[]string`. Relative paths to supporting files seeded alongside `SKILL.md` in a `references/` subdirectory.
- `examples` — (Optional) `[]string`. Relative paths to example files seeded in an `examples/` subdirectory.
- `scripts` — (Optional) `[]string`. Relative paths to executable scripts seeded in a `scripts/` subdirectory.
- `assets` — (Optional) `[]string`. Relative paths to binary or data assets seeded alongside `SKILL.md`.
- `targets` — (Optional) `map[string]TargetOverride`. Per-provider overrides.

## Filesystem-as-Schema

When a skill `.xcaf` file lives at `xcaf/skills/<name>/skill.xcaf`, the `kind:` and `name:` fields can be omitted from the YAML. The parser infers:
- `kind: skill` from the parent directory name (`skills/`)
- `name:` from the grandparent directory name (e.g., `component-patterns` from `skills/component-patterns/skill.xcaf`)

When `kind:` or `name:` are present in the YAML, they must match the inferred values.

## Compiled Output

<ProviderTabs>
  <ProviderTab id="claude">
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

    Reference files are placed at `.claude/skills/component-patterns/references/design-tokens.md` and `.claude/skills/component-patterns/references/shadcn-primitives.md`.
  </ProviderTab>

  <ProviderTab id="cursor">
    **Output path**: `.cursor/skills/component-patterns/SKILL.md`

    Content is identical to the Claude output. Reference files are seeded at `.cursor/skills/component-patterns/references/`.
  </ProviderTab>

  <ProviderTab id="copilot">
    **Output path**: `.github/skills/component-patterns/SKILL.md`

    Content is identical to the Claude output. Reference files are seeded at `.github/skills/component-patterns/references/`.
  </ProviderTab>

  <ProviderTab id="gemini">
    **Output path**: `.gemini/skills/component-patterns/SKILL.md`

    Content is identical to the Claude output. Reference files are seeded at `.gemini/skills/component-patterns/references/`.
  </ProviderTab>

  <ProviderTab id="antigravity">
    **Output path**: `.agents/skills/component-patterns/SKILL.md`

    Content is identical to the Claude output. Reference files are seeded at `.agents/skills/component-patterns/references/`.
  </ProviderTab>
</ProviderTabs>

> Skills have full parity across all five providers. The `SKILL.md` content and directory structure are identical. Only the root output directory differs.
