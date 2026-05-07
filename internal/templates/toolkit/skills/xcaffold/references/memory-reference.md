# ============================================================
# Memory Kind — Full Field Reference
# ============================================================
# This file is NOT parsed by xcaffold.
# Memory files live at: xcaf/agents/<agent-id>/memory/<id>.md
# Provider support: Claude only (dropped by all others)
# ============================================================
# IMPORTANT: Memory uses Content (not Body) for its text field.
# Content is populated by the compiler from the .md file body,
# NOT from frontmatter extraction. The compiler discovers .md files
# under xcaf/agents/<agent-id>/memory/ automatically.
#
# Memory has NO targets (per-provider overrides).
# Memory does NOT participate in extends: global inheritance.
# ============================================================
#
# File convention: xcaf/agents/<agent-id>/memory/<id>.md
# Example:         xcaf/agents/xaff/memory/project-context.md
#
# ── Frontmatter Fields ───────────────────────────────────────

---
kind: memory
version: "1.0"

# ── Identity ─────────────────────────────────────────────────
name: my-memory-entry       # REQUIRED. Lowercase + hyphens. Pattern: ^[a-z0-9-]+$
description: "..."          # Optional. What this memory entry contains.

# ── Content (compiler-populated) ─────────────────────────────
# The content field is NOT a YAML field — it is populated by the
# compiler's filesystem scan at compile time from the markdown body below.
# Do NOT set content: in the frontmatter. Write your content below ---.

---
Your memory content goes here. This plain markdown text becomes the
content of the compiled memory entry injected into the agent's context.

Use memory entries to store:
- Project-specific context the agent should always remember
- Persistent facts about the codebase or team conventions
- Long-lived instructions that should survive context compaction
