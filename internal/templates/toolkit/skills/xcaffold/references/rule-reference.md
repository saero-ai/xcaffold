# ============================================================
# Rule Kind — Full Field Reference
# ============================================================
# This file is NOT parsed by xcaffold.
# Copy fields from here into your xcf/rules/<name>/<name>.xcf
# Provider support: YES = compiled, dropped = silently removed
# ============================================================

---
kind: rule
version: "1.0"

# ── Identity ─────────────────────────────────────────────────
name: my-rule               # REQUIRED. Lowercase + hyphens. Pattern: ^[a-z0-9-]+$
description: "..."          # Optional. What this rule enforces.

# ── Activation ───────────────────────────────────────────────
# activation: controls when this rule is injected into agent context.
# Enum: always | path-glob | model-decided | manual-mention | explicit-invoke
#
#   always          — injected on every request (replaces always-apply: true)
#   path-glob       — injected when the active file matches a path in paths:
#   model-decided   — model decides whether to apply based on context
#   manual-mention  — user must explicitly mention the rule
#   explicit-invoke — programmatic invocation only
#
activation: always          # claude: YES, cursor: YES, gemini: YES, copilot: YES, antigravity: YES

# paths: required when activation is path-glob.
# paths:                    # claude: YES, cursor: YES, gemini: dropped, copilot: YES, antigravity: YES
#   - "src/**"
#   - "lib/**"

# always-apply: LEGACY — prefer activation: always for new manifests.
# Pointer type: omitting differs from false.
# always-apply: true        # claude: YES, cursor: YES, gemini: YES, copilot: YES, antigravity: YES

# ── Provider-Specific ────────────────────────────────────────
# exclude-agents: Copilot only. List of agent types that should NOT receive this rule.
# Silently ignored by all non-Copilot renderers.
# enum: code-review | cloud-agent
# exclude-agents:           # claude: dropped, cursor: dropped, gemini: dropped, copilot: YES, antigravity: dropped
#   - code-review

# ── Multi-Target (per-provider overrides) ────────────────────
# targets: keys are provider names: claude, cursor, copilot, gemini, antigravity
# targets:
#   cursor:
#     instructions-override: |
#       Cursor-specific version of this rule.

# ── Instructions (always last) ───────────────────────────────
---
Your rule instructions here. This becomes the rule body injected into
agent context when the activation condition is met.

All providers that support this rule will receive this body verbatim,
unless overridden via targets: above.
