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

Update `CHANGELOG.md` for every user-facing change. Add entries under `[Unreleased]`. For breaking changes, add a `Breaking Changes` entry.

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
    manifest.go     # ProviderManifest declaration and init()
    renderer.go     # TargetRenderer implementation
    importer.go     # ProviderImporter implementation
    renderer_test.go
    importer_test.go
```

Reference `providers/claude/` as the canonical example. Mature providers may also include `model_resolver.go`, `global.go`, and a `testdata/` directory. Do not modify `providers/registry.go` — providers self-register via `init()`.

### 3. Implement the Manifest

In `manifest.go`, declare a `providers.ProviderManifest` and call `providers.Register()` from `init()`. See `providers/claude/manifest.go` for the canonical pattern — field names, registration order, and capability declaration are all illustrated there.

### 4. Implement the Renderer

Implement the `renderer.TargetRenderer` interface defined in `internal/renderer/renderer.go`. For unsupported kinds, return an empty `map[string]string` and no error — the orchestrator emits `RENDERER_KIND_UNSUPPORTED` automatically based on the `CapabilitySet`. Reuse helpers from `internal/renderer/helpers.go` (`SortedKeys`, `YAMLScalar`, `StripAllFrontmatter`, `FlattenToSkillRoot`) rather than reimplementing them.

See `providers/claude/renderer.go` for the canonical implementation.

### 5. Implement the Importer

Implement the `importer.ProviderImporter` interface defined in `internal/importer/importer.go`. Embed `importer.BaseImporter` to satisfy `Provider()` and `InputDir()` without boilerplate. Define `Classify()`, `Extract()`, and `Import()` methods — `Import()` can delegate to `importer.RunImport()` for the standard walk-and-dispatch loop.

See `providers/claude/importer.go` for the canonical implementation.

### 6. Declare Capabilities Accurately

The `CapabilitySet` returned by the renderer's `Capabilities()` method must match what the provider's official documentation states. Overstating capabilities causes cross-provider test failures. Understating causes incorrect `RENDERER_KIND_UNSUPPORTED` emission and suppresses output that the provider could have produced.

### 7. Required Tests

- `providers/<name>/renderer_test.go` — table-driven unit tests for each `Compile*` method
- `providers/<name>/importer_test.go` — unit tests for `Classify()` and `Extract()`
- `internal/renderer/cross_provider_test.go` must pass — no code change needed if `CapabilitySet` is correctly declared
- `make test` and `make test-e2e` must pass with the new provider registered

### 8. Documentation Updates

Add the provider to `docs/reference/supported-providers.md`. Follow the Diátaxis pillar structure for any conceptual pages. Ensure the `ProviderManifest.KindSupport` map and the renderer's `CapabilitySet` are both accurately declared — they serve different layers of the compilation pipeline. Update `CHANGELOG.md` under `[Unreleased]`.

## Updating an Existing Provider

When a provider CLI changes its configuration format or adds new features, xcaffold's output may need updating. This is one of the most valuable community contributions — you're often the first to notice when a provider changes.

### When to File an Issue

If you notice that `xcaffold apply --target <provider>` produces output that your provider CLI ignores, misinterprets, or rejects, open an issue using the **Provider Incompatibility** template. Include your provider CLI version — this helps maintainers reproduce the issue.

### How Provider Updates Work

Provider changes are typically localized to three files:

1. **Renderer** (`providers/<name>/renderer.go`) — update `Compile*` methods to produce the new output format
2. **Ground truth** (`docs/agentic/data/ground_truth/db/`) — update the relevant JSON database entry with the new verified facts
3. **Golden tests** (`providers/<name>/testdata/`) — update expected output fixtures to match the new format

### Workflow

1. **Identify the change.** Compare the provider's current documentation or changelog against the ground truth entry in `docs/agentic/data/ground_truth/db/`. Note what changed.
2. **Update ground truth first.** Edit the relevant JSON file. Set `verified_at` to today's date. This is the source of truth that the renderer reads.
3. **Update the renderer.** Modify `providers/<name>/renderer.go` to produce output matching the new format.
4. **Update golden tests.** Run `make test-update-golden` to regenerate expected output, then review the diff to confirm the changes are correct.
5. **Run the full test suite.** `make test` and `make test-e2e` must both pass. The cross-provider invariant test ensures your changes don't break other providers.
6. **Update CHANGELOG.md.** Add an entry under `[Unreleased]` describing what changed and which provider is affected.

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

### .xcaf File Format and Key Convention

`.xcaf` files use two distinct formats depending on the resource kind:

- **Frontmatter + optional markdown body** (`---` delimiters): `agent`, `skill`, `rule`, `context`, `memory`, `workflow`
- **Pure YAML** (no delimiters): `hooks`, `settings`, `mcp`, `global`, `project`, `policy`, `blueprint`

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

## Good First Issues

Look for issues labeled `good first issue`. Good starting points: documentation fixes, adding test cases for existing providers, fixing bugs that have a clear repro. Before starting, comment on the issue to signal that you are working on it.
