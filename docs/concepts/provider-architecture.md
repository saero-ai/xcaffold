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
    |-- Unclassified -> ProviderExtras
    |
    v
ast.XcaffoldConfig (shared IR)
    |
    v
TargetRenderer.Compile()
    |-- Per-kind rendering
    |-- FidelityNotes for information loss
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
2. Create `internal/renderer/<name>/<name>.go` implementing `TargetRenderer`
3. Register both via `init()` functions
4. Add golden test data in `testdata/input/`

No changes needed to the core compiler, parser, or AST.
