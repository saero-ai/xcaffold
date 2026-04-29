---
title: "Best Practices"
description: "Recommended configuration patterns and operational conventions for xcaffold projects."
---

# Best Practices

Targeted recommendations for when to use specific patterns within xcaffold. Each guide is grounded in the actual compiler, parser, and schema — not generalized advice.

- [Project Layouts](project-layouts) — From single-file setups to domain-driven structures, with the parser rules that enforce them.
- [Workspace Context](workspace-context) — Using `kind: context` to anchor agent awareness without polluting every interaction.
- [Skill Organization](skill-organization) — The required subdirectory layout, when to use `references/`, `scripts/`, `assets/`, and `examples/`.
- [Blueprint Design](blueprint-design) — Opt-in subset selection for large projects. Includes an experimental label for upcoming marketplace features.
- [Policy Organization](policy-organization) — Writing `kind: policy` constraints with the correct `target`, `match`, `require`, and `deny` fields.
- [Cross-Provider Deployment](cross-provider) — Compiling one source to multiple AI providers, understanding fidelity notes, and using per-target overrides.
