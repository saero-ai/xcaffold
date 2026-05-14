---
title: "kind: agent"
description: "Defines a named subagent/specialist delegated to by the main AI session. Source: xcaf/agents/<name>/agent.xcaf. Compiled to agents/<name>.md per provider."
---

# `kind: agent`

Defines a named subagent that the main AI session delegates to. Each agent carries its own system prompt, tool access, and optional skill, rule, and MCP bindings. xcaffold compiles it to `agents/<id>.md` with YAML frontmatter for each supported target provider.

In every provider, the compiled output lands in that provider's agents directory — `.claude/agents/`, `.cursor/agents/`, `.github/agents/`, `.gemini/agents/` — where the main session picks it up and dispatches to it by name.

> **Required:** `kind`, `version`, `name`

## Source Directory

```
xcaf/agents/<name>/agent.xcaf
```

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

## Field Reference

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | `string` | Unique agent identifier. Must match `[a-z0-9-]+`. |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `description` | `string` | Human-readable purpose of this agent. Used for delegation and auto-invocation. |
| `model` | `string` | LLM model identifier or alias resolved at compile time. Bare tier aliases (`sonnet`, `opus`, `haiku`) are passed through for runtime resolution. |
| `effort` | `string` | Reasoning effort level hint for the model provider. |
| `max-turns` | `int` | Maximum conversation turns before the agent exits. |
| `tools` | `[]string` | Ordered list of tools this agent may invoke. Omit to inherit all available tools. |
| `disallowed-tools` | `[]string` | Tools explicitly denied to this agent. |
| `readonly` | `bool` | When `true`, restricts the agent to read-only tool access. |
| `permission-mode` | `string` | Security mode controlling tool authorization behavior. |
| `disable-model-invocation` | `bool` | Prevents the agent from spawning sub-agents. |
| `user-invocable` | `bool` | Whether users can invoke this agent directly via slash command. |
| `background` | `bool` | Runs the agent in background mode without interactive prompts. |
| `isolation` | `string` | Process isolation level for the agent session. |
| `memory` | `string` or `[]string` | Named memory banks attached to this agent. A single string value is accepted for backward compatibility. |
| `color` | `string` | Display color for terminal output differentiation. |
| `initial-prompt` | `string` | System prompt prepended to every conversation. |
| `skills` | `[]string` | Skill resource IDs attached to this agent. Must match top-level `skills:` map keys. |
| `rules` | `[]string` | Rule resource IDs governing this agent. Must match top-level `rules:` map keys. |
| `mcp` | `[]string` | MCP server resource IDs available to this agent. Must match top-level `mcp:` map keys. |
| `mcp-servers` | `map[string]MCPConfig` | Inline MCP server definitions keyed by server name. Same field schema as [`kind: mcp`](./mcp). |
| `hooks` | `HookConfig` | Agent-scoped lifecycle hooks. Same structure as [`kind: hooks`](./hooks). |
| `targets` | `map[string]TargetOverride` | Per-provider overrides keyed by provider name (`claude`, `cursor`, `copilot`, `gemini`, `antigravity`). |

### `hooks` block

Agent-scoped lifecycle hooks. Same structure as [`kind: hooks`](hooks.md).

- `PreToolUse` — Runs before any tool invocation.
- `PostToolUse` — Runs after any tool invocation.
- `SessionStart`, `Stop`, `Notification`, `SubagentStop`, `InstructionsLoaded`, `PreCompact`, `ConfigChange` — Session lifecycle events.

### `mcp-servers` block

Agent-scoped MCP server definitions. Not merged with project-level `mcp:`. Same field schema as [`kind: mcp`](./mcp).

### `targets` block

Per-provider overrides keyed by provider name. When present, the resource compiles only for listed providers. See [Targets](../../../concepts/configuration/targets.md) for the full concept, including filtering semantics and the dual-purpose map syntax.

Each key under `agents.<id>.targets.<target>` maps to a `TargetOverride` struct. Valid target keys are: `claude`, `cursor`, `antigravity`, `copilot`, `gemini`.

| Field | Type | Status |
|---|---|---|
| `suppress-fidelity-warnings` | `*bool` | Parsed, not compiled |
| `hooks` | `map[string]string` | Parsed, not compiled |
| `skip-synthesis` | `*bool` | Parsed, not compiled |

## Provider Fidelity

Not all fields survive compilation to every provider. The table below shows which xcaffold fields are emitted for each target. Fields marked `—` are dropped silently unless noted.

| Field | Claude | Cursor | Copilot | Gemini | Antigravity |
|-------|--------|--------|---------|--------|-------------|
| `name` | yes | yes | yes | yes | body only |
| `description` | yes | yes | yes | yes | body only |
| `model` | yes (resolved) | mapped only ¹ | yes (resolved) | yes (resolved) | body only |
| `effort` | yes | — | — | — | — |
| `max-turns` | yes | — | — | `max_turns` ² | — |
| `tools` | yes (inline) | — | yes (YAML list) | yes (YAML list) | — |
| `disallowed-tools` | yes | — | — | — | — |
| `readonly` | transforms ³ | yes | — | — | — |
| `permission-mode` | yes | — | — | — | — |
| `disable-model-invocation` | — | — | yes | — | — |
| `user-invocable` | — | — | yes | — | — |
| `background` | yes | `is_background` ² | — | — | — |
| `isolation` | yes | — | — | — | — |
| `memory` | yes | — | — | — | — |
| `color` | yes | — | — | — | — |
| `initial-prompt` | yes | — | — | — | — |
| `skills` | yes (inline) | — | — | — | — |
| `hooks` | yes (YAML) | — | — | — | — |
| `mcp-servers` | yes (YAML) | — | yes (YAML) | `mcpServers` ² | — |

**Footnotes**

1. Cursor emits `model` only when `IsMappedModel` returns true. Unmapped values are dropped with an `AGENT_MODEL_UNMAPPED` warning to stderr.
2. Key name differs from the xcaffold canonical: `max_turns` (snake_case), `is_background`, `mcpServers` (camelCase).
3. Claude: when `readonly: true` and no `tools` are specified, emits `tools: [Read, Grep, Glob]` rather than a `readonly` key.

## Filesystem-as-Schema

When an agent `.xcaf` file lives at `xcaf/agents/<name>/agent.xcaf`, the `kind:` and `name:` fields can be omitted from the YAML. The parser infers:
- `kind: agent` from the parent directory name (`agents/`)
- `name:` from the grandparent directory name (e.g., `researcher` from `agents/researcher/agent.xcaf`)

When `kind:` or `name:` are present in the YAML, they must match the inferred values.

## Compiled Output

### Claude

**Output path**: `.claude/agents/react-developer.md`

```yaml
---
name: react-developer
description: >-
  Implements React components, hooks, and UI features for frontend-app.
  Authorized to modify src/components/, src/hooks/, src/pages/, and
  src/styles/. Consults the component-patterns skill before authoring
  any new component.
model: claude-sonnet-4-5
tools: [Read, Edit, Write, Bash, Glob, Grep]
memory: project
skills: [component-patterns]
---
# React Developer

## Role
Implements and maintains React UI components for `frontend-app`.
…
```

All supported fields are emitted. The `model` alias (`sonnet`) is resolved to the full model ID. `skills` is inlined. `disable-model-invocation` and `user-invocable` are not Claude fields and are not emitted.

### Cursor

**Output path**: `.cursor/agents/react-developer.md`

```yaml
---
name: react-developer
description: >-
  Implements React components, hooks, and UI features for frontend-app.
  Authorized to modify src/components/, src/hooks/, src/pages/, and
  src/styles/. Consults the component-patterns skill before authoring
  any new component.
readonly: true
---
# React Developer

## Role
…
```

> Cursor emits `name`, `description`, `model` (mapped models only — unmapped values dropped with `AGENT_MODEL_UNMAPPED` warning), `readonly`, and `background` (renamed to `is_background`). All other fields — `tools`, `memory`, `skills`, `rules`, `mcp`, `effort`, `permission-mode`, `color`, `initial-prompt`, `hooks`, `mcp-servers` — are dropped.

### Copilot

**Output path**: `.github/agents/react-developer.agent.md`

```yaml
---
name: react-developer
description: >-
  Implements React components, hooks, and UI features for frontend-app.
  Authorized to modify src/components/, src/hooks/, src/pages/, and
  src/styles/. Consults the component-patterns skill before authoring
  any new component.
model: claude-sonnet-4-6
tools:
  - Read
  - Edit
  - Write
  - Bash
  - Glob
  - Grep
---
# React Developer

## Role
…
```

> Copilot emits `name`, `description`, `model` (resolved), `tools` (YAML list), `disable-model-invocation`, `user-invocable`, and `mcp-servers`. Fields `memory`, `skills`, `rules`, `mcp`, `effort`, `permission-mode`, `readonly`, `background`, `isolation`, `color`, `initial-prompt`, and `hooks` are dropped.
>
> **Native passthrough**: when `.claude/agents/` is present in the project, Copilot skips compilation for agents and reads the Claude output directly (`CLAUDE_NATIVE_PASSTHROUGH`).

### Gemini

**Output path**: `.gemini/agents/react-developer.md`

```yaml
---
name: react-developer
description: >-
  Implements React components, hooks, and UI features for frontend-app.
  Authorized to modify src/components/, src/hooks/, src/pages/, and
  src/styles/. Consults the component-patterns skill before authoring
  any new component.
model: gemini-2.5-flash
tools:
  - Read
  - Edit
  - Write
  - Bash
  - Glob
  - Grep
max_turns: 10
mcpServers:
  browser-tools:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-browser-tools"]
---
# React Developer

## Role
…
```

> Gemini emits `name`, `description`, `model` (resolved), `tools` (YAML list), `max_turns` (snake_case, from `max-turns`), and `mcpServers` (camelCase, from `mcp-servers`). Fields `memory`, `skills`, `rules`, `mcp`, `effort`, `permission-mode`, `readonly`, `background`, `isolation`, `color`, `initial-prompt`, and `hooks` are dropped.

### Antigravity

**Output path**: `.agents/agents/react-developer.md`

Agent compiled as a specialist note (persona profile). Fidelity note `RENDERER_KIND_DOWNGRADED` emitted to stderr indicating that Antigravity does not support native agent definitions. `name`, `description`, `model`, and body content are folded into the note body; no frontmatter fields are emitted.
