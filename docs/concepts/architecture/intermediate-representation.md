---
title: "Intermediate Representation (IR)"
description: "The provider-agnostic, in-memory form of an agent configuration"
---

# Intermediate Representation (IR)

The **Intermediate Representation (IR)** is the provider-agnostic, in-memory form of an agent configuration — a fully parsed `ast.XcaffoldConfig` struct that contains all agents, skills, rules, workflows, hooks, memory entries, and MCP servers without any output-format concerns.

The IR is the bridge between every import, validation, and compilation phase:

| Phase | IR role |
|---|---|
| `xcaffold import` | Reads provider source files → builds IR → writes to `project.xcaf` (persists IR to disk) |
| `xcaffold validate` | Reads `project.xcaf` → builds IR → checks syntax, schema invariants, and cross-references |
| `xcaffold apply` | Reads `project.xcaf` (the persisted IR) → passes IR to compiler + optimizer → emits target output files |
| `xcaffold status` | Reads persisted IR from `.xcaffold/project.xcaf.state` hashes → detects drift |
| `xcaffold graph` | Builds IR → traverses dependency graph → visualizes resource topology |

The IR is intentionally format-neutral: the same struct that represents a Claude Code `.claude/agents/developer.md` agent also represents a Cursor `agents/developer.md` agent or an Antigravity agent. Renderers receive this struct and decide how to map it to the target format.

When you run `xcaffold import`, provider configurations are reverse-engineered into the IR and serialized to disk as `project.xcaf` and `xcaf/` manifests. This lets you inspect the IR, version-control it as a managed project, and feed it into `xcaffold apply` for ongoing GitOps management.

> **Why "IR"?** The term is borrowed from compiler design, where an IR is the normalized, source-language-independent form between parsing and code generation. xcaffold's IR plays the same role: it normalizes disjoint provider formats into a shared data model, then generates target-specific output from that model.
