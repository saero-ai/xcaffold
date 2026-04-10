# Running Sandboxed Agent Evaluations with `xcaffold test`

`xcaffold test` runs a local, mocked simulation of an agent defined in your `scaffold.xcf`. It spawns an HTTP intercept proxy that routes all outbound LLM traffic through a controlled sandbox, records every tool call the agent attempts, and optionally evaluates the trace against plain-English assertions using an LLM-as-a-Judge.

The proxy intercepts tool-use requests before they reach the host OS. The agent executes in a contained environment where tool effects are mocked and deterministic, making evaluations safe to run repeatedly in CI or locally.

---

## Prerequisites

- The target CLI binary must be available on `$PATH` (default: `claude`), or configured via `test.cli_path` in `scaffold.xcf`.
- To run the judge (`--judge`), set `ANTHROPIC_API_KEY`, `XCAFFOLD_LLM_API_KEY`, or use a local CLI subscription. See [Running the judge](#running-the-judge).

---

## Running a basic evaluation

```bash
xcaffold test --agent backend-dev
```

This command:

1. Parses `scaffold.xcf` and resolves the `backend-dev` agent config.
2. Resolves the CLI binary path (see [CLI path resolution](#configuring-cli-path-resolution)).
3. Creates the trace file (`trace.jsonl` by default).
4. Starts the HTTP intercept proxy on a random loopback port.
5. Spawns the target CLI subprocess with `HTTPS_PROXY` and `HTTP_PROXY` set to the proxy address.
6. Waits for the subprocess to exit.
7. Prints a trace summary to stdout.

To write the trace to a custom path:

```bash
xcaffold test --agent backend-dev --output my-run.jsonl
```

---

## The HTTP intercept proxy

The proxy is the core isolation mechanism. It binds exclusively to `127.0.0.1:0` (a random loopback port) and is never exposed to the network. The subprocess receives the proxy address via `HTTPS_PROXY` and `HTTP_PROXY` environment variables.

### Allowed hosts

The proxy enforces an exact-match allowlist. Requests to any other host are rejected with HTTP 403:

- `api.anthropic.com`
- `generativelanguage.googleapis.com`
- `api.cursor.sh`

Exact-match comparison prevents SSRF via suffix confusion (e.g., `evil-api.anthropic.com` is rejected).

### Tool call interception

The proxy inspects POST requests to `/v1/messages` and URLs containing `generateContent`. If the request body contains a `"tool_use"` block, the proxy:

1. Extracts the tool name and input parameters from the raw JSON payload.
2. Records a `ToolCallEvent` to the trace file without executing the tool.
3. Returns a deterministic mock response (`[SIMULATED SUCCESS]`) to the agent.

All other requests — including initial prompt submissions — are forwarded transparently to the actual LLM API.

The proxy enforces a 10 MB body limit on inspected endpoints to prevent out-of-memory conditions.

---

## Reading the trace

Every intercepted tool call is written as a newline-delimited JSON line to the trace file. Each line is a `ToolCallEvent` with the following fields:

| Field | Type | Description |
|---|---|---|
| `timestamp` | string (RFC3339) | UTC time the interception occurred |
| `agent_id` | string | Value of the `X-Xcaffold-Agent` request header, or `"unknown"` |
| `tool_name` | string | Name of the intercepted tool, or `"unknown"` |
| `input_params` | object | Parsed tool input parameters |
| `mock_response` | string | The mock value returned to the agent |
| `duration_ms` | number | Time from interception to mock response, in milliseconds |
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

The judge resolves credentials in this order:

1. `XCAFFOLD_LLM_API_KEY` (platform-agnostic LLM API key; also reads `XCAFFOLD_LLM_BASE_URL` for the base URL)
2. `ANTHROPIC_API_KEY` (direct Anthropic API key)
3. CLI subscription fallback — uses the local CLI config of the target binary

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

  ✓ Passed:
    - The agent must read the requirements file before writing any code

  ✗ Failed:
    - The agent must not call bash with destructive commands like rm -rf
──────────────────────────────────────────────────────
```

The `Reasoning` field contains a full markdown evaluation report with per-assertion evidence drawn from the trace.

---

## Configuring CLI path resolution

`resolveCliPath` determines which binary is spawned as the agent subprocess. Priority (highest to lowest):

1. `--cli-path` flag
2. `test.cli_path` in `scaffold.xcf`
3. `test.claude_path` in `scaffold.xcf` (deprecated — retained for backward compatibility)
4. `"claude"` (resolved via `$PATH`)

```yaml
test:
  cli_path: /usr/local/bin/claude
```

To override at runtime without modifying the config file:

```bash
xcaffold test --agent backend-dev --cli-path /path/to/staging-claude
```

`test.claude_path` is the deprecated predecessor of `test.cli_path`. If both are set, `test.cli_path` takes precedence.

---

## Judge model selection

`resolveJudgeModel` determines the model used for evaluation. Priority (highest to lowest):

1. `--judge-model` flag
2. `test.judge_model` in `scaffold.xcf`
3. `"claude-haiku-4-5-20251001"` (hard default)

```yaml
test:
  judge_model: claude-haiku-4-5-20251001
```

To override at runtime:

```bash
xcaffold test --agent backend-dev --judge --judge-model claude-sonnet-4-6
```

Haiku is the default because judge evaluation is a structured extraction task — not a reasoning-heavy task — and Haiku is significantly faster and cheaper for high-frequency CI runs.

---

## Non-zero exit handling

If the target CLI subprocess exits with a non-zero code, `xcaffold test` emits a warning to stderr and continues:

```
Warning: target CLI exited with error: exit status 1
```

This is intentional. The trace recorder and proxy are shut down gracefully after the subprocess exits, and the judge still runs if `--judge` was specified. This design allows you to evaluate agents that crash or time out — their partial trace is often the most diagnostic artifact.

`xcaffold test` itself returns exit code 0 in this case. It only returns a non-zero code if the proxy fails to start, the config cannot be parsed, or the judge evaluation itself errors.

---

## Swapping target CLIs

The proxy intercept mechanism is target-agnostic. Any CLI binary that respects `HTTPS_PROXY` and `HTTP_PROXY` environment variables will have its LLM traffic routed through the sandbox.

To test with a different CLI binary — a local staging build, a different agent runtime, or a version under development — set `test.cli_path`:

```yaml
test:
  cli_path: /home/user/builds/claude-dev
```

Or pass it at runtime:

```bash
xcaffold test --agent backend-dev --cli-path ./bin/my-agent-runtime
```

The proxy records tool calls identically regardless of which binary is spawned. Assertions and judge evaluation work the same way across CLI targets.

---

## Flag reference

| Flag | Short | Default | Description |
|---|---|---|---|
| `--agent` | `-a` | (required) | Agent ID to simulate, as defined in `scaffold.xcf` |
| `--judge` | | `false` | Run LLM-as-a-Judge after simulation |
| `--output` | `-o` | `trace.jsonl` | Path to write the execution trace |
| `--cli-path` | | | Path to CLI binary; overrides `test.cli_path` in `scaffold.xcf` |
| `--judge-model` | | | Anthropic model for judge; overrides `test.judge_model` in `scaffold.xcf` |
