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

These kinds use **pure YAML format** (no frontmatter `---` delimiters) with the exception of `project`, which supports an optional markdown body after closing `---` for project-level instructions.
