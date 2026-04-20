---
title: "Configuring Per-Target Overrides"
description: "Declare renderer-specific behavior on agents using the targets: block for multi-platform compilation"
---

> **EXPERIMENTAL**: The `targets:` block is parsed and stored in the AST but is **not fully compiled**. Defining `targets:` overrides in your `.xcf` file will not change compiler output for most fields. A warning is emitted to stderr on every `xcaffold apply` run when any agent defines a `targets:` block. The `--check-permissions` flag is fully functional and independent of the `targets:` block.

# Configuring Per-Target Overrides

You compile the same `.xcf` to multiple platforms but need certain agents to behave differently on specific targets — for example, suppressing fidelity warnings for Cursor or checking permission compatibility. xcaffold compiles a single `project.xcf` source into platform-native output directories. The `targets:` block on an agent allows you to declare renderer-specific behavior — different instructions, skipped synthesis, suppressed warnings — without duplicating the entire agent definition.

**When to use this:** When you compile to `cursor` or `antigravity` and want to silence per-agent fidelity drop warnings for agents that intentionally carry security fields without a target equivalent.

**Prerequisites:** Completed [Getting Started](../tutorials/getting-started.md) tutorial.

---

## Current status

The `targets:` block is parsed by the strict YAML parser (`yaml.KnownFields(true)`) and stored in the `AgentConfig.Targets` field of the AST. It does not cause a parse error. However, the compiler does not act on most fields during the current compilation pass.

When `xcaffold apply` encounters any agent with a non-empty `targets:` block, it emits the following warning to stderr and continues:

```
[project] Warning: 'targets' block is experimental and currently uncompiled.
```

The warning fires once per compilation run, regardless of how many agents define `targets:` blocks.

## The `TargetOverride` fields

Each key under `agents.<id>.targets.<target>` maps to a `TargetOverride` struct.

| Field | Type | Status |
|---|---|---|
| `suppress_fidelity_warnings` | `*bool` | Parsed, not compiled |
| `hooks` | `map[string]string` | Parsed, not compiled |
| `skip_synthesis` | `*bool` | Parsed, not compiled |
| `instructions_override` | `string` | Parsed, not compiled |

Valid target keys are: `claude`, `cursor`, `antigravity`, `copilot`, `gemini`.

## `suppress_fidelity_warnings`

When fully implemented, setting `suppress_fidelity_warnings: true` for a target will silence per-agent warnings about security fields being dropped.

The cursor and antigravity renderers currently emit warnings for each agent whose security fields (`permission-mode`, `disallowed-tools`, `isolation`) will be dropped — these fields have no enforcement equivalent in those renderers.

**Example — suppress cursor fidelity warnings for a specific agent:**

```yaml
project:
  agents:
    security-auditor:
      name: Security Auditor
      model: claude-opus-4-5
      permission-mode: restricted
      disallowed-tools:
        - Bash
      targets:
        cursor:
          suppress_fidelity_warnings: true
```

When compiling to `--target cursor`, the renderer checks:

```go
if override, ok := agent.Targets["cursor"]; ok && override.SuppressFidelityWarnings != nil && *override.SuppressFidelityWarnings {
    suppress = true
}
```

The cursor renderer emits separate warnings for each dropped field:

```
WARNING (cursor): agent "security-auditor" permission-mode "bypassAll" dropped — Cursor has no permission mode equivalent.
WARNING (cursor): agent "security-auditor" disallowed-tools dropped — tool restrictions will NOT be enforced by Cursor.
WARNING (cursor): agent "security-auditor" isolation "sandbox" dropped — Cursor has no process isolation model.
```

The `antigravity` renderer also emits separate warnings for each field:

```
WARNING (antigravity): agent "security-auditor" permissionMode dropped; Antigravity has no permission mode equivalent
WARNING (antigravity): agent "security-auditor" disallowedTools dropped; tool restrictions will NOT be enforced by Antigravity
WARNING (antigravity): agent "security-auditor" isolation dropped; Antigravity has no process isolation model
```

## Using `--check-permissions` today

The `--check-permissions` flag is fully functional and independent of the `targets:` block. It performs a read-only audit of your configuration against a selected target and reports which security fields will be dropped during compilation. It never modifies files.

```
xcaffold apply --check-permissions --target <target>
```

`securityFieldReport()` inspects the following fields for `cursor` and `antigravity` targets:

- `settings.permissions` — dropped with no enforcement equivalent
- `settings.sandbox` — dropped, no sandbox model
- Per-agent `permission-mode` — dropped
- Per-agent `disallowed-tools` — dropped, tool restrictions will not be enforced
- Per-agent `isolation` — dropped

It also detects conflicts: if an agent's `tools` list includes a tool that appears in `settings.permissions.deny`, that is reported as an `[ERROR]`.

For the `claude` target, all security fields are natively supported. `securityFieldReport()` produces no findings.

### Dual-target comparison

Consider a configuration with the following security fields set:

```yaml
project:
  settings:
    permissions:
      allow:
        - "Bash(go test *)"
      deny:
        - "Bash(rm *)"
    sandbox:
      enabled: true

  agents:
    deployer:
      name: Deployer
      permission-mode: restricted
      disallowed-tools:
        - WebSearch
      isolation: container
```

**`xcaffold apply --check-permissions --target cursor`:**

```
[WARNING] cursor: settings.permissions will be dropped — no enforcement equivalent
[WARNING] cursor: settings.sandbox will be dropped — no sandbox model
[WARNING] cursor: agent "deployer" permission-mode "restricted" will be dropped
[WARNING] cursor: agent "deployer" disallowed-tools will be dropped — tool restrictions will NOT be enforced
[WARNING] cursor: agent "deployer" isolation "container" will be dropped
```

Exit code: `0` (warnings do not fail the command; only `[ERROR]` lines do).

**`xcaffold apply --check-permissions --target claude`:**

```
[INFO]    claude: all security fields are supported
```

Exit code: `0`.

If `settings.permissions.deny` listed `"Bash(rm *)"` and an agent's `tools` included `"Bash(rm *)"`, the cursor audit would additionally emit:

```
[ERROR]   permissions.deny: rule "Bash(rm *)" conflicts with agent "deployer" tools list
```

Exit code: `1`.

## Intended use cases

When the `targets:` block is fully compiled, it will enable the following patterns:

**`instructions_override`** — provide renderer-specific instructions without defining a second agent. A `claude` compilation uses the base `instructions:` field; a `cursor` compilation substitutes `instructions_override:`.

**`skip_synthesis`** — exclude the agent from a specific renderer's output entirely. Useful for agents that rely on Claude-specific capabilities with no cursor or antigravity equivalent.

**`hooks`** — map per-agent hooks to renderer-specific event names where lifecycle event naming differs between platforms.

## Supported targets

| Target flag | Output directory | Description |
|---|---|---|
| `claude` | `.claude/` | Claude Code agent files (YAML frontmatter + Markdown) |
| `cursor` | `.cursor/` | Cursor agent rules (YAML frontmatter + Markdown) |
| `antigravity` | `.agents/` | Antigravity workflow definitions (plain Markdown) |
| `copilot` | `.github/` | GitHub Copilot instructions and prompt files |
| `gemini` | `.gemini/` | Gemini CLI agent and rules files |

The default target when `--target` is omitted is `claude`.

---

## Verification

Run a permission check against the cursor target to confirm the `targets:` block is parsed correctly:

```bash
xcaffold apply --check-permissions --target cursor
```

Once implemented, if an agent defines `suppress_fidelity_warnings: true` under `targets.cursor`, the per-agent security warnings for that agent will be absent from the output. All other agents with dropped security fields will still emit warnings.

---

## Troubleshooting

| Error | Cause | Fix |
|---|---|---|
| `unknown field "suppress_fidelity_warnings"` | Using snake_case key outside a `provider:` pass-through block | This field is in the `targets.<name>` block directly — confirm indentation is correct |
| Fidelity warnings still appear after setting `suppress_fidelity_warnings: true` | The `suppress_fidelity_warnings` override has not yet been implemented in the renderers. Watch for updates in upcoming releases. |
| `[project] Warning: 'targets' block is experimental` on every apply | Expected — the warning fires whenever any agent defines a non-empty `targets:` block | This is informational; `targets:` blocks are parsed but not yet compiled into renderer behavior |

---

## Related

- [CLI Reference: xcaffold apply](../reference/cli.md#xcaffold-apply)
- [Schema Reference: TargetOverride](../reference/schema.md#targetoverride)
- [Concepts: Multi-Target Rendering](../concepts/architecture.md#multi-target-rendering)
