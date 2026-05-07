---
title: "kind: rule"
description: "Defines a constraint. Source: xcaf/rules/<name>/rule.xcaf. Compiled to rules/<name>.md (or .mdc) per provider."
---

# `kind: rule`

Defines a constraint the agent must follow at all times, or only when working within specific file paths. Compiled to provider-native rule files for all five targets.

> **Required:** `kind`, `version`, `name`

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

## Argument Reference

The following arguments are supported:

- `name` — (Required) Unique rule identifier. Must match `[a-z0-9-]+`.
- `description` — (Optional) `string`. What constraint this rule enforces.
- `activation` — (Optional) `string`. Controls when the rule is applied. Values:
  - `always` (default) — rule is active in every context
  - `path-glob` — rule activates only when the agent's working files match `paths`
  - `model-decided` — the model decides whether to apply the rule based on context
  - `manual-mention` — rule only activates when explicitly referenced by the user
  - `explicit-invoke` — rule only activates when directly invoked (e.g., `@rule-name`)
- `paths` — (Optional) `[]string`. Glob patterns used when `activation: path-glob`. Each pattern is matched against the agent's currently open or modified files.
- `targets` — (Optional) `map[string]TargetOverride`. Per-provider overrides.

## Filesystem-as-Schema

When a rule `.xcaf` file lives at `xcaf/rules/<name>/rule.xcaf`, the `kind:` and `name:` fields can be omitted from the YAML. The parser infers:
- `kind: rule` from the parent directory name (`rules/`)
- `name:` from the grandparent directory name (e.g., `react-conventions` from `rules/react-conventions/rule.xcaf`)

When `kind:` or `name:` are present in the YAML, they must match the inferred values.

## Compiled Output

<ProviderTabs>
  <ProviderTab id="claude">
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
    globs:
      - "src/components/**"
      - "src/hooks/**"
    ---

    # No Server Imports in UI
    …
    ```
  </ProviderTab>

  <ProviderTab id="cursor">
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
    globs:
      - "src/components/**"
      - "src/hooks/**"
    ---

    # No Server Imports in UI
    …
    ```

    > Cursor uses `.mdc` extension. Always-apply rules emit `always-apply: true`. Path-scoped rules emit `globs:` instead.
  </ProviderTab>

  <ProviderTab id="copilot">
    **Output path**: `.github/copilot-instructions.md`

    Rules are appended inline into a single instructions file:

    ```markdown
    # React Conventions

    ## Component Structure
    - Use functional components only — no class components.
    …

    # No Server Imports in UI

    Never import from modules marked `server-only`…
    ```

    > Copilot has no per-rule file concept. All rules are concatenated into `.github/copilot-instructions.md`. Path scoping is not natively supported and is noted in the file as a comment.
  </ProviderTab>

  <ProviderTab id="gemini">
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

    **Path-scoped** → nested `GEMINI.md` placed inside each matching directory:

    `src/components/GEMINI.md` and `src/hooks/GEMINI.md`:
    ```
    @.gemini/rules/no-server-imports-in-ui.md
    ```

    > Gemini implements path scoping via nested `GEMINI.md` files placed in each directory matching the glob pattern. The individual rule file lives at `.gemini/rules/no-server-imports-in-ui.md`.
  </ProviderTab>

  <ProviderTab id="antigravity">
    **Output path**: `.agents/rules/react-conventions.md`

    ```markdown
    ---
    description: "Enforces React 18+ functional component patterns, hook rules, and TypeScript type annotations for all UI code in frontend-app."
    ---

    # React Conventions
    …
    ```

    Path-scoped rules emit `paths:` in the frontmatter.
  </ProviderTab>
</ProviderTabs>

> [!WARNING]
> **Copilot**: All rules are merged into a single `.github/copilot-instructions.md` file. Path-scoped rules (`activation: path-glob`) are not natively enforced by Copilot — the path constraint is included as a markdown comment for informational purposes only.
>
> **Gemini**: Path-scoped rules are implemented via Gemini's JIT scoping — nested `GEMINI.md` files are placed in each directory matching the glob. This requires the compiler to know the directory structure of your project at compile time.
