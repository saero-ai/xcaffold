---
title: "Declarative Compilation"
description: "Why xcaffold uses a one-way deterministic compiler instead of bidirectional sync"
---

# Declarative Compilation

Agent configurations are software artifacts. Like infrastructure-as-code, they should be versioned, audited, and reproduced from source — not edited in place and synced back. xcaffold enforces this by treating `.xcf` YAML as the authoritative source of record and compiling it, in one direction, into native runtime files for each target platform. This document explains the reasoning behind that architectural choice.

## agent configuration vs. Agent-as-Prompt

The conventional approach to configuring AI agents is to paste instructions directly into an IDE panel or settings dialog. The instructions exist only in that session; they are not versioned, they cannot be diffed, and reproducing them exactly on another machine or for another team member requires manual copy-paste. This is the "Agent-as-Prompt" model.

xcaffold introduces a different model: agent configuration compiler. Agents, skills, rules, hooks, and MCP server bindings are declared in `.xcf` YAML files that live alongside the codebase. Changes are made to the YAML, committed with a message, reviewed in a pull request, and rolled back with `git revert` if needed. The runtime configuration is generated, not authored.

The distinction matters beyond aesthetics. Agent-as-Prompt configurations are invisible to code review. They cannot be linted, tested, or audited. A typo in a system prompt has no stack trace. Declarative configuration surfaces these problems at compile time, before any agent executes.

## The AST as the Separation Boundary

The compiler does not transform YAML strings directly into Markdown or JSON. Between parsing and rendering sits an Abstract Syntax Tree (AST): the `XcaffoldConfig` struct, defined in `internal/ast/types.go`.

```go
type XcaffoldConfig struct {
    Kind    string `yaml:"kind,omitempty"`
    Version string `yaml:"version"`
    Extends string `yaml:"extends,omitempty"`

    Settings SettingsConfig `yaml:"settings,omitempty"`

    ResourceScope `yaml:",inline"` // Global-level resources
    Project *ProjectConfig `yaml:"project,omitempty"`
}
```

`ResourceScope` holds all the agentic primitives — agents, skills, rules, hooks, MCP servers, workflows — as typed Go maps. The AST has no knowledge of any target platform. It carries no Markdown syntax, no JSON keys specific to any runtime, no platform-specific field names.

This separation is the boundary that makes multi-target rendering possible. The same `AgentConfig` is handed to the claude renderer, which writes `.claude/agents/<id>.md`; to the cursor renderer, which writes `.cursor/rules/<id>.md`; and to the agentsmd renderer, which writes `.agents/<id>.md`. The YAML source does not change. Platform-specific concerns never leak into the data model.

## Determinism as a Contract

xcaffold makes a hard guarantee: given the same `.xcf` source, every invocation of the compiler produces byte-for-byte identical output. There are no timestamps embedded in generated file content, no random identifiers, no environment-dependent paths inside compiled files.

This guarantee is what makes drift detection meaningful. After compilation, `state.GenerateWithOpts()` (`internal/state/state.go:70`) hashes every output artifact with SHA-256 and records the results in a lock file:

```go
hash := sha256.Sum256([]byte(content))
manifest.Artifacts = append(manifest.Artifacts, Artifact{
    Path: path,
    Hash: fmt.Sprintf("sha256:%x", hash),
})
```

On the next run, the lock file is compared against the current hashes of the same paths. Any divergence means someone edited a generated file directly. If determinism were not guaranteed, the compiler itself would appear to produce drift every run, making the lock file useless.

Determinism is also what makes CI verification possible. A pipeline that runs `xcaffold apply` and then checks for uncommitted changes in `.claude/` only works if clean-source-in produces clean-output-out, every time.

## Multi-Document YAML Parsing

A single `.xcf` file can contain multiple YAML documents separated by `---`. The parser's `parsePartial` function loops over each document in the stream and routes it by `kind:`:

- `kind: project` populates `ProjectConfig` with the project name, targets, and resource reference lists.
- `kind: hooks`, `kind: settings`, and other resource kinds merge their contents into `ResourceScope` maps.
- `kind: global` contains global-scope resources and settings.

When `ParseDirectory` scans a project tree, it discovers all `.xcf` files recursively and merges all parsed documents into a single configuration. Strict deduplication is enforced: if the same resource ID (e.g., agent `deployer`) appears in two different files, parsing fails with a duplicate ID error. This prevents ambiguous precedence and ensures every resource has exactly one authoritative definition.

## The Fail-Closed Parser

The YAML parser is strict by design. `parsePartial()` (`internal/parser/parser.go:50`) creates a `yaml.Decoder` and calls `KnownFields(true)` before decoding:

```go
func parsePartial(r io.Reader, opts ...parseOptionFunc) (*ast.XcaffoldConfig, error) {
    config := &ast.XcaffoldConfig{}
    decoder := yaml.NewDecoder(r)
    decoder.KnownFields(true)
    if err := decoder.Decode(config); err != nil {
        return nil, fmt.Errorf("failed to parse .xcf YAML: %w", err)
    }
    ...
}
```

`KnownFields(true)` instructs the decoder to return an error if the YAML document contains any field that does not map to a struct tag in the AST. The parse fails immediately on the first unknown field; there is no partial result, no warning, no silent skip.

The alternative — accepting unknown fields and ignoring them — would make typos invisible. A misspelled field like `instrctions:` would silently produce an agent with no instructions, and the user would debug agent behavior rather than configuration syntax. By failing closed, the parser makes the schema the contract: anything accepted by the parser is structurally valid.

The same strict posture extends to cross-resource references. `validateCrossReferences()` (`internal/parser/parser.go:956`) verifies that every agent-referenced skill ID, rule ID, and MCP server ID is defined in the same config. A reference to an undefined resource is a parse-time error, not a runtime surprise.

## One-Way Compilation as a Trust Boundary

Generated files in `.claude/`, `.cursor/`, and `.agents/` are machine outputs. They are not intended to be edited by hand, and xcaffold does not read them back. The compilation direction is fixed: `.xcf` in, platform files out.

`Compile()` (`internal/compiler/compiler.go:34`) makes this flow explicit. It merges project-scoped resources over global-scoped resources, strips inherited resources that should not be duplicated locally, then dispatches to the appropriate renderer:

```go
func Compile(config *ast.XcaffoldConfig, baseDir string, target string) (*Output, error) {
    if config.Project != nil {
        mergeResourceScope(&config.ResourceScope, &config.Project.ResourceScope)
    }
    config.StripInherited()
    switch target {
    case TargetClaude:
        r := claude.New()
        return r.Compile(config, baseDir)
    case TargetCursor:
        ...
    }
}
```

There is no code path that reads `.claude/` and updates `.xcf`. This asymmetry is the trust boundary. When a generated file is found to differ from what the compiler would produce, the answer is always "recompile," never "sync back." The YAML is the truth; the generated files are a derived view of it.

Bidirectional sync would collapse this boundary. If edits to generated files were propagated back into source YAML, the system would have two authorities for the same configuration and no principled way to resolve conflicts. One-way compilation avoids that class of problem entirely.

## Instructions vs. Instructions File

Every resource type that carries agent instructions — agents, skills, rules, workflows — supports two mutually exclusive ways to provide that content.

`instructions` accepts inline YAML content compiled verbatim into the output. `instructions-file` accepts a relative path to a Markdown file. At compile time, `resolver.ResolveInstructions()` (`internal/resolver/resolver.go:35`) reads the file, strips any YAML frontmatter, and embeds the result:

```go
func ResolveInstructions(inline, filePath, conventionPath, baseDir string) (string, error) {
    if inline != "" {
        return inline, nil
    }
    // filePath resolved relative to baseDir, path traversal rejected
    ...
    b, err := os.ReadFile(bestPath)
    content := StripFrontmatter(string(b))
    return content, nil
}
```

The mutual exclusivity is enforced at parse time by `validateInstructionOrFile()` (`internal/parser/parser.go:949`):

```go
func validateInstructionOrFile(kind, id, inst, file string, globalScope bool) error {
    if inst != "" && file != "" {
        return fmt.Errorf("%s %q: instructions and instructions-file are mutually exclusive; set one or the other", kind, id)
    }
    ...
}
```

Setting both fields is an immediate parse error. This prevents ambiguity about which content would win. The design also prevents a specific circular dependency: `instructions-file` paths that point into compiler output directories (`.claude/`, `.cursor/`, `.agents/`) are explicitly rejected. A compiled file cannot be its own source.

The `instructions-file` mechanism exists because long agent system prompts benefit from Markdown authoring tools, syntax highlighting, and review comments. Separating long-form prose into dedicated `.md` files is an ergonomic choice that does not compromise the compilation model: the content is still embedded at compile time, and the `.xcf` file remains the single configuration entry point.
