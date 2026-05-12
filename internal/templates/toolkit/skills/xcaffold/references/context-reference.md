# ============================================================
# Context Kind — Full Field Reference
# ============================================================
# This file is NOT parsed by xcaffold.
# Copy fields from here into your xcaf/contexts/<name>/context.xcaf
# Context compiles into the provider's root instruction file
# (CLAUDE.md for Claude, GEMINI.md for Gemini, AGENTS.md for Antigravity).
# ============================================================

---
kind: context
version: "1.0"

# ── Identity ─────────────────────────────────────────────────
name: my-context             # REQUIRED. Lowercase + hyphens. Pattern: ^[a-z0-9-]+$
description: "..."           # Optional. Purpose of this context block.

# ── Behavior ─────────────────────────────────────────────────
default: false              # When true, acts as tie-breaker if multiple contexts match.

# ── Targeting ────────────────────────────────────────────────
# targets: list of provider names this context applies to.
# Unlike agents/skills/rules, context targets is a STRING ARRAY, not a map.
# Omit to compile for ALL configured providers.
# targets:
#   - claude
#   - cursor
#   - gemini

# ── Instructions (always last) ───────────────────────────────
---
Your context instructions here. This body text is compiled verbatim
into the provider's root instruction file (e.g., CLAUDE.md).

Use kind: context for workspace-level ambient instructions shared
across all agents — project conventions, coding standards, tool policies.
