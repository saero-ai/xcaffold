---
title: "Multi-Target Rendering"
description: "How the AST enables the same configuration to compile to different platform formats"
---

# Multi-Target Rendering

A single `.xcaf` file describes your agent configuration once. xcaffold compiles that description into whichever native format a target AI platform expects — `.claude/` for Claude Code, `.cursor/` for Cursor, `.agents/` for Antigravity, `.github/` for GitHub Copilot, or `.gemini/` for Gemini CLI. The same source, different outputs, without editing the configuration between runs.

This works because xcaffold treats configuration as data and delegates all format concerns to per-target renderers.

## AST as Data/Presentation Separation

When xcaffold parses a `.xcaf` file, the result is a typed Go struct — `ast.XcaffoldConfig` — that holds all configuration data in a platform-agnostic form: agent identities, skill definitions, rule bodies, hook commands, MCP server declarations, and settings values. The struct knows nothing about output formats. It does not know whether a rule becomes a `.md` file or a `.mdc` file, whether a hook gets serialized as JSON or ignored, or whether there is even a directory to write into.

Translation is delegated entirely to the `TargetRenderer` interface (`internal/renderer/renderer.go`):

```go
type TargetRenderer interface {
    Target() string
    OutputDir() string
    Capabilities() CapabilitySet

    CompileAgents(agents map[string]AgentConfig, baseDir string) (map[string]string, []FidelityNote, error)
    CompileSkills(skills map[string]SkillConfig, baseDir string) (map[string]string, []FidelityNote, error)
    // ... per-resource methods for rules, workflows, hooks, settings, MCP, project instructions

    Finalize(files map[string]string) (map[string]string, []FidelityNote, error)
}
```

Each renderer declares which resource kinds it supports via a `CapabilitySet`. The `Orchestrate()` function dispatches compilation to per-resource methods for supported kinds and automatically emits `RENDERER_KIND_UNSUPPORTED` fidelity notes for unsupported ones — no renderer silently drops resources. The AST is never modified during rendering; the compiler passes it by pointer but renderers treat it as read-only input.

The consequence: a rule defined as `paths: ["src/**/*.ts"]` with a Markdown body appears as `rules/<id>.md` with a `paths:` frontmatter key when compiled for one target, and as `rules/<id>.mdc` with a `globs:` key and `always-apply: true` when compiled for another. The rule's *data* — its ID, scope patterns, and instruction body — is stable. Its *presentation* is determined entirely by the renderer.

## Target Fidelity and Warnings

> For a complete matrix of capabilities, supported fields, and fidelity mappings per target, see the [Kind Reference](../../reference/kinds/index.md).

The fidelity warnings are not errors. Compilation always succeeds; warnings inform you that a configuration concept present in the source had no representation in the target format and was silently dropped.

## The Five-Target Architecture

xcaffold ships five renderers, each targeting a distinct platform:

| Target | Output directory | Format |
|---|---|---|
| `claude` | `.claude/` | YAML-frontmatter Markdown agents, `settings.json`, `mcp.json` |
| `cursor` | `.cursor/` | YAML-frontmatter Markdown agents, `.mdc` rules, `mcp.json` |
| `antigravity` | `.agents/` | Plain Markdown workflow definitions, `mcp_config.json` |
| `copilot` | `.github/` | GitHub Copilot instructions and prompt files |
| `gemini` | `.gemini/` | YAML-frontmatter Markdown agents, `rules/*.md`, `settings.json` |

Each renderer is an independent implementation of the `TargetRenderer` interface. Renderers differ in which fields they support, how they name output files, and how they serialize MCP and hook configuration. A field that is fully supported in `claude` may be silently dropped in `gemini` with a fidelity warning. A field with no equivalent in any target is still parsed and stored in the AST — it is simply not emitted.

The `gemini` target writes project-level instructions to `GEMINI.md` at the repository root, using Gemini's native `@`-import syntax to reference rule files stored under `.gemini/rules/`. Agent system prompts are written to `.gemini/agents/<id>.md` with YAML frontmatter. Hooks and MCP servers are serialized to `.gemini/settings.json`.

## Target-Determined Output Directories

No output directory is assumed at the time the `.xcaf` file is parsed. The compiler never writes to a default location. The target determines the directory at the point `compiler.OutputDir(target)` is called:

```go
func OutputDir(target string) string {
    r, err := resolveRenderer(target)
    if err != nil {
        return ""
    }
    return r.OutputDir()
}
```

`resolveRenderer()` maps a target name to its `TargetRenderer` instance. Unknown targets return an error and `OutputDir` returns an empty string — the compiler does not silently default to any provider's directory.

Each renderer's `OutputDir()` method owns the answer. The compiler calls the method; it does not hardcode the path. Adding a new renderer for a new target requires only implementing `TargetRenderer` and registering it — no changes to the compiler's dispatch logic or to any path-resolution logic outside the new renderer.

When the target is `"cursor"`, every file path in the output `map[string]string` is interpreted relative to `.cursor/`. When the target is `"gemini"`, paths are relative to `.gemini/`. The file map structure is identical in both cases; only the base directory differs.

## MCP Shorthand and Settings Merge

The `.xcaf` schema provides two ways to declare MCP servers. A top-level `mcp:` block is a shorthand for listing servers directly without nesting them under `settings:`. A `settings.mcpServers` block is the fully qualified path. Both can appear in the same file.

During compilation, the Claude renderer merges both sources in `compileClaudeMCP` (`internal/renderer/claude/claude.go:415–437`). The merge is additive: `mcp:` entries populate the output map first, then `settings.mcpServers` entries are written over them. When both define a server with the same key, `settings.mcpServers` wins.

The merge happens entirely in the renderer. The raw `.xcaf` YAML is not modified. The `ast.XcaffoldConfig` struct retains `MCP` and `Settings.MCPServers` as separate fields throughout the compilation pipeline. The merged result appears only in the rendered `mcp.json`.

For the `cursor` target, only the `mcp:` shorthand block is compiled to `mcp.json` (`internal/renderer/cursor/cursor.go:97–104`). For the `antigravity` target, MCP servers are written to `mcp_config.json` using a reduced schema that supports only `command`, `args`, and `env` — the `url` and `headers` fields used for HTTP-based MCP servers have no equivalent and are silently dropped.

## Per-Target State Files as Proof of Separation

A project compiled for both targets produces a single `.xcaffold/project.xcaf.state` file containing artifact hashes for both targets under separate target sections. Per-blueprint compilations produce `.xcaffold/<blueprint-name>.xcaf.state`. Each state file records the SHA-256 hashes of that context's artifacts, the xcaffold version, and the timestamp of the last apply.

This separation is significant for teams that maintain multiple deployment contexts from a single `.xcaf` file. Advancing a `claude` compilation — adding new rules, updating agent definitions — does not invalidate the `cursor` state section, and vice versa. Drift detection operates independently per target. A team can keep one target stable while iterating on another, with the state file providing the audit trail for each independently.

## Import Side

The import direction mirrors the render direction. Each provider has a `ProviderImporter` implementation (`internal/importer/<provider>/`) symmetric to its `TargetRenderer`. Where renderers translate `ast.XcaffoldConfig` → native files, importers translate native files → `ast.XcaffoldConfig`. The same five-provider model applies in both directions, and the AST serves as the shared IR between them.

Files that no importer recognizes go to `ProviderExtras`, a per-provider bucket that preserves unclassified artifacts for same-provider round-trips without contaminating the typed AST. This keeps the AST strictly typed while ensuring no data is silently discarded during import.

> See [Provider Architecture](provider-architecture.md) for the full pipeline diagram and kind classification layout table.
