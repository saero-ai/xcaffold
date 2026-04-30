---
name: go-standards
description: "Go code quality standards for the CLI codebase."
paths: ["xcaffold/**/*.go", "cmd/**/*.go", "internal/**/*.go"]
---
## Function Length

Keep functions under 50 lines. Extract sub-functions when a function has multiple responsibilities. Never suppress cyclomatic complexity linters with `//nolint` directives.

## Error Handling

Always wrap errors with context using `fmt.Errorf("operation %s: %w", name, err)`. Never use `panic()` in library code — return `(result, error)` instead. Never swallow errors into warning lists without surfacing them.

## Naming

- Test functions: `Test<Command>_<Feature>_<Scenario>`
- Helpers: verb-first, descriptive (`findImporterByProvider`, `pruneOrphanMemory`)
- Constants: camelCase unexported, PascalCase exported
- Interfaces: verb+er suffix (`ProviderImporter`, `TargetRenderer`)

## Interface Discipline

All provider interactions go through `ProviderImporter` or `TargetRenderer`. No inline `switch provider` blocks in `cmd/`. If adding a new provider requires changes to `cmd/`, the architecture has regressed.

## Testing

Use `t.TempDir()` for temporary directories. Use `testify/require` for fatal assertions, `testify/assert` for non-fatal. Always add negative tests alongside positive ones.
