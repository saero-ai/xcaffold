# Contributing to xcaffold

## Ways to Contribute

- **Bug reports** — open a GitHub issue with a minimal reproduction case.
- **Documentation** — fix errors, improve examples, or fill coverage gaps in `docs/`.
- **New providers** — add support for a new output target (see "Adding a New Provider").
- **Core compiler changes** — improvements to the parser, AST, BIR, optimizer, or renderer pipeline.

## Setting Up

Prerequisites: Go 1.26 or later.

```
make setup
```

This installs `golangci-lint` globally and links the pre-commit hook from `scripts/pre-commit.sh`. No other setup is required.

## Reporting Bugs

Open a GitHub issue. Include:

- `xcaffold version` output
- OS and Go version
- Minimal `.xcaf` file that reproduces the issue
- Full error output (stdout and stderr)

## Proposing Features

For significant changes — new providers, AST changes, pipeline modifications — open a GitHub Discussion before writing code. Maintainers will confirm scope before you invest implementation time. For small, well-scoped improvements, a PR with a clear description is sufficient.

## Pull Requests

### Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/) format:

```
feat(renderer): add cursor provider renderer
fix(parser): handle empty skill body
docs(contributing): update provider workflow
test(importer): add roundtrip test for gemini
```


### Changelog

Changelogs are generated automatically by [release-please](https://github.com/googleapis/release-please) from Conventional Commit messages. Do not edit `CHANGELOG.md` manually — your commit messages become the changelog entries. Write clear, user-facing commit descriptions.

For breaking changes, include a `BREAKING CHANGE:` footer in the commit message body.

### PR Checklist

- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] Documentation updated if behavior changed

## Testing

### Running Tests

```
make test       # unit tests
make test-e2e   # end-to-end tests
```

### Writing Tests

Use table-driven tests with `t.Run()`. For tests that register providers, use `providers.SwapRegistryForTest` to isolate the registry — see `providers/registry_test.go` for the pattern. Reference `providers/claude/` test files as examples of renderer and importer test structure.

### Cross-Provider Invariant

`internal/renderer/cross_provider_test.go` enforces that every AST field is either rendered or produces a `FidelityNote` for every registered provider. Every new provider must pass this test. No code change to the test is required if the `CapabilitySet` is correctly declared — correct capability declarations drive automatic `RENDERER_KIND_UNSUPPORTED` emission.

## Documentation

Documentation lives under `docs/` and follows the [Diátaxis](https://diataxis.fr/) framework:

| Pillar | Directory | Contents |
|--------|-----------|----------|
| Tutorials | `docs/tutorials/` | Step-by-step learning paths |
| Best Practices | `docs/best-practices/` | Task-oriented procedures |
| Concepts | `docs/concepts/` | Explanatory background |
| Reference | `docs/reference/` | Commands, kinds, fields |

Every `.md` file must have YAML frontmatter:

```yaml
---
title: "Page Title"
description: "One-sentence summary."
---
```

Do not add reference content to `README.md`. It must remain a lightweight overview with links into `docs/`.

## Adding a New Provider

This is the highest-leverage community contribution. Read this section in full before writing code.

### 1. Open a Discussion First

Open a GitHub issue before writing any code. Include:

- Provider name and the `--target` flag value you propose
- Target output directory (e.g., `.cursor`)
- Which resource kinds the provider natively supports
- Link to the provider's official documentation

Maintainers will confirm scope before you invest implementation time.

### 2. Package Layout

Create a package under `providers/<name>/`:

```
providers/<name>/
    manifest.go          # ProviderManifest + init() registration
    renderer.go          # TargetRenderer implementation
    importer.go          # ProviderImporter implementation
    model_resolver.go    # ModelResolver for alias → ID mapping
    fields.yaml          # Field support declarations per resource kind
    renderer_test.go
    importer_test.go
    testdata/            # Test fixtures
        input/           # Sample provider directory for import tests
```

Reference `providers/claude/` as the canonical example. Do not modify `providers/registry.go` — providers self-register via `init()`.

The `fields.yaml` file declares which AST fields each resource kind supports for this provider. Valid support levels are `required`, `optional`, and `unsupported`. The compiler uses this to validate `.xcaf` manifests against provider capabilities. See `providers/claude/fields.yaml` for the canonical example.

### 3. Implement the Manifest

In `manifest.go`, declare a `providers.ProviderManifest` and call `providers.Register()` from `init()`. See `providers/claude/manifest.go` for the canonical pattern — field names, registration order, and capability declaration are all illustrated there.

The `init()` function must make three registration calls:
```go
func init() {
    providers.Register(Manifest)
    importer.Register(NewImporter())
    renderer.RegisterModelResolver("<name>", NewModelResolver())
}
```

Additionally, add a blank import in `providers/all/all.go`:
```go
_ "github.com/saero-ai/xcaffold/providers/<name>"
```

Key manifest fields to configure:
- `Name`: canonical provider identifier used with `--target`
- `OutputDir`: target directory (e.g., `.cursor`, `.gemini`)
- `ValidNames`: canonical name plus any aliases
- `RequiredPasses`: optimizer passes needed before rendering. Common passes: `flatten-scopes` (merges scope hierarchy), `inline-imports` (resolves file references). Check which passes existing providers with similar characteristics use.
- `DefaultBudget` / `BudgetKind`: recommended token/character budget for project instructions. Use `0`/`""` for unlimited.
- `KindSupport`: map of resource kind names to boolean support flags
- `RootContextFile`: filename for project-level instructions (e.g., `CLAUDE.md`, `GEMINI.md`, `AGENTS.md`)
- `SubdirMap`: maps canonical artifact directories (`references`, `scripts`, `assets`, `examples`) to provider-native names. Empty string means flatten to parent directory.
- `DisplayLabel`: human-readable name for CLI output
- `CLIBinary`: the provider's CLI command name
- `DefaultModel`: suggested default model ID

### 4. Implement the Renderer

Implement the `renderer.TargetRenderer` interface defined in `internal/renderer/renderer.go`. For unsupported kinds, return an empty `map[string]string` and no error — the orchestrator emits `RENDERER_KIND_UNSUPPORTED` automatically based on the `CapabilitySet`. Reuse helpers from `internal/renderer/helpers.go` (`SortedKeys`, `YAMLScalar`, `StripAllFrontmatter`, `FlattenToSkillRoot`) rather than reimplementing them.

See `providers/claude/renderer.go` for the canonical implementation.

The `Capabilities()` method returns a `CapabilitySet` struct that drives the cross-provider invariant tests. Key fields:

- `Agents`, `Skills`, `Rules`, `Workflows`, `Hooks`, `Settings`, `MCP`, `Memory`, `ProjectInstructions`: boolean flags declaring which resource kinds this renderer handles
- `SkillArtifactDirs`: maps canonical artifact names (`references`, `scripts`, `assets`, `examples`) to provider output subdirectory names. Empty string means flatten to the skill root.
- `RuleActivations`: list of activation modes this renderer supports (e.g., `["always", "path-glob"]`)
- `RuleEncoding`: how rule metadata is encoded — `Description` and `Activation` each take `"frontmatter"`, `"prose"`, or `"omit"`
- `AgentNativeToolsOnly`: set `true` only if this provider's native tool vocabulary IS the xcaffold core tool set. Most providers set this to `false`.

The compilation pipeline calls `renderer.Orchestrate()` which dispatches to per-resource `Compile*` methods — you do not need to call them directly.

### 5. Implement the Importer

Implement the `importer.ProviderImporter` interface defined in `internal/importer/importer.go`. Embed `importer.BaseImporter` to satisfy `Provider()` and `InputDir()` without boilerplate. Define `Classify()`, `Extract()`, and `Import()` methods — `Import()` can delegate to `importer.RunImport()` for the standard walk-and-dispatch loop.

See `providers/claude/importer.go` for the canonical implementation.

The standard classification pattern uses a `KindMapping` slice:

```go
var myMappings = []importer.KindMapping{
    {Pattern: "agents/*.md", Kind: importer.KindAgent, Layout: importer.FlatFile},
    {Pattern: "skills/*/SKILL.md", Kind: importer.KindSkill, Layout: importer.DirectoryPerEntry},
    {Pattern: "rules/**/*.md", Kind: importer.KindRule, Layout: importer.FlatFile},
    // ... add mappings for each file type the provider uses
}
```

`Classify()` iterates mappings in order; first match wins. Available `Kind` values: `KindAgent`, `KindSkill`, `KindSkillAsset`, `KindRule`, `KindMCP`, `KindHook`, `KindHookScript`, `KindSettings`, `KindMemory`, `KindWorkflow`, `KindPolicy`. Available `Layout` values: `FlatFile`, `DirectoryPerEntry`, `StandaloneJSON`, `EmbeddedJSONKey`.

For `Import()`, delegate to `importer.RunImport()` for the standard walk-and-dispatch loop, or implement a custom walk if the provider has special directory traversal needs. Use shared extractors from `internal/importer/` when possible: `DefaultExtractRule()`, `DefaultExtractSkillAsset()`, `DefaultExtractHookScript()`.

### 6. Declare Capabilities Accurately

The `CapabilitySet` returned by the renderer's `Capabilities()` method must match what the provider's official documentation states. Overstating capabilities causes cross-provider test failures. Understating causes incorrect `RENDERER_KIND_UNSUPPORTED` emission and suppresses output that the provider could have produced.

### 7. Required Tests

- `providers/<name>/renderer_test.go` — table-driven unit tests for each `Compile*` method
- `providers/<name>/importer_test.go` — unit tests for `Classify()` and `Extract()`
- `internal/renderer/cross_provider_test.go` must pass — no code change needed if `CapabilitySet` is correctly declared
- `make test` and `make test-e2e` must pass with the new provider registered

The cross-provider test suite in `internal/renderer/cross_provider_test.go` runs four invariant tests against every registered renderer:

1. **RenderOrNote**: every resource kind either produces output files or emits a `RENDERER_KIND_UNSUPPORTED` fidelity note. Silent drops are a regression.
2. **NoRawModelAliases**: no renderer outputs raw xcaffold model aliases (e.g., `sonnet-4`) — all must be resolved to provider-specific model IDs.
3. **NoClaudeEnvVars**: non-Claude renderers must not emit `$CLAUDE_PROJECT_DIR` or other Claude-specific variables.
4. **FidelityCodesValid**: every fidelity note code must be registered in the global code registry.

To pass these tests, your renderer must:
- Declare capabilities accurately in `Capabilities()`
- Implement a `ModelResolver` that maps xcaffold aliases to provider model IDs
- Not leak implementation details from other providers

### 8. Documentation Updates

Add the provider to `docs/reference/supported-providers.md`. Follow the Diátaxis pillar structure for any conceptual pages. Ensure the `ProviderManifest.KindSupport` map and the renderer's `CapabilitySet` are both accurately declared — they serve different layers of the compilation pipeline. Changelog entries are generated automatically from your commit messages.

## Updating an Existing Provider

When a provider CLI changes its configuration format or adds new features, xcaffold's output may need updating. This is one of the most valuable community contributions — you're often the first to notice when a provider changes.

### When to File an Issue

If you notice that `xcaffold apply --target <provider>` produces output that your provider CLI ignores, misinterprets, or rejects, open an issue using the **Provider Incompatibility** template. Include your provider CLI version — this helps maintainers reproduce the issue.

### How Provider Updates Work

Provider changes are typically localized to two files:

1. **Renderer** — update `Compile*` methods in `providers/<name>/renderer.go`
2. **Golden tests** — update expected output fixtures in `providers/<name>/testdata/`

### Workflow

1. **Identify the change.** Compare the provider's current documentation or changelog against xcaffold's current renderer output. Note what changed.
2. **Update the renderer.** Modify `providers/<name>/renderer.go` to produce output matching the new format.
3. **Update golden tests.** Run `make test-update-golden` to regenerate expected output, then review the diff to confirm the changes are correct.
4. **Run the full test suite.** `make test` and `make test-e2e` must both pass. The cross-provider invariant test ensures your changes don't break other providers.
5. **Write a clear commit message.** Use `feat(<provider>):` or `fix(<provider>):` — release-please generates the changelog from commit messages.

## Architectural Constraints

### Compilation Direction and Import

xcaffold has two discrete, directional operations:

- **`xcaffold apply`** — compiles `.xcaf` manifests to provider-native output directories. This is the primary compilation direction. The `.xcaf` manifest is the source of truth for all compiled output.
- **`xcaffold import`** — reads an existing provider output directory and generates an equivalent `.xcaf` manifest. This is an explicit, one-shot capture operation, not a sync mechanism.

Do not introduce automatic bidirectional sync, file-watching reconciliation, or any mechanism that modifies `.xcaf` files in response to provider output changes without explicit user invocation. The distinction between a discrete import and a sync daemon is fundamental to the architecture.

### Provider-Agnostic Principle

No provider receives special treatment in `internal/` packages. Every provider is equal. Provider-specific conditional logic in shared code is a bug. If you find it, report it.

### Fidelity Notes — Not Silent Drops

When a provider does not support a resource kind, return an empty `map[string]string` from the relevant `Compile*` method. The orchestrator emits `RENDERER_KIND_UNSUPPORTED` automatically. Never silently drop data. Never return an error for unsupported-but-valid input — unsupported kinds are a normal, expected condition.

### Fidelity Notes

When a renderer cannot fully represent a resource, it emits a `FidelityNote` instead of silently dropping data. Fidelity notes are informational — they tell the user what was lost or approximated during compilation.

Return fidelity notes from `Compile*` methods alongside the output files. Use registered fidelity codes from `internal/renderer/fidelity_codes.go`. Never invent ad-hoc string codes — all codes must be registered so the cross-provider test can validate them.

### Optional Renderer Interfaces

- `MemoryAwareRenderer`: implement `SetMemoryRefs(agentRefs map[string]bool)` if this provider supports agent-scoped memory. The orchestrator calls this before `CompileAgents()` so agent output can reference memory configuration.
- `GlobalScanner`: set the `GlobalScanner` field in the manifest if this provider has user-global configuration (e.g., `~/.provider/`). The scanner discovers global resources and merges them into the compilation.

### .xcaf File Format and Key Convention

`.xcaf` files use two distinct formats depending on the resource kind:

- **Frontmatter + optional markdown body** (`---` delimiters): `agent`, `skill`, `rule`, `context`, `memory`
- **Pure YAML** (no delimiters): `hooks`, `settings`, `mcp`, `global`, `project`, `policy`, `blueprint`, `workflow`

Using the wrong format produces a parse error. When writing test fixtures, check which format the kind expects before authoring the file.

All `.xcaf` keys use kebab-case (e.g., `allowed-tools`, `disable-model-invocation`). Go struct field names are PascalCase; only the `yaml:` struct tag uses kebab-case. The parser uses Go's `yaml.Decoder.KnownFields(true)` to reject unknown keys at parse time — incorrect casing is a parse error, not a silent ignore.

### Schema Codegen

AST struct fields in `internal/ast/types.go` carry `// +xcaf:` markers that drive schema generation. Any new field you add requires the correct set of markers — use the existing fields in that file as the authoritative reference for which markers apply and what values are valid.

After any change to `internal/ast/types.go`, run:

```
make generate
make verify-markers
make verify-generate
```

CI enforces that generated files are fresh. A PR with stale generated files will not pass CI.

## Breaking Changes

xcaffold is pre-v1.0. Breaking changes to `.xcaf` schema, CLI flags, or compiler output are expected and do not require a deprecation cycle. Document breaking changes in the CHANGELOG under `Breaking Changes`.

## Response Times

Maintainers aim to respond to pull requests within 72 hours and to issues within one week. Review turnaround may be longer for large architectural changes.

## Governance

Project governance is documented in [GOVERNANCE.md](docs/governance/GOVERNANCE.md). It covers the decision-making process, maintainer responsibilities, and the contribution ladder.

## Good First Issues

Look for issues labeled `good first issue`. Good starting points: documentation fixes, adding test cases for existing providers, fixing bugs that have a clear repro. Before starting, comment on the issue to signal that you are working on it.
