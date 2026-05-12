---
title: "Translation Pipeline"
description: "How xcaffold lowers WorkflowConfig resources into provider-native primitives via the internal/translator package"
---

# Translation Pipeline

The translation pipeline converts a `WorkflowConfig` resource — parsed from a `kind: workflow` `.xcaf` file — into one or more provider-native output artifacts called **target primitives**. This work is performed entirely by the `internal/translator` package. The renderer invokes `TranslateWorkflow` (via `shared.LowerWorkflows`) after parsing and before writing files to disk.

---

## Core Types

`internal/translator/rules.go` defines two types shared across all lowering strategies.

**`TargetPrimitive`** is a single output artifact:

```go
type TargetPrimitive struct {
    Kind    string // "skill", "rule", "permission", "workflow", "prompt-file", "custom-command"
    ID      string
    Body    string // legacy field kept for backward compatibility
    Content string // full text of the artifact
    Path    string // set only for primitives that carry their own output path
}
```

`Path` is non-empty only for `prompt-file` and `custom-command` primitives, which determine their own output paths (`.github/prompts/<name>.prompt.md` and `.gemini/commands/<name>.md` respectively). For all other kinds, the renderer determines the path.

**`TranslationResult`** wraps a slice of primitives when results are aggregated by callers that iterate over multiple resources. `TranslateWorkflow` itself returns `([]TargetPrimitive, []renderer.FidelityNote)` directly.

---

## TranslateWorkflow

`TranslateWorkflow(wf *ast.WorkflowConfig, target string) ([]TargetPrimitive, []renderer.FidelityNote)` is the single entry point for workflow translation (`internal/translator/workflow.go`).

It determines the lowering strategy for the named target in this priority order:

1. If the target override contains `promote-rules-to-workflows: true` → **native workflow** lowering.
2. If the target override sets `lowering-strategy: prompt-file` → **prompt-file** lowering.
3. If the target override sets `lowering-strategy: custom-command` → **custom-command** lowering.
4. Otherwise → **rule-plus-skill** lowering (the default for all targets).

Strategy lookup uses the `loweringStrategy` helper, which reads `wf.Targets[target].Provider["lowering-strategy"]` and returns an empty string when the key is absent.

---

## Lowering Strategies

### rule-plus-skill (default)

`lowerRulePlusSkill` is the default strategy applied when a target has no override or when `lowering-strategy` is explicitly set to `"rule-plus-skill"`.

It emits one rule primitive and one skill primitive per workflow step:

**Rule primitive** (`Kind: "rule"`, `ID: "<name>-workflow"`)  
Contains an `x-xcaffold:` provenance block encoded as an inline YAML fence:

```yaml
x-xcaffold:
  compiled-from: workflow
  workflow-name: <name>
  api-version: workflow/v1
  step-order: [<step-names…>]
  step-skills:
    - <name>-01-<step-slug>
    - <name>-02-<step-slug>
    …
```

Followed by a plain-text invocation instruction listing skill IDs in order.

**Skill primitives** (`Kind: "skill"`)  
One per step. The `ID` is produced by `stepSkillID(workflowName, i, stepName)`, which generates `<workflow-name>-<NN zero-padded>-<step-name-slugified>`. For example, step 0 named `"analyze"` in workflow `"code-review"` produces `code-review-01-analyze`.

The skill `Content` is the step's `Body` field, falling back to `Description` when `Body` is empty.

This strategy emits a `renderer.FidelityNote` at `LevelWarning` with code `CodeWorkflowLoweredToRulePlusSkill`, informing the caller that the target has no native workflow primitive.

### native workflow

`lowerNativeWorkflow` is selected when `wf.Targets[target].Provider["promote-rules-to-workflows"]` is `true`. Any provider can opt in via a target override.

It emits a single primitive (`Kind: "workflow"`, `ID: "<name>"`) whose `Content` concatenates each step body under a `## <step-name>` heading.

This strategy emits a `renderer.FidelityNote` at `LevelInfo` with code `CodeWorkflowLoweredToNative`.

### prompt-file

`lowerPromptFile` is selected by setting `lowering-strategy: prompt-file` on the target override. It is intended for targets that consume GitHub Copilot prompt files.

It emits a single primitive:

- `Kind: "prompt-file"`
- `ID: "<name>"`
- `Path: ".github/prompts/<name>.prompt.md"`
- `Content`: a frontmatter block with `mode: agent` and an `x-xcaffold:` provenance section, followed by all step bodies concatenated.

This strategy emits a `renderer.FidelityNote` at `LevelInfo` with code `CodeWorkflowLoweredToPromptFile`.

### custom-command

`lowerCustomCommand` is selected by setting `lowering-strategy: custom-command` on the target override. It is intended for targets that consume Gemini CLI command files.

It emits a single primitive:

- `Kind: "custom-command"`
- `ID: "<name>"`
- `Path: ".gemini/commands/<name>.md"`
- `Content`: all step bodies concatenated.

This strategy emits a `renderer.FidelityNote` at `LevelInfo` with code `CodeWorkflowLoweredToCustomCommand`.

---

## Provenance and Round-Trip Fidelity

The `rule-plus-skill` strategy embeds an `x-xcaffold:` provenance block in the emitted rule. The block records the workflow name, API version, step order, and the list of step-skill IDs. This marker is written as a fenced YAML block in the rule body, making it readable by both the target platform and any future tooling that parses compiled output.

---

## Fidelity Notes

Every lowering function returns a `[]renderer.FidelityNote` alongside the primitives. The compiler collects these notes and passes them to `printFidelityNotes` in `cmd/xcaffold/fidelity.go`, which writes them to the output writer during `xcaffold apply`. Warning-level notes indicate a lossy lowering; info-level notes are informational only. Error-level notes from other pipeline stages block compilation.

---

## Related

- [Architecture](overview.md) — full internal package map
- [Workflows kind reference](../../reference/kinds/provider/workflow.md) — `WorkflowConfig` field reference and `.xcaf` authoring guide
- [Multi-Target Rendering](multi-target-rendering.md) — how fidelity notes surface per-target differences
