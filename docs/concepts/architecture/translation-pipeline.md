---
title: "Cross-Platform Translation Pipeline (BIR)"
description: "How xcaffold imports and translates provider source files into a format-neutral structural graph"
---

# Cross-Platform Translation Pipeline (BIR)

When `xcaffold import --source` is used, the engine runs a semantic translation pipeline that builds the IR from provider source files:

```
Source .md files
  → bir.ImportWorkflow()         builds SemanticUnit (ID, kind, resolvedBody)
  → bir.DetectIntents()          static regex analysis (no LLM)
      IntentProcedure  → numbered steps or ## Steps section
      IntentConstraint → lines containing MUST/NEVER/ALWAYS/DO NOT/MANDATORY/REQUIRED
      IntentAutomation → lines containing // turbo annotation
  → translator.Translate()       maps intents to target primitives
      IntentProcedure  → TargetPrimitive{Kind: "skill",      ID: <id>}
      IntentConstraint → TargetPrimitive{Kind: "rule",       ID: <id>-constraints}
      IntentAutomation → TargetPrimitive{Kind: "permission", ID: <id>-permissions}
  → injectIntoConfig()           (--source mode only) inlines instructions + writes split .xcf files
```

If a `SemanticUnit` has no detected intents, it falls back to a single `skill` primitive containing the full body.

`WorkflowConfig` is lowered separately via `translator.TranslateWorkflow()` using four tiered strategies:

1. **native Antigravity** — `promote-rules-to-workflows: true` in `targets.antigravity.provider` emits a single `workflows/<name>.md` primitive
2. **`prompt-file`** — Copilot lowering emits `.github/prompts/<name>.prompt.md` with xcaffold provenance frontmatter
3. **`custom-command`** — Gemini lowering emits workflow guidance to GEMINI.md context files
4. **`rule-plus-skill`** — default for Claude and Cursor; emits one rule (with `x-xcaffold:` provenance `yaml` block) + one skill per step

Provenance markers in `rule-plus-skill` output are consumed by `bir.ReassembleWorkflow()` during round-trip import to reconstruct the original `WorkflowConfig` with step fidelity.

> **Note:** `injectIntoConfig()` is used exclusively by the `--source` cross-platform translation mode. The main `xcaffold import` command uses `WriteSplitFiles` to produce `kind: project` manifests with inline instructions and individual `.xcf` resource files under `xcf/`.
