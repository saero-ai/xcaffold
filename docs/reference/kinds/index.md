---
title: "Resource Kinds"
description: "All xcaf resource kinds supported by the xcaffold compiler."
---

# Resource Kinds

xcaffold compiles `.xcaf` manifest files into provider-native output directories. Each file declares exactly one resource using a `kind:` field that tells the compiler how to parse and render it.

Kinds fall into two categories:

## Provider Kinds

These kinds compile directly into files inside the provider output directories (`.claude/`, `.cursor/`, `.gemini/`, `.github/`, `.agents/`). Each provider renders the resource in its native format.

| Kind | Compiles to | Description |
|---|---|---|
| [`agent`](./provider/agent) | `agents/<id>.md` per provider | Named AI persona with a system prompt, tools, and optional skill/rule bindings |
| [`skill`](./provider/skill) | `skills/<id>/SKILL.md` per provider | Reusable procedure invoked by agents on-demand |
| [`rule`](./provider/rule) | `rules/<id>.md` per provider | Constraint always active or scoped to specific file paths |
| [`mcp`](./provider/mcp) | Provider-specific JSON config | Model Context Protocol server declaration |
| [`workflow`](./provider/workflow) | `workflows/<id>/WORKFLOW.md` per provider | Multi-step scripted procedure |
| [`context`](./provider/context) | `CLAUDE.md`, `GEMINI.md`, etc. | Workspace-level ambient context |

## Xcaffold Kinds

These kinds govern the compiler itself — they configure compilation targets, enforce policies, and structure the project manifest. They produce **no output files**.

| Kind | Description |
|---|---|
| [`project`](./xcaffold/project) | Root manifest — declares targets, references all resources |
| [`policy`](./xcaffold/policy) | Compile-time constraint evaluated during `apply` and `validate` |
| [`blueprint`](./xcaffold/blueprint) | Named resource subset for conditional compilation |
| [`global`](./xcaffold/global) | Shared resource definitions inherited across the project |

---

## Resource File Format (One Kind Per File)

`xcaffold` enforces a **single-kind-per-file** layout. Each `.xcaf` file declares exactly one `kind:` field. Resources under `xcaf/` are discovered automatically, parsed recursively, and merged prior to compilation. Unknown fields will cause an immediate parse error.

xcaffold leverages two distinct file formats depending on whether a structural resource requires a multi-line textual instruction block.

### Body-Bearing Kinds (Frontmatter Format)

`agent`, `skill`, `rule`, and `workflow` embed their instruction body organically as Markdown appended *after* a standard YAML frontmatter block.

```markdown
---
kind: agent
version: "1.0"
name: developer
model: claude-sonnet-4-6
---
You are a senior backend developer. Focus heavily on code-quality.
```

The `---` delimiters demarcate the boundaries of the frontmatter config. All characters after the closing `---` comprise the instruction body (equivalent to the UI prompting box).

### Structural Config Kinds (Pure YAML)

`policy`, `mcp`, `global`, and `memory` have no concept of open-ended instruction bodies. They use pure YAML format without `---` delimiters.

`project` is the exception: it uses frontmatter+body format with `---` delimiters. The content after the closing `---` is compiled into project-level instructions for each provider's root context file (e.g., `CLAUDE.md`, `AGENTS.md`). The body is optional — a project manifest with no body is valid.

```yaml
---
kind: project
version: "1.0"
name: my-api
targets:
  - claude
---
Optional project-level instructions compiled to CLAUDE.md and other provider root files.
```
