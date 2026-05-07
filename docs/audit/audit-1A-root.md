# Audit Report: Root Files

**Date:** 2026-05-07
**Agent:** docs-specialist
**Scope:** README.md, CHANGELOG.md, CONTRIBUTING.md, CODE_OF_CONDUCT.md, SECURITY.md

---

## Summary

| Metric | Count |
|--------|-------|
| Files audited | 5 |
| Current (no changes needed) | 2 |
| Needs update | 3 |
| Needs rewrite | 0 |
| Redundant | 0 |
| Missing (should exist but doesn't) | 0 |

---

## Per-File Assessment

### README.md

| Dimension | Rating | Details |
|-----------|--------|---------|
| Accuracy | stale | Three inaccuracies found (see Specific Issues below) |
| Completeness | partial | Quick-start omits `graph`, `list`, `status`; How-to link target missing |
| Terminology | clean | All `.xcaf` and `xcaf/` references correct; no stale `.xcf` usage |
| Structure | matches-template | Overview, install, quick-start, feature table, docs links, contributing |
| Cross-References | broken-links | `docs/how-to/index.md` does not exist on disk |

**Verdict:** Needs-Update
**Priority:** P1 (user-facing accuracy)

**Specific Issues:**

1. **`xcaffold diff` referenced twice — command no longer exists.**
   - Line 33: "…`xcaffold diff` shows exactly what changed and why."
   - Line 102 (Quick-start code block): `xcaffold diff # Detect manual drift in output directories`
   - Source verification: no `diff.go` exists under `cmd/xcaffold/`. CHANGELOG `[Unreleased]` confirms `xcaffold diff` was deprecated and officially replaced by `xcaffold status`. The `--target` flag on `xcaffold status` covers the targeted drift check use case.
   - **Fix:** Replace both mentions with `xcaffold status`.

2. **Go version badge is stale.**
   - Line 11: `[![Go Version](https://img.shields.io/badge/go-1.24-blue.svg)](https://golang.org/dl/)`
   - `go.mod` declares `go 1.25.0`.
   - **Fix:** Update badge URL to `go-1.25-blue.svg` and link to `https://go.dev/dl/`.

3. **`docs/how-to/index.md` link target does not exist.**
   - Line 127: `[How-To Guides](docs/how-to/index.md)`
   - The `docs/how-to/` directory does not exist anywhere in the worktree. All other Documentation section links (`docs/tutorials/index.md`, `docs/concepts/index.md`, `docs/reference/index.md`) resolve correctly.
   - **Fix:** Either remove the How-to line until the directory is created, or replace it with a link to an existing path (e.g., `docs/best-practices/index.md`).

4. **Quick-start example omits key commands: `graph`, `list`, `status`.**
   - The code block at lines 99–105 shows only `init`, `apply`, `diff`, `validate`, `import`. The `status`, `list`, and `graph` commands are among the 7 public commands registered in `cmd/xcaffold/` but are absent from the "Run the lifecycle" example. This is a completeness issue, not accuracy — aside from `diff`.
   - **Priority:** P2 (completeness).

---

### CHANGELOG.md

| Dimension | Rating | Details |
|-----------|--------|---------|
| Accuracy | stale | Version numbering is inconsistent across sources |
| Completeness | partial | No versioned release entry other than `[1.0.0-dev]` |
| Terminology | clean | No stale `.xcf` references; all format names correct |
| Structure | needs-restructure | Multiple distinct sub-sections share a single `[Unreleased]` block with duplicate `### Added` headers |
| Cross-References | valid | No internal doc links present |

**Verdict:** Needs-Update
**Priority:** P2 (completeness and structure)

**Specific Issues:**

1. **Duplicate `### Added` headers inside `[Unreleased]`.**
   - The `[Unreleased]` section contains the heading `### Added` four times and `### Changed` twice. Keep-a-Changelog format requires each heading to appear at most once per version block. Distinct feature areas should use parenthetical sub-labels (e.g., `### Added (Provider-Agnostic Renderer)`) or be merged into a single grouped list — not both a plain `### Added` and a labeled `### Added (Provider-Agnostic Renderer)`.
   - **Fix:** Merge or consolidate duplicate sub-section headings within `[Unreleased]`.

2. **Version mismatch across sources.**
   - `cmd/xcaffold/main.go` line 16: `version = "0.2.0-dev"`
   - `.release-please-manifest.json`: `"." : "0.1.0"`
   - `CHANGELOG.md`: Only `[1.0.0-dev] - 2026-04-02` as a labeled release entry
   - These three version identifiers do not align. A reader cannot determine the current release version from any single source.
   - **Fix:** Align the manifest, the `main.go` version string, and the CHANGELOG entries. If `0.2.0-dev` is the active development version, CHANGELOG should show `[Unreleased]` at the top (as it does) and the last released version should appear below with a proper tag (e.g., `[0.1.0] - YYYY-MM-DD`).

3. **`[1.0.0-dev]` label is misleading.**
   - The entry labeled `[1.0.0-dev] - 2026-04-02` describes an early prototype phase, but the current development version is `0.2.0-dev`. Using `1.0.0-dev` for a historical checkpoint while `0.2.0-dev` is the current dev version is contradictory versioning.
   - **Fix:** Relabel as `[0.1.0] - 2026-04-02` or `[0.1.0-dev] - 2026-04-02`, consistent with the release-please manifest entry `"0.1.0"`.

---

### CONTRIBUTING.md

| Dimension | Rating | Details |
|-----------|--------|---------|
| Accuracy | current | All build commands match Makefile targets exactly; `.xcaf` format references correct |
| Completeness | complete | Covers bug reporting, feature proposals, PR process, testing, docs pillars, provider development, architectural constraints, and breaking changes |
| Terminology | clean | No stale `.xcf` references; all field names use kebab-case correctly |
| Structure | matches-template | Logically ordered; "Adding a New Provider" section is complete and well-structured |
| Cross-References | valid | No broken internal links; external links (Conventional Commits, Diátaxis) are stable |

**Verdict:** Current
**Priority:** —

**Notes:**
- `make setup`, `make lint`, `make test`, `make test-e2e`, `make generate`, `make verify-generate`, `make verify-markers` all exist in the Makefile and match exactly what CONTRIBUTING documents.
- The Diátaxis pillar table lists `docs/how-to/` as a valid directory even though it does not exist on disk. This is consistent with CONTRIBUTING's role as an aspirational contribution guide describing intended structure.

---

### CODE_OF_CONDUCT.md

| Dimension | Rating | Details |
|-----------|--------|---------|
| Accuracy | current | Contributor Covenant 3.0; enforcement ladder is complete |
| Completeness | complete | Pledge, encouraged behaviors, restricted behaviors, reporting, enforcement, scope, attribution |
| Terminology | clean | No xcaffold-specific terminology to verify |
| Structure | matches-template | Standard Contributor Covenant 3.0 structure |
| Cross-References | valid | Attribution links to contributor-covenant.org are stable external references |

**Verdict:** Current
**Priority:** —

**Notes:**
- Contact email `xcaffold@saero.ai` (line 49) is specific and attributable.
- Version 3.0 is the current Contributor Covenant version as of audit date.

---

### SECURITY.md

| Dimension | Rating | Details |
|-----------|--------|---------|
| Accuracy | current | Scope section accurately reflects xcaffold's compilation pipeline surface |
| Completeness | complete | Disclosure process, response timeline, supported versions, in-scope/out-of-scope |
| Terminology | clean | References `xcaffold apply`, `xcaffold import`, `xcaffold validate` — all are valid registered commands |
| Structure | matches-template | Standard security policy structure |
| Cross-References | valid | GitHub advisory database link is a stable external reference |

**Verdict:** Current
**Priority:** —

**Notes:**
- Contact `security@saero.ai` is distinct from `xcaffold@saero.ai` in CODE_OF_CONDUCT — appropriate separation of security and community contacts.
- "Latest release only" support policy is correct and standard for a CLI at this stage.

---

## Cross-Cutting Observations

1. **`xcaffold diff` is a pervasive stale reference.** It appears in README.md (2 occurrences, P1) and across the tutorials and concepts docs tree (multiple occurrences confirmed in other audit scopes). The root files audit surfaces the highest-priority instances — they are the first user-facing entry points.

2. **`docs/how-to/` is a systemic missing directory.** README.md, CONTRIBUTING.md, docs/tutorials/index.md, docs/concepts/index.md, and docs/reference/index.md all reference this directory or files within it. The absence breaks Diátaxis pillar navigation from every entry point. Either create the directory with at least an `index.md`, or audit and redirect all cross-references to existing equivalents.

3. **Version misalignment is a P1 release hygiene issue.** Three authoritative version sources (`main.go`, `.release-please-manifest.json`, CHANGELOG) disagree. This will cause incorrect version reporting in binaries built from this branch and will break automated tooling that reads version from the manifest.

4. **`xcaffold translate` references in non-root docs are out of scope here** but confirmed present in `docs/reference/supported-providers.md` and `docs/concepts/architecture/` files. Both reference the removed command. Those findings are covered by the reference and concepts audit scopes.

---

## Missing Documentation

- **`docs/how-to/` directory** — Missing. Referenced from README.md (line 127), CONTRIBUTING.md (line 82), and three docs index pages. Absence breaks navigation across all four Diátaxis pillars.
- **A properly versioned CHANGELOG release entry** — No `[0.1.0]` entry exists. The release-please manifest declares `0.1.0` as the released version, but CHANGELOG only has `[Unreleased]` and `[1.0.0-dev]`.
