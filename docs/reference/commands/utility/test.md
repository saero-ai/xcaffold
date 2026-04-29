---
title: "xcaffold test"
description: "Simulate and test compiled agents directly against the LLM via tool-call generation."
---

> **Note:** This command is in **Preview**. It is available natively in the binary but its execution schemas subject to changes alongside Anthropic capability shifts.

# xcaffold test

Simulates an active compiled workspace agent by routing your prompt architecture dynamically against an LLM interface through zero-shot prompts and logging what instructions the agent would inherently invoke.

Instead of writing physical integration tests targeting the API surface directly, the `test` command acts as a simulated harness evaluating reasoning execution.

## Usage

```bash
xcaffold test --agent <id> [flags]
```

## Prerequisites

- You must execute `xcaffold apply` explicitly ahead of `test` boundaries because testing sources execution logic directly from compiled artifact artifacts (e.g., `.claude/agents/<agent>.md`).
- A localized Anthropic provider is required via `ANTHROPIC_API_KEY` (or the internal alias `XCAFFOLD_LLM_API_KEY`) set inside your system environment parameters.

## Options

| Flag | Default | Description |
|---|---|---|
| `-a, --agent <string>` | *(Required)* | Targeted Agent ID mapping directly to the underlying definition ID. |
| `--cli-path <string>`| `""` | Provide custom executable path boundaries targeting alternative localized CLI architectures if bypassing Anthropic API endpoints directly. |
| `--judge` | `false` | Following the core tool-call simulation matrix, route output alongside explicitly defined schema assertions backward into the simulation stack, returning boolean accuracy feedback loops. |
| `--judge-model <string>` | `""` | Assign an alternative reasoning node (Anthropic Model definition) for running evaluation protocols explicitly overriding defaults. |
| `-o, --output <string>` | `"trace.jsonl"` | Designated target file path for writing JSONL-based execution stream. |

## Behavior

`xcaffold test` isolates reasoning layers without applying direct side effects to your system variables.

The architecture fundamentally functions on three sequential phases:
1. **Prompt Compilation:** It extracts standard system instructions formatted natively inside the `agents/` directories and merges those inputs implicitly into an isolated API transaction stream.
2. **Evaluation Layer:** Native declarations invoked by the responding model evaluating your configuration are scraped directly from JSON block schemas inside the stream. 
3. **Execution Trace Logging:** Each independent action the model requested authorization to execute is pushed locally into an output log (`trace.jsonl`). You leverage this trace mapping statically alongside programmatic regression validations matching specific outputs to expected inputs.

## Examples

**Run a baseline tool execution simulation matching a configured `backend-dev` definition:**
```bash
xcaffold test --agent backend-dev
```

**Evaluate behavior explicitly alongside schema assertions for a `data-analyst` specification:**
```bash
xcaffold test --agent data-analyst --judge
```

**Export tool invocation outputs dynamically to a custom trace file:**
```bash
xcaffold test -a frontend-dev --output custom_trace.jsonl
```
