---
title: "Best Practices"
description: "Recommended configuration patterns and operational conventions for xcaffold projects."
---

# Best Practices

Targeted recommendations for when to use specific patterns within xcaffold. Each guide is grounded in the actual compiler, parser, and schema — not generalized advice.

- [Project Structure](project-layouts) — From quickstart flat files to domain-driven structures for large teams.
- [Workspace Context](workspace-context) — Using `kind: context` to anchor agent awareness without polluting every interaction.
- [Skill Organization](skill-organization) — The required subdirectory layout, when to use `references/`, `scripts/`, `assets/`, and `examples/`.
- [Blueprint Design](blueprint-design) — Opt-in subset selection for large projects. Includes an experimental label for upcoming marketplace features.
- [Policy Organization](policy-organization) — Writing `kind: policy` constraints with the correct `target`, `match`, `require`, and `deny` fields.
- [Agent Design Patterns](agent-design-patterns) — Specialized agents, composition with skills and rules, tool scoping, and per-provider overrides.
- [Rule Organization](rule-organization) — Activation modes, path-scoped rules, provider fallback behavior, and file layout patterns.
- [Variables and Overrides](variables-and-overrides) — Injecting shared values with variable files and customizing resources per provider with override manifests.
- [Multi-Target Compilation](multi-target-compilation) — Managing a single manifest set that compiles to multiple providers, with fidelity notes and target scoping.
