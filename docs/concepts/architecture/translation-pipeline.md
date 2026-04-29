---
title: "Internal: BIR Architecture"
description: "Internal compiler intermediate representation for semantic analysis and cross-platform optimization"
---

# Internal: BIR Architecture

> **Internal architecture.** The BIR pipeline is used internally by the compiler for semantic analysis and optimization. It is not part of the user-facing `import` command. For import documentation, see [xcaffold import](../../reference/commands/lifecycle/import.md).

The compiler's semantic translation pipeline builds the IR from provider source files:

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
  → injectIntoConfig()           inlines instructions + writes split .xcf files
```

If a `SemanticUnit` has no detected intents, it falls back to a single `skill` primitive containing the full body.

`WorkflowConfig` is lowered separately via `translator.TranslateWorkflow()` using four tiered strategies:

1. **native Antigravity** — `promote-rules-to-workflows: true` in `targets.antigravity.provider` emits a single `workflows/<name>.md` primitive
2. **`prompt-file`** — Copilot lowering emits `.github/prompts/<name>.prompt.md` with xcaffold provenance frontmatter
3. **`custom-command`** — Gemini lowering emits workflow guidance to GEMINI.md context files
4. **`rule-plus-skill`** — default for Claude and Cursor; emits one rule (with `x-xcaffold:` provenance `yaml` block) + one skill per step

Provenance markers in `rule-plus-skill` output are consumed by `bir.ReassembleWorkflow()` during round-trip import to reconstruct the original `WorkflowConfig` with step fidelity.

> **Note:** The main `xcaffold import` command uses `WriteSplitFiles` to produce `kind: project` manifests with inline instructions and individual `.xcf` resource files under `xcf/`. The BIR pipeline operates as an internal compiler stage.
