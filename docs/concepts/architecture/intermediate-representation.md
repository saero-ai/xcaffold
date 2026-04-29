---
title: "Intermediate Representation (IR)"
description: "The provider-agnostic, in-memory form of an agent configuration"
---

# Intermediate Representation (IR)

The **Intermediate Representation (IR)** is the provider-agnostic, in-memory form of an agent configuration — a fully parsed `ast.XcaffoldConfig` struct that contains all agents, skills, rules, workflows, hooks, memory entries, and MCP servers without any output-format concerns.

The IR is the bridge between every import, translation, and compilation phase:

| Phase | IR role |
|---|---|
| `xcaffold import` | Reads provider source files → builds IR → writes to `project.xcf` (persists IR to disk) |
| `xcaffold translate` — phase 1 | Reads provider source files → builds IR (in-memory only, never written unless `--save-xcf` is set) |
| `xcaffold translate` — phase 2 | Passes IR to the compiler + optimizer → emits target output files |
| `xcaffold apply` | Reads `project.xcf` from disk (the persisted IR) → compiles to target |
| `xcaffold diff` | Reads persisted IR from `.xcaffold/project.xcf.state` hashes → detects drift |

The IR is intentionally format-neutral: the same struct that represents a Claude Code `.claude/agents/developer.md` agent also represents a Cursor `agents/developer.md` agent or an Antigravity skill. Renderers receive this struct and decide how to map it to the target format.

When you run `xcaffold translate --save-xcf ir.xcf`, the in-memory IR is serialized to disk as a `project.xcf` file. This lets you inspect the IR before compilation, version-control it as a managed project, or feed it into `xcaffold apply` for ongoing GitOps management.

> **Why "IR"?** The term is borrowed from compiler design, where an IR is the normalized, source-language-independent form between parsing and code generation. xcaffold's IR plays the same role: it normalizes disjoint provider formats into a shared data model, then generates target-specific output from that model.
