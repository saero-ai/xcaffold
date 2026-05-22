---
title: "Xcaffold Kinds"
description: "xcaf kinds that govern the compiler — no output files produced."
---

# Xcaffold Kinds

Xcaffold kinds govern the CLI and compiler. Most are evaluated at compile time during `xcaffold apply` and `xcaffold validate` and produce **no output files** in provider directories. The `registry` kind is an internal operational format managed by the CLI.

| Kind | Role |
|---|---|
| [`project`](./project) | Root manifest — declares compilation targets and configures project-wide settings |
| [`policy`](./policy) | Compile-time constraint; blocks or warns when a resource violates the rule |
| [`blueprint`](./blueprint) | Named resource subset used for conditional or partial compilation |
| [`global`](./global) | Shared resource definitions inherited by projects via `extends:` |
| [`registry`](./registry) | Internal project registry — tracks managed projects across the machine |
