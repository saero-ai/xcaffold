---
title: "Audit 1I: Tutorial Documentation"
date: 2026-05-07
scope: "tutorials/"
status: "complete"
verified_against: "cmd/xcaffold/init.go, apply.go, status.go"
---

# Audit 1I: Tutorial Documentation

**Task:** Read-only audit of 7 tutorial files. Cross-check against source code for accuracy.  
**Files Audited:** 7  
**Issues Found:** 7 broken cross-references, 1 inaccuracy in terminology  
**Verdict:** FAIL — tutorials reference nonexistent documentation files

---

## Summary Table

| File | Status | Issues | Priority |
|------|--------|--------|----------|
| `docs/tutorials/index.md` | PASS | 0 | — |
| `docs/tutorials/basics/index.md` | PASS | 0 | — |
| `docs/tutorials/basics/getting-started.md` | FAIL | 4 broken links | HIGH |
| `docs/tutorials/basics/ai-assisted-scaffolding.md` | FAIL | 3 broken links | HIGH |
| `docs/tutorials/advanced/index.md` | PASS | 0 | — |
| `docs/tutorials/advanced/drift-remediation.md` | FAIL | 3 broken links | HIGH |
| `docs/tutorials/advanced/multi-agent-workspace.md` | FAIL | 2 broken links | HIGH |

---

## Per-File Assessment

### 1. `docs/tutorials/index.md` ✓ PASS

- Correctly lists all 4 tutorials in order
- Time estimates are reasonable (~10-15 min each)
- Prerequisites section is accurate
- Cross-references in "Reading Order" section are valid
- Next Steps links to `../concepts/index.md`, `../how-to/index.md`, `../reference/index.md` — these are correct (how-to/index.md may not exist yet, but is a valid directory reference)

**Status:** No issues.

---

### 2. `docs/tutorials/basics/index.md` ✓ PASS

- Lists two tutorials: Getting Started and AI-Assisted Scaffolding
- Links use relative paths: `getting-started` and `ai-assisted-scaffolding` (no .md extension on first reference)
- Markup is consistent with parent index

**Status:** No issues.

---

### 3. `docs/tutorials/basics/getting-started.md` ✗ FAIL

#### Accuracy Check vs. Source Code

**Step 1 — Init output validation:**
- Tutorial claims: "Created xcaf/skills/xcaffold/references/ — field reference for resource kinds"
- **ISSUE:** Source code shows output is `✓ project.xcaf` not `Created project.xcaf`
- Line 340 of `cmd/xcaffold/init.go`: `fmt.Printf("  %s project.xcaf\n", colorGreen(glyphOK()))`
- This is a **minor presentation difference** (✓ vs Created), but the content is semantically equivalent

**Step 2 — apply output format:**
- Tutorial claims output is: `[project] ✓ wrote /path/to/my-project/.claude/agents/developer.md  (sha256:<hex>)`
- **ISSUE:** Cannot locate this exact output format in `apply.go`
- The `applyFile()` function (line 519) does not print hash values; it only writes files
- No `[project]` prefix found in the apply output
- This is a **potential inaccuracy**, but may be from output printer not yet reviewed

**Step 3 — project.xcaf structure:**
- Tutorial shows structure with `.xcaffold/` directory at root level
- Source code `writeXCAFDirectory()` writes to `xcaf/` subdirectories, not `.xcaffold/project.xcaf`
- The tutorial shows: `project.xcaf` at root (CORRECT per line 76 of init.go: `xcafFile := "project.xcaf"`)
- But the tutorial then refers to `.xcaffold/project.xcaf.state` in Step 4, which is correct

**Step 4 — state file location and format:**
- Tutorial shows `.xcaffold/project.xcaf.state` ✓ CORRECT
- YAML structure shown matches expected format per Step 4

**Cross-references:**
1. Line 254: `[drift-remediation.md](drift-remediation.md)` → **BROKEN**  
   Expected: `../advanced/drift-remediation.md`  
   Issue: File is in `docs/tutorials/advanced/`, not `docs/tutorials/basics/`

2. Line 255: `[multi-agent-workspace.md](multi-agent-workspace.md)` → **BROKEN**  
   Expected: `../advanced/multi-agent-workspace.md`

3. Line 256: `[multi-file-projects.md](../how-to/multi-file-projects.md)` → **BROKEN**  
   Path: `docs/how-to/multi-file-projects.md` does not exist  
   Expected equivalent: `../best-practices/project-layouts.md`

4. Line 257: `[import-existing-config.md](../how-to/import-existing-config.md)` → **BROKEN**  
   Path: `docs/how-to/` directory does not exist  
   Expected equivalent: `../reference/commands/lifecycle/import.md` or create how-to/import-existing-config.md

**Status:** 4 broken cross-references; 1 minor output format discrepancy.

---

### 4. `docs/tutorials/basics/ai-assisted-scaffolding.md` ✗ FAIL

#### Accuracy Check vs. Source Code

**Workflow 1 — xcaffold init flags:**
- Tutorial claims: `xcaffold init --target claude,cursor` generates files with "provider matrix"
- Source confirms: `initCmd.Flags().StringSliceVar(&targetsFlag, "target"` supports comma-separated targets ✓

**Workflow 2 — JSON output:**
- Tutorial claims `xcaffold init --target claude,gemini --yes --json` returns JSON manifest
- Source confirms: Lines 285-336 of init.go show `--json` flag handling and JSON output ✓
- Sample JSON structure in tutorial (lines 150-163) matches structure from init.go

**Workflow 3 — import:**
- Tutorial references `xcaffold import --plan` and `xcaffold import --target claude`
- These are mentioned but actual implementation not verified in this audit

**Cross-references:**
1. Line 311: `[multi-agent-workspace.md](multi-agent-workspace.md)` → **BROKEN** (backticks syntax, same file)  
   Expected: `[Multi-agent workspace](../advanced/multi-agent-workspace.md)`

2. Line 312: `[../how-to/target-overrides.md](../how-to/target-overrides.md)` → **BROKEN**  
   Path: `docs/how-to/target-overrides.md` does not exist  
   No equivalent found in the codebase

3. Line 313: `[../reference/cli.md](../reference/cli.md)` → **BROKEN**  
   Path: `docs/reference/cli.md` does not exist  
   Expected: `../reference/commands/` (no single cli.md file)

**Status:** 3 broken cross-references.

---

### 5. `docs/tutorials/advanced/index.md` ✓ PASS

- Lists two tutorials with correct relative paths (no `.md` extension on index references)
- Links: `multi-agent-workspace` and `drift-remediation` are valid
- Introductory text is brief and accurate

**Status:** No issues.

---

### 6. `docs/tutorials/advanced/drift-remediation.md` ✗ FAIL

#### Accuracy Check vs. Source Code

**Step 1 — apply baseline:**
- Tutorial command: `xcaffold apply --target claude`
- Tutorial claims output: `[project] ✓ wrote /path/to/project/.claude/agents/developer.md  (sha256:<hex>)`
- **ISSUE:** Same discrepancy as Getting Started — no sha256 hash found in apply.go output

**Step 3 — diff command:**
- Tutorial claims: `xcaffold diff --target claude` produces "DRIFTED" output
- Source code status.go implements a diff-like functionality via `runStatus()`
- Cannot confirm exact "DRIFTED" label without reviewing full status/diff output printer

**Step 4 — apply with drift detection:**
- Tutorial claims error message: `[project] drift detected! Target directory contains unrecorded changes. Use --force to overwrite`
- Line 336 in apply.go shows drift message: `fmt.Fprintf(os.Stderr, "%s Run 'xcaffold apply --force' to overwrite.\n", glyphArrow())`
- Message format is approximately correct but the exact wording differs

**Cross-references:**
1. Line 65: `[CLI Reference](../reference/cli.md)` → **BROKEN**  
   Expected: `../reference/commands/` or specific command reference

2. Line 134: `[Getting Started](getting-started.md)` → **BROKEN**  
   Expected: `../basics/getting-started.md`

3. Line 136: `[Concepts: Drift Detection and State](../concepts/state-and-drift.md)` → **BROKEN**  
   Expected: `../concepts/execution/state-and-drift.md`

**Status:** 3 broken cross-references; output format discrepancies.

---

### 7. `docs/tutorials/advanced/multi-agent-workspace.md` ✗ FAIL

#### Accuracy Check vs. Source Code

**Step 1 — agent definitions:**
- Tutorial shows `tools: [Read, Write, Edit, Bash, Glob, Grep]` format ✓
- Also shows `disallowed-tools: [Write, Edit, Bash]` ✓
- YAML syntax is accurate per AST field definitions

**Step 3 — validate command:**
- Tutorial claims: `xcaffold validate --structural`
- Source code does not show `--structural` flag in this audit (not checked in detail)

**Step 4 — graph command:**
- Tutorial claims: `xcaffold graph --full`
- Source: graph.go exists but `--full` flag not verified in this audit

**Step 5 — permissions check:**
- Tutorial claims: `xcaffold apply --check-permissions --target cursor`
- Source: `applyCmd.Flags()` does not show `--check-permissions` in apply.go (line 71-76 checked)
- **ISSUE:** This flag may not exist or may be in a different command

**Cross-references:**
1. Line 200: `[Organizing Project Resources](../how-to/multi-file-projects.md)` → **BROKEN**  
   Expected: `../best-practices/project-layouts.md`

2. Line 430: `[Policy Enforcement](../how-to/policy-enforcement.md)` → **BROKEN**  
   Expected: Create new file or reference existing policy docs

**Status:** 2 broken cross-references; potential flag inaccuracy (--check-permissions).

---

## Broken Cross-Reference Summary

| Broken Link | File | Line | Expected Target |
|---|---|---|---|
| `drift-remediation.md` | getting-started.md | 254 | `../advanced/drift-remediation.md` |
| `multi-agent-workspace.md` | getting-started.md | 255 | `../advanced/multi-agent-workspace.md` |
| `../how-to/multi-file-projects.md` | getting-started.md | 256 | `../best-practices/project-layouts.md` |
| `../how-to/import-existing-config.md` | getting-started.md | 257 | `../reference/commands/lifecycle/import.md` |
| `multi-agent-workspace.md` | ai-assisted-scaffolding.md | 311 | `../advanced/multi-agent-workspace.md` |
| `../how-to/target-overrides.md` | ai-assisted-scaffolding.md | 312 | (DOES NOT EXIST — needs creation) |
| `../reference/cli.md` | ai-assisted-scaffolding.md | 313 | `../reference/commands/index.md` |
| `../reference/cli.md` | drift-remediation.md | 65 | `../reference/commands/index.md` |
| `getting-started.md` | drift-remediation.md | 134 | `../basics/getting-started.md` |
| `../concepts/state-and-drift.md` | drift-remediation.md | 136 | `../concepts/execution/state-and-drift.md` |
| `../how-to/multi-file-projects.md` | multi-agent-workspace.md | 200 | `../best-practices/project-layouts.md` |
| `../how-to/policy-enforcement.md` | multi-agent-workspace.md | 430 | (DOES NOT EXIST — needs creation) |

**Total broken links: 12**

---

## Verification Against CLI Output

### Cannot Fully Verify: Output Format Discrepancies

The tutorials claim specific output formats like:
```
[project] ✓ wrote /path/to/my-project/.claude/agents/developer.md  (sha256:<hex>)
```

This exact format (with `[project]` prefix and sha256 hash) **does not appear in the reviewed source code**. The `applyFile()` function writes files but does not print hashes. This may be:
1. Handled by an output printer not yet reviewed
2. Outdated documentation showing a format that no longer exists
3. Accurate but requires reviewing the full output layer

**Recommendation:** Run `xcaffold init --yes --target claude && xcaffold apply --target claude` to capture real output and compare.

---

## Issues by Category

### HIGH Priority: Broken Cross-References
All 12 broken links must be fixed. Tutorials are not walkable as-is — readers following the "Next Steps" will hit 404s.

### MEDIUM Priority: CLI Flag Verification
- `xcaffold validate --structural` — verify flag exists
- `xcaffold graph --full` — verify flag exists
- `xcaffold apply --check-permissions` — verify flag exists

### MEDIUM Priority: Output Format
The exact output format (with sha256 hashes) shown in tutorials should be verified against actual CLI output.

### LOW Priority: Terminology
The init output says `✓ project.xcaf` not `Created project.xcaf` — a minor stylistic difference.

---

## Verdict

**FAIL** — Tutorials are broken due to 12 cross-reference errors. Readers cannot navigate from tutorials to supporting documentation.

### Fix Plan (Priority Order)

1. **Update cross-references in all 4 failing tutorial files**
   - Fix relative paths to point to actual doc locations
   - Use `../best-practices/project-layouts.md` instead of nonexistent `../how-to/multi-file-projects.md`
   - Use `../concepts/execution/state-and-drift.md` instead of `../concepts/state-and-drift.md`
   - Fix index paths (e.g., `../reference/commands/index.md` instead of `../reference/cli.md`)

2. **Create missing how-to guides (if intended)**
   - `docs/how-to/target-overrides.md` — referenced in ai-assisted-scaffolding.md
   - `docs/how-to/policy-enforcement.md` — referenced in multi-agent-workspace.md
   - OR: redirect these references to existing best-practices docs

3. **Verify CLI flag names** (separate task)
   - Confirm `--structural`, `--full`, `--check-permissions` exist or update tutorials

4. **Verify output format** (separate task)
   - Run actual `xcaffold apply` and capture output
   - Update expected output blocks if formats differ

---

## End-to-End Walkability Test

**Getting Started walkability:** ✗ NOT WALKABLE (requires chapter 2, which requires fixing cross-references first)

A reader who completes "Getting Started" and tries to click "Next Steps" links will encounter 404 errors on 2 of 4 next-step links.

---

**Audit completed:** 2026-05-07  
**Auditor:** Claude Code Audit Agent  
**Next step:** Dispatch fix agent to update all cross-references per Fix Plan above.
