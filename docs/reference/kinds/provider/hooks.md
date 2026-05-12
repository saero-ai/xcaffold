---
title: "kind: hooks"
description: "Declares lifecycle scripts run automatically before or after tool invocations. Compiled into provider-native hook configuration at compile time."
---

# `kind: hooks`

Declares lifecycle scripts that run automatically at defined points in a provider session — before tool calls, after tool calls, on session start, and so on. Xcaffold compiles hook declarations into each provider's native hook configuration at compile time. Hooks produce **no standalone output files**; they are merged into existing provider config.

Uses **pure YAML format** (no frontmatter `---` delimiters).

## Source Directory

```
xcaf/hooks/<name>/hooks.xcaf
```

Each named hook block lives in its own subdirectory. Script files referenced by hook commands should be placed in the same directory alongside the manifest and listed under `artifacts:` so xcaffold copies them into the provider hook directory during compilation.

## Example Usage

### Validate bash commands before execution

```yaml
kind: hooks
version: "1.0"
name: project-hooks
description: "Project-level lifecycle hooks."

events:
  PreToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: $XCAF_PROJECT_DIR/xcaf/hooks/project-hooks/scripts/validate-bash.sh
          timeout: 10
          status-message: "Validating bash command"
  PostToolUse:
    - matcher: "Edit"
      hooks:
        - type: command
          command: $XCAF_PROJECT_DIR/xcaf/hooks/project-hooks/scripts/post-edit-lint.sh
          async: true
          timeout: 15
```

### Full example with multiple events

```yaml
kind: hooks
version: "1.0"
name: golden-hooks
description: "Example hook block with lifecycle event handlers."
artifacts: [scripts, templates]

events:
  PreToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: "echo 'pre-tool-use: bash about to run'"
          async: false
          timeout: 10
          once: false
          if: "true"
          shell: /bin/sh
          status-message: "Validating bash command"
          allowed-env-vars: [PATH, HOME, TERM]
        - type: command
          command: ./scripts/pre-bash-check.sh
          async: true
          timeout: 5
  PostToolUse:
    - matcher: "Edit"
      hooks:
        - type: command
          command: "echo 'post-tool-use: edit complete'"
          timeout: 15
          status-message: "Formatting after edit"
    - matcher: "Write"
      hooks:
        - type: command
          command: ./scripts/post-write.sh
          async: true
          timeout: 30
  SessionStart:
    - matcher: ""
      hooks:
        - type: command
          command: echo "session started"
          once: true
          timeout: 5
          status-message: "Initializing session"
          allowed-env-vars: [HOME, USER, PATH]
  Stop:
    - matcher: ""
      hooks:
        - type: command
          command: echo "session stopping"
          timeout: 5
  Notification:
    - matcher: ""
      hooks:
        - type: command
          command: ./scripts/notify.sh
          async: true
          timeout: 10
```

## Field Reference

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | `string` | Unique identifier for this hook block. Must match `[a-z0-9-]+`. |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `description` | `string` | Human-readable description of this hook block's purpose. |
| `artifacts` | `[]string` | Named subdirectories within `xcaf/hooks/<name>/` to copy into the provider hook directory during compilation (e.g., `scripts`, `templates`). |
| `events` | `map[string][]HookMatcherGroup` | Lifecycle event handlers keyed by event name. See [Events](#events) below. |
| `targets` | `map[string]TargetOverride` | Per-provider behavioral overrides for this hook block. |

### Events

`events:` is a map from event name to an array of `HookMatcherGroup` objects. Event names are PascalCase:

| Event Name | When It Fires |
|------------|--------------|
| `PreToolUse` | Before a tool call executes. |
| `PostToolUse` | After a tool call completes. |
| `SessionStart` | When a provider session begins. |
| `Stop` | When a provider session ends. |
| `SubagentStop` | When a subagent finishes. |
| `InstructionsLoaded` | After system instructions are loaded at session start. |
| `PreCompact` | Before context compaction runs. |
| `Notification` | When the provider emits a notification. |
| `ConfigChange` | When provider configuration changes at runtime. |

### HookMatcherGroup

Each entry in an event array is a matcher group. The group runs when the tool name matches `matcher`.

| Field | Type | Description |
|-------|------|-------------|
| `matcher` | `string` | Tool name to match (e.g., `"Bash"`, `"Edit"`, `"Write"`). Empty string matches all tools. Optional. |
| `hooks` | `[]HookHandler` | One or more handlers to execute when the matcher fires. Required. |

### HookHandler

Each entry in `hooks:` defines a single executable action.

| Field | Type | Description |
|-------|------|-------------|
| `type` | `string` | Handler type. Use `"command"` for shell commands. Required. |
| `command` | `string` | Shell command to execute. Use `$XCAF_PROJECT_DIR` for the project root. |
| `url` | `string` | URL to invoke for webhook-style handlers. |
| `prompt` | `string` | Prompt text passed to the handler. |
| `model` | `string` | Model identifier used when the handler invokes a model. |
| `headers` | `map[string]string` | HTTP headers sent when `url` is set. |
| `async` | `bool` | When `true`, the hook runs without blocking the provider. Default: `false`. |
| `timeout` | `int` | Maximum execution time in milliseconds before the hook is killed. |
| `once` | `bool` | When `true`, the hook fires only on the first matching event in a session. |
| `shell` | `string` | Shell binary to use (e.g., `/bin/sh`, `/bin/bash`). Provider default when omitted. |
| `status-message` | `string` | Text displayed in the provider UI while the hook is running. |
| `allowed-env-vars` | `[]string` | Environment variable names passed to the hook process. |
| `if` | `string` | Conditional expression. Hook is skipped when this evaluates to false. |

### Variable substitution

Use `$XCAF_PROJECT_DIR` in `command` values to refer to the project root directory. xcaffold rewrites this to the provider-native equivalent during compilation:

| Provider | Compiled To |
|----------|------------|
| Claude | `$CLAUDE_PROJECT_DIR` |
| Gemini | `$GEMINI_PROJECT_DIR` |
| Cursor | `$CURSOR_PROJECT_DIR` |
| Copilot | `$GITHUB_WORKSPACE` |

For backward compatibility, `$CLAUDE_PROJECT_DIR` in `.xcaf` source is also translated to the target provider's variable.

## Compiled Output

Hook declarations are merged into provider-native configuration at compile time.

### Claude

**Output path**: `.claude/settings.json` (`hooks` key)

`$XCAF_PROJECT_DIR` is translated to `$CLAUDE_PROJECT_DIR`. Event names and hook structure are preserved as-is (Claude Code's native format matches the xcaffold schema).

```json
{
  "$schema": "https://json.schemastore.org/claude-code-settings.json",
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "$CLAUDE_PROJECT_DIR/xcaf/hooks/project-hooks/scripts/validate-bash.sh",
            "timeout": 10,
            "statusMessage": "Validating bash command"
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Edit",
        "hooks": [
          {
            "type": "command",
            "command": "$CLAUDE_PROJECT_DIR/xcaf/hooks/project-hooks/scripts/post-edit-lint.sh",
            "async": true,
            "timeout": 15
          }
        ]
      }
    ]
  }
}
```

### Gemini

**Output path**: `.gemini/settings.json` (`hooks` key)

Event names are translated: `PreToolUse` → `BeforeTool`, `PostToolUse` → `AfterTool`. `$XCAF_PROJECT_DIR` is rewritten to `$GEMINI_PROJECT_DIR`.

```json
{
  "hooks": {
    "BeforeTool": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "$GEMINI_PROJECT_DIR/xcaf/hooks/project-hooks/scripts/validate-bash.sh",
            "timeout": 10
          }
        ]
      }
    ],
    "AfterTool": [
      {
        "matcher": "Edit",
        "hooks": [
          {
            "type": "command",
            "command": "$GEMINI_PROJECT_DIR/xcaf/hooks/project-hooks/scripts/post-edit-lint.sh",
            "timeout": 15
          }
        ]
      }
    ]
  }
}
```

### Cursor

**Output path**: `.cursor/hooks.json`

Event names are translated to camelCase: `PreToolUse` → `preToolUse`, `PostToolUse` → `postToolUse`. The xcaffold three-level structure (event → matcher groups → handlers) is flattened to two levels. `$XCAF_PROJECT_DIR` is rewritten to `$CURSOR_PROJECT_DIR`.

```json
{
  "version": 1,
  "preToolUse": [
    {
      "type": "command",
      "command": "$CURSOR_PROJECT_DIR/xcaf/hooks/project-hooks/scripts/validate-bash.sh",
      "timeout": 10,
      "statusMessage": "Validating bash command"
    }
  ],
  "postToolUse": [
    {
      "type": "command",
      "command": "$CURSOR_PROJECT_DIR/xcaf/hooks/project-hooks/scripts/post-edit-lint.sh",
      "async": true,
      "timeout": 15
    }
  ]
}
```

### Copilot

**Output path**: `.github/hooks/xcaffold-hooks.json`

The `command` field is mapped to `bash`. Timeout is converted from milliseconds to seconds. `$XCAF_PROJECT_DIR` is rewritten to `$GITHUB_WORKSPACE`.

```json
{
  "version": 1,
  "hooks": {
    "preToolUse": [
      {
        "type": "command",
        "bash": "$GITHUB_WORKSPACE/xcaf/hooks/project-hooks/scripts/validate-bash.sh",
        "timeoutSec": 0
      }
    ],
    "postToolUse": [
      {
        "type": "command",
        "bash": "$GITHUB_WORKSPACE/xcaf/hooks/project-hooks/scripts/post-edit-lint.sh"
      }
    ]
  }
}
```

### Antigravity

Antigravity does not support hooks. Xcaffold emits a `RENDERER_KIND_UNSUPPORTED` fidelity note and produces no hook output for that target.

## Provider Support

| Provider | Supported | Output File | Event Name Style |
|----------|-----------|-------------|-----------------|
| Claude | Yes | `.claude/settings.json` (`hooks` key) | PascalCase (`PreToolUse`) |
| Gemini | Yes | `.gemini/settings.json` (`hooks` key) | Translated (`BeforeTool`, `AfterTool`) |
| Cursor | Yes | `.cursor/hooks.json` | camelCase (`preToolUse`) |
| Copilot | Yes | `.github/hooks/xcaffold-hooks.json` | camelCase (`preToolUse`) |
| Antigravity | No | — | — |

### Provider event name mappings

| xcaffold event | Claude | Gemini | Cursor | Copilot |
|----------------|--------|--------|--------|---------|
| `PreToolUse` | `PreToolUse` | `BeforeTool` | `preToolUse` | `preToolUse` |
| `PostToolUse` | `PostToolUse` | `AfterTool` | `postToolUse` | `postToolUse` |
| `SessionStart` | `SessionStart` | `SessionStart` | `sessionStart` | `sessionStart` |
| `Stop` | `Stop` | —¹ | `stop` | `agentStop` |
| `SubagentStop` | `SubagentStop` | —¹ | `subagentStop` | `subagentStop` |
| `InstructionsLoaded` | `InstructionsLoaded` | —¹ | camelCase fallback² | —¹ |
| `PreCompact` | `PreCompact` | —¹ | camelCase fallback² | —¹ |
| `ConfigChange` | `ConfigChange` | —¹ | camelCase fallback² | —¹ |
| `Notification` | `Notification` | `Notification` | camelCase fallback² | —¹ |

¹ Event is dropped and xcaffold emits a `CodeFieldUnsupported` fidelity warning. The event does not appear in the compiled output.

² Cursor has no verified mapping for this event. Xcaffold emits the event name in camelCase with a `CodeFieldUnsupported` fidelity warning advising verification against Cursor documentation.

> [!WARNING]
> Hook scripts referenced in `command` are **not** created by xcaffold. You must author and commit them to your repository. If a referenced script is absent, the provider will error at runtime.
