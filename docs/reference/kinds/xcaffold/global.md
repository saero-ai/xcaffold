---
title: "kind: global"
description: "Root manifest for user-wide global scope. Declares settings and hooks that apply to all projects."
---

# `kind: global`

The root manifest for user-wide global scope. Lives at `~/.xcaffold/xcaf/global.xcaf` and declares settings, hooks, and inline resources that apply to all projects on the machine.

Resources defined in global scope are **implicitly inherited** by every project. There is no need to add `extends: global` to a project's `project.xcaf` — the compiler calls `loadGlobalBase()` automatically during parsing and merges global resources into the project config with `Inherited = true`.

`xcaffold apply --global` compiles global scope resources and writes output to user-wide provider directories (`~/.claude/`, `~/.gemini/`, etc.), independent of any project. `xcaffold validate --global` validates the global config independently — no project root is required.

Uses **pure YAML format** (no frontmatter `---` delimiters).

> **Required:** `kind`, `version`

## Source Location

```
~/.xcaffold/xcaf/global.xcaf
```

The global config is a single file. Agents, skills, rules, and other resources in global scope live as separate files under `~/.xcaffold/xcaf/` using the standard directory-per-resource layout:

```
~/.xcaffold/
  xcaf/
    global.xcaf                Root global manifest
    agents/
      my-agent/
        my-agent.xcaf          Global agent definition
    skills/
      my-skill/
        my-skill.xcaf          Global skill definition
    rules/
      my-rule/
        my-rule.xcaf           Global rule definition
  state/                       Compilation state for global scope
```

## Example Usage

### Minimal global config

`~/.xcaffold/xcaf/global.xcaf`:
```yaml
kind: global
version: "1.0"
```

This is what `xcaffold init --global` creates. All projects on the machine inherit any resources placed under `~/.xcaffold/xcaf/` as separate files.

### Global config with settings and hooks

```yaml
kind: global
version: "1.0"

settings:
  model: claude-sonnet-4-5
  permissions:
    allow:
      - "Bash(go test ./...)"
    default-mode: default

hooks:
  PreToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: echo "global pre-tool-use"
          timeout: 10
```

Settings and hooks declared here apply to all projects. Project-level settings and hooks take precedence when both are defined.

## Field Reference

### Required Fields

None beyond the envelope fields `kind` and `version`.

### Optional Fields

| Field | Type | Description |
|:------|:-----|:------------|
| `extends` | `string` | Path to a parent config to inherit from. Used for explicit config chaining (separate from the implicit global inheritance). |
| `settings` | `SettingsConfig` | Settings block (model, permissions, shell, etc.). See [`kind: settings`](../provider/settings). |
| `hooks` | `HookConfig` | Hook definitions keyed by event name (`PreToolUse`, `PostToolUse`, etc.). See [`kind: hooks`](../provider/hooks). |
| `agents` | `map[string]AgentConfig` | Inline agent definitions. Keys are agent IDs. |
| `skills` | `map[string]SkillConfig` | Inline skill definitions. Keys are skill IDs. |
| `rules` | `map[string]RuleConfig` | Inline rule definitions. Keys are rule IDs. |
| `mcp` | `map[string]MCPConfig` | Inline MCP server definitions. Keys are server IDs. |
| `workflows` | `map[string]WorkflowConfig` | Inline workflow definitions. Keys are workflow IDs. |
| `policies` | `map[string]PolicyConfig` | Inline policy definitions. Keys are policy IDs. |
| `contexts` | `map[string]ContextConfig` | Inline context definitions. Keys are context IDs. |
| `memory` | `map[string]MemoryConfig` | Inline memory definitions. Keys are memory IDs. |
| `templates` | `map[string]TemplateConfig` | Inline template definitions. Schema-only — parsed but produces no compilation output. |

Resources can be defined inline in `global.xcaf` using map format, but the recommended pattern is to use separate files under `~/.xcaffold/xcaf/` with the standard directory-per-resource layout.

## Behavior

### Implicit Inheritance

Every project automatically inherits global scope resources. During parsing, the compiler:

1. Calls `loadGlobalBase()` to discover and parse `~/.xcaffold/xcaf/`.
2. Marks all global resources with `Inherited = true`.
3. Merges global resources into the project's compiled resource set.
4. Project-local resources with the same ID override the inherited global resource.

This happens without any `extends:` declaration in `project.xcaf`. The `extends:` field on `globalDocument` is a separate mechanism for chaining one global config to another parent config.

### Global Apply

`xcaffold apply --global` compiles the global scope and writes output to user-wide provider directories. The output directory is derived from `~/.xcaffold/` — for example, global Claude output goes to `~/.claude/`. Global output is independent of any project's output.

If `--target` is specified with `--global`, the renderer must support global scope (`SupportsGlobalScope()`). Renderers that do not support global scope produce an error.

### Global Validate

`xcaffold validate --global` validates the global config at `~/.xcaffold/xcaf/` independently. No project root is required — the parse root is set to `~/.xcaffold/xcaf/` directly.

## Notes

- `xcaffold init --global` bootstraps the `~/.xcaffold/xcaf/` directory with a starter `global.xcaf` and subdirectories for agents, skills, and rules. It is idempotent.
- The `~/.xcaffold/` directory itself (and `registry.xcaf`) is created by `EnsureGlobalHome()` on the first run of any xcaffold command, not specifically by `init --global`.
