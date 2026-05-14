---
title: "Multi-Target Compilation"
description: "How to manage a single set of .xcaf manifests that compiles cleanly to multiple providers, using target scoping, provider overrides, and fidelity notes."
---

# Multi-Target Compilation

xcaffold compiles one set of `.xcaf` manifests into provider-native output for every provider you list in your project configuration. A single `xcaffold apply` run produces output for Claude, Cursor, Gemini, Copilot, and Antigravity — or any subset you choose.

This guide covers how to structure and maintain a multi-target project: declaring targets, writing resources that work across all providers, scoping resources to specific providers when needed, customizing per-provider behavior through override files, and interpreting fidelity notes when a field or resource is not fully supported.

## Declaring Targets

Targets are declared in your `kind: project` file. Every target listed here is compiled on every `xcaffold apply` run.

```yaml
# xcaf/project.xcaf
kind: project
version: "1.0"
name: my-project
targets:
  - claude
  - cursor
  - gemini
  - copilot
  - antigravity
  - codex
```

`xcaffold apply` processes each target in sequence and writes provider-native output to the corresponding output directory (`.claude/`, `.cursor/`, `.gemini/`, `.github/`, `.agents/`). All targets must be listed here — there is no separate per-target project file.

To apply a single target without modifying your project declaration, use the `--target` flag:

```bash
xcaffold apply --target claude
```

## Write Once, Compile Everywhere

The simplest approach to multi-target compilation is to write resources with no target-specific configuration. xcaffold compiles them for every declared target, and any fields that a provider does not support are silently dropped or produce a fidelity note.

```yaml
# xcaf/agents/developer/agent.xcaf
kind: agent
version: "1.0"
name: developer
description: "General-purpose software developer."
model: sonnet
tools: [Read, Write, Edit, Bash, Glob, Grep]
skills: [tdd, code-review]
rules: [secure-code]
```

```yaml
# xcaf/skills/tdd/skill.xcaf
kind: skill
version: "1.0"
name: tdd
description: "Test-driven development workflow."
allowed-tools: [Bash, Read, Write, Edit]
---
Follow the Red-Green-Refactor cycle. Write a failing test before writing implementation code.
```

These resources compile for all six providers without any additional configuration. Fields that a provider cannot represent are dropped; xcaffold tells you exactly which ones and why.

### Fidelity Notes as Feedback

When xcaffold drops a field or skips a resource during compilation, it emits a fidelity note to stderr:

- **info** — informational; no action needed. The output is complete and correct.
- **warning** — a field was dropped or downgraded. Compilation succeeds. The output is usable but loses some information.
- **error** — something requires investigation. The output may be incomplete.

Warnings do not block compilation. They are a signal to either adjust your manifest or acknowledge that the drop is intentional for that provider.

Use `xcaffold validate --target <provider>` to run field-level fidelity checks before compiling:

```bash
xcaffold validate --target gemini
xcaffold validate --target copilot
```

This surfaces fidelity issues without writing any output — useful in CI before applying to production.

## Scoping Resources to Specific Targets

Every resource kind supports a `targets:` map. When this map is non-empty and the current compilation target is not listed in it, xcaffold removes the resource from that target's output entirely and emits a warning note.

When `targets:` is absent, the resource is included for all declared targets.

### Restricting to one provider

A Claude-only MCP server that requires Claude Code's native MCP runtime:

```yaml
# xcaf/mcp/filesystem-server.xcaf
kind: mcp
version: "1.0"
name: filesystem-server
description: "Local filesystem access via MCP."
targets:
  claude: {}
```

The empty `{}` value is required — it marks the target as explicitly included. This MCP server is compiled for Claude only; all other targets skip it.

### Restricting a rule to one provider

A Cursor-specific rule that uses the `manual-mention` activation mode:

```yaml
# xcaf/rules/cursor-focus.xcaf
kind: rule
version: "1.0"
name: cursor-focus
description: "Focus guidance for Cursor chat workflows."
activation: manual-mention
targets:
  cursor: {}
---
When this rule is explicitly mentioned, restrict your edits to the files opened in the current editor tab only.
```

### Including two providers, excluding others

```yaml
kind: skill
version: "1.0"
name: workspace-lint
description: "Lint the current workspace."
targets:
  claude: {}
  cursor: {}
```

This skill compiles only for Claude and Cursor. Gemini, Copilot, and Antigravity receive an info note and skip it.

## Provider-Specific Overrides

For cases where a resource needs different field values per provider — a different model, a different description, a different body — use override files. An override file sits alongside the base resource and is merged into it during compilation for a specific target only.

### Naming convention

```
xcaf/agents/<name>/agent.xcaf          ← base resource
xcaf/agents/<name>/agent.<provider>.xcaf   ← provider override
```

Every override must have a corresponding base resource. An override without a base is a parse error.

### Example: different model per provider

```
xcaf/agents/developer/
├── agent.xcaf              ← base: uses 'sonnet' model
├── agent.claude.xcaf       ← Claude override: uses 'opus'
└── agent.cursor.xcaf       ← Cursor override: uses 'sonnet' (no change needed, but shown for clarity)
```

```yaml
# xcaf/agents/developer/agent.xcaf
kind: agent
version: "1.0"
name: developer
description: "General-purpose software developer."
model: sonnet
tools: [Read, Write, Edit, Bash, Glob, Grep]
```

```yaml
# xcaf/agents/developer/agent.claude.xcaf
kind: agent
version: "1.0"
name: developer
model: opus
```

When compiling for Claude, xcaffold deep-merges the override into the base: scalar fields replace, lists replace, maps deep-merge, and the body replaces. The compiled Claude agent uses `opus`; all other providers use `sonnet`.

### Example: different skill body per provider

```yaml
# xcaf/skills/code-review/skill.xcaf
kind: skill
version: "1.0"
name: code-review
description: "Code review workflow."
allowed-tools: [Read, Glob, Grep]
---
Review the diff for correctness, test coverage, and documentation.
```

```yaml
# xcaf/skills/code-review/skill.gemini.xcaf
kind: skill
version: "1.0"
name: code-review
---
Review the diff for correctness and test coverage. Use search tools to verify imported symbols exist before reporting them as issues.
```

The Gemini override replaces only the body; all frontmatter fields come from the base.

### Override merge rules

| Element | Merge behavior |
|---|---|
| Scalar field (string, number, bool) | Override value replaces base value |
| List field | Override list replaces base list entirely |
| Map field | Deep merge — override keys added or replaced, base-only keys kept |
| Body (markdown prose) | Override body replaces base body entirely |
| Bool pointer field | Override value replaces base value |

If you omit a field in the override, the base value is used unchanged.

## Understanding Fidelity Notes

Fidelity notes are the primary mechanism xcaffold uses to communicate the gap between what you authored and what a provider received.

### Viewing notes during apply

Fidelity notes print to stderr during `xcaffold apply`. Redirect stderr to a file to review them after the fact:

```bash
xcaffold apply 2>fidelity.log
```

### Suppressing expected warnings

When a field is intentionally dropped for a provider — you know it is not supported and the drop is acceptable — silence the warning using `suppress-fidelity-warnings` in the resource's `targets:` map:

```yaml
# xcaf/skills/tdd/skill.xcaf
---
kind: skill
version: "1.0"
name: tdd
description: "TDD workflow."
allowed-tools: [Bash, Read, Write, Edit]
disable-model-invocation: true
targets:
  gemini:
    suppress-fidelity-warnings: true
  copilot:
    suppress-fidelity-warnings: true
---
Follow the Red-Green-Refactor cycle.
```

`disable-model-invocation` is a Claude-only field. Without suppression, Gemini and Copilot emit a warning. With `suppress-fidelity-warnings: true`, the warning is silenced for those targets. The field is still dropped; suppression only removes the stderr noise.

### Provider-native pass-through

`targets.<provider>.provider:` is an opaque map for provider-native keys that xcaffold does not model in its schema. These are passed through verbatim to the compiled output for that provider only. xcaffold does not validate or interpret these keys.

```yaml
targets:
  copilot:
    provider:
      applyTo: "**/*.go"
```

Use pass-through fields sparingly. They create an invisible dependency on provider-native behavior that xcaffold cannot validate or migrate automatically when a provider changes its schema.

### Skipping synthesis

To skip all synthesis for a resource on a specific target without removing it from compilation:

```yaml
targets:
  antigravity:
    skip-synthesis: true
```

`skip-synthesis: true` instructs the renderer to include the resource in the target's manifest but skip any synthesis transformations. Use this when a resource should be present in the output directory but should not be post-processed by xcaffold's synthesis pipeline.

## Codex-Specific Considerations

Codex (Preview) introduces two multi-target situations that require explicit handling.

**`AGENTS.md` collision.** Both Codex and Cursor produce an `AGENTS.md` file at the project root. When both providers are active targets, xcaffold emits a `ROOT_FILE_COLLISION` fidelity note and writes only one of the two files. To avoid this, use `kind: context` resources with a `targets:` map to route project-level instructions explicitly:

```yaml
# xcaf/context/project-instructions.xcaf — Claude, Gemini, Copilot, Antigravity
kind: context
version: "1.0"
name: project-instructions
targets:
  claude: {}
  gemini: {}
  copilot: {}
  antigravity: {}
---
Your project-level instructions here.
```

```yaml
# xcaf/context/project-instructions-cursor.xcaf — Cursor only
kind: context
version: "1.0"
name: project-instructions-cursor
targets:
  cursor: {}
---
Your project-level instructions here.
```

```yaml
# xcaf/context/project-instructions-codex.xcaf — Codex only
kind: context
version: "1.0"
name: project-instructions-codex
targets:
  codex: {}
---
Your project-level instructions here.
```

**Shared `.agents/skills/` directory.** Codex and Antigravity both read skills from `.agents/skills/*/SKILL.md`. This is not a conflict — the output format is identical. A single compile pass that targets both providers writes skills once to `.agents/skills/` and both providers consume them. No additional configuration is needed.

**Rules gap.** Codex does not support rule compilation. xcaffold emits a `RENDERER_KIND_UNSUPPORTED` fidelity note for each rule when Codex is a target. Rules defined in your `.xcaf` manifests are compiled for all other declared targets and skipped for Codex. If a rule is required behavior for your workflow, scope it explicitly to the providers that support it using `targets:` so the note is suppressed.

## Capability Differences

Provider support for resource fields varies. Some fields compile to all providers; others are silently dropped or produce fidelity notes when targeting a provider that does not support them. See the [Supported Providers](../reference/supported-providers.md) reference for the full capability matrix.

Rules using `manual-mention` or `model-decided` activation should be scoped to the provider that supports them using the `targets:` map. Otherwise xcaffold compiles the rule for all providers but emits a warning that the activation encoding was omitted.

## Decision Guide

| I want to… | Approach |
|---|---|
| Deploy the same resource to all providers | Omit `targets:` entirely — the resource is included for every declared target |
| Exclude a resource from one provider | Use `targets:` listing all providers except the one to exclude |
| Include a resource for only one provider | Use `targets:` with only that provider listed |
| Use a different field value per provider (e.g. model) | Author an override file: `agent.<provider>.xcaf` next to the base `agent.xcaf` |
| Pass provider-native keys xcaffold does not model | Use `targets.<provider>.provider:` opaque map |
| Silence a known fidelity warning | Use `targets.<provider>.suppress-fidelity-warnings: true` |
| Verify fidelity issues before compiling | Run `xcaffold validate --target <provider>` |
| Check what a specific blueprint resolves to | Run `xcaffold list --blueprint <name> --resolved` |

## Related

- [Supported Providers](../reference/supported-providers.md) — full provider capability matrix and fidelity notes
- [Variables and Overrides](variables-and-overrides.md) — customizing field values per provider without duplication
- [Blueprint Design](blueprint-design.md) — narrowing compiled output to specific resource subsets
