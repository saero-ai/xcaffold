---
title: "Tutorials"
description: "Learning-oriented, step-by-step guides for xcaffold"
---

# Tutorials

xcaffold tutorials are learning-oriented, step-by-step guides that take you from zero to a working result. xcaffold is a Harness-as-Code tool — it manages the complete agent harness (system prompts, tools, rules, memory, hooks, MCP servers, and policies) as version-controlled `.xcaf` manifests that compile to native provider formats. Each tutorial prioritizes a successful first experience over comprehensive coverage — consult the reference documentation once the workflow is familiar.

## Prerequisites

- xcaffold installed: `brew install saero-ai/tap/xcaffold` or `go install github.com/saero-ai/xcaffold/cmd/xcaffold@latest`
- A terminal
- Any text editor

No AI subscription or API key is required for `init`, `apply`, `status`, or `validate`.

## Tutorials

Work through these in order. Getting Started is a prerequisite for the other three.

| Order | Tutorial | Description | Time |
|-------|----------|-------------|------|
| 1 | [Getting Started](basics/getting-started.md) | Initialize a project, compile your first agent, and understand the `.xcaf` → output pipeline | ~10 min |
| 2 | [AI-Assisted Scaffolding](basics/ai-assisted-scaffolding.md) | Use an AI coding assistant to fill in your scaffold without hallucinating provider-specific formats | ~15 min |
| 3 | [Multi-Agent Workspace](advanced/multi-agent-workspace.md) | Configure a team of differentiated agents with distinct tool permissions, shared rules and skills, and validated output | ~15 min |
| 4 | [Drift Remediation](advanced/drift-remediation.md) | Detect, diagnose, and restore managed files when compiled output has been modified directly | ~10 min |

## Reading Order

**Getting Started** must be completed first — it establishes the project structure and compilation workflow that all other tutorials build on.

**AI-Assisted Scaffolding** is the natural second step — it shows how the multi-file layout from Getting Started becomes the editing surface for an AI assistant.

**Multi-Agent Workspace** and **Drift Remediation** are independent of each other and can be read in either order after Getting Started.

## Next Steps

- [`Concepts`](../concepts/index.md) — deep dives into architecture and compilation scopes
- [`Reference`](../reference/index.md) — full schema reference and CLI command reference
