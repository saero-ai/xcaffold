package templates

// RenderHooksReference returns an annotated template showing every field
// of the hooks kind with descriptions, types, defaults, and provider support notes.
//
// The generated content is written to .xcaffold/schemas/hooks.xcf.reference
// and is NOT parsed by xcaffold. Users copy fields from this file into their
// xcf/hooks/<name>/hooks.xcf as needed.
//
// Note: hooks uses pure YAML format (no frontmatter --- delimiters).
// Parser uses strict mode — unknown YAML keys are rejected.
func RenderHooksReference() string {
	return `# ============================================================
# Hooks Kind — Full Field Reference
# ============================================================
# This file is NOT parsed by xcaffold.
# Copy fields from here into your xcf/hooks/<name>/hooks.xcf
# Provider support: hooks are Claude-only (dropped by all others)
# ============================================================
# IMPORTANT: Hooks use pure YAML format — no frontmatter delimiters (---).
# Unknown YAML keys are rejected. Event names are case-sensitive PascalCase.
# Multiple hooks documents with the same name are MERGED (events appended).
# ============================================================

kind: hooks
version: "1.0"
name: default               # Optional. Defaults to "default" when omitted.

# ── Events ───────────────────────────────────────────────────
# events: map of event name to array of HookMatcherGroup.
# Valid event names (PascalCase, case-sensitive):
#   PreToolUse | PostToolUse | Notification | Stop | SubagentStop
#   InstructionsLoaded | PreCompact | SessionStart | ConfigChange

events:

  # ── PreToolUse: fires before any tool call ──────────────────
  PreToolUse:
    - matcher: "Bash"       # Optional. Pattern to match against event payload.
      hooks:
        - type: command     # REQUIRED on every handler.
          command: "echo 'about to run bash'"
          shell: bash       # Optional. bash | powershell
          timeout: 5000     # Optional. Milliseconds. Pointer type (omit = no timeout).
          async: false      # Optional. Three-state: true | false | omit. Pointer type.
          once: false       # Optional. When true, fires only once per session. Pointer type.
          if: ""            # Optional. Conditional expression. Hook runs only when true.
          status-message: "Running pre-tool check"  # Optional.
          allowed-env-vars: # Optional. Environment variables passed to the command.
            - "PATH"
            - "HOME"

  # ── PostToolUse: fires after any tool call ──────────────────
  PostToolUse:
    - matcher: "Write"
      hooks:
        - type: command
          command: "lint.sh"

  # ── Notification: fires on agent notifications ───────────────
  Notification:
    - hooks:
        - type: command
          command: "notify.sh"

  # ── Stop: fires when the agent stops ────────────────────────
  Stop:
    - hooks:
        - type: command
          command: "cleanup.sh"

  # ── SubagentStop: fires when a subagent stops ───────────────
  SubagentStop:
    - hooks:
        - type: command
          command: "subagent-cleanup.sh"

  # ── InstructionsLoaded: fires after instructions load ────────
  InstructionsLoaded:
    - hooks:
        - type: command
          command: "on-instructions-loaded.sh"

  # ── PreCompact: fires before context compaction ──────────────
  PreCompact:
    - hooks:
        - type: command
          command: "pre-compact.sh"

  # ── SessionStart: fires when a session starts ────────────────
  SessionStart:
    - hooks:
        - type: command
          command: "session-start.sh"

  # ── ConfigChange: fires on config changes ────────────────────
  ConfigChange:
    - hooks:
        - type: command
          command: "on-config-change.sh"

# ── Handler Types Reference ──────────────────────────────────
# Every handler requires a type field. Available types vary by provider.
#
# type: command   — run a shell command
#   command: "script.sh"
#   shell: bash
#
# type: prompt    — inject a prompt into the conversation
#   prompt: "Summarize what just happened."
#   model: "claude-sonnet-4-6"
#
# type: url       — HTTP callback
#   url: "https://webhook.example.com/hook"
#   headers:
#     Authorization: "Bearer ${TOKEN}"
`
}
