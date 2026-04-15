package templates

// RenderSkillReference returns an annotated template showing every field
// of the skill kind with descriptions, types, defaults, and provider notes.
//
// The generated content is written to xcf/references/skill.xcf.reference and
// is NOT parsed by xcaffold. Users copy fields from this file into their
// scaffold.xcf as needed.
func RenderSkillReference() string {
	return `# ============================================================
# Skill Kind — Full Field Reference
# ============================================================
# This file is NOT parsed by xcaffold.
# Copy fields from here into your scaffold.xcf as needed.
# Field definitions follow the xcaffold schema specification.
# Spec: docs/superpowers/specs/2026-04-15-skill-kind-cross-provider-schema-design.md
# ============================================================

---
kind: skill
version: "1.0"

# ── Identity (xcaffold envelope) ─────────────────────────────
name: my-skill              # REQUIRED. Unique ID, lowercase + hyphens.
description: "..."          # Recommended. What the skill does and when to use it.
when-to-use: "..."            # Optional. Detailed activation guidance (Claude appends to description).
license: MIT                # Optional. SPDX identifier (Cursor, Copilot).

# ── Tool Access ──────────────────────────────────────────────
allowed-tools: [Read, Grep]  # Pre-approved tool list (Claude, Copilot).
                            # Renderer emits as "allowed-tools" for Claude/Copilot conventions.

# ── Permissions & Invocation Control ─────────────────────────
disable-model-invocation: false  # If true, only user can invoke via /skill-name (Claude, Cursor).
user-invocable: true         # If false, only the model can invoke (no slash command) (Claude).
argument-hint: "[arg]"       # Autocomplete hint shown on the slash command (Claude).

# ── Composition / Supporting Files (agentskills.io convention) ─
references:                 # Glob patterns — copied to skills/<id>/references/.
  - "docs/skill-refs/*.md"
scripts:                    # Glob patterns — copied to skills/<id>/scripts/ as executable helpers.
  - "scripts/skill-helpers/*.sh"
assets:                     # Glob patterns — copied to skills/<id>/assets/ as static files.
  - "templates/*.tmpl"

# ── Multi-Target (per-provider overrides + provider: pass-through) ─
targets:
  claude:
    provider:               # Claude-only execution context — opaque to parser, validated by renderer.
      context: fork         # Run skill in a subagent (preserve main context).
      agent: Explore        # Subagent type (Explore / Plan / general-purpose / custom).
      model: sonnet         # Model alias when context: fork (sonnet/opus/haiku) or full ID.
      effort: medium        # low / medium / high / max (max = Opus only).
      shell: bash           # bash / powershell — for dynamic context injection.
      paths: ["docs/**"]    # Path-specific rules.
      # hooks: { ... }      # Skill-scoped lifecycle hooks (same shape as top-level hooks).
  cursor:
    provider:
      compatibility: "cursor >= 2.4"
      metadata:
        category: ops

# ── Instructions (ALWAYS last — mutually exclusive) ──────────
instructions: |
  Inline SKILL.md body content goes here.
# instructions_file: ./SKILL.md   # OR external file reference (not both).
`
}
