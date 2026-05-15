# Changelog

All notable changes to this project will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.1](https://github.com/saero-ai/xcaffold/compare/v0.4.0...v0.4.1) (2026-05-15)


### Bug Fixes

* **ci:** trigger release on GitHub release events ([53675ef](https://github.com/saero-ai/xcaffold/commit/53675ef87228b00e5b58f8f7f98ff224b3cba5f9))
* **importer:** body-priority base selection and import pipeline fixes ([72eec68](https://github.com/saero-ai/xcaffold/commit/72eec685984aeb679f2ffca6ccdd47c782afd742))

## [0.4.0](https://github.com/saero-ai/xcaffold/compare/v0.3.0...v0.4.0) (2026-05-15)


### Features

* **cli:** show conditional status for provider-required optional fields ([e002eaf](https://github.com/saero-ai/xcaffold/commit/e002eaf36eaf81c33709c56ccedcfe179c6b3d49))
* Codex provider, schema enforcement, and blueprint improvements ([0d4e9e5](https://github.com/saero-ai/xcaffold/commit/0d4e9e5d5ac769c5af0fbbf4daf205abf612ff20))
* **parser:** reject agents without description field ([6336c10](https://github.com/saero-ai/xcaffold/commit/6336c10aa98c082b062aaed967f21ae9b1d0736f))
* **schema:** make agent description required and improve help display ([cb8998e](https://github.com/saero-ai/xcaffold/commit/cb8998e22505d21b617a73d3816dc5729c33c5db))
* **schema:** make agent description required at xcaffold level ([57194ca](https://github.com/saero-ai/xcaffold/commit/57194cab9b8c9539fa91fca2686c01711fc0e771))


### Bug Fixes

* **blueprint:** increase max extends depth to 10 ([a284dd0](https://github.com/saero-ai/xcaffold/commit/a284dd07a150eba5a726079f26f54dff0bd4b917))
* **blueprint:** resolve 5 implementation gaps ([dd9dc85](https://github.com/saero-ai/xcaffold/commit/dd9dc850d09a69754ae6ca9a2e29177c16a11d76))
* **blueprint:** use ClearableList for resource selectors ([12576fa](https://github.com/saero-ai/xcaffold/commit/12576fa76082e58bbb1d673961624a478d2cf39a))
* **cli:** unhide blueprint discovery flags ([7261f59](https://github.com/saero-ai/xcaffold/commit/7261f59062089f3336f98d06aacc7bdb55c74a91))
* **renderer:** resolve settings and hooks dynamically ([eac3280](https://github.com/saero-ai/xcaffold/commit/eac3280f61de38ff636b5f99f1aaa0c1bc5bea7a))
* **schema:** address code review findings for agent description ([f1462e8](https://github.com/saero-ai/xcaffold/commit/f1462e83fa4f042d9a50e1b0a01f8c877dc70486))

## [Unreleased]

### Breaking Changes

* **schema:** `description` is now required on `kind: agent` resources — existing `.xcaf` files that omit `description` will fail validation. (parser, schema)

### Features

* **cli:** `xcaffold help --xcaf` now shows `optional*` for fields that are optional at xcaffold level but required by specific providers, with a note explaining which providers require them. (cli)

## [0.3.0](https://github.com/saero-ai/xcaffold/compare/v0.2.0...v0.3.0) (2026-05-14)


### Features

* add Codex provider, schema cleanup, and doc improvements ([4e7c11a](https://github.com/saero-ai/xcaffold/commit/4e7c11ae905af4d893ce39db51f09272f0fb300b))
* **codex:** add Codex provider ([0256987](https://github.com/saero-ai/xcaffold/commit/0256987c3a37619e014f603712ff8ea78012a1ba))
* **codex:** add Codex provider core ([1137631](https://github.com/saero-ai/xcaffold/commit/1137631bb81ba92c4c3742fc087bdbb39678eb25))
* **codex:** add renderer unit tests ([21250cc](https://github.com/saero-ai/xcaffold/commit/21250cc5f7d2a4cde591a3807691c37ac6011593))


### Bug Fixes

* **ci:** stabilize lint config and clean up unused version vars ([6dcb5d8](https://github.com/saero-ai/xcaffold/commit/6dcb5d8279f38eead00e19d4aed529df1513d935))
* **ci:** use default govet analyzers and tune lint thresholds ([46876cd](https://github.com/saero-ai/xcaffold/commit/46876cdf0f55463c5726816593f1d22b8dd00c14))
* **cli:** simplify version output to match industry standard ([d2a6c5e](https://github.com/saero-ai/xcaffold/commit/d2a6c5ee0e91fb138ad8c0c8be11afcc3f29c08c))
* **schema:** remove unnecessary --- delimiters from pure YAML kinds ([fd0f384](https://github.com/saero-ai/xcaffold/commit/fd0f3843a30c948492fd870b60918c5e8355d764))

## [0.2.0](https://github.com/saero-ai/xcaffold/releases/tag/v0.2.0) (2026-05-14)

### Breaking Changes

- `.xcf` → `.xcaf` file extension — all resource files must be renamed; the parser no longer accepts `.xcf`. (parser)
- `kind: config` removed — use `kind: project` with individual resource documents. Files with an empty or missing `kind:` field now produce a descriptive error with migration guidance. (parser)
- `tools:` renamed to `allowed-tools:` under `kind: skill` — `AgentConfig.tools` is unchanged; the rename applies only to skills. (ast, renderer)
- `xcaffold apply` no longer defaults to `--target claude` when no target is configured — set `targets:` in `project.xcaf` or pass `--target`. (cli)
- `xcaffold init --yes` requires `--target` when no known provider CLI is detected on `$PATH`. (cli)
- `graph --format json` uses snake_case field names (`config_path`, `disk_entries`, `blocked_tools`) — breaks existing JSON consumers. (graph)

### Added

**Providers**

- **Codex provider (Preview)** — compile `.xcaf` manifests to OpenAI Codex output (`.codex/`). Supports agents (TOML), skills (shared `.agents/skills/`), hooks (JSON), MCP (TOML), and project instructions (`AGENTS.md`). Rules and memory unsupported with fidelity notes. ([providers/codex/](providers/codex/))

**Resource kinds**

- `kind: global` — resource kind for `~/.xcaffold/global.xcaf`; holds shared resources and settings without project metadata. (ast, parser)
- `kind: policy` — declarative constraint engine with `require` and `deny` rules evaluated during `apply` and `validate`. Four built-in policies ship with the binary: `path-safety`, `settings-schema`, `agent-has-description`, `no-empty-skills`. Projects reference policies via a `policies:` list in `kind: project`. Create a same-name `kind: policy` file with `severity: off` to disable a built-in. (ast, compiler, policy)
- `kind: context` — shared prompt context blocks composable into agents and blueprints; defined in `xcaf/contexts/`. (ast, parser, renderer)

**Schema features**

- `targets` field on `kind: blueprint` — blueprints can declare independent compilation targets. (ast)
- `ClearableList` type for list fields — setting a list field to `[]` explicitly clears inherited values; absent continues to inherit, empty clears, populated replaces. (ast)
- Two-layer field classification with `+xcaf:role=` markers on all config struct fields (`identity`, `rendering`, `composition`, `metadata`, `filtering`). (ast)
- `disable-model-invocation` (`*bool`) and `user-invocable` (`*bool`) on `AgentConfig`. (ast)
- `provider` pass-through map (`map[string]any`) on `TargetOverride` for provider-native fields. (ast)
- `whenToUse`, `license`, `disableModelInvocation` (`*bool`), `userInvocable` (`*bool`), `argumentHint` on `SkillConfig`. (ast)
- `targets` (`map[string]TargetOverride`) on `SkillConfig` for per-provider overrides and provider pass-through. (ast)
- `AllowedEnvVars` on `ProjectConfig` — security filtering for env var injection via `${env.NAME}`. (ast)
- `task` and `max_turns` fields on `TestConfig` (schema `project.test`). (ast)

**Variable resolution system**

- `--var-file` flag on `apply`, `validate`, and related commands. (cli)
- Variable expansion in `.xcaf` files: `${var.name}` for project variables, `${env.NAME}` for environment variables. (parser, compiler)
- Variable stack loading from `project.xcaf`, `vars.xcaf`, and `--var-file` sources. (compiler)

**CLI commands and flags**

- `xcaffold status` command — sync and drift metrics across all applied targets with inline file status reporting, replacing `xcaffold diff`. (cli)
- `xcaffold list` — adaptive 3-column output displaying all managed projects with path, targets, resource counts, and last-applied timestamp. (cli)
- `xcaffold graph` — deep hierarchical topology visualization; segments global components, renders blocked and allowed tools, separates inherited skills from rules. (cli)
- `xcaffold graph --project <name>` — queries any registered project's topology from any location. (cli)
- `xcaffold graph --all` — combined global and registered projects view. (cli)
- `xcaffold help <kind>` — shows per-provider field annotations for a resource kind and generates annotated templates. (cli)
- `--target <provider>` flag on `xcaffold validate` — compile-time field validation per provider, with provider name in the header and a field validation summary in the footer. (cli)
- `--blueprint` flag on `xcaffold validate`. (cli)
- `--json` flag on `xcaffold init` — machine-readable manifest output. (cli)
- `--target` string-slice flag on `xcaffold init` — multi-select provider targeting. (cli)
- `--force` and `--backup` flags on `xcaffold apply` — drift circumvention and timestamped backup. (cli)
- `--check` flag on `xcaffold apply` — fail-fast schema validation without writing artifacts. (cli)
- `--global / -g` boolean flag replaces `--scope global|project|all` across all commands. (cli)
- `--target` flag on `apply` and `import` for isolating platform outputs. (cli)
- `xcaffold apply --dry-run` — preview and orphan detection without writing. (cli)
- Idempotent `xcaffold init` — re-running updates rather than overwrites. (cli)
- Incremental `xcaffold import` — imports only new or changed resources. (importer)
- Apply preview — `xcaffold apply` shows a diff preview before writing. (cli)
- `xcaffold import --source` — semantic cross-provider translation during import. (importer)

**Compiler and optimizer**

- Multi-target compilation support. (compiler)
- `TargetRenderer` registry — pluggable compiler architecture; all provider dispatch goes through `resolveRenderer()` + `renderer.Orchestrate()`. (compiler, renderer)
- Smart compilation skipping via multi-file source hashing. (compiler)
- Deterministic orphan purge with `--dry-run` preview. (compiler)
- Walk-up configuration search from project subdirectories, bounded by `$HOME`. (compiler)
- Lockfile standardization with per-target naming (`scaffold.claude.lock`, `scaffold.cursor.lock`). (compiler)
- Skill artifact auto-discovery by compiler. (compiler)
- `xcaffold apply` runs optimizer passes after compilation and before policy evaluation. (compiler)
- Security invariant policies: output path confinement, settings schema, hook URL validation. (policy)

**Renderer — provider-agnostic surface**

- `CapabilitySet` type declaring per-resource support for each renderer. (renderer)
- `Orchestrate()` function dispatching compilation to per-resource methods based on `CapabilitySet`. (renderer)
- `TargetRenderer` per-resource methods: `CompileAgents`, `CompileSkills`, `CompileRules`, `CompileWorkflows`, `CompileHooks`, `CompileSettings`, `CompileMCP`, `CompileProjectInstructions`, plus `Capabilities()` and `Finalize()`. (renderer)
- Cross-provider invariant test suite: render-or-note, no raw aliases, no Claude env var leakage, reference fidelity, code catalog completeness. (renderer)
- `provider_features_test.go` — ground truth assertions for all five providers' capability sets, target names, and output directories. (renderer)
- Shared renderer helpers: `CompileSkillSubdir`, `SortedKeys`, `YAMLScalar`, `StripAllFrontmatter`. (renderer)
- `LowerWorkflows` in `renderer/shared/` to avoid import cycles. (renderer)
- `FidelityNote` struct with FidelityLevel (`info` / `warning` / `error`) and `NewNote()` constructor. (renderer)
- Stable fidelity code catalog in `fidelity_codes.go` — 16 codes including SKILL_SCRIPTS_DROPPED, SKILL_ASSETS_DROPPED, SETTINGS_FIELD_UNSUPPORTED, `AGENT_MODEL_UNMAPPED`, `AGENT_SECURITY_FIELDS_DROPPED`, HOOK_INTERPOLATION_REQUIRES_ENV_SYNTAX. (renderer)
- `AllCodes()` enumeration for tooling introspection. (renderer)
- `cmd/xcaffold/fidelity.go` with `printFidelityNotes()` and `buildSuppressedResourcesMap()` for command-layer suppression. (cmd)
- Antigravity renderer — agents rendered as specialist notes. (renderer)
- Gemini CLI renderer (`--target gemini`) — instructions to `GEMINI.md`, rules to `.gemini/rules/`, skills to `.gemini/skills/`, agents to `.gemini/agents/`, hooks and MCP to `.gemini/settings.json`. (renderer)
- `ProviderManifest` registry — replaces hardcoded provider switches. (renderer)
- gen-schema tooling for `+xcaf:` marker extraction and schema registry. (ast)
- Override parsing expanded to 9 resource kinds. (parser)

**Importer**

- `ProviderImporter` interface with per-provider implementations for claude, cursor, gemini, copilot, and antigravity. (importer)
- `ProviderExtras` catchall for genuinely unclassified provider-specific artifacts. (ast, importer)
- `SourceProvider` annotation on all AST resource types for import provenance tracking. (ast)
- `ReclassifyExtras` — auto-graduates `ProviderExtras` entries when an importer recognizes them. (parser)
- `KindHookScript` and canonical hook-file routing across claude, cursor, copilot, and gemini. (importer)
- Shared importer helpers: `ParseFrontmatter`, `ParseFrontmatterLenient`, `MatchGlob`, `AppendUnique`. (importer)
- `import --global` scans all provider directories and merges all discovered resources. (importer)

**Schema and golden files**

- Golden manifest reference files in `schema/golden/` for every resource kind. (schema)
- CI test validating all golden manifests parse without error. (schema)
- Per-kind reference guides (`agent-reference.md`, `skill-reference.md`, etc.) generated inside the xcaffold skill during `xcaffold init`. (init)

**Other**

- `xcaffold init` generates a self-referential `/xcaffold` skill (`xcaf/skills/xcaffold/skill.xcaf`) teaching AI assistants local schema constraints. (init)
- `xcaffold init` multi-file generator scaffolds a full `xcaf/` directory, replacing the legacy single-file builder. (init)
- `instructions-file:` directive on agents, skills, and rules for sourcing prompts from external markdown files. (ast)
- `references:` directive on skills for copying supplementary context files (glob patterns). (ast)
- Provider override list merge with tri-state `cleared` signal — `cleared: true` empties an inherited list field. (ast)
- Claude provider pass-through for skills — keys under `targets.claude.provider:` emitted into SKILL.md frontmatter. (renderer)
- File-origin error reporting for duplicate resource IDs across multiple `.xcaf` files. (parser)
- Walk-up `EnsureGlobalHome()` migrates or initializes `~/.xcaffold/` automatically on first run. (cli)
- Project auto-registration into global registry on `init`, `import`, and `apply`. (cli)
- `xcaffold apply --project <name>` resolves project paths from the global registry. (cli)
- `hooks` and `workflows` included in `xcaffold graph` topology output. (graph)
- `review project.xcaf` displays skills, rules, hooks, MCP servers, and workflows in addition to agents. (review)
- `knownTools` validation extended with Task, Computer, AskUserQuestion, Agent, ExitPlanMode, EnterPlanMode. (parser)
- GoReleaser — pre-built binaries for Linux (amd64/arm64), macOS (amd64/arm64), Windows (amd64) with Homebrew tap. (release)
- `AGENTS.md` following the [agents.txt](https://agentstext.com) convention. (docs)
- `llms.txt` AI discovery index at repository root. (docs)
- `docs/concepts/architecture/overview.md` — system architecture documentation with Mermaid diagrams. (docs)
- Shared `internal/auth` package eliminating `AuthMode` type duplication. (internal)
- `make install` target with `LDFLAGS` injection for version propagation. (build)

### Changed

- `TargetRenderer` interface: monolithic `Compile()`/`Render()` replaced by per-resource methods with `Capabilities()` and `Finalize()`. (renderer)
- `compiler.Compile()` signature: `(*Output, []FidelityNote, error)` — second return carries fidelity notes. (compiler)
- `compiler.Compile()` uses `resolveRenderer()` + `renderer.Orchestrate()` instead of a direct target switch. (compiler)
- `compiler.OutputDir()` returns empty string for unknown targets instead of `.claude`. (compiler)
- `suppress-fidelity-warnings` enforcement moved from individual renderers to the command layer; renderers emit notes unconditionally. (cmd, renderer)
- `xcaffold apply`, `xcaffold export`, and `xcaffold validate` receive and print fidelity notes via the shared helper. (cmd)
- Cursor renderer: 12 stderr writes replaced with typed fidelity notes. (renderer)
- Antigravity renderer: 4 stderr writes replaced with typed fidelity notes. (renderer)
- `AgentConfig` struct fields reordered to canonical grouping: identity, model and execution, tool access, permissions and invocation, lifecycle, memory and context, composition references, inline composition, targets, instructions last. (ast)
- `SkillConfig` struct fields reordered to canonical six-group layout: identity, tool access, permissions and invocation, composition files, targets, instructions last. (ast)
- Claude renderer emits new agent frontmatter fields: `disable-model-invocation`, `user-invocable`, `memory` (after `isolation`). (renderer)
- Claude renderer emits skill frontmatter fields: `when_to_use`, `license`, `allowed-tools`, `disable-model-invocation`, `user-invocable`, `argument-hint`. (renderer)
- Attribute resolver regex broadened to accept kebab-case field names (e.g. `${skill.tdd.allowed-tools}`). (resolver)
- `fields.yaml` entries reclassified from `xcaffold-only` to `unsupported`. (renderer)
- Parser name/kind mismatch warnings collected in `XcaffoldConfig.ParseWarnings` instead of printing to stderr. (parser)
- `xcaffold apply` output: header breadcrumb, glyph helpers, file count summary, import hint footer. (cli)
- `xcaffold apply` lists each drifted file with path and status before aborting. (cli)
- `xcaffold graph` dependency rendering overhauled — rules grouped by folder prefix, agent memory nested dynamically. (graph)
- Import pipeline unified on `ProviderImporter` interface — multi-directory import now uses `ProviderImporter.Import()` per directory; memory, MCP, settings, hooks, and project instructions no longer dropped in multi-dir mode. (importer)
- `isConfigFile()` renamed to `isParseableFile()` — now rejects empty and `config` kind values. (parser)
- `WriteSplitFiles()` emits separate files with frontmatter for body-bearing kinds. (compiler)
- `~/.xcaffold/global.xcaf` uses `kind: global` instead of `kind: config`. (ast)
- Project manifest relocated from `./project.xcaf` to `.xcaffold/project.xcaf`. (compiler, init, importer)
- `--scope global|project|all` replaced with `--global / -g` boolean flag. (cli)
- `xcaffold test` rewrites compilation to send the compiled system prompt directly to the LLM API via `internal/llmclient`; trace records declared tool calls from the response. `test.task` in `project.xcaf` sets the task prompt. (test)
- `xcaffold test --claude-path` renamed to `--cli-path` for provider-agnostic binary resolution. (cli)
- Memory rendering transitioned to convention-based `.md` files in `xcaf/agents/<id>/memory/` — discovered by the compiler at compile time. (compiler, renderer)
- Lockfile format standardized with per-target naming; V1 lock files upgraded automatically. (compiler)
- `validate --target` includes provider name in header and appends a field validation summary. (cli)
- README rewritten with badge row, "Why xcaffold?" section, Homebrew install target, expanded schema documentation, and multi-platform output tables. (docs)
- Diátaxis `index.md` files standardized with unified cross-navigation sections. (docs)

### Fixed

- `tagResourcesWithProvider` skipping MCP, hooks, and settings during multi-provider import; all 7 resource kinds now receive provider-scoped `targets` entries. (importer)
- `xcaffold apply --backup` skipping backup for 2nd and subsequent targets in multi-target projects. (cli)
- `xcaffold status --all` silently ignored without `--target`; now appends per-provider grouped file listing in overview mode. (cli)
- `xcaffold status` exits with code 1 on drift detection, enabling scriptable CI checks. (cli)
- Copilot renderer path-doubling: `OutputDir()` returns `.github`, all emitted paths are relative. (renderer)
- Global-scope memory file leakage during `xcaffold import` — orphaned files not owned by declared project agents are now pruned. (importer)
- Model alias resolution for gemini, copilot, and cursor — raw aliases like `sonnet-4` now map to provider-specific identifiers. (renderer)
- Antigravity renderer silently dropping agents without emitting a `RENDERER_KIND_UNSUPPORTED` fidelity note. (renderer)
- Copilot `InstructionsFile` rendering and model resolution. (renderer)
- Copilot MCP config layout — correctly emits `.vscode/mcp.json`. (renderer)
- `graph` hardcoded `.claude` fallback replaced with `compiler.OutputDir()`. (graph)
- `graph` excluding inherited global resources from project-scope topology output. (graph)
- `diff` surfacing `FindXCAFFiles` errors instead of reporting false-positive `SRC DELETED`. (diff)
- `apply` excluding `registry.xcaf` from source file tracking. (apply)
- Memory import path-safe slugification and compounding `project_` prefixes during recursive import. (importer)
- Project root derivation with nested `.xcaffold/` namespace. (compiler)
- `analyze` no longer errors when no `project.xcaf` is present. (analyze)
- `export --output` flag correctly sets the destination path. (export)
- `init --global` with a local `project.xcaf` present. (init)
- `apply --check` returns non-zero exit code on validation errors. (apply)
- `apply --check-permissions --global` reads the global config directory. (apply)
- `init` generating stale `version: "1.0"` templates and incorrect `agents:` indentation. (init)
- Schema versions and YAML structure in README examples. (docs)
- Unmapped `model` declarations failing string resolution in `settings.json` renderer. (renderer)
- Compiler silently discarding `skills`, `rules`, `hooks`, and `mcp` blocks. (compiler)
- `statusLine` and `enabledPlugins` strict typing in settings renderer. (renderer)
- `trace.Recorder` data race — added `sync.Mutex` for concurrent HTTP handler writes. (internal)
- SSRF in `internal/proxy` — replaced `strings.HasSuffix` host check with strict equality. (internal)
- `os.Exit(1)` in `diff.go` and `validate.go` replaced with `return fmt.Errorf(...)`. (cli)

### Removed

- `xcaffold plan` command — use `apply --dry-run`. (cli)
- `xcaffold diff` command — replaced by `xcaffold status`. (cli)
- `xcaffold translate` command — translation via `import --source` and cross-provider `apply`. (cli)
- `xcaffold migrate` command — had no consumers. (cli)
- `--target agentsmd` compilation target — AGENTS.md is generated by the cursor and copilot renderers. (renderer)
- `--scope all` compilation mode. (cli)
- `kind: memory` from parser — memory entries are now plain `.md` files discovered by the compiler. (parser)
- `MemoryConfig.Instructions`, `MemoryConfig.InstructionsFile`, `MemoryConfig.Inherited` fields. (ast)
- `MemorySeed.Lifecycle` field and seed-once lifecycle with `--reseed` flag. (state, cli)
- `resolveMemoryBody`, `renderMemoryMarkdown`, `CompileWithPriorSeeds`, `WithReseed` from Claude renderer. (renderer)
- `CodeMemorySeedSkipped` and `CodeMemoryBodyEmpty` fidelity codes. (renderer)
- `MemoryOptions.Reseed` and `MemoryOptions.PriorHashes` from renderer interface. (renderer)
- `memoryDoc` struct and `WriteSplitFiles` memory block. (renderer)
- `internal/mascot` package. (internal)
- `renderer.Register()`, `renderer.Get()`, `renderer.Registered()` dead-code functions. (renderer)
- `bir.Analyze()` unused function. (bir)
- `buildConfigFromDir` and 10 provider-specific extraction functions from `import.go`. (importer)
- `extractAgents`, `extractSkills`, `extractRules`, `extractWorkflows` legacy functions. (importer)
- Unreachable fallback branch in `importScope`. (importer)
- Duplicate `rendererForTarget` in `apply.go`. (compiler)
- Duplicate `detectAllGlobalPlatformDirs` / `detectAllPlatformDirs` merged into parameterized `detectPlatformDirs`. (cli)
- `wazero` WASM runtime and `golang.org/x/sys` transitive dependency. (internal)
- `--tokens` flag on `xcaffold graph`. (graph)

## [0.1.0] - 2026-04-02
### Added
- Complete rewrite of the CLI compiler replacing the deprecated TypeScript prototype with a robust Go binary.
- One-Way Compilation architecture targeting Anthropic Claude Code configurations natively.
- Automatic creation and formatting of `.claude/agents/*.md` and `.claude/settings.json`.
- `.xcaffold/project.xcaf.state` manifest generation tracking SHA-256 state blobs of output configurations.
- `xcaffold plan` command for static parsing and pre-deployment analysis.
- `xcaffold diff` command to enforce GitOps strictness and identify shadow configuration modifications (drift).
- Support for `tools`, `skills`, `blocked_tools`, `effort`, `model`, and `mcp` declarations within `project.xcaf`.

### Removed
- Support for multi-provider prompt polyfilling has been explicitly removed in V1 in favor of the strict native ecosystem.
- Support for Bi-Directional Compilation (Decompilation of `.claude/` files back to `.xcaf`).

### Security
- Replaced ambiguous degradation warnings with a fail-closed schema validator (`exit 1`) to ensure security rules are not bypassed during configuration generation.
