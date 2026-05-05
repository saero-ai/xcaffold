# Contributing to xcaffold

## Ways to Contribute

- **Bug reports** — open a GitHub issue with a minimal reproduction case.
- **Documentation** — fix errors, improve examples, or fill coverage gaps in `docs/`.
- **New providers** — add support for a new output target (see "Adding a New Provider").
- **Core compiler changes** — improvements to the parser, AST, BIR, optimizer, or renderer pipeline.

## Setting Up

Prerequisites: Go 1.24 or later.

```
make setup
```

This installs `golangci-lint` globally and links the pre-commit hook from `scripts/pre-commit.sh`. No other setup is required.

## Reporting Bugs

Open a GitHub issue. Include:

- `xcaffold version` output
- OS and Go version
- Minimal `.xcf` file that reproduces the issue
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

Update `CHANGELOG.md` for every user-facing change. Add entries under `[Unreleased]`. For breaking changes, add both a `Breaking Changes` entry and a `Migration` entry.

### PR Checklist

- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] Documentation updated if behavior changed
- [ ] `CHANGELOG.md` updated

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
| How-to guides | `docs/how-to/` | Task-oriented procedures |
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
    manifest.go     # ProviderManifest declaration and init()
    renderer.go     # TargetRenderer implementation
    importer.go     # ProviderImporter implementation
```

Reference `providers/claude/` as the canonical example. Do not modify `providers/registry.go` — providers self-register via `init()`.

### 3. Implement the Manifest

In `manifest.go`, declare a `providers.ProviderManifest` and call `providers.Register()` from `init()`. See `providers/claude/manifest.go` for the canonical pattern — field names, registration order, and capability declaration are all illustrated there.

### 4. Implement the Renderer

Implement the `renderer.TargetRenderer` interface defined in `internal/renderer/renderer.go`. For unsupported kinds, return an empty `map[string]string` — the orchestrator emits `RENDERER_KIND_UNSUPPORTED` automatically based on the `CapabilitySet`. Reuse helpers from `internal/renderer/helpers.go` (`SortedKeys`, `YAMLScalar`, `StripAllFrontmatter`, `FlattenToSkillRoot`) rather than reimplementing them.

See `providers/claude/renderer.go` for the canonical implementation.

### 5. Implement the Importer

Implement the `importer.ProviderImporter` interface defined in `internal/importer/importer.go`. Embed `importer.BaseImporter` to satisfy `Provider()`, `InputDir()`, and `GetWarnings()` without boilerplate. Define a `KindMapping` table, implement `Classify()` and `Extract()`, and delegate the walk-and-dispatch loop to `RunImport()` from the base.

See `providers/claude/importer.go` for the canonical implementation.

### 6. Declare Capabilities Accurately

The `CapabilitySet` returned by `Capabilities()` must match what the provider's official documentation states. Overstating capabilities causes cross-provider test failures. Understating causes incorrect `RENDERER_KIND_UNSUPPORTED` emission and suppresses output that the provider could have produced.

### 7. Required Tests

- `providers/<name>/renderer_test.go` — table-driven unit tests for each `Compile*` method
- `providers/<name>/importer_test.go` — unit tests for `Classify()` and `Extract()`
- `internal/renderer/cross_provider_test.go` must pass — no code change needed if `CapabilitySet` is correctly declared
- `make test` and `make test-e2e` must pass with the new provider registered

### 8. Documentation Updates

Add the provider to `docs/reference/supported-providers.md`. Follow the Diátaxis pillar structure for any conceptual pages. Update `CHANGELOG.md` under `[Unreleased]`.

## Architectural Constraints

### Compilation Direction and Import

xcaffold has two discrete, directional operations:

- **`xcaffold apply`** — compiles `.xcf` manifests to provider-native output directories. This is the primary compilation direction. The `.xcf` manifest is the source of truth for all compiled output.
- **`xcaffold import`** — reads an existing provider output directory and generates an equivalent `.xcf` manifest. This is an explicit, one-shot capture operation, not a sync mechanism.

Do not introduce automatic bidirectional sync, file-watching reconciliation, or any mechanism that modifies `.xcf` files in response to provider output changes without explicit user invocation. The distinction between a discrete import and a sync daemon is fundamental to the architecture.

### Provider-Agnostic Principle

No provider receives special treatment in `internal/` packages. Every provider is equal. Provider-specific conditional logic in shared code is a bug. If you find it, report it.

### Fidelity Notes — Not Silent Drops

When a provider does not support a resource kind, return an empty `map[string]string` from the relevant `Compile*` method. The orchestrator emits `RENDERER_KIND_UNSUPPORTED` automatically. Never silently drop data. Never return an error for unsupported-but-valid input — unsupported kinds are a normal, expected condition.

### .xcf File Format and Key Convention

`.xcf` files use two distinct formats depending on the resource kind:

- **Frontmatter + optional markdown body** (`---` delimiters): `agent`, `skill`, `rule`, `workflow`
- **Pure YAML** (no delimiters): `hooks`, `settings`, `mcp`, `global`, `project`

Using the wrong format produces a parse error. When writing test fixtures, check which format the kind expects before authoring the file.

All `.xcf` keys use kebab-case (e.g., `allowed-tools`, `disable-model-invocation`). Go struct field names are PascalCase; only the `yaml:` struct tag uses kebab-case. The parser's `KnownFields` setting rejects unknown keys at parse time — incorrect casing is a parse error, not a silent ignore.

### Schema Codegen

AST struct fields in `internal/ast/types.go` carry `// +xcf:` markers that drive schema generation. Any new field you add requires the correct set of markers — use the existing fields in that file as the authoritative reference for which markers apply and what values are valid.

After any change to `internal/ast/types.go`, run:

```
make generate
make verify-markers
make verify-generate
```

CI enforces that generated files are fresh. A PR with stale generated files will not pass CI.

## Breaking Changes

A breaking change requires existing users to modify their `.xcf` files, CLI invocations, or tooling. Process:

1. Deprecate in the same PR with a runtime warning
2. Keep the old behavior working for at least one minor release
3. Remove in the following release
4. Add a `BREAKING CHANGE:` footer to the commit message
5. Add `Breaking Changes` and `Migration` entries to `CHANGELOG.md`

## Good First Issues

Look for issues labeled `good first issue`. Good starting points: documentation fixes, adding test cases for existing providers, fixing bugs that have a clear repro. Before starting, comment on the issue to signal that you are working on it.
