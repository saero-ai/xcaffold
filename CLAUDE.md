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
  → compiler (AST → markdown rendering)
  → state (SHA-256 lock manifest for drift detection)
  → .claude/agents/*.md + scaffold.lock
```

### Key Packages

| Package | Role |
|---------|------|
| `cmd/xcaffold/` | Cobra CLI commands — one file per command (init, analyze, plan, apply, diff, test, review, validate) |
| `internal/ast` | Core types only (`types.go`): `XcaffoldConfig`, `AgentConfig`, `SkillConfig`, `RuleConfig`, `HookConfig`, `MCPConfig`, `TestConfig` |
| `internal/parser` | Strict YAML parsing with `KnownFields(true)` — unknown fields fail immediately |
| `internal/compiler` | Translates AST → `compiler.Output{Files: map[path]content}` — one-way, deterministic |
| `internal/state` | Reads/writes `scaffold.lock` with SHA-256 hashes per artifact |
| `internal/analyzer` | Token counting via wazero WASM runtime; reverse-engineers project context |
| `internal/generator` | Calls Anthropic API to generate scaffold configs; outputs `audit.json` payloads |
| `internal/proxy` | HTTP intercept proxy for sandboxed agent simulation during `xcaffold test` |
| `internal/trace` | JSONL execution trace recording |
| `internal/judge` | LLM-as-a-Judge evaluation against agent assertions |
| `tools/fuzzer` | Fuzz testing harness for parser robustness |

### Design Invariants

- **One-way compilation**: `.xcf` → `.claude/` only; no decompile/import from existing markdown
- **Fail-closed validation**: Parser rejects unknown YAML fields; compiler never panics
- **Path safety**: All output paths go through `filepath.Clean` to prevent traversal
- **Dual auth**: `internal/judge` and `internal/generator` support `ANTHROPIC_API_KEY` env var or `claude` CLI subprocess
- **Deterministic output**: No randomness in compilation; same input always produces same output

## .xcf Schema

The `scaffold.xcf` file has these top-level blocks: `version` (required), `project` (required), `agents`, `skills`, `rules`, `hooks`, `mcp`, `test`. All maps are keyed by ID string. See `internal/ast/types.go` for the full struct definitions.

## Dependencies

- **cobra** — CLI framework
- **yaml.v3** — YAML parsing with strict mode
- **testify** — Test assertions
- **wazero** — WebAssembly runtime for token counting (no CGO)

## Environment Variables

- `ANTHROPIC_API_KEY` — Required by `generator` and `judge` packages for API access (alternative: `claude` CLI on PATH)

## Semantic Context & Gotchas

- **Contribution Guard-Rails**: Follow the 3x Public-Eyes Check. Never commit anything referencing internal SaaS components. Use simple, non-marketing terminology.
- **Test Fixture Convention**: Check `testing/fixtures/trace_pass.jsonl` and `trace_fail_avoidance.jsonl`. When modifying judge evaluation states, supply explicit traces instead of hardcoding structs unless purely atomic.
- **Security Invariants**: 
  - Do NOT remove `filepath.Clean` references anywhere in the codebase.
  - Do NOT modify `proxy.go` without respecting `http.MaxBytesReader()` boundaries against untrusted AST strings.
  - Command injections are prevented via `filepath.Base` validation in `generator.go`/`judge.go` — leave these untouched.
- **Ast Additions**: `yaml.KnownFields(true)` dictates that adding a new AST struct field mandates updating ALL relevant `fixtures/*.xcf` testing files or parsers will error.
