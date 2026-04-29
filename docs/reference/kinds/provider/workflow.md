---
title: "kind: workflow"
description: "Defines a named, multi-step procedure compiled to workflows/<id>/WORKFLOW.md for Antigravity. All other providers silently skip this kind."
---

# `kind: workflow`

Defines a named, reusable multi-step procedure. Compiled exclusively to `.agents/workflows/<id>/WORKFLOW.md` for the **Antigravity** provider.

> **Required:** `kind`, `version`, `name`
>
> **Antigravity-only.** Claude, Cursor, Copilot, and Gemini silently ignore workflow definitions. Declaring a workflow with those providers in `targets` produces no output and no error.

## Source Directory

```
xcf/workflows/<name>/workflow.xcf
```

Each workflow is a directory-per-resource under `xcf/workflows/`. The `.xcf` file contains the frontmatter and optional body for that workflow.

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
---
```
```yaml
steps:
  - name: implement
    description: Build the component using the component-patterns skill.
    instructions: |
      1. Create a git worktree: `git worktree add .worktrees/feat-<name> feat/<name>`
      2. Invoke the component-patterns skill to scaffold the component.
      3. Run `pnpm test --filter frontend-app` — all tests must pass.

  - name: review
    description: Self-review for convention compliance.
    instructions: |
      1. Run `pnpm lint` — zero ESLint errors.
      2. Check every new component has a co-located `.test.tsx`.
      3. Confirm no `any` types appear in props.

  - name: pr
    description: Open a pull request.
    instructions: |
      1. Commit with: `feat(ui): add <ComponentName> component`
      2. Push the branch and open a PR targeting `main`.
      3. Include a Storybook screenshot in the PR description.
```

## Argument Reference

The following arguments are supported:

- `name` — (Required) Unique workflow identifier. Must match `[a-z0-9-]+`.
- `api-version` — (Optional) `string`. Schema discriminator. Default: `"workflow/v1"`.
- `description` — (Optional) `string`. What this workflow accomplishes.
- `steps` — (Optional) `[]WorkflowStep`. Ordered procedural steps. Use when the workflow has multiple phases. Mutually exclusive with a top-level instruction body.
- `targets` — (Optional) `map[string]TargetOverride`. Per-provider overrides. Resources with a `targets:` field are compiled only for the listed providers. When absent, the resource is compiled for all applicable providers.

### `steps` entry

Each entry in `steps` supports:

- `name` — (Required) Step identifier used as a heading in the compiled output.
- `description` — (Optional) `string`. One-line step summary.

The step content is provided as the markdown body after the frontmatter `---` (for single-step) or as the `instructions` inline value for multi-step definitions.

## Compiled Output

<ProviderTabs>
  <ProviderTab id="claude">
    > **Target Skipped**: Claude Code has no native workflow format. No files are written for this provider.
  </ProviderTab>

  <ProviderTab id="cursor">
    > **Target Skipped**: Cursor has no native workflow format. No files are written for this provider.
  </ProviderTab>

  <ProviderTab id="copilot">
    > **Target Skipped**: GitHub Copilot has no native workflow format. No files are written for this provider.
  </ProviderTab>

  <ProviderTab id="gemini">
    > **Target Skipped**: Gemini CLI has no native workflow format. No files are written for this provider.
  </ProviderTab>

  <ProviderTab id="antigravity">
    **Output path**: `.agents/workflows/ship-feature/WORKFLOW.md`

    ```markdown
    ---
    description: "End-to-end workflow for shipping a new React component from branch to PR."
    ---

    ### Step 1: implement

    Build the component using the component-patterns skill.

    1. Create a git worktree: `git worktree add .worktrees/feat-<name> feat/<name>`
    2. Invoke the component-patterns skill to scaffold the component.
    3. Run `pnpm test --filter frontend-app` — all tests must pass.

    ### Step 2: review

    Self-review for convention compliance.

    1. Run `pnpm lint` — zero ESLint errors.
    2. Check every new component has a co-located `.test.tsx`.
    3. Confirm no `any` types appear in props.

    ### Step 3: pr

    Open a pull request.

    1. Commit with: `feat(ui): add <ComponentName> component`
    2. Push the branch and open a PR targeting `main`.
    3. Include a Storybook screenshot in the PR description.
    ```

    Antigravity flattens `steps` into sequentially numbered `### Step N: <name>` headings. The `description` is placed as a subheading beneath each step heading.
  </ProviderTab>
</ProviderTabs>

> [!IMPORTANT]
> Workflows are an Antigravity-exclusive provider kind. All other providers silently drop the resource with no diagnostic. If you need multi-step procedures that work across providers, use a `kind: skill` with numbered sections instead.
