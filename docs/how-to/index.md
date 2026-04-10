# How-To Guides

Task-oriented guides for engineers who already understand xcaffold's core model and need to accomplish a specific task. Each guide solves one problem — no tutorials, no conceptual explanations.

## Guides

- [Splitting a Project Into Multiple .xcf Files](multi-file-projects.md) — Split a monolithic `scaffold.xcf` into domain-scoped files with automatic merge, duplicate detection, and per-target lock tracking.

- [Inheriting Configuration with `extends:` and Linking with `references:`](ast-inheritance-and-cross-referencing.md) — Share agents, rules, and MCP servers across projects via a base config, and embed supplementary files into skill output with `references:`.

- [Configuring Per-Target Overrides](target-overrides.md) (Experimental) — Declare renderer-specific behavior on agents using the `targets:` block. Currently parsed but not fully compiled; `suppress_fidelity_warnings` and `--check-permissions` are functional today.

- [Running Sandboxed Agent Evaluations with `xcaffold test`](sandboxed-evaluations.md) — Spawn an HTTP intercept proxy, record tool call traces, and evaluate agent behavior against assertions using LLM-as-a-Judge.

- [Binding MCP Tool Servers to Agents](mcp-server-integration.md) — Define stdio, SSE, and HTTP MCP servers, reference them from agents, and understand how they compile to `settings.json` (Claude) vs `mcp.json` (Cursor).

- [Enforcing Project Policies](policy-enforcement.md) — Write custom constraint policies, override built-in rules, deny content patterns in compiled output, and interpret violation diagnostics from `apply` and `validate`.
