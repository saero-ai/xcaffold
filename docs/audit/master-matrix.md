# Documentation Refresh — Master Audit Matrix

**Date:** 2026-05-07
**Audit Reports:** 10 domain audits (1A-1J)
**Total Files Audited:** 66

## Summary Dashboard

| Domain | Files | Current | Needs-Update | Needs-Rewrite | Redundant | Missing |
|--------|-------|---------|-------------|--------------|-----------|---------|
| Root (1A) | 5 | 2 | 3 | 0 | 0 | 0 |
| Ref: Commands Diag+Life (1B) | 8 | 6 | 1 | 1 | 0 | 0 |
| Ref: Commands Utility (1C) | 7 | 4 | 3 | 0 | 0 | 0 |
| Ref: Provider Kinds (1D) | 10 | 8 | 2 | 0 | 0 | 0 |
| Ref: xcaffold Kinds (1E) | 8 | 2 | 5 | 1 | 0 | 0 |
| Concepts: Architecture (1F) | 6 | 3 | 0 | 3 | 0 | 0 |
| Concepts: Configuration (1G) | 6 | 6 | 0 | 0 | 0 | 0 |
| Concepts: Execution (1H) | 4 | 3 | 1 | 0 | 0 | 0 |
| Tutorials (1I) | 7 | 3 | 4 | 0 | 0 | 0 |
| Best Practices (1J) | 6 | 3 | 2 | 0 | 0 | 1 |
| **TOTAL** | **67** | **40** | **21** | **5** | **0** | **1** |

## Priority Distribution

| Priority | Count | Description |
|----------|-------|-------------|
| P1 | 22 | User-facing accuracy issues: wrong flags, phantom fields, broken examples, invented schemas, stale command refs |
| P2 | 10 | Completeness gaps: missing fields, incomplete coverage, broken navigation links, version misalignment |
| P3 | 3 | Style issues: singular vs plural flag names, formatting inconsistencies |

## Cross-Cutting Issues

### 1. Broken cross-references (12+ instances)
Tutorials reference a non-existent `docs/how-to/` directory (6 links), wrong relative paths within tutorials (6 links). Best practices index links to non-existent `cross-provider.md`. Reference index links to non-existent `how-to/index.md`.

### 2. Non-existent BIR package documented (3 files)
`architecture.md`, `translation-pipeline.md`, and `multi-target-rendering.md` reference an `internal/bir/` package that does not exist. Functions like `bir.ImportWorkflow()`, `bir.DetectIntents()`, `bir.ReassembleWorkflow()` are fabricated.

### 3. Pure-YAML kind examples use frontmatter delimiters
`blueprint.md` and `policy.md` wrap examples in `---` delimiters. These are pure-YAML kinds — `KnownFields(true)` rejects frontmatter. Users copying examples get parse errors.

### 4. Outdated CapabilitySet struct (provider-architecture.md)
Documents `ModelField` (removed), `SkillSubdirs: []string` (now `SkillArtifactDirs: map[string]string`). Missing `RuleEncoding` and `AgentNativeToolsOnly` fields.

### 5. `.xcf` remnants in filesystem-as-schema sections
`blueprint.md` and `policy.md` reference `blueprint.xcf`/`policy.xcf` instead of `.xcaf`.

### 6. Missing reference pages (linked but not created)
`docs/reference/kinds/xcaffold/settings.md` and `docs/reference/kinds/xcaffold/hooks.md` are linked from indexes but do not exist.

### 7. Provider capability matrix partially inconsistent with manifests
`supported-providers.md` was written before manifest-driven registry stabilized. Copilot hooks and Antigravity settings entries conflict with `KindSupport` maps.

### 8. `xcaffold diff` command referenced but removed
README.md references `xcaffold diff` twice. It was replaced by `xcaffold status`. Likely referenced in other docs too (tutorials, concepts). Cross-cutting terminology scan in Plan 4 will catch all instances.

### 9. Version misalignment across sources
`main.go` says `0.2.0-dev`, `.release-please-manifest.json` says `0.1.0`, CHANGELOG says `[1.0.0-dev]`. Three authoritative sources disagree.

### 10. global.md invents a ResourceRef schema
Documents a `ResourceRef{id, path}` type that does not exist. Real format uses inline resource maps. Users following this will write invalid YAML rejected by the parser.

---

## Per-File Fix List

### Fix Group A: Reference Commands (13 files)

| # | File | Domain | Verdict | Priority | Key Issues | Effort |
|---|------|--------|---------|----------|------------|--------|
| 1 | reference/commands/index.md | 1B | Current | — | No changes needed | — |
| 2 | reference/commands/diagnostic/index.md | 1B | Current | — | No changes needed | — |
| 3 | reference/commands/diagnostic/graph.md | 1B | Needs-Rewrite | P1 | 7 phantom flags, 4 real flags missing, wrong behavior description | large |
| 4 | reference/commands/diagnostic/list.md | 1B | Current | P3 | Minor: --hooks/--hook singular vs plural | small |
| 5 | reference/commands/diagnostic/status.md | 1B | Current | — | No changes needed | — |
| 6 | reference/commands/lifecycle/index.md | 1B | Needs-Update | P2 | 2 broken link paths, incomplete command list | small |
| 7 | reference/commands/lifecycle/apply.md | 1B | Current | — | No changes needed | — |
| 8 | reference/commands/lifecycle/import.md | 1B | Current | P3 | Minor: --hooks/--hook singular vs plural | small |
| 9 | reference/commands/lifecycle/init.md | 1C | Needs-Update | P1 | Missing --template and --no-policies flags | medium |
| 10 | reference/commands/lifecycle/validate.md | 1C | Needs-Update | P1 | Missing --var-file, --blueprint flags | medium |
| 11 | reference/commands/utility/index.md | 1C | Current | — | No changes needed | — |
| 12 | reference/commands/utility/export.md | 1C | Needs-Update | P1 | Wrong --target default ("claude" vs ""), missing required indicator for --output | medium |
| 13 | reference/commands/utility/help.md | 1C | Current | — | No changes needed | — |
| 14 | reference/commands/utility/test.md | 1C | Current | — | No changes needed | — |
| 15 | reference/commands/utility/registry.md | 1C | Current | — | No changes needed | — |

### Fix Group B: Reference Kinds (18 files)

| # | File | Domain | Verdict | Priority | Key Issues | Effort |
|---|------|--------|---------|----------|------------|--------|
| 16 | reference/kinds/provider/index.md | 1D | Needs-Update | P2 | Workflow listed as "all 5 providers" — Antigravity only | small |
| 17 | reference/kinds/provider/agent.md | 1D | Needs-Update | P3 | Phantom `mode` field to remove | small |
| 18 | reference/kinds/provider/skill.md | 1D | Current | — | No changes needed | — |
| 19 | reference/kinds/provider/rule.md | 1D | Current | — | No changes needed | — |
| 20 | reference/kinds/provider/mcp.md | 1D | Current | — | No changes needed | — |
| 21 | reference/kinds/provider/hooks.md | 1D | Needs-Update | P2 | Missing `artifacts` field | small |
| 22 | reference/kinds/provider/settings.md | 1D | Current | — | Selective coverage intentional | — |
| 23 | reference/kinds/provider/memory.md | 1D | Current | — | No changes needed | — |
| 24 | reference/kinds/provider/context.md | 1D | Current | — | No changes needed | — |
| 25 | reference/kinds/provider/workflow.md | 1D | Current | — | No changes needed | — |
| 26 | reference/kinds/xcaffold/index.md | 1E | Needs-Update | P2 | project misclassified as pure-YAML | small |
| 27 | reference/kinds/xcaffold/blueprint.md | 1E | Needs-Update | P1 | Frontmatter delimiters on pure-YAML kind, .xcf remnant | medium |
| 28 | reference/kinds/xcaffold/global.md | 1E | Needs-Rewrite | P1 | Invented ResourceRef schema, wrong resource format throughout | large |
| 29 | reference/kinds/xcaffold/policy.md | 1E | Needs-Update | P1 | Frontmatter delimiters on pure-YAML kind, .xcf remnant | medium |
| 30 | reference/kinds/xcaffold/project.md | 1E | Needs-Update | P1 | Missing test, local, target-options fields; agents format incomplete | medium |
| 31 | reference/index.md | 1E | Needs-Update | P2 | Broken how-to/index.md link | small |
| 32 | reference/kinds/index.md | 1E | Needs-Update | P1 | Broken links to settings/hooks pages; project misclassified | medium |
| 33 | reference/supported-providers.md | 1E | Needs-Update | P1 | Capability matrix discrepancies with manifests; broken schema.md link | medium |

### Fix Group C: Concepts (16 files)

| # | File | Domain | Verdict | Priority | Key Issues | Effort |
|---|------|--------|---------|----------|------------|--------|
| 34 | concepts/architecture/index.md | 1F | Current | — | No changes needed | — |
| 35 | concepts/architecture/architecture.md | 1F | Needs-Rewrite | P1 | Claims non-existent internal/bir/ package | large |
| 36 | concepts/architecture/intermediate-representation.md | 1F | Current | — | No changes needed | — |
| 37 | concepts/architecture/multi-target-rendering.md | 1F | Current | P3 | Minor context gap on CapabilitySet mechanism | small |
| 38 | concepts/architecture/provider-architecture.md | 1F | Needs-Rewrite | P1 | CapabilitySet struct outdated (3 fields wrong), capability matrix has phantom ModelField | large |
| 39 | concepts/architecture/translation-pipeline.md | 1F | Needs-Rewrite | P1 | References 10+ non-existent BIR functions | large |
| 40 | concepts/configuration/index.md | 1G | Current | — | No changes needed | — |
| 41 | concepts/configuration/configuration-scopes.md | 1G | Current | — | No changes needed | — |
| 42 | concepts/configuration/declarative-compilation.md | 1G | Current | — | No changes needed | — |
| 43 | concepts/configuration/field-model.md | 1G | Current | — | No changes needed | — |
| 44 | concepts/configuration/layer-precedence.md | 1G | Current | — | No changes needed | — |
| 45 | concepts/configuration/variables.md | 1G | Current | — | No changes needed | — |
| 46 | concepts/execution/index.md | 1H | Current | — | No changes needed | — |
| 47 | concepts/execution/agent-memory.md | 1H | Current | — | No changes needed | — |
| 48 | concepts/execution/sandboxing.md | 1H | Needs-Update | P2 | 4 line number citations off by ~593 lines | medium |
| 49 | concepts/execution/state-and-drift.md | 1H | Current | — | No changes needed | — |

### Fix Group D: Tutorials + Best Practices (13 files)

| # | File | Domain | Verdict | Priority | Key Issues | Effort |
|---|------|--------|---------|----------|------------|--------|
| 50 | tutorials/index.md | 1I | Current | — | No changes needed | — |
| 51 | tutorials/basics/index.md | 1I | Current | — | No changes needed | — |
| 52 | tutorials/basics/getting-started.md | 1I | Needs-Update | P1 | 4 broken cross-references | medium |
| 53 | tutorials/basics/ai-assisted-scaffolding.md | 1I | Needs-Update | P1 | 3 broken cross-references | medium |
| 54 | tutorials/advanced/index.md | 1I | Current | — | No changes needed | — |
| 55 | tutorials/advanced/drift-remediation.md | 1I | Needs-Update | P1 | 3 broken cross-references, output format discrepancy | medium |
| 56 | tutorials/advanced/multi-agent-workspace.md | 1I | Needs-Update | P1 | 2 broken links, potential phantom --check-permissions flag | medium |
| 57 | best-practices/index.md | 1J | Needs-Update | P1 | Broken link to non-existent cross-provider.md | small |
| 58 | best-practices/blueprint-design.md | 1J | Needs-Update | P2 | Missing memory field example | small |
| 59 | best-practices/policy-organization.md | 1J | Current | — | No changes needed | — |
| 60 | best-practices/project-layouts.md | 1J | Current | — | No changes needed | — |
| 61 | best-practices/skill-organization.md | 1J | Needs-Update | P2 | Missing artifacts field declaration in examples | medium |
| 62 | best-practices/workspace-context.md | 1J | Current | — | No changes needed | — |

### Fix Group E: Root Files (5 files)

| # | File | Domain | Verdict | Priority | Key Issues | Effort |
|---|------|--------|---------|----------|------------|--------|
| 63 | README.md | 1A | Needs-Update | P1 | `xcaffold diff` referenced (removed command), Go version badge stale (1.24 vs 1.25), broken how-to link, quick-start incomplete | medium |
| 64 | CHANGELOG.md | 1A | Needs-Update | P2 | Duplicate ### Added headers, version mismatch (main.go vs manifest vs CHANGELOG), misleading 1.0.0-dev label | medium |
| 65 | CONTRIBUTING.md | 1A | Current | — | No changes needed (how-to ref is aspirational, acceptable) | — |
| 66 | CODE_OF_CONDUCT.md | 1A | Current | — | No changes needed | — |
| 67 | SECURITY.md | 1A | Current | — | No changes needed | — |

---

## Decision Points for User Review

### 1. Missing pages: create or remove links?
Four pages are linked from indexes but don't exist:
- `docs/reference/kinds/xcaffold/settings.md` — Create? (SettingsConfig has 30+ fields)
- `docs/reference/kinds/xcaffold/hooks.md` — Create? (NamedHookConfig is documented in provider/hooks.md)
- `docs/reference/schema.md` — Create? (field fidelity mapping reference)
- `docs/how-to/index.md` — Create the how-to section?

### 2. Non-existent how-to directory
6 tutorial cross-references point to `docs/how-to/` which doesn't exist. Options:
a) Redirect all how-to links to existing best-practices or reference pages
b) Create how-to stubs
c) Remove the links entirely

### 3. BIR package documentation
`translation-pipeline.md` references a non-existent `internal/bir/` package extensively. Options:
a) Rewrite to document the actual `internal/translator/` package
b) Remove the file if the concept is adequately covered elsewhere

### 4. cross-provider.md best practice
`best-practices/index.md` links to a non-existent `cross-provider.md`. Options:
a) Create it (covers targets, fidelity notes, target-options)
b) Remove the link

---

## Files Requiring NO Changes (38+)

These files passed audit with no issues or only P3 cosmetic notes:

commands/index.md, diagnostic/index.md, diagnostic/list.md, diagnostic/status.md,
lifecycle/apply.md, lifecycle/import.md, utility/index.md, utility/help.md,
utility/test.md, utility/registry.md, provider/skill.md, provider/rule.md,
provider/mcp.md, provider/settings.md, provider/memory.md, provider/context.md,
provider/workflow.md, concepts/architecture/index.md, concepts/architecture/intermediate-representation.md,
concepts/configuration/* (all 6), concepts/execution/index.md,
concepts/execution/agent-memory.md, concepts/execution/state-and-drift.md,
tutorials/index.md, tutorials/basics/index.md, tutorials/advanced/index.md,
best-practices/policy-organization.md, best-practices/project-layouts.md,
best-practices/workspace-context.md, CONTRIBUTING.md, CODE_OF_CONDUCT.md, SECURITY.md
