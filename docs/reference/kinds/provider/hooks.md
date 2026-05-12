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

## Argument Reference

### Top-level fields

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `kind` | Yes | `string` | Always `hooks`. |
| `version` | Yes | `string` | Schema version. Must be `"1.0"` (quoted). |
| `name` | No | `string` | Unique identifier for this hook block. Must match `[a-z0-9-]+`. Defaults to `"default"`. |
| `description` | No | `string` | Human-readable description of this hook block's purpose. |
| `artifacts` | No | `[]string` | Named subdirectories within `xcaf/hooks/<name>/` to copy into the provider hook directory during compilation (e.g., `scripts`, `templates`). |
| `targets` | No | `map[string]TargetOverride` | Per-provider overrides keyed by provider name. |
| `events` | No | `map[string][]HookMatcherGroup` | Lifecycle event handlers keyed by event name. See [Events](#events) below. |

### Events

`events:` is a map from event name to an array of `HookMatcherGroup` objects. Event names are PascalCase:

| Event Name | When It Fires |
|------------|--------------|
| `PreToolUse` | Before a tool call executes. |
| `PostToolUse` | After a tool call completes. |
| `SessionStart` | When a provider session begins. |
| `Stop` | When a provider session ends. |
| `Notification` | When the provider emits a notification. |

### `HookMatcherGroup`

Each entry in an event array is a matcher group. The group runs when the tool name matches `matcher`.

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `matcher` | No | `string` | Tool name to match (e.g., `"Bash"`, `"Edit"`, `"Write"`). Empty string matches all tools. |
| `hooks` | Yes | `[]HookHandler` | One or more handlers to execute when the matcher fires. |

### `HookHandler`

Each entry in `hooks:` defines a single executable action.

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `type` | Yes | `string` | Handler type. Use `"command"` for shell commands. |
| `command` | No | `string` | Shell command to execute. Use `$XCAF_PROJECT_DIR` for the project root. |
| `async` | No | `bool` | When `true`, the hook runs without blocking the provider. Default: `false`. |
| `timeout` | No | `int` | Maximum execution time in milliseconds before the hook is killed. |
| `once` | No | `bool` | When `true`, the hook fires only on the first matching event in a session. |
| `shell` | No | `string` | Shell binary to use (e.g., `/bin/sh`, `/bin/bash`). Provider default when omitted. |
| `status-message` | No | `string` | Text displayed in the provider UI while the hook is running. |
| `allowed-env-vars` | No | `[]string` | Environment variable names passed to the hook process. |
| `if` | No | `string` | Conditional expression. Hook is skipped when this evaluates to false. |

### Variable substitution

Use `$XCAF_PROJECT_DIR` in `command` values to refer to the project root directory. Xcaffold rewrites this to the provider-native equivalent during compilation:

| Provider | Expanded Value |
|----------|---------------|
| Claude | `$CLAUDE_PROJECT_DIR` |
| Gemini | `$GEMINI_PROJECT_DIR` |
| Cursor | `$CURSOR_PROJECT_DIR` |
| Copilot | `$GITHUB_WORKSPACE` |

## Compiled Output

Hook declarations are merged into provider-native configuration at compile time.

<ProviderTabs
  claude={`// Merged into .claude/settings.json under the "hooks" key.
{
  "$schema": "https://cdn.jsdelivr.net/npm/@anthropic-ai/claude-code@latest/config-schema.json",
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
}`}
  gemini={`// Merged into .gemini/settings.json under the "hooks" key.
// Event names are translated: PreToolUse → BeforeTool, PostToolUse → AfterTool.
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
}`}
  cursor={`// Written to .cursor/hooks.json.
// Event names are translated to camelCase: PreToolUse → preToolUse, PostToolUse → postToolUse.
// The 3-level xcaffold structure (event → matcher groups → handlers) is flattened to 2 levels.
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
}`}
  github={`// Written to .github/hooks/xcaffold-hooks.json.
// The "command" field is mapped to "bash". Timeout is converted from milliseconds to seconds.
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
}`}
/>

> [!WARNING]
> Hook scripts referenced in `command` are **not** created by xcaffold. You must author and commit them to your repository. If a referenced script is absent, the provider will error at runtime.

> [!NOTE]
> Antigravity does not support hooks. Xcaffold emits a `RENDERER_KIND_UNSUPPORTED` fidelity note and produces no hook output for that target.

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
| `Stop` | `Stop` | — | `stop` | `agentStop` |
| `Notification` | `Notification` | `Notification` | camelCase fallback | — |
