---
title: "kind: agent"
description: "Defines a named AI persona. Source: xcf/agents/<name>/agent.xcf. Compiled to agents/<name>.md per provider."
---

# `kind: agent`

Defines an AI persona with a system prompt, tool access, and optional skill, rule, and MCP bindings. Compiled to `agents/<id>.md` with YAML frontmatter for each supported target provider.

> **Required:** `kind`, `version`, `name`

## Example Usage

### Minimal agent

```yaml
---
kind: agent
version: "1.0"
name: reviewer
description: Reviews pull requests for correctness and style.
tools: [Read, Glob, Grep]
readonly: true
---
# Reviewer

Review the provided diff for correctness, style, and test coverage.
Flag issues with a severity: `CRITICAL`, `WARN`, or `INFO`.
Report: "Reviewed <N> files, found <M> issues."
```

### Full agent — React developer

```yaml
---
kind: agent
version: "1.0"
name: react-developer
description: >-
  Implements React components, hooks, and UI features for frontend-app.
  Authorized to modify src/components/, src/hooks/, src/pages/, and
  src/styles/. Consults the component-patterns skill before authoring
  any new component.
model: sonnet
tools:
  - Read
  - Edit
  - Write
  - Bash
  - Glob
  - Grep
memory:
  - user-prefs
  - project-context
skills:
  - component-patterns
rules:
  - react-conventions
mcp:
  - browser-tools
---
# React Developer

## Role
Implements and maintains React UI components for `frontend-app`.
You are the ONLY agent authorized to modify `src/components/`.

## Stack
- React 18, TypeScript 5.x, Next.js 15 (App Router)
- Tailwind CSS v4, Shadcn/UI
- Vitest + React Testing Library, Storybook

## Component Rules
- **Co-locate tests**: every component gets a `<Component>.test.tsx` beside it.
- **No inline styles**: Tailwind utility classes only.
- **Export from barrel**: add new components to `src/components/index.ts`.
- **Accessibility first**: all interactive elements need `aria-label`.

## Mandatory Deliverables
1. Run `pnpm test --filter frontend-app` — all tests must pass.
2. Run `pnpm lint` — zero ESLint errors.
3. Report: "Added <N> components, <M> tests passing."

## DO NOT
- Modify files outside `src/components/`, `src/hooks/`, `src/pages/`, `src/styles/`.
- Use class-based components.
- Leave `any` types in props.
```

## Argument Reference

The following arguments are supported:

- `name` — (Required) Unique agent identifier. Must match `[a-z0-9-]+`.
- `description` — (Optional) `string`. Purpose and scope description. Used for delegation and auto-invocation.
- `model` — (Optional) `string`. LLM model identifier. Supports aliases: `sonnet` → `claude-sonnet-4-5-20250514`, `opus` → `claude-opus-4-5-20250514`, `haiku` → `claude-haiku-4-5-20251001`. Resolved per-provider at compile time.
- `effort` — (Optional) `string`. Resource utilization level: `"low"`, `"medium"`, `"high"`, `"max"`.
- `max-turns` — (Optional) `int`. Maximum conversation turns before the agent stops.
- `mode` — (Optional) `string`. Agent execution mode reserved for provider-native use.
- `tools` — (Optional) `[]string`. Runtime tools granted to the agent: `Read`, `Edit`, `Write`, `Bash`, `Glob`, `Grep`, `WebSearch`, etc. Omit to inherit all available tools.
- `disallowed-tools` — (Optional) `[]string`. Tools explicitly forbidden. Applied before `tools` resolution.
- `readonly` — (Optional) `bool`. When `true` and `tools` is unset, Claude emits `tools: [Read, Grep, Glob]`; Cursor emits `readonly: true`. Mutually exclusive with `tools`.
- `permission-mode` — (Optional) `string`. Default execution permission mode: `default`, `acceptEdits`, `auto`, `bypassPermissions`, `plan`.
- `disable-model-invocation` — (Optional) `bool`. When `true`, the host model cannot auto-invoke this agent.
- `user-invocable` — (Optional) `bool`. When `false`, the agent is only accessible programmatically.
- `background` — (Optional) `bool`. Executes without blocking the UI. Claude emits `background`; Cursor emits `is_background`.
- `isolation` — (Optional) `string`. Worktree or environment isolation preference.
- `when` — (Optional) `string`. Compile-time conditional for inclusion.
- `memory` — (Optional) `[]string`. Agent memory references. A single string value is accepted for backward compatibility. See [`kind: memory`](./memory).
- `color` — (Optional) `string`. Terminal UI color attribute (Claude-specific).
- `initial-prompt` — (Optional) `string`. Default message auto-submitted as the first turn.
- `skills` — (Optional) `[]string`. Skill IDs to grant. Must match top-level `skills:` map keys.
- `rules` — (Optional) `[]string`. Rule IDs to enforce. Must match top-level `rules:` map keys.
- `mcp` — (Optional) `[]string`. MCP server IDs to load. Must match top-level `mcp:` map keys.
- `assertions` — (Optional) `[]string`. Behavioral constraints evaluated by `xcaffold test --judge`.
- `targets` — (Optional) `map[string]TargetOverride`. Per-provider overrides. Keys: `claude`, `cursor`, `copilot`, `gemini`, `antigravity`.

### `hooks` block

Agent-scoped lifecycle hooks. Same structure as [`kind: hooks`](/docs/cli/reference/kinds/xcaffold/hooks).

- `pre-tool-call` — Scripts run before any tool invocation.
- `post-tool-call` — Scripts run after any tool invocation.

### `mcp-servers` block

Agent-scoped MCP server definitions. Not merged with project-level `mcp:`. Same field schema as [`kind: mcp`](./mcp).

### `targets` block

The `targets:` field serves two purposes:
1. **Compilation filtering**: When the `targets` key is present on a resource (e.g., `targets: [claude, gemini]` at the top level), the resource is compiled only for listed providers. When absent, the resource is universal.
2. **Provider overrides**: The `targets:` map under the agent definition holds provider-native pass-through fields via `TargetOverride`.

Each key under `agents.<id>.targets.<target>` maps to a `TargetOverride` struct. Valid target keys are: `claude`, `cursor`, `antigravity`, `copilot`, `gemini`.

| Field | Type | Status |
|---|---|---|
| `suppress-fidelity-warnings` | `*bool` | Parsed, not compiled |
| `hooks` | `map[string]string` | Parsed, not compiled |
| `skip-synthesis` | `*bool` | Parsed, not compiled |
| `instructions-override` | `string` | Parsed, not compiled |

## Filesystem-as-Schema

When an agent `.xcf` file lives at `xcf/agents/<name>/agent.xcf`, the `kind:` and `name:` fields can be omitted from the YAML. The parser infers:
- `kind: agent` from the parent directory name (`agents/`)
- `name:` from the grandparent directory name (e.g., `researcher` from `agents/researcher/agent.xcf`)

When `kind:` or `name:` are present in the YAML, they must match the inferred values.

## Compiled Output

<ProviderTabs>
  <ProviderTab id="claude">
    **Output path**: `.claude/agents/react-developer.md`

    ```yaml
    ---
    name: react-developer
    description: >-
      Implements React components, hooks, and UI features for frontend-app.
      Authorized to modify src/components/, src/hooks/, src/pages/, and
      src/styles/. Consults the component-patterns skill before authoring
      any new component.
    model: claude-sonnet-4-5-20250514
    tools: [Read, Edit, Write, Bash, Glob, Grep]
    memory: project
    skills: [component-patterns]
    ---
    # React Developer

    ## Role
    Implements and maintains React UI components for `frontend-app`.
    …
    ```

    All supported fields are emitted. The `model` alias (`sonnet`) is resolved to the full model ID.
  </ProviderTab>

  <ProviderTab id="cursor">
    **Output path**: `.cursor/agents/react-developer.md`

    ```yaml
    ---
    name: react-developer
    description: >-
      Implements React components, hooks, and UI features for frontend-app.
      Authorized to modify src/components/, src/hooks/, src/pages/, and
      src/styles/. Consults the component-patterns skill before authoring
      any new component.
    model: claude-sonnet-4-5-20250514
    ---
    # React Developer

    ## Role
    …
    ```

    > `tools`, `memory`, `skills`, `rules`, `mcp`, `effort`, `permission-mode`, `color`, `initial-prompt`, `hooks`, `mcp-servers` are dropped. Only `name`, `description`, `model`, `readonly`, and `background` (→ `is_background`) are emitted.
  </ProviderTab>

  <ProviderTab id="copilot">
    **Output path**: `.github/agents/react-developer.agent.md`

    ```yaml
    ---
    name: react-developer
    description: >-
      Implements React components, hooks, and UI features for frontend-app.
      Authorized to modify src/components/, src/hooks/, src/pages/, and
      src/styles/. Consults the component-patterns skill before authoring
      any new component.
    model: claude-sonnet-4-5-20250514
    tools: [Read, Edit, Write, Bash, Glob, Grep]
    ---
    # React Developer

    ## Role
    …
    ```

    > `memory`, `skills`, `rules`, `mcp` are dropped. `name`, `description`, `model`, `tools` are emitted.
  </ProviderTab>

  <ProviderTab id="gemini">
    **Output path**: `.gemini/agents/react-developer.md`

    ```yaml
    ---
    name: react-developer
    description: >-
      Implements React components, hooks, and UI features for frontend-app.
      Authorized to modify src/components/, src/hooks/, src/pages/, and
      src/styles/. Consults the component-patterns skill before authoring
      any new component.
    ---
    # React Developer

    ## Role
    …
    ```

    > Only `name` and `description` are emitted. `model`, `tools`, `memory`, `skills`, `rules`, `mcp` are all dropped.
  </ProviderTab>

  <ProviderTab id="antigravity">
    > **Target Skipped**: Antigravity has no file-based agent configuration. Agent behavior is controlled entirely via UI settings. `AGENT_NO_NATIVE_TARGET` fidelity note emitted to stderr.
  </ProviderTab>
</ProviderTabs>

> [!WARNING]
> **Cursor**: Drops all fields except `name`, `description`, `model`, `readonly`, and `background`. Unmapped `model` values emit a stderr warning and are omitted from output.
>
> **Antigravity**: Agent compilation is skipped entirely. No files are written for this target.
