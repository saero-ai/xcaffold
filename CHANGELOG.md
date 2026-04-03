# Changelog

All notable changes to this project will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Added
- Full compiler surface: `xcaffold apply` now emits `.claude/skills/*.md`, `.claude/rules/*.md`, `.claude/hooks.json`, and `.claude/settings.json` (with MCP) in addition to agents.
- `plan.json` file output — `xcaffold plan` now writes a structured JSON token budget report to disk in addition to stdout.
- GoReleaser configuration — pre-built release binaries for Linux (amd64/arm64), macOS (amd64/arm64), and Windows (amd64). Homebrew tap formula included.
- `AGENTS.md` — universal agent instruction file following the [agents.txt](https://agentstext.com) convention.
- `llms.txt` — AI discovery index at repository root.
- `.github/copilot-instructions.md` — workspace-specific Copilot context.
- `docs/architecture.md` — system architecture documentation with Mermaid diagrams.
- Shared `internal/auth` package — eliminates `AuthMode` type duplication between `judge` and `generator` packages.

### Changed
- README rewritten with badge row, "Why xcaffold?" section, Homebrew install target, and expanded full-schema reference covering all compiler outputs.
- `xcaffold analyze` now references `auth.AuthModeSubscription` from the shared auth package.
- Token estimation explicitly documented as a `÷4` byte-count heuristic approximation with accuracy bounds.

### Fixed
- Compiler now emits all schema blocks. Previously, `skills`, `rules`, `hooks`, and `mcp` were silently discarded.
- `trace.Recorder` data race — added `sync.Mutex` to protect concurrent writes from HTTP handler goroutines.
- SSRF in `internal/proxy` — replaced `strings.HasSuffix` host check with strict equality, preventing `evil-api.anthropic.com` bypass.
- `os.Exit(1)` in `diff.go` and `validate.go` replaced with `return fmt.Errorf(...)` to allow Cobra to handle exit codes and deferred cleanup.
- CI `go-version` aligned to `1.24` to match `go.mod` declaration.

### Removed
- `wazero` WASM runtime — the `wasmBytecode` embed was always nil (no `//go:embed` directive), making the runtime initialization dead code. Removed from `go.mod` and `go.sum`.
- `golang.org/x/sys` transitive dependency (was pulled in by `wazero`).

## [1.0.0-dev] - 2026-04-02
### Added
- Complete rewrite of the CLI compiler replacing the deprecated TypeScript prototype with a robust Go binary.
- One-Way Compilation architecture targeting Anthropic Claude Code configurations natively.
- Automatic creation and formatting of `.claude/agents/*.md` and `.claude/settings.json`.
- `scaffold.lock` manifest generation tracking SHA-256 state blobs of output configurations.
- `xcaffold plan` command for static parsing and pre-deployment analysis.
- `xcaffold diff` command to enforce GitOps strictness and identify shadow configuration modifications (drift).
- Support for `tools`, `skills`, `blocked_tools`, `effort`, `model`, and `mcp` declarations within `scaffold.xcf`.

### Removed
- Support for multi-provider prompt polyfilling (Copilot, Cursor) has been explicitly removed in V1 in favor of the strict Claude-only ecosystem.
- Support for Bi-Directional Compilation (Decompilation of `.claude/` files back to `.xcf`).

### Security
- Replaced ambiguous degradation warnings with a fail-closed schema validator (`exit 1`) to ensure security rules are not bypassed during configuration generation.
