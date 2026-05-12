---
title: "Declarative Compilation"
description: "Why xcaffold enforces one-way compilation, the fail-closed parser, and the AST trust boundary"
---

# Declarative Compilation

Agent configurations are software artifacts. Like infrastructure-as-code, they should be versioned, audited, and reproduced from source — not edited in place and synced back. This is the core of Harness-as-Code: the agent harness is software, not a settings panel. xcaffold enforces this by treating `.xcaf` files as the authoritative source of record and compiling them, in one direction, into native runtime files for each target platform. This document explains the reasoning behind that architectural choice.

### Declarative Configuration vs Prompt-at-Runtime

The conventional approach to configuring AI agents is to paste instructions directly into an IDE panel or settings dialog. The instructions exist only in that session; they are not versioned, they cannot be diffed, and reproducing them exactly on another machine or for another team member requires manual copy-paste. This is the prompt-at-runtime model.

xcaffold uses declarative configuration compiled ahead of time. Agents, skills, rules, hooks, and MCP server bindings are declared in `.xcaf` files that live alongside the codebase. Changes are made to the `.xcaf` file, committed with a message, reviewed in a pull request, and rolled back with `git revert` if needed. The runtime configuration is generated, not authored.

The distinction matters beyond aesthetics. Prompt-at-runtime configurations are invisible to code review. They cannot be linted, tested, or audited. A typo in a system prompt has no stack trace. Declarative configuration surfaces these problems at compile time, before any agent executes.

### The AST as the Separation Boundary

The compiler does not transform YAML strings directly into Markdown or JSON. Between parsing and rendering sits an Abstract Syntax Tree (AST): the `XcaffoldConfig` struct, defined in `internal/ast/types.go`.

```go
type XcaffoldConfig struct {
    Kind    string `yaml:"-"`    // Set by parser routing, not decoded from YAML
    Version string `yaml:"version"`
    Extends string `yaml:"extends,omitempty"`

    Settings   map[string]SettingsConfig  `yaml:"settings,omitempty"`
    Blueprints map[string]BlueprintConfig `yaml:"blueprints,omitempty"`

    ResourceScope `yaml:",inline"` // Global-level resources
    Project *ProjectConfig `yaml:"project,omitempty"`
}
```

`ResourceScope` holds all the agentic primitives — agents, skills, rules, hooks, MCP servers, workflows — as typed Go maps. The AST has no knowledge of any target platform. It carries no Markdown syntax, no JSON keys specific to any runtime, no platform-specific field names.

This separation is the boundary that makes multi-target rendering possible. The same `AgentConfig` is handed to the claude renderer, which writes `.claude/agents/<id>.md`; and to the cursor renderer, which writes `.cursor/rules/<id>.mdc`. The `.xcaf` source does not change. Platform-specific concerns never leak into the data model.

### Determinism as a Contract

xcaffold makes a hard guarantee: given the same `.xcaf` file, every invocation of the compiler produces byte-for-byte identical output. There are no timestamps embedded in generated file content, no random identifiers, no environment-dependent paths inside compiled files.

This guarantee is what makes drift detection meaningful. After compilation, `state.GenerateState()` (`internal/state/state.go`) hashes every output artifact with SHA-256 and records the results in a state file:

```go
hash := sha256.Sum256([]byte(content))
artifacts = append(artifacts, Artifact{
    Path: path,
    Hash: fmt.Sprintf("sha256:%x", hash),
})
```

On the next run, the state file is compared against the current hashes of the same paths. Any divergence means someone edited a generated file directly. If determinism were not guaranteed, the compiler itself would appear to produce drift every run, making the state file useless.

Determinism is also what makes CI verification possible. A pipeline that runs `xcaffold apply` and then checks for uncommitted changes in the target output directory only works if clean-source-in produces clean-output-out, every time.

### Single-Resource File Format

Each `.xcaf` file contains exactly one resource document. The parser enforces this constraint: if a second `kind:` declaration is detected in the same file, parsing fails immediately with an error directing the author to split the file.

The `---` delimiters in a `.xcaf` file separate YAML frontmatter from the Markdown body — they do not separate multiple YAML documents. Frontmatter contains the resource's typed fields (name, description, tools, model, etc.). The body, if present, carries free-form instructions that become the resource's `Body` field.

When `ParseDirectory` scans a project tree, it discovers all `.xcaf` files recursively and merges all parsed resources into a single configuration. Strict deduplication is enforced: if the same resource ID (e.g., agent `deployer`) appears in two different files, parsing fails with a duplicate ID error. This prevents ambiguous precedence and ensures every resource has exactly one authoritative definition.

### The Fail-Closed Parser

The YAML parser is strict by design. `parsePartial()` (`internal/parser/parser.go`) creates a `yaml.Decoder` and calls `KnownFields(true)` before decoding:

```go
func parsePartial(r io.Reader, opts ...parseOptionFunc) (*ast.XcaffoldConfig, error) {
    data, err := io.ReadAll(r)
    // ... variable expansion via resolver.ExpandVariables() ...

    frontmatter, body, err := extractFrontmatterAndBody(data)

    config := &ast.XcaffoldConfig{}
    decoder := yaml.NewDecoder(bytes.NewReader(frontmatter))
    // ... routes each document by kind: field ...
}
```

During per-resource-kind parsing (`internal/parser/resource_kinds.go`), `KnownFields(true)` instructs the decoder to return an error if the YAML document contains any field that does not map to a struct tag in the AST. The parse fails immediately on the first unknown field; there is no partial result, no warning, no silent skip.

The alternative — accepting unknown fields and ignoring them — would make typos invisible. A misspelled field like `instrctions:` would silently produce an agent with no instructions, and the user would debug agent behavior rather than configuration syntax. By failing closed, the parser makes the schema the contract: anything accepted by the parser is structurally valid.

The same strict posture extends to cross-resource references. `validateCrossReferences()` (`internal/parser/parser_validation.go`) verifies that every agent-referenced skill ID, rule ID, and MCP server ID is defined in the same config. A reference to an undefined resource is a parse-time error, not a runtime surprise.

### One-Way Compilation as a Trust Boundary

Generated files in `.claude/`, `.cursor/`, and `.agents/` are machine outputs. They are not intended to be edited by hand, and xcaffold does not read them back. The compilation direction is fixed: `.xcaf` in, platform files out.

`Compile()` (`internal/compiler/compiler.go`) makes this flow explicit. It merges project-scoped resources over global-scoped resources, strips inherited resources that should not be duplicated locally, resolves the target renderer, and dispatches via the orchestrator:

```go
func Compile(config *ast.XcaffoldConfig, baseDir, target, blueprintName, varFile string) (*output.Output, []renderer.FidelityNote, error) {
    if config.Project != nil {
        mergeResourceScope(&config.ResourceScope, &config.Project.ResourceScope)
    }
    resolver.ResolveAttributes(config)
    // ... variable loading, blueprint resolution ...
    config.StripInherited()
    r, err := ResolveRenderer(target)
    if err != nil {
        return nil, nil, err
    }
    return renderer.Orchestrate(r, config, baseDir)
}
```

`renderer.Orchestrate()` iterates each resource kind, checks the renderer's `Capabilities()`, and calls the appropriate `Compile*` method or emits a `RENDERER_KIND_UNSUPPORTED` note. There is no code path that reads compiled output files and updates `.xcaf`. This asymmetry is the trust boundary. When a generated file is found to differ from what the compiler would produce, the answer is always "recompile," never "sync back." The `.xcaf` source is the truth; the generated files are a derived view of it.

Bidirectional sync would collapse this boundary. If edits to generated files were propagated back into the `.xcaf` source, the system would have two authorities for the same configuration and no principled way to resolve conflicts. One-way compilation avoids that class of problem entirely.

### The Frontmatter Body as Instructions

Every resource type that carries agent instructions — agents, skills, rules, workflows — uses the frontmatter body to provide that content. The `.xcaf` file format uses `---` delimiters: YAML configuration sits between the delimiters, and Markdown content follows:

```yaml
---
kind: agent
version: "1.0"
name: reviewer
description: "Code review specialist."
model: sonnet
tools: [Read, Glob, Grep]
---
You are a code reviewer. Focus on correctness, security, and maintainability.
Never approve code that introduces panics in library packages.
```

The parser's `extractFrontmatterAndBody()` splits the file at compile time. The frontmatter is decoded into the resource's typed struct fields. The body is stored as the `Body` string field (tagged `yaml:"-"` — it is never decoded from YAML). Renderers embed this body verbatim into the compiled output as the resource's instructions.

This design means long agent system prompts benefit from Markdown authoring tools, syntax highlighting, and review comments. The `.xcaf` file remains the single configuration entry point — there is no separate instructions file to keep in sync.
