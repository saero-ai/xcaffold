---
title: "Provider Architecture"
description: "Symmetric ProviderImporter and TargetRenderer interfaces, the CapabilitySet model, and the import/render pipeline"
---

# Provider Architecture

xcaffold uses a symmetric import/render architecture for each supported provider. Every provider — Claude Code, Cursor, Gemini CLI, GitHub Copilot, Antigravity, and Codex (Preview) — has two components that mirror each other:

- **ProviderImporter** (`internal/importer/<provider>/`) reads the provider's native directory structure and populates the AST
- **TargetRenderer** (`internal/renderer/<provider>/`) compiles the AST into the provider's native directory structure

The AST (`ast.XcaffoldConfig`) sits between these two halves as a shared intermediate representation. An import from one provider and a render to another is a cross-provider translation; an import and render to the same provider is a round-trip that should preserve all classified data.

All six current providers are **harness providers** — AI coding tools that read static configuration files at runtime. The provider architecture also supports declarative agent SDK frameworks as future expansion targets, where xcaffold would scaffold starting configurations rather than continuously compile managed state.

---

## Pipeline Overview

```
Provider Directory (.claude/, .cursor/, .gemini/, .github/, .agents/, .codex/)
    │
    ▼
ProviderImporter.Import(dir, config)
    ├── Classify(path) → Kind + Layout
    ├── Extract(path, data) → populate AST
    ├── Shared helpers: ParseFrontmatter, MatchGlob, ReadFile
    └── Unclassified files → ProviderExtras
    │
    ▼
ast.XcaffoldConfig (shared IR)
    │
    ▼
renderer.Orchestrate(TargetRenderer, config, baseDir)
    ├── Capabilities() → CapabilitySet (declares supported resource kinds)
    ├── Per-kind dispatch: CompileAgents, CompileSkills, CompileRules, ...
    ├── Unsupported kinds → RENDERER_KIND_UNSUPPORTED FidelityNote
    └── Finalize() → post-processing pass (path normalization, key merging)
    │
    ▼
Provider Directory (output)
```

The pipeline is intentionally asymmetric in one respect: importers populate a single shared `ast.XcaffoldConfig`, but renderers receive that struct and produce a `map[string]string` file map. The importer writes to a rich typed struct; the renderer reads from it and emits flat files. This asymmetry reflects the compilation model — the AST is the authority, and generated files are a derived view.

---

## The Importer Side

### ProviderImporter Interface

```go
type ProviderImporter interface {
    Provider() string
    InputDir() string
    Classify(rel string, isDir bool) (Kind, Layout)
    Extract(rel string, data []byte, config *ast.XcaffoldConfig) error
    Import(dir string, config *ast.XcaffoldConfig) error
}
```

| Method | Purpose |
|---|---|
| `Provider()` | Returns the canonical name (e.g. `"claude"`, `"cursor"`) |
| `InputDir()` | Returns the directory the provider reads from (e.g. `".claude"`, `".github"`) |
| `Classify()` | Maps a relative file path to a `Kind` and `Layout` using the provider's pattern table |
| `Extract()` | Parses a single file's content and populates the appropriate AST field |
| `Import()` | Orchestrates the full directory walk: iterates files, classifies, extracts, collects warnings |

As of the import pipeline refactoring, all import code paths use the `ProviderImporter` interface. The single-directory path (`importScope`) and multi-directory merge path (`mergeImportDirs`) both call `ProviderImporter.Import()` per detected provider. Legacy per-provider extraction functions that previously bypassed the interface have been removed.

### Kind Classification

Each importer owns a `[]KindMapping` table that maps file patterns to AST resource kinds. When a file is encountered during the directory walk, the importer checks it against its patterns in order:

| Kind | Description |
|---|---|
| `agent` | Agent system prompt with optional YAML frontmatter |
| `skill` | Skill definition with optional `references/`, `scripts/`, `assets/`, `examples/` subdirectories |
| `rule` | Rule file with activation metadata |
| `hook` | Shell hook declarations (often embedded in a settings JSON file) |
| `mcp` | MCP server configuration |
| `settings` | Provider-specific settings |
| `memory` | Agent memory entries |
| `workflow` | Workflow definitions |
| `""` (unknown) | No pattern matched — stored in `ProviderExtras` |

### Layout Types

The `Layout` enum describes how a provider stores resources on disk:

| Layout | Example | Description |
|---|---|---|
| `FlatFile` | `agents/*.md` | One file per resource |
| `DirectoryPerEntry` | `skills/*/SKILL.md` | One subdirectory per resource with a canonical file |
| `StandaloneJSON` | `mcp.json` | Single JSON file holding all resources of one kind |
| `EmbeddedJSONKey` | `settings.json#hooks` | Key inside a container JSON file |
| `InlineInParent` | Agent fields inside a workflow | Embedded inside another resource definition |

Files that match no pattern go to `ProviderExtras` — a genuinely-unknown catchall that preserves round-trip fidelity for same-provider workflows. These extras are written back to disk during `xcaffold apply` when the target matches the source provider.

### Shared Importer Helpers

Five helpers are shared across all provider importers in `internal/importer/helpers.go` to eliminate duplication:

| Helper | Purpose |
|---|---|
| `ParseFrontmatter` | Splits YAML frontmatter from markdown body; returns error on malformed YAML |
| `ParseFrontmatterLenient` | Same split, but returns body with zero-value metadata on malformed YAML (used by Claude importer for user-edited files) |
| `MatchGlob` | Matches relative paths against glob patterns with `*` (single segment) and `**` (any depth) support |
| `ReadFile` | Thin wrapper over `os.ReadFile` for consistent import paths |
| `AppendUnique` | Deduplicates string slices during resource collection |

### Provider Detection

`DetectProviders()` scans a root directory for known provider directories and returns importers for those that exist on disk. This is used by `xcaffold import` to auto-detect which providers are present without requiring the user to specify them.

---

## The Renderer Side

### TargetRenderer Interface

```go
type TargetRenderer interface {
    Target() string
    OutputDir() string
    Capabilities() CapabilitySet

    CompileAgents(agents map[string]AgentConfig, baseDir string) (map[string]string, []FidelityNote, error)
    CompileSkills(skills map[string]SkillConfig, baseDir string) (map[string]string, []FidelityNote, error)
    CompileRules(rules map[string]RuleConfig, baseDir string) (map[string]string, []FidelityNote, error)
    CompileWorkflows(workflows map[string]WorkflowConfig, baseDir string) (map[string]string, []FidelityNote, error)
    CompileHooks(hooks HookConfig, baseDir string) (map[string]string, []FidelityNote, error)
    CompileSettings(settings SettingsConfig) (map[string]string, []FidelityNote, error)
    CompileMCP(servers map[string]MCPConfig) (map[string]string, []FidelityNote, error)
    CompileProjectInstructions(project *ProjectConfig, baseDir string) (map[string]string, []FidelityNote, error)

    Finalize(files map[string]string) (map[string]string, []FidelityNote, error)
}
```

| Method | Purpose |
|---|---|
| `Target()` | Returns the canonical name (e.g. `"claude"`, `"copilot"`) |
| `OutputDir()` | Returns the output directory (e.g. `".claude"`, `".github"`) |
| `Capabilities()` | Declares which resource kinds this renderer supports |
| `Compile*` methods | Each translates one resource kind from AST to file map entries |
| `Finalize()` | Post-processing pass after all resources are compiled (key merging, path normalization) |

Every `Compile*` method returns three values: a `map[string]string` of relative file paths to content, a slice of `FidelityNote` structs describing any information loss, and an error. The file maps from all methods are merged by the orchestrator into a single output.

### CapabilitySet

Each renderer declares its supported resource kinds via a `CapabilitySet` struct:

```go
type RuleEncodingCapabilities struct {
    // Description: "frontmatter" | "prose" | "omit"
    Description string
    // Activation: "frontmatter" | "omit"
    Activation  string
}

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
    RuleEncoding        RuleEncodingCapabilities
    // AgentNativeToolsOnly declares that this provider's native tool vocabulary
    // IS the Claude Core tool set. Only Claude sets this true.
    AgentNativeToolsOnly bool
}
```

The `CapabilitySet` serves two purposes. First, the orchestrator uses the boolean fields to decide whether to call the corresponding `Compile*` method or emit a `RENDERER_KIND_UNSUPPORTED` fidelity note. Second, the extended fields (`SkillArtifactDirs`, `RuleActivations`, `RuleEncoding`, `AgentNativeToolsOnly`) provide structured metadata about the renderer's feature support for tooling and tests.

The current capability declarations:

| Field | Claude | Cursor | Gemini | Copilot | Antigravity | Codex |
|---|---|---|---|---|---|---|
| Agents | yes | yes | yes | yes | yes | yes |
| Skills | yes | yes | yes | yes | yes | yes |
| Rules | yes | yes | yes | yes | yes | no |
| Workflows | yes¹ | yes¹ | yes¹ | yes¹ | yes | no² |
| Hooks | yes | yes | yes | yes | no | yes |
| Settings | yes | yes | yes | yes | yes | no |
| MCP | yes | yes | yes | yes | yes | yes |
| Memory | yes | no | no | no | no | no |
| ProjectInstructions | yes | yes | yes | yes | yes | yes |
| AgentNativeToolsOnly | yes | no | no | no | no | no |
| RuleEncoding.Description | frontmatter | frontmatter | prose | frontmatter | frontmatter | — |
| RuleEncoding.Activation | frontmatter | frontmatter | omit | frontmatter | frontmatter | — |

¹ **Lowered workflows.** Claude, Cursor, Gemini, and Copilot have no native workflow primitive. Their `CompileWorkflows()` implementations lower each workflow to a combination of rules and skills — the rule provides the activation trigger (when to invoke the workflow) and the skill carries the workflow body (the steps to execute). The fidelity note `WORKFLOW_LOWERED_TO_RULE_PLUS_SKILL` is emitted to signal the structural change. Antigravity is the only provider with a first-class `workflows/*.md` format that preserves the workflow structure directly.

² **Codex workflow gap.** Codex supports skills but not rules. Since workflow lowering requires both primitives — a rule for activation and a skill for the body — Codex cannot receive the complete lowered form. The skill half could be compiled, but without the rule activation trigger the workflow would exist as a passive skill that is never automatically invoked. `Capabilities().Workflows` is set to `false`, and the orchestrator emits `RENDERER_KIND_UNSUPPORTED` for any workflow targeting Codex. If Codex adds rule support in the future, or if xcaffold introduces a skill-only lowering strategy that uses a different activation mechanism, this gap can be closed.

These declarations are locked by `provider_features_test.go`, which asserts the exact capability set for every provider. Changing a capability without updating the test is a compile-time failure.

### The Orchestrator

`renderer.Orchestrate()` is the central dispatch function. It receives a `TargetRenderer`, an `ast.XcaffoldConfig`, and a base directory, then:

1. Reads the renderer's `Capabilities()`
2. For each resource kind with data in the config:
   - If the capability is `true`, calls the corresponding `Compile*` method
   - If the capability is `false`, emits one `RENDERER_KIND_UNSUPPORTED` fidelity note per resource ID
3. Calls `Finalize()` on the merged file map for post-processing
4. Returns the complete `output.Output` with all accumulated fidelity notes

This design eliminates silent resource drops. Before the orchestrator, a renderer that did not handle a resource kind simply ignored it — the config was processed, the resource was skipped, and no feedback was given. With the orchestrator, every resource either produces output or produces a note. There is no third option.

### FidelityNotes

Every `Compile*` method returns a `[]FidelityNote` alongside its file map. A `FidelityNote` carries a stable code (e.g. `RENDERER_KIND_UNSUPPORTED`, `FIELD_UNSUPPORTED`, `AGENT_MODEL_UNMAPPED`), a severity level, the target name, and a human-readable reason.

Fidelity notes are not errors. Compilation always succeeds. Notes inform the user that a configuration concept present in the source had no representation in the target format and was dropped, transformed, or degraded. The full code catalog is defined in `internal/renderer/fidelity_codes.go`.

### Shared Renderer Helpers

Cross-renderer utilities live in the root `internal/renderer/` package:

| Helper | Purpose |
|---|---|
| `CompileSkillSubdir` | Copies skill subdirectory files (`references/`, `scripts/`, `assets/`) into the output map |
| `SortedKeys` | Returns map keys in deterministic alphabetical order |
| `YAMLScalar` | Emits a YAML scalar with correct quoting for frontmatter |
| `StripAllFrontmatter` | Removes YAML frontmatter from markdown content |

The `LowerWorkflows` helper lives in `internal/renderer/shared/` — a separate subpackage — because it depends on the `translator` package, which would create an import cycle if placed in the root `renderer` package.

### Per-Provider Subdirectory Translation

Skills support four canonical subdirectories (`references/`, `scripts/`, `assets/`, `examples/`). Each renderer translates these canonical names to provider-native directory names at the renderer boundary. The translation is declared in each renderer's `CompileSkills` method and executed by the shared `CompileSkillSubdir` helper.

| Canonical | Claude Code | Gemini CLI | Cursor | GitHub Copilot | Antigravity | Codex |
|---|---|---|---|---|---|---|
| `references/` | `references/` | `references/` | `references/` | co-located | `examples/` | `references/` |
| `scripts/` | `scripts/` | `scripts/` | `scripts/` | co-located | `scripts/` | `scripts/` |
| `assets/` | `assets/` | `assets/` | `assets/` | co-located | `resources/` | `assets/` |
| `examples/` | flat alongside SKILL.md | collapse into `references/` | collapse into `references/` | co-located | `examples/` | flat alongside SKILL.md |

**FidelityNote (unsupported).** When a canonical subdirectory has no equivalent in the target provider, the renderer emits a `FidelityNote` with code `FIELD_UNSUPPORTED` and drops the files. Claude Code does not support `assets/` — placing files there produces a fidelity warning, not an error. Compilation succeeds; the warning informs the user that those files were not emitted.

**Co-located (Copilot).** GitHub Copilot does not use subdirectories within skill folders. All supporting files are placed flat alongside `SKILL.md` in the skill's output directory. The canonical subdirectory structure is flattened during rendering.

**Collapse.** Gemini CLI and Cursor merge `examples/` files into the `references/` output directory. The semantic distinction between "demonstrate" and "inform" does not exist in these providers' native layouts, so both are emitted under the provider's references directory.

**Flat alongside SKILL.md (Claude Code examples).** Claude Code's `examples/` files are placed directly in the skill directory next to `SKILL.md`, not in a subdirectory. This matches Claude Code's native convention for example content.

#### Provider Passthrough (`xcaf/provider/`)

When a provider-native feature has no canonical equivalent — a file type or directory structure specific to one provider that xcaffold's canonical schema does not model — users can place files in `xcaf/provider/<provider-name>/`. These files are copied verbatim into the provider's output directory during compilation, bypassing the canonical translation layer entirely.

This is the file-level equivalent of `target-options:` (which provides field-level passthrough). Both mechanisms share the same principle: xcaffold cannot anticipate every present and future provider feature, so it provides an explicit escape hatch for provider-native content that falls outside the canonical schema.

---

## The Six Providers

| Provider | Import Dir | Output Dir | Notable Characteristics |
|---|---|---|---|
| **Claude Code** | `.claude/` | `.claude/` | Full feature support including memory and skill subdirectories; hooks in `settings.json` |
| **Cursor** | `.cursor/` | `.cursor/` | Skill subdirectories supported; no model field; manual-mention rule activation |
| **Gemini CLI** | `.gemini/` | `.gemini/` | Uses `@-import` syntax for project instructions; hooks and MCP in `settings.json` |
| **GitHub Copilot** | `.github/` | `.github/` | Instructions in `.github/copilot-instructions.md`; prompts as `.prompt.md` files |
| **Antigravity** | `.agents/` | `.agents/` | No hooks support; has memory (knowledge items); manual rule activation |
| **Codex** (Preview) | `.codex/` | `.codex/` | TOML agent definitions; rules unsupported (Starlark paradigm); memory is API-managed; shares `.agents/skills/` with Antigravity; `AGENTS.md` root context file |

Codex is the only provider that uses TOML for agent definitions — all others use Markdown with optional YAML frontmatter. Rules are not compiled for Codex because it uses Starlark `.rules` files, a fundamentally different paradigm from xcaffold's YAML-based rule model. Memory is managed via API rather than files on disk, so xcaffold emits no memory output for this target. Skills are written to `.agents/skills/` via the `rootFiles` mechanism — the same directory that Antigravity reads — meaning a single compile pass produces skill output readable by both providers. Codex's root context file is `AGENTS.md`, the same filename Cursor uses, which creates a collision risk when both providers are active targets; see [Multi-Target Compilation](../../best-practices/multi-target-compilation.md) for guidance. The "(Preview)" designation signals that Codex renderer behavior may change in minor releases.

### Native Provider Runtime Loading

Each provider natively loads resources from multiple execution scopes autonomously at runtime. Understanding this is critical: xcaffold explicitly strips `global` scope elements during typical project compilation because it knows the native provider binary will merge its own global directory itself.

| Provider | User-global scope | Project scope | Behavior when same name exists |
|---|---|---|---|
| **Claude Code** | `~/.claude/agents/` | `.claude/agents/` | **Project wins** — higher priority; user-global is silently dropped |
| **Claude Code** | `~/.claude/rules/` | `.claude/rules/` | **Additive** — both loaded; project scope takes precedence on conflict |
| **Claude Code** | `~/.claude/settings.json` → `mcpServers` | `.claude/settings.json` → `mcpServers` | **Project wins** — same server name: project replaces user-global |
| **Cursor** | User Rules (Settings UI, no files) | `.cursor/rules/` | **Additive** — all merged; Team > Project > User on conflict |
| **Gemini CLI** | `~/.gemini/GEMINI.md` | `GEMINI.md` (CWD) | **Additive** — concatenated; CWD loaded last (practical precedence) |
| **GitHub Copilot** | Personal instructions | `.github/copilot-instructions.md` | **All additive** — all instruction types sent simultaneously |

Because providers already handle user-global loading natively, **xcaffold never writes global resources into the project output directory**. Doing so would cause double-injection (the provider loads user-global from `~/.claude/` AND the project from `.claude/` — and one would shadow the other unpredictably).

---

## Adding a New Provider

Adding a new provider requires no changes to the core compiler, parser, or AST. The steps:

1. **Create the importer** — `internal/importer/<name>/<name>.go` implementing `ProviderImporter`. Define a `[]KindMapping` table mapping file patterns to AST kinds. Implement `Import()` to walk the directory.
2. **Create the renderer** — `internal/renderer/<name>/<name>.go` implementing `TargetRenderer`. Implement per-resource `Compile*` methods for each supported kind. Return a `CapabilitySet` from `Capabilities()` declaring what the renderer supports.
3. **Register both** — Call `importer.Register()` and add the renderer to the compiler's `resolveRenderer()` switch in `internal/compiler/compiler.go`.
4. **Lock the ground truth** — Add a row to `provider_features_test.go` asserting the expected `CapabilitySet`, `Target()`, and `OutputDir()`.
5. **Add test fixtures** — Golden test data in `testdata/input/` for round-trip verification.

The orchestrator automatically dispatches to per-resource methods and emits fidelity notes for unsupported kinds based on the `CapabilitySet`. No orchestrator changes are needed.

---

## Related

- [Architecture](overview.md) — full compilation pipeline and internal package map
- [Multi-Target Rendering](multi-target-rendering.md) — how the AST enables same-source, different-output compilation
- [Supported Providers](../../reference/supported-providers.md) — capability matrix and per-provider field support
- [Kind Reference](../../reference/kinds/index.md) — per-target fidelity mappings
