---
title: "How-To Guides"
description: "Task-oriented guides for specific xcaffold operations"
---

# How-To Guides

Task-oriented guides for engineers who already understand xcaffold's core model and need to accomplish a specific task. Each guide solves one problem — no tutorials, no conceptual explanations.

## Guides

- [Splitting a Project Into Multiple .xcf Files](multi-file-projects.md) — Split a monolithic `scaffold.xcf` into domain-scoped files with automatic merge, duplicate detection, and per-target lock tracking.



- [Import Existing Config](import-existing-config.md) — Adopt xcaffold on an existing project by importing `.claude/`, `.cursor/`, or `.agents/` directories into `scaffold.xcf` + `xcf/` split files.

- [Inheriting Configuration with `extends:` and Linking with `references:`](ast-inheritance-and-cross-referencing.md) — Share agents, rules, and MCP servers across projects via a base config, and embed supplementary files into skill output with `references:`.

- [Configuring Per-Target Overrides](target-overrides.md) (Experimental) — Declare renderer-specific behavior on agents using the `targets:` block. Currently parsed but not fully compiled; `suppress_fidelity_warnings` and `--check-permissions` are functional today.

- [Running Sandboxed Agent Evaluations with `xcaffold test`](sandboxed-evaluations.md) — Simulate a compiled agent against a task, record tool call traces without executing them, and evaluate behavior against assertions using LLM-as-a-Judge.

- [Binding MCP Tool Servers to Agents](mcp-server-integration.md) — Define stdio, SSE, and HTTP MCP servers, reference them from agents, and understand how they compile to provider-specific MCP config files: `.mcp.json` (Claude Code), `.cursor/mcp.json` (Cursor), with transport and field normalizations.

- [Enforcing Project Policies](policy-enforcement.md) — Write custom constraint policies, override built-in rules, deny content patterns in compiled output, and interpret violation diagnostics from `apply` and `validate`.

- [Translating Configurations Between Providers](xcaffold-translate.md) — Convert agent configurations directly between providers (`claude`, `cursor`, `antigravity`, `copilot`, `gemini`) using fidelity modes, workflow lowering strategies, and optimizer passes. Covers `--dry-run`, `--diff`, `--save-xcf`, `--audit-out`, and memory translation.

- [Best Practices](best-practices.md) — Configuration and structural best practices.

## Next Steps

- [`Tutorials`](../tutorials/index.md) — learning-oriented, step-by-step guides
- [`Concepts`](../concepts/index.md) — deep dives into architecture and compilation scopes
- [`Reference`](../reference/index.md) — full schema reference and CLI command reference
