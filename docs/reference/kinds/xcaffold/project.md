---
title: "kind: project"
description: "Root manifest that declares compilation targets and references all named xcf resources. One per project."
---

# `kind: project`

The root manifest for an xcaffold project. Declares which providers to target, references all named resources (agents, skills, rules, MCP servers, policies), and optionally provides project-level instructions compiled to each provider's root instructions file.

There is **exactly one** project manifest per project, located at `.xcaffold/project.xcf`. Produces no provider output files — the manifest drives the compilation pipeline.

> **Required:** `kind`, `version`, `name`, `targets`

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
agents:
  - id: react-developer
    path: xcf/agents/react-developer/react-developer.xcf
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
agents:
  - id: react-developer
    path: xcf/agents/react-developer/react-developer.xcf
skills:
  - component-patterns
rules:
  - react-conventions
  - no-server-imports-in-ui
mcp:
  - browser-tools
policies:
  - require-approved-model
---
This project uses xcaffold to manage AI agent configuration across all providers.
Agents must follow all declared rules at all times.
To add a new UI component, invoke the component-patterns skill before writing any code.
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
- `targets` — (Required) `[]string`. Provider targets to compile: `claude`, `cursor`, `gemini`, `copilot`, `antigravity`. At least one required.
- `agents` — (Optional) `[]AgentManifestEntry`. Agents to include (see [agents block](#agents-block)).
- `skills` — (Optional) `[]string`. Skill IDs declared in `xcf/skills/`.
- `rules` — (Optional) `[]string`. Rule IDs declared in `xcf/rules/`.
- `mcp` — (Optional) `[]string`. MCP server IDs declared in `xcf/mcp/`.
- `policies` — (Optional) `[]string`. Policy IDs declared in `xcf/policies/`.
- `backup-dir` — (Optional) `string`. Directory for provider backup files before overwrite.

### `agents` block

Each entry in the `agents` list supports:

- `id` — (Required) Agent identifier. Must match the `name` in the referenced `.xcf` file.
- `path` — (Required) Relative path to the agent's `.xcf` file from the project root.

## Behavior

The project manifest body (content after the closing `---`) is compiled to project-level instructions:

| Provider | Output path |
|---|---|
| Claude | `CLAUDE.md` (project root) |
| Gemini | `GEMINI.md` (project root, with rule imports) |
| Antigravity | `AGENTS.md` (project root) |
| Cursor | Prepended to `.cursor/rules/project-instructions.md` |
| Copilot | Prepended to `.github/copilot-instructions.md` |

## Import

```bash
xcaffold import --provider claude
```

`xcaffold import` reverse-engineers existing provider directories into `.xcf` source files and reconstructs a `project.xcf` with discovered resources.
