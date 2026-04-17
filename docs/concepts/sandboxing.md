---
title: "Sandboxing"
description: "The two sandboxing concepts in xcaffold: OS-level runtime sandbox configuration and the xcaffold test evaluation simulation"
---

# Sandboxing

xcaffold deals with two distinct sandboxing concepts that serve entirely different purposes. The first is `settings.sandbox` — a runtime configuration block that instructs the target AI platform to isolate the agent's process at the operating system level. The second is the API simulation used by `xcaffold test` — a direct LLM API call that reads the compiled agent system prompt, sends a task, and records tool calls declared in the response. These two mechanisms share a name in the broader agentic ecosystem but are architecturally unrelated, and conflating them produces incorrect assumptions about what xcaffold controls at runtime.

---

## Two Sandboxes, Two Purposes

The `settings.sandbox` block (`internal/ast/types.go:183-193`, type `SandboxConfig`) declares OS-level process isolation properties. When compiled to a target that supports it, this configuration is embedded in the platform's settings output. The platform — not xcaffold — is responsible for enforcing the isolation. xcaffold's role is strictly compilation: it translates the `.xcf` declaration into whatever format the target requires. Once the output is on disk, xcaffold's involvement ends.

The `xcaffold test` simulation reads the compiled agent system prompt from the target output directory (e.g., `.claude/agents/<id>.md` for the `claude` target), sends a user task to the LLM API directly via `internal/llmclient`, and extracts tool call declarations from the model's response. It records these to a JSONL trace file. The simulation does not execute tools against the host OS — it captures what the model declares it would do, not what the tools actually produce. The simulation does not enforce `settings.sandbox` configuration and has no effect on any network policy declared in `SandboxNetwork`. It exists to make agent behavior observable during authoring.

The distinction is consequential: `settings.sandbox` is a production guarantee delegated to the platform; the test simulation is a development instrument that imposes no runtime guarantees at all.

---

## Target Fidelity for Runtime Sandbox

`settings.sandbox` produces compiled output and is supported for multiple targets:

- **Claude Code** — supports filesystem and network sandboxing via `settings.sandbox` (full fidelity)
- **Cursor** — supports per-command sandboxing via hook-level `sandbox` boolean configuration
- **Antigravity** — supports kernel-level sandboxing (Seatbelt on macOS, nsjail on Linux)
- **Gemini CLI** — supports tool-level sandboxing via `tools.sandboxAllowedPaths` and `tools.sandboxNetworkAccess`
- **GitHub Copilot** — does NOT support sandbox configuration (field is dropped)

This target-fidelity boundary is explicitly reported by `apply --check-permissions`. The `securityFieldReport()` function (`cmd/xcaffold/apply.go:410-451`) inspects the parsed configuration against the active target and emits a `[WARNING]` line for each security field that would be dropped:

```
github-copilot: settings.sandbox will be dropped — no sandbox model
```

The function also detects conflicts between `settings.permissions.deny` rules and per-agent `tools` lists, emitting `[ERROR]` lines for those. `--check-permissions` is read-only and never modifies any output file. It returns a non-zero exit code only when `[ERROR]`-level conflicts are found; dropped security fields are warnings, not errors.

This design reflects a principle xcaffold applies throughout multi-target rendering: the source of truth is always the `.xcf` file, and per-target fidelity differences are surfaced explicitly rather than silently accepted. Users who rely on sandbox isolation for security properties must understand that those properties only exist when the compiled output is consumed by a platform that supports them.

---

## Filesystem Isolation Model

The `SandboxFilesystem` struct (`internal/ast/types.go:196-201`) defines four path-glob arrays:

- `AllowWrite` — paths the sandboxed process may write to
- `DenyWrite` — paths explicitly denied for writes
- `AllowRead` — paths the sandboxed process may read from
- `DenyRead` — paths explicitly denied for reads

Each array accepts glob patterns. xcaffold compiles these arrays into the platform's settings output verbatim. It performs no evaluation of whether the patterns are sensible, whether they overlap, or whether they would produce an effective isolation boundary. That evaluation is the platform's responsibility.

This separation is intentional. Glob semantics for OS-level sandbox policies are platform-specific: what one platform interprets as a recursive wildcard another may treat literally. xcaffold avoids encoding assumptions about how a specific platform resolves path patterns. The `SandboxFilesystem` struct is a typed passthrough, not a policy engine.

---

## Network Isolation Model

The `SandboxNetwork` struct (`internal/ast/types.go:204-214`) configures outbound network policy for the sandboxed process:

- `AllowedDomains` — explicit list of domains the process may connect to
- `AllowManagedDomainsOnly` — boolean that restricts connections to a platform-managed allowlist rather than a user-defined one
- `HTTPProxyPort` — port number for an HTTP proxy the sandboxed process should route traffic through
- `SOCKSProxyPort` — port number for a SOCKS proxy
- `AllowUnixSockets` — list of Unix domain socket paths the process may connect to; an empty list denies all, `["*"]` allows all
- `AllowLocalBinding` — boolean permitting the process to bind to localhost ports

These fields are compiled and passed through to the target platform's settings. As with filesystem isolation, the enforcement semantics are platform-defined. xcaffold guarantees only that the configuration reaches the output file in the correct format for the active target.

`HTTPProxyPort` and `SOCKSProxyPort` direct production runtime traffic through an external inspection or filtering proxy chosen by the operator. They are unrelated to `xcaffold test`, which calls the LLM API directly rather than through a local proxy.

---

## The Evaluation Simulation

`xcaffold test` reads the compiled agent system prompt from the target output directory (e.g., `.claude/agents/<id>.md` for the `claude` target) and sends it along with a configurable task to the LLM API via `internal/llmclient`. The client resolves credentials in priority order: `XCAFFOLD_LLM_API_KEY` + `XCAFFOLD_LLM_BASE_URL`, then `ANTHROPIC_API_KEY`, then a CLI binary subscription fallback.

The model's response is parsed for `tool_use` blocks. Two formats are handled:

1. A structured Anthropic content array (`{"content": [{"type": "tool_use", ...}]}`).
2. Inline JSON objects with `"type": "tool_use"` embedded in free-form text.

Each extracted tool call is recorded as a `trace.ToolCallEvent` via `trace.Recorder.Record()` (`internal/trace/trace.go:37-53`). `trace.Recorder` writes each event as a newline-delimited JSON line (JSONL) to its writer. It is safe for concurrent use via an internal mutex.

The simulation does not execute tools against the host OS. It captures what the model declares it would do given the system prompt and task — not the actual results of running those tools.

> **Note:** `internal/proxy` remains in the codebase for potential future use but is not invoked by `xcaffold test`.

---

## LLM-as-a-Judge Evaluation

When `xcaffold test` completes a session, the recorded `trace.Summary` — containing all intercepted tool calls, their parameters, and their mock responses — is available for post-session evaluation. If `--judge` is specified, xcaffold constructs a `judge.Judge` (`internal/judge/judge.go:31-34`) and calls `Evaluate()` (`internal/judge/judge.go:68-85`), passing the trace summary alongside the agent's `assertions` list from the `.xcf` file.

`Evaluate()` assembles a structured prompt via `buildPrompt()` (`internal/judge/judge.go:88-142`). The prompt presents the trace summary, the detailed tool call log with parameters and mock responses, the assertions list, and an adversarial verification instruction: the judge model is explicitly told to treat unverified success claims — assertions that claim passing without supporting trace evidence — as failures.

The judge model returns a markdown reasoning section followed by a JSON block. `parseTextReport()` (`internal/judge/judge.go:145-190`) extracts the JSON block by searching for the last ` ```json ` fence and parses it into a `judge.Report`:

- `Verdict` — `"PASS"`, `"FAIL"`, or `"PARTIAL"`
- `Reasoning` — the full markdown analysis produced by the judge model
- `PassedAssertions` — list of assertion strings that the judge determined were satisfied
- `FailedAssertions` — list of assertion strings that the judge determined were not satisfied

The default judge model is `claude-haiku-4-5-20251001` (`internal/judge/judge.go:15`), configurable via `TestConfig.JudgeModel` in the `.xcf` file's `project.test` block (`internal/ast/types.go:252-262`).

This evaluation is explicitly a soft check using LLM reasoning, not deterministic rule matching. The judge model interprets the trace evidence against each assertion using natural language understanding. Two runs against identical trace data may produce marginally different reasoning text, though the verdict is generally stable for well-defined assertions. Users who require deterministic pass/fail signals should express assertions in terms of concrete, observable trace properties — specific tool names called, specific parameter values — rather than behavioral descriptions that require inference.

If no assertions are defined in the agent's configuration, `Evaluate()` returns a `Report` with an explanatory `Reasoning` field and no verdict, reminding the author to add assertions to make the evaluation meaningful.

---

## When This Matters

- **Deciding whether `settings.sandbox` provides a security guarantee** — it does only when the compiled output is consumed by a target platform that enforces it; xcaffold compiles the declaration but does not enforce it. GitHub Copilot does not support sandbox configuration; Claude Code, Cursor, Antigravity, and Gemini CLI each have sandbox implementations with different fidelity and configuration mechanisms.
- **Understanding what `xcaffold test` does and does not test** — the evaluation simulation captures tool call declarations from the model's response; it does not execute tools against the host OS, enforce sandbox network policies, or provide any runtime isolation.
- **Using `--check-permissions` before applying to a new target** — `securityFieldReport()` surfaces which security fields will be silently dropped for the active target, allowing an informed decision before writing output files.
- **Interpreting non-deterministic judge verdicts** — the LLM-as-a-judge evaluation uses natural language reasoning; for stable pass/fail signals in CI, assertions should reference concrete trace properties (specific tool names and parameter values) rather than behavioral descriptions.

---

## Related

- [Sandboxed Evaluations](../how-to/sandboxed-evaluations.md) — how to configure and run `xcaffold test` with assertions
- [Architecture](architecture.md) — internal package map including `internal/llmclient`, `internal/judge`, and `internal/trace`
- [Multi-Target Rendering](architecture.md#multi-target-rendering) — target fidelity model and how fields are dropped per target
- [Schema Reference](../reference/schema.md) — `settings.sandbox`, `SandboxFilesystem`, `SandboxNetwork`, and `test:` block field reference
