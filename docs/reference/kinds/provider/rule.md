---
title: "kind: rule"
description: "Defines a constraint. Source: xcaf/rules/<name>/rule.xcaf. Compiled to rules/<name>.md (or .mdc) per provider."
---

# `kind: rule`

Defines a constraint the agent must follow at all times, or only when working within specific file paths. Compiled to provider-native rule files for all five targets.

> **Required:** `kind`, `version`, `name`

## Source Directory

```
xcaf/rules/<name>/rule.xcaf
```

## Example Usage

### Always-apply rule

```yaml
---
kind: rule
version: "1.0"
name: react-conventions
description: >-
  Enforces React 18+ functional component patterns, hook rules, and
  TypeScript type annotations for all UI code in frontend-app.
---
# React Conventions

## Component Structure
- Use functional components only — no class components.
- Export components with named exports; default exports only in page files.
- Props must use `type` (not `interface`) with a `ComponentProps` suffix.

## Hooks
- Never call hooks conditionally or inside loops.
- Custom hooks must start with `use` and live in `src/hooks/`.
- `useState` initial values must be typed explicitly when not inferrable.

## TypeScript
- No `any` types in component props or hook return values.
- Use `ComponentProps<'element'>` to extend HTML element props.
- Never suppress TS errors with `@ts-ignore`.

## Styling
- No inline `style={{}}` attributes — use Tailwind utility classes only.
- Use `cn()` from `@/lib/utils` for conditional classes.
- Responsive classes must be mobile-first (`sm:`, `md:`, `lg:`).

## Testing
- Every component file requires a co-located `.test.tsx` file.
- Use React Testing Library — no Enzyme, no direct DOM access.
- Assert on accessible roles: `getByRole`, `getByLabelText`.
```

### Path-scoped rule

```yaml
---
kind: rule
version: "1.0"
name: no-server-imports-in-ui
description: Prevents server-only modules from being imported inside client components.
activation: path-glob
paths:
  - "src/components/**"
  - "src/hooks/**"
---
# No Server Imports in UI

Never import from modules marked `server-only` or from `src/server/` inside
components or hooks. This causes Next.js build failures on the client bundle.

**Allowed**: `@/lib/utils`, `@/types`, `@/components/ui`
**Forbidden**: `server-only`, `next/headers`, `src/server/**`
```

## Field Reference

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | `string` | Unique rule identifier. Must match `[a-z0-9-]+`. |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `description` | `string` | Human-readable description of what constraint this rule enforces. |
| `activation` | `string` | Controls when the rule is applied. Values: `always` (default), `path-glob`, `manual-mention`, `model-decided`, `explicit-invoke`. |
| `always-apply` | `bool` | Legacy alias for `activation`. `true` maps to `activation: always`; `false` maps to `activation: manual-mention`. When both `always-apply` and `activation` are present, `activation` takes precedence. |
| `paths` | `[]string` | Glob patterns used when `activation: path-glob`. Each pattern is matched against the agent's currently open or modified files. |
| `exclude-agents` | `[]string` | Prevents the rule from being applied by specific agent types. Accepted values: `code-review`, `cloud-agent`. Copilot-specific. |
| `targets` | `map[string]TargetOverride` | Per-provider overrides keyed by provider name. |

## Filesystem-as-Schema

When a rule `.xcaf` file lives at `xcaf/rules/<name>/rule.xcaf`, the `kind:` and `name:` fields can be omitted from the YAML. The parser infers:
- `kind: rule` from the parent directory name (`rules/`)
- `name:` from the grandparent directory name (e.g., `react-conventions` from `rules/react-conventions/rule.xcaf`)

When `kind:` or `name:` are present in the YAML, they should match the inferred values. A mismatch produces a parse warning.

## Compiled Output

### Claude

**Always-apply** → `.claude/rules/react-conventions.md`

```markdown
---
description: "Enforces React 18+ functional component patterns, hook rules, and TypeScript type annotations for all UI code in frontend-app."
---

# React Conventions

## Component Structure
- Use functional components only — no class components.
…
```

**Path-scoped** → `.claude/rules/no-server-imports-in-ui.md`

```markdown
---
description: "Prevents server-only modules from being imported inside client components."
paths: [src/components/**, src/hooks/**]
---

# No Server Imports in UI
…
```

### Cursor

**Always-apply** → `.cursor/rules/react-conventions.mdc`

```markdown
---
description: "Enforces React 18+ functional component patterns, hook rules, and TypeScript type annotations for all UI code in frontend-app."
always-apply: true
---

# React Conventions
…
```

**Path-scoped** → `.cursor/rules/no-server-imports-in-ui.mdc`

```markdown
---
description: "Prevents server-only modules from being imported inside client components."
globs: [src/components/**, src/hooks/**]
---

# No Server Imports in UI
…
```

> Cursor uses `.mdc` extension. Supported activation modes: `always`, `path-glob`, and `manual-mention`. Always-apply rules emit `always-apply: true`. Path-scoped rules emit `globs:`. Manual-mention rules emit no activation key.

### Copilot

Each rule compiles to its own file under `.github/instructions/`.

**Always-apply** → `.github/instructions/react-conventions.instructions.md`

```markdown
---
applyTo: "**"
---

# React Conventions

## Component Structure
- Use functional components only — no class components.
…
```

**Path-scoped** → `.github/instructions/no-server-imports-in-ui.instructions.md`

```markdown
---
applyTo: "src/components/**, src/hooks/**"
---

# No Server Imports in UI

Never import from modules marked `server-only`…
```

> Copilot writes one `.github/instructions/<name>.instructions.md` file per rule. Always-apply rules emit `applyTo: "**"`. Path-scoped rules emit `applyTo:` set to the joined glob patterns.

### Gemini

**Always-apply** → import line added to root `GEMINI.md`:

```
@.gemini/rules/react-conventions.md
```

Individual rule file `.gemini/rules/react-conventions.md`:

```markdown
# React Conventions

## Component Structure
…
```

**Path-scoped** → single rule file at `.gemini/rules/no-server-imports-in-ui.md` with a two-line entry added to the root `GEMINI.md`:

Root `GEMINI.md` addition:
```
Apply this rule when accessing src/components/**, src/hooks/**:
@.gemini/rules/no-server-imports-in-ui.md
```

Individual rule file `.gemini/rules/no-server-imports-in-ui.md`:

```markdown
# No Server Imports in UI

Never import from modules marked `server-only`…
```

> Gemini scoping is implemented via a two-line entry in the root `GEMINI.md`: a plain-text path constraint on the first line, followed by the `@-import` directive on the second. The individual rule file contains only the rule body — no path comment or frontmatter. Nested per-directory `GEMINI.md` files are not used.

### Antigravity

**Always-apply** → `.agents/rules/react-conventions.md`

```markdown
---
description: "Enforces React 18+ functional component patterns, hook rules, and TypeScript type annotations for all UI code in frontend-app."
---

# React Conventions
…
```

**Path-scoped** → `.agents/rules/no-server-imports-in-ui.md`

```markdown
---
description: "Prevents server-only modules from being imported inside client components."
trigger: glob
globs: src/components/**,src/hooks/**
---

# No Server Imports in UI
…
```

Path-scoped rules emit `trigger: glob` and `globs: <comma-joined-patterns>` in the frontmatter. Always-apply rules emit no activation key.

**Model-decided** → `.agents/rules/react-conventions.md`

```markdown
---
description: "Enforces React 18+ functional component patterns, hook rules, and TypeScript type annotations for all UI code in frontend-app."
trigger: model_decision
---

# React Conventions
…
```

Antigravity is the only provider that natively supports `model-decided` activation. Rules compiled with this activation emit `trigger: model_decision` in the frontmatter.

> [!NOTE]
> **model-decided** and **explicit-invoke**: These activation modes are not natively supported by Claude Code, Cursor, Copilot, or Gemini. Rules compiled with these modes include a fidelity note explaining the limitation. Antigravity natively supports `model-decided`.
