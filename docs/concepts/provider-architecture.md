---
title: "Provider Architecture"
description: "Symmetric ProviderImporter and TargetRenderer interfaces and the import/render pipeline"
---

# Provider Architecture

xcaffold uses a symmetric import/render architecture. Each supported provider (Claude Code, Cursor, Gemini CLI, GitHub Copilot, Antigravity) has two components:

- **ProviderImporter** (`internal/importer/<provider>/`) — reads the provider's native directory structure and populates the AST
- **TargetRenderer** (`internal/renderer/<provider>/`) — compiles the AST into the provider's native directory structure

## Pipeline

```
Provider Directory (.claude/, .cursor/, .gemini/, .github/, .agents/)
    |
    v
ProviderImporter.Import()
    |-- Classify(path) -> Kind + Layout
    |-- Extract(path, data) -> populate AST
    |-- Shared helpers: ParseFrontmatter, MatchGlob, ReadFile
    |-- Unclassified -> ProviderExtras
    |
    v
ast.XcaffoldConfig (shared IR)
    |
    v
renderer.Orchestrate(TargetRenderer, config, baseDir)
    |-- Capabilities() -> CapabilitySet (declares supported resource kinds)
    |-- Per-kind dispatch: CompileAgents, CompileSkills, CompileRules, ...
    |-- Unsupported kinds -> RENDERER_KIND_UNSUPPORTED FidelityNote
    |-- Finalize() -> post-processing pass
    |
    v
Provider Directory (output)
```

## Kind Classification

Each importer owns a `[]KindMapping` table that maps file patterns to AST kinds:

| Layout | Example | Description |
|--------|---------|-------------|
| FlatFile | `agents/*.md` | One file per resource |
| DirectoryPerEntry | `skills/*/SKILL.md` | One subdirectory per resource |
| StandaloneJSON | `mcp.json` | Single JSON file for all resources of one kind |
| EmbeddedJSONKey | `settings.json#hooks` | Key inside a container JSON file |

Files that match no pattern go to `ProviderExtras` — a genuinely-unknown catchall that preserves round-trip fidelity for same-provider workflows.

## Adding a New Provider

1. Create `internal/importer/<name>/<name>.go` implementing `ProviderImporter`
2. Create `internal/renderer/<name>/<name>.go` implementing `TargetRenderer` (per-resource methods + `Capabilities()` + `Finalize()`)
3. Register both via `init()` functions
4. Add a row to `provider_features_test.go` to lock the capability ground truth
5. Add golden test data in `testdata/input/`

No changes needed to the core compiler, parser, or AST. The orchestrator automatically dispatches to per-resource methods and emits fidelity notes for unsupported kinds based on the `CapabilitySet`.
