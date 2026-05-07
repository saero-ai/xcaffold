---
title: "kind: memory"
description: "Defines persistent context for an agent persona. Source: `xcaf/agents/<agent-id>/memory/<id>/memory.xcaf`."
---

# `kind: memory`

Defines persistent context or "long-term memory" for a specific agent persona. Memory resources are discovered via directory convention and synthesized into the provider-native agent configuration during compilation.

## Example Usage

```yaml
---
kind: memory
version: "1.0"
name: project-context
description: Architectural overview and core constraints.
---
# Project Context

This project uses a monorepo structure with a Go CLI and a Next.js platform.
All code must adhere to the R1 architectural standards.
```

## Argument Reference

### Required Arguments

| Argument | Type | Description |
| :--- | :--- | :--- |
| `kind` | `string` | Must be `memory`. |
| `version` | `string` | Resource schema version (e.g., `"1.0"`). |
| `name` | `string` | Unique identifier for the memory entry. Must match `^[a-z0-9-]+$`. |

### Optional Arguments

| Argument | Type | Description | Provider Support |
| :--- | :--- | :--- | :--- |
| `description` | `string` | One-sentence summary of the memory content. | **Claude**: Supported as metadata in agent frontmatter.<br>**Others**: Dropped. |

## Filesystem-as-Schema

When a memory block is defined at `xcaf/agents/<agent-id>/memory/<id>/memory.xcaf`, Xcaffold automatically infers:
- **kind**: `memory` derived from the `memory/` directory.
- **name**: `<id>` derived from the directory segment between the kind and the filename.
- **agent**: `<agent-id>` derived from the grandparent agent directory.

## Compiled Output

Memory content is synthesized based on the target provider's capabilities:

- **Claude Code**: Emits entries in the `memory` list of the agent's Markdown frontmatter. The markdown body of the memory file is preserved.
- **Cursor**: Currently unsupported. Memory content is omitted from the compiled `.cursor/agents/` output.
- **Gemini CLI**: Appended to the system prompt context if referenced by the agent.
- **Antigravity**: Indexed for the custom agentic engine to drive context-aware retrieval.
