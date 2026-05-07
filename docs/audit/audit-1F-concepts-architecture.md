---
title: "Audit 1F: Concepts — Architecture Documentation"
date: 2026-05-07
scope: "6 architecture concept documents"
auditor: "Claude Haiku (read-only)"
---

# Audit Report: Architecture Concepts (Task 1F)

## Executive Summary

The architecture documentation is **70% accurate** with **3 critical structural gaps** and **1 field-level inconsistency**. The core concepts (One-Way Compiler, TargetRenderer/ProviderImporter symmetry, fidelity notes, multi-target rendering) are sound and well-explained. However, the documentation claims an `internal/bir/` package that does not exist, and the CapabilitySet struct definition in the code has diverged from what the documentation describes.

**Verdict:** MAJOR REVIEW REQUIRED — File a parallel task to reconcile the CapabilitySet documentation and either remove or populate the claimed BIR package.

---

## Per-File Assessment

### 1. `docs/concepts/architecture/index.md`
- **Status:** ACCURATE
- **Coverage:** 100% — brief index linking to detail pages
- **Issues:** None detected

### 2. `docs/concepts/architecture/architecture.md`
- **Status:** MAJOR ISSUE
- **Accuracy:** 85% — Structure and diagram are sound; package map contains false claim
- **Coverage:** 95% — Comprehensive internal package listing and lifecycle phases

**Critical Issue: Non-existent Package**
- **Line 173:** Claims `bir` package exists at `internal/bir/` containing `SemanticUnit`, `FunctionalIntent`, `ProjectIR`, and `bir.ReassembleWorkflow()`.
- **Reality:** No `internal/bir/` directory exists. `ls internal/` shows 24 subdirectories: `analyzer`, `ast`, `auth`, `blueprint`, `compiler`, `generator`, `importer`, `integration`, `judge`, `llmclient`, `optimizer`, `output`, `parser`, `policy`, `prompt`, `proxy`, `registry`, `renderer`, `resolver`, `state`, `templates`, `trace`, `translator` — no `bir`.
- **Impact:** Medium — Users reading this doc will look for a non-existent module in the codebase. Import statements referencing `bir` will fail.
- **Recommendation:** 
  - EITHER: Populate the claimed `internal/bir/` package with the interfaces described
  - OR: Remove the `bir` row from the package map and move BIR concepts to the `translator` section if they exist there

**Minor Issue: CapabilitySet Fields**
- **Line 155:** References `CapabilitySet` but doesn't detail its structure.
- **Context:** The detailed description appears in `provider-architecture.md` (see Issue #3 below).

**Accurate Sections:**
- Global Home bootstrap mechanism (lines 115–127) ✓
- File Taxonomy / kind: discriminator (lines 130–143) ✓
- CLI Lifecycle 8-phase orchestration (lines 224–246) ✓
- Compilation Output Structure diagram (lines 188–220) ✓

### 3. `docs/concepts/architecture/intermediate-representation.md`
- **Status:** ACCURATE
- **Accuracy:** 100%
- **Coverage:** 100% — Concise, clear explanation of IR's role across import/translate/apply/diff phases

**Verified Claims:**
- IR = provider-agnostic `ast.XcaffoldConfig` struct ✓
- Used in command phases (import, translate, apply, diff) ✓
- Serialization to disk via `--save-xcaf` flag ✓

### 4. `docs/concepts/architecture/multi-target-rendering.md`
- **Status:** ACCURATE with CONTEXT GAP
- **Accuracy:** 95% — Core concepts correct; one section relies on undefined package
- **Coverage:** 95% — AST as data/presentation separation well-explained; fidelity system clearly described

**Minor Issue: BIR Dependency**
- **Line 34:** References `paths: ["src/**/*.ts"]` vs `globs:` rendering difference.
- **Implied claim:** This difference is rendered per-target based on CapabilitySet fidelity mappings.
- **Accuracy:** The concept is correct, but the documentation does not explain the concrete mechanism (which `SkillArtifactDirs`, `RuleEncoding`, or field-level mapping produces this difference). Minor pedagogical gap, not a factual error.

**Accurate Sections:**
- AST as neutral data structure (lines 12–35) ✓
- TargetRenderer interface dispatch (lines 15–32) ✓
- CapabilitySet auto-dispatch model (lines 32–33) ✓
- Fidelity notes as non-errors (lines 40–42) ✓
- Five-target architecture (lines 44–56) ✓
- Target-determined output directories (lines 58–77) ✓
- MCP Shorthand and Settings Merge (lines 78–87) ✓
- ProviderImporter symmetry (lines 94–99) ✓

### 5. `docs/concepts/architecture/provider-architecture.md`
- **Status:** PARTIAL ACCURACY
- **Accuracy:** 80% — Core architecture sound; CapabilitySet definition outdated
- **Coverage:** 95% — Comprehensive import/render pipeline; missing CapabilitySet field mapping details

**Critical Issue: Outdated CapabilitySet Struct Definition**
- **Lines 153–171:** Documents CapabilitySet with these fields:
  ```
  Agents, Skills, Rules, Workflows, Hooks, Settings, MCP, Memory, ProjectInstructions,
  SkillSubdirs ([]string),
  ModelField (bool),
  RuleActivations ([]string)
  ```
- **Actual fields in code** (`internal/renderer/capabilities.go`):
  ```go
  Agents, Skills, Rules, Workflows, Hooks, Settings, MCP, Memory, ProjectInstructions,
  SkillArtifactDirs (map[string]string),  ← CHANGED from []string
  RuleEncoding (RuleEncodingCapabilities),  ← NEW field
  AgentNativeToolsOnly (bool),  ← NEW field, not in doc
  RuleActivations ([]string)
  ```
- **Missing from doc:** `ModelField` doesn't exist in code; `RuleEncoding` struct not documented; `AgentNativeToolsOnly` field is new.
- **Impact:** High — Code examples in the doc will not compile; developers cannot determine what fields are actually available on CapabilitySet.

**Capability Matrix Table (Lines 177–188): Partially Inaccurate**
- **Row "ModelField":** Claims ModelField values (claude=yes, cursor=no, gemini=yes, copilot=yes, antigravity=no).
- **Reality:** No `ModelField` in actual CapabilitySet. The test file (`provider_features_test.go`) asserts `SkillArtifactDirs`, `RuleActivations`, but NOT `ModelField`.
- **Interpretation:** The doc's "ModelField" may be a leftover from an earlier design where agents could declare a model field. The actual code moved this to `AgentNativeToolsOnly` (only Claude = true).

**Accurate Sections:**
- ProviderImporter interface (lines 49–68) ✓
- Kind Classification table (lines 71–85) ✓
- Layout Types (lines 87–98) ✓
- Shared Importer Helpers (lines 101–111) ✓
- Provider Detection (lines 113–115) ✓
- TargetRenderer interface (lines 122–140) ✓
- Orchestrator dispatch logic (lines 193–203) ✓
- FidelityNotes section (lines 205–209) ✓
- Per-provider passthrough (lines 243–247) ✓
- Native Provider Runtime Loading (lines 261–274) ✓
- Adding a New Provider (lines 278–288) ✓

### 6. `docs/concepts/architecture/translation-pipeline.md`
- **Status:** MAJOR ISSUE
- **Accuracy:** 60% — References non-existent BIR package throughout; core translator logic undefined
- **Coverage:** 40% — Claims extensive BIR functionality not visible in code

**Critical Issue: BIR Package Claims**
- **Line 14:** `bir.ImportWorkflow()` — No such function in codebase.
- **Line 15:** `bir.DetectIntents()` — No such function.
- **Line 19–22:** `translator.Translate()` — Function exists (`internal/translator/workflow.go`), but it's not documented what signature or intents it processes.
- **Line 35:** `bir.ReassembleWorkflow()` — No such function in `internal/translator/` or any other package.

**Reality Check:**
- `internal/translator/` exists and contains:
  - `rules.go` — Rule translation
  - `workflow.go` — Workflow lowering (TranslateWorkflow)
  - `workflow_test.go` — Tests
- **Missing:** No BIR pipeline, no SemanticUnit parsing, no intent detection per the documentation.
- **Inference:** The document describes a planned or discarded architecture that was not implemented, or the functionality exists elsewhere (e.g., in compiler or importer) and is not exposed as a `bir` package.

**Accurate Sections:**
- Workflow lowering concept (lines 28–33) — general idea correct, specific function names wrong
- Four-tiered strategies for TranslateWorkflow (lines 30–33) — strategy names match code ✓

---

## Terminology Audit

### Correct Terminology
- `.xcaf` format — Correct throughout ✓
- `kind:` discriminator — Correct ✓
- `TargetRenderer` interface — Correct ✓
- `ProviderImporter` interface — Correct ✓
- `FidelityNote` — Correct ✓
- `CapabilitySet` — Correct (but struct fields are outdated) ⚠️
- `ProviderExtras` — Correct ✓

### Obsolete/Missing Terminology
- `ModelField` — Documented in capability matrix but does not exist in code ✗
- `bir` package — Claimed throughout but does not exist ✗
- `SemanticUnit`, `FunctionalIntent`, `ProjectIR` — Claimed as part of BIR but not found ✗
- `RuleEncoding` struct — Exists in code but not documented in CapabilitySet section ✗
- `AgentNativeToolsOnly` field — Exists in code but not documented ✗

---

## Cross-Check Against Source Code

### Package Structure (architecture.md lines 148–184)
| Claimed | Exists? | Notes |
|---------|---------|-------|
| `ast` | ✓ | Verified: `internal/ast/` contains types, AST definitions |
| `parser` | ✓ | Verified: `internal/parser/` contains YAML parsing |
| `policy` | ✓ | Verified: `internal/policy/` contains policy engine |
| `compiler` | ✓ | Verified: `internal/compiler/` contains main Compile() |
| `renderer` | ✓ | Verified: `internal/renderer/` contains TargetRenderer interface |
| `renderer/shared` | ✓ | Verified: `internal/renderer/shared/` exists |
| `renderer/claude` | ✓ | Verified: `internal/renderer/claude/` exists |
| `renderer/cursor` | ✓ | Verified: `internal/renderer/cursor/` exists |
| `renderer/copilot` | ✓ | Verified: `internal/renderer/copilot/` exists |
| `renderer/gemini` | ✓ | Verified: `internal/renderer/gemini/` exists |
| `renderer/antigravity` | ✓ | Verified: `internal/renderer/antigravity/` exists |
| `importer` | ✓ | Verified: `internal/importer/` contains ProviderImporter interface |
| `importer/claude` | ✓ | Verified |
| `importer/cursor` | ✓ | Verified |
| `importer/gemini` | ✓ | Verified |
| `importer/copilot` | ✓ | Verified |
| `importer/antigravity` | ✓ | Verified |
| `output` | ✓ | Verified |
| `state` | ✓ | Verified |
| `registry` | ✓ | Verified |
| `templates` | ✓ | Verified |
| `analyzer` | ✓ | Verified |
| **`bir`** | **✗** | **NOT FOUND** — 0 files, 0 packages |
| `translator` | ✓ | Verified |
| `optimizer` | ✓ | Verified |
| `resolver` | ✓ | Verified |
| `generator` | ✓ | Verified |
| `judge` | ✓ | Verified |
| `proxy` | ✓ | Verified |
| `trace` | ✓ | Verified |
| `auth` | ✓ | Verified |
| `llmclient` | ✓ | Verified |
| `prompt` | ✓ | Verified |
| `integration` | ✓ | Verified |

---

## CapabilitySet Struct Divergence (Detailed Analysis)

### Documentation Claims (provider-architecture.md lines 156–170)
```go
type CapabilitySet struct {
    Agents              bool
    Skills              bool
    Rules               bool
    Workflows           bool
    Hooks               bool
    Settings            bool
    MCP                 bool
    Memory              bool
    ProjectInstructions bool
    SkillSubdirs        []string   // e.g., ["references", "scripts", "assets"]
    ModelField          bool       // <-- CLAIMED
    RuleActivations     []string   // e.g., ["always", "path-glob"]
}
```

### Actual Code (internal/renderer/capabilities.go lines 20–41)
```go
type CapabilitySet struct {
    Agents              bool
    Skills              bool
    Rules               bool
    Workflows           bool
    Hooks               bool
    Settings            bool
    MCP                 bool
    Memory              bool
    ProjectInstructions bool
    // SkillArtifactDirs maps canonical artifact names to provider output subdirectory
    // names. An empty string value means the artifact files are flattened to the
    // skill root directory alongside SKILL.md (no subdirectory created).
    SkillArtifactDirs   map[string]string // canonical name → provider output subdir ("" = flatten to root)
    RuleActivations     []string          // e.g., ["always", "path-glob"]
    RuleEncoding        RuleEncodingCapabilities // <-- NEW, NOT IN DOC
    AgentNativeToolsOnly bool  // <-- NEW, NOT IN DOC
}
```

### Discrepancies
1. **SkillSubdirs → SkillArtifactDirs**: Changed from `[]string` to `map[string]string`.
   - **Impact:** Documentation type is wrong; code uses a mapping, not a list.
2. **ModelField removed**: No longer in struct.
   - **Impact:** Capability matrix table (lines 177–188) claims values for non-existent field.
3. **RuleEncoding added**: New struct field not documented.
   - **Details:** Defined as:
     ```go
     type RuleEncodingCapabilities struct {
         Description string  // "frontmatter" | "prose" | "omit"
         Activation  string  // "frontmatter" | "omit"
     }
     ```
4. **AgentNativeToolsOnly added**: New bool field not documented.
   - **Comment in code:** "Only Claude should set this true."

---

## Fidelity Notes System

**Verdict:** Correctly documented and verified. ✓

The documentation accurately describes:
- FidelityNote struct (lines 205–209 of provider-architecture.md)
- RENDERER_KIND_UNSUPPORTED codes
- FIELD_UNSUPPORTED codes
- Fidelity codes catalog in `internal/renderer/fidelity_codes.go`
- Auto-emission by orchestrator when CapabilitySet capability is false

---

## Capability Matrix Accuracy

### Documented (provider-architecture.md lines 177–188)
| Field | Claude | Cursor | Gemini | Copilot | Antigravity |
|---|---|---|---|---|---|
| ModelField | yes | no | yes | yes | **no** |

### Actual (from provider_features_test.go)
- Test does NOT assert `ModelField` — field doesn't exist.
- Test DOES assert `SkillArtifactDirs` and `RuleActivations` — these match doc's intent but doc calls them different things.

### Status
- **Memory (Antigravity):** Doc says `no` (correct). Test confirms `memory: false` with comment "deferred — native format not yet implemented".
- **All other rows:** Boolean flags (Agents, Skills, etc.) are accurate per test assertions.

---

## Structural Issues Summary

| Issue | Severity | File | Location | Impact |
|-------|----------|------|----------|--------|
| BIR package claim | CRITICAL | architecture.md | Line 173 | Package referenced but doesn't exist |
| BIR pipeline (10+ fn claims) | CRITICAL | translation-pipeline.md | Lines 14–35 | Describes non-existent functions; misleads on architecture |
| CapabilitySet struct outdated | CRITICAL | provider-architecture.md | Lines 153–171 | Code uses different fields; examples won't compile |
| ModelField in matrix | HIGH | provider-architecture.md | Line 188 | Capability matrix references non-existent field |
| SkillSubdirs type wrong | HIGH | provider-architecture.md | Line 167 | Doc says `[]string`, code is `map[string]string` |

---

## Recommendations

### Immediate (Blocking Review)
1. **Resolve BIR vs. Translator Architecture:**
   - Determine if `internal/bir/` should be populated (if per design) OR
   - Move any BIR concepts to the `translator` package documentation if they exist elsewhere
   - Rewrite translation-pipeline.md with actual function signatures from `internal/translator/workflow.go`

2. **Update CapabilitySet Documentation:**
   - Replace `SkillSubdirs: []string` with `SkillArtifactDirs: map[string]string` with example mapping
   - Remove the `ModelField` row from capability matrix (lines 177–188)
   - Add `RuleEncoding: RuleEncodingCapabilities` field documentation with enum values
   - Add `AgentNativeToolsOnly: bool` field documentation with note "Only Claude: true"
   - Update capability matrix to show `RuleEncoding` support per provider (or drop if identical)

3. **Remove/Reconcile ModelField References:**
   - Search codebase for any remaining `ModelField` references
   - If deprecated, document the migration to `AgentNativeToolsOnly`
   - If still used, restore the field to CapabilitySet

### Secondary (Quality Improvement)
1. **Add concrete examples to CapabilitySet section:**
   - Show actual SkillArtifactDirs mapping for each provider (e.g., Claude: `{references: "references", ...}`)
   - Link to `provider_features_test.go` line ranges for each provider's exact capabilities

2. **Clarify translator vs. BIR terminology:**
   - Define where semantic analysis actually occurs (compiler? translator? importer?)
   - Document the actual workflow for lowering WorkflowConfig to provider primitives

3. **Cross-link to provider_features_test.go:**
   - Capability matrix should reference this test as the source of truth with line numbers

---

## Audit Checklist

| Item | Result | Notes |
|------|--------|-------|
| All .xcaf terminology consistent | ✓ | Correct throughout |
| Package map matches internal/ structure | ✗ | BIR package doesn't exist |
| CapabilitySet definition accurate | ✗ | Fields diverged; ModelField removed; SkillArtifactDirs type changed |
| Capability matrix matches code | ✗ | ModelField row is meaningless; no AgentNativeToolsOnly row |
| FidelityNote system described | ✓ | Accurate and complete |
| ProviderImporter interface correct | ✓ | Matches code |
| TargetRenderer interface correct | ✓ | Matches code |
| Renderer dispatch logic correct | ✓ | Orchestrate() described accurately |
| Multi-target rendering concept | ✓ | AST separation clearly explained |
| Intermediate Representation | ✓ | Concise and accurate |
| BIR architecture described | ✗ | Package doesn't exist; functions not found |

---

## Files Requiring Update (Priority Order)

1. **docs/concepts/architecture/provider-architecture.md** — Update CapabilitySet struct (3 fields changed); fix capability matrix
2. **docs/concepts/architecture/architecture.md** — Remove or populate `bir` package row; update line 173
3. **docs/concepts/architecture/translation-pipeline.md** — Rewrite BIR pipeline section; verify translator functions against actual code
4. Cross-check: Verify whether `internal/bir/` was intentionally deferred or mistakenly documented

---

## Conclusion

The architecture documentation provides a strong foundational understanding of the One-Way Compiler model, TargetRenderer/ProviderImporter symmetry, and multi-target rendering. However, it claims a non-existent BIR package in three places and documents an outdated CapabilitySet struct definition. These gaps must be resolved before the documentation can be considered authoritative for developers implementing new providers or extending the renderer architecture.

**Overall Assessment:** 70% accurate. Core concepts solid; structural discrepancies require immediate reconciliation.
