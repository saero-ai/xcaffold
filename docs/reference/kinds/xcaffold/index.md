---
title: "Xcaffold Kinds"
description: "xcf kinds that govern the compiler — no output files produced."
---

# Xcaffold Kinds

Xcaffold kinds configure the compiler itself. They are evaluated at compile time during `xcaffold apply` and `xcaffold validate` but produce **no output files** in provider directories.

| Kind | Role |
|---|---|
| [`project`](./project) | Root manifest — declares targets, references all resources, provides project-level instructions |
| [`policy`](./policy) | Compile-time constraint; blocks or warns when a resource violates the rule |
| [`settings`](./settings) | Global and workspace-scoped provider settings (permissions, model defaults, MCP merge) |
| [`hooks`](./hooks) | Lifecycle scripts run before or after tool invocations at the project level |
| [`blueprint`](./blueprint) | Named resource subset used for conditional or partial compilation |
| [`reference`](./reference) | Supporting files seeded verbatim into provider output directories |
| [`global`](./global) | Shared resource definitions that are inherited across the entire project |

These kinds use **pure YAML format** (no frontmatter `---` delimiters) with the exception of `project`, which supports an optional markdown body after closing `---` for project-level instructions.
