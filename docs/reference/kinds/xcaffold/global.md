---
title: "kind: global"
description: "Declares shared resource definitions inherited by all projects that extend this base config. Produces no output files."
---

# `kind: global`

Declares shared resource definitions inherited by all projects via the `extends:` field in `project.xcaf`. Global configs serve as a base layer — any resource (agent, skill, rule, MCP server, settings, hooks) declared in a global config is available to inheriting projects without re-declaring it.

Global configs produce **no output files** on their own. Output is produced only when a project config compiles with `extends:` pointing to this global.

Uses **pure YAML format** (no frontmatter `---` delimiters).

> **Required:** `kind`, `version`

## Example Usage

### Shared organization baseline

`xcaf/global/org-baseline.xcaf`:
```yaml
kind: global
version: "1.0"

extends: ""

settings:
  default:
    model: claude-sonnet-4-5
    effort-level: high
    language: en
    output-style: concise
    default-shell: /bin/zsh
    include-git-instructions: true
    permissions:
      allow:
        - "Bash(go test ./...)"
      deny:
        - "Bash(rm -rf /)"
      default-mode: default

hooks:
  PreToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: echo "global pre-tool-use"
          async: false
          timeout: 10

agents:
  code-reviewer:
    name: code-reviewer
    description: "Reviews code for correctness and style."
    model: claude-sonnet-4-5
    tools: [Read, Glob, Grep]

skills:
  conventional-commits:
    name: conventional-commits
    description: "Enforces conventional commit message format."
    allowed-tools: [Bash]

rules:
  no-secrets-in-code:
    name: no-secrets-in-code
    description: "Prevents secrets from appearing in source code."
    always-apply: true

mcp:
  global-ref-mcp:
    name: global-ref-mcp
    type: stdio
    command: node
    args: [./global-server.js]
```

Project `project.xcaf` inheriting the global:
```yaml
kind: project
version: "1.0"

project:
  name: frontend-app
  extends: xcaf/global/org-baseline.xcaf
  targets:
    - claude
    - cursor
    - gemini

agents:
  react-developer:
    name: react-developer
    description: "React specialist for frontend tasks."
    model: claude-sonnet-4-5
    tools: [Read, Write, Edit, Bash]

skills:
  component-patterns:
    name: component-patterns
    description: "React component design patterns."
    allowed-tools: [Read]
```

The compiled output for `frontend-app` includes both project-local resources (`react-developer`, `component-patterns`) and the inherited global resources (`code-reviewer`, `conventional-commits`, `no-secrets-in-code`).

## Argument Reference

| Field | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `kind` | `string` | Yes | Must be `global`. |
| `version` | `string` | Yes | Schema version. Use `"1.0"`. |
| `extends` | `string` | No | Path to a parent global config to inherit from. |
| `settings` | `map[string]SettingsConfig` | No | Named settings blocks. Keys are settings IDs. |
| `hooks` | `map[string]NamedHookConfig` | No | Named hook config blocks. Keys are hook set IDs. |
| `agents` | `map[string]AgentConfig` | No | Inline agent definitions. Keys are agent IDs. |
| `skills` | `map[string]SkillConfig` | No | Inline skill definitions. Keys are skill IDs. |
| `rules` | `map[string]RuleConfig` | No | Inline rule definitions. Keys are rule IDs. |
| `mcp` | `map[string]MCPConfig` | No | Inline MCP server definitions. Keys are server IDs. |
| `workflows` | `map[string]WorkflowConfig` | No | Inline workflow definitions. Keys are workflow IDs. |
| `policies` | `map[string]PolicyConfig` | No | Inline policy definitions. Keys are policy IDs. |
| `memory` | `map[string]MemoryConfig` | No | Inline memory definitions. Keys are memory IDs. |
| `contexts` | `map[string]ContextConfig` | No | Inline context definitions. Keys are context IDs. |

Resource fields use **inline map format** — each key in the map is the resource ID, and its value is the full resource config object. This differs from `kind: blueprint`, which uses string ID arrays for selection.

## Behavior

When a project declares `extends:`, the compiler:

1. Parses the global config at the referenced path.
2. Marks all global resources with `Inherited = true`.
3. Merges global resources into the project's compiled resource set.
4. Strips inherited resources from local project file output (they are not re-emitted as new `.xcaf` files).

Inherited resources can be overridden by re-declaring the same ID in the project config. The local declaration wins.

> [!WARNING]
> Global configs are not validated in isolation — `xcaffold validate` always requires a project root. Run `xcaffold validate --from-root <project-dir>` to validate a project that extends a global.
