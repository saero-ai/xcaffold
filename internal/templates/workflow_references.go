package templates

// RenderWorkflowReference returns an annotated template showing every field
// of the workflow kind with descriptions, types, defaults, and provider support notes.
//
// The generated content is written to .xcaffold/schemas/workflow.xcf.reference
// and is NOT parsed by xcaffold. Users copy fields from this file into their
// xcf/workflows/<name>/<name>.xcf as needed.
func RenderWorkflowReference() string {
	return `# ============================================================
# Workflow Kind — Full Field Reference
# ============================================================
# This file is NOT parsed by xcaffold.
# Copy fields from here into your xcf/workflows/<name>/<name>.xcf
# Provider support: YES = compiled, dropped = silently removed
# ============================================================

---
kind: workflow
version: "1.0"

# ── Identity ─────────────────────────────────────────────────
api-version: workflow/v1    # Optional. Schema shape discriminator. Default: workflow/v1.
name: my-workflow           # REQUIRED. Lowercase + hyphens. Pattern: ^[a-z0-9-]+$
description: "..."          # Optional. What this workflow does.

# ── Multi-Target (per-provider overrides) ────────────────────
# targets: per-provider overrides and lowering-strategy directives.
# targets:
#   claude:
#     instructions-override: |
#       Claude-specific version of this workflow.

# ── Steps (mutually exclusive with body) ─────────────────────
# steps: ordered procedural body for multi-step workflows.
# Mutually exclusive with the top-level body below.
# Each step requires a name field.
# Step bodies are populated by the parser from ## heading content.
#
# steps:
#   - name: step-one
#     description: "First step."
#     # body is populated by the parser from ## step-one heading content
#   - name: step-two
#     description: "Second step."

# ── Instructions (mutually exclusive with steps) ──────────────
---
Top-level workflow instructions here. This is the body for single-step
or legacy workflows.

IMPORTANT: steps: and body are mutually exclusive. Use one or the other.

## step-one
If using steps:, each ## heading populates the body of the matching step.
The heading name must match the step name exactly.

## step-two
Instructions for step two go here.
`
}
