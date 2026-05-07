---
title: "kind: hooks"
description: "Declares lifecycle scripts run automatically before or after tool invocations. Compiled into provider-native hook configuration. Produces no standalone output files."
---

# `kind: hooks`

Declares lifecycle scripts run automatically before or after tool invocations. Hook declarations are merged into each provider's native hook configuration file at compile time. They produce **no standalone output files**.

Uses **pure YAML format** (no frontmatter `---` delimiters).

> **Required:** `kind`, `version`, `name`

## Source Directory

```
xcaf/hooks/hooks.xcaf
```

`kind: hooks` is a singleton — there is no `<name>` directory. All hook declarations for a project are defined in a single `xcaf/hooks/hooks.xcaf` file.

## Example Usage

### Lint after every file edit

```yaml
kind: hooks
version: "1.0"
name: frontend-app-hooks
post-tool-call:
  - matcher: "Edit|Write"
    run: '"$PROJECT_DIR"/.claude/hooks/post-edit-auto-lint.sh'
```

### Full hooks — lint, git safety, worktree enforcement

```yaml
kind: hooks
version: "1.0"
name: frontend-app-hooks
pre-tool-call:
  - matcher: "Bash"
    run: '"$PROJECT_DIR"/.claude/hooks/enforce-gitops.sh'
  - matcher: "Write|Edit"
    run: '"$PROJECT_DIR"/.claude/hooks/enforce-worktree.sh'
post-tool-call:
  - matcher: "Edit|Write"
    run: '"$PROJECT_DIR"/.claude/hooks/post-edit-auto-lint.sh'
```

## Argument Reference

The following arguments are supported:

- `name` — (Required) Unique hooks identifier. Must match `[a-z0-9-]+`.
- `pre-tool-call` — (Optional) `[]HookEntry`. Scripts run before any matching tool invocation.
- `post-tool-call` — (Optional) `[]HookEntry`. Scripts run after any matching tool invocation.
- `targets` — (Optional) `map[string]TargetOverride`. Per-provider overrides. Hooks rarely use `targets:` since lifecycle scripts are typically provider-universal, but the field is available for cases where hook behavior differs across providers.

### `HookEntry`

Each entry in `pre-tool-call` or `post-tool-call` supports:

- `matcher` — (Optional) `string`. Pipe-separated tool name pattern (e.g., `"Edit|Write"`, `"Bash"`). Omit to match all tools.
- `run` — (Required) `string`. Shell command to execute. Use `$PROJECT_DIR` for the project root path.

## Compiled Output

Hooks are merged into provider-native configuration files during compilation.

<ProviderTabs
  claude={`{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          { "type": "command", "command": "\\"$PROJECT_DIR\\"/.claude/hooks/enforce-gitops.sh" }
        ]
      }
    ]
  }
}`}
  cursor={`{
  "hooks": {
    "beforeTool": [
      {
        "matcher": "Bash",
        "command": "\\"$PROJECT_DIR\\"/.cursor/hooks/enforce-gitops.sh"
      }
    ]
  }
}`}
  github={`{
  "preToolUse": [
    {
      "matcher": "Bash",
      "command": "\\"$PROJECT_DIR\\"/.github/hooks/enforce-gitops.sh"
    }
  ]
}`}
  gemini={`{
  "hooks": {
    "beforeTool": [
      {
        "matcher": "Bash",
        "command": "\\"$PROJECT_DIR\\"/.gemini/hooks/enforce-gitops.sh"
      }
    ]
  }
}`}
  antigravity={`{
  "hooks": {
    "preToolCall": [
      {
        "matcher": "Bash",
        "command": "\\"$PROJECT_DIR\\"/.agents/hooks/enforce-gitops.sh"
      }
    ]
  }
}`}
/>

> [!WARNING]
> Hook shell scripts referenced in `run` are **not** created by xcaffold — you must author and commit them to your repository separately. The `run` command is only registered in the provider's hook configuration; if the referenced script does not exist, the provider will error at runtime.
