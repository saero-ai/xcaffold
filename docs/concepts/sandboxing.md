---
title: "Sandboxing"
description: "Runtime sandbox configuration and compile-time evaluation sandbox"
---

# Sandboxing

xcaffold deals with two distinct sandboxing concepts that serve entirely different purposes. The first is `settings.sandbox` — a runtime configuration block that instructs the target AI platform to isolate the agent's process at the operating system level. The second is the intercept proxy used by `xcaffold test` — a loopback HTTP server that records tool-use calls during development-time evaluation without enforcing any restriction. These two mechanisms share a name in the broader agentic ecosystem but are architecturally unrelated, and conflating them produces incorrect assumptions about what xcaffold controls at runtime.

---

## Two Sandboxes, Two Purposes

The `settings.sandbox` block (`internal/ast/types.go:183-193`, type `SandboxConfig`) declares OS-level process isolation properties. When compiled to a target that supports it, this configuration is embedded in the platform's settings output. The platform — not xcaffold — is responsible for enforcing the isolation. xcaffold's role is strictly compilation: it translates the YAML declaration into whatever format the target requires. Once the output is on disk, xcaffold's involvement ends.

The `xcaffold test` proxy (`internal/proxy/proxy.go`) is a development-time observability tool. It binds a loopback HTTP server, intercepts outbound AI provider API calls made by a running CLI subprocess, records tool-use events to a JSONL trace file, and returns deterministic mock responses so that test scenarios complete without executing real tools. The proxy does not enforce the `settings.sandbox` configuration, cannot restrict filesystem access, and has no effect on any network policy declared in `SandboxNetwork`. It exists to make agent behavior observable during authoring — not to enforce production policies.

The distinction is consequential: `settings.sandbox` is a production guarantee delegated to the platform; the test proxy is a development instrument that imposes no guarantees at all.

---

## Target Fidelity for Runtime Sandbox

`settings.sandbox` only produces meaningful compiled output when targeting the `claude` target. For the `cursor` and `antigravity` targets, the entire `SandboxConfig` block is silently dropped from the output because neither target has a sandbox model equivalent to translate it into.

This target-fidelity boundary is explicitly reported by `apply --check-permissions`. The `securityFieldReport()` function (`cmd/xcaffold/apply.go:410-451`) inspects the parsed configuration against the active target and emits a `[WARNING]` line for each security field that would be dropped:

```
cursor: settings.sandbox will be dropped — no sandbox model
antigravity: settings.sandbox will be dropped — no sandbox model
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

`HTTPProxyPort` and `SOCKSProxyPort` are conceptually distinct from the test proxy port. The `SandboxNetwork` proxy ports direct production runtime traffic through an external inspection or filtering proxy chosen by the operator. The test proxy port is assigned dynamically at the loopback interface and exists only for the duration of a single `xcaffold test` session.

---

## The Evaluation Proxy

`xcaffold test` launches a local HTTP intercept proxy and injects its address into the subprocess environment so that AI provider API calls pass through it. The proxy is constructed by `proxy.New()` (`internal/proxy/proxy.go:42-64`), which binds exclusively to the loopback interface on a random port:

```go
ln, err := net.Listen("tcp", "127.0.0.1:0")
```

The zero port causes the OS to assign an available ephemeral port. The proxy is never exposed to the network. `proxy.Addr()` returns the actual bound address (e.g., `127.0.0.1:54321`), and `proxy.ProxyURL()` wraps it in an HTTP URL suitable for injection as a subprocess environment variable.

Host validation is enforced by `handleRequest()` (`internal/proxy/proxy.go:93-128`) before any request is processed. The allowed set is a static map (`internal/proxy/proxy.go:25-29`):

```go
var allowedHosts = map[string]bool{
    "api.anthropic.com":                 true,
    "generativelanguage.googleapis.com": true,
    "api.cursor.sh":                     true,
}
```

Comparison uses exact map lookup after lowercasing the `Host` header. The comment in the source is explicit: "Use exact equality to prevent SSRF via suffix confusion (e.g. evil-api.anthropic.com)." Requests targeting any host not in this map receive a `403 Forbidden` response. The proxy does not follow redirects or attempt to resolve unknown hosts.

When a POST request arrives at a recognized AI messaging endpoint (`/v1/messages` or a path containing `generateContent`), `handleRequest()` reads and inspects the body. If the payload contains a `"tool_use"` block, it is dispatched to `handleToolUse()` (`internal/proxy/proxy.go:139-169`). That function extracts the tool name and input parameters, constructs a `trace.ToolCallEvent`, records it via `trace.Recorder.Record()` (`internal/trace/trace.go:37-53`), and returns a deterministic mock response:

```
[SIMULATED SUCCESS]
```

The mock response is returned immediately without executing the tool against the host OS. The agent receives a plausible completion that allows the session to continue. All other requests — including non-tool-use messages and streaming completions — are forwarded transparently to the real AI provider API via `forward()` (`internal/proxy/proxy.go:172-186`).

`trace.Recorder` writes each `ToolCallEvent` as a newline-delimited JSON line (JSONL) to its writer. It is safe for concurrent use (`internal/trace/trace.go:24-28`) via an internal mutex, which is relevant because multiple goroutines may record events during a multi-turn agent session. After the CLI subprocess exits, the proxy shuts down via `proxy.Close()`, which calls `http.Server.Close()` for a graceful shutdown.

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
