---
title: "Multi-Target Rendering"
description: "How the AST enables the same configuration to compile to different platform formats"
---

# Multi-Target Rendering

A single `.xcf` file describes your agent configuration once. xcaffold compiles that description into whichever native format a target AI platform expects — `.claude/` for one runtime, `.cursor/` for another, `.agents/` for a third, or a portable `AGENTS.md` for any runtime that reads the community-standard format. The same source, different outputs, without editing the configuration between runs.

This works because xcaffold treats configuration as data and delegates all format concerns to per-target renderers.

## AST as Data/Presentation Separation

When xcaffold parses a `.xcf` file, the result is a typed Go struct — `ast.XcaffoldConfig` — that holds all configuration data in a platform-agnostic form: agent identities, skill definitions, rule bodies, hook commands, MCP server declarations, and settings values. The struct knows nothing about output formats. It does not know whether a rule becomes a `.md` file or a `.mdc` file, whether a hook gets serialized as JSON or ignored, or whether there is even a directory to write into.

Translation is delegated entirely to the `TargetRenderer` interface (`internal/renderer/renderer.go`):

```go
type TargetRenderer interface {
    Target() string
    OutputDir() string
    Render(files map[string]string) *output.Output
}
```

Each renderer implements this interface and receives the same `ast.XcaffoldConfig`. The renderer decides how each field maps to the target platform's file format, which fields have equivalents, and which must be dropped with a fidelity warning. The AST is never modified during rendering; the compiler passes it by pointer but renderers treat it as read-only input.

The consequence: a rule defined as `paths: ["src/**/*.ts"]` with a Markdown body appears as `rules/<id>.md` with a `paths:` frontmatter key when compiled for one target, and as `rules/<id>.mdc` with a `globs:` key and `always-apply: true` when compiled for another. The rule's *data* — its ID, scope patterns, and instruction body — is stable. Its *presentation* is determined entirely by the renderer.

## Target Fidelity and Warnings

> For a complete matrix of capabilities, supported fields, and fidelity mappings per target, see the [Schema Reference](../reference/schema.md).

The fidelity warnings are not errors. Compilation always succeeds; warnings inform you that a configuration concept present in the source had no representation in the target format and was silently dropped.

## The Portable Baseline: agentsmd

The `agentsmd` target produces `AGENTS.md` files at the repository root (and, for path-scoped rules, in the appropriate subdirectory). It is the only target whose `OutputDir()` returns `"."` rather than a hidden directory (`internal/renderer/agentsmd/agentsmd.go:33`).

The format is intentionally minimal. Each resource type — agents, skills, rules, workflows — becomes a Markdown section headed by `##`. Individual resources become `###` subsections. No YAML frontmatter appears in the output. The goal is plain prose that any AI runtime capable of reading a Markdown file can ingest without knowing anything about xcaffold's internal conventions.

Because `agentsmd` is the most permissive in terms of readability and the most restrictive in terms of structured metadata, it serves as a lowest-common-denominator format for cross-platform distribution. A configuration that compiles cleanly to `agentsmd` is guaranteed to be readable by any tool that accepts the community `AGENTS.md` convention.

The tradeoff is fidelity. Fields that depend on platform-specific semantics — tool lists, permission modes, hook commands, MCP server declarations, sandbox constraints — have no `AGENTS.md` equivalent and are dropped. The renderer emits a warning for each dropped field via `warnLossyAgent`, `warnLossySkill`, and `warnLossyRule` (`internal/renderer/agentsmd/agentsmd.go:346–430`).

## Target-Determined Output Directories

No output directory is assumed at the time the `.xcf` file is parsed. The compiler never writes to a default location. The target determines the directory at the point `compiler.OutputDir(target)` is called (`internal/compiler/compiler.go:103–119`):

```go
func OutputDir(target string) string {
    if target == "" {
        target = TargetClaude
    }
    switch target {
    case TargetClaude:      return claude.New().OutputDir()      // ".claude"
    case TargetCursor:      return cursor.New().OutputDir()      // ".cursor"
    case TargetAntigravity: return antigravity.New().OutputDir() // ".agents"
    case TargetAgentsMD:    return agentsmd.New().OutputDir()    // "."
    default:                return ".claude"
    }
}
```

When no `--target` flag is provided, the empty string defaults to `TargetClaude` before the switch is evaluated. This is the only place in the compiler where a default target is assumed.

Each renderer's `OutputDir()` method owns the answer. The compiler calls the method; it does not hardcode the path. This means adding a new renderer for a new target requires only implementing `TargetRenderer` and registering it — no changes to the compiler's dispatch logic or to any path-resolution logic outside the new renderer.

When the target is `"cursor"`, every file path in the output `map[string]string` is interpreted relative to `.cursor/`. When the target is `"agentsmd"`, paths are relative to the project root. The file map structure is identical in both cases; only the base directory differs.

## MCP Shorthand and Settings Merge

The `.xcf` schema provides two ways to declare MCP servers. A top-level `mcp:` block is a shorthand for listing servers directly without nesting them under `settings:`. A `settings.mcpServers` block is the fully qualified path. Both can appear in the same file.

During compilation, the Claude renderer merges both sources in `compileClaudeMCP` (`internal/renderer/claude/claude.go:415–437`). The merge is additive: `mcp:` entries populate the output map first, then `settings.mcpServers` entries are written over them. When both define a server with the same key, `settings.mcpServers` wins.

The merge happens entirely in the renderer. The raw `.xcf` YAML is not modified. The `ast.XcaffoldConfig` struct retains `MCP` and `Settings.MCPServers` as separate fields throughout the compilation pipeline. The merged result appears only in the rendered `mcp.json`.

For the `cursor` target, only the `mcp:` shorthand block is compiled to `mcp.json` (`internal/renderer/cursor/cursor.go:97–104`). For the `antigravity` target, MCP servers are written to `mcp_config.json` using a reduced schema that supports only `command`, `args`, and `env` — the `url` and `headers` fields used for HTTP-based MCP servers have no equivalent and are silently dropped.

## Per-Target Lock Files as Proof of Separation

Each compilation target produces its own lock file. `state.LockFilePath` computes the path from the base lock filename and the active target (`internal/state/state.go:22–29`):

```go
func LockFilePath(basePath string, target string) string {
    if target == "" {
        target = "claude"
    }
    ext := filepath.Ext(basePath)
    base := strings.TrimSuffix(basePath, ext)
    return base + "." + target + ext
}
```

A project compiled for both `claude` and `cursor` produces `scaffold.claude.lock` and `scaffold.cursor.lock` as independent files. Each lock records the SHA-256 hashes of that target's artifacts, the xcaffold version, and the timestamp of the last apply. Neither lock file references the other.

This separation is significant for teams that maintain multiple deployment contexts from a single `.xcf` file. Advancing a `claude` compilation — adding new rules, updating agent definitions — does not invalidate the `cursor` lock, and vice versa. Drift detection operates independently per target. A team can keep one target stable while iterating on another, with the lock file providing the audit trail for each independently.
