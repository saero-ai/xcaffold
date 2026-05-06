---
title: "Concepts"
description: "Understanding-oriented explanations of xcaffold architecture and design rationale"
---

# Concepts

Understanding-oriented explanations of xcaffold's architectural and philosophical foundations. These documents explain *why* the system is designed the way it is — not how to use it.

## Architecture

- [Core Architecture](architecture/architecture.md) — The one-way deterministic compiler architecture.
- [Translation Pipeline](architecture/translation-pipeline.md) — How .xcf manifests move from discovery to provider output.
- [Intermediate Representation](architecture/intermediate-representation.md) — The BIR (Blueprint Intermediate Representation) graph.
- [Multi-Target Rendering](architecture/multi-target-rendering.md) — How the AST compiles to different provider formats.
- [Provider Architecture](architecture/provider-architecture.md) — The ProviderImporter and TargetRenderer interfaces.

## Configuration

- [Configuration Scopes](configuration/configuration-scopes.md) — Project, Agent, and Skill scoping isolation.
- [Declarative Compilation](configuration/declarative-compilation.md) — The manifest-driven approach to agent configuration.
- [Project Variables](configuration/variables.md) — Cross-file value reuse and environment configuration.
- [Field Classification Model](configuration/field-model.md) — Two-layer classification of resource fields.
- [Layer Precedence](configuration/layer-precedence.md) — Understanding the target resolution hierarchy.

## Execution

- [Agent Memory](execution/agent-memory.md) — Understanding the durable, agent-scoped context model.
- [State & Drift Detection](execution/state-and-drift.md) — How xcaffold tracks compilation output and detects manual edits.
- [Sandboxing](execution/sandboxing.md) — Runtime sandbox configuration and compile-time evaluation sandbox.

## Next Steps

- [`Tutorials`](../tutorials/index.md) — learning-oriented, step-by-step guides
- [`How-To Guides`](../how-to/index.md) — task-oriented guides for specific operations
- [`Reference`](../reference/index.md) — full schema reference and CLI command reference
