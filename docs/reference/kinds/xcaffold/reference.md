---
title: "kind: reference"
description: "Declares a supporting file seeded verbatim into provider output directories at compile time. Produces no transformed output."
---

# `kind: reference`

Declares a supporting document or data file that is seeded verbatim into provider output directories at compile time. Reference files are not transformed — their content is copied as-is. Common uses: runbooks, style guides, decision registries, and data files that agents read during sessions.

Reference files are declared in `project.xcf` and sourced from paths relative to the project root.

> Reference files are **not** authored using a `kind: reference` declaration in a standalone `.xcf` file. They are declared inline in the project manifest or skill definitions and sourced from existing files on disk.

## Example Usage

### Referencing a design system guide

In `project.xcf`:
```yaml
---
kind: project
version: "1.0"
name: frontend-app
targets: [claude, cursor, gemini]
agents:
  - id: react-developer
    path: xcf/agents/react-developer/react-developer.xcf
---
```

In the skill definition (`xcf/skills/component-patterns.xcf`):
```yaml
---
kind: skill
version: "1.0"
name: component-patterns
description: Step-by-step procedure for implementing a React component.
references:
  - xcf/skills/component-patterns/references/design-tokens.md
  - xcf/skills/component-patterns/references/shadcn-primitives.md
---
# Component Patterns
…
```

Source file (`xcf/skills/component-patterns/references/design-tokens.md`):
```markdown
# Design Tokens

## Color Tokens
All colors reference tokens defined in `tailwind.config.ts`:
- `colors.brand.primary` — primary action color (#6366F1)
- `colors.brand.secondary` — secondary accent (#EC4899)
- `colors.surface.default` — card/panel background (#1E1E2E)
- `colors.text.primary` — body text (#E2E8F0)

## Spacing Tokens
Use Tailwind's default spacing scale. Never use arbitrary values like `px-[13px]`.
```

## Compiled Output

Reference files are seeded into the `references/` subdirectory alongside the skill or agent that declared them:

```
.claude/
  skills/
    component-patterns/
      SKILL.md
      references/
        design-tokens.md           ← seeded verbatim
        shadcn-primitives.md       ← seeded verbatim
```

The same file is seeded into all five providers at the equivalent path under their output directory (`.cursor/`, `.gemini/`, `.github/`, `.agents/`).

## Argument Reference

References are declared as path strings, not as standalone `.xcf` files. The following fields apply to each path entry:

- `path` — (Required) Relative path from the project root to the source file. The file must exist on disk at compile time or `xcaffold apply` will fail with a missing reference error.
- `name` — (Optional) Display name for the reference. Defaults to the filename stem.
- `description` — (Optional) One-line description. Defaults to the first line of the file.

> [!WARNING]
> `xcaffold apply` fails with a non-zero exit code if any declared reference path does not exist. Reference paths are resolved relative to the project root — not relative to the `.xcf` file that declares them.
