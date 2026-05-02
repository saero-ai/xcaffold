---
title: "Skill Organization"
description: "How to structure skill resources with the correct subdirectory layout, and when to use references, scripts, assets, and examples."
---

# Skill Organization

Skills are reusable prompt packages compiled to `skills/<id>/SKILL.md` in each provider's output directory. A skill's markdown body becomes its instruction content; its frontmatter metadata controls invocation, tool access, and composition.

## Required Directory Structure

Unlike rules and policies, skills **must** be authored in a subdirectory under `xcf/skills/`. A flat file at `xcf/skills/git-workflow.xcf` causes an immediate parse error:

```
✗ skill file "git-workflow.xcf" must be in a subdirectory under xcf/skills/<id>/
```

The correct structure is:

```
xcf/skills/
└── git-workflow/     ← subdirectory named after the skill ID
    └── skill.xcf     ← file must be named skill.xcf
```

The validator enforces the filename `skill.xcf` exactly — `git-workflow.xcf` or any other name at the skill root will fail validation. The skill's `name:` field in the frontmatter is the resource ID, independent of the directory name.

## Graduated Complexity

Start with the single `.xcf` file. Add supporting subdirectories only when the skill genuinely needs them.

### Stage 1 — Instruction only

A skill that provides inline guidance needs only the `.xcf` file:

```
xcf/skills/
└── git-workflow/
    └── skill.xcf
```

### Stage 2 — With supporting files

When the skill needs reference documents, executable scripts, output templates, or worked examples, graduate to the canonical layout:

```
xcf/skills/
└── code-review/
    ├── skill.xcf             ← the orchestrator
    ├── references/
    │   └── team-standards.md ← read-only background knowledge
    ├── scripts/
    │   └── check-lint.sh     ← executable invoked by the AI
    ├── assets/
    │   └── pr-template.json  ← templates the AI fills out or copies
    └── examples/
        └── good-review.md    ← "golden rule" demonstration
```

> [!NOTE]
> When **authoring** skill directories, the parser validates subdirectory names at parse time. Only `references/`, `scripts/`, `assets/`, and `examples/` are recognized as canonical subdirectories. Maximum depth is 1 level — nesting inside `references/references/` is not supported.
>
> During **import**, non-standard subdirectories found in provider skill directories are preserved under `xcf/skills/<id>/<subdir>/` alongside canonical subdirectories, rather than being rejected.

## Choosing the Right Subdirectory

| Question | Use |
|---|---|
| Does the AI **read** this file to inform decisions? | `references/` |
| Does the AI **copy or fill out** this file to generate output? | `assets/` |
| Does this file show a **perfectly finished example** for the AI to replicate? | `examples/` |
| Is this an **executable script** the AI must invoke? | `scripts/` |

### `references/`

Background knowledge the AI needs to execute the skill correctly — API specs, coding standards, domain glossaries. Files here are read-only contextual facts. The `.xcf` orchestrator stays clean; heavy documentation lives separately and is copied to `skills/<id>/references/` at compile time.

Rather than embedding a 10-page API specification in the skill body, instruct the AI in the body to read from `./references/api-spec.md`. The file is always present after `xcaffold apply`.

### `scripts/`

Executable helpers the AI must invoke — linting wrappers, build scripts, data migration tools. Instead of asking the LLM to "figure out how to run the linter", be explicit in the skill body: "Execute `./scripts/check-lint.sh` to determine layout validity." Scripts are copied to `skills/<id>/scripts/` and should be committed with execute permissions (`chmod +x`).

### `assets/`

Output artifact templates — JSON schemas, route stubs, boilerplate files — that the AI copies or fills out during execution. The distinction from `references/` is that assets are *transformed or produced into the codebase*, not just read. A `route-stub.ts` asset is more useful than a general instruction to "create a route file."

### `examples/`

Golden-rule demonstrations of correct output. Two precise `good` vs `bad` examples significantly outperform twenty mediocre ones. Use `examples/` sparingly — provider behavior varies:

- **Claude**: flattened directly into `skills/<id>/` (no subdirectory) — Claude auto-loads all `.md` files from the skill root
- **Cursor, Antigravity, Copilot**: copied to `skills/<id>/examples/` as its own subdirectory
- **Gemini**: compiled into `skills/<id>/references/` (examples have no native concept; they are treated as reference material)

## What xcaffold Compiles Per Provider

Every provider receives a `skills/<id>/SKILL.md`. The tables below show exactly what each renderer writes — verified line-by-line from the renderer source.

> [!NOTE]
> The skill **body** (the markdown prose below the frontmatter delimiter) is always included verbatim in every provider's SKILL.md. It is the universal instruction layer regardless of which frontmatter fields are supported.

### SKILL.md Frontmatter Fields

| Field | Claude | Cursor | Gemini | Copilot | Antigravity |
|---|---|---|---|---|---|
| `name` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `description` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `when_to_use`¹ | ✅ | ❌ | ❌ ² | ❌ ² | ✅ |
| `allowed-tools` | ✅ | ❌ | ❌ ² | ✅ | ❌ |
| `license` | ✅ | ❌ | ❌ | ✅ | ❌ |
| `disable-model-invocation` | ✅ | ❌ | ❌ ² | ❌ ² | ❌ |
| `user-invocable` | ✅ | ❌ | ❌ | ❌ ² | ❌ |
| `argument-hint` | ✅ | ❌ | ❌ | ❌ ² | ❌ |

**Legend:**
- ✅ Written to SKILL.md frontmatter
- ❌ Field not written; stays in `xcf/` source only
- ❌ ² Field not written **and** xcaffold prints a warning to stderr (a fidelity note) telling you the field was dropped for this provider

**Notes:**

1. `when_to_use` uses an underscore in the output file — not a hyphen. Claude Code's native SKILL.md schema requires exactly `when_to_use:`. xcaffold handles this automatically; you author it in `.xcf` as `when-to-use:` (kebab-case) and the Claude renderer converts it on output.

2. For fields marked ❌ ², xcaffold prints a line like `[WARN] skill "my-skill": field "when-to-use" dropped for gemini — no native equivalent` to stderr during `xcaffold apply`. Compilation still succeeds and SKILL.md is still written.

### Supporting Subdirectory Handling

| Subdirectory | Claude | Cursor | Gemini | Copilot | Antigravity |
|---|---|---|---|---|---|
| `references/` | ✅ `references/` | ✅ `references/` | ✅ `references/`³ | ✅ `references/` | ✅ `references/` |
| `scripts/` | ✅ `scripts/` | ✅ `scripts/` | ✅ `scripts/` | ✅ `scripts/` | ✅ `scripts/` |
| `assets/` | ✅ `assets/` | ✅ `assets/` | ✅ `assets/` | ✅ `assets/` | ✅ `assets/` |
| `examples/` | → skill root⁴ | → `references/`⁵ | → `references/`⁶ | ✅ `examples/` | ✅ `examples/` |

**Notes:**

3. Gemini demotes subdir compilation errors to warnings rather than hard failures — the rest of the skill still compiles if a referenced file is missing.
4. Claude flattens `examples/` files directly into `skills/<id>/` (no subdirectory). Claude auto-loads all `.md` files from the skill root, so a separate subdirectory is not needed.
5. Cursor routes `examples/` into `references/` — it has no distinct examples concept.
6. Gemini also routes `examples/` into `references/` for the same reason.

### Output Paths

| Provider | Output directory |
|---|---|
| Claude Code | `.claude/skills/<id>/SKILL.md` |
| Cursor | `.cursor/skills/<id>/SKILL.md` |
| Gemini CLI | `.gemini/skills/<id>/SKILL.md` |
| GitHub Copilot | `.github/skills/<id>/SKILL.md`⁷ |
| Antigravity | `.agents/skills/<id>/SKILL.md` |

7. If a `.claude/` directory already exists in your project, the Copilot renderer skips writing skills entirely and emits an info note: GitHub Copilot natively loads `.claude/skills/` automatically, so re-generating them in `.github/` would be redundant.

