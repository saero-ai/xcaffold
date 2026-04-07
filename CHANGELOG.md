# Changelog

All notable changes to this project will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Added
- Centralized architecture: `~/.xcaffold/` is the global home for user preferences, project registry, and global agent resources.
- `global.xcf` magical bootstrapping: CLI automatically runs `EnsureGlobalHome()`, migrating your legacy `~/.claude/global.xcf` entirely seamlessly or safely initialises boilerplate without demanding explicit `--scope global` setup.
- Provider SDK registry: Added extensible `platformProvider` interface and multi-platform scanner to deduplicate global discoveries across Claude, Antigravity, and Cursor.
- Internal registry metadata files standardized to `.xcf` (`registry.xcf`, `settings.xcf`).
- Fleet auto-registration: `xcaffold init`, `xcaffold import`, and `xcaffold apply` now automatically detect your scope and auto-register cloned projects into your global registry.
- `xcaffold list` command displays all managed projects with path, targets, resource counts, and last-applied timestamp.
- `xcaffold graph --project <name>` queries any registered project's topology from any location.
- `xcaffold migrate` command detects legacy flat-layout projects, upgrades them to reference-in-place paths (including skills, rules, and references), and registers them in the central registry.
- Reference-in-place import: `xcaffold import` generates `scaffold.xcf` entries pointing to existing instruction files without duplication.
- Walk-up configuration search: CLI commands work from project subdirectories by walking up to find the nearest `scaffold.xcf` (bounded by `$HOME`).
- Semantic Translation Engine: cross-platform agent capabilities decomposed via static intent heuristics, accessible through `xcaffold import --source`.
- `xcaffold test` execution flag `--claude-path` renamed to `--cli-path` to support fallback binary resolution for `cursor` or other detected proxies.
- `xcaffold apply` now supports the `--check` flag to perform fail-fast schema syntax validation without creating artifacts.
- `xcaffold graph` now supports the `--tokens` flag, calculating abstract token weights from node AST depth, unifying prior `plan` analysis into the visualization topology.
- Multi-target compilation support: CLI commands (`apply`, `import`) now support a `--target` flag (`claude`, `cursor`, `antigravity`) to isolate platform outputs.
- `TargetRenderer` Registry: Pluggable compiler architecture delegating to platform-specific layout generation.
- Full compiler surface: `xcaffold apply` now emits `.claude/skills/*.md`, `.claude/rules/*.md`, `.claude/hooks.json`, and `.claude/settings.json` (with MCP) in addition to agents.
- `xcaffold graph` command with deep hierarchical topology visualization (segments global components, natively renders blocked/allowed tools, and separates inherited skills from rules automatically).
- `instructions_file:` directive across agents, skills, and rules to allow sourcing prompts from external markdown files.
- `references:` directive for skills to support copying supplementary context files (supports glob patterns).
- `plan.json` file output — `xcaffold plan` now writes a structured JSON token budget report to disk in addition to stdout.
- GoReleaser configuration — pre-built release binaries for Linux (amd64/arm64), macOS (amd64/arm64), and Windows (amd64). Homebrew tap formula included.
- `AGENTS.md` — universal agent instruction file following the [agents.txt](https://agentstext.com) convention.
- `llms.txt` — AI discovery index at repository root.
- `.github/copilot-instructions.md` — workspace-specific Copilot context.
- `docs/architecture.md` — system architecture documentation with Mermaid diagrams.
- Shared `internal/auth` package — eliminates `AuthMode` type duplication between `judge` and `generator` packages.
- `make install` target added to `Makefile` with dynamic `LDFLAGS` injection for version propagation.

### Changed
- Command Consolidation: The `translate`, `plan`, and `validate` workflows were absorbed into their logical primary operations (`import`, `graph`, and `apply` respectively) to reduce the CLI verb surface.
- Platform neutral scopes: the internal `globalClaudeDir` has been renamed to `globalXcfHome`, aligning `xcaffold init` multi-platform detection for native Claude, Cursor, and Antigravity defaults.
- README rewritten with badge row, "Why xcaffold?" section, Homebrew install target, expanded schema documentation, and multi-platform output tables.
- `xcaffold analyze` now references `auth.AuthModeSubscription` from the shared auth package.
- Token estimation explicitly documented as a `÷4` byte-count heuristic approximation with accuracy bounds.

### Fixed
- Compiler now emits all schema blocks. Previously, `skills`, `rules`, `hooks`, and `mcp` were silently discarded.
- `xcaffold import` completely refactored to be highly faithful, dynamically discovering and preserving external file structures.
- Settings structure type limitations fixed: `statusLine` and `enabledPlugins` are now strictly typed structures instead of untyped maps.
- `trace.Recorder` data race — added `sync.Mutex` to protect concurrent writes from HTTP handler goroutines.
- SSRF in `internal/proxy` — replaced `strings.HasSuffix` host check with strict equality, preventing `evil-api.anthropic.com` bypass.
- `os.Exit(1)` in `diff.go` and `validate.go` replaced with `return fmt.Errorf(...)` to allow Cobra to handle exit codes and deferred cleanup.
- CI `go-version` aligned to `1.24` to match `go.mod` declaration.

### Removed
- Top-level CLI commands `xcaffold translate`, `xcaffold plan`, and `xcaffold validate` were deprecated and removed entirely in favor of flag-driven behaviors on `import`, `graph`, and `apply`.
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
- Support for multi-provider prompt polyfilling has been explicitly removed in V1 in favor of the strict native ecosystem.
- Support for Bi-Directional Compilation (Decompilation of `.claude/` files back to `.xcf`).

### Security
- Replaced ambiguous degradation warnings with a fail-closed schema validator (`exit 1`) to ensure security rules are not bypassed during configuration generation.
