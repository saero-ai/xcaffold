# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
make setup          # Install golangci-lint, git hooks
make build          # Compile binary → ./xcaffold
make test           # Run all tests (go test -v ./...)
make lint           # golangci-lint (falls back to go vet)
make clean          # Remove binary and .claude/ output dir

# Single test
go test -v -run TestName ./internal/parser/

# Single package
go test -v ./internal/compiler/
```

## Architecture

xcaffold is a deterministic agent configuration compiler engine that compiles `.xcf` YAML blueprints into Claude Code's native `.claude/` markdown files through a 6-phase lifecycle:

**init → analyze → plan → apply → diff → test**

### Data Flow

```
scaffold.xcf (YAML)
  → parser (strict YAML decode + semantic validation)
  → ast.XcaffoldConfig (in-memory AST)
  → compiler (AST → markdown rendering, resolves instructions_file: refs)
  → state (SHA-256 lock manifest for drift detection)
  → .claude/agents/*.md
  → .claude/skills/*/SKILL.md           ← skills compile to DIRECTORIES
  → .claude/skills/*/references/*.md    ← reference files (when declared)
  → .claude/rules/*.md
  → .claude/settings.json               ← merged mcp: + settings: blocks
  → .claude/hooks.json
  → scaffold.lock
```

### Key Packages

| Package | Role |
|---------|------|
| `cmd/xcaffold/` | Cobra CLI commands — one file per command (init, analyze, plan, apply, diff, test, review, validate, import, graph) |
| `internal/ast` | Core types only (`types.go`): `XcaffoldConfig`, `AgentConfig`, `SkillConfig`, `RuleConfig`, `HookConfig`, `MCPConfig`, `SettingsConfig`, `TestConfig` |
| `internal/parser` | Strict YAML parsing with `KnownFields(true)` — unknown fields fail immediately |
| `internal/compiler` | Translates AST → `compiler.Output{Files: map[path]content}` — one-way, deterministic. Receives `baseDir` for resolving `instructions_file:` |
| `internal/state` | Reads/writes `scaffold.lock` with SHA-256 hashes per artifact |
| `internal/analyzer` | Token counting via wazero WASM runtime; reverse-engineers project context |
| `internal/generator` | Calls Anthropic API to generate scaffold configs; outputs `audit.json` payloads |
| `internal/proxy` | HTTP intercept proxy for sandboxed agent simulation during `xcaffold test` |
| `internal/trace` | JSONL execution trace recording |
| `internal/judge` | LLM-as-a-Judge evaluation against agent assertions |
| `tools/fuzzer` | Fuzz testing harness for parser robustness |

### Design Invariants

- **One-way compilation**: `.xcf` → `.claude/` only; no decompile from existing markdown
- **Fail-closed validation**: Parser rejects unknown YAML fields; compiler never panics
- **Path safety**: All output paths go through `filepath.Clean`; `instructions_file:` and `references:` paths validated against `..` traversal
- **Dual auth**: `internal/judge` and `internal/generator` support `ANTHROPIC_API_KEY` env var or `claude` CLI subprocess
- **Deterministic output**: No randomness in compilation; same input always produces same output
- **Skills as directories**: Skills compile to `skills/<id>/SKILL.md`, never flat `skills/<id>.md`

## .xcf Schema

The `scaffold.xcf` file has these top-level blocks:

| Block | Required | Description |
|-------|:--------:|-------------|
| `version` | ✓ | Schema version string |
| `project` | ✓ | `name`, `description` |
| `agents` | — | Map of agent IDs → `AgentConfig` |
| `skills` | — | Map of skill IDs → `SkillConfig` |
| `rules` | — | Map of rule IDs → `RuleConfig` |
| `hooks` | — | Map of hook IDs → `HookConfig` |
| `mcp` | — | Map of MCP server IDs → `MCPConfig` |
| `settings` | — | Full `SettingsConfig` (env, statusLine, enabledPlugins, etc.) |
| `test` | — | `TestConfig` (claude_path, judge_model) |

### AgentConfig fields

`name`, `description`, `instructions`, `instructions_file`, `model`, `effort`, `memory`, `maxTurns`, `tools`, `blocked_tools`, `skills`, `rules`, `mcp`, `assertions`, `targets`

> `instructions` and `instructions_file` are mutually exclusive. Set one or the other, never both.

### SkillConfig fields

`name`, `type`, `description`, `instructions`, `instructions_file`, `tools`, `allowed-tools`, `paths`, `references`

> `references:` is a list of supplementary file paths (relative to scaffold.xcf) compiled into `skills/<id>/references/`.

### RuleConfig fields

`description`, `paths`, `instructions`, `instructions_file`

### SettingsConfig fields

`env` (map), `statusLine` (object: `{type, command}`), `enabledPlugins` (map[string]bool), `alwaysThinkingEnabled`, `effortLevel`, `skipDangerousModePermissionPrompt`, `permissions`, `mcpServers`

> The top-level `mcp:` block is a convenience shorthand for `mcpServers`. During compilation, both are merged — `settings.mcpServers` takes precedence on key conflicts.

See `internal/ast/types.go` for exact struct definitions.


## Dependencies

- **cobra** — CLI framework
- **yaml.v3** — YAML parsing with strict mode
- **testify** — Test assertions
- **wazero** — WebAssembly runtime for token counting (no CGO)

## Environment Variables

- `ANTHROPIC_API_KEY` — Required by `generator` and `judge` packages for API access (alternative: `claude` CLI on PATH)

## Semantic Context & Gotchas

- **Contribution Guard-Rails**: Use simple, non-marketing terminology and maintain strict open-source boundaries.
- **Test Fixture Convention**: When adding a new AST field, you MUST update all `testing/fixtures/*.xcf` files or `yaml.KnownFields(true)` will fail the parser tests. Run `go test ./internal/parser/...` immediately after any `types.go` change.
- **`instructions_file:` path safety**: Validate that `instructions_file` paths are relative and do not contain `..`. Use the same `validateID` pattern from `internal/parser/parser.go`.
- **Compiler base directory**: `compiler.Compile()` receives a `baseDir string` argument (the directory containing `scaffold.xcf`). Always use this to resolve `instructions_file:` paths — never use `os.Getwd()`.
- **Security Invariants**:
  - Do NOT remove `filepath.Clean` references anywhere in the codebase.
  - Do NOT modify `proxy.go` without respecting `http.MaxBytesReader()` boundaries against untrusted AST strings.
  - Command injections are prevented via `filepath.Base` validation in `generator.go`/`judge.go` — leave these untouched.
- **Skills output structure**: Skills compile to `skills/<id>/SKILL.md` (directory), never `skills/<id>.md` (flat). The `apply.go` command uses `os.MkdirAll(filepath.Dir(absPath))` before every write to create nested directories on demand.
