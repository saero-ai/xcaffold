---
title: "Xcaffold Kinds"
description: "xcaf kinds that govern the compiler — no output files produced."
---

# Xcaffold Kinds

Xcaffold kinds configure the compiler itself. They are evaluated at compile time during `xcaffold apply` and `xcaffold validate` but produce **no output files** in provider directories.

| Kind | Role |
|---|---|
| [`project`](./project) | Root manifest — declares targets, references all resources, provides project-level instructions |
| [`policy`](./policy) | Compile-time constraint; blocks or warns when a resource violates the rule |
| [`blueprint`](./blueprint) | Named resource subset used for conditional or partial compilation |
| [`global`](./global) | Shared resource definitions that are inherited across the entire project |
| [`registry`](./registry) | Machine-wide project index — tracks initialized repos and apply timestamps |

`project` uses **frontmatter+body format** (`---` delimiters required). The content after the closing `---` is compiled into provider root context files (`CLAUDE.md`, `AGENTS.md`, etc.). The body is optional but the delimiters are required.

All other xcaffold kinds — `policy`, `blueprint`, `global`, `registry` — use **pure YAML format** (no `---` delimiters).
