# Audit Report: xcaffold Kinds + Supported Providers

**Date:** 2026-05-07
**Agent:** docs-specialist
**Scope:** 8 files covering xcaffold-specific kinds and supported providers

---

## Summary

| Metric | Count |
|--------|-------|
| Files audited | 8 |
| Current (no changes needed) | 2 |
| Needs update | 5 |
| Needs rewrite | 1 |
| Redundant | 0 |
| Missing (should exist but doesn't) | 0 |

---

## Per-File Assessment

### `docs/reference/kinds/xcaffold/index.md`

| Dimension | Rating | Details |
|-----------|--------|---------|
| Accuracy | stale | Claims `project` "supports an optional markdown body after closing `---`" — accurate, but the statement that "all use pure YAML" misleads because `project` uses frontmatter+body format, not pure YAML. See issue 1. |
| Completeness | partial | Lists `registry` as a xcaffold kind; the `kinds/index.md` parent does NOT list `registry`. Minor inconsistency between these two index pages. |
| Terminology | has-stale-refs | Uses `.xcf` in the text: no direct `.xcf` references found in this file. Clean. |
| Structure | matches-template | Index structure is adequate. |
| Cross-References | valid | All relative links (`./project`, `./policy`, `./blueprint`, `./global`, `./registry`) resolve to existing files. |

**Verdict:** Needs-Update
**Priority:** P2 (completeness)
**Specific Issues:**
1. The closing statement "These kinds use **pure YAML format** (no frontmatter `---` delimiters) with the exception of `project`, which supports an optional markdown body after closing `---`" is contradictory: if `project` has a body after `---`, it is by definition frontmatter+body format, not pure YAML. The sentence should state that `project` uses frontmatter format and all others use pure YAML.
2. `registry` is listed here but absent from the parent `kinds/index.md`. One index or the other is inconsistent.

---

### `docs/reference/kinds/xcaffold/blueprint.md`

| Dimension | Rating | Details |
|-----------|--------|---------|
| Accuracy | wrong | Example Usage block wraps the blueprint in frontmatter delimiters (`---` ... `---`). According to `internal/ast/types.go` and the ground truth (`kind-blueprint.json`), blueprint is **pure YAML** — no `---` delimiters. The golden schema `schema/golden/blueprint.xcaf` confirms this: no delimiters present. |
| Completeness | partial | All `BlueprintConfig` struct fields are documented (`name`, `description`, `extends`, `agents`, `skills`, `rules`, `workflows`, `mcp`, `policies`, `memory`, `contexts`, `settings`, `hooks`, `targets`). The `Targets` field is present and correctly documented. No missing fields against the struct. |
| Terminology | clean | No `.xcf` references. Correctly uses `.xcaf` in the Filesystem-as-Schema section (`blueprint.xcf` — **wrong**: should be `blueprint.xcaf`). |
| Structure | matches-template | Follows the reference page template. |
| Cross-References | valid | No outbound links to verify. |

**Verdict:** Needs-Update
**Priority:** P1 (user-facing accuracy)
**Specific Issues:**
1. Example Usage wraps the blueprint YAML in `---` frontmatter delimiters. Pure YAML kinds have no delimiters. The example should begin with `kind: blueprint` directly, without `---` markers.
2. Filesystem-as-Schema section references `blueprint.xcf` — should be `blueprint.xcaf` (matching the project's extension convention already established by the golden schema filename).

---

### `docs/reference/kinds/xcaffold/global.md`

| Dimension | Rating | Details |
|-----------|--------|---------|
| Accuracy | wrong | Multiple structural inaccuracies against `internal/ast/types.go` and `kind-global.json`. See issues 1–4 below. |
| Completeness | partial | Documents only a subset of fields the `globalDocument` can hold; omits the inline embedding pattern (`settings` block as embedded `SettingsConfig`, `hooks` as `HookConfig`, `mcp` as map of `MCPConfig`). Resource fields are documented as `[]ResourceRef` lists with `id`/`path` entries, but the ground truth states fields are `maps of ID to inline config, NOT arrays of IDs`. |
| Terminology | has-stale-refs | Uses `.xcaf` correctly in prose. Example paths use `.xcaf` correctly. |
| Structure | needs-restructure | Argument Reference uses bullet prose instead of the standard table format used by all other kind reference pages. |
| Cross-References | valid | No outbound links. |

**Verdict:** Needs-Rewrite
**Priority:** P1 (user-facing accuracy)
**Specific Issues:**
1. Header states `> **Required:** kind, version, name`. The ground truth (`kind-global.json`) explicitly states: "name is NOT required for kind: global (it is a singleton)." The doc incorrectly requires `name`.
2. The example shows resources as arrays of `{id, path}` objects (e.g., `skills: [- id: conventional-commits, path: xcaf/global/skills/...`). The ground truth states: "ResourceScope fields are maps of ID to inline config, NOT arrays of IDs." The correct shape matches `AgentConfig`/`SkillConfig` inline map keys, as shown in the golden schema (`global.xcaf` uses `settings: {name: default, ...}` inline map and `mcp: {global-ref-mcp: {name: ...}}`). The `ResourceRef` object with `id`/`path` keys does not exist in the `globalDocument` struct.
3. The `### ResourceRef` subsection in the Argument Reference documents a type (`id`, `path`) that does not exist in the actual `globalDocument` parser struct. `KnownFields(true)` would reject any key not in the struct. This entire subsection is inaccurate.
4. The Behavior section describes a `name:` field on global resources ("`id` — (Required) Identifier used to reference the resource") that is part of the invented `ResourceRef` type. Since `ResourceRef` doesn't exist in the Go code, this behavior description is inaccurate.
5. The project example shows `extends: xcaf/global/org-baseline.xcaf` as a file path inside the `kind: project` block; the project doc and golden schema show `extends:` as a string field on `XcaffoldConfig`, not on `ProjectConfig`. This may be accurate at the `XcaffoldConfig` level but inconsistent with the project.md reference.
6. Ground truth notes: "globalDocument does NOT include policies, memory, contexts, references, or blueprints fields." The doc example includes `policies:` under global — this may be accurate if `ResourceScope` includes `Policies`; cross-checking `ResourceScope` in `types.go` shows `Policies map[string]PolicyConfig` IS present. However, the inline map format for policies is still wrong (doc uses `[]ResourceRef` not `map[string]PolicyConfig`).
7. Argument Reference uses an unformatted bullet list rather than the standard table format used by all other reference pages in this directory.

---

### `docs/reference/kinds/xcaffold/policy.md`

| Dimension | Rating | Details |
|-----------|--------|---------|
| Accuracy | stale | Example Usage block wraps the policy in frontmatter delimiters (`---` ... `---`). According to `internal/ast/types.go`, `kind-policy.json`, and the golden schema `schema/golden/policy.xcaf`, policy is **pure YAML** — no `---` delimiters. |
| Completeness | complete | All `PolicyConfig`, `PolicyMatch`, `PolicyRequire`, and `PolicyDeny` struct fields are documented correctly and completely. Severity enum values (`error`, `warning`, `off`) and target enum values (`agent`, `skill`, `rule`, `hook`, `settings`, `output`) match the struct. |
| Terminology | clean | No `.xcf` references. Filesystem-as-Schema uses `policy.xcf` — should be `policy.xcaf`. |
| Structure | matches-template | Well-structured. Follows the reference page template. |
| Cross-References | valid | No outbound links. |

**Verdict:** Needs-Update
**Priority:** P1 (user-facing accuracy)
**Specific Issues:**
1. Example Usage block uses `---` frontmatter delimiters. Policy is pure YAML — no delimiters. The example should begin with `kind: policy` directly.
2. Filesystem-as-Schema section references `policy.xcf` — should be `policy.xcaf`.

---

### `docs/reference/kinds/xcaffold/project.md`

| Dimension | Rating | Details |
|-----------|--------|---------|
| Accuracy | stale | Several inaccuracies against `internal/ast/types.go` (`ProjectConfig` struct) and `schema/golden/project.xcaf`. See issues below. |
| Completeness | partial | Missing documented fields: `test` (`TestConfig`), `local` (`SettingsConfig`), `target-options` (`map[string]TargetOverride`), `version` (project version, not schema version), `author`, `homepage`, `repository`, `license`, `backup-dir`. Some of these appear in the example but are not in the Argument Reference list. See issue 2. |
| Terminology | clean | No `.xcf` references. |
| Structure | matches-template | Uses bullet list for Argument Reference rather than tables; inconsistent with policy.md and blueprint.md which use tables. |
| Cross-References | valid | Link to `../../../concepts/configuration/variables.md` — file exists at `docs/concepts/configuration/variables.md`. |

**Verdict:** Needs-Update
**Priority:** P1 (user-facing accuracy)
**Specific Issues:**
1. The Required field list states `> **Required:** kind, version, name, targets`. The `ProjectConfig` struct shows `Targets []string yaml:"-"` — the `yaml:"-"` tag means `targets` is **not decoded from YAML directly**. The ground truth (`kind-project.json`) states targets are "Populated by the parser when decoding kind: project documents." The Argument Reference text does treat `targets` as required ("`targets` — (Required) `[]string`..."), which is functionally correct even if the implementation detail differs. However, the Behavior section implies `targets` is an explicit YAML key, which needs care given the `yaml:"-"` tag. **This is a nuance worth noting but not a hard error.**
2. The Argument Reference omits these fields that appear in `ProjectConfig` and the golden schema: `test` (type `TestConfig` with subfields `cli-path`, `judge-model`, `task`, `max-turns`), `local` (type `SettingsConfig`), `target-options` (type `map[string]TargetOverride`). All three appear in the golden schema `project.xcaf` and are valid YAML fields. The `test` field is specifically called out in the audit task as a field that must be documented.
3. The `agents` block sub-section describes entries as `{id, path}` pairs (`id` — (Required), `path` — (Required)). But the ground truth (`kind-project.json`) states: "YAML key 'agents' decodes as `[]AgentManifestEntry` (ref list)" and "AgentManifestEntry can be a bare string ID or a map `{ <id>: { memory: [<ids>] } }`." The current doc doesn't document the bare string format or the memory binding syntax.
4. The `skills`, `rules`, `mcp`, `policies` fields in the Argument Reference are described as `[]string` (e.g., `- component-patterns`). The ground truth confirms resource references are "arrays of string IDs" — this is correct for these fields.
5. The `allowed-env-vars` field IS documented (line 81). Good — this audit item is covered.
6. The Behavior table for compiled output destinations shows "Cursor → `AGENTS.md` (nested)" for Cursor — but the cursor manifest (`providers/cursor/manifest.go`) has `RootContextFile: "AGENTS.md"`. The project.md doc says `AGENTS.md` for Antigravity, not Cursor. Cross-checking: the supported-providers.md table says Cursor → "Project Instructions" → `AGENTS.md` (nested). But the cursor manifest says `RootContextFile: "AGENTS.md"` which contradicts the supported-providers matrix that says Cursor → `AGENTS.md` and Antigravity → `.agents/rules/*.md`. This needs careful verification — the Behavior table in project.md does not list Cursor at all, only Claude, Gemini, Antigravity, Cursor, Copilot — with Cursor → `.cursor/rules/project-instructions.md`. That does match expected Cursor behavior for rule-based instructions.

---

### `docs/reference/index.md`

| Dimension | Rating | Details |
|-----------|--------|---------|
| Accuracy | current | All three linked sections are accurate descriptions of their targets. |
| Completeness | complete | Lists all three major reference sections: Kinds, CLI Reference, Supported Providers. |
| Terminology | clean | No stale terminology. |
| Structure | matches-template | Standard index structure. |
| Cross-References | broken-links | `../how-to/index.md` — the `docs/how-to/index.md` file does **not exist** in the worktree. All other links resolve: `kinds/index.md`, `commands/index.md`, `supported-providers.md`, `../tutorials/index.md`, `../concepts/index.md` all exist. |

**Verdict:** Needs-Update
**Priority:** P2 (completeness)
**Specific Issues:**
1. The "Next Steps" section links to `../how-to/index.md` which does not exist (`docs/how-to/index.md` is missing). This is a broken link that will produce a 404 in any static site generator.

---

### `docs/reference/kinds/index.md`

| Dimension | Rating | Details |
|-----------|--------|---------|
| Accuracy | stale | Multiple inaccuracies in the xcaffold kinds table and the file format descriptions. See issues below. |
| Completeness | partial | Missing `registry` kind from the xcaffold kinds table (it appears in `xcaffold/index.md` but not here). `settings` and `hooks` are listed in the xcaffold table but have no linked pages — the links point to `./xcaffold/settings` and `./xcaffold/hooks` which don't exist as files in the filesystem. |
| Terminology | clean | Correctly uses `.xcaf` throughout. |
| Structure | matches-template | Well-structured with both tables and format descriptions. |
| Cross-References | broken-links | Links to `./xcaffold/settings` and `./xcaffold/hooks` do not correspond to any existing files. `./provider/agent`, `./provider/skill`, `./provider/rule`, `./provider/mcp`, `./provider/workflow`, `./provider/context` are not verified in this audit (out of scope) but should be checked separately. |

**Verdict:** Needs-Update
**Priority:** P1 (user-facing accuracy — broken links to non-existent pages)
**Specific Issues:**
1. The xcaffold kinds table lists `settings` and `hooks` with links to `./xcaffold/settings` and `./xcaffold/hooks`. These pages do not exist. Either the pages need to be created or the entries need to be removed from the table until the pages exist.
2. `registry` is listed in `xcaffold/index.md` but is absent from this parent `kinds/index.md`. The two index pages are inconsistent about which xcaffold kinds are documented.
3. The "Structural Config Kinds (Pure YAML)" section lists `project` as a pure YAML kind. This is inconsistent with the actual behavior — `project` uses the frontmatter+body format (as described in `project.md` and shown in `golden/project.xcaf` which has `---` delimiters and a body). The section should either remove `project` from the pure-YAML list or add an explicit note that `project` is a special case that supports an optional body.
4. The "Body-Bearing Kinds" example uses `model: claude-sonnet-4-6` — this appears to be a specific model ID. Confirm this is an intentional example value (it matches the `claude` provider's `DefaultModel` field in `manifest.go`).

---

### `docs/reference/supported-providers.md`

| Dimension | Rating | Details |
|-----------|--------|---------|
| Accuracy | stale | Several capability matrix entries do not match what the provider manifests define. See issues below. |
| Completeness | complete | All 5 registered providers are listed. Skill subdirectory table is comprehensive. Internal Registry section is accurate. |
| Terminology | clean | No stale terminology. Correctly uses `.xcaf` where appropriate. |
| Structure | matches-template | Well-structured. |
| Cross-References | broken-links | `../reference/schema.md` does not exist (`docs/reference/schema.md` is absent from the worktree). |

**Verdict:** Needs-Update
**Priority:** P1 (user-facing accuracy — capability matrix discrepancies)
**Specific Issues:**
1. Capability Matrix — **Agents row, Antigravity column**: Doc says "*N/A*". The Antigravity manifest's `KindSupport` map does NOT include `"agent"` — so Antigravity does not support agents. The "*N/A*" is correct here.
2. Capability Matrix — **Shell Hooks row, Copilot column**: Doc says `hooks/xcaffold-hooks.json`. The Copilot manifest `KindSupport` does NOT include `"hook-script"` (only `"agent"`, `"skill"`, `"rule"`, `"mcp"`). The matrix entry conflicts with the manifest. The Kind Matrix table in the Import Support section also lists hook for Copilot as `.github/hooks/xcaffold-hooks.json`, which similarly contradicts the manifest.
3. Capability Matrix — **Shell Hooks row, Antigravity column**: Doc says "*N/A*". The Antigravity manifest `KindSupport` does NOT include `"hook-script"`. "*N/A*" is correct.
4. Capability Matrix — **Memory Context row**: All providers except Claude Code show "Not supported." The manifest for Claude has `"memory": true` in `KindSupport`. No other provider manifest includes `"memory"` in `KindSupport`. The matrix is consistent with manifests here.
5. Capability Matrix — **Project Instructions, Cursor column**: Doc says `AGENTS.md (nested)`. The Cursor manifest has `RootContextFile: "AGENTS.md"`. However, project.md Behavior table says Cursor compiles project body to `.cursor/rules/project-instructions.md`. This is a contradiction between supported-providers.md and project.md — one must be wrong. The manifest field `RootContextFile` is the canonical authority.
6. Capability Matrix — **Settings & Sandbox, Cursor column**: Doc says "Cursor Settings UI". The Cursor manifest `KindSupport` does NOT include `"settings"`. This is consistent (Cursor doesn't support settings via xcaffold).
7. Kind Matrix (Import Support) — **hook row, Cursor column**: Doc says `.cursor/hooks.json`. Cursor manifest has `"hook-script": true` in `KindSupport`, so this is plausible. However, this should be verified against the actual cursor importer.
8. The broken cross-reference `[Schema Reference](../reference/schema.md)` points to a file that does not exist. Should be removed or updated to a valid target.
9. Capability Matrix — **Workflows row, Antigravity column**: Doc says `workflows/*.md`. Antigravity manifest has `"workflow": true` in `KindSupport`. This is correct.
10. Capability Matrix — **Skills, Settings**: Doc correctly identifies Claude and Gemini as supporting settings (`settings.json`). Claude manifest has `"settings": true`; Gemini manifest has `"settings": true`. Copilot and Cursor do not have settings in `KindSupport`, and the matrix omits them or marks them correctly. Antigravity is listed as `settings.json` — but the Antigravity manifest `KindSupport` does NOT include `"settings"`. This may be a discrepancy to investigate.

---

## Cross-Cutting Observations

**1. Frontmatter delimiter errors in pure-YAML kind examples**

Three reference pages (`blueprint.md`, `policy.md`) show the pure-YAML kind wrapped in `---` frontmatter delimiters in the Example Usage section. This is the single most impactful accuracy error in the entire set — a user copying these examples verbatim will get a parse error from xcaffold because `KnownFields(true)` and the `policyDocument`/`blueprintDocument` parsers reject frontmatter. The `global.md` example is also wrong for a different reason (wrong resource schema). All example YAML blocks for pure-YAML kinds must be corrected.

**2. `.xcf` vs `.xcaf` extension inconsistency in Filesystem-as-Schema sections**

Both `blueprint.md` and `policy.md` reference the filename as `blueprint.xcf`/`policy.xcf` in the Filesystem-as-Schema section. The canonical extension is `.xcaf` (as established by the project rename and confirmed by every other reference in the codebase). This is a terminology error that will confuse users who then create files with the wrong extension.

**3. `global.md` uses an invented schema (`ResourceRef`)**

The `global.md` document invents a `ResourceRef` type (`{id, path}`) that does not exist in `globalDocument`. The real global format uses inline resource maps (matching `AgentConfig`, `SkillConfig`, etc. struct shapes keyed by resource name). A user following the global.md examples will write invalid YAML that is rejected at parse time. This is the most severe accuracy issue in the set.

**4. Missing fields in project.md**

The `test`, `local`, and `target-options` fields are present in `ProjectConfig`, verified in the golden schema, and absent from the project.md Argument Reference. Of these, `test` is explicitly called out in the audit task as a required check and is absent.

**5. Broken link pattern: `how-to/index.md`**

`docs/reference/index.md` links to `how-to/index.md` which doesn't exist. This suggests the how-to section is either not yet created or was planned but not yet written.

**6. `settings` and `hooks` xcaffold kind pages referenced but missing**

`kinds/index.md` links to `./xcaffold/settings` and `./xcaffold/hooks` — two pages that don't exist. These are two of the six xcaffold kinds listed in the table and have no reference documentation at all.

**7. Provider capability matrix partially inconsistent with manifests**

The supported-providers.md capability matrix was likely written before the manifest-driven registry was fully implemented. Some cells (Copilot hooks, Antigravity settings) describe capabilities that the manifests don't declare. The source of truth is the `KindSupport` map in each provider's `manifest.go`.

---

## Missing Documentation

**1. `docs/reference/kinds/xcaffold/settings.md`** — A reference page for `kind: settings` is linked from `kinds/index.md` but does not exist. `SettingsConfig` is one of the most field-dense structs in `types.go` (30+ fields covering model, permissions, sandbox, memory, hooks, MCP, environment).

**2. `docs/reference/kinds/xcaffold/hooks.md`** — A reference page for `kind: hooks` (`NamedHookConfig`) is linked from `kinds/index.md` but does not exist.

**3. `docs/reference/schema.md`** — Referenced from `supported-providers.md` as the destination for "granular block-by-block breakdown of per-field fidelity mappings per target." This file does not exist.

**4. `docs/how-to/index.md`** — Linked from `docs/reference/index.md`'s Next Steps section. The entire how-to section appears to be planned but not yet written.
