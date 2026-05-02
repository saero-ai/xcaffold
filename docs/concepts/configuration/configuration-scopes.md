---
title: "Configuration Scopes"
description: "Understanding configuration contexts, scopes, implicit global inheritance, and boundaries"
---

# Configuration Scopes

xcaffold defines three compilation scopes: **global scope** (`~/.xcaffold/`), **project scope** (the directory containing `project.xcf`), and **blueprint scope** (a named subset of project resources). Each scope compiles independently and produces its own state file. During project compilation, xcaffold loads global resources as a base layer and merges project-scope resources over them by ID before rendering output.

## Three Compilation Scopes

| Scope | Flag | What Compiles | State File Path |
| :--- | :--- | :--- | :--- |
| Global | `--global` | User-wide personal config (`~/.xcaffold/global.xcf`) | `~/.xcaffold/<state>.xcf.state` |
| Project | (default) | All resources in `xcf/` | `.xcaffold/project.xcf.state` |
| Blueprint | `--blueprint <name>` | Named resource subset selected by a `kind: blueprint` file | `.xcaffold/<blueprint-name>.xcf.state` |

> **Note:** Scopes are mutually exclusive. One context per compiled invocation.

---

## Multi-Kind Project Scope

A project scope is defined by exactly one `kind: project` document. This document declares the project name, targets, and resource references (bare name lists of agents, skills, rules, etc.). The actual resource definitions live in separate `.xcf` files under `xcf/` subdirectories.

`ParseDirectory` discovers all `.xcf` files recursively from the project root, parses each document, and routes it by `kind:`:

| Kind | Role |
|---|---|
| `project` | Project manifest — name, targets, child resource refs. Exactly 1 required. |
| `hooks` | Standalone hooks with `events:` wrapper |
| `settings` | Standalone settings |
| `global` | Global-scope configuration |

All discovered resource documents are merged into a single `ResourceScope` with strict deduplication. If the same resource ID appears in two files, parsing fails immediately with a duplicate ID error.

### Global Scope

Resources in `~/.xcaffold/` define the global scope. These are loaded via `loadGlobalBase` and provide system-wide defaults. Project-scope resources override global resources by ID during compilation.

---

## Configuration Classification

Xcaffold acts as a deterministic compiler that sits between an agnostic declarative schema (`project.xcf`) and the provider-specific agent constraints required by the LLM target (e.g. Claude, Cursor, GitHub Copilot).

To satisfy this architectural boundary, the YAML configuration schema maintains a strict, philosophical separation between two distinct classes of structures:

1. **Xcaffold-Specific Configurations:** The mechanics of the compilation engine.
2. **Provider-Specific Agent Configurations:** The native behaviors of the LLM itself.

This document serves as the canonical map separating procedural compiler rules from generative intelligent personas.

---

## Xcaffold-Specific Configurations

These configuration blocks control how Xcaffold evaluates paths, manages execution policies, configures local sandbox bounds, hooks build lifecycle events, and enforces compile-time restrictions.

**Crucially, nothing defined in these configurations is natively loaded into the AI's contextual window.** They represent classical, deterministic computing mechanics that execute surrounding the LLM prompt, without ever polluting it.

### `global` (Global Scope)
System-wide declarations residing outside your project repository (`~/.xcaffold/global.xcf`). Evaluated under the overarching workspace scope to ensure consistent inheritance of security boundaries across all local projects.

### `project` (Project Scope)
The authoritative root bounding the local workspace (`project:` object). It registers metadata signatures tracked in state files and limits the execution blast radius to localized overrides and dependencies.

### `policy` (Evaluation Engine)
AST tree-walking rules (`policies:`) that statically enforce or forbid specific field patterns within the Xcaffold configuration. Used to deny unsafe shell execution (`deny`) or mandate strict tool inclusions (`require`). This allows organizations to police agentic permissions purely at compilation time before passing control to the AI.

### `template` (Structural Bootstrapping)
File structures and boilerplate primitives orchestrating initial project construction. Currently evaluated natively by initializing workflows without a persistent compilation AST structure.

---

## Provider-Specific Agent Configurations

These configurations describe the behavioral identity, situational awareness, environmental restrictions, and explicit constraints ultimately passed to the underlying Large Language Model.

Unlike deterministic compiler settings, these definitions are passed down and compiled directly into the provider-native format (e.g., `.claude/agents/`, `.cursor/rules/`, `.github/instructions/`) to explicitly steer LLM cognition.

### `agent` (Personas)
The identity structure mapping the LLM system prompt. Modulates conversational alignment, defines memory tracking layers, limits inference turn constraints, and orchestrates model routing.

### `skill` (Specialization)
Action-oriented execution blocks instructing the AI on exactly how to solve a singular problem algorithmically. Bound dynamically via invocation tools or `@-import` directives to assemble complex abilities at runtime.

### `rule` (Behavioral Guidelines)
Targeted contextual boundaries that constrain LLM autonomy when navigating paths or domains within the active codebase. Limits token pollution by conditionally injecting context strictly based on real-time code proximity (via glob match mechanisms).

### `workflow` (Procedural Automations)
Complex sequential tasks decomposed into rigidly ordered pipelines (lowered natively for Antigravity, translated using internal tool chains for Claude/Cursor/Copilot). Allows deterministic human processes to govern LLM intuition across arbitrary checkpoint chains.

### `mcp` (Model Context Protocols)
Definitions routing real-time external observability integrations, database introspection capabilities, or sandboxed browser execution servers strictly required for LLM semantic decision-making. 

### `memory` (Cognitive Continuity)
Cross-iteration continuity anchors tracking architectural constraints across separate agent sessions without human recalibration. Ensures deep longitudinal history is snapshotted to preserve long-term agent context.

### `settings` (Environment Bounds)
Execution bindings explicitly managed by target platform IDEs (such as Claude Code's `settings.json`). Configures telemetry, IDE conventions, allowed domains, network proxy connections, and CLI styling.

### `hooks` (Lifecycle Shell Dispatch)
Event handlers triggered sequentially when navigating execution stages. Intercepts specific phases (like `PreToolUse` or `SessionStart`) to dispatch shell commands or bash automation completely independent from the core agent contextual workflow.

## Implicit Global Inheritance

xcaffold implicitly evaluates constraints from both contexts at parse-time. Whenever xcaffold evaluates a project configuration, it automatically scans and parses your global directory (`~/.xcaffold/`) as a baseline template behind the scenes.

This merge produces two notable behaviors at the parser level:

1. **Cross-Reference Safety**: A project-scoped agent can `instructions-file:` reference a skill defined only in the global repository. The compiler parses both scopes before resolving references, so the cross-scope reference resolves correctly rather than producing a parse failure.
2. **Holistic Topology**: `xcaffold graph` produces a combined view of the full agent topology — global resources plus project resources — reflecting what the target platform will load at runtime.

### The Compiler Hard Boundary

Implicit global inheritance deliberately extends *only* to the parser boundary limitation.

When you execute an `xcaffold apply` inside your project, the compilation runtime securely **strips** global resources from the aggregated AST before physically generating files.

Global resources are intentionally **NOT** written to the local project output directories (like `.claude/` or `.cursor/`). Because Claude Code, Cursor, Gemini CLI, and GitHub Copilot all autonomously combine the user's global environment at inference time, duplicating physical files into the local workspace would cause duplicate instruction injection and pollute git history unnecessarily. (Antigravity's scope loading behavior is not documented.)

To synchronize global changes to disc, execute a global-scoped apply independent of your project:

```bash
xcaffold apply --global
```

## Override Mechanics

When a configuration hierarchy is resolved (whether through implicit global resolution or explicit `extends:` directives), xcaffold's internal parser applies the overriding file on top of the base file.

### Provider Override Files

Provider-specific overrides use `<kind>.<provider>.xcf` files alongside base resources:

```
xcf/agents/developer/
  agent.xcf                # base definition (universal)
  agent.claude.xcf         # Claude-specific overrides
  agent.gemini.xcf         # Gemini-specific overrides
```

Override merge uses Terraform-style semantics:
- Scalars and lists: override REPLACES base
- Maps: DEEP MERGE (override keys win)
- Body: override REPLACES if present, INHERITS if absent

**Example:** An agent base file defines the full instructions body. The Claude override adds only a model and hooks:

```
xcf/agents/developer/
  agent.xcf                # base: full instructions + shared fields
  agent.claude.xcf         # override: model: opus, hooks: {...} (no body — inherits from base)
```

Override files that share the same body as the base should omit the body entirely. The compiler inherits the base body automatically during merge.

The compiler applies overrides between `ApplyBlueprint()` and `DiscoverAgentMemory()`.

### Directory Scan Deduplication (File vs File in Same Scope)

When discovering multiple `.xcf` files recursively within the *same* scope (e.g., across `xcf/agents/*.xcf`), resources are safely aggregated.

| Resource | Merge behavior |
|----------|---------------|
| `agents` | Additive. Duplicate ID = hard error. |
| `skills` | Additive. Duplicate ID = hard error. |
| `rules` | Additive. Duplicate ID = hard error. |
| `mcp` | Additive. Duplicate ID = hard error. |
| `workflows` | Additive. Duplicate ID = hard error. |
| `hooks` | Additive per event. Handlers from all files are merged, not replaced. |
| `version` | First file that declares it wins. Conflicting values = hard error. |
| `project` | First file that declares `project.name` wins. Conflicting values = hard error. |
| `extends` | First file that declares it wins. Conflicting values = hard error. |
| `settings` | Last file wins (full struct overwrite — not field-by-field merge). |
| `test` | Last file that declares a non-empty block wins. |

The alphabetical sort order of file names determines "first" and "last" for `version`, `project`, and `settings`. Keep `settings` in a single file to avoid unexpected overwrite behavior.

### Cross-Scope Merge Behavior (Parent vs Child)

| Resource type | Merge rule |
|---|---|
| `agents:` | Child entry replaces base entry per ID. IDs present only in the base are kept. |
| `skills:` | Child entry replaces base entry per ID. IDs present only in the base are kept. |
| `rules:` | Child entry replaces base entry per ID. IDs present only in the base are kept. |
| `mcp:` | Child entry replaces base entry per ID. IDs present only in the base are kept. |
| `workflows:` | Child entry replaces base entry per ID. IDs present only in the base are kept. |
| `hooks:` | Additive. Both base and child handlers are kept. Child handlers are appended to base handlers for each event. |
| `project.test:` | Field-by-field overlay within `ProjectConfig`. `cli_path`, `cli-path`, and `judge-model` are replaced individually only when the child sets a non-empty value. |
| `settings:` | Last file in the directory wins (single-settings-file convention). Inherited and merged via `extends:`. |
| `project.local:` | Machine-local settings override within `ProjectConfig`. Not inherited via `extends:`. Compiles to `settings.local.json`. |

### Compiler Scope Merge

When a project config defines resources at both root level (global scope, from `extends:` or implicit global loading) and inside the `project:` block (workspace scope), the compiler merges them before rendering. Workspace resources override global resources by ID:

| Resource level | Source | Priority |
|---|---|---|
| Root-level `agents:`, `skills:`, etc. | Global or inherited | Lower (base) |
| `project.agents:`, `project.skills:`, etc. | Workspace-specific | Higher (override) |

> **Format note:** In `kind: project` format, workspace-scoped resources are defined as bare name lists at the top level (e.g. `agents:` lists agent names, with definitions in separate `kind: agent` files).

After merging, inherited resources (those originating from `extends:` chains) are stripped from the compilation output to prevent duplication.

### Target Filtering

Resources with a `targets:` field are compiled only for listed providers:
- `targets:` absent → universal (compiled for all targets)
- `targets:` present → restricted (compiled only for listed providers)

Resources excluded by target filtering produce a fidelity warning.

### Additive Hook Pipelines

Unlike static models (agents, rules), runtime execution lifecycle hooks accumulate across the inheritance chain securely. If the global base defines a `PreToolUse` handler and the local project also defines a `PreToolUse` handler, both run chronologically. The child's project-specific handler fires immediately after the overarching global policy baseline.

```yaml
# ~/.xcaffold/global.xcf
kind: global
version: "1.0"
hooks:
  PreToolUse:
    - hooks:
        - type: command
          command: "echo pre-tool-use from global baseline"

# ./hooks.xcf (kind: hooks — standalone format, recommended)
kind: hooks
version: "1.0"
events:
  PreToolUse:
    - hooks:
        - type: command
          command: "echo pre-tool-use from project override"
```

> Hooks are declared in a standalone `kind: hooks` file with an `events:` wrapper. This is the only supported format for hook declarations.

The compiled evaluation contains both handlers in order: global base, then project override.

## Circular Dependency Prevention

xcaffold prevents endless configuration loops by tracking file dependencies during parsing.

An explicit error terminates compilation instantly if:
- Implicit `global` resolution references itself
- Graph trace detects topological loops during `extends: /path/to/base.xcf` resolution

Example compiler termination:
```
circular extends detected: "/abs/path/to/base.xcf"
```

---

## When This Matters

- **A project agent references a globally-defined skill** — because xcaffold loads both scopes before resolving cross-references, the reference resolves cleanly at parse time; no special declaration is required in the project manifest.
- **Running `xcaffold apply --global` vs `xcaffold apply`** — the two commands compile independent scopes and write independent state files; a change to one scope never invalidates the other's drift state.
- **Debugging a duplicate ID error across files** — strict deduplication operates within each scope independently; the same ID appearing in both global and project scopes is valid (project overrides global), but the same ID in two files within the same scope is an error.
- **Understanding why global resources do not appear in the project output directory after a project apply** — global resources are stripped from project-scope output before writing, because Claude Code, Cursor, Gemini CLI, and GitHub Copilot all load global and project configurations separately at runtime. (Antigravity's scope loading behavior is not documented.)

---

## Related

- [Getting Started](../tutorials/getting-started.md) — walkthrough of project-scope initialization and first apply
- [Multi-Agent Workspace](../tutorials/multi-agent-workspace.md) — example project using both global and project-scoped agents
- [Split Configs](../how-to/multi-file-projects.md) — how to organize a project across multiple `.xcf` files
- [Architecture](architecture.md) — the full compilation pipeline including scope merge mechanics
- [Schema Reference](../reference/schema.md) — `kind: global`, `kind: project`, and `extends:` field definitions
