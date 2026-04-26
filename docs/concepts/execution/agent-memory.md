---
title: "Agent Memory"
description: "Durable, agent-scoped context that persists across sessions via convention-based directory discovery."
---

# Agent Memory

Agent Memory is the mechanism by which the compiler discovers and persists durable context elements—such as architectural decisions, coding patterns, or user feedback—that must survive across discrete agent sessions.

Unlike other elements of the configuration graph (Agents, Skills, Rules) which are strictly modeled in the AST, memory exists outside the parsing boundary. It acts as a bridge between the deterministic compiler output and the dynamic session context handled by provider integrations.

---

## The Core Mechanism

Memory operates entirely via a convention-based directory structure (`xcf/agents/<agent-id>/memory/`). The compiler discovers plain `.md` files present in these directories at compile time rather than relying on the YAML parse tree. 

When you run `xcaffold apply`, the compiler aggregates these discovered `.md` files according to the capabilities of the target provider. 

```text
xcf/
  agents/
    backend-dev/
      backend-dev.xcf           # Parsed agent resource
      memory/                   # Discovered convention directory
        database-schema.md      # Memory entry
        api-patterns.md         # Memory entry
```

A memory entry file is plain markdown, optionally prefixed with a YAML frontmatter block to declare a canonical `name` and `description`. 

> [!NOTE]
> Agent Memory is natively supported only for the Claude provider for subagents, which tracks and indexes cross-session memory dynamically. Other providers fall back to partial support (e.g., appending text) or lack capabilities entirely.

### Provider Execution Models

Native support (Claude Code) generates an auto-generated `MEMORY.md` index file containing clickable links to the individual `.md` files copied adjacent to it. Partial support (Gemini CLI, Antigravity) aggregates memory contents directly into their baseline instruction strings or serialized knowledge artifacts. Unsupported targets (Cursor, Copilot) emit `MEMORY_NO_NATIVE_TARGET` block notes and bypass memory emission entirely.

---

## Design Decisions

Memory was originally implemented as an AST resource (`kind: memory`) with explicit YAML metadata fields like `type` and `lifecycle`. We removed the parser-translation and migrated to a convention-based `.md` filesystem approach to eliminate format friction.

**Why `.md` instead of `.xcf`?** 
Memory content is fundamentally prose, not configuration. Wrapping memory inside YAML blocks created friction, serialization edge cases, and made it incompatible with native text-based subagent import/export operations. Dropping the parser requirement eliminated the entire class of translation bugs and provided zero-ceremony authoring.

**Why no seed-once lifecycle?**
The `--reseed` flag and `seed-once` lifecycle exceptions were abandoned to unify the behavioral contract of the compiler: `apply` always strictly overwrites output directories. Bidirectional syncing is managed globally via `xcaffold import` rather than per-resource state overrides.

---

## Interaction with Other Concepts

Memory binds directly to the **Compilation Targets** execution graph. Since it skips the parser and does not participate in constraint resolution or blueprint evaluation, memory propagation relies purely on the target provider's specific rendering engine and directory scanning behaviors. Memory entries interact heavily with **State and Drift Detection**, where `.xcaffold/project.xcf.state` tracks the SHA-256 hashes of compiled output specifically to prevent silent overwrites of provider-modified artifacts without throwing a drift error constraint.

### Import Behavior

Multi-directory imports (when two or more provider directories are detected) now correctly import memory entries from each provider. Previously, multi-directory imports silently dropped memory. Memory entries from all providers are merged using a union strategy, with first-seen winning on key collision within a single agent's memory scope.

---

## When This Matters

Memory configurations matter when provisioning long-lived service agents that build autonomous understanding vectors. 

- Subagent offboarding: Bootstrapping an agent onto a complex codebase requires restoring historical decisions (e.g., ORM choices, library constraints) without embedding those details strictly into rigid `kind: rule` configurations.
- Inter-agent transitions: Passing the output context of a research subagent to an implementation subagent natively integrates via cross-session injected memory blocks.

## Related

- [State and Drift](state-and-drift.md) — verifying artifact hashes and preventing file overwrite anomalies
- [Multi-Target Rendering](multi-target-rendering.md) — understanding capability variance across provider outputs
