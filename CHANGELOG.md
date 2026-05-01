# Changelog

All notable changes to this project will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- Fixed `xcaffold apply --backup` skipping backup for 2nd and subsequent targets in multi-target projects; backup now runs for every target regardless of source-change detection.

### Added

- Added `xcaffold status` command to replace `xcaffold diff`, providing high-level sync/drift metrics across all applied targets with inline file status reporting.
- Added adaptive 3-column terminal output for `xcaffold list`, intelligently scoping and grouping registered Rules and Memory items natively.

### Changed

- Command `xcaffold graph` overhauled dependency rendering to naturally group rules by folder prefixes and nest active agent memory dynamically.
- `xcaffold diff` is now officially deprecated, safely delegating any active usage directly to `xcaffold status` with migration hints natively.
- **Import pipeline unified on ProviderImporter interface** — `mergeImportDirs` (multi-directory import) now uses the registered `ProviderImporter.Import()` per directory instead of legacy extraction functions. All resource types (agents, skills, rules, workflows, memory, hooks, MCP, settings, project instructions) are now imported in multi-dir mode. Previously, multi-dir imports silently dropped memory, MCP, settings, hooks, and project instructions.

### Removed

- Removed `xcaffold translate` command — cross-provider translation now happens automatically during `xcaffold apply`, and explicit cross-provider import is handled by `xcaffold import --source`. The `internal/translator` package and all workflow lowering logic remain unchanged.
- `xcaffold migrate` command removed — schema version migration infrastructure had no consumers; legacy layout transitions have no external audience
- Removed `buildConfigFromDir` and 10 provider-specific extraction functions from `import.go` (dead code with 0 production callers).
- Removed `extractAgents`, `extractSkills`, `extractRules`, `extractWorkflows` legacy functions (replaced by `ProviderImporter.Import()`).
- Removed unreachable fallback branch in `importScope` (all 5 providers have registered importers).
- Removed duplicate `rendererForTarget` in `apply.go` (consolidated into `compiler.ResolveRenderer`).
- Removed duplicate `detectAllGlobalPlatformDirs` (import.go) and `detectAllPlatformDirs` (init.go), merged into parameterized `detectPlatformDirs`.

### Added (Provider-Agnostic Renderer)

- Added `CapabilitySet` type declaring per-resource support for each renderer (renderer)
- Added `Orchestrate()` function dispatching compilation to per-resource methods based on capability declarations (renderer)
- Added cross-provider invariant test suite asserting render-or-note, no raw aliases, no Claude env var leakage, reference fidelity, and code catalog completeness (renderer)
- Added `provider_features_test.go` with ground truth assertions for all five providers' capability sets, target names, and output directories (renderer)
- Added shared `CompileSkillSubdir`, `SortedKeys`, `YAMLScalar`, `StripAllFrontmatter` helpers (renderer)
- Added `LowerWorkflows` helper in `renderer/shared/` subpackage to avoid import cycles (renderer)
- Added shared `ParseFrontmatter`, `ParseFrontmatterLenient`, `MatchGlob`, `ReadFile`, `AppendUnique` helpers to eliminate duplication across five provider importers (importer)

### Changed (Provider-Agnostic Renderer)

- Relocated project manifest from `./project.xcf` to `.xcaffold/project.xcf` — the manifest is a tool-generated file (compiler/init/import)
- Transitioned `kind: memory` rendering to provider-agnostic system respecting per-provider ground truth: Claude (full render), Gemini & Antigravity (partial render with note), Cursor & Copilot (dropped with note) (compiler/renderer)
- Schema: Updated `ProjectConfig.AgentRefs` to use `AgentManifestEntry` to support deterministic memory linkage within manifest files (schema)
- Changed `TargetRenderer` interface from monolithic `Compile()`/`Render()` to per-resource methods (`CompileAgents`, `CompileSkills`, `CompileRules`, `CompileWorkflows`, `CompileHooks`, `CompileSettings`, `CompileMCP`, `CompileProjectInstructions`) with `Capabilities()` and `Finalize()` hooks (renderer)
- Changed `compiler.Compile()` to use `resolveRenderer()` + `renderer.Orchestrate()` instead of a target switch with direct renderer construction (compiler)
- Changed `compiler.OutputDir()` to return empty string for unknown targets instead of defaulting to `.claude` (compiler)
- Changed `xcaffold apply` to run optimizer required passes (e.g. `flatten-scopes`, `inline-imports`) after compilation and before policy evaluation (apply)
- Changed internal `claudeDir` variable to `projectRoot` across all CLI commands for provider-agnostic path resolution (cmd)

### Added (Schema)

- Added golden manifest reference files in `schema/golden/` exercising every field per resource kind (schema)
- Added CI test validating all golden manifests parse without error (schema)

### Removed (Convention-Based Memory)

- Removed `kind: memory` from parser — memory entries are now plain `.md` files in `xcf/agents/<id>/memory/`, discovered by the compiler at compile time (parser/compiler)
- Removed `MemoryConfig.Instructions`, `MemoryConfig.InstructionsFile`, `MemoryConfig.Inherited` fields — replaced by `MemoryConfig.Content` populated from `.md` file body (ast)
- Removed `MemorySeed.Lifecycle` field from state tracking (state)
- Removed seed-once lifecycle and `--reseed` flag — `apply` always overwrites memory output, matching all other resource kinds (renderer/cli)
- Removed `resolveMemoryBody`, `renderMemoryMarkdown`, `CompileWithPriorSeeds`, `WithReseed` from Claude renderer (renderer)
- Removed `CodeMemorySeedSkipped` and `CodeMemoryBodyEmpty` fidelity codes (renderer)
- Removed `MemoryOptions.Reseed` and `MemoryOptions.PriorHashes` from renderer interface (renderer)
- Removed `memoryDoc` struct and `WriteSplitFiles` memory block — import writes `.md` files directly (cli)
- Removed `xcaffold translate` command and all associated flags (cli)

### Fixed (Parser)

- Fixed missing `case "memory":` in frontmatter body assignment — memory `.xcf` files with frontmatter + body now correctly assign body to Instructions field (parser) — *subsequently removed in Convention-Based Memory migration*

### Fixed (Provider-Agnostic Renderer)

- Enforced path-safe slugification for imported agent-scoped memory files across all renderers to ensure high-fidelity synchronization (compiler)
- Prevented compounding `project_` prefixes during recursive memory import derivations (bir)
- Derived accurate project roots when manifests reside in the nested `.xcaffold/` namespace (validator)

- Fixed model alias resolution for gemini, copilot, and cursor agent rendering — raw aliases like `sonnet-4` are now mapped to provider-specific model identifiers (renderer)
- Fixed antigravity renderer silently dropping agents without emitting a `RENDERER_KIND_UNSUPPORTED` fidelity note (renderer)
- Fixed data loss in copilot `InstructionsFile` rendering and copilot/gemini model resolution (renderer)
- Fixed `graph` command hardcoded `.claude` fallback to use `compiler.OutputDir()` (graph)
- Fixed `diff` command inconsistent target normalization between global and project scope (diff)
- Fixed Copilot MCP config generation layout, correctly emitting standard layout JSON objects out to `.vscode/mcp.json` (renderer)

### Added
- `xcaffold init` automatically generates a self-referential `/xcaffold` skill (`xcf/skills/xcaffold.xcf`) out of the box, teaching AI assistants local schema constraints and provider support matrices natively.
- `xcaffold init` multi-file generator that scaffolds an entire `xcf/` directory, replacing the legacy single `project.xcf` builder.
- `xcaffold init` `--target` string slice flag and multi-select UI prompt for concurrent platform targeting (`claude`, `cursor`, `antigravity`, etc).
- `xcaffold init` `--no-policies` flag to skip starter policy generation.
- `xcaffold init` `--json` manifest mode for machine-readable output tailored for autonomous agent execution.
- `internal/templates` provider matrix renderer emitting exact field support tables for selected compilation targets.
- `internal/importer`: ProviderImporter interface with per-provider implementations for claude, cursor, gemini, copilot, and antigravity
- `ast.XcaffoldConfig.ProviderExtras`: genuinely-unclassified file catchall for provider-specific artifacts
- `SourceProvider` annotation field on all AST resource types for import provenance tracking
- `parser.ReclassifyExtras`: auto-graduates ProviderExtras files when importers recognize them
- Apply-time fidelity notes for cross-provider extras that cannot be translated
- `importer.KindHookScript` and canonical routing mapping `hooks/**` retaining raw hook script files dynamically across Claude, Cursor, Copilot and Gemini providers (importer)

### Fixed
- `xcaffold status` command now exits with code 1 when drift is detected (artifact drift or source drift), enabling scriptable drift checks in CI/CD. Previously exited with code 0 even when drift was present (cli/status)
- Copilot renderer path-doubling bug: OutputDir() now returns ".github" and all emitted file paths are relative
- Fixed leakage of global-scope agent memory files during `xcaffold import` by pruning orphaned files not explicitly owned by declared project agents (cli/import)

### Removed

- **agentsmd renderer**: The `--target agentsmd` compilation target has been removed. AGENTS.md is an open standard for project instructions, not a provider. Cursor (`--target cursor`) and Copilot (`--target copilot`) generate AGENTS.md files via their own instruction renderers.

### Added

- **Gemini CLI renderer**: `--target gemini` compiles all resource kinds to Gemini CLI native format — instructions to `GEMINI.md`, rules to `.gemini/rules/`, skills to `.gemini/skills/`, agents to `.gemini/agents/`, hooks and MCP to `.gemini/settings.json`.

### Added (FidelityNote Return Surface)

- Added `renderer.FidelityNote` struct and `FidelityLevel` (`info` / `warning` / `error`) for structured, machine-readable fidelity reporting (renderer)
- Added `renderer.NewNote()` constructor and a stable code catalog in `internal/renderer/fidelity_codes.go` covering 16 codes including `SKILL_SCRIPTS_DROPPED`, `SKILL_ASSETS_DROPPED`, `SETTINGS_FIELD_UNSUPPORTED`, `AGENT_MODEL_UNMAPPED`, `AGENT_SECURITY_FIELDS_DROPPED`, and `HOOK_INTERPOLATION_REQUIRES_ENV_SYNTAX` (renderer)
- Added `renderer.AllCodes()` enumeration for tooling that needs to introspect known codes (renderer)
- Added `cmd/xcaffold/fidelity.go` with `printFidelityNotes()` for human-readable output and `buildSuppressedResourcesMap()` for applying per-resource suppression at the command layer (cmd)
- Added propagation test verifying fidelity notes flow from renderer through `compiler.Compile` to the caller (compiler)

### Changed (FidelityNote Return Surface)

- Changed `compiler.Compile` signature to `(*Output, []FidelityNote, error)`; the second return carries fidelity notes for the selected target (compiler)
- Changed `TargetRenderer` interface to return notes from `Compile`, consolidating around the real semantic entry point rather than the thin `Render` wrapper (renderer)
- Changed the cursor renderer to replace 12 stderr writes with typed notes (renderer/cursor)
- Changed the antigravity renderer to replace 4 stderr writes with typed notes and added scripts/assets coverage for parity with cursor (renderer/antigravity)
- Changed the agentsmd renderer to replace the package-level `warningWriter` and `warnLossy*` helpers with `collectNotes{Agent,Skill,Rule}` functions returning notes (renderer/agentsmd)
- Moved `suppress-fidelity-warnings` enforcement out of every renderer and into `cmd/xcaffold/fidelity.go`; renderers now emit notes unconditionally and the command layer filters them (cmd, renderer)
- Updated `xcaffold apply`, `xcaffold export`, and `xcaffold validate` to receive fidelity notes from the compiler and print them via the shared helper (cmd)

### Added (Agent Schema Normalization)

- Added `disable-model-invocation` (`*bool`) and `user-invocable` (`*bool`) fields to `AgentConfig` (ast)
- Added `provider` pass-through map (`map[string]any`) to `TargetOverride` for carrying provider-native fields such as `temperature`, `timeout_mins`, `kind`, `target`, `metadata` (ast)
- Added `--no-references` flag to `xcaffold init` to skip generation of reference template files (init)
- Added `xcf/references/agent.xcf.reference` generation during `xcaffold init` — an annotated, non-parsed field catalog for the Agent kind (init)
- Added `internal/templates.RenderAgentReference()` for rendering the Agent kind reference template (templates)
- Added real-data integration tests validating agent schema round-tripping against provider fixtures (integration)

### Changed (Agent Schema Normalization)

- Reordered `AgentConfig` struct fields so compiled output emits them in a canonical order: identity, then model and execution, tool access, permissions and invocation, lifecycle, memory and context, composition references, inline composition, targets, and instructions last (ast)
- Claude renderer now emits `disable-model-invocation` and `user-invocable` in agent frontmatter when set (renderer)
- Claude renderer now emits `memory` after `isolation` (before `color`) to match the canonical order (renderer)
- Reordered agent field emission in `rest-api`, `cli-tool`, and `frontend-app` init templates so `instructions` appears last (templates)
- Reordered agent document emitted by `xcaffold init` so `instructions` appears after `tools`, matching the canonical order (init)
- Added inline comment in generated `project.xcf` pointing users to `xcf/references/agent.xcf.reference` for the full field catalog (init)

### Added (Skill Schema Normalization)

- Added `whenToUse` (`string`) field to `SkillConfig` for detailed activation guidance (ast)
- Added `license` (`string`) field to `SkillConfig` for SPDX identifier (ast)
- Added `disableModelInvocation` (`*bool`) field to `SkillConfig` — if true, the skill is user-invocable only (ast)
- Added `userInvocable` (`*bool`) field to `SkillConfig` — if false, the skill is model-only with no slash command (ast)
- Added `argumentHint` (`string`) field to `SkillConfig` for slash-command autocomplete (ast)
- Added `targets` (`map[string]TargetOverride`) field to `SkillConfig` for per-provider overrides and provider pass-through (ast)
- Added `xcf/references/skill.xcf.reference` generation during `xcaffold init` — an annotated, non-parsed field catalog for the Skill kind (init)
- Added `internal/templates.RenderSkillReference()` for rendering the Skill kind reference template (templates)
- Added Claude provider pass-through for skills — keys under `targets.claude.provider:` (`context`, `agent`, `model`, `effort`, `shell`, `paths`, `hooks`) are emitted into compiled SKILL.md frontmatter (renderer)
- Added real-data integration tests validating skill schema round-tripping against provider fixtures (integration)

### Changed (Skill Schema Normalization)

- Renamed `SkillConfig.Tools` to `SkillConfig.AllowedTools` with canonical YAML key `allowed-tools`, aligning with the agentskills.io open standard and the Claude and Copilot published conventions (ast, renderer)
- Reordered `SkillConfig` struct fields into the canonical six-group layout: identity, tool access, permissions and invocation, composition files, targets, and instructions last (ast)
- Claude renderer now emits `when_to_use`, `license`, `allowed-tools`, `disable-model-invocation`, `user-invocable`, and `argument-hint` in skill frontmatter when set (renderer)
- All skill provider pass-through scalars now route through yaml.Marshal for correct escaping — previously vulnerable values containing newlines could have terminated the frontmatter block (renderer)
- Broadened the attribute resolver regex to accept kebab-case field names in resource references like `${skill.tdd.allowed-tools}` (resolver)
- Updated the shipped multi-kind reference example and schema documentation to use `allowed-tools` under the skill block (docs)

### Breaking Changes

- **Removed `kind: config`**: The legacy monolithic format has been removed. Use `kind: project` with individual resource documents (`kind: agent`, `kind: skill`, etc.). For global configuration, use `kind: global`. Files with empty or missing `kind:` fields now produce a descriptive error with migration guidance.
- **Renamed `tools:` to `allowed-tools:` under `kind: skill`**: Any `.xcf` file using `tools:` under a skill block must rename to `allowed-tools:` to parse successfully. `AgentConfig.tools` is unchanged — the rename applies only to skills. This aligns with the cross-provider canonical name from the agentskills.io open standard.

### Added

- **`kind: global`**: New kind for `~/.xcaffold/global.xcf`. Contains shared resources and settings without project metadata.
- **`kind: policy`**: Declarative constraint engine. Define `require` and `deny` rules evaluated during `apply` and `validate`. Four built-in policies ship with the binary: `path-safety`, `settings-schema`, `agent-has-description`, `no-empty-skills`.
- **Policy references in `kind: project`**: Projects can reference policies via `policies:` list, same as agents, skills, and rules.
- **Built-in policy overrides**: Create a `kind: policy` file with the same `name` as a built-in and set `severity: off` to disable it.

### Changed

- `isConfigFile()` renamed to `isParseableFile()` — now rejects empty and `config` kind values.
- `WriteSplitFiles()` emits each resource as a separate file with frontmatter format for body-bearing kinds.
- `~/.xcaffold/global.xcf` now uses `kind: global` instead of `kind: config`.

### Changed
- Refactored `README.md` "Why xcaffold?" section with a provider-agnostic ecosystem narrative, removing inaccurate "token budgeting" claims in favor of policy enforcement and agent topology visibility (docs)
- Updated Homebrew and Scoop package descriptions in `.goreleaser.yaml` to reflect provider-agnostic agent configuration positioning (release)
- Standardized Diátaxis `index.md` files across `docs/` with unified cross-navigation "Next Steps" sections (docs)
- Populated empty information-oriented reference index and created a dedicated `examples/README.md` for proper IDE Markdown parsing (docs)
- Replaced `--scope global|project|all` flag with `--global / -g` boolean flag across all commands (cli)
- Changed `validate` command to accept `--global` for validating `~/.xcaffold/global.xcf` (cli)
- Changed global config template to omit `project:` block (registry)
- Rewrote `xcaffold test` to send the compiled agent system prompt directly to the LLM API via `internal/llmclient` instead of spawning a CLI subprocess through an HTTP intercept proxy; trace records declared tool calls extracted from the response (test)
- `xcaffold test` now reads the task from `test.task` in `project.xcf`; defaults to a capabilities-description prompt if unset (test)
- `graph --format json` now uses snake_case field names (`config_path`, `disk_entries`, `blocked_tools`) — breaking change for JSON consumers (graph)
- `import --global` now scans all provider directories (`~/.claude/`, `~/.cursor/`, `~/.agents/`) and merges all discovered resources into `global.xcf` (import)

### Added
- Added `--all` flag to `graph` command for combined global and registered projects view (graph)
- Added hooks and workflows to `graph` topology output (graph)
- Added `task` and `max_turns` fields to `TestConfig` (schema `project.test`) (ast)
- Extended `review project.xcf` to display skills, rules, hooks, MCP servers, and workflows in addition to agents (review)
- Updated `knownTools` validation to include `Task`, `Computer`, `AskUserQuestion`, `Agent`, `ExitPlanMode`, and `EnterPlanMode` (parser)

### Fixed
- `analyze` no longer errors when no `project.xcf` exists in the current directory (analyze)
- `export --output` flag now correctly sets the destination path (export)
- `init --global` no longer fails when a local `project.xcf` is present (init)
- `apply --check` now returns a non-zero exit code when validation errors are found (apply)
- `apply --check-permissions --global` now reads the global config directory instead of the project directory (apply)
- `diff` now surfaces `FindXCFFiles` errors instead of reporting false-positive `SRC DELETED` for valid source files (diff)
- `apply` excludes `registry.xcf` from source file tracking, preventing unnecessary recompilation on every run (apply)
- `graph` no longer includes inherited global resources in project-scope topology output (graph)

### Removed
- Removed `plan` command — use `apply --dry-run` instead (cli)
- Removed `--scope all` compilation mode (cli)
- Removed `internal/mascot` package (unused terminal animation) (internal)
- Removed `renderer.Register()`, `renderer.Get()`, and `renderer.Registered()` dead-code functions (renderer)
- Removed `bir.Analyze()` unused function (bir)

### Added
- Smart Compilation Skipping: `xcaffold apply` tracks multi-file source hashing to skip redundant compilation automatically.
- Deterministic Orphan Purge: `xcaffold apply` identifies and silently prunes missing artifacts to prevent config bloat, supporting `--dry-run` previews natively.
- Legacy Lock Migration: `xcaffold apply` seamlessly upgrades older V1 lock files format mapping targets automatically.
- Source File Drift Tracking: `xcaffold diff` explicitly reports modifications within `project.xcf` dependencies indicating required compilation cycles.
- Parser `ParseDirectory` API: Programmatic support for parsing and merging multiple `.xcf` files within a directory hierarchy, skipping hidden/nested repositories.
- Hardened global inheritance parser `resolveExtendsGlobal` resolving `~/.xcaffold/` first with strict circular dependency traversal detection.
- File-origin error reporting: Duplicate resource IDs (Agents, Skills, Rules, Workflows, MCPs) declared across multiple configuration files now report precise file locations in strict merge conflicts.
- Centralized architecture: `~/.xcaffold/` is the global home for user preferences, project registry, and global agent resources.
- `global.xcf` magical bootstrapping: CLI automatically runs `EnsureGlobalHome()`, migrating your legacy `~/.claude/global.xcf` entirely seamlessly or safely initialises boilerplate without demanding explicit `--scope global` setup.
- Provider SDK registry: Added extensible `platformProvider` interface and multi-platform scanner to deduplicate global discoveries across Claude, Antigravity, and Cursor.
- Internal registry metadata files standardized to `.xcf` (`registry.xcf`, `settings.xcf`).
- Fleet auto-registration: `xcaffold init`, `xcaffold import`, and `xcaffold apply` now automatically detect your scope and auto-register cloned projects into your global registry.
- `xcaffold list` command displays all managed projects with path, targets, resource counts, and last-applied timestamp.
- `xcaffold graph --project <name>` queries any registered project's topology from any location.
- `xcaffold apply` safely resolves project paths from the global registry when invoked using `--project <name>`.
- `xcaffold plan` command for static parsing and pre-deployment execution dry-runs.
- Reference-in-place import: `xcaffold import` generates `project.xcf` entries pointing to existing instruction files without duplication.
- `xcaffold import` natively extracts `hooks.json` mapping parameters and workflow assets directly into the merged definitions.
- Walk-up configuration search: CLI commands work from project subdirectories by walking up to find the nearest `project.xcf` (bounded by `$HOME`).
- Semantic Translation Engine: cross-platform agent capabilities decomposed via static intent heuristics, accessible through `xcaffold import --source`.
- `xcaffold test` execution flag `--claude-path` renamed to `--cli-path` to support fallback binary resolution for `cursor` or other detected proxies.
- `xcaffold apply` safeguards: integrated drift-detection mechanism natively blocks overwrites to locally mutated unrecorded output files.
- `xcaffold apply` overrides: included `--force` flag for drift circumvention and `--backup` flag utilizing localized timestamped clones.
- `xcaffold apply` now supports the `--check` flag to perform fail-fast schema syntax validation without creating artifacts.
- Multi-target compilation support: CLI commands (`apply`, `import`) now support a `--target` flag (`claude`, `cursor`, `antigravity`) to isolate platform outputs.
- `TargetRenderer` Registry: Pluggable compiler architecture delegating to platform-specific layout generation.
- Full compiler surface: `xcaffold apply` now emits `.claude/skills/*.md`, `.claude/rules/*.md`, `.claude/hooks.json`, and `.claude/settings.json` (with MCP) in addition to agents.
- `xcaffold graph` command with deep hierarchical topology visualization (segments global components, natively renders blocked/allowed tools, and separates inherited skills from rules automatically).
- `instructions-file:` directive across agents, skills, and rules to allow sourcing prompts from external markdown files.
- `references:` directive for skills to support copying supplementary context files (supports glob patterns).
- GoReleaser configuration — pre-built release binaries for Linux (amd64/arm64), macOS (amd64/arm64), and Windows (amd64). Homebrew tap formula included.
- `AGENTS.md` — universal agent instruction file following the [agents.txt](https://agentstext.com) convention.
- `llms.txt` — AI discovery index at repository root.
- `.github/` — workspace-specific AI coding context files.
- `docs/architecture.md` — system architecture documentation with Mermaid diagrams.
- Shared `internal/auth` package — eliminates `AuthMode` type duplication between `judge` and `generator` packages.
- `make install` target added to `Makefile` with dynamic `LDFLAGS` injection for version propagation.

### Changed
- Lockfile standardization: state hashes are now enforced under explicit output conventions globally (`scaffold.claude.lock`, `scaffold.cursor.lock`).
- Command Consolidation: The `translate` and `validate` workflows were absorbed into their logical primary operations (`import` and `apply` respectively) to reduce the CLI verb surface.
- Platform neutral scopes: the internal `globalClaudeDir` has been renamed to `globalXcfHome`, aligning `xcaffold init` multi-platform detection for native Claude, Cursor, and Antigravity defaults.
- README rewritten with badge row, "Why xcaffold?" section, Homebrew install target, expanded schema documentation, and multi-platform output tables.
- `xcaffold analyze` now references `auth.AuthModeSubscription` from the shared auth package.
### Fixed
- Fixed `xcaffold init` generating stale `version: "1.0"` templates, and fixed inner `agents:` struct indentation to correctly fall under the `project:` scope (cli)
- Fixed schema versions and YAML structure in `README.md` examples and `project.xcf` (docs)
- Fixed unmapped `model` declarations failing string resolution in native `settings.json` renderer loops.
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
- Token estimation feature (`--tokens` flag on `xcaffold graph`) — cross-provider accuracy is not feasible with a single byte-count heuristic.

## [1.0.0-dev] - 2026-04-02
### Added
- Complete rewrite of the CLI compiler replacing the deprecated TypeScript prototype with a robust Go binary.
- One-Way Compilation architecture targeting Anthropic Claude Code configurations natively.
- Automatic creation and formatting of `.claude/agents/*.md` and `.claude/settings.json`.
- `.xcaffold/project.xcf.state` manifest generation tracking SHA-256 state blobs of output configurations.
- `xcaffold plan` command for static parsing and pre-deployment analysis.
- `xcaffold diff` command to enforce GitOps strictness and identify shadow configuration modifications (drift).
- Support for `tools`, `skills`, `blocked_tools`, `effort`, `model`, and `mcp` declarations within `project.xcf`.

### Removed
- Support for multi-provider prompt polyfilling has been explicitly removed in V1 in favor of the strict native ecosystem.
- Support for Bi-Directional Compilation (Decompilation of `.claude/` files back to `.xcf`).

### Security
- Replaced ambiguous degradation warnings with a fail-closed schema validator (`exit 1`) to ensure security rules are not bypassed during configuration generation.
