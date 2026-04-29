---
title: "Concepts"
description: "Understanding-oriented explanations of xcaffold architecture and design rationale"
---

# Concepts

Understanding-oriented explanations of xcaffold's architectural and philosophical foundations. These documents explain *why* the system is designed the way it is — not how to use it.

## Documents

- [Architecture](architecture.md) — The one-way deterministic compiler architecture, multi-target pipelines, and GitOps state.
- [Agent Memory](agent-memory.md) — Understanding the durable, agent-scoped context model and execution lifecycle across targets.
- [Configuration Theory](configuration-scopes.md) — Understanding configuration contexts, scopes, implicit global inheritance, and boundaries.
- [Sandboxing](sandboxing.md) — Runtime sandbox configuration and compile-time evaluation sandbox.
- [State Files and Drift Detection](state-and-drift.md) — How xcaffold tracks compilation output and detects manual edits.
- [Multi-Target Rendering](multi-target-rendering.md) — How the AST compiles to different provider formats.
- [Provider Architecture](provider-architecture.md) — The ProviderImporter and TargetRenderer interfaces.
- [Blueprints](blueprints.md) — Named compilation scopes (Preview).

## Next Steps

- [`Tutorials`](../tutorials/index.md) — learning-oriented, step-by-step guides
- [`How-To Guides`](../how-to/index.md) — task-oriented guides for specific operations
- [`Reference`](../reference/index.md) — full schema reference and CLI command reference
