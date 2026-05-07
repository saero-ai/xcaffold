---
title: "Audit 1J: Best Practices Documentation"
status: complete
date: 2026-05-07
auditor: code-auditor
---

# Audit 1J: Best Practices Documentation

## Summary

**Files Audited:** 6  
**Status:** 1 Critical Issue, 1 Broken Link, 2 Warnings, 2 Passes  
**Verdict:** BLOCKED — fix critical issue and broken link before merge  
**Priority:** HIGH (blocking issue affects user experience)

---

## Executive Summary Table

| File | Lines | Status | Issues | Grade |
|------|-------|--------|--------|-------|
| `index.md` | 15 | ❌ BLOCKED | Broken link to non-existent file | F |
| `blueprint-design.md` | 236 | ⚠️ WARN | Field naming inconsistency, missing memory field mention | C+ |
| `policy-organization.md` | 207 | ✅ PASS | Complete, accurate, well-structured | A |
| `project-layouts.md` | 131 | ✅ PASS | Complete, examples match schema | A |
| `skill-organization.md` | 154 | ⚠️ WARN | Incomplete artifacts documentation, missing examples | B- |
| `workspace-context.md` | 198 | ✅ PASS | Clear, well-scoped, accurate | A |

---

## Detailed Assessment

### 1. `index.md` — CRITICAL ISSUE

**Status:** ❌ BLOCKED  
**Lines:** 15  
**Grade:** F

#### Issue 1: Broken Cross-Reference Link
Line 15 references a non-existent file:
```markdown
- [Cross-Provider Deployment](cross-provider) — Compiling one source to multiple AI providers, understanding fidelity notes, and using per-target overrides.
```

**Finding:** The file `docs/best-practices/cross-provider.md` does not exist. This link will break in CI/CD systems that validate markdown references.

**Cross-check:** No file with "cross" or "provider" in the name exists in `docs/best-practices/`.

**Action Required:** Either:
1. Create the missing `cross-provider.md` documentation file, OR
2. Remove the reference from the index

**Severity:** CRITICAL — broken link blocks documentation builds and confuses users.

---

### 2. `blueprint-design.md` — WARNING

**Status:** ⚠️ WARNING  
**Lines:** 236  
**Grade:** C+

#### Issue 1: Field Naming Inconsistency
**Lines 33, 88, 99, 164**

In YAML examples, the field is shown as `mcp:` (lowercase plural):
```yaml
mcp:
  - database-tools
```

But in `internal/ast/types.go` BlueprintConfig, the field is:
```go
MCP []string `yaml:"mcp,omitempty"`
```

**Verification:** This is correct per schema — no issue. The examples use valid kebab-case naming ✅

#### Issue 2: Memory Field Not Documented
**Lines 8, 47–49, 98–102, 160–177**

BlueprintConfig struct in types.go shows:
```go
Memory []string `yaml:"memory,omitempty"`
```

But the documentation does not show memory selection in examples. Line 8 mentions "memory entries" in prose but no YAML example includes `memory:` field.

**Example found missing:** Blueprint definition does not show:
```yaml
memory:
  - memory-entry-name
```

**Cross-check:** The golden blueprint at `schema/golden/blueprint.xcaf` **does** include `memory: []` (line 21), confirming this is a valid field.

**Action Required:** Add memory field to at least one blueprint example (recommend line 38 or 78 where agents list is shown).

**Severity:** MEDIUM — documentation incomplete but not incorrect.

#### Issue 3: Inconsistent Defaults
Lines 88–100 show `settings: development` and `hooks: pre-commit` without explaining default behavior when omitted. The text says "all named configurations are included" (line 210) but examples show explicit selection. Recommend clarifying whether line 210 applies to blueprints or to non-blueprint projects.

**Severity:** LOW — helpful to clarify but not blocking.

---

### 3. `policy-organization.md` — PASS

**Status:** ✅ PASS  
**Lines:** 207  
**Grade:** A

**Findings:**
- All six PolicyConfig fields (`name`, `description`, `severity`, `target`, `match`, `require`, `deny`) are correctly explained
- Examples match the golden policy schema at `schema/golden/policy.xcaf`
- Severity levels (error, warning, off) match schema enum: `error,warning,off` ✅
- Target types match schema: `agent,skill,rule,hook,settings,output` ✅
- Match conditions (`has-tool`, `has-field`, `name-matches`, `target-includes`) are correct
- Use cases are practical and well-scoped
- No references to removed or deprecated patterns
- Clear severity decision guide (lines 199–207)

**Cross-checks Passed:**
- ✅ All kebab-case keys (no camelCase)
- ✅ Examples use `.xcaf` format
- ✅ No references to xcf/ or internal terminology
- ✅ Field names match schema exactly

**Recommendation:** This is exemplary documentation. No changes required.

---

### 4. `project-layouts.md` — PASS

**Status:** ✅ PASS  
**Lines:** 131  
**Grade:** A

**Findings:**
- Correctly states that `xcaf/` is scanned recursively ✅
- Directory-per-resource requirement is accurate: "agents use `xcaf/agents/<name>/agent.xcaf`" ✅
- Golden examples match implementation (single-file, organized-by-type, domain-based layouts)
- Clear distinction between source files (`xcaf/`), state files (`.xcaffold/`), and compiled output (`.claude/`, `.cursor/`, etc.) ✅
- Emphasizes filesystem as source of truth ✅
- Key rules summary (lines 124–131) is correct:
  - ✅ Agents use `xcaf/agents/<name>/agent.xcaf`
  - ✅ Skills use `xcaf/skills/<name>/skill.xcaf`
  - ✅ Duplicate resource IDs rejected ✅ (`mergeAllStrict` enforced)
  - ✅ `.xcaffold/` is gitignored ✅
  - ✅ Never edit compiled output ✅

**Cross-checks Passed:**
- ✅ All examples use `.xcaf` format and correct directory structure
- ✅ Canonical filenames enforced (no `flat-agent.xcaf` examples)
- ✅ No references to `xcf/`
- ✅ All kebab-case naming

**Verification Against Source:**
- `skill_validation.go` enforces canonical filename `skill.xcaf` at line 26 ✅
- Parser rejects flat files as claimed ✅
- Example paths are correct ✅

**Recommendation:** No changes required. This is complete and accurate.

---

### 5. `skill-organization.md` — WARNING

**Status:** ⚠️ WARNING  
**Lines:** 154  
**Grade:** B-

#### Issue 1: Artifacts Field Mentioned in Text but Not in Examples
**Line 42 mentions artifacts** in code example structure, but the `skill.xcaf` frontmatter does not show the `artifacts:` declaration. 

**Finding:** The text says skills "should be authored in a subdirectory" and shows the layout with `references/`, `scripts/`, `assets/`, `examples/` subdirectories, but does not explain that the `.xcaf` file must *declare* these via the `artifacts:` field.

**Correct Schema (from types.go):**
```yaml
---
kind: skill
version: "1.0"
name: code-review
artifacts:
  - references
  - scripts
  - assets
  - examples
---
```

**Current Documentation Problem:** Lines 42–57 show the directory structure but the `.xcaf` file is not shown declaring `artifacts:`. A user following this guide would create the directory structure but their skill would fail to compile because the `.xcaf` frontmatter doesn't list the artifacts.

**Action Required:** Add to line 49 or nearby:
```yaml
---
kind: skill
version: "1.0"
name: code-review
artifacts:
  - references
  - scripts
  - assets
  - examples
---
```

**Severity:** MEDIUM — users could fail to compile skills without knowing why.

#### Issue 2: Missing Canonical Subdirectories List
**Lines 61–64** warn about "only `references/`, `scripts/`, `assets/`, and `examples/` are recognized" but the **artifacts field is never shown in examples** to demonstrate how to declare them.

**Cross-check:** `skill_validation.go` line 12–17 confirms these are the only canonical subdirs:
```go
var subdirExtensionRules = map[string]bool{
    "references": true,
    "scripts":    true,
    "assets":     true,
    "examples":   true,
}
```

**Action Required:** Add a note or code example showing:
```yaml
artifacts:
  - references
  - scripts
  - assets
  - examples
```

**Severity:** MEDIUM — documentation mentions artifacts directory names but never shows how to declare them.

#### Issue 3: Incomplete Coverage of Artifact Behavior
**Lines 127–142** explain how provider renderers handle subdirectories but do not explain what happens if `artifacts:` is declared but no matching directory exists on disk. 

**Finding:** `skill_validation.go` line 100–106 shows that declared artifacts that don't exist on disk produce an error:
```go
for _, declared := range artifacts {
    if !declaredArtifactsSeen[declared] {
        result.Errors = append(result.Errors, fmt.Errorf(
            "declared artifact %q does not exist as subdirectory in skill %q",
            declared, skillID))
    }
}
```

This validation is not mentioned in the documentation.

**Action Required:** Add note: "If you declare an artifact in `artifacts:` but the directory does not exist on disk, `xcaffold validate` will fail with an error."

**Severity:** LOW — helpful but not critical since error messages would guide the user.

---

### 6. `workspace-context.md` — PASS

**Status:** ✅ PASS  
**Lines:** 198  
**Grade:** A

**Findings:**
- Correctly identifies `kind: context` as the mechanism for workspace instructions ✅
- Provider output file mappings are accurate (lines 10–16):
  - Claude → `CLAUDE.md` ✅
  - Cursor → `AGENTS.md` ✅
  - Gemini → `GEMINI.md` ✅
  - Antigravity → `GEMINI.md` ✅
  - Copilot → `.github/copilot-instructions.md` ✅
- Correctly notes that Antigravity and Gemini both write to `GEMINI.md` (lines 17–19) ✅
- Four rules are well-articulated:
  - Rule 1: Document only absolute laws ✅
  - Rule 2: Isolate tactical rules to `kind: rule` with `paths:` ✅
  - Rule 3: Write instructions, not documentation ✅ (imperative voice guidance is excellent)
  - Rule 4: Use multiple context documents for multi-provider projects ✅
- ContextConfig fields match schema:
  - `name` ✅
  - `description` ✅
  - `default` ✅
  - `targets` ✅
  - Body (markdown) ✅
- Examples use correct kebab-case naming ✅
- "200-line signal" guidance (lines 131–148) is practical and well-reasoned ✅
- Split decision reference (lines 142–148) is clear and helpful ✅

**Cross-checks Passed:**
- ✅ All examples use `.xcaf` format
- ✅ All field names are kebab-case
- ✅ No references to internal terminology
- ✅ Output paths are correct
- ✅ Provider scoping via `targets:` is accurately described

**Verification Against Schema:**
- ContextConfig struct at types.go includes all documented fields ✅
- Example at line 34–39 correctly shows targets as list ✅
- Line 154 Rule 4 example correctly shows provider-scoped contexts ✅

**Recommendation:** No changes required. Exemplary documentation.

---

## Cross-Reference Verification Results

| Reference | Target | Status | Finding |
|-----------|--------|--------|---------|
| index.md:15 | `cross-provider.md` | ❌ MISSING | File does not exist |
| blueprint-design.md (multi) | internal/ast BlueprintConfig | ✅ MATCH | Schema fields accurate |
| policy-organization.md (multi) | internal/ast PolicyConfig | ✅ MATCH | Schema fields accurate |
| project-layouts.md:79 | skill canonical filename | ✅ MATCH | `skill.xcaf` enforced by parser |
| skill-organization.md:18–64 | subdirectory rules | ✅ MATCH | `skill_validation.go` confirms rules |
| workspace-context.md:10–16 | provider output paths | ✅ MATCH | Renderer behavior verified |

---

## Schema Compliance Checklist

### Naming Convention (kebab-case)
- ✅ All YAML keys use kebab-case
- ✅ No camelCase examples
- ✅ No snake_case fields
- ✅ All resource IDs follow `^[a-z0-9-]+$` pattern

### File Format Conventions
- ✅ All examples use `.xcaf` extension
- ✅ All use frontmatter + body format where applicable
- ✅ All resource kinds are recognized types

### Directory Structure
- ✅ Directory-per-resource pattern is standard
- ✅ Canonical filenames are consistent (`agent.xcaf`, `skill.xcaf`, `rule.xcaf`, `workflow.xcaf`, `mcp.xcaf`)
- ✅ Subdirectory limits (max depth 1) are mentioned

### Schema Field Accuracy
- ✅ BlueprintConfig fields documented
- ✅ PolicyConfig fields documented
- ✅ ContextConfig fields documented
- ✅ SkillConfig artifacts field referenced (but examples incomplete)

---

## Verdict

**BLOCKED — Two issues must be resolved before merge:**

1. **CRITICAL:** Remove or create the broken `cross-provider` link in `index.md` (line 15)
2. **MEDIUM:** Add `artifacts:` field declaration to skill examples in `skill-organization.md` (lines 42–57)

**WARNINGS (should address before merge):**
- Add memory field example to blueprint documentation
- Clarify artifact validation error behavior in skill documentation

**PASSES (no action required):**
- ✅ `policy-organization.md`
- ✅ `project-layouts.md`
- ✅ `workspace-context.md`

---

## Recommendations for Future Updates

1. **Add schema reference links:** Each guide could link to the authoritative schema docs (e.g., `See [BlueprintConfig Reference](../reference/schema.md#blueprintconfig)`)

2. **Add validation error examples:** Show what errors users will see if they misconfigure resources, with solutions

3. **Create the cross-provider guide:** This is a significant gap. Topics should include:
   - How to target specific providers with `targets:` field
   - Fidelity notes for dropped fields
   - Provider-specific field overrides with `target-options:`
   - Examples for multi-target projects

4. **Expand artifacts documentation:** Add a dedicated section showing:
   - How to declare artifacts in `.xcaf` frontmatter
   - Validation rules for declared vs. actual directories
   - Error messages and how to fix them

---

## Files Affected

- `docs/best-practices/index.md` — Remove broken link or create missing file
- `docs/best-practices/skill-organization.md` — Add `artifacts:` field to examples
- `docs/best-practices/blueprint-design.md` — Add memory field example (optional)

---

**Audit completed:** 2026-05-07  
**Auditor:** code-auditor (read-only documentation audit)  
**Next step:** Fix CRITICAL issue before merge; address MEDIUM warnings in next iteration
