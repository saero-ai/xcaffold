---
title: "xcaffold test"
description: "Run a sandboxed local simulation of an agent."
---

# `xcaffold test`

Simulates your compiled agent by sending a task directly to the LLM and recording all declared tool calls. This allows for rapid iteration and validation of agent behavior without manually interacting with the CLI.

> [!IMPORTANT]
> You must run `xcaffold apply` before testing, as the command reads from the compiled provider-native files (e.g., `.claude/agents/`).

## Usage

```bash
xcaffold test --agent <id> [flags]
```

## Flags

| Flag | Short | Type | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `--agent` | `-a` | `string` | `""` | The ID of the agent to simulate. **Required**. |
| `--judge` | | `bool` | `false` | Run LLM-as-a-Judge evaluation after the simulation. |
| `--output` | `-o` | `string` | `trace.jsonl` | Path to write the execution trace. |
| `--cli-path` | | `string` | `""` | Path to the underlying CLI binary (overrides `project.xcf`). |
| `--judge-model`| | `string` | `""` | The model to use for the judge (overrides `project.xcf`). |

## Behavior

### Simulation Workflow

1.  **Compilation Check**: Verifies that the agent has been compiled to `.claude/agents/<id>.md`.
2.  **LLM Interaction**: Sends the system prompt and a test task to the LLM.
3.  **Trace Recording**: Captures every `tool_use` block emitted by the model and writes it to a JSONL trace file.
4.  **Evaluation (Optional)**: If `--judge` is set, an independent LLM evaluates the trace against the agent's `assertions`.

## Prerequisites

-   **API Keys**: You must have `ANTHROPIC_API_KEY` or `XCAFFOLD_LLM_API_KEY` set in your environment.
-   **Compiled Output**: The agent must have been successfully compiled via `xcaffold apply`.

## Examples

**Simulate the 'developer' agent:**

```bash
xcaffold test --agent developer
```

**Simulate and evaluate using the judge:**

```bash
xcaffold test --agent developer --judge
```

## Sample Output

```text
xcaffold-project  ·  simulating agent 'developer'

  → Task: "Add a new endpoint to the API"
  i  Anthropic API: model=claude-3-5-sonnet-20240620
  
  [tool_use]  read_file { path: "api/routes.go" }
  [tool_use]  write_file { path: "api/new_endpoint.go", content: "..." }
  
  ✓  Simulation complete. Trace written to trace.jsonl.
  ✓  Judge: 4/4 assertions passed.
```

## Exit Codes

| Code | Meaning |
| :--- | :--- |
| `0` | Success |
| `1` | Failure (e.g., agent not found, LLM error, or judge failure) |
