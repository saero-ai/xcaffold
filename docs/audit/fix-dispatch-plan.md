# Fix Dispatch Plan — Documentation Refresh

**Date:** 2026-05-07
**Scope:** All high and medium priority findings from master audit matrix
**Branch:** docs/documentation-audit

## Execution Sequence

```
Fix Group A (Ref Commands) ──┐
Fix Group B (Ref Kinds)    ──┤ parallel
                             ▼
Fix Group C (Concepts)
                             ▼
Fix Group D (Tutorials + Best Practices)
                             ▼
Fix Group E (Root Files)
```

---

## Fix Group A: Reference Commands (2 agents)

### Agent A1: Rewrite graph.md + fix lifecycle/index.md

**Files to modify (2):**
1. `docs/reference/commands/diagnostic/graph.md`
2. `docs/reference/commands/lifecycle/index.md`

**Source to read:**
- `cmd/xcaffold/graph.go` (all flag definitions in init())

**Fixes:**
1. **graph.md — REWRITE flags section:**
   - DELETE phantom flags: --skill, --rule, --workflow, --mcp, --context, --hooks, --settings
   - ADD real missing flags with descriptions from source:
     - `--project <name>` / `-p` — "Target a specific managed project by registered name or path"
     - `--full` / `-f` — "Show the fully expanded topology tree (always true if targeting an agent)"
     - `--scan-output` — "Scan compiled output directories for undeclared artifacts"
     - `--all` — "Show global topology and all registered projects"
   - KEEP existing accurate flags: --agent/-a, --format, --blueprint (hidden), --global/-g, --no-color
   - REWRITE "Kind-filter mode" section — remove it entirely. There is no kind filtering.
   - UPDATE all examples to use real flags only.

2. **lifecycle/index.md — Fix broken links:**
   - Change `](/docs/cli/reference/commands/lifecycle/apply)` to `](./apply)`
   - Change `](/docs/cli/reference/commands/lifecycle/import)` to `](./import)`

---

### Agent A2: Fix init.md + validate.md + export.md

**Files to modify (3):**
1. `docs/reference/commands/lifecycle/init.md`
2. `docs/reference/commands/lifecycle/validate.md`
3. `docs/reference/commands/utility/export.md`

**Source to read:**
- `cmd/xcaffold/init.go` (flag definitions + template logic)
- `cmd/xcaffold/validate.go` (flag definitions)
- `cmd/xcaffold/export.go` (flag definitions + required markers)

**Fixes:**
1. **init.md:** ADD --template and --no-policies flags to table. ADD example.
2. **validate.md:** ADD --var-file and --blueprint flags. ADD --global to inherited flags.
3. **export.md:** CHANGE --target default from "claude" to "". Mark --output as required.

---

## Fix Group B: Reference Kinds (3 agents)

### Agent B1: Fix provider kinds (3 small fixes)

**Files to modify (3):**
1. `docs/reference/kinds/provider/index.md`
2. `docs/reference/kinds/provider/agent.md`
3. `docs/reference/kinds/provider/hooks.md`

**Source to read:**
- `internal/ast/types.go` (AgentConfig, NamedHookConfig structs)

**Fixes:**
1. **provider/index.md:** Change workflow provider support from "All 5 providers" to "Antigravity"
2. **provider/agent.md:** Remove phantom `mode` field from Argument Reference
3. **provider/hooks.md:** Add `artifacts` field documentation

---

### Agent B2: Rewrite global.md + fix blueprint.md + policy.md

**Files to modify (3):**
1. `docs/reference/kinds/xcaffold/global.md` — REWRITE
2. `docs/reference/kinds/xcaffold/blueprint.md`
3. `docs/reference/kinds/xcaffold/policy.md`

**Source to read:**
- `internal/ast/types.go` (GlobalConfig, globalDocument struct)
- `schema/golden/global.xcaf`, `schema/golden/blueprint.xcaf`, `schema/golden/policy.xcaf`

**Fixes:**
1. **global.md — REWRITE:** Remove invented ResourceRef schema. Replace with inline resource maps per golden schema. Remove name from Required fields. Convert to table format.
2. **blueprint.md:** Remove `---` frontmatter delimiters (pure YAML). Change `.xcf` to `.xcaf`.
3. **policy.md:** Remove `---` frontmatter delimiters (pure YAML). Change `.xcf` to `.xcaf`.

---

### Agent B3: Fix project.md + indexes + supported-providers.md

**Files to modify (5):**
1. `docs/reference/kinds/xcaffold/project.md`
2. `docs/reference/kinds/index.md`
3. `docs/reference/supported-providers.md`
4. `docs/reference/index.md`
5. `docs/reference/kinds/xcaffold/index.md`

**Source to read:**
- `internal/ast/types.go` (ProjectConfig struct)
- `schema/golden/project.xcaf`
- Provider manifests: `providers/*/manifest.go` (KindSupport maps)

**Fixes:**
1. **project.md:** ADD test, local, target-options fields. FIX agents entry format.
2. **kinds/index.md:** REMOVE broken settings/hooks links. FIX project classification.
3. **supported-providers.md:** Reconcile capability matrix against manifests. REMOVE broken schema.md link.
4. **reference/index.md:** Replace how-to link with best-practices link.
5. **xcaffold/index.md:** Clarify project format description.

---

## Fix Group C: Concepts (3 agents)

### Agent C1: Rewrite architecture.md

**File:** `docs/concepts/architecture/architecture.md`
**Source:** `ls internal/` to verify package list
**Fix:** Remove `bir` package row. Update package count. Verify all other rows.

---

### Agent C2: Rewrite provider-architecture.md

**File:** `docs/concepts/architecture/provider-architecture.md`
**Source:** `internal/renderer/capabilities.go` (CapabilitySet and RuleEncodingCapabilities structs)
**Fix:** Update CapabilitySet: SkillSubdirs to SkillArtifactDirs, remove ModelField, add RuleEncoding and AgentNativeToolsOnly. Update capability matrix.

---

### Agent C3: Rewrite translation-pipeline.md + fix sandboxing.md

**Files:** `docs/concepts/architecture/translation-pipeline.md`, `docs/concepts/execution/sandboxing.md`
**Source:** `internal/translator/workflow.go`, `internal/translator/rules.go`, `internal/ast/types.go` lines 776-807

**Fixes:**
1. **translation-pipeline.md:** Remove all BIR references. Document actual translator functions.
2. **sandboxing.md:** Update 4 line number citations (+593). Verify securityFieldReport().

---

## Fix Group D: Tutorials + Best Practices (2 agents)

### Agent D1: Fix tutorial cross-references

**Files (4):** getting-started.md, ai-assisted-scaffolding.md, drift-remediation.md, multi-agent-workspace.md

**Fixes:** Fix all 12 broken cross-references:
- `how-to/multi-file-projects.md` to `best-practices/project-layouts.md`
- `how-to/import-existing-config.md` to `reference/commands/lifecycle/import.md`
- `reference/cli.md` to `reference/commands/index.md`
- Fix relative paths within tutorials (../advanced/, ../basics/)
- REMOVE links to how-to/target-overrides.md and how-to/policy-enforcement.md (no equivalents)

---

### Agent D2: Fix best practices

**Files (3):** index.md, skill-organization.md, blueprint-design.md

**Fixes:**
1. **index.md:** REMOVE broken cross-provider.md link
2. **skill-organization.md:** ADD artifacts field to examples, add validation error note
3. **blueprint-design.md:** ADD memory field to blueprint example

---

## Fix Group E: Root Files (1 agent)

### Agent E1: Fix README.md + CHANGELOG.md

**Files (2):** README.md, CHANGELOG.md

**Fixes:**
1. **README.md:** Replace `xcaffold diff` with `xcaffold status` (2 places). Update Go badge 1.24 to 1.25. Fix how-to link. Expand quick-start.
2. **CHANGELOG.md:** Merge duplicate Added headers. Relabel 1.0.0-dev to 0.1.0.

---

## Summary

| Group | Agents | Files | Depends On |
|-------|--------|-------|------------|
| A | 2 | 5 | None |
| B | 3 | 11 | None (parallel with A) |
| C | 3 | 4 | A, B |
| D | 2 | 7 | C |
| E | 1 | 2 | D |
| **Total** | **11** | **29** | |
