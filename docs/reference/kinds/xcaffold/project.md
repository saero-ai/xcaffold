---
title: "kind: project"
description: "Root manifest that declares compilation targets and references all named xcaf resources. One per project."
---

# `kind: project`

The root manifest for an xcaffold project. Declares which providers to target, references all named resources (agents, skills, rules, MCP servers, policies), and configures project-wide settings.

There is **exactly one** project manifest per project, located at `project.xcaf` (at the repository root). Produces no provider output files — the manifest drives the compilation pipeline.

> [!IMPORTANT]
> The `kind: project` manifest does **not** support a markdown body. Workspace-level instructions must be declared using [`kind: context`](../provider/context). Adding a body after the closing `---` in a project manifest will cause a parse error.

> **Required:** `kind`, `version`, `name`, \`targets\`

## Example Usage

### Minimal project

```yaml
---
kind: project
version: "1.0"
name: frontend-app
targets:
  - claude
  - cursor
---
```

### Full project manifest — React TypeScript SaaS

```yaml
---
kind: project
version: "1.0"
name: frontend-app
description: >-
  React TypeScript SaaS frontend built with Next.js 15, Tailwind CSS v4, and
  Shadcn/UI. Deployed on Vercel. All components use the design system defined
  in src/components/ui/.
author: Acme Corp
repository: https://github.com/acme/frontend-app
targets:
  - claude
  - cursor
  - gemini
  - copilot
  - antigravity
test:
  cli-path: /usr/local/bin/claude
  judge-model: claude-opus-4-5
  task: "Demonstrate all capabilities and confirm every feature works end-to-end."
  max-turns: 10
target-options:
  copilot:
    hooks:
      copilot-instructions: ".copilot/instructions.md"
  cursor:
    suppress-fidelity-warnings: false
---
```

## Argument Reference

The following arguments are supported:

- `name` — (Required) Unique project identifier. Must match `[a-z0-9-]+`.
- `version` — (Required) Schema version. Use `"1.0"`.
- `description` — (Optional) `string`. Human-readable project description.
- `author` — (Optional) `string`. Project author or organization name.
- `homepage` — (Optional) `string`. Project URL.
- `repository` — (Optional) `string`. Source repository URL.
- `license` — (Optional) `string`. SPDX license identifier.
- `backup-dir` — (Optional) `string`. Directory where provider output is backed up before each overwrite.
- `allowed-env-vars` — (Optional) `[]string`. Environment variable names that may be injected via `${env.NAME}` inside `.xcaf` files. Variables not listed here are rejected at compile time to prevent accidental secret leakage. See [Project Variables](../../../concepts/configuration/variables.md) for details.
- `targets` — (Required) `[]string`. Provider targets to compile: `claude`, `cursor`, `gemini`, `copilot`, `antigravity`. At least one required.
- `agents` — (Optional) `map[string]AgentConfig`. Named agents defined inline (see [agents block](#agents-block)). Filesystem-discovered agents under `xcaf/agents/` are merged automatically and do not require inline declaration.
- `skills` — (Optional) `map[string]SkillConfig`. Named skills defined inline or discovered under `xcaf/skills/`.
- `rules` — (Optional) `map[string]RuleConfig`. Named rules defined inline or discovered under `xcaf/rules/`.
- `mcp` — (Optional) `map[string]MCPConfig`. Named MCP server declarations.
- `policies` — (Optional) `map[string]PolicyConfig`. Named compile-time policy constraints.
- `test` — (Optional) `TestConfig`. Configuration for `xcaffold test` (see [test block](#test-block)).
- `local` — (Optional) `SettingsConfig`. Local settings override applied to this project only (see [local block](#local-block)).
- `target-options` — (Optional) `map[string]TargetOverride`. Per-provider compile-time overrides (see [target-options block](#target-options-block)).

### `agents` block

Agents can be declared inline in the project manifest or discovered automatically from `xcaf/agents/<name>/agent.xcaf`. Both forms are merged during parsing.

Inline declaration uses a map keyed by agent name:

```yaml
agents:
  react-developer:
    description: "React and TypeScript specialist."
    model: sonnet
    tools: [Read, Write, Edit, Bash]
```

Filesystem-discovered agents under `xcaf/agents/<name>/agent.xcaf` require no explicit entry — the parser finds and merges them automatically.

> **Note:** The project manifest does not support a `path:` reference syntax. Agent files are either declared inline or discovered by directory convention.

### `test` block

Configures the `xcaffold test` command. All fields are optional.

- `cli-path` — `string`. Path to the CLI binary used for simulation (e.g., `/usr/local/bin/claude`).
- `judge-model` — `string`. Generative model used for LLM-as-a-Judge evaluation of test output.
- `task` — `string`. User prompt sent to the agent during test simulation. Defaults to a generic capability discovery prompt when empty.
- `max-turns` — `int`. Maximum simulated conversation turns. Reserved for future multi-turn support.

### `local` block

A `SettingsConfig` block applied only to this project. Fields mirror those of a named `settings` resource. Use `local:` to apply project-specific provider settings without creating a named settings resource. See [kind: settings](./settings) for the full field reference.

### `target-options` block

Per-provider compile-time overrides keyed by provider name. Each value is a `TargetOverride` with the following fields:

- `hooks` — `map[string]string`. Provider-specific hook path overrides.
- `suppress-fidelity-warnings` — `bool`. Suppress fidelity notes for this provider during `apply`.
- `skip-synthesis` — `bool`. Skip synthesis passes for this provider.
- `provider` — `map[string]any`. Opaque pass-through map emitted verbatim into the provider's native config.

```yaml
target-options:
  copilot:
    hooks:
      copilot-instructions: ".copilot/instructions.md"
    provider:
      groups:
        - copilot-chat
  cursor:
    suppress-fidelity-warnings: false
    skip-synthesis: false
```

### `targets` on resources vs. project

The `targets:` field appears in two contexts:
- **On the project manifest**: Lists which providers to compile output for. Required.
- **On individual resources**: Controls compilation filtering — a resource with `targets: [claude]` is compiled only for Claude. When absent, the resource is universal (compiled for all project targets).

## Import

```bash
xcaffold import --target claude
```

`xcaffold import` reverse-engineers existing provider directories into `.xcaf` source files and reconstructs a `project.xcaf` with discovered resources. Instructions found in provider root files (e.g. `CLAUDE.md`) are imported as `kind: context` resources.
