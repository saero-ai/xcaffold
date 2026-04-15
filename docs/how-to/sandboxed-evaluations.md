# Running Agent Evaluations with `xcaffold test`

`xcaffold test` simulates a compiled agent by reading its system prompt from `.claude/agents/<id>.md`, sending a task directly to the LLM API, and recording every tool call the model declares. The trace is then optionally evaluated against plain-English assertions using an LLM-as-a-Judge.

The simulation does not execute tools against the host OS. It captures what the model declares it would do given its system prompt and task — making evaluations safe to run repeatedly in CI or locally without side effects.

---

## Prerequisites

- Run `xcaffold apply` before testing — the agent must be compiled to `.claude/agents/` before `xcaffold test` can read its system prompt.
- Set `ANTHROPIC_API_KEY` or `XCAFFOLD_LLM_API_KEY` in your environment for the simulation run. See [Auth resolution](#auth-resolution).
- To run the judge (`--judge`), the same API key is used. A local CLI subscription is the fallback if no key is set.

---

## Running a basic evaluation

```bash
xcaffold test --agent backend-dev
```

This command:

1. Parses `scaffold.xcf` and resolves the `backend-dev` agent config.
2. Reads the compiled system prompt from `.claude/agents/backend-dev.md`.
3. Creates the trace file (`trace.jsonl` by default).
4. Sends the task to the LLM API (defaults to `"Describe what tools you have available and what you would do first."` if `test.task` is not set).
5. Parses `tool_use` blocks from the model's response and records them.
6. Prints a trace summary to stdout.

To write the trace to a custom path:

```bash
xcaffold test --agent backend-dev --output my-run.jsonl
```

---

## Configuring the task

The task is the user prompt sent to the agent during simulation. Set it in `scaffold.xcf` under `project.test`:

```yaml
project:
  name: my-app
  test:
    task: "Review the open pull requests and summarize what needs attention."
```

If `task` is not set, the default prompt is used: `"Describe what tools you have available and what you would do first."`

---

## Reading the trace

Every declared tool call is written as a newline-delimited JSON line to the trace file. Each line is a `ToolCallEvent` with the following fields:

| Field | Type | Description |
|---|---|---|
| `timestamp` | string (RFC3339) | UTC time the event was recorded |
| `agent_id` | string | Agent ID from `scaffold.xcf` |
| `tool_name` | string | Name of the declared tool |
| `input_params` | object | Parsed tool input parameters |
| `mock_response` | string | Reserved; empty for API simulation runs |
| `duration_ms` | number | Time from call to record, in milliseconds |
| `metadata` | object | Optional key-value pairs (omitted if empty) |

### Reviewing the trace

```bash
xcaffold review trace.jsonl
```

Output format:

```
=== XCAFFOLD TRACE LOG (3 events) ===
 [1] 2026-04-10T14:22:01Z -> bash
 [2] 2026-04-10T14:22:03Z -> read_file
 [3] 2026-04-10T14:22:05Z -> write_file
```

`xcaffold review all` also includes `trace.jsonl` in its scan when the file is present in the current directory.

---

## Writing effective assertions

Assertions are plain-English strings declared directly on an agent in `scaffold.xcf`. The judge receives the full trace and evaluates each assertion against the recorded tool calls.

```yaml
project:
  agents:
    backend-dev:
      model: claude-sonnet-4-6
      tools:
        - bash
        - read_file
        - write_file
      assertions:
        - The agent must read the requirements file before writing any code
        - The agent must not call bash with destructive commands like rm -rf
        - All file writes must target the src/ directory
```

`assertions` is a `[]string` field on `AgentConfig`. There is no separate `test` block on the agent — assertions live directly under the agent key.

### Effective assertions

Effective assertions describe observable, trace-verifiable behavior:

- "The agent must read the config file before invoking the build tool" — verifiable from tool call order in the trace.
- "The agent must not call bash with arguments containing `sudo`" — verifiable from `input_params`.
- "The agent must write output to `dist/output.json`" — verifiable from the `write_file` input params.

Weak assertions that reference intent or reasoning ("The agent should understand the requirements") cannot be verified from the tool call trace and will consistently produce `FAIL` verdicts. The judge prompt explicitly instructs the evaluator to treat claimed success without a confirming trace event as `FAIL`.

---

## Running the judge

Add `--judge` to run LLM-as-a-Judge evaluation after the simulation completes:

```bash
xcaffold test --agent backend-dev --judge
```

The judge reads the trace summary and each assertion, constructs an adversarial evaluation prompt, and returns a structured verdict.

### Auth resolution

Both the simulation and the judge resolve credentials in the same order:

1. `XCAFFOLD_LLM_API_KEY` (also reads `XCAFFOLD_LLM_BASE_URL` for the base URL)
2. `ANTHROPIC_API_KEY` (direct Anthropic API key)
3. CLI subscription fallback — uses the local CLI binary configured via `test.cli_path`

### Verdict types

| Verdict | Meaning |
|---|---|
| `PASS` | All assertions verified against the trace |
| `FAIL` | One or more assertions failed or could not be verified |
| `PARTIAL` | Some assertions passed, some failed |

### Judge output

```
── Judge Evaluation ──────────────────────────────────
  Model: claude-haiku-4-5-20251001
  Auth:  Target Provider API Key
  Assertions: 3

  Verdict: PARTIAL
  Reasoning: ...

  Passed:
    - The agent must read the requirements file before writing any code

  Failed:
    - The agent must not call bash with destructive commands like rm -rf
──────────────────────────────────────────────────────
```

The `Reasoning` field contains a full markdown evaluation report with per-assertion evidence drawn from the trace.

---

## Judge model selection

`resolveJudgeModel` determines the model used for evaluation. Priority (highest to lowest):

1. `--judge-model` flag
2. `test.judge-model` in `scaffold.xcf`
3. `"claude-haiku-4-5-20251001"` (hard default)

```yaml
project:
  name: my-app
  test:
    judge-model: claude-haiku-4-5-20251001
```

To override at runtime:

```bash
xcaffold test --agent backend-dev --judge --judge-model claude-sonnet-4-6
```

Haiku is the default because judge evaluation is a structured extraction task — not a reasoning-heavy task — and Haiku is significantly faster and cheaper for high-frequency CI runs.

---

## Flag reference

| Flag | Short | Default | Description |
|---|---|---|---|
| `--agent` | `-a` | (required) | Agent ID to simulate, as defined in `scaffold.xcf` |
| `--judge` | | `false` | Run LLM-as-a-Judge after simulation |
| `--output` | `-o` | `trace.jsonl` | Path to write the execution trace |
| `--cli-path` | | | Path to CLI binary used as judge subscription fallback; overrides `test.cli_path` in `scaffold.xcf` |
| `--judge-model` | | | Anthropic model for judge; overrides `test.judge-model` in `scaffold.xcf` |
