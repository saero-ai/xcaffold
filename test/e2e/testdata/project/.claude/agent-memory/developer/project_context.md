---
name: Project Context
description: Architecture decisions and project structure context
type: project
---
## Architecture

The project follows a clean architecture pattern with three layers:
- `cmd/` — CLI entry points and Cobra command wiring
- `internal/` — Business logic, parsers, compilers, renderers
- `test/` — Integration and stress tests

## Key Design Decisions

- YAML frontmatter is the canonical configuration format
- The compilation pipeline is one-way: `.xcf` source to provider-native output
- Each provider has its own renderer implementing the `TargetRenderer` interface
- Import uses `ProviderImporter` interface for provider-specific file parsing

## Conventions

- Error wrapping: `fmt.Errorf("context: %w", err)`
- Test naming: `Test<Command>_<Feature>_<Scenario>`
- No package-level mutable state except Cobra flag bindings
